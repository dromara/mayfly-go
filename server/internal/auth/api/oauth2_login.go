package api

import (
	"context"
	"fmt"
	"io"
	"mayfly-go/internal/auth/api/vo"
	"mayfly-go/internal/auth/application"
	"mayfly-go/internal/auth/config"
	"mayfly-go/internal/auth/domain/entity"
	"mayfly-go/internal/auth/imsg"
	msgapp "mayfly-go/internal/msg/application"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/stringx"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

type Oauth2Login struct {
	Oauth2App  application.Oauth2 `inject:""`
	AccountApp sysapp.Account     `inject:""`
	MsgApp     msgapp.Msg         `inject:""`
}

func (a *Oauth2Login) OAuth2Login(rc *req.Ctx) {
	client, _ := a.getOAuthClient()
	state := stringx.Rand(32)
	cache.SetStr("oauth2:state:"+state, "login", 5*time.Minute)
	rc.Redirect(http.StatusFound, client.AuthCodeURL(state))
}

func (a *Oauth2Login) OAuth2Bind(rc *req.Ctx) {
	client, _ := a.getOAuthClient()
	state := stringx.Rand(32)
	cache.SetStr("oauth2:state:"+state, "bind:"+strconv.FormatUint(rc.GetLoginAccount().Id, 10),
		5*time.Minute)
	rc.Redirect(http.StatusFound, client.AuthCodeURL(state))
}

func (a *Oauth2Login) OAuth2Callback(rc *req.Ctx) {
	client, oauth := a.getOAuthClient()

	code := rc.Query("code")
	biz.NotEmpty(code, "code cannot be empty")

	state := rc.Query("state")
	biz.NotEmpty(state, "state canot be empty")

	stateAction := cache.GetStr("oauth2:state:" + state)
	biz.NotEmpty(stateAction, "state已过期, 请重新登录")

	token, err := client.Exchange(rc, code)
	biz.ErrIsNilAppendErr(err, "get OAuth2 accessToken fail: %s")

	// 获取用户信息
	httpCli := client.Client(rc.GetRequest().Context(), token)
	resp, err := httpCli.Get(oauth.ResourceURL)
	biz.ErrIsNilAppendErr(err, "get user info error: %s")
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	biz.ErrIsNilAppendErr(err, "failed to read the response user information: %s")

	// UserIdentifier格式为 type:fieldPath。如：string:user.username 或 number:user.id
	userIdTypeAndFieldPath := strings.Split(oauth.UserIdentifier, ":")
	biz.IsTrue(len(userIdTypeAndFieldPath) == 2, "oauth2 configuration property 'UserIdentifier' is not compliant")

	// 解析用户唯一标识
	userIdFieldPath := userIdTypeAndFieldPath[1]
	userId := ""
	if userIdTypeAndFieldPath[0] == "string" {
		userId, err = jsonx.GetStringByBytes(b, userIdFieldPath)
		biz.ErrIsNilAppendErr(err, "failed to resolve the user unique identity: %s")
	} else {
		intUserId, err := jsonx.GetIntByBytes(b, userIdFieldPath)
		biz.ErrIsNilAppendErr(err, "failed to resolve the user unique identity: %s")
		userId = fmt.Sprintf("%d", intUserId)
	}
	biz.NotBlank(userId, "the user unique identification field value cannot be null")

	// 判断是登录还是绑定
	if stateAction == "login" {
		a.doLoginAction(rc, userId, oauth)
	} else if sAccountId, ok := strings.CutPrefix(stateAction, "bind:"); ok {
		// 绑定
		accountId, err := strconv.ParseUint(sAccountId, 10, 64)
		biz.ErrIsNilAppendErr(err, "failed to bind user: %s")

		account := new(sysentity.Account)
		account.Id = accountId
		err = a.AccountApp.GetByCond(model.NewModelCond(account).Columns("username"))
		biz.ErrIsNilAppendErr(err, "this account does not exist")
		rc.ReqParam = collx.Kvs("username", account.Username, "type", "bind")

		err = a.Oauth2App.GetOAuthAccount(&entity.Oauth2Account{
			AccountId: accountId,
		}, "account_id", "identity")
		biz.IsTrue(err != nil, "the account has been linked by another user")

		err = a.Oauth2App.GetOAuthAccount(&entity.Oauth2Account{
			Identity: userId,
		}, "account_id", "identity")
		biz.IsTrue(err != nil, "you are bound to another account")

		now := time.Now()
		err = a.Oauth2App.BindOAuthAccount(&entity.Oauth2Account{
			AccountId:  accountId,
			Identity:   userId,
			CreateTime: &now,
			UpdateTime: &now,
		})
		biz.ErrIsNilAppendErr(err, "failed to bind user: %s")
		res := collx.M{
			"action": "oauthBind",
			"bind":   true,
		}
		rc.ResData = res
	} else {
		panic(errorx.NewBiz("state is invalid"))
	}
}

// 指定登录操作
func (a *Oauth2Login) doLoginAction(rc *req.Ctx, userId string, oauth *config.Oauth2Login) {
	// 查询用户是否存在
	oauthAccount := &entity.Oauth2Account{Identity: userId}
	err := a.Oauth2App.GetOAuthAccount(oauthAccount, "account_id", "identity")
	ctx := rc.MetaCtx
	var accountId uint64
	isFirst := false
	// 不存在,进行注册
	if err != nil {
		biz.IsTrueI(ctx, oauth.AutoRegister, imsg.ErrOauth2NoAutoRegister)
		now := time.Now()
		account := &sysentity.Account{
			Model: model.Model{
				CreateModel: model.CreateModel{
					CreateTime: &now,
					CreatorId:  0,
					Creator:    "oauth2",
				},
				UpdateTime: &now,
			},
			Name:     userId,
			Username: userId,
		}
		biz.ErrIsNil(a.AccountApp.Create(context.TODO(), account))
		// 绑定
		err := a.Oauth2App.BindOAuthAccount(&entity.Oauth2Account{
			AccountId:  account.Id,
			Identity:   oauthAccount.Identity,
			CreateTime: &now,
			UpdateTime: &now,
		})
		biz.ErrIsNilAppendErr(err, "failed to bind user: %s")
		accountId = account.Id
		isFirst = true
	} else {
		accountId = oauthAccount.AccountId
	}

	// 进行登录
	account, err := a.AccountApp.GetById(accountId, "Id", "Name", "Username", "Password", "Status", "LastLoginTime", "LastLoginIp", "OtpSecret")
	biz.ErrIsNilAppendErr(err, "get user info error: %s")

	clientIp := getIpAndRegion(rc)
	rc.ReqParam = collx.Kvs("username", account.Username, "ip", clientIp, "type", "login")

	res := LastLoginCheck(ctx, account, config.GetAccountLoginSecurity(), clientIp)
	res["action"] = "oauthLogin"
	res["isFirstOauth2Login"] = isFirst
	rc.ResData = res
}

func (a *Oauth2Login) getOAuthClient() (*oauth2.Config, *config.Oauth2Login) {
	oath2LoginConfig := config.GetOauth2Login()
	biz.IsTrue(oath2LoginConfig.Enable, "please configure oauth2 or enable oauth2 login first")
	biz.IsTrue(oath2LoginConfig.ClientId != "", "oauth2 clientId cannot be empty")

	client := &oauth2.Config{
		ClientID:     oath2LoginConfig.ClientId,
		ClientSecret: oath2LoginConfig.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  oath2LoginConfig.AuthorizationURL,
			TokenURL: oath2LoginConfig.AccessTokenURL,
		},
		RedirectURL: oath2LoginConfig.RedirectURL + "/#/oauth2/callback",
		Scopes:      strings.Split(oath2LoginConfig.Scopes, ","),
	}
	return client, oath2LoginConfig
}

func (a *Oauth2Login) Oauth2Status(ctx *req.Ctx) {
	res := &vo.Oauth2Status{}
	oauth2LoginConfig := config.GetOauth2Login()
	res.Enable = oauth2LoginConfig.Enable
	if res.Enable {
		err := a.Oauth2App.GetOAuthAccount(&entity.Oauth2Account{
			AccountId: ctx.GetLoginAccount().Id,
		}, "account_id", "identity")
		res.Bind = err == nil
	}

	ctx.ResData = res
}

func (a *Oauth2Login) Oauth2Unbind(rc *req.Ctx) {
	a.Oauth2App.Unbind(rc.GetLoginAccount().Id)
}

// 获取oauth2登录配置信息，因为有些字段是敏感字段，故单独使用接口获取
func (c *Oauth2Login) Oauth2Config(rc *req.Ctx) {
	oauth2LoginConfig := config.GetOauth2Login()
	rc.ResData = collx.M{
		"enable": oauth2LoginConfig.Enable,
		"name":   oauth2LoginConfig.Name,
	}
}
