package domain

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
