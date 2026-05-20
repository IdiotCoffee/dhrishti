package resolver

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func ResolveContainerID(pid uint32) (string, error) {
	path := fmt.Sprintf("/proc/%d/cgroup", pid)

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	cgroup := string(data)
	// need to slice the string to get the actual container ID
	// 0::/system.slice/docker-e65f5a1c82bae560c37fb526f8f2255cafa52f68a31c5badcd85a6875e5843b3.scope
	start := strings.Index(cgroup, "docker-")
	end := strings.Index(cgroup, ".scope")
	if start == -1 || end == -1 {
		return "", fmt.Errorf("container ID not found")
	}
	start += len("docker-")
	containerID := cgroup[start:end]
	// fmt.Println(string(containerID))
	return containerID, nil
}

func ResolveServiceName(containerID string) (string, error) {
	inspectPath := exec.Command("docker", "inspect", "--format={{index .Config.Labels \"com.docker.compose.service\"}}", containerID)
	containerName, err := inspectPath.Output()
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(string(containerName)), nil
}
