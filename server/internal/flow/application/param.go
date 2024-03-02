package application

// 启动流程实例请求入参
type StarProcParam struct {
	BizType string // 业务类型
	BizKey  string // 业务key
	Remark  string // 备注
	BizForm string // 业务表单信息
}

type CompleteProcinstTaskParam struct {
	TaskId uint64
	Remark string // 备注
}
