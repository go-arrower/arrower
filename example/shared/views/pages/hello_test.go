package pages_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-arrower/arrower/renderer"

	"github.com/go-arrower/skeleton/shared/domain"
	"github.com/go-arrower/skeleton/shared/views"
	"github.com/go-arrower/skeleton/shared/views/pages"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

var (
	ctx = context.Background()
)

func TestHelloPage_ShowTeamBanner(t *testing.T) {
	t.Parallel()

	page := pages.PresentHello(domain.TeamMember{
		Name:         "Peter",
		TeamTime:     time.Now(),
		IsTeamMember: true,
	})

	assert.NotEmpty(t, page.TeamTimeFmt)
	assert.True(t, page.ShowTeamBanner())
}

func TestSome(t *testing.T) {
	t.Parallel()

	renderer, _ := renderer.Test(views.SharedViews, nil)

	t.Run("minimal", func(t *testing.T) {
		rassert, err := renderer.Render(t, ctx, "", "hello", nil)
		assert.NoError(t, err)
		rassert.NotEmpty()
		rassert.Contains("Hello")
		rassert.Contains("Hallo", "msg", "und", "so")
	})

	t.Run("member", func(t *testing.T) {
		rassert, err := renderer.Render(t, ctx, "", "hello", echo.Map{"Member": pages.PresentHello(domain.TeamMember{
			IsTeamMember: true,
		})})
		assert.NoError(t, err)
		rassert.NotEmpty()
	})

	t.Run("member time", func(t *testing.T) {
		rassert, err := renderer.Render(t, ctx, "", "hello", echo.Map{"Member": pages.PresentHello(domain.TeamMember{
			TeamTime:     time.Now(),
			IsTeamMember: true,
		})})
		assert.NoError(t, err)
		rassert.NotEmpty()
	})

	t.Run("exxtra long name", func(t *testing.T) {
		rassert, err := renderer.Render(t, ctx, "", "hello", echo.Map{"Member": pages.PresentHello(domain.TeamMember{
			Name: "this is an exxtra long name, to see how it looks with it",
		})})
		assert.NoError(t, err)
		rassert.NotEmpty()
	})
}
