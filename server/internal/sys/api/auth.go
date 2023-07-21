package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
	"io"
	msgapp "mayfly-go/internal/msg/application"
	form2 "mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/api/vo"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils"
	"net/http"
	"strconv"
	"strings"
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
		"{\"name\":\"Scopes\",\"model\":\"scopes\",\"placeholder\":\"read_user\"}," +
		"{\"name\":\"自动注册\",\"model\":\"autoRegister\",\"placeholder\":\"开启自动注册将会自动注册账号, 否则需要手动创建账号后再进行绑定\",\"type\":\"checkbox\"}]"
	AuthOAuth2Remark string = "自定义oauth2.0 server登录"
)

type Auth struct {
	ConfigApp  application.Config
	AuthApp    application.Auth
	AccountApp application.Account
	MsgApp     msgapp.Msg
}

func (a *Auth) OAuth2Login(rc *req.Ctx) {
	client, _, err := a.getOAuthClient()
	if err != nil {
		biz.ErrIsNil(err, "获取oauth2 client失败: "+err.Error())
		return
	}
	state := utils.RandString(32)
	cache.SetStr("oauth2:state:"+state, "login", 5*time.Minute)
	rc.GinCtx.Redirect(http.StatusFound, client.AuthCodeURL(state))
}

func (a *Auth) OAuth2Callback(rc *req.Ctx) {
	client, oauth, err := a.getOAuthClient()
	if err != nil {
		biz.ErrIsNil(err, "获取oauth2 client失败: "+err.Error())
	}
	code := rc.GinCtx.Query("code")
	if code == "" {
		biz.ErrIsNil(errors.New("code不能为空"), "code不能为空")
	}
	state := rc.GinCtx.Query("state")
	if state == "" {
		biz.ErrIsNil(errors.New("state不能为空"), "state不能为空")
	}
	stateAction := cache.GetStr("oauth2:state:" + state)
	if stateAction == "" {
		biz.ErrIsNil(errors.New("state已过期，请重新登录"), "state已过期，请重新登录")
	}
	token, err := client.Exchange(rc.GinCtx, code)
	if err != nil {
		biz.ErrIsNil(err, "获取token失败: "+err.Error())
	}
	// 获取用户信息
	httpCli := client.Client(rc.GinCtx.Request.Context(), token)
	resp, err := httpCli.Get(oauth.ResourceURL)
	if err != nil {
		biz.ErrIsNil(err, "获取用户信息失败: "+err.Error())
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		biz.ErrIsNil(err, "获取用户信息失败: "+err.Error())
	}
	userInfo := make(map[string]interface{})
	err = json.Unmarshal(b, &userInfo)
	if err != nil {
		biz.ErrIsNil(err, "解析用户信息失败: "+err.Error())
	}

	// 获取用户唯一标识
	keys := strings.Split(oauth.UserIdentifier, ".")
	var identifier interface{} = userInfo
	endKey := keys[len(keys)-1]
	keys = keys[:len(keys)-1]
	for _, key := range keys {
		identifier = identifier.(map[string]interface{})[key]
	}
	identifier = identifier.(map[string]interface{})[endKey]
	userId := ""
	switch identifier.(type) {
	case string:
		userId = identifier.(string)
	case int, int32, int64:
		userId = fmt.Sprintf("%d", identifier)
	case float32, float64:
		userId = fmt.Sprintf("%.0f", identifier.(float64))
	}
	// 查询用户是否存在
	oauthAccount := &entity.OAuthAccount{Identity: userId}
	err = a.AuthApp.GetOAuthAccount(oauthAccount, "account_id", "identity")
	// 判断是登录还是绑定
	if stateAction == "login" {
		var accountId uint64
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				biz.ErrIsNil(err, "查询用户失败: "+err.Error())
			}
			// 不存在,进行注册
			if !oauth.AutoRegister {
				biz.ErrIsNil(errors.New("用户不存在，请先注册"), "用户不存在，请先注册")
			}
			now := time.Now()
			account := &entity.Account{
				Model: model.Model{
					CreateTime: &now,
					CreatorId:  0,
					Creator:    "oauth2",
					UpdateTime: &now,
				},
				Name:     userId,
				Username: userId,
			}
			a.AccountApp.Create(account)
			// 绑定
			if err := a.AuthApp.BindOAuthAccount(&entity.OAuthAccount{
				AccountId:  account.Id,
				Identity:   oauthAccount.Identity,
				CreateTime: &now,
				UpdateTime: &now,
			}); err != nil {
				biz.ErrIsNil(err, "绑定用户失败: "+err.Error())
			}
			accountId = account.Id
		} else {
			accountId = oauthAccount.AccountId
		}
		// 进行登录
		account := &entity.Account{
			Model: model.Model{Id: accountId},
		}
		if err := a.AccountApp.GetAccount(account, "Id", "Name", "Username", "Password", "Status", "LastLoginTime", "LastLoginIp", "OtpSecret"); err != nil {
			biz.ErrIsNil(err, "获取用户信息失败: "+err.Error())
		}
		biz.IsTrue(account.IsEnable(), "该账号不可用")
		// 访问系统使用的token
		accessToken := req.CreateToken(accountId, account.Username)
		// 默认为不校验otp
		otpStatus := OtpStatusNone
		clientIp := rc.GinCtx.ClientIP()
		rc.ReqParam = fmt.Sprintf("oauth2 login username: %s | ip: %s", account.Username, clientIp)

		res := map[string]any{
			"name":          account.Name,
			"username":      account.Username,
			"lastLoginTime": account.LastLoginTime,
			"lastLoginIp":   account.LastLoginIp,
		}

		accountLoginSecurity := a.ConfigApp.GetConfig(entity.ConfigKeyAccountLoginSecurity).ToAccountLoginSecurity()
		// 判断otp
		if accountLoginSecurity.UseOtp {
			otpInfo, otpurl, otpToken := useOtp(account, accountLoginSecurity.OtpIssuer, accessToken)
			otpStatus = otpInfo.OptStatus
			if otpurl != "" {
				res["otpUrl"] = otpurl
			}
			accessToken = otpToken
		} else {
			// 保存登录消息
			go saveLogin(a.AccountApp, a.MsgApp, account, rc.GinCtx.ClientIP())
		}
		// 赋值otp状态
		res["action"] = "oauthLogin"
		res["otp"] = otpStatus
		res["token"] = accessToken
		b, err = json.Marshal(res)
		biz.ErrIsNil(err, "数据序列化失败")
		rc.GinCtx.Header("Content-Type", "text/html; charset=utf-8")
		rc.GinCtx.Writer.WriteHeader(http.StatusOK)
		_, _ = rc.GinCtx.Writer.WriteString("<html>" +
			"<script>top.opener.postMessage(" + string(b) + ")</script>" +
			"</html>")
	} else if sAccountId, ok := strings.CutPrefix(stateAction, "bind:"); ok {
		// 绑定
		accountId, err := strconv.ParseUint(sAccountId, 10, 64)
		if err != nil {
			biz.ErrIsNil(err, "绑定用户失败: "+err.Error())
		}
		now := time.Now()
		if err := a.AuthApp.BindOAuthAccount(&entity.OAuthAccount{
			AccountId:  accountId,
			Identity:   oauthAccount.Identity,
			CreateTime: &now,
			UpdateTime: &now,
		}); err != nil {
			biz.ErrIsNil(err, "绑定用户失败: "+err.Error())
		}
		res := map[string]any{
			"action": "oauthBind",
			"bind":   true,
		}
		b, err = json.Marshal(res)
		biz.ErrIsNil(err, "数据序列化失败")
		rc.GinCtx.Header("Content-Type", "text/html; charset=utf-8")
		rc.GinCtx.Writer.WriteHeader(http.StatusOK)
		_, _ = rc.GinCtx.Writer.WriteString("<html>" +
			"<script>top.opener.postMessage(" + string(b) + ")</script>" +
			"</html>")
	} else {
		biz.ErrIsNil(errors.New("state不合法"), "state不合法")
	}
}

