package internal

import (
	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

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

func HotReloadHandler(notify <-chan File) func(c echo.Context) error {
	return func(c echo.Context) error {
		websocket.Handler(func(ws *websocket.Conn) {
			defer ws.Close()

			for f := range notify {
				var err error

				if f.IsCSS() {
					err = websocket.Message.Send(ws, RefreshCSSCmd)
				} else {
					err = websocket.Message.Send(ws, ReloadCmd)
				}

				if err != nil {
					c.Logger().Error(err)

					return
				}
			}
		}).
			ServeHTTP(c.Response(), c.Request())

		return nil
	}
}
