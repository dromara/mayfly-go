package api

import (
	"context"
	"fmt"
	"io"
	"mayfly-go/internal/docker/api/form"
	"mayfly-go/internal/docker/api/vo"
	"mayfly-go/internal/docker/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/ws"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gorilla/websocket"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cast"
)

type Container struct {
}

func (d *Container) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", d.GetContainers),
		req.NewGet("/stats", d.GetContainersStats),
		req.NewPost("/stop", d.ContainerStop).Log(req.NewLogSaveI(imsg.LogDockerContainerStop)),
		req.NewPost("/remove", d.ContainerRemove).Log(req.NewLogSaveI(imsg.LogDockerContainerRemove)),
		req.NewPost("/restart", d.ContainerRestart).Log(req.NewLogSaveI(imsg.LogDockerContainerStop)),
		req.NewPost("/create", d.ContainerCreate).Log(req.NewLogSaveI(imsg.LogDockerContainerCreate)),

		req.NewGet("/exec", d.ContainerExecAttach).NoRes(),
		req.NewGet("/logs", d.ContainerLogs).NoRes(),
	}

	return req.NewConfs("docker/:id/containers", reqs[:]...)
}

func (d *Container) GetContainers(rc *req.Ctx) {
	cli := GetCli(rc)
	cs, err := cli.ContainerList()
	biz.ErrIsNil(err)

	rc.ResData = collx.ArrayMap(cs, func(val container.Summary) vo.Container {
		c := vo.Container{
			ContainerId: val.ID,
			Name:        val.Names[0][1:],
			ImageId:     strings.Split(val.ImageID, ":")[1],
			ImageName:   val.Image,
			State:       val.State,
			Status:      val.Status,
			CreateTime:  time.Unix(val.Created, 0),
			Ports:       transPortToStr(val.Ports),
		}

		if val.NetworkSettings != nil && len(val.NetworkSettings.Networks) > 0 {
			if ns := val.NetworkSettings.Networks; len(ns) > 0 {
				networks := make([]string, 0, len(ns))
				for key := range ns {
					networks = append(networks, ns[key].IPAddress)
				}
				sort.Strings(networks)
				c.Networks = networks
			}
		}

		return c
	})
}

func (d *Container) GetContainersStats(rc *req.Ctx) {
	cli := GetCli(rc)
	cs, err := cli.ContainerList()
	biz.ErrIsNil(err)

	var wg sync.WaitGroup
	wg.Add(len(cs))

	var mu sync.Mutex
	allStats := make([]vo.ContainerStats, 0)
	for _, c := range cs {
		go func(item container.Summary) {
			defer wg.Done()
			if item.State != "running" {
				return
			}

			stats, err := cli.ContainerStats(c.ID)
			if err != nil {
				logx.Error("get docker container stats err", err)
				return
			}
			var cs vo.ContainerStats
			cs.ContainerId = c.ID

			cs.CPUTotalUsage = stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage
			cs.SystemUsage = stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage
			cs.CPUPercent = calculateCPUPercentUnix(stats)
			cs.PercpuUsage = len(stats.CPUStats.CPUUsage.PercpuUsage)

			cs.MemoryCache = stats.MemoryStats.Stats["cache"]
			cs.MemoryUsage = stats.MemoryStats.Usage
			cs.MemoryLimit = stats.MemoryStats.Limit

			cs.MemoryPercent = calculateMemPercentUnix(stats.MemoryStats)

			mu.Lock()
			allStats = append(allStats, cs)
			mu.Unlock()
		}(c)
	}

	wg.Wait()
	rc.ResData = allStats
}

