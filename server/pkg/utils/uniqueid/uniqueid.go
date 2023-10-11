package uniqueid

import "sync/atomic"

var id uint64 = 0

func IncrementID() uint64 {
	return atomic.AddUint64(&id, 1)
}
