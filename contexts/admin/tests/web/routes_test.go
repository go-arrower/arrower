//go:build e2e

package web_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/e2e"
)

func TestScenario_InspectAllRoutes(t *testing.T) {
	t.Parallel()

	page := e2e.Test(t).Goto("/admin/routes")

	table := page.Table("table")
	table.Exists()
	table.NotEmpty()
	table.Headers("Path", "Method", "Name")

	table.Cols(3)
	assert.Greater(t, table.RowCount(), 10)

	table.Row(0).Contains("/")
	table.Row(0).Contains("GET")
	table.Row(0).Col(0).Contains("/")
	table.Row(0).Col(1).Contains("GET")
}
