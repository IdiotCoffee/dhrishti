package resolver

import (
	"github.com/moby/moby/client"
)

type DockerResolver struct {
	cli *client.Client
}

func NewDockerResolver() (*DockerResolver, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}

	return &DockerResolver{
		cli: cli,
	}, nil
}
