package model

import (
	"sync"
	"time"
)

// IdGenerator 应用侧主键生成策略（可选能力位）。
//
// 主键默认由数据库自增生成；实体实现了本接口时，基座仓储在新增（主键为 0）的实体上
// 调用它发号。开闭原则的落点：新增一种生成方式（时间戳、雪花、号段等）由实体自行实现，
// 基座不发分支、不改代码。
//
// 发号结果的唯一性由策略自己负责：TimestampId 保证单进程内不重，多实例部署应选用自带
// 机器号与序列位的策略（如雪花），并在实现内接入实例标识
type IdGenerator interface {
	// GenerateId 生成一个主键。基座只在实体主键为 0（即本次是新增）时调用
	GenerateId() uint64
}

// timestampIdState 秒级时间戳发号的进程内状态：last 记录上一个已发出的值
var timestampIdState struct {
	sync.Mutex
	last uint64
}

// TimestampId 生成秒级时间戳主键，并保证单进程内严格递增：
// 同一秒内的连续发号取上一值加一，批量新增与同秒手工写入不会在发号瞬间撞主键。
//
// 适用边界要说清：它保证的是单进程唯一。多个实例同秒启动发号仍可能取到相同值，
// 多实例部署或高频写入请实现自带机器号与序列位的策略（如雪花）
func TimestampId() uint64 {
	id := uint64(time.Now().Unix())
	timestampIdState.Lock()
	defer timestampIdState.Unlock()
	if id <= timestampIdState.last {
		id = timestampIdState.last + 1
	}
	timestampIdState.last = id
	return id
}
