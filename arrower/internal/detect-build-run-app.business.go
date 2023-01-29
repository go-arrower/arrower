package internal

import (
	"context"
	"log"
)

func WatchBuildAndRunApp(ctx context.Context, path string) error {
	filesChanged := make(chan File)

	go func() {
		err := WatchFolder(ctx, path, filesChanged)
		if err != nil {
			panic(err)
		}
	}()

	// ! ensure only one process is running at the same time and others are "queued"

	stop, err := BuildAndRunApp(path)
	if err != nil {
		log.Println("Arrower: build & run failed: ", err)
	}

	for {
		select {
		case f := <-filesChanged:
			log.Println("Arrower: change detected: ", f)

			checkAndStop(stop)

			stop, err = BuildAndRunApp(path)
			if err != nil {
				log.Println("Arrower: build & run failed: ", err)
			}
		case <-ctx.Done():
			checkAndStop(stop)

			return nil
		}
	}
}

func checkAndStop(stop func() error) {
	if stop != nil {
		err := stop()
		if err != nil {
			log.Println("Arrower: shutdown failed: ", err)
		}
	}
}
