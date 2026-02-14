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
	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/disk"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/mackerelio/go-osstat/network"
	"github.com/robotjoosen/go-rabbit"
	"github.com/wagslane/go-rabbitmq"
)

type SysUsageMessage struct {
	Name string           `json:"name"`
	Mem  MemoryMessage    `json:"memory"`
	Cpu  CPUMessage       `json:"cpu"`
	Nic  []NetworkMessage `json:"network_interfaces"`
	Dsk  []DiskMessage    `json:"disks"`
}

type MemoryMessage struct {
	Free  uint64 `json:"free"`
	Used  uint64 `json:"used"`
	Total uint64 `json:"total"`
}

type CPUMessage struct {
	System float64 `json:"system"`
	Idle   float64 `json:"idle"`
	User   float64 `json:"user"`
}

type NetworkMessage struct {
	Name string `json:"name"`
	Rx   uint64 `json:"rx_bytes"`
	Tx   uint64 `json:"tx_bytes"`
}

type DiskMessage struct {
	Name   string `json:"name"`
	Reads  uint64 `json:"reads"`
	Writes uint64 `json:"writes"`
}

const maxRetries = 100

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
		messageData := SysUsageMessage{
			Name: hostname,
		}

		before, err := cpu.Get()
		if err != nil {
			slog.Error("failed to get CPU usage", slog.String("err", err.Error()))

			continue
		}
		<-time.NewTicker(time.Second * 1).C
		after, err := cpu.Get()
		if err != nil {
			slog.Error("failed to get CPU usage", slog.String("err", err.Error()))

			continue
		}
		total := float64(after.Total - before.Total)

		messageData.Cpu = CPUMessage{
			System: float64(after.System-before.System) / total * 100,
			User:   float64(after.User-before.User) / total * 100,
			Idle:   float64(after.Idle-before.Idle) / total * 100,
		}

		memoryStats, err := memory.Get()
		if err != nil {
			slog.Error("failed to get memory usage", slog.String("err", err.Error()))

			continue
		}

		messageData.Mem = MemoryMessage{
			Free:  memoryStats.Free,
			Used:  memoryStats.Used,
			Total: memoryStats.Total,
		}

		networkStats, err := network.Get()
		if err != nil {
			slog.Error("failed to get network stats", slog.String("err", err.Error()))

			continue
		}

		messageData.Nic = make([]NetworkMessage, 0, len(networkStats))
		for _, n := range networkStats {
			messageData.Nic = append(messageData.Nic, NetworkMessage{
				Name: n.Name,
				Rx:   n.RxBytes,
				Tx:   n.TxBytes,
			})
		}

		diskStats, err := disk.Get()
		if err != nil {
			slog.Error("failed to get disk stats", slog.String("err", err.Error()))

			continue
		}

		messageData.Dsk = make([]DiskMessage, 0, len(diskStats))
		for _, d := range diskStats {
			messageData.Dsk = append(messageData.Dsk, DiskMessage{
				Name:   d.Name,
				Reads:  d.ReadsCompleted,
				Writes: d.WritesCompleted,
			})
		}

		msg, err := json.Marshal(messageData)
		if err != nil {
			slog.Error("failed to marshal message", "err", err.Error())

			continue
		}

		if err = rabbit.Publish(
			msg,
			e.MessageBusRoutingKey,
			e.MessageBusExchange,
			uuid.New().String(),
			p,
		); err != nil {
			slog.Error("failed to publish message", slog.String("err", err.Error()))

			continue
		}

		<-time.NewTicker(time.Second * 5).C
	}
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
