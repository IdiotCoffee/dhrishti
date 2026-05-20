// server.py currently does not have a PID being displayed directly, so I will need to resolve the IP
// package resolver
package resolver

import (
	"encoding/json"
	"os/exec"
	"strings"
)

type DockerInspect []struct {
	NetworkSettings struct {
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`

	Config struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
}

func BuildIPServiceMap() (map[string]string, error) {
	ipMap := make(map[string]string)

	cmd := exec.Command("docker", "ps", "-q")

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	containerIDs := strings.Fields(string(output))

	for _, id := range containerIDs {
		inspectCmd := exec.Command("docker", "inspect", id)

		inspectOutput, err := inspectCmd.Output()
		if err != nil {
			continue
		}

		var data DockerInspect

		err = json.Unmarshal(inspectOutput, &data)
		if err != nil {
			continue
		}

		if len(data) == 0 {
			continue
		}

		service := data[0].Config.Labels["com.docker.compose.service"]

		for _, network := range data[0].NetworkSettings.Networks {
			ip := network.IPAddress

			if ip != "" && service != "" {
				ipMap[ip] = service
			}
		}
	}

	return ipMap, nil
}
