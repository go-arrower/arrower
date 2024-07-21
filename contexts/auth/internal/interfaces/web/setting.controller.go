package web

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository/models"
)

/*
Proposal for naming conventions:
	- index (list)
	- create (new)
	- store (new)
	- show
	- edit
	- update
	- delete
*/

func NewSettingsController(queries *models.Queries) *SettingsController {
	return &SettingsController{queries: queries}
}

type SettingsController struct {
	queries *models.Queries
}

func (sc SettingsController) List() func(echo.Context) error {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "settings", nil)
	}
}
