package arrower

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/go-arrower/arrower/secret"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// Config is a structure used for service configuration.
// It is intended to be mapped by viper.
type Config struct {
	OrganisationName string `mapstructure:"organisation_name"`
	ApplicationName  string `mapstructure:"application_name"`
	InstanceName     string `mapstructure:"instance_name"`

	Environment Environment `mapstructure:"environment"`

	HTTP     HTTP     `mapstructure:"http"`
	Postgres Postgres `mapstructure:"postgres"`
	OTEL     OTEL     `mapstructure:"otel"`
}

const (
	LocalEnv       Environment = "local"
	TestEnv        Environment = "test"
	DevelopmentEnv Environment = "dev"
	ProductionEnv  Environment = "prod"
)

// Environments is the list of all supported environments.
func Environments() []Environment {
	return []Environment{LocalEnv, TestEnv, DevelopmentEnv, ProductionEnv}
}

type Environment string

type (
	HTTP struct {
		Port                  int           `mapstructure:"port"                    json:"port"`
		CookieSecret          secret.Secret `mapstructure:"cookie_secret,squash"    json:"-"`
		StatusEndpointEnabled bool          `mapstructure:"status_endpoint_enabled" json:"-"`
		StatusEndpointPort    int           `mapstructure:"status_endpoint_port"    json:"-"`
	}

	Postgres struct {
		User     string        `mapstructure:"user"            json:"user"`
		Password secret.Secret `mapstructure:"password,squash" json:"-"`
		Database string        `mapstructure:"database"        json:"database"`
		Host     string        `mapstructure:"host"            json:"host"`
		Port     int           `mapstructure:"port"            json:"port"`
		SSLMode  string        `mapstructure:"ssl_mode"        json:"sslMode"`
		MaxConns int           `mapstructure:"max_conns"       json:"maxConns"`
	}

	OTEL struct {
		Host     string `mapstructure:"host"     json:"host"`
		Port     int    `mapstructure:"port"     json:"port"`
		Hostname string `mapstructure:"hostname" json:"hostname"`
	}
)

// DefaultViper returns a new viper instance with all default values set.
func DefaultViper() *viper.Viper {
	vip := viper.New()

	vip.SetDefault("organisation_name", "")
	vip.SetDefault("application_name", "")
	vip.SetDefault("instance_name", "")

	vip.SetDefault("environment", "local")

	vip.SetDefault("http.port", 8080)
	vip.SetDefault("http.cookie_secret", "secret")
	vip.SetDefault("http.status_endpoint_enabled", true)
	vip.SetDefault("http.status_endpoint_port", 2223) // todo find better port

	vip.SetDefault("postgres.user", "arrower")
	vip.SetDefault("postgres.password", "secret")
	vip.SetDefault("postgres.database", "arrower")
	vip.SetDefault("postgres.host", "localhost")
	vip.SetDefault("postgres.port", 5432)
	vip.SetDefault("postgres.ssl_mode", "disable")
	vip.SetDefault("postgres.max_conns", 10)

	vip.SetDefault("otel.host", "localhost")
	vip.SetDefault("otel.port", 4317)
	vip.SetDefault("otel.hostname", "")

	return vip
}

func AllowedEnvironmentHookFunc() mapstructure.DecodeHookFuncType {
	return func(_ reflect.Type, t reflect.Type, data any) (interface{}, error) {
		if t != reflect.TypeOf(Environment("")) {
			return data, nil
		}

		env := Environments()
		if slices.Contains(env, Environment(data.(string))) {
			return data, nil
		}

		e := make([]string, 0, len(env))
		for _, env := range env {
			e = append(e, string(env))
		}

		return data, fmt.Errorf("value is not allowed, use one of: %s", strings.Join(e, ", ")) //nolint:err113,lll // accept dynamic error
	}
}
