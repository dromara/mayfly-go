package ws

import (
	"mayfly-go/pkg/logx"
	"sync"
	"time"
)

// 心跳间隔
const heartbeatInterval = 25 * time.Second

// 单个用户的全部的连接, key->clientId, value->Client
type UserClients map[string]*Client

func (ucs UserClients) GetByCid(clientId string) *Client {
	return ucs[clientId]
}

func (ucs UserClients) AddClient(client *Client) {
	ucs[client.ClientId] = client
}

func (ucs UserClients) DeleteByCid(clientId string) {
	delete(ucs, clientId)
}

func (ucs UserClients) Count() int {
	return len(ucs)
}

// 连接管理
type ClientManager struct {
	UserClientsMap map[UserId]UserClients // 全部的用户连接, key->userid, value->UserClients
	RwLock         sync.RWMutex           // 读写锁

	ConnectChan    chan *Client // 连接处理
	DisConnectChan chan *Client // 断开连接处理
	MsgChan        chan *Msg    //  消息信息channel通道
}

func NewClientManager() (clientManager *ClientManager) {
	return &ClientManager{
		UserClientsMap: make(map[UserId]UserClients),
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
	for _, client := range manager.GetByUid(userId) {
		manager.CloseClient(client)
	}
}

// 获取所有的客户端
func (manager *ClientManager) AllUserClient() map[UserId]UserClients {
	manager.RwLock.RLock()
	defer manager.RwLock.RUnlock()

	return manager.UserClientsMap
}

// 通过userId获取用户所有客户端信息
func (manager *ClientManager) GetByUid(userId UserId) UserClients {
	manager.RwLock.RLock()
	defer manager.RwLock.RUnlock()
	return manager.UserClientsMap[userId]
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
	manager.RwLock.RLock()
	defer manager.RwLock.RUnlock()
	return len(manager.UserClientsMap)
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
						logx.Warnf("ws消息发送失败[uid=%d, cid=%s]: %s", uid, cid, err.Error())
					}
				} else {
					logx.Warnf("[uid=%v, cid=%s]的ws连接不存在", uid, cid)
				}
				continue
			}

			// cid为空，则向该用户所有客户端发送该消息
			for _, cli := range manager.GetByUid(uid) {
				if err := cli.WriteMsg(msg); err != nil {
					logx.Warnf("ws消息发送失败[uid=%d, cid=%s]: %s", uid, cli.ClientId, err.Error())
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
			for userId, clis := range manager.AllUserClient() {
				for _, cli := range clis {
					if cli == nil || cli.WsConn == nil {
						continue
					}
					if err := cli.Ping(); err != nil {
						manager.CloseClient(cli)
						logx.Debugf("WS发送心跳失败: uid=%v, cid=%s, usercount=%d", userId, cli.ClientId, Manager.Count())
					} else {
						logx.Debugf("WS发送心跳成功: uid=%v, cid=%s", userId, cli.ClientId)
					}
				}
			}
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
	logx.Debugf("WS客户端已连接: uid=%d, cid=%s, usercount=%d", client.UserId, client.ClientId, manager.Count())
}

// 处理断开连接
func (manager *ClientManager) doDisconnect(client *Client) {
	//关闭连接
	if client.WsConn != nil {
		_ = client.WsConn.Close()
		client.WsConn = nil
	}
	manager.delUserClient4Map(client)
	logx.Debugf("WS客户端已断开: uid=%d, cid=%s, usercount=%d", client.UserId, client.ClientId, Manager.Count())
}

func (manager *ClientManager) addUserClient2Map(client *Client) {
	manager.RwLock.Lock()
	defer manager.RwLock.Unlock()

	userClients := manager.UserClientsMap[client.UserId]
	if userClients == nil {
		userClients = make(UserClients)
		manager.UserClientsMap[client.UserId] = userClients
	}
	userClients.AddClient(client)
}

func (manager *ClientManager) delUserClient4Map(client *Client) {
	manager.RwLock.Lock()
	defer manager.RwLock.Unlock()

	userClients := manager.UserClientsMap[client.UserId]
	if userClients != nil {
		userClients.DeleteByCid(client.ClientId)
		// 如果用户所有客户端都关闭，则移除manager中的UserClientsMap值
		if userClients.Count() == 0 {
			delete(manager.UserClientsMap, client.UserId)
		}
	}
}
