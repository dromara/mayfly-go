package api

import (
	"encoding/json"
	form2 "mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/api/vo"
	"mayfly-go/internal/sys/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/req"
	"time"
)

const (
	AuthOAuth2Name  string = "OAuth2.0客户端配置"
	AuthOAuth2Key   string = "AuthOAuth2"
	AuthOAuth2Param string = "[{\"name\":\"Client ID\",\"model\":\"clientID\",\"placeholder\":\"客户端id\"}," +
		"{\"name\":\"Client Secret\",\"model\":\"clientSecret\",\"placeholder\":\"客户端密钥\"}," +
		"{\"name\":\"Authorization URL\",\"model\":\"authorizationURL\",\"placeholder\":\"https://example.com/oauth/authorize\"}," +
		"{\"name\":\"Access Token URL\",\"model\":\"accessTokenURL\",\"placeholder\":\"https://example.com/oauth/token\"}," +
		"{\"name\":\"Resource URL\",\"model\":\"resourceURL\",\"placeholder\":\"https://example.com/oauth/token\"}," +
		"{\"name\":\"Redirect URL\",\"model\":\"redirectURL\",\"placeholder\":\"http://mayfly地址/\"}," +
		"{\"name\":\"User identifier\",\"model\":\"userIdentifier\",\"placeholder\":\"\"}," +
		"{\"name\":\"Scopes\",\"model\":\"scopes\",\"placeholder\":\"read_user\"}]"
	AuthOAuth2Remark string = "自定义oauth2.0 server登录"
)

type Auth struct {
	ConfigApp application.Config
}

// GetInfo 获取认证平台信息
func (a *Auth) GetInfo(rc *req.Ctx) {
	config := a.ConfigApp.GetConfig(AuthOAuth2Key)
	oauth2 := &vo.OAuth2VO{}
	if config.Value != "" {
		if err := json.Unmarshal([]byte(config.Value), oauth2); err != nil {
			global.Log.Warnf("解析自定义oauth2配置失败，err：%s", err.Error())
			biz.ErrIsNil(err, "解析自定义oauth2配置失败")
		}
	}
	rc.ResData = &vo.AuthVO{
		OAuth2VO: oauth2,
	}
}

func (a *Auth) SaveOAuth2(rc *req.Ctx) {
	form := &form2.OAuth2Form{}
	form = ginx.BindJsonAndValid(rc.GinCtx, form)
	rc.ReqParam = form
	// 先获取看看有没有
	config := a.ConfigApp.GetConfig(AuthOAuth2Key)
	now := time.Now()
	if config.Id == 0 {
		config.CreatorId = rc.LoginAccount.Id
		config.Creator = rc.LoginAccount.Username
		config.CreateTime = &now
	}
	config.ModifierId = rc.LoginAccount.Id
	config.Modifier = rc.LoginAccount.Username
	config.UpdateTime = &now
	config.Name = AuthOAuth2Name
	config.Key = AuthOAuth2Key
	config.Params = AuthOAuth2Param
	b, err := json.Marshal(form)
	if err != nil {
		biz.ErrIsNil(err, "json marshal error")
		return
	}
	config.Value = string(b)
	config.Remark = AuthOAuth2Remark
	a.ConfigApp.Save(config)
}
