//go:build integration

package setting_test

import (
	"os"
	"testing"

	"github.com/go-arrower/arrower/setting"
	"github.com/go-arrower/arrower/tests"
)

var pgHandler *tests.PostgresDocker

func TestMain(m *testing.M) {
	pgHandler = tests.GetPostgresDockerForIntegrationTestingInstance()

	//
	// Run tests
	code := m.Run()

	pgHandler.Cleanup()
	os.Exit(code)
}

func TestPostgresSettings(t *testing.T) {
	t.Parallel()

	setting.TestSuite(t, func() setting.Settings { return setting.NewPostgresSettings(pgHandler.NewTestDatabase()) })
}
