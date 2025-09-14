package models

import "time"

type Container struct {
	ID      string            `json:"id"`
	Names   []string          `json:"names"`
	Image   string            `json:"image"`
	ImageID string            `json:"imageId"`
	Command string            `json:"command"`
	Created int64             `json:"created"`
	Ports   []Port            `json:"ports"`
	Labels  map[string]string `json:"labels"`
	State   string            `json:"state"`
	Status  string            `json:"status"`
	Mounts  []Mount           `json:"mounts"`
}

type Port struct {
	IP          string `json:"ip"`
	PrivatePort uint16 `json:"privatePort"`
	PublicPort  uint16 `json:"publicPort"`
	Type        string `json:"type"`
}

type Mount struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Driver      string `json:"driver"`
	Mode        string `json:"mode"`
	RW          bool   `json:"rw"`
	Propagation string `json:"propagation"`
}

type ContainerStats struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	CPUUsage float64   `json:"cpuUsage"`
	Memory   Memory    `json:"memory"`
	Network  Network   `json:"network"`
	BlockIO  BlockIO   `json:"blockIO"`
	Time     time.Time `json:"time"`
}

type Memory struct {
	Usage    uint64  `json:"usage"`
	Limit    uint64  `json:"limit"`
	Percent  float64 `json:"percent"`
}

type Network struct {
	RxBytes   uint64 `json:"rxBytes"`
	TxBytes   uint64 `json:"txBytes"`
	RxPackets uint64 `json:"rxPackets"`
	TxPackets uint64 `json:"txPackets"`
}

type BlockIO struct {
	ReadBytes  uint64 `json:"readBytes"`
	WriteBytes uint64 `json:"writeBytes"`
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Stream    string    `json:"stream"`
}

type ContainerAction struct {
	Action string `json:"action"`
}

type Image struct {
	ID       string   `json:"id"`
	RepoTags []string `json:"repoTags"`
	Size     int64    `json:"size"`
	Created  int64    `json:"created"`
}

type PullImageRequest struct {
	ImageName string `json:"imageName"`
}