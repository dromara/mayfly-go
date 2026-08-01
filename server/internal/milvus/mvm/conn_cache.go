package mvm

import (
	"context"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/pool"
)

var (
	poolGroup = pool.NewPoolGroup[*MilvusConn]()

	// 通过id记录连接名
	connIds = make(map[uint64][]string)
)

// 从缓存中获取 milvus 连接信息，若缓存中不存在则会使用回调函数获取 milvusInfo 进行连接并缓存
func GetMilvusConn(ctx context.Context, milvusId uint64, db string, ac string, getMilvusInfo func() (*MilvusInfo, error)) (*MilvusConn, error) {
	connId := getConnId(milvusId, db, ac)
	p, err := poolGroup.GetCachePool(connId, func() (*MilvusConn, error) {
		// 若缓存中不存在，则从回调函数中获取 info
		mi, err := getMilvusInfo()
		if err != nil {
			return nil, err
		}

		if v, ok := connIds[milvusId]; ok {
			v = append(v, connId)
		} else {
			connIds[milvusId] = []string{connId}
		}

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
	gox.Go(func() {
		err := poolGroup.Close(getConnId(id, database, ac))
		if err != nil {
			logx.Errorf("close milvus connection failed: %v", err)
			return
		}
	})
}

func CloseAll(id uint64) {
	if v, ok := connIds[id]; ok {
		for _, connId := range v {
			poolGroup.Close(connId)
		}
	}
	delete(connIds, id)
}
