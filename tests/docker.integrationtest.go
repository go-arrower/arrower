//go:build integration

package tests

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var ErrDockerFailure = errors.New("docker failure")

const dockerTimeout = 120 * time.Second

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
	mu.Lock()
	defer mu.Unlock()

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
	if errors.Is(err, docker.ErrContainerAlreadyExists) { // use existing container instead
		resource, err = getRunningContainer(pool, runOptions.Name)
		if err != nil {
			return nil, err
		}
	}
	if err != nil && !errors.Is(err, docker.ErrContainerAlreadyExists) { //nolint:wsl
		return nil, fmt.Errorf("%w: could not start resource: %v, options: %v", ErrDockerFailure, err, runOptions)
	}

	_ = resource.Expire(uint(dockerTimeout / time.Second)) // tell docker to hard kill the container in 120 seconds

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	pool.MaxWait = dockerTimeout
	if err := pool.Retry(retryFunc(resource)); err != nil {
		return nil, fmt.Errorf("%w: could not connect to docker: %v", ErrDockerFailure, err)
	}

	cleanup := getCleanupFunc(pool, resource)

	singleContainerInstance[resource.Container.Name]++

	return cleanup, nil
}

//nolint:gochecknoglobals // the variables are used on purpose for a singleton pattern.
var (
	mu                      = sync.Mutex{}
	singleContainerInstance = map[string]int{}
)

func getRunningContainer(pool *dockertest.Pool, name string) (*dockertest.Resource, error) {
	var resource *dockertest.Resource

	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = time.Second * 5 //nolint:gomnd
	bo.MaxElapsedTime = dockerTimeout

	if err := backoff.Retry(func() error {
		res, ok := pool.ContainerByName(name)
		if ok {
			resource = res

			return nil
		}

		return fmt.Errorf("%w: could not get running container", ErrDockerFailure)
	}, bo); err != nil {
		if bo.NextBackOff() == backoff.Stop {
			return nil, err //nolint:wrapcheck
		}

		return nil, err //nolint:wrapcheck
	}

	return resource, nil
}

func getCleanupFunc(pool *dockertest.Pool, resource *dockertest.Resource) func() error {
	return func() error {
		mu.Lock()
		defer mu.Unlock()

		singleContainerInstance[resource.Container.Name]--

		if singleContainerInstance[resource.Container.Name] == 0 {
			if err := pool.Purge(resource); err != nil {
				var noSuchContainer *docker.NoSuchContainer
				if errors.As(err, &noSuchContainer) { // container already stopped => don't report as error
					return nil
				}

				return fmt.Errorf("%w: could not purge resource: %v", ErrDockerFailure, err)
			}
		}

		return nil
	}
}
