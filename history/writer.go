package history

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"dhrishti/api"
	"dhrishti/config"
	"dhrishti/types"

	"github.com/redis/go-redis/v9"
)

// Writer periodically samples graph state into Redis TimeSeries and snapshots.
type Writer struct {
	client      *redis.Client
	interval    time.Duration
	retention   time.Duration
	createdKeys sync.Map
}

// StartWriter connects to Redis and begins background sampling.
// Returns nil when history is disabled; errors are logged and sampling is skipped.
func StartWriter(
	graph *types.Graph,
	unknown *types.UnknownIPRegistry,
	cfg config.Config,
	historyCfg Config,
) *Writer {
	if !historyCfg.Enabled {
		log.Println("[history] disabled (set DHRISHTI_HISTORY_ENABLED=true to enable)")
		return nil
	}

	addr, password, db := parseRedisURL(historyCfg.RedisURL)
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("[history] redis unavailable (%s): %v — history writer disabled", addr, err)
		_ = client.Close()
		return nil
	}

	w := &Writer{
		client:    client,
		interval:  historyCfg.Interval,
		retention: historyCfg.Retention,
	}

	log.Printf(
		"[history] writing to redis %s every %s (retention %s)",
		addr,
		historyCfg.Interval,
		historyCfg.Retention,
	)

	go w.loop(graph, unknown, cfg)
	return w
}

func (w *Writer) loop(
	graph *types.Graph,
	unknown *types.UnknownIPRegistry,
	cfg config.Config,
) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for range ticker.C {
		w.sample(graph, unknown, cfg)
	}
}

func (w *Writer) sample(
	graph *types.Graph,
	unknown *types.UnknownIPRegistry,
	cfg config.Config,
) {
	now := time.Now()
	tsMs := now.UnixMilli()
	retentionMs := retentionMs(w.retention)

	response := api.BuildGraphResponse(graph, unknown, cfg)
	payload, err := json.Marshal(struct {
		Timestamp string           `json:"timestamp"`
		Graph     api.GraphResponse `json:"graph"`
	}{
		Timestamp: now.UTC().Format(time.RFC3339),
		Graph:     response,
	})
	if err != nil {
		log.Printf("[history] marshal snapshot: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipe := w.client.Pipeline()

	pipe.Set(ctx, snapshotKey(tsMs), payload, w.retention)
	pipe.ZAdd(ctx, snapshotsIndexKey, redis.Z{Score: float64(tsMs), Member: tsMs})
	pipe.ZRemRangeByScore(ctx, snapshotsIndexKey, "-inf", fmt.Sprintf("%d", tsMs-retentionMs))

	serviceAgg := make(map[string]*serviceMetrics)

	for _, edge := range response.Edges {
		source := edge.Source
		dest := edge.Target

		pipe.SAdd(ctx, servicesKnownKey, source, dest)
		pipe.SAdd(ctx, edgesKnownKey, edgeKnownMember(source, dest))

		for metric, value := range map[string]float64{
			"rps":                edge.RequestsPerSecond,
			"failure_rate":       edge.FailureRate,
			"p95_latency_ms":     float64(edge.P95LatencyMs),
			"avg_latency_ms":     float64(edge.RecentAverageLatencyMs),
			"active_connections": float64(edge.ActiveConnections),
		} {
			key := edgeSeriesKey(source, dest, metric)
			labels := map[string]string{
				"source": source, "dest": dest, "metric": metric,
			}
			w.ensureEdgeSeries(ctx, key, retentionMs, labels)
			pipe.Do(ctx, "TS.ADD", key, tsMs, value)
		}

		accumulateService(serviceAgg, source, edge, true)
		accumulateService(serviceAgg, dest, edge, false)
	}

	for service, agg := range serviceAgg {
		w.writeServiceSeries(ctx, pipe, service, agg, tsMs, retentionMs)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("[history] redis pipeline: %v", err)
	}
}

type serviceMetrics struct {
	outboundRPS       float64
	inboundRPS        float64
	outboundFailed    float64
	outboundSamples   float64
	maxOutboundP95    float64
	activeConnections int
}

func accumulateService(agg map[string]*serviceMetrics, service string, edge api.EdgeResponse, outbound bool) {
	m, ok := agg[service]
	if !ok {
		m = &serviceMetrics{}
		agg[service] = m
	}

	if outbound {
		m.outboundRPS += edge.RequestsPerSecond
		m.outboundFailed += edge.FailureRate * edge.RequestsPerSecond
		m.outboundSamples += edge.RequestsPerSecond
		if float64(edge.P95LatencyMs) > m.maxOutboundP95 {
			m.maxOutboundP95 = float64(edge.P95LatencyMs)
		}
		m.activeConnections += edge.ActiveConnections
	} else {
		m.inboundRPS += edge.RequestsPerSecond
	}
}

func (w *Writer) writeServiceSeries(
	ctx context.Context,
	pipe redis.Pipeliner,
	service string,
	agg *serviceMetrics,
	tsMs int64,
	retentionMs int64,
) {

	failureRate := 0.0
	if agg.outboundSamples > 0 {
		failureRate = agg.outboundFailed / agg.outboundSamples
	}

	for metric, value := range map[string]float64{
		"outbound_rps":       agg.outboundRPS,
		"inbound_rps":        agg.inboundRPS,
		"failure_rate":       failureRate,
		"p95_latency_ms":     agg.maxOutboundP95,
		"active_connections": float64(agg.activeConnections),
	} {
		key := serviceSeriesKey(service, metric)
		w.ensureServiceSeries(ctx, key, retentionMs, service, metric)
		pipe.Do(ctx, "TS.ADD", key, tsMs, value)
	}
}

func (w *Writer) ensureEdgeSeries(
	ctx context.Context,
	key string,
	retentionMs int64,
	labels map[string]string,
) {
	if _, ok := w.createdKeys.Load(key); ok {
		return
	}

	args := []interface{}{
		"TS.CREATE", key,
		"RETENTION", retentionMs,
		"DUPLICATE_POLICY", "LAST",
	}
	for k, v := range labels {
		args = append(args, "LABELS", k, v)
	}
	if err := w.client.Do(ctx, args...).Err(); err != nil && !isBusyKey(err) {
		log.Printf("[history] TS.CREATE %s: %v", key, err)
		return
	}
	w.createdKeys.Store(key, struct{}{})
}

func (w *Writer) ensureServiceSeries(
	ctx context.Context,
	key string,
	retentionMs int64,
	service string,
	metric string,
) {
	if _, ok := w.createdKeys.Load(key); ok {
		return
	}

	err := w.client.Do(
		ctx,
		"TS.CREATE", key,
		"RETENTION", retentionMs,
		"DUPLICATE_POLICY", "LAST",
		"LABELS", "service", service, "metric", metric,
	).Err()
	if err != nil && !isBusyKey(err) {
		log.Printf("[history] TS.CREATE %s: %v", key, err)
		return
	}
	w.createdKeys.Store(key, struct{}{})
}

func isBusyKey(err error) bool {
	return err != nil && strings.Contains(err.Error(), "already exists")
}
