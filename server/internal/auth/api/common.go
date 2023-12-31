package api

import (
	"context"
	"fmt"
	"mayfly-go/internal/auth/config"
	msgapp "mayfly-go/internal/msg/application"
	msgentity "mayfly-go/internal/msg/domain/entity"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/otp"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/netx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/timex"
	"time"
)

const (
	OtpStatusNone  = -1 // 未启用otp校验
	OtpStatusReg   = 1  // 用户otp secret已注册
	OtpStatusNoReg = 2  // 用户otp secret未注册
)

// 最后的登录校验（共用）。校验通过返回登录成功响应结果map
func LastLoginCheck(account *sysentity.Account, accountLoginSecurity *config.AccountLoginSecurity, loginIp string) map[string]any {
	biz.IsTrue(account.IsEnable(), "该账号不可用")
	username := account.Username

	res := collx.M{
		"name":          account.Name,
		"username":      username,
		"lastLoginTime": account.LastLoginTime,
		"lastLoginIp":   account.LastLoginIp,
	}

	// 默认为不校验otp
	otpStatus := OtpStatusNone
	// 访问系统使用的token
	accessToken, err := req.CreateToken(account.Id, username)
	biz.ErrIsNilAppendErr(err, "token创建失败: %s")

	// 若系统配置中设置开启otp双因素校验，则进行otp校验
	if accountLoginSecurity.UseOtp {
		otpInfo, otpurl, otpToken := useOtp(account, accountLoginSecurity.OtpIssuer, accessToken)
		otpStatus = otpInfo.OptStatus
		if otpurl != "" {
			res["otpUrl"] = otpurl
		}
		accessToken = otpToken
	} else {
		// 不进行otp二次校验则直接返回accessToken
		// 保存登录消息
		go saveLogin(account, loginIp)
	}

	// 赋值otp状态
	res["otp"] = otpStatus
	res["token"] = accessToken
	return res
}

func useOtp(account *sysentity.Account, otpIssuer, accessToken string) (*OtpVerifyInfo, string, string) {
	biz.ErrIsNil(account.OtpSecretDecrypt())
	otpSecret := account.OtpSecret
	// 修改状态为已注册
	otpStatus := OtpStatusReg
	otpUrl := ""
	// 该token用于otp双因素校验
	token := stringx.Rand(32)
	// 未注册otp secret或重置了秘钥
	if otpSecret == "" || otpSecret == "-" {
		otpStatus = OtpStatusNoReg
		key, err := otp.NewTOTP(otp.GenerateOpts{
			AccountName: account.Username,
			Issuer:      otpIssuer,
		})
		biz.ErrIsNilAppendErr(err, "otp生成失败: %s")
		otpUrl = key.URL()
		otpSecret = key.Secret()
	}
	// 缓存otpInfo, 只有双因素校验通过才可返回真正的accessToken
	otpInfo := &OtpVerifyInfo{
		AccountId:   account.Id,
		Username:    account.Username,
		OptStatus:   otpStatus,
		OtpSecret:   otpSecret,
		AccessToken: accessToken,
	}
	cache.SetStr(fmt.Sprintf("otp:token:%s", token), jsonx.ToStr(otpInfo), time.Minute*time.Duration(3))
	return otpInfo, otpUrl, token
}

// 获取ip与归属地信息
func getIpAndRegion(rc *req.Ctx) string {
	clientIp := rc.GinCtx.ClientIP()
	return fmt.Sprintf("%s %s", clientIp, netx.Ip2Region(clientIp))
}

// 保存更新账号登录信息
func saveLogin(account *sysentity.Account, ip string) {
	// 更新账号最后登录时间
	now := time.Now()
	updateAccount := &sysentity.Account{LastLoginTime: &now}
	updateAccount.Id = account.Id
	updateAccount.LastLoginIp = ip
	// 偷懒为了方便直接获取accountApp
	biz.ErrIsNil(sysapp.GetAccountApp().Update(context.TODO(), updateAccount))

	// 创建登录消息
	loginMsg := &msgentity.Msg{
		RecipientId: int64(account.Id),
		Msg:         fmt.Sprintf("于[%s]-[%s]登录", ip, timex.DefaultFormat(now)),
		Type:        1,
	}
	loginMsg.CreateTime = &now
	loginMsg.Creator = account.Username
	loginMsg.CreatorId = account.Id
	msgapp.GetMsgApp().Create(context.TODO(), loginMsg)
}
