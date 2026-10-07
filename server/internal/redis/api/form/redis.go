package form

import "mayfly-go/internal/redis/domain/entity"

type Redis struct {
	Id                 uint64   `json:"id"`
	Name               string   `json:"name"`
	Host               string   `json:"host" binding:"required"`
	Username           string   `json:"username"`
	Password           string   `json:"password"`
	RedisNodePassword  string   `json:"redisNodePassword"`
	Mode               string   `json:"mode"`
	Db                 string   `json:"db"`
	SshTunnelMachineId int      `json:"sshTunnelMachineId"` // ssh隧道机器id
	TagCodePaths       []string `binding:"required" json:"tagCodePaths"`
	Remark             string   `json:"remark"`
	FlowProcdefKey     string   `json:"flowProcdefKey"` // 审批流-流程定义key（有值则说明关键操作需要进行审批执行）,使用指针为了方便更新空字符串(取消流程审批)
}

type RedisScanForm struct {
	Cursor map[string]uint64 `json:"cursor"`
	Match  string            `json:"match"`
	Count  int64             `json:"count"`
}

type ScanForm struct {
	Key    string `json:"key"`
	Cursor uint64 `json:"cursor"`
	Match  string `json:"match"`
	Count  int64  `json:"count"`
}

type RunCmdForm struct {
	Id     uint64 `json:"id"`
	Db     int    `json:"db"`
	Cmd    []any  `json:"cmd"`
	Remark string `json:"remark"`
	// AckWarn 操作者已在确认框里选「直接执行」，用于命中「仅提醒」后的重试
	AckWarn bool `json:"ackWarn"`
}

// KeyMemberPageForm key 成员分页读取
type KeyMemberPageForm struct {
	Key     string `json:"key" binding:"required"`
	View    string `json:"view"`
	Cursor  string `json:"cursor"`
	Offset  int64  `json:"offset"`
	Size    int64  `json:"size"`
	Keyword string `json:"keyword"`

	// AckWarn 「仅提醒」命中的确认位：面板读内容也是用户点出来的，与写操作同一套确认流
	AckWarn bool `json:"ackWarn"`
}

// KeyMemberWriteForm 成员新增/修改/删除，args 为该视角表单收集的字段值
type KeyMemberWriteForm struct {
	Key     string            `json:"key" binding:"required"`
	View    string            `json:"view"`
	Op      string            `json:"op" binding:"required"`
	Member  *entity.Member    `json:"member"`
	Members []*entity.Member  `json:"members"`
	Args    map[string]string `json:"args"`
	Ttl     int64             `json:"ttl"`

	// AckWarn 「仅提醒」命中的确认位：界面弹框后用户选了「直接执行」才为 true，
	// 首次请求为 false（此时命令不会执行，服务端返回确认码让界面去问）
	AckWarn bool `json:"ackWarn"`
}

// KeyOpForm 视角扩展操作
type KeyOpForm struct {
	Key  string            `json:"key" binding:"required"`
	View string            `json:"view"`
	Op   string            `json:"op" binding:"required"`
	Args map[string]string `json:"args"`

	// AckWarn 「仅提醒」命中的确认位：界面弹框后用户选了「直接执行」才为 true，
	// 首次请求为 false（此时命令不会执行，服务端返回确认码让界面去问）
	AckWarn bool `json:"ackWarn"`
}

// KeyTtlForm 设置 key 过期时间，ttl <= 0 表示持久化
type KeyTtlForm struct {
	Key string `json:"key" binding:"required"`
	Ttl int64  `json:"ttl"`

	// AckWarn 「仅提醒」命中的确认位：界面弹框后用户选了「直接执行」才为 true，
	// 首次请求为 false（此时命令不会执行，服务端返回确认码让界面去问）
	AckWarn bool `json:"ackWarn"`
}

// KeyRenameForm key 重命名
type KeyRenameForm struct {
	Key    string `json:"key" binding:"required"`
	NewKey string `json:"newKey" binding:"required"`

	// AckWarn 「仅提醒」命中的确认位：界面弹框后用户选了「直接执行」才为 true，
	// 首次请求为 false（此时命令不会执行，服务端返回确认码让界面去问）
	AckWarn bool `json:"ackWarn"`
}

// KeyCopyForm 复制 key，TargetDb 缺省表示留在当前库
type KeyCopyForm struct {
	Key      string `json:"key" binding:"required"`
	NewKey   string `json:"newKey" binding:"required"`
	TargetDb *int   `json:"targetDb"`
	Replace  bool   `json:"replace"`

	// AckWarn 「仅提醒」命中的确认位：界面弹框后用户选了「直接执行」才为 true，
	// 首次请求为 false（此时命令不会执行，服务端返回确认码让界面去问）
	AckWarn bool `json:"ackWarn"`
}

// KeysForm 批量 key 操作（类型摘要、批量删除）
type KeysForm struct {
	Keys []string `json:"keys" binding:"required"`

	// AckWarn 「仅提醒」命中的确认位：界面弹框后用户选了「直接执行」才为 true，
	// 首次请求为 false（此时命令不会执行，服务端返回确认码让界面去问）
	AckWarn bool `json:"ackWarn"`
}