func (d *Container) ContainerCreate(rc *req.Ctx) {
	containerCreate := &form.ContainerCreate{}
	biz.ErrIsNil(rc.BindJSON(containerCreate))

	rc.ReqParam = containerCreate

	cli := GetCli(rc)

	config, hostConfig, networkConfig, err := loadConfigInfo(true, containerCreate, nil)
	biz.ErrIsNil(err)

	ctx := rc.MetaCtx
	con, err := cli.DockerClient.ContainerCreate(ctx, config, hostConfig, networkConfig, &v1.Platform{}, containerCreate.Name)

	if err != nil {
		_ = cli.DockerClient.ContainerRemove(ctx, containerCreate.Name, container.RemoveOptions{RemoveVolumes: true, Force: true})
		panic(errorx.NewBizf("create container failed, err: %v", err))
	}

	logx.Infof("create container %s successful! now check if the container is started and delete the container information if it is not.", containerCreate.Name)

	if err := cli.DockerClient.ContainerStart(ctx, con.ID, container.StartOptions{}); err != nil {
		_ = cli.DockerClient.ContainerRemove(ctx, containerCreate.Name, container.RemoveOptions{RemoveVolumes: true, Force: true})
		panic(errorx.NewBizf("create successful but start failed, err: %v", err))
	}
}

func (d *Container) ContainerStop(rc *req.Ctx) {
	containerOp := &form.ContainerOp{}
	biz.ErrIsNil(rc.BindJSON(containerOp))

	cli := GetCli(rc)
	rc.ReqParam = collx.Kvs("addr", cli.Server.Addr, "containerId", containerOp.ContainerId)

	biz.ErrIsNil(cli.ContainerStop(containerOp.ContainerId))
}

func (d *Container) ContainerRemove(rc *req.Ctx) {
	containerOp := &form.ContainerOp{}
	biz.ErrIsNil(rc.BindJSON(containerOp))

	cli := GetCli(rc)
	rc.ReqParam = collx.Kvs("addr", cli.Server.Addr, "containerId", containerOp.ContainerId)

	biz.ErrIsNil(cli.ContainerRemove(containerOp.ContainerId))
}

func (d *Container) ContainerRestart(rc *req.Ctx) {
	containerOp := &form.ContainerOp{}
	biz.ErrIsNil(rc.BindJSON(containerOp))

	cli := GetCli(rc)
	rc.ReqParam = collx.Kvs("addr", cli.Server.Addr, "containerId", containerOp.ContainerId)

	biz.ErrIsNil(cli.ContainerRestart(containerOp.ContainerId))
}

func (d *Container) ContainerLogs(rc *req.Ctx) {
	wsConn, err := ws.Upgrader.Upgrade(rc.GetWriter(), rc.GetRequest(), nil)
	defer func() {
		if wsConn != nil {
			if err := recover(); err != nil {
				wsConn.WriteMessage(websocket.TextMessage, []byte(anyx.ToString(err)))
			}
			wsConn.Close()
		}
	}()
	biz.ErrIsNilAppendErr(err, "Upgrade websocket fail: %s")

	cli := GetCli(rc)
	ctx, cancel := context.WithCancel(rc.MetaCtx)
	defer cancel()

	// 设置日志选项
	logOptions := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     rc.Query("follow") == "1",
		Timestamps: false,
		Since:      rc.Query("since"),
	}
	tail := rc.QueryInt("tail")
	if tail > 0 {
		logOptions.Tail = cast.ToString(tail)
	}

	logs, err := cli.DockerClient.ContainerLogs(ctx, rc.Query("containerId"), logOptions)
	biz.ErrIsNil(err)
	defer logs.Close()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, _, err := wsConn.ReadMessage()
				// 读取ws关闭错误，取消日志输出
				if err != nil {
					cancel()
					return
				}
			}
		}
	}()

	buf := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := logs.Read(buf)
			if err != nil {
				if err != io.EOF && err != context.Canceled {
					logx.ErrorTrace("Read container log error", err)
				}
				wsConn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if !utf8.Valid(buf[:n]) {
				continue
			}
			if err := wsConn.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
				logx.ErrorTrace("Write container log error", err)
				return
			}
		}
	}
}

