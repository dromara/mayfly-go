package api

import (
	"context"
	"fmt"
	"mayfly-go/internal/auth/api/form"
	"mayfly-go/internal/auth/config"
	"mayfly-go/internal/auth/imsg"
	msgapp "mayfly-go/internal/msg/application"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/captcha"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/otp"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/cryptox"
	"mayfly-go/pkg/ws"
	"strconv"
	"time"
)

type AccountLogin struct {
	AccountApp sysapp.Account `inject:""`
	MsgApp     msgapp.Msg     `inject:""`
}

/**   用户账号密码登录   **/

// @router /auth/accounts/login [post]
func (a *AccountLogin) Login(rc *req.Ctx) {
	loginForm := req.BindJsonAndValid(rc, new(form.LoginForm))
	ctx := rc.MetaCtx

	accountLoginSecurity := config.GetAccountLoginSecurity()
	// 判断是否有开启登录验证码校验
	if accountLoginSecurity.UseCaptcha {
		// 校验验证码
		biz.IsTrueI(ctx, captcha.Verify(loginForm.Cid, loginForm.Captcha), imsg.ErrCaptchaErr)
	}

	username := loginForm.Username

	clientIp := getIpAndRegion(rc)
	rc.ReqParam = collx.Kvs("username", username, "ip", clientIp)

	originPwd, err := cryptox.DefaultRsaDecrypt(loginForm.Password, true)
	biz.ErrIsNilAppendErr(err, "decryption password error: %s")

	account := &sysentity.Account{Username: username}
	err = a.AccountApp.GetByCond(model.NewModelCond(account).Columns("Id", "Name", "Username", "Password", "Status", "LastLoginTime", "LastLoginIp", "OtpSecret"))

	failCountKey := fmt.Sprintf("account:login:failcount:%s", username)
	nowFailCount := cache.GetInt(failCountKey)
	loginFailCount := accountLoginSecurity.LoginFailCount
	loginFailMin := accountLoginSecurity.LoginFailMin
	biz.IsTrueI(ctx, nowFailCount < loginFailCount, imsg.ErrLoginRestrict, "failCount", loginFailCount, "min", loginFailMin)

	if err != nil || !cryptox.CheckPwdHash(originPwd, account.Password) {
		nowFailCount++
		cache.SetStr(failCountKey, strconv.Itoa(nowFailCount), time.Minute*time.Duration(loginFailMin))
		panic(errorx.NewBizI(ctx, imsg.ErrLoginFail, "failCount", nowFailCount))
	}

	// 校验密码强度（新用户第一次登录密码与账号名一致）
	// biz.IsTrueBy(utils.CheckAccountPasswordLever(originPwd), errorx.NewBizCode(401, "您的密码安全等级较低，请修改后重新登录"))
	rc.ResData = LastLoginCheck(ctx, account, accountLoginSecurity, clientIp)
}

type OtpVerifyInfo struct {
	AccountId    uint64
	Username     string
	OptStatus    int
	AccessToken  string
	RefreshToken string
	OtpSecret    string
}

// OTP双因素校验
func (a *AccountLogin) OtpVerify(rc *req.Ctx) {
	otpVerify := new(form.OtpVerfiy)
	req.BindJsonAndValid(rc, otpVerify)
	ctx := rc.MetaCtx

	tokenKey := fmt.Sprintf("otp:token:%s", otpVerify.OtpToken)
	otpInfo := new(OtpVerifyInfo)
	ok := cache.Get(tokenKey, otpInfo)
	biz.IsTrueI(ctx, ok, imsg.ErrOtpTokenInvalid)

	failCountKey := fmt.Sprintf("account:otp:failcount:%d", otpInfo.AccountId)
	failCount := cache.GetInt(failCountKey)
	biz.IsTrueI(ctx, failCount < 5, imsg.ErrOtpCheckRestrict)

	otpStatus := otpInfo.OptStatus
	accessToken := otpInfo.AccessToken
	accountId := otpInfo.AccountId
	otpSecret := otpInfo.OtpSecret

	if !otp.Validate(otpVerify.Code, otpSecret) {
		cache.SetStr(failCountKey, strconv.Itoa(failCount+1), time.Minute*time.Duration(10))
		panic(errorx.NewBizI(ctx, imsg.ErrOtpCheckFail))
	}

	// 如果是未注册状态，则更新account表的otpSecret信息
	if otpStatus == OtpStatusNoReg {
		update := &sysentity.Account{OtpSecret: otpSecret}
		update.Id = accountId
		biz.ErrIsNil(update.OtpSecretEncrypt())
		biz.ErrIsNil(a.AccountApp.Update(context.Background(), update))
	}

	la := &sysentity.Account{Username: otpInfo.Username}
	la.Id = accountId
	go saveLogin(ctx, la, getIpAndRegion(rc))

	cache.Del(tokenKey)
	rc.ResData = collx.Kvs("token", accessToken, "refresh_token", otpInfo.RefreshToken)
}

func (a *AccountLogin) RefreshToken(rc *req.Ctx) {
	refreshToken := rc.Query("refresh_token")
	biz.NotEmpty(refreshToken, "refresh_token cannot be empty")

	accountId, username, err := req.ParseToken(refreshToken)
	biz.IsTrueBy(err == nil, errorx.PermissionErr)

	token, refreshToken, err := req.CreateToken(accountId, username)
	biz.ErrIsNil(err)
	rc.ResData = collx.Kvs("token", token, "refresh_token", refreshToken)
}

func (a *AccountLogin) Logout(rc *req.Ctx) {
	la := rc.GetLoginAccount()
	req.GetPermissionCodeRegistery().Remove(la.Id)
	ws.CloseClient(ws.UserId(la.Id))
}
