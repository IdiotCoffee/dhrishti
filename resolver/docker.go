package resolver

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/moby/moby/client"
)

/*
DockerResolver owns runtime identity resolution.

VERY IMPORTANT:

Container networking is dynamic runtime state.

Container IPs:
- change
- disappear
- get recreated

So:
identity resolution must become
a continuously refreshed subsystem.

NOT a startup snapshot.
*/
type DockerResolver struct {
	cli *client.Client

	/*
		Runtime IP -> service cache.

		This cache evolves continuously
		as containers:
		- restart
		- appear
		- disappear
		- scale

		VERY IMPORTANT:

		This cache is shared concurrently by:
		- telemetry pipelines
		- graph updates
		- API requests

		So synchronization is mandatory.
	*/
	mu sync.RWMutex

	ipToService map[string]string

	lastRefresh time.Time
}

/*
NewDockerResolver initializes:
- Docker API client
- runtime identity cache
*/
func NewDockerResolver() (*DockerResolver, error) {

	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)

	if err != nil {
		return nil, err
	}

	resolver := &DockerResolver{
		cli: cli,

		ipToService: make(map[string]string),
	}

	/*
		Initial runtime reconciliation.

		This prevents startup-time
		unknown service mappings.
	*/
	if err := resolver.RefreshCache(); err != nil {
		return nil, err
	}

	return resolver, nil
}

/*
RefreshCache rebuilds runtime identity state.

VERY IMPORTANT:

We build into a NEW map first,
then atomically swap ownership.

This prevents:
- partial visibility
- concurrent mutation races
- inconsistent reads
*/
func (d *DockerResolver) RefreshCache() error {

	newMap := make(map[string]string)

	/*
		Query all currently running containers.
	*/
	containers, err := d.cli.ContainerList(
		context.Background(),
		client.ContainerListOptions{},
	)

	if err != nil {

		log.Printf(
			"[resolver] listing containers failed: %v",
			err,
		)

		return err
	}

	/*
		Inspect each container individually.

		We derive:
		IP -> service identity
	*/
	for _, c := range containers {

		inspect, err := d.cli.ContainerInspect(
			context.Background(),
			c.ID,
		)

		if err != nil {

			log.Printf(
				"[resolver] inspect failed: %v",
				err,
			)

			continue
		}

		/*
			Docker Compose service identity.

			Example:
				gateway
				order-service
				payment-service
		*/
		service :=
			inspect.Config.Labels["com.docker.compose.service"]

		if service == "" {
			continue
		}

		/*
			Containers may belong to
			multiple Docker networks.

			So:
			we extract ALL IPs.
		*/
		for _, network := range inspect.NetworkSettings.Networks {

			ip := network.IPAddress

			if ip != "" {

				newMap[ip] = service
			}
		}
	}

	/*
		Atomically replace runtime cache.

		This is VERY important.

		Readers should NEVER observe:
		partially updated identity state.
	*/
	d.mu.Lock()

	d.ipToService = newMap
	d.lastRefresh = time.Now()

	d.mu.Unlock()

	return nil
}

/*
StartRefreshLoop continuously reconciles
runtime container identity.

This allows the observability graph
to evolve dynamically as services:
- appear
- disappear
- restart
- scale

VERY IMPORTANT:

Distributed systems are dynamic runtime environments.

Identity must evolve continuously.
*/
func (d *DockerResolver) StartRefreshLoop(
	interval time.Duration,
) {

	go func() {

		for {

			if err := d.RefreshCache(); err != nil {

				log.Printf(
					"[resolver] refresh failed: %v",
					err,
				)

			} else {

				log.Printf(
					"[resolver] refreshed runtime identity cache",
				)
			}

			time.Sleep(interval)
		}
	}()
}

/*
ResolveIP performs thread-safe
runtime identity lookup.
*/
func (d *DockerResolver) ResolveIP(
	ip string,
) string {

	d.mu.RLock()
	defer d.mu.RUnlock()

	service, exists :=
		d.ipToService[ip]

	if !exists {
		return "external"
	}

	return service
}