func (d *Container) ContainerExecAttach(rc *req.Ctx) {
	wsConn, err := ws.Upgrader.Upgrade(rc.GetWriter(), rc.GetRequest(), nil)
	defer func() {
		if wsConn != nil {
			if err := recover(); err != nil {
				wsConn.WriteMessage(websocket.TextMessage, []byte(anyx.ToString(err)))
			}
			wsConn.Close()
		}
	}()
	biz.ErrIsNilAppendErr(err, "Upgrade websocket fail: %s")
	wsConn.WriteMessage(websocket.TextMessage, []byte("Connecting to container..."))

	cli := GetCli(rc)
	cols := rc.QueryIntDefault("cols", 80)
	rows := rc.QueryIntDefault("rows", 32)

	err = cli.ContainerAttach(rc.Query("containerId"), wsConn, rows, cols)
	if err != nil {
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error attaching to container: %s", err.Error())))
	}
}

func (d *Container) ContainerProxy(rc *req.Ctx) {
	// 获取 containerId 和剩余路径
	pathParts := strings.Split(rc.GetRequest().URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(rc.GetWriter(), "Invalid path", http.StatusBadRequest)
		return
	}

	containerID := pathParts[2]
	remainingPath := strings.Join(pathParts[3:], "/")

	cli := GetCli(rc)
	ctx := rc.MetaCtx
	containerJSON, err := cli.DockerClient.ContainerInspect(ctx, containerID)
	biz.ErrIsNil(err)

	// 获取容器的网络信息
	networkSettings := containerJSON.NetworkSettings
	if networkSettings == nil || len(networkSettings.Networks) == 0 {
		panic(errorx.NewBiz("container network settings not found"))
	}

	// 假设我们使用第一个网络的IP地址
	var containerIP string
	for _, network := range networkSettings.Networks {
		containerIP = network.IPAddress
		break
	}

	// 获取容器的端口映射
	var containerPort string
	portBindings := containerJSON.HostConfig.PortBindings
	if len(portBindings) > 0 {
		for _, bindings := range portBindings {
			if len(bindings) > 0 {
				containerPort = bindings[0].HostPort
				break
			}
		}
	}

	if containerIP == "" || containerPort == "" {
		panic(errorx.NewBiz("container IP or port not found"))
	}

	// 构建目标URL
	targetURL, err := url.Parse(fmt.Sprintf("http://%s:%s", containerIP, containerPort))
	biz.ErrIsNil(err)

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 修改请求头中的主机地址和路径
	proxy.Director = func(req *http.Request) {
		req.Header.Set("X-Real-IP", req.RemoteAddr)
		req.Header.Set("X-Forwarded-For", req.RemoteAddr)
		req.Header.Set("X-Forwarded-Proto", "http")
		req.Host = targetURL.Host

		// 重写请求路径
		req.URL.Path = "/" + remainingPath
		req.URL.RawPath = "/" + remainingPath
	}

	// 处理请求
	proxy.ServeHTTP(rc.GetWriter(), rc.GetRequest())
}

func calculateCPUPercentUnix(stats container.StatsResponse) float64 {
	cpuPercent := 0.0
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage) - float64(stats.PreCPUStats.SystemUsage)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * 100.0
		if len(stats.CPUStats.CPUUsage.PercpuUsage) != 0 {
			cpuPercent = cpuPercent * float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
		}
	}
	return cpuPercent
}

func calculateMemPercentUnix(memStats container.MemoryStats) float64 {
	memPercent := 0.0
	memUsage := float64(memStats.Usage)
	memLimit := float64(memStats.Limit)
	if memUsage > 0.0 && memLimit > 0.0 {
		memPercent = (memUsage / memLimit) * 100.0
	}
	return memPercent
}

