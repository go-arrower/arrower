package init

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// registerAPIRoutes initialises all api routes of this Context. API routes require a valid auth.APIKey.
// It is best practise to version your API.
func (c *AuthContext) registerAPIRoutes(v1 *echo.Group) {
	v1 = v1.Group(fmt.Sprintf("/v1/%s", contextName))

	v1.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "HELLO FROM AUTH API")
	})
}
