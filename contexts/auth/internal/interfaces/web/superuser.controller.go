package web

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"

	"github.com/go-arrower/arrower/contexts/auth"
	"github.com/go-arrower/arrower/contexts/auth/internal/interfaces/repository/models"
)

type SuperuserController struct {
	Queries *models.Queries
}

func (cont SuperuserController) AdminLoginAsUser() echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.Param("userID")

		user, err := cont.Queries.FindUserByID(c.Request().Context(), uuid.MustParse(userID))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		sess, err := session.Get(auth.SessionName, c)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		if originalUserID, ok := sess.Values[auth.SessKeyUserID].(string); ok {
			sess.Values[auth.SessIsSuperuserLoggedInAsUser] = true
			sess.Values[auth.SessSuperuserOriginalUserID] = originalUserID

			sess.Values[auth.SessKeyUserID] = user.ID.String()
			sess.AddFlash(fmt.Sprintf("Angemeldet als Nutzer: %s", user.Login))

			err = sess.Save(c.Request(), c.Response())
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
		}

		return c.Redirect(http.StatusSeeOther, "/")
	}
}

func (cont SuperuserController) AdminLeaveUser() echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get(auth.SessionName, c)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		if originalUserID, ok := sess.Values[auth.SessSuperuserOriginalUserID].(string); ok {
			delete(sess.Values, auth.SessIsSuperuserLoggedInAsUser)
			delete(sess.Values, auth.SessSuperuserOriginalUserID)

			sess.Values[auth.SessKeyUserID] = originalUserID
			sess.AddFlash("Left user and back to superuser")

			err = sess.Save(c.Request(), c.Response())
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
		}

		return c.Redirect(http.StatusSeeOther, "/admin/")
	}
}
