package ws

import (
	"mayfly-go/pkg/logx"
	"sync"
	"time"
)

// 心跳间隔
const heartbeatInterval = 25 * time.Second

// UserClient 用户全部的连接
type UserClient struct {
	clients map[string]*Client // key->clientId, value->Client
	mutex   sync.RWMutex
}

func NewUserClient() *UserClient {
	return &UserClient{
		clients: make(map[string]*Client),
	}
}

// AllClients 获取全部的连接
func (ucs *UserClient) AllClients() []*Client {
	ucs.mutex.RLock()
	defer ucs.mutex.RUnlock()
	result := make([]*Client, 0, len(ucs.clients))
	for _, client := range ucs.clients {
		result = append(result, client)
	}
	return result
}

// GetByCid 获取指定客户端ID的客户端
func (ucs *UserClient) GetByCid(clientId string) *Client {
	ucs.mutex.RLock()
	defer ucs.mutex.RUnlock()
	return ucs.clients[clientId]
}

// AddClient 添加客户端
func (ucs *UserClient) AddClient(client *Client) {
	ucs.mutex.Lock()
	defer ucs.mutex.Unlock()
	ucs.clients[client.ClientId] = client
}

// DeleteByCid 删除指定客户端ID的客户端
func (ucs *UserClient) DeleteByCid(clientId string) {
	ucs.mutex.Lock()
	defer ucs.mutex.Unlock()
	delete(ucs.clients, clientId)
}

// Count 返回客户端数量
func (ucs *UserClient) Count() int {
	ucs.mutex.RLock()
	defer ucs.mutex.RUnlock()
	return len(ucs.clients)
}

// 连接管理
type ClientManager struct {
	UserClientMap sync.Map // 全部的用户连接, key->userid, value->*UserClient

	ConnectChan    chan *Client // 连接处理
	DisConnectChan chan *Client // 断开连接处理
	MsgChan        chan *Msg    // 消息信息channel通道
}

func NewClientManager() (clientManager *ClientManager) {
	return &ClientManager{
		UserClientMap:  sync.Map{},
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
func (manager *ClientManager) CloseByUid(userId UserId) {
	userClient := manager.GetByUid(userId)
	for _, client := range userClient.AllClients() {
		manager.CloseClient(client)
	}
}

// 获取所有的客户端
func (manager *ClientManager) AllUserClient() map[UserId]*UserClient {
	result := make(map[UserId]*UserClient)
	manager.UserClientMap.Range(func(key, value any) bool {
		userId := key.(UserId)
		userClients := value.(*UserClient)
		result[userId] = userClients
		return true
	})
	return result
}

// 通过userId获取用户所有客户端信息
func (manager *ClientManager) GetByUid(userId UserId) *UserClient {
	if value, ok := manager.UserClientMap.Load(userId); ok {
		return value.(*UserClient)
	}
	return nil
}

// 通过userId和clientId获取客户端信息
func (manager *ClientManager) GetByUidAndCid(uid UserId, clientId string) *Client {
	if clients := manager.GetByUid(uid); clients != nil {
		return clients.GetByCid(clientId)
	}
	return nil
}

// 客户端数量
func (manager *ClientManager) Count() int {
	count := 0
	manager.UserClientMap.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

// 发送json数据给指定用户
func (manager *ClientManager) SendJsonMsg(userId UserId, clientId string, data any) {
	manager.MsgChan <- &Msg{ToUserId: userId, ToClientId: clientId, Data: data, Type: JsonMsg}
}

// 监听并发送给客户端信息
func (manager *ClientManager) WriteMessage() {
	go func() {
		for {
			msg := <-manager.MsgChan
			uid := msg.ToUserId
			cid := msg.ToClientId
			// 客户端id不为空，则向指定客户端发送消息即可
			if cid != "" {
				cli := manager.GetByUidAndCid(uid, cid)
				if cli != nil {
					if err := cli.WriteMsg(msg); err != nil {
						logx.Warnf("ws send message failed - [uid=%d, cid=%s]: %s", uid, cid, err.Error())
					}
				} else {
					logx.Warnf("[uid=%v, cid=%s] ws conn not exist", uid, cid)
				}
				continue
			}

			// cid为空，则向该用户所有客户端发送该消息
			userClients := manager.GetByUid(uid)
			if userClients != nil {
				for _, cli := range userClients.AllClients() {
					if err := cli.WriteMsg(msg); err != nil {
						logx.Warnf("ws send message failed - [uid=%d, cid=%s]: %s", uid, cli.ClientId, err.Error())
					}
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
			manager.UserClientMap.Range(func(key, value any) bool {
				userId := key.(UserId)
				userClient := value.(*UserClient)
				for _, cli := range userClient.AllClients() {
					if cli == nil || cli.WsConn == nil {
						continue
					}
					if err := cli.Ping(); err != nil {
						manager.CloseClient(cli)
						logx.Debugf("WS - failed to send heartbeat: uid=%v, cid=%s, usercount=%d", userId, cli.ClientId, manager.Count())
					} else {
						logx.Debugf("WS - send heartbeat successfully: uid=%v, cid=%s", userId, cli.ClientId)
					}
				}
				return true
			})
		}
	}()
}

// 处理建立连接
func (manager *ClientManager) doConnect(client *Client) {
	cli := manager.GetByUidAndCid(client.UserId, client.ClientId)
	if cli != nil {
		manager.doDisconnect(cli)
	}
	manager.addUserClient2Map(client)
	logx.Debugf("WS client connected: uid=%d, cid=%s, usercount=%d", client.UserId, client.ClientId, manager.Count())
}

// 处理断开连接
func (manager *ClientManager) doDisconnect(client *Client) {
	//关闭连接
	if client.WsConn != nil {
		_ = client.WsConn.Close()
		client.WsConn = nil
	}
	manager.delUserClient4Map(client)
	logx.Debugf("WS client disconnected: uid=%d, cid=%s, usercount=%d", client.UserId, client.ClientId, manager.Count())
}

func (manager *ClientManager) addUserClient2Map(client *Client) {
	// 先尝试加载现有的UserClients
	if value, ok := manager.UserClientMap.Load(client.UserId); ok {
		userClient := value.(*UserClient)
		userClient.AddClient(client)
	} else {
		// 创建新的UserClients
		userClient := NewUserClient()
		userClient.AddClient(client)
		manager.UserClientMap.Store(client.UserId, userClient)
	}
}

func (manager *ClientManager) delUserClient4Map(client *Client) {
	if value, ok := manager.UserClientMap.Load(client.UserId); ok {
		userClient := value.(*UserClient)
		userClient.DeleteByCid(client.ClientId)
		// 如果用户所有客户端都关闭，则移除manager中的UserClientsMap值
		if userClient.Count() == 0 {
			manager.UserClientMap.Delete(client.UserId)
		}
	}
}
