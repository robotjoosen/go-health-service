//go:build linux
// +build linux

package main

import (
	"log/slog"
	"os"

	"github.com/robotjoosen/go-health-service/pkg/env"
)

const (
	modeDevelopment = "DEV"

	defaultMode          = modeDevelopment
	defaultLogLevel      = "INFO"
	defaultMessageBusURL = "amqp://guest:guest@localhost:5672"
	defaultRoutingKey    = "health.ping"
	defaultExchange      = "health"
)

type Environment struct {
	Mode                 string     `mapstructure:"MODE"`
	LogLevel             slog.Level `mapstructure:"LOG_LEVEL"`
	MessagebusURL        string     `mapstructure:"MESSAGE_BUS_URL"`
	MessageBusExchange   string     `mapstructure:"MESSAGE_BUS_EXCHANGE"`
	MessageBusRoutingKey string     `mapstructure:"MESSAGE_BUS_ROUTING_KEY"`
}

func initLog(level slog.Level) {
	hostname, err := os.Hostname()
	if err != nil {
		os.Exit(1)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})).
		With(
			slog.String("hostname", hostname),
		))
}

func loadEnv() Environment {
	e, err := env.Load[Environment](map[string]any{
		"MODE":                    defaultMode,
		"LOG_LEVEL":               defaultLogLevel,
		"MESSAGE_BUS_URL":         defaultMessageBusURL,
		"MESSAGE_BUS_EXCHANGE":    defaultExchange,
		"MESSAGE_BUS_ROUTING_KEY": defaultRoutingKey,
	})
	if err != nil {
		slog.Error("failed to load environment", "err", err.Error())

		os.Exit(1)
	}

	return e
}
