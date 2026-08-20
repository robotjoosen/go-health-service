//go:build linux
// +build linux

package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/robotjoosen/go-health-service/pkg/metrics"
	"github.com/robotjoosen/go-rabbit"
	"github.com/wagslane/go-rabbitmq"
)

const (
	maxRetries      = 100
	publishInterval = time.Second * 5
)

func main() {
	e := loadEnv()
	initLog(e.LogLevel)

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	conn := connectMessageBus(e.MessagebusURL)
	p, err := rabbit.NewPublisher(conn, e.MessageBusExchange)
	if err != nil {
		panic(err)
	}

	for {
		messageData, err := metrics.Collect(hostname)
		if err != nil {
			slog.Error("failed to collect system usage", slog.String("err", err.Error()))

			continue
		}

		if err = publishSysUsage(e, p, messageData); err != nil {
			slog.Error("failed to publish message", slog.String("err", err.Error()))

			continue
		}

		<-time.NewTicker(publishInterval).C
	}
}

func publishSysUsage(e Environment, p *rabbitmq.Publisher, messageData metrics.SysUsageMessage) error {
	msg, err := json.Marshal(messageData)
	if err != nil {
		return err
	}

	return rabbit.Publish(
		msg,
		e.MessageBusRoutingKey,
		e.MessageBusExchange,
		uuid.New().String(),
		p,
	)
}

func connectMessageBus(u string) *rabbitmq.Conn {
	mbu, err := url.Parse(u)
	if err != nil {
		panic(err)
	}

	retries := 0
	for {
		if retries >= maxRetries {
			panic(errors.New("cannot connect to message bus"))
		}

		if _, err := net.DialTimeout("tcp", mbu.Host, 1*time.Second); err != nil {
			retries++

			<-time.NewTicker(2 * time.Second).C

			continue
		}

		break
	}

	conn, err := rabbit.NewConnection(u)
	if err != nil {
		panic(err)
	}

	return conn
}
