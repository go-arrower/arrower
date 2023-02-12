package internal

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

var ErrConnectionFailed = errors.New("ws connection failed")

const (
	// HotReloadPort is the port on which arrower apps can listen for hot reload signals.
	HotReloadPort = 3030

	// ReloadCmd is the command send to the browser, to reload a tab.
	ReloadCmd = "reload"
	// RefreshCSSCmd is the command send to the browser, to reload and swap css files only.
	RefreshCSSCmd = "refreshCSS"
)

func NewHotReloadServer(notify <-chan File) (*echo.Echo, error) {
	e := echo.New()
	e.HideBanner = true

	e.GET("/ws", HotReloadHandler(notify))

	return e, nil
}

type browserTab struct {
	ws    *websocket.Conn
	close chan struct{}
}

func (tab *browserTab) notify(msg string) error {
	err := websocket.Message.Send(tab.ws, msg)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	return nil
}

type browserSessions struct {
	openConnections map[string]browserTab
	mu              sync.Mutex
}

func (l *browserSessions) add(id string, ws *websocket.Conn) chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()

	c := make(chan struct{})
	l.openConnections[id] = browserTab{
		ws:    ws,
		close: c,
	}

	return c
}

func (l *browserSessions) remove(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	close(l.openConnections[id].close)
	delete(l.openConnections, id)
}

func HotReloadHandler(notify <-chan File) func(c echo.Context) error {
	browsers := &browserSessions{
		mu:              sync.Mutex{},
		openConnections: map[string]browserTab{},
	}

	go func() {
		for file := range notify {
			var err error

			browsers.mu.Lock()

			for id, conn := range browsers.openConnections {
				if file.IsCSS() {
					err = conn.notify(RefreshCSSCmd)
				} else {
					err = conn.notify(ReloadCmd)
				}

				if err != nil {
					browsers.remove(id)

					continue
				}
			}

			browsers.mu.Unlock()
		}
	}()

	const idLength = 5

	return func(c echo.Context) error {
		websocket.Handler(func(ws *websocket.Conn) {
			defer ws.Close()

			id := randomString(idLength)
			wsClosed := browsers.add(id, ws)

			<-wsClosed
		}).
			ServeHTTP(c.Response(), c.Request())

		return nil
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") //nolint:gochecknoglobals
func randomString(n int) string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // used for ids, not security

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rnd.Intn(len(letters))]
	}

	return string(b)
}
