package init

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/go-arrower/arrower/contexts/auth"
	"github.com/go-arrower/arrower/contexts/auth/internal/views"
)

// registerWebRoutes initialises all routes of this Context.
func (c *AuthContext) registerWebRoutes(router *echo.Group) {
	c.shared.WebRouter.StaticFS("/static/auth/", echo.MustSubFS(views.PublicAssets, "static"))

	addNamedRoute(router, "GET", "/login", auth.RouteLogin, c.userController.Login())
	router.POST("/login", c.userController.Login())
	addNamedRoute(router, "GET", "/logout", auth.RouteLogout, c.userController.Logout()) // todo make POST to prevent CSRF
	router.GET("/register", c.userController.Create())
	router.POST("/register", c.userController.Register())
	addNamedRoute(router, "GET", "/:userID/verify/:token", auth.RouteVerifyUser, c.userController.Verify())

	addNamedRoute(router, "GET", "/profile", "auth.profile", c.userController.Profile(), auth.EnsureUserIsLoggedInMiddleware)
	router.GET("/", nil, func(_ echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			return c.Render(http.StatusOK, "home", nil)
		}
	})
}

func addNamedRoute(group *echo.Group, method, path, name string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) {
	_, err := group.AddRoute(echo.Route{
		Method:      method,
		Path:        path,
		Name:        name,
		Handler:     handler,
		Middlewares: middleware,
	})
	if err != nil {
		panic(err)
	}
}