func calculateBlockIO(blkio container.BlkioStats) (blkRead float64, blkWrite float64) {
	for _, bioEntry := range blkio.IoServiceBytesRecursive {
		switch strings.ToLower(bioEntry.Op) {
		case "read":
			blkRead = (blkRead + float64(bioEntry.Value)) / 1024 / 1024
		case "write":
			blkWrite = (blkWrite + float64(bioEntry.Value)) / 1024 / 1024
		}
	}
	return
}

func calculateNetwork(network map[string]container.NetworkStats) (float64, float64) {
	var rx, tx float64

	for _, v := range network {
		rx += float64(v.RxBytes) / 1024
		tx += float64(v.TxBytes) / 1024
	}
	return rx, tx
}

func transPortToStr(ports []container.Port) []string {
	var (
		ipv4Ports []container.Port
		ipv6Ports []container.Port
	)
	for _, port := range ports {
		if strings.Contains(port.IP, ":") {
			ipv6Ports = append(ipv6Ports, port)
		} else {
			ipv4Ports = append(ipv4Ports, port)
		}
	}
	list1 := simplifyPort(ipv4Ports)
	list2 := simplifyPort(ipv6Ports)
	return append(list1, list2...)
}

func simplifyPort(ports []container.Port) []string {
	var datas []string
	if len(ports) == 0 {
		return datas
	}
	if len(ports) == 1 {
		ip := ""
		if len(ports[0].IP) != 0 {
			ip = ports[0].IP + ":"
		}
		itemPortStr := fmt.Sprintf("%s%v/%s", ip, ports[0].PrivatePort, ports[0].Type)
		if ports[0].PublicPort != 0 {
			itemPortStr = fmt.Sprintf("%s%v->%v/%s", ip, ports[0].PublicPort, ports[0].PrivatePort, ports[0].Type)
		}
		datas = append(datas, itemPortStr)
		return datas
	}

	sort.Slice(ports, func(i, j int) bool {
		return ports[i].PrivatePort < ports[j].PrivatePort
	})
	start := ports[0]

	for i := 1; i < len(ports); i++ {
		if ports[i].PrivatePort != ports[i-1].PrivatePort+1 || ports[i].IP != ports[i-1].IP || ports[i].PublicPort != ports[i-1].PublicPort+1 || ports[i].Type != ports[i-1].Type {
			if ports[i-1].PrivatePort == start.PrivatePort {
				itemPortStr := fmt.Sprintf("%s:%v/%s", start.IP, start.PrivatePort, start.Type)
				if start.PublicPort != 0 {
					itemPortStr = fmt.Sprintf("%s:%v->%v/%s", start.IP, start.PublicPort, start.PrivatePort, start.Type)
				}
				if len(start.IP) == 0 {
					itemPortStr = strings.TrimPrefix(itemPortStr, ":")
				}
				datas = append(datas, itemPortStr)
			} else {
				itemPortStr := fmt.Sprintf("%s:%v-%v/%s", start.IP, start.PrivatePort, ports[i-1].PrivatePort, start.Type)
				if start.PublicPort != 0 {
					itemPortStr = fmt.Sprintf("%s:%v-%v->%v-%v/%s", start.IP, start.PublicPort, ports[i-1].PublicPort, start.PrivatePort, ports[i-1].PrivatePort, start.Type)
				}
				if len(start.IP) == 0 {
					itemPortStr = strings.TrimPrefix(itemPortStr, ":")
				}
				datas = append(datas, itemPortStr)
			}
			start = ports[i]
		}
		if i == len(ports)-1 {
			if ports[i].PrivatePort == start.PrivatePort {
				itemPortStr := fmt.Sprintf("%s:%v/%s", start.IP, start.PrivatePort, start.Type)
				if start.PublicPort != 0 {
					itemPortStr = fmt.Sprintf("%s:%v->%v/%s", start.IP, start.PublicPort, start.PrivatePort, start.Type)
				}
				if len(start.IP) == 0 {
					itemPortStr = strings.TrimPrefix(itemPortStr, ":")
				}
				datas = append(datas, itemPortStr)
			} else {
				itemPortStr := fmt.Sprintf("%s:%v-%v/%s", start.IP, start.PrivatePort, ports[i].PrivatePort, start.Type)
				if start.PublicPort != 0 {
					itemPortStr = fmt.Sprintf("%s:%v-%v->%v-%v/%s", start.IP, start.PublicPort, ports[i].PublicPort, start.PrivatePort, ports[i].PrivatePort, start.Type)
				}
				if len(start.IP) == 0 {
					itemPortStr = strings.TrimPrefix(itemPortStr, ":")
				}
				datas = append(datas, itemPortStr)
			}
		}
	}
	return datas
}

