package esi

import (
	"net/http/httputil"
	"sync"
)

type BufferPool struct {
	pool *sync.Pool
}

// 需要实现 httputil.BufferPool
var _ httputil.BufferPool = (*BufferPool)(nil)

func NewBufferPool() *BufferPool {
	return &BufferPool{&sync.Pool{
		New: func() interface{} {
			return make([]byte, 32*1024)
		},
	}}
}

func (b *BufferPool) Get() []byte {
	return b.pool.Get().([]byte)
}

func (b *BufferPool) Put(buf []byte) {
	b.pool.Put(buf)
}
