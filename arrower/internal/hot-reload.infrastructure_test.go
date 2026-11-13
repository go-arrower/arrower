package internal_test

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/websocket"

	"github.com/go-arrower/arrower/arrower/internal"
)

func TestNewHotReloadServer(t *testing.T) {
	t.Parallel()

	t.Run("create hot reload server", func(t *testing.T) {
		t.Parallel()

		s, err := internal.NewHotReloadServer(make(chan internal.File))
		assert.NoError(t, err)
		assert.NotNil(t, s)
		assert.IsType(t, &echo.Echo{}, s)
	})

	t.Run("start server", func(t *testing.T) {
		t.Parallel()

		server, _ := internal.NewHotReloadServer(make(chan internal.File))
		port := 1024 * rand.Intn(10) //nolint:gosec // rand does not need to be secure for port number
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan error, 1)
		started := make(chan struct{})
		go func() {
			done <- (echo.StartConfig{
				Address: fmt.Sprintf(":%d", port),
				ListenerAddrFunc: func(net.Addr) {
					close(started)
				},
			}).Start(ctx, server)
		}()

		<-started
		cancel()
		assert.NoError(t, <-done)
	})

	t.Run("receive ws reload messages", func(t *testing.T) {
		t.Parallel()

		ch := make(chan internal.File, 1)
		s, err := internal.NewHotReloadServer(ch)
		assert.NoError(t, err)

		server := httptest.NewServer(s)
		defer server.CloseClientConnections()
		defer server.Close()

		// connect via websocket
		addr := server.Listener.Addr().String()
		ws, err := websocket.Dial("ws://"+addr+"/ws", "", "http://localhost/")
		assert.NoError(t, err)

		defer ws.Close()

		t.Run("reload on view file change", func(t *testing.T) { //nolint:paralleltest
			ch <- "some.html"

			msg := ""
			err = websocket.Message.Receive(ws, &msg)
			assert.NoError(t, err)
			assert.Equal(t, internal.ReloadCmd, msg)
		})

		t.Run("refresh css only", func(t *testing.T) { //nolint:paralleltest
			ch <- "some.css"

			msg := ""
			err = websocket.Message.Receive(ws, &msg)
			assert.NoError(t, err)
			assert.Equal(t, internal.RefreshCSSCmd, msg)
		})
	})

	t.Run("receive ws reload messages on multiple (dropping out) browser connections", func(t *testing.T) {
		t.Parallel()

		const maxConnections = 10

		ch := make(chan internal.File, maxConnections)
		defer close(ch)

		s, err := internal.NewHotReloadServer(ch)
		assert.NoError(t, err)

		server := httptest.NewServer(s)
		defer server.CloseClientConnections()
		defer server.Close()

		addr := server.Listener.Addr().String()

		wg := sync.WaitGroup{}
		wgBrowsersConnected := sync.WaitGroup{}

		wg.Add(maxConnections)
		wgBrowsersConnected.Add(maxConnections)

		for range maxConnections {
			go func(wg *sync.WaitGroup) {
				defer wg.Done()

				ws, err := websocket.Dial("ws://"+addr+"/ws", "", "http://localhost/")
				if !assert.NoError(t, err) {
					wgBrowsersConnected.Done()
					return
				}

				wgBrowsersConnected.Done()

				msg := ""
				err = websocket.Message.Receive(ws, &msg)
				assert.NoError(t, err)
				assert.Equal(t, internal.ReloadCmd, msg)

				// Simulate a browser disconnect after receiving its first reload.
				assert.NoError(t, ws.Close())
			}(&wg)
		}

		wgBrowsersConnected.Wait()

		clientsClosed := make(chan struct{})
		go func() {
			wg.Wait()
			close(clientsClosed)
		}()

		ch <- "some.html"
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		deadline := time.NewTimer(5 * time.Second)
		defer deadline.Stop()
		for {
			select {
			case <-clientsClosed:
				goto clientsDisconnected
			case <-ticker.C:
				ch <- "some.html"
			case <-deadline.C:
				t.Fatal("timed out waiting for browser connections to receive reloads")
			}
		}

	clientsDisconnected:
		// A later write lets the server detect the closed websocket connections and release handlers.
		for range maxConnections {
			ch <- "some.html"
		}
	})
}
