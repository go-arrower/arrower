package init

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/go-arrower/arrower/contexts/auth"
)

// registerWebRoutes initialises all routes of this Context.
func (c *AuthContext) registerWebRoutes(router *echo.Group) {
	router.GET("/login", c.userController.Login()).Name = auth.RouteLogin
	router.POST("/login", c.userController.Login())
	router.GET("/logout", c.userController.Logout()).Name = auth.RouteLogout // todo make POST to prevent CSRF
	router.GET("/register", c.userController.Create())
	router.POST("/register", c.userController.Register())
	router.GET("/:userID/verify/:token", c.userController.Verify()).Name = auth.RouteVerifyUser

	router.GET("/profile", c.userController.Profile(), auth.EnsureUserIsLoggedInMiddleware).Name = "auth.profile"
	router.GET("/", nil, func(_ echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return c.Render(http.StatusOK, "home", nil)
		}
	})
}
