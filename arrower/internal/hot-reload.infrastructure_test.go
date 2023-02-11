package internal_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/labstack/echo/v4"
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
		assert.IsType(t, &echo.Echo{}, s) //nolint:exhaustruct
	})

	t.Run("start server", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}
		server, _ := internal.NewHotReloadServer(make(chan internal.File))

		wg.Add(1)
		go func() {
			wg.Done()

			err := server.Start(fmt.Sprintf(":%d", internal.HotReloadPort))
			if !errors.Is(err, http.ErrServerClosed) {
				assert.NoError(t, err)
			}
		}()

		wg.Wait()
		err := server.Shutdown(context.Background())
		assert.NoError(t, err)
	})

	t.Run("receive ws reload messages", func(t *testing.T) {
		t.Parallel()

		ch := make(chan internal.File)
		s, err := internal.NewHotReloadServer(ch)
		assert.NoError(t, err)

		server := httptest.NewServer(s.Server.Handler)
		defer server.CloseClientConnections()
		defer server.Close()

		// connect via websocket
		addr := server.Listener.Addr().String()
		ws, err := websocket.Dial("ws://"+addr+"/ws", "", "http://localhost/")
		assert.NoError(t, err)
		defer ws.Close()

		t.Run("reload on view file change", func(t *testing.T) {
			ch <- "some.html"

			msg := ""
			err = websocket.Message.Receive(ws, &msg)
			assert.NoError(t, err)
			assert.Equal(t, internal.ReloadCmd, msg)
		})

		t.Run("refresh css only", func(t *testing.T) {
			ch <- "some.css"

			msg := ""
			err = websocket.Message.Receive(ws, &msg)
			assert.NoError(t, err)
			assert.Equal(t, internal.RefreshCSSCmd, msg)
		})
	})
}