func checkPortStats(ports []form.ExposedPort) (nat.PortMap, error) {
	portMap := make(nat.PortMap)
	if len(ports) == 0 {
		return portMap, nil
	}
	for _, port := range ports {
		if strings.Contains(port.ContainerPort, "-") {
			if !strings.Contains(port.HostPort, "-") {
				return portMap, errorx.NewBiz("exposed port error")
			}

			hostStart := cast.ToInt(strings.Split(port.HostPort, "-")[0])
			hostEnd := cast.ToInt(strings.Split(port.HostPort, "-")[1])
			containerStart := cast.ToInt(strings.Split(port.ContainerPort, "-")[0])
			containerEnd := cast.ToInt(strings.Split(port.ContainerPort, "-")[1])
			if (hostEnd-hostStart) <= 0 || (containerEnd-containerStart) <= 0 {
				return portMap, errorx.NewBiz("exposed port error")
			}
			if (containerEnd - containerStart) != (hostEnd - hostStart) {
				return portMap, errorx.NewBiz("exposed port error")
			}
			for i := 0; i <= hostEnd-hostStart; i++ {
				bindItem := nat.PortBinding{HostPort: strconv.Itoa(hostStart + i), HostIP: port.HostIP}
				portMap[nat.Port(fmt.Sprintf("%d/%s", containerStart+i, port.Protocol))] = []nat.PortBinding{bindItem}
			}
		} else {
			portItem := 0
			if strings.Contains(port.HostPort, "-") {
				portItem = cast.ToInt(strings.Split(port.HostPort, "-")[0])
			} else {
				portItem = cast.ToInt(port.HostPort)
			}
			bindItem := nat.PortBinding{HostPort: cast.ToString(portItem), HostIP: port.HostIP}
			portMap[nat.Port(fmt.Sprintf("%s/%s", port.ContainerPort, port.Protocol))] = []nat.PortBinding{bindItem}
		}
	}
	return portMap, nil
}

