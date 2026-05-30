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
	lines := strings.Split(cgroup, "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		if id := extractContainerIDFromCgroupLine(line); id != "" {
			return id, nil
		}
	}

	return "", fmt.Errorf("container ID not found in cgroup")
}

func extractContainerIDFromCgroupLine(line string) string {
	// Common forms:
	// - 0::/system.slice/docker-<id>.scope
	// - 0::/docker/<id>
	// - .../cri-containerd-<id>.scope
	if idx := strings.Index(line, "docker-"); idx != -1 {
		rest := line[idx+len("docker-"):]
		if end := strings.Index(rest, ".scope"); end != -1 {
			if id := normalizeContainerID(rest[:end]); id != "" {
				return id
			}
		}
	}

	if idx := strings.Index(line, "cri-containerd-"); idx != -1 {
		rest := line[idx+len("cri-containerd-"):]
		if end := strings.Index(rest, ".scope"); end != -1 {
			if id := normalizeContainerID(rest[:end]); id != "" {
				return id
			}
		}
	}

	for _, part := range strings.Split(line, "/") {
		if id := normalizeContainerID(part); id != "" {
			return id
		}
	}

	return ""
}

func normalizeContainerID(raw string) string {
	id := strings.TrimSpace(raw)
	id = strings.TrimSuffix(id, ".scope")
	id = strings.TrimPrefix(id, "docker-")
	id = strings.TrimPrefix(id, "cri-containerd-")

	// Docker short/full IDs are lowercase hex, typically 12 or 64 chars.
	if len(id) < 12 {
		return ""
	}

	for _, ch := range id {
		isDigit := ch >= '0' && ch <= '9'
		isLowerHex := ch >= 'a' && ch <= 'f'
		if !isDigit && !isLowerHex {
			return ""
		}
	}

	if len(id) > 64 {
		id = id[:64]
	}

	return id
}

func (d *DockerResolver) ResolveServiceName(containerID string) (string, error) {
	d.mu.RLock()
	service, exists := d.containerToService[containerID]
	d.mu.RUnlock()

	if exists {
		return service, nil
	}

	// Cache miss (e.g. container started after last refresh). Inspect once.
	inspect, err := d.cli.ContainerInspect(
		context.Background(),
		containerID,
	)
	if err != nil {
		return "", err
	}

	var labels map[string]string
	var image string
	if inspect.Config != nil {
		labels = inspect.Config.Labels
		image = inspect.Config.Image
	}

	service = deriveServiceIdentity(
		labels,
		inspect.Name,
		image,
	)

	if service == "" {
		return "", fmt.Errorf("service label not found")
	}

	d.mu.Lock()
	d.containerToService[containerID] = service
	if len(inspect.ID) >= 12 {
		d.containerToService[inspect.ID[:12]] = service
	}
	d.containerToService[inspect.ID] = service
	d.mu.Unlock()

	return service, nil
}
