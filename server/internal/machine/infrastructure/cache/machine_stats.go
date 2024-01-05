package cache

import (
	"errors"
	"fmt"
	"mayfly-go/internal/machine/mcm"
	global_cache "mayfly-go/pkg/cache"
	"mayfly-go/pkg/utils/jsonx"
	"time"
)

const MachineStatCacheKey = "mayfly:machine:%d:stat"

func SaveMachineStats(machineId uint64, stat *mcm.Stats) error {
	return global_cache.SetStr(fmt.Sprintf(MachineStatCacheKey, machineId), jsonx.ToStr(stat), 10*time.Minute)
}

func GetMachineStats(machineId uint64) (*mcm.Stats, error) {
	cacheStr := global_cache.GetStr(fmt.Sprintf(MachineStatCacheKey, machineId))
	if cacheStr == "" {
		return nil, errors.New("不存在该值")
	}
	return jsonx.To(cacheStr, new(mcm.Stats))
}
