package api

import (
	"mayfly-go/internal/msg/api/form"
	"mayfly-go/internal/msg/application"
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/imsg"
	"mayfly-go/internal/msg/msgx"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"strings"

	"github.com/spf13/cast"
)

type MsgChannel struct {
	msgChannelApp application.MsgChannel `inject:"T"`
}

func (m *MsgChannel) ReqConfs() *req.Confs {
	basePermCode := "msg:channel:base"

	reqs := [...]*req.Conf{
		req.NewGet("", m.GetMsgChannels).RequiredPermissionCode(basePermCode),
		// 单条详情：编辑回填唯一合法来源。存在理由见 GetMsgChannel：渠道的 extra 里带着钉钉/企微
		// 机器人的签名密钥，它只能走与保存同一权限点的出口，不能随列表（菜单权限即达）外溢
		req.NewGet("/:id", m.GetMsgChannel).RequiredPermissionCode("msg:channel:save"),
		req.NewGet("simple", m.GetSimpleChannels).RequiredPermissionCode(basePermCode),
		req.NewPost("", m.SaveMsgChannels).Log(req.NewLogSaveI(imsg.LogMsgChannelSave)).RequiredPermissionCode("msg:channel:save"),
		req.NewDelete("", m.DelMsgChannels).Log(req.NewLogSaveI(imsg.LogMsgChannelDelete)).RequiredPermissionCode("msg:channel:del"),
	}

	return req.NewConfs("/msg/channels", reqs[:]...)
}

func (m *MsgChannel) GetMsgChannels(rc *req.Ctx) {
	// 搜索框发什么就必须按什么筛：此前这里不读任何查询参数、直接拿空实体去查，
	// 于是「按名称搜渠道」永远返回全量（与同文件的 GetMsgTmpls 不同口径），而界面搜索框照常在
	condition := &entity.MsgChannel{
		Code: rc.Query("code"),
		Name: rc.Query("name"),
	}
	res, err := m.msgChannelApp.GetPageList(condition, rc.GetPageParam())
	biz.ErrIsNil(err)
	// 凭据不得随列表外溢：extra 里的 secret 是 bot 类渠道的 HMAC 签名密钥（ding_bot 拿它算
	// &sign=），拿到就能以该渠道身份往外部群伪造消息。而列表按菜单权限即可达，
	// 与「需要凭据的动作走受保存权限的详情接口」这一既有口径（触发器 webhookToken）对齐
	for _, channel := range res.List {
		channel.SetExtraValue(msgx.ExtraKeySecret, "")
	}
	rc.ResData = res
}

// GetMsgChannel 单条渠道详情（编辑回填用，带完整 extra 含签名密钥）
//
// 权限点刻意与保存一致而不是菜单基权限：能改这个渠道的人本就该看得它的密钥（前端要回显才能
// 保存不丢值），无关的只读账号不该拿到。与触发器详情同口径
func (m *MsgChannel) GetMsgChannel(rc *req.Ctx) {
	channel, err := m.msgChannelApp.GetById(cast.ToUint64(rc.PathParam("id")))
	biz.ErrIsNil(err)
	rc.ResData = channel
}

// GetSimpleChannels 返回所有启用渠道的 id 和 name（供告警渠道选择器等使用）
type simpleChannel struct {
	Id   uint64 `json:"id"`
	Name string `json:"name"`
}

func (m *MsgChannel) GetSimpleChannels(rc *req.Ctx) {
	channels, err := m.msgChannelApp.ListByCond(nil)
	biz.ErrIsNil(err)
	result := make([]simpleChannel, 0, len(channels))
	for _, ch := range channels {
		if ch.Status == entity.ChannelStatusEnable {
			result = append(result, simpleChannel{Id: ch.Id, Name: ch.Name})
		}
	}
	rc.ResData = result
}

func (m *MsgChannel) SaveMsgChannels(rc *req.Ctx) {
	form, channel := rc.BindJsonAndCopyTo[form.MsgChannel, entity.MsgChannel]()
	rc.ReqParam = form
	err := m.msgChannelApp.SaveChannel(rc.MetaCtx, channel)
	biz.ErrIsNil(err)
}

func (m *MsgChannel) DelMsgChannels(rc *req.Ctx) {
	idsStr := rc.Query("id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		biz.ErrIsNil(m.msgChannelApp.DeleteChannel(rc.MetaCtx, cast.ToUint64(v)))
	}
}
