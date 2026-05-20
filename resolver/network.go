// server.py currently does not have a PID being displayed directly, so I will need to resolve the IP
// package resolver
package resolver

import (
	"context"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// making a docker client connector to connect to docker via a socket.

func NewDockerClient() (*client.Client, error) {
	return client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
}

// type DockerInspect []struct {
// 	NetworkSettings struct {
// 		Networks map[string]struct {
// 			IPAddress string `json:"IPAddress"`
// 		} `json:"Networks"`
// 	} `json:"NetworkSettings"`

// 	Config struct {
// 		Labels map[string]string `json:"Labels"`
// 	} `json:"Config"`
// }

func BuildIPServiceMap() (map[string]string, error) {
	cli, err := NewDockerClient()
	if err != nil {
		log.Fatalln(err)
	}

	ipMap := make(map[string]string)

	// cmd := exec.Command("docker", "ps", "-q")
	// repacing the old shell command method with actual docker client library code
	containers, err := cli.ContainerList(
		context.Background(),
		types.ContainerListOptions{},
	)

	// output, err := cmd.Output()
	// if err != nil {
	// 	return nil, err
	// }

	// containerIDs := strings.Fields(string(containers))
	for _, c := range containers {
		// inspectCmd := exec.Command("docker", "inspect", id)
		inspect, err := cli.ContainerInspect(
			context.Background(),
			c.ID,
		)
		if err != nil {
			log.Println(err)
			continue
		}

		// inspectOutput, err := inspectCmd.Output()
		// if err != nil {
		// 	continue
		// }

		// var data DockerInspect

		// err = json.Unmarshal(inspect, &data)
		// if err != nil {
		// 	continue
		// }

		// if len(data) == 0 {
		// 	continue
		// }

		service := inspect.Config.Labels["com.docker.compose.service"]

		for _, network := range inspect.NetworkSettings.Networks {
			ip := network.IPAddress

			if ip != "" && service != "" {
				ipMap[ip] = service
			}
		}
	}

	return ipMap, nil
}
