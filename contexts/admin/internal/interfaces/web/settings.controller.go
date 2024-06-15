package web

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func NewSettingsController(routes *echo.Group) *SettingsController {
	return &SettingsController{
		r: routes,
	}
}

type SettingsController struct {
	r *echo.Group
}

func (sc *SettingsController) List() {
	sc.r.GET("/settings", func(c echo.Context) error {
		return c.String(http.StatusOK, "not implemented")
	}).Name = "admin.settings"
}