func loadConfigInfo(isCreate bool, req *form.ContainerCreate, oldContainer *types.ContainerJSON) (*container.Config, *container.HostConfig, *network.NetworkingConfig, error) {
	var config container.Config
	var hostConf container.HostConfig
	if !isCreate {
		config = *oldContainer.Config
		hostConf = *oldContainer.HostConfig
	}
	var networkConf network.NetworkingConfig

	portMap, err := checkPortStats(req.ExposedPorts)
	if err != nil {
		return nil, nil, nil, err
	}
	exposed := make(nat.PortSet)
	for port := range portMap {
		exposed[port] = struct{}{}
	}
	config.Image = req.Image
	config.Cmd = req.Cmd
	config.Entrypoint = req.Entrypoint
	config.Env = req.Envs
	config.Labels = stringsToMap(req.Labels)
	config.ExposedPorts = exposed
	config.OpenStdin = req.OpenStdin
	config.Tty = req.Tty

	hostConf.Privileged = req.Privileged
	hostConf.AutoRemove = req.AutoRemove
	hostConf.CPUShares = req.CPUShares
	hostConf.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyMode(req.RestartPolicy)}
	if req.RestartPolicy == "on-failure" {
		hostConf.RestartPolicy.MaximumRetryCount = 5
	}
	hostConf.NanoCPUs = int64(req.NanoCPUs * 1000000000)
	hostConf.Memory = int64(req.Memory * 1024 * 1024 * 1024)
	hostConf.MemorySwap = 0
	hostConf.PortBindings = portMap
	hostConf.Binds = []string{}
	hostConf.Mounts = []mount.Mount{}
	hostConf.ShmSize = int64(req.ShmSize * 1024 * 1024 * 1024)
	hostConf.CapAdd = req.CapAdd
	hostConf.NetworkMode = container.NetworkMode(req.NetworkMode)

	if len(req.Devices) > 0 {
		hostConf.DeviceRequests = collx.ArrayMap(req.Devices, func(val form.DeviceRequest) container.DeviceRequest {
			return container.DeviceRequest{
				Driver:       val.Driver,
				Count:        val.Count,
				DeviceIDs:    val.DeviceIDs,
				Capabilities: [][]string{val.Capabilities},
				Options:      val.Options,
			}
		})
	}
	// hostConf.DeviceRequests = []container.DeviceRequest{
	// 	{
	// 		Driver: "nvidia",
	// 		Count:  2, // 限制使用 2 个 GPU
	// 		Capabilities: [][]string{
	// 			{"gpu"},
	// 		},
	// 	},
	// }
	// hostConf.Runtime = "nvidia"

	config.Volumes = make(map[string]struct{})
	for _, volume := range req.Volumes {
		if volume.Type == "volume" {
			hostConf.Mounts = append(hostConf.Mounts, mount.Mount{
				Type:   mount.Type(volume.Type),
				Source: volume.HostDir,
				Target: volume.ContainerDir,
			})
			config.Volumes[volume.ContainerDir] = struct{}{}
		} else {
			hostConf.Binds = append(hostConf.Binds, fmt.Sprintf("%s:%s:%s", volume.HostDir, volume.ContainerDir, volume.Mode))
		}
	}
	return &config, &hostConf, &networkConf, nil
}

func stringsToMap(list []string) map[string]string {
	var labelMap = make(map[string]string)
	for _, label := range list {
		if strings.Contains(label, "=") {
			sps := strings.SplitN(label, "=", 2)
			labelMap[sps[0]] = sps[1]
		}
	}
	return labelMap
}

func reCreateAfterUpdate(name string, client *client.Client, config *container.Config, hostConf *container.HostConfig, networkConf *types.NetworkSettings) {
	ctx := context.Background()

	var oldNetworkConf network.NetworkingConfig
	if networkConf != nil {
		for networkKey := range networkConf.Networks {
			oldNetworkConf.EndpointsConfig = map[string]*network.EndpointSettings{networkKey: {}}
			break
		}
	}

	oldContainer, err := client.ContainerCreate(ctx, config, hostConf, &oldNetworkConf, &v1.Platform{}, name)
	if err != nil {
		logx.Errorf("recreate after container update failed, err: %v", err)
		return
	}
	if err := client.ContainerStart(ctx, oldContainer.ID, container.StartOptions{}); err != nil {
		logx.Errorf("restart after container update failed, err: %v", err)
	}
	logx.Info("recreate after container update successful")
}

func loadVolumeBinds(binds []types.MountPoint) []form.Volume {
	var datas []form.Volume
	for _, bind := range binds {
		var volumeItem form.Volume
		volumeItem.Type = string(bind.Type)
		if bind.Type == "volume" {
			volumeItem.HostDir = bind.Name
		} else {
			volumeItem.HostDir = bind.Source
		}
		volumeItem.ContainerDir = bind.Destination
		volumeItem.Mode = "ro"
		if bind.RW {
			volumeItem.Mode = "rw"
		}
		datas = append(datas, volumeItem)
	}
	return datas
}
