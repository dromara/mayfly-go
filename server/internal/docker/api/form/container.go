package form

import "mayfly-go/pkg/model"

type ContainerSave struct {
	model.ExtraData

	Id           uint64   `json:"id"`
	Addr         string   `json:"addr" binding:"required"`
	Name         string   `json:"name" binding:"required"`
	Remark       string   `json:"remark"`
	TagCodePaths []string `json:"tagCodePaths" binding:"required"`
}

type ContainerOp struct {
	ContainerId string `json:"containerId" binding:"required"`
}

type ContainerCreate struct {
	ContainerID   string          `json:"containerId"`
	Name          string          `json:"name" validate:"required"`
	Image         string          `json:"image" validate:"required"`
	ForcePull     bool            `json:"forcePull"`
	ExposedPorts  []ExposedPort   `json:"exposedPorts"`
	Tty           bool            `json:"tty"`
	OpenStdin     bool            `json:"openStdin"`
	Cmd           []string        `json:"cmd"`
	Entrypoint    []string        `json:"entrypoint"`
	CPUShares     int64           `json:"cpuShares"`
	NanoCPUs      float64         `json:"nanoCpus"`
	Memory        float64         `json:"memory"`
	CapAdd        []string        `json:"capAdd"`
	ShmSize       float64         `json:"shmSize"`
	NetworkMode   string          `json:"networkMode"`
	Privileged    bool            `json:"privileged"`
	AutoRemove    bool            `json:"autoRemove"`
	RestartPolicy string          `json:"restartPolicy"`
	Volumes       []Volume        `json:"volumes"`
	Devices       []DeviceRequest `json:"devices"`
	Runtime       string          `json:"runtime"`
	Labels        []string        `json:"labels"`
	Envs          []string        `json:"envs"`
}

type ExposedPort struct {
	HostIP        string `json:"hostIP"`
	HostPort      string `json:"hostPort"`
	ContainerPort string `json:"containerPort"`
	Protocol      string `json:"protocol"`
}

type Volume struct {
	Type         string `json:"type"`
	HostDir      string `json:"hostDir"`
	ContainerDir string `json:"containerDir"`
	Mode         string `json:"mode"`
}

type DeviceRequest struct {
	Driver       string            `json:"driver"`       // Name of device driver
	Count        int               `json:"count"`        // Number of devices to request (-1 = All)
	DeviceIDs    []string          `json:"deviceIds"`    // List of device IDs as recognizable by the device driver
	Capabilities []string          `json:"capabilities"` // An OR list of AND lists of device capabilities (e.g. "gpu")
	Options      map[string]string `json:"Options"`      // Options to pass onto the device driver
}
