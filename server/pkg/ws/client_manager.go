package ws

import (
	"mayfly-go/pkg/logx"
	"sync"
	"time"
)

// 心跳间隔
const heartbeatInterval = 25 * time.Second

// 连接管理
type ClientManager struct {
	ClientMap map[UserId]*Client // 全部的连接, key->userid, value->&client
	RwLock    sync.RWMutex       // 读写锁

	ConnectChan    chan *Client // 连接处理
	DisConnectChan chan *Client // 断开连接处理
	MsgChan        chan *Msg    //  消息信息channel通道
}

func NewClientManager() (clientManager *ClientManager) {
	return &ClientManager{
		ClientMap:      make(map[UserId]*Client),
		ConnectChan:    make(chan *Client, 10),
		DisConnectChan: make(chan *Client, 10),
		MsgChan:        make(chan *Msg, 100),
	}
}

// 管道处理程序
func (manager *ClientManager) Start() {
	manager.HeartbeatTimer()
	manager.WriteMessage()
	for {
		select {
		case client := <-manager.ConnectChan:
			// 建立连接
			manager.doConnect(client)
		case conn := <-manager.DisConnectChan:
			// 断开连接
			manager.doDisconnect(conn)
		}
	}
}

// 添加客户端
func (manager *ClientManager) AddClient(client *Client) {
	manager.ConnectChan <- client
}

// 关闭客户端
func (manager *ClientManager) CloseClient(client *Client) {
	if client == nil {
		return
	}
	manager.DisConnectChan <- client
}

// 根据用户id关闭客户端连接
func (manager *ClientManager) CloseByUid(uid UserId) {
	manager.CloseClient(manager.GetByUid(UserId(uid)))
}

// 获取所有的客户端
func (manager *ClientManager) AllClient() map[UserId]*Client {
	manager.RwLock.RLock()
	defer manager.RwLock.RUnlock()

	return manager.ClientMap
}

// 通过userId获取
func (manager *ClientManager) GetByUid(userId UserId) *Client {
	manager.RwLock.RLock()
	defer manager.RwLock.RUnlock()
	return manager.ClientMap[userId]
}

// 客户端数量
func (manager *ClientManager) Count() int {
	manager.RwLock.RLock()
	defer manager.RwLock.RUnlock()
	return len(manager.ClientMap)
}

// 发送json数据给指定用户
func (manager *ClientManager) SendJsonMsg(userId UserId, data any) {
	logx.Debugf("发送消息: toUid=%v, data=%v", userId, data)
	manager.MsgChan <- &Msg{ToUserId: userId, Data: data, Type: JsonMsg}
}

// 监听并发送给客户端信息
func (manager *ClientManager) WriteMessage() {
	go func() {
		for {
			msg := <-manager.MsgChan
			if cli := manager.GetByUid(msg.ToUserId); cli != nil {
				if err := cli.WriteMsg(msg); err != nil {
					manager.CloseClient(cli)
				}
			}
		}
	}()
}

// 启动定时器进行心跳检测
func (manager *ClientManager) HeartbeatTimer() {
	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()
		for {
			<-ticker.C
			//发送心跳
			for userId, cli := range manager.AllClient() {
				if cli == nil || cli.WsConn == nil {
					continue
				}
				if err := cli.Ping(); err != nil {
					manager.CloseClient(cli)
					logx.Debugf("WS发送心跳失败: %v 总连接数：%d", userId, Manager.Count())
				} else {
					logx.Debugf("WS发送心跳成功: uid=%v", userId)
				}
			}
		}

	}()
}

// 处理建立连接
func (manager *ClientManager) doConnect(client *Client) {
	cli := manager.GetByUid(client.UserId)
	if cli != nil {
		manager.doDisconnect(cli)
	}
	manager.addClient2Map(client)
	logx.Debugf("WS客户端已连接: uid=%d, count=%d", client.UserId, manager.Count())
}

// 处理断开连接
func (manager *ClientManager) doDisconnect(client *Client) {
	//关闭连接
	if client.WsConn != nil {
		_ = client.WsConn.Close()
		client.WsConn = nil
	}
	manager.delClient4Map(client)
	logx.Debugf("WS客户端已断开: uid=%d, count=%d", client.UserId, Manager.Count())
}

func (manager *ClientManager) addClient2Map(client *Client) {
	manager.RwLock.Lock()
	defer manager.RwLock.Unlock()
	manager.ClientMap[client.UserId] = client
}

func (manager *ClientManager) delClient4Map(client *Client) {
	manager.RwLock.Lock()
	defer manager.RwLock.Unlock()
	delete(manager.ClientMap, client.UserId)
}
