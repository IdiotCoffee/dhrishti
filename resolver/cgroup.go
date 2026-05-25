package resolver

import (
	"context"
	"fmt"
	"os"
	"strings"
)

func ResolveContainerID(pid uint32) (string, error) {
	path := fmt.Sprintf("/proc/%d/cgroup", pid)

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	cgroup := string(data)

	// Example:
	// 0::/system.slice/docker-<container-id>.scope

	start := strings.Index(cgroup, "docker-")
	if start == -1 {
		return "", fmt.Errorf("docker prefix not found in cgroup")
	}

	relativeEnd := strings.Index(cgroup[start:], ".scope")
	if relativeEnd == -1 {
		return "", fmt.Errorf(".scope suffix not found in cgroup")
	}

	end := start + relativeEnd

	start += len("docker-")

	containerID := cgroup[start:end]

	if containerID == "" {
		return "", fmt.Errorf("empty container ID extracted")
	}

	return containerID, nil
}

func (d *DockerResolver) ResolveServiceName(containerID string) (string, error) {
	cli := d.cli

	inspect, err := cli.ContainerInspect(
		context.Background(),
		containerID,
	)
	if err != nil {
		return "", err
	}

	service := inspect.Config.Labels["com.docker.compose.service"]

	if service == "" {
		return "", fmt.Errorf("service label not found")
	}

	return service, nil
}
