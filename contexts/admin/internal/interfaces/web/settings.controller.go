package web

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func NewSettingsController() *SettingsController {
	return &SettingsController{}
}

type SettingsController struct{}

func (ctrl *SettingsController) Index() func(c echo.Context) error {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "settings.index", nil)
	}
}
