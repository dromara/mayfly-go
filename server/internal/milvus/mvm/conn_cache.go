package mvm

import (
	"context"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/pool"
	"sync"
)

var (
	poolGroup = pool.NewPoolGroup[*MilvusConn]()

	// connIdsLock 保护 connIds：连接工厂在并发请求中执行，多实例同时建池会并发写 map
	connIdsLock sync.Mutex
	// 通过id记录连接名
	connIds = make(map[uint64][]string)
)

// addConnId 记录实例 id 到连接名的映射，CloseAll 依赖该映射回收同一实例的全部库/账号连接
func addConnId(milvusId uint64, connId string) {
	connIdsLock.Lock()
	defer connIdsLock.Unlock()

	for _, id := range connIds[milvusId] {
		if id == connId {
			return
		}
	}
	connIds[milvusId] = append(connIds[milvusId], connId)
}

// removeConnId 摘除已关闭的连接名：不摘除会让 CloseAll 重复关闭已释放的池
func removeConnId(milvusId uint64, connId string) {
	connIdsLock.Lock()
	defer connIdsLock.Unlock()

	ids := connIds[milvusId]
	remain := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != connId {
			remain = append(remain, id)
		}
	}
	if len(remain) == 0 {
		delete(connIds, milvusId)
		return
	}
	connIds[milvusId] = remain
}

// 从缓存中获取 milvus 连接信息，若缓存中不存在则会使用回调函数获取 milvusInfo 进行连接并缓存
func GetMilvusConn(ctx context.Context, milvusId uint64, db string, ac string, getMilvusInfo func() (*MilvusInfo, error)) (*MilvusConn, error) {
	connId := getConnId(milvusId, db, ac)
	p, err := poolGroup.GetCachePool(connId, func() (*MilvusConn, error) {
		// 若缓存中不存在，则从回调函数中获取 info
		mi, err := getMilvusInfo()
		if err != nil {
			return nil, err
		}

		addConnId(milvusId, connId)

		// 连接 milvus
		return mi.Conn()
	})

	if err != nil {
		return nil, err
	}
	// 从连接池中获取一个可用的连接
	return p.Get(ctx)
}

// 关闭连接，并移除缓存连接
func CloseConn(id uint64, database string, ac string) {
	connId := getConnId(id, database, ac)
	gox.Go(func() {
		err := poolGroup.Close(connId)
		if err != nil {
			logx.Errorf("close milvus connection failed: %v", err)
			return
		}
		removeConnId(id, connId)
	})
}

func CloseAll(id uint64) {
	connIdsLock.Lock()
	ids := append([]string(nil), connIds[id]...)
	delete(connIds, id)
	connIdsLock.Unlock()

	for _, connId := range ids {
		poolGroup.Close(connId)
	}
}
