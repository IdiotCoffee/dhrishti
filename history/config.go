package history

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config controls optional Redis history publishing.
type Config struct {
	Enabled   bool
	RedisURL  string
	Interval  time.Duration
	Retention time.Duration
}

func LoadConfig() Config {
	redisURL := strings.TrimSpace(os.Getenv("DHRISHTI_REDIS_URL"))
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	enabled := true
	if v := strings.TrimSpace(os.Getenv("DHRISHTI_HISTORY_ENABLED")); v != "" {
		enabled = v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
	}

	interval := 10 * time.Second
	if v := strings.TrimSpace(os.Getenv("DHRISHTI_HISTORY_INTERVAL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			interval = d
		}
	}

	retention := 24 * time.Hour
	if v := strings.TrimSpace(os.Getenv("DHRISHTI_HISTORY_RETENTION")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			retention = d
		}
	}

	return Config{
		Enabled:   enabled,
		RedisURL:  redisURL,
		Interval:  interval,
		Retention: retention,
	}
}

func retentionMs(d time.Duration) int64 {
	ms := d.Milliseconds()
	if ms <= 0 {
		return 86400000
	}
	return ms
}

func parseRedisURL(raw string) (addr string, password string, db int) {
	addr = "localhost:6379"
	db = 0

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return addr, "", db
	}

	if strings.HasPrefix(raw, "redis://") || strings.HasPrefix(raw, "rediss://") {
		withoutScheme := strings.TrimPrefix(strings.TrimPrefix(raw, "rediss://"), "redis://")
		if at := strings.LastIndex(withoutScheme, "@"); at >= 0 {
			userinfo := withoutScheme[:at]
			withoutScheme = withoutScheme[at+1:]
			if colon := strings.Index(userinfo, ":"); colon >= 0 {
				password = userinfo[colon+1:]
			}
		}
		if slash := strings.Index(withoutScheme, "/"); slash >= 0 {
			hostport := withoutScheme[:slash]
			if n, err := strconv.Atoi(withoutScheme[slash+1:]); err == nil {
				db = n
			}
			addr = hostport
		} else {
			addr = withoutScheme
		}
		return addr, password, db
	}

	return raw, "", db
}
