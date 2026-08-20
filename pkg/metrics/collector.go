//go:build linux
// +build linux

package metrics

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/disk"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/mackerelio/go-osstat/network"
)

const (
	cpuSampleWindow  = time.Second * 1
	thermalZoneGlob  = "/sys/class/thermal/thermal_zone*/temp"
	milliCelsiusUnit = 1000.0
)

func Collect(hostname string) (SysUsageMessage, error) {
	cpuUsage, err := collectCPU()
	if err != nil {
		return SysUsageMessage{}, err
	}

	memUsage, err := collectMemory()
	if err != nil {
		return SysUsageMessage{}, err
	}

	nicUsage, err := collectNetwork()
	if err != nil {
		return SysUsageMessage{}, err
	}

	dskUsage, err := collectDisk()
	if err != nil {
		return SysUsageMessage{}, err
	}

	tmpUsage, err := collectTemperature()
	if err != nil {
		return SysUsageMessage{}, err
	}

	return SysUsageMessage{
		Name: hostname,
		Cpu:  cpuUsage,
		Mem:  memUsage,
		Nic:  nicUsage,
		Dsk:  dskUsage,
		Tmp:  tmpUsage,
	}, nil
}

func collectCPU() (CPUMessage, error) {
	before, err := cpu.Get()
	if err != nil {
		return CPUMessage{}, err
	}

	<-time.NewTicker(cpuSampleWindow).C

	after, err := cpu.Get()
	if err != nil {
		return CPUMessage{}, err
	}

	total := float64(after.Total - before.Total)

	return CPUMessage{
		System: float64(after.System-before.System) / total * 100,
		User:   float64(after.User-before.User) / total * 100,
		Idle:   float64(after.Idle-before.Idle) / total * 100,
	}, nil
}

func collectMemory() (MemoryMessage, error) {
	memoryStats, err := memory.Get()
	if err != nil {
		return MemoryMessage{}, err
	}

	return MemoryMessage{
		Free:  memoryStats.Free,
		Used:  memoryStats.Used,
		Total: memoryStats.Total,
	}, nil
}

func collectNetwork() ([]NetworkMessage, error) {
	networkStats, err := network.Get()
	if err != nil {
		return nil, err
	}

	nic := make([]NetworkMessage, 0, len(networkStats))
	for _, n := range networkStats {
		nic = append(nic, NetworkMessage{
			Name: n.Name,
			Rx:   n.RxBytes,
			Tx:   n.TxBytes,
		})
	}

	return nic, nil
}

func collectDisk() ([]DiskMessage, error) {
	diskStats, err := disk.Get()
	if err != nil {
		return nil, err
	}

	dsk := make([]DiskMessage, 0, len(diskStats))
	for _, d := range diskStats {
		dsk = append(dsk, DiskMessage{
			Name:   d.Name,
			Reads:  d.ReadsCompleted,
			Writes: d.WritesCompleted,
		})
	}

	return dsk, nil
}

func collectTemperature() ([]TemperatureMessage, error) {
	zones, err := filepath.Glob(thermalZoneGlob)
	if err != nil {
		return nil, err
	}

	tmp := make([]TemperatureMessage, 0, len(zones))
	for _, zonePath := range zones {
		raw, err := os.ReadFile(zonePath)
		if err != nil {
			continue
		}

		milliCelsius, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
		if err != nil {
			continue
		}

		tmp = append(tmp, TemperatureMessage{
			Name:    thermalZoneName(zonePath),
			Celsius: milliCelsius / milliCelsiusUnit,
		})
	}

	return tmp, nil
}

func thermalZoneName(zonePath string) string {
	zoneDir := filepath.Dir(zonePath)

	raw, err := os.ReadFile(filepath.Join(zoneDir, "type"))
	if err != nil {
		return filepath.Base(zoneDir)
	}

	return strings.TrimSpace(string(raw))
}