func (a *Auth) getOAuthClient() (*oauth2.Config, *vo.OAuth2VO, error) {
	config := a.ConfigApp.GetConfig(AuthOAuth2Key)
	oauth2Vo := &vo.OAuth2VO{}
	if config.Value != "" {
		if err := json.Unmarshal([]byte(config.Value), oauth2Vo); err != nil {
			global.Log.Warnf("解析自定义oauth2配置失败，err：%s", err.Error())
			return nil, nil, errors.New("解析自定义oauth2配置失败")
		}
	}
	if oauth2Vo.ClientID == "" {
		biz.ErrIsNil(nil, "请先配置oauth2")
		return nil, nil, errors.New("请先配置oauth2")
	}
	client := &oauth2.Config{
		ClientID:     oauth2Vo.ClientID,
		ClientSecret: oauth2Vo.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  oauth2Vo.AuthorizationURL,
			TokenURL: oauth2Vo.AccessTokenURL,
		},
		RedirectURL: oauth2Vo.RedirectURL + "api/sys/auth/oauth2/callback",
		Scopes:      strings.Split(oauth2Vo.Scopes, ","),
	}
	return client, oauth2Vo, nil
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

func (a *Auth) OAuth2Bind(rc *req.Ctx) {
	client, _, err := a.getOAuthClient()
	if err != nil {
		biz.ErrIsNil(err, "获取oauth2 client失败: "+err.Error())
		return
	}
	state := utils.RandString(32)
	cache.SetStr("oauth2:state:"+state, "bind:"+strconv.FormatUint(rc.LoginAccount.Id, 10),
		5*time.Minute)
	rc.GinCtx.Redirect(http.StatusFound, client.AuthCodeURL(state))
}

func (a *Auth) Auth2Status(ctx *req.Ctx) {
	res := &vo.AuthStatusVO{}
	config := a.ConfigApp.GetConfig(AuthOAuth2Key)
	if config.Value != "" {
		oauth2 := &vo.OAuth2VO{}
		if err := json.Unmarshal([]byte(config.Value), oauth2); err != nil {
			global.Log.Warnf("解析自定义oauth2配置失败，err：%s", err.Error())
			biz.ErrIsNil(err, "解析自定义oauth2配置失败")
		} else if oauth2.ClientID != "" {
			res.Enable.OAuth2 = true
		}
	}
	if res.Enable.OAuth2 {
		err := a.AuthApp.GetOAuthAccount(&entity.OAuthAccount{
			AccountId: ctx.LoginAccount.Id,
		}, "account_id", "identity")
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				biz.ErrIsNil(err, "查询用户失败: "+err.Error())
			}
		} else {
			res.Bind.OAuth2 = true
		}
	}
	ctx.ResData = res
}
