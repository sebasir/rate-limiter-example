package config

import (
	"fmt"
	"github.com/kelseyhightower/envconfig"
)

type AppConfig struct {
	Debug                int    `envconfig:"DEBUG" default:"1"`
	RateLimiterHttpPort  int    `envconfig:"RATE_LIMITER_HTTP_PORT" default:"8080"`
	RedisHost            string `envconfig:"REDIS_HOST" default:"localhost"`
	RedisPort            int    `envconfig:"REDIS_EXPOSED_PORT" default:"6379"`
	NotificationHost     string `envconfig:"NOTIFICATION_HOST" default:"localhost"`
	NotificationHTTPPort int    `envconfig:"NOTIFICATION_HTTP_PORT" default:"8280"`
	NotificationGRPCPort int    `envconfig:"NOTIFICATION_GRPC_PORT" default:"8281"`
}

func (lc *AppConfig) Load() error {
	if err := envconfig.Process("", lc); err != nil {
		return err
	}

	return nil
}

func FormatAddress(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}
