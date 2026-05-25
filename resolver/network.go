// server.py currently does not have a PID being displayed directly, so I will need to resolve the IP
// package resolver
package resolver

import (
	"context"
	"log"

	// "github.com/moby/moby/api/types"
	"github.com/moby/moby/client"
)

// making a docker client connector to connect to docker via a socket.

// function to map IP to a service name
func (d *DockerResolver) BuildIPServiceMap() (map[string]string, error) {
	cli := d.cli

	ipMap := make(map[string]string)
	// list the containers
	containers, err := cli.ContainerList(
		context.Background(),
		// types.ContainerListOptions{},
		client.ContainerListOptions{},
	)
	if err != nil {
		log.Printf("Error in listing containers")
		return nil, err
	}
	// for each container, find the information
	for _, c := range containers {
		// run the inspect command on each container id
		inspect, err := cli.ContainerInspect(
			context.Background(),
			c.ID,
		)
		if err != nil {
			log.Println(err)
			continue
		}
		// check for all IP addresses
		service := inspect.Config.Labels["com.docker.compose.service"]
		// map ip address to a non-empty service.
		for _, network := range inspect.NetworkSettings.Networks {
			ip := network.IPAddress

			if ip != "" && service != "" {
				ipMap[ip] = service
			}
		}
	}

	return ipMap, nil
}
