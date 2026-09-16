package cache

import (
	"fmt"
	"mayfly-go/internal/machine/mcm"
	global_cache "mayfly-go/pkg/cache"
	"mayfly-go/pkg/utils/jsonx"
	"time"
)

const MachineStatCacheKey = "mayfly:machine:%d:stat"

func SaveMachineStats(machineId uint64, stat *mcm.Stats) error {
	return global_cache.Set(fmt.Sprintf(MachineStatCacheKey, machineId), stat, 10*time.Minute)
}

func GetMachineStats(machineId uint64) (*mcm.Stats, error) {
	cacheKey := fmt.Sprintf(MachineStatCacheKey, machineId)
	cacheStr := global_cache.GetStr(cacheKey)
	if cacheStr == "" {
		return nil, fmt.Errorf("machine stats cache miss, key=%s", cacheKey)
	}
	return jsonx.ToByStr[mcm.Stats](cacheStr)
}
