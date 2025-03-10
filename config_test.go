package arrower_test

import (
	"testing"

	"github.com/go-arrower/arrower"
	"github.com/stretchr/testify/assert"
)

func TestDefaultViper(t *testing.T) {
	t.Parallel()

	vip := arrower.DefaultViper()
	assert.NotEmpty(t, vip)

	// This test enforces the default values, so whenever they change,
	// make sure to also update the example config file!

	assert.Empty(t, vip.Get("organisation_name"))
	assert.Empty(t, vip.Get("application_name"))
	assert.Empty(t, vip.Get("instance_name"))

	assert.Equal(t, arrower.LocalEnv, arrower.Environment(vip.GetString("environment")))

	assert.Equal(t, 8080, vip.GetInt("http.port"))
	assert.Equal(t, "secret", vip.GetString("http.cookie_secret"))
	assert.True(t, vip.GetBool("http.status_endpoint_enabled"))
	assert.Equal(t, 2223, vip.GetInt("http.status_endpoint_port"))

	assert.Equal(t, "arrower", vip.GetString("postgres.user"))
	assert.Equal(t, "secret", vip.GetString("postgres.password"))
	assert.Equal(t, "arrower", vip.GetString("postgres.database"))
	assert.Equal(t, "localhost", vip.GetString("postgres.host"))
	assert.Equal(t, 5432, vip.GetInt("postgres.port"))
	assert.Equal(t, "disable", vip.GetString("postgres.ssl_mode"))
	assert.Equal(t, 10, vip.GetInt("postgres.max_conns"))

	assert.Equal(t, "localhost", vip.GetString("otel.host"))
	assert.Equal(t, 4317, vip.GetInt("otel.port"))
	assert.Equal(t, "", vip.GetString("otel.hostname"))
}

func TestDefaultViper_CustomTypes(t *testing.T) {
	t.Parallel()

	t.Run("invalid environment", func(t *testing.T) {
		t.Parallel()

		vip := arrower.DefaultViper()
		vip.SetConfigFile("./testdata/config/invalid-config.yaml")
		err := vip.ReadInConfig()
		assert.NoError(t, err)

		conf := arrower.Config{}

		err = vip.Unmarshal(&conf)
		assert.Error(t, err, "should fail when using unsupported enum values")
		assert.Contains(t, err.Error(), "use one of: ", "error message should list out all accepted environments")
	})

	t.Run("valid environment", func(t *testing.T) {
		t.Parallel()

		vip := arrower.DefaultViper()
		vip.SetConfigFile("./testdata/config/test-config.yaml")
		err := vip.ReadInConfig()
		assert.NoError(t, err)

		conf := arrower.Config{}

		err = vip.Unmarshal(&conf)
		assert.NoError(t, err)
		assert.Equal(t, arrower.TestEnv, conf.Environment)
	})

	t.Run("unmarshal arrower secret", func(t *testing.T) {
		t.Parallel()

		vip := arrower.DefaultViper()
		vip.SetConfigFile("./testdata/config/test-config.yaml")
		err := vip.ReadInConfig()
		assert.NoError(t, err)

		conf := arrower.Config{}

		err = vip.Unmarshal(&conf)
		assert.NoError(t, err)
		assert.Equal(t, "my-cookie-secret", conf.HTTP.CookieSecret.Secret())
		assert.Equal(t, "my-db-secret", conf.Postgres.Password.Secret())
	})

	t.Run("custom config", func(t *testing.T) {
		t.Parallel()

		type MyConfig struct {
			SomeStructField struct{ A string }
			arrower.Config  `mapstructure:",squash"`
		}

		vip := arrower.DefaultViper()
		vip.SetConfigFile("./testdata/config/test-config.yaml")
		err := vip.ReadInConfig()
		assert.NoError(t, err)

		conf := MyConfig{}

		err = vip.Unmarshal(&conf)
		assert.NoError(t, err)
		assert.Equal(t, "my-cookie-secret", conf.HTTP.CookieSecret.Secret())
		assert.Equal(t, "my-db-secret", conf.Postgres.Password.Secret())
	})
}
