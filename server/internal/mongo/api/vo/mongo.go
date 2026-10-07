package vo

import (
	"mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/internal/mongo/mgm"
	"mayfly-go/pkg/model"
)

// Mongo 实例对外视图。
type Mongo struct {
	model.Model

	Code               string `orm:"column(code)" json:"code"`
	Name               string `orm:"column(name)" json:"name"`
	Uri                string `orm:"column(uri)" json:"uri"`
	SshTunnelMachineId int    `orm:"column(ssh_tunnel_machine_id)" json:"sshTunnelMachineId"` // ssh隧道机器id
}

func (m *Mongo) GetCode() string {
	return m.Code
}

// OfMongo 实体转对外视图，连接串在此统一脱敏。
//
// 明文 uri 含账号密码，列表与详情只要回传它就等于向所有具备资源可见性的账号泄露凭证；
// 脱敏收口在这一个构造函数里，避免「某个新接口忘了掩码」。需要改连接串时由使用者重新输入
// （表单里留空表示保持原值，见 application.SaveMongo）。
func OfMongo(me *entity.Mongo) *Mongo {
	return &Mongo{
		Model:              me.Model,
		Code:               me.Code,
		Name:               me.Name,
		Uri:                mgm.MaskUri(me.Uri),
		SshTunnelMachineId: me.SshTunnelMachineId,
	}
}

// OfMongos 批量转换。
func OfMongos(list []*entity.Mongo) []*Mongo {
	vos := make([]*Mongo, 0, len(list))
	for _, item := range list {
		vos = append(vos, OfMongo(item))
	}
	return vos
}

// PageOfMongos 分页转换。
func PageOfMongos(res *model.PageResult[*entity.Mongo]) *model.PageResult[*Mongo] {
	if res == nil {
		return model.NewEmptyPageResult[*Mongo]()
	}
	return &model.PageResult[*Mongo]{Total: res.Total, List: OfMongos(res.List)}
}

// ExportResult 导出结果。
//
// 内容作为普通字段回传而不走文件流：NoRes 流式响应下框架不输出错误体，
// 权限不足/超时会让前端拿到一个「成功但空」的响应；文件名与下载由前端负责。
type ExportResult struct {
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Count       int    `json:"count"`
	Content     string `json:"content"`
}
