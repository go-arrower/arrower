//go:build integration

package tests

import (
	"errors"
	"fmt"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var ErrDockerFailure = errors.New("docker failure")

// RetryFunc is the function you use to connect to the docker container.
// Return a func that will be used by the dockertest.Pool for the actual connection,
// it will be called multiple times in attempts to connect to the container, while that's still starting up.
type RetryFunc func(resource *dockertest.Resource) func() error

// StartDockerContainer connects to the local docker service and starts a container for integration testing.
// Configure the container to start by setting the dockertest.RunOptions, the most important ones:
// - Repository:	is the dockerhub repo to pull, e.g. "postgres"
// - Tag:			is the tag to pull, e.g. 15
// - Env:			are the env variables to set for the container
// For more options check out the RunOptions struct.
func StartDockerContainer(runOptions *dockertest.RunOptions, retryFunc RetryFunc) (func() error, error) {
	if runOptions == nil {
		return nil, fmt.Errorf("%w: invalid run options", ErrDockerFailure)
	}

	if retryFunc == nil {
		return nil, fmt.Errorf("%w: invalid retry func", ErrDockerFailure)
	}

	pool, err := dockertest.NewPool("") // uses a sensible default on windows (tcp/http) and linux/osx (socket)
	if err != nil {
		return nil, fmt.Errorf("%w: could not create new pool: %v", ErrDockerFailure, err)
	}

	err = pool.Client.Ping()
	if err != nil {
		return nil, fmt.Errorf("%w: could not connect to docker: %v", ErrDockerFailure, err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(
		runOptions,
		func(config *docker.HostConfig) {
			config.AutoRemove = true // set AutoRemove to true so that stopped container goes away by itself
			config.RestartPolicy = docker.RestartPolicy{Name: "no", MaximumRetryCount: 0}
		})
	if err != nil {
		return nil, fmt.Errorf("%w: could not start resource: %v", ErrDockerFailure, err)
	}

	const dockerTimeout = 120
	_ = resource.Expire(dockerTimeout) // tell docker to hard kill the container in 120 seconds

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	pool.MaxWait = dockerTimeout * time.Second
	if err := pool.Retry(retryFunc(resource)); err != nil {
		return nil, fmt.Errorf("%w: could not connect to docker: %v", ErrDockerFailure, err)
	}

	return func() error {
		if err := pool.Purge(resource); err != nil {
			return fmt.Errorf("%w: could not purge resource: %v", ErrDockerFailure, err)
		}

		return nil
	}, nil
}
