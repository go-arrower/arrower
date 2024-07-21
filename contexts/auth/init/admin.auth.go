package init

import (
	"github.com/labstack/echo/v4"

	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/web"
)

// registerAdminRoutes initialises all admin routes of this Context. To access the user has to have admin permissions.
// The admin routes work best in combination with the Admin Context initialised.
func (c *AuthContext) registerAdminRoutes(router *echo.Group, di localDI) {
	sCont := web.SuperuserController{Queries: di.queries}

	router.GET("/as_user/:userID", sCont.AdminLoginAsUser())
	router.GET("/leave_user", sCont.AdminLeaveUser())

	router.GET("/settings", c.settingsController.List())

	router.GET("/users", c.userController.List()).Name = "admin.users"
	router.POST("/users", c.userController.Register())
	router.GET("/users/:userID", c.userController.Show())
	router.GET("/users/:userID/sessions/:sessionKey", c.userController.DestroySession(di.queries))
	router.GET("/users/new", c.userController.New())
	router.POST("/users/new", c.userController.Store())

	c.userController.BlockUser()
	c.userController.UnBlockUser()
}
