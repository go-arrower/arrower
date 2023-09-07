//go:build integration

package tests

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var (
	ErrDockerFailure       = errors.New("docker failure")
	ErrMissingInstanceName = errors.New("missing docker instance name")
)

// RetryFunc is the function you use to connect to the docker container.
// Return a func that will be used by the dockertest.Pool for the actual connection,
// it will be called multiple times in attempts to connect to the container, while that's still starting up.
type RetryFunc func(resource *dockertest.Resource) func() error

// GetDockerContainerInstance returns a fully running container.
// Subsequent calls return a cleanup function for the same container to prevent multiple docker containers to spin up,
// if you have a lot of integration tests running in parallel.
// runOptions.Name does need to match to return the same container, other options will be ignored in this case.
func GetDockerContainerInstance(runOptions *dockertest.RunOptions, retryFunc RetryFunc) (func() error, error) {
	if runOptions.Name == "" {
		return nil, ErrMissingInstanceName
	}

	name := "/" + runOptions.Name
	if singleContainerInstance[name] == nil {
		_, err := StartDockerContainer(runOptions, retryFunc)
		if err != nil {
			return nil, err
		}
	}

	mu.Lock()
	defer mu.Unlock()
	singleContainerInstance[name].running++

	return singleContainerInstance[name].cleanup, nil
}

type containerInstance struct {
	cleanup func() error
	running int // how often an instance got requested via GetDockerContainerInstance
}

//nolint:gochecknoglobals // GetDockerContainerInstance is a singleton so multiple tests share a docker container.
var (
	singleContainerInstance = map[string]*containerInstance{}
	mu                      = sync.Mutex{}
)

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

	cleanup := getCleanupFunc(pool, resource)

	mu.Lock()
	defer mu.Unlock()

	singleContainerInstance[resource.Container.Name] = &containerInstance{
		cleanup: cleanup,
		running: 1,
	}

	return cleanup, nil
}

func getCleanupFunc(pool *dockertest.Pool, resource *dockertest.Resource) func() error {
	return func() error {
		mu.Lock()
		defer mu.Unlock()

		singleContainerInstance[resource.Container.Name].running--
		if singleContainerInstance[resource.Container.Name].running != 0 {
			return nil // don't stop container, as some tests are still using it
		}

		if err := pool.Purge(resource); err != nil {
			return fmt.Errorf("%w: could not purge resource: %v", ErrDockerFailure, err)
		}

		return nil
	}
}
