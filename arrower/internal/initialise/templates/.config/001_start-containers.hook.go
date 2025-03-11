package main

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"time"

	. "github.com/go-arrower/arrower/arrower/hooks"
)

func init() {
	Register(Hook{
		Name: "DevOps",
		OnStart: func() {
			dockerPullOnesLock := "/tmp/docker-hook-lock"

			_, err := os.Stat(dockerPullOnesLock)
			if errors.Is(err, fs.ErrNotExist) {
				cmd := exec.Command("docker-compose", "-f", "devops/docker-compose.yaml", "pull")
				cmd.Stderr = os.Stderr
				cmd.Run()
			}

			file, err := os.Create(dockerPullOnesLock)
			if err != nil {
				fmt.Println("DevOps Plugin: could not create lock file:", err)
				os.Exit(1)
			}
			defer file.Close()

			cmd := exec.Command("docker-compose", "-f", "devops/docker-compose.yaml", "up", "-d")
			cmd.Stderr = os.Stderr
			cmd.Run()

			// open useful services in the browser
			const (
				pgAdminURL = "http://localhost:8081"
				grafanaURL = "http://localhost:3000"

				maxWaiting = 20
			)

			fmt.Println("...")

			for i := 0; i <= maxWaiting; i++ {
				if res, err := http.Get(pgAdminURL); err == nil {
					if res.StatusCode == 200 {
						exec.Command("xdg-open", pgAdminURL).Run()
						break
					}
				}

				if i >= maxWaiting {
					fmt.Println("DevOps Plugin: error could not reach pgAdmin")
					os.Exit(1)
				}

				time.Sleep(250 * time.Millisecond)
			}

			for i := 0; i <= maxWaiting; i++ {
				if res, err := http.Get(grafanaURL); err == nil {
					if res.StatusCode == 200 {
						exec.Command("xdg-open", grafanaURL).Run()
						break
					}
				}

				if i >= maxWaiting {
					fmt.Println("DevOps Plugin: error could not reach grafana")
					os.Exit(1)
				}

				time.Sleep(100 * time.Millisecond)
			}
		},
		OnShutdown: func() {
			fmt.Println("DevOps Plugin: manually stop docker services: docker-compose -f devops/docker-compose.yaml down")
		},
	})
}
