package web

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func NewSettingsController() *SettingsController {
	return &SettingsController{}
}

type SettingsController struct{}

func (sc *SettingsController) List() func(c echo.Context) error {
	return func(c echo.Context) error {
		return c.String(http.StatusOK, "not implemented")
	}
}
