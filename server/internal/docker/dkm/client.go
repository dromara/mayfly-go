package dkm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/pool"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/gorilla/websocket"
	"github.com/spf13/cast"
)

var (
	poolGroup = pool.NewPoolGroup[*Client]()
)

const (
	DefaultServer = "unix:///var/run/docker.sock"
)

type ContainerServer struct {
	Id   uint64
	Addr string
}

type Client struct {
	Server       *ContainerServer
	DockerClient *client.Client
}

func (c *Client) Close() error {
	return c.DockerClient.Close()
}

func (c *Client) Ping() error {
	_, err := c.DockerClient.Ping(context.Background())
	return err
}

// GetCli get docker cli
func GetCli(id uint64, getContainer func(uint64) (*ContainerServer, error)) (*Client, error) {
	pool, err := poolGroup.GetCachePool(fmt.Sprintf("%d", id), func() (*Client, error) {
		containerServer, err := getContainer(id)
		if err != nil {
			return nil, err
		}
		return NewClient(containerServer)
	})
	if err != nil {
		return nil, err
	}
	return pool.Get(context.Background())
}

func CloseCli(id uint64) error {
	return poolGroup.Close(fmt.Sprintf("%d", id))
}

// NewClient new docker client
func NewClient(server *ContainerServer) (*Client, error) {
	if server.Addr == "" {
		server.Addr = DefaultServer
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithHost(server.Addr), client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &Client{
		DockerClient: cli,
		Server:       server,
	}, nil
}

func (c Client) ContainerList() ([]container.Summary, error) {
	var (
		options container.ListOptions
	)
	options.All = true
	containers, err := c.DockerClient.ContainerList(context.Background(), options)
	if err != nil {
		return nil, err
	}
	return containers, nil
}

func (c Client) ContainerStats(containerID string) (container.StatsResponse, error) {
	var stats container.StatsResponse
	res, err := c.DockerClient.ContainerStats(context.Background(), containerID, false)
	if err != nil {
		return stats, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return stats, err
	}

	if err := json.Unmarshal(body, &stats); err != nil {
		return stats, err
	}

	return stats, nil
}

func (c Client) ContainerRestart(containerID string) error {
	return c.DockerClient.ContainerRestart(context.Background(), containerID, container.StopOptions{})
}

func (c Client) ContainerStop(containerID string) error {
	return c.DockerClient.ContainerStop(context.Background(), containerID, container.StopOptions{})
}

func (c Client) ContainerAttach(containerID string, wsConn *websocket.Conn, rows, cols int) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建exec配置（启用TTY）
	execID, err := c.DockerClient.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{"/bin/bash"}, // 或指定其他shell
	})
	if err != nil {
		return err
	}

	hijackedResp, err := c.DockerClient.ContainerExecAttach(ctx, execID.ID, container.ExecAttachOptions{
		Tty:         true,
		ConsoleSize: &[2]uint{cast.ToUint(rows), cast.ToUint(cols)},
	})
	if err != nil {
		return err
	}
	defer hijackedResp.Close()

	wsConn.WriteMessage(websocket.TextMessage, []byte("\033[2J\033[3J\033[1;1H")) // 清屏

	// 转发容器输出到前端
	go func() {
		buf := make([]byte, 1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := hijackedResp.Reader.Read(buf)
				if err != nil {
					if err != io.EOF {
						logx.ErrorTrace("Read container output error:", err)
					}
					// 容器退出时主动关闭WebSocket
					wsConn.WriteMessage(websocket.CloseMessage, []byte{})
					cancel()
					return
				}
				wsConn.WriteMessage(websocket.TextMessage, buf[:n])
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			_, input, err := wsConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err) {
					logx.Debug("WebSocket closed:", err)
				}
				return nil
			}

			// 解析消息
			msgObj, err := mcm.ParseMsg(input)
			if err != nil {
				wsConn.WriteMessage(websocket.TextMessage, []byte("failed to parse the message content..."))
				logx.Error("terminal message parsing failed: ", err)
				return nil
			}

			switch msgObj.Type {
			case mcm.MsgTypeResize:
				if msgObj.Cols > 0 && msgObj.Rows > 0 {
					c.DockerClient.ContainerExecResize(ctx, execID.ID, container.ResizeOptions{
						Height: cast.ToUint(msgObj.Rows),
						Width:  cast.ToUint(msgObj.Cols),
					})
				}
			case mcm.MsgTypeData:
				data := []byte(msgObj.Msg)
				hijackedResp.Conn.Write(data)
			case mcm.MsgTypePing:

			}
		}
	}
}

func (c Client) ContainerRemove(containerID string) error {
	return c.DockerClient.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true, RemoveVolumes: true})
}

func (c Client) ContainerInspect(containerID string) (types.ContainerJSON, error) {
	return c.DockerClient.ContainerInspect(context.Background(), containerID)
}

func (c Client) ImageRemove(imageID string) error {
	if _, err := c.DockerClient.ImageRemove(context.Background(), imageID, image.RemoveOptions{Force: true}); err != nil {
		return err
	}
	return nil
}

func (c Client) ImageList() ([]image.Summary, error) {
	return c.DockerClient.ImageList(context.Background(), image.ListOptions{
		All:            false,
		ContainerCount: true,
	})
}

func (c Client) PullImage(imageName string, force bool) error {
	if !force {
		exist, err := c.CheckImageExist(imageName)
		if err != nil {
			return err
		}
		if exist {
			return nil
		}
	}
	if _, err := c.DockerClient.ImagePull(context.Background(), imageName, image.PullOptions{}); err != nil {
		return err
	}
	return nil
}

func (c Client) GetImageIDByName(imageName string) (string, error) {
	filter := filters.NewArgs()
	filter.Add("reference", imageName)
	list, err := c.DockerClient.ImageList(context.Background(), image.ListOptions{
		Filters: filter,
	})
	if err != nil {
		return "", err
	}
	if len(list) > 0 {
		return list[0].ID, nil
	}
	return "", nil
}

func (c Client) CheckImageExist(imageName string) (bool, error) {
	filter := filters.NewArgs()
	filter.Add("reference", imageName)
	list, err := c.DockerClient.ImageList(context.Background(), image.ListOptions{
		Filters: filter,
	})
	if err != nil {
		return false, err
	}
	return len(list) > 0, nil
}

func (c Client) CreateNetwork(name string) error {
	_, err := c.DockerClient.NetworkCreate(context.Background(), name, network.CreateOptions{
		Driver: "bridge",
	})
	return err
}

func (c Client) NetworkExist(name string) bool {
	var options network.ListOptions
	options.Filters = filters.NewArgs(filters.Arg("name", name))
	networks, err := c.DockerClient.NetworkList(context.Background(), options)
	if err != nil {
		return false
	}
	return len(networks) > 0
}
