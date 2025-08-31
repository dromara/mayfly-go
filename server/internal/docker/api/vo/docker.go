package vo

import "time"

type Container struct {
	ContainerId string    `json:"containerId"`
	Name        string    `json:"name"`
	ImageId     string    `json:"imageId"`
	ImageName   string    `json:"imageName"`
	State       string    `json:"state"`
	Status      string    `json:"status"`
	CreateTime  time.Time `json:"createTime"`
	Networks    []string  `json:"networks"`
	Ports       []string  `json:"ports"`
}

type ContainerStats struct {
	ContainerId string `json:"containerId"`

	CPUTotalUsage uint64  `json:"cpuTotalUsage"`
	SystemUsage   uint64  `json:"systemUsage"`
	CPUPercent    float64 `json:"cpuPercent"`
	PercpuUsage   int     `json:"percpuUsage"`

	MemoryCache   uint64  `json:"memoryCache"`
	MemoryUsage   uint64  `json:"memoryUsage"`
	MemoryLimit   uint64  `json:"memoryLimit"`
	MemoryPercent float64 `json:"memoryPercent"`
}

type Image struct {
	Id string `json:"id"`

	Size       int64     `json:"size"`
	CreateTime time.Time `json:"createTime"`
	Tags       []string  `json:"tags"`
	IsUse      bool      `json:"isUse"`
}
