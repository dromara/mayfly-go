package entity

import (
	"errors"
	"mayfly-go/internal/common/utils"
	"mayfly-go/pkg/model"

	"github.com/may-fly/cast"
)

const (
	ExtraKeyPassphrase = "passphrase"
)

// 资源授权凭证
type ResourceAuthCert struct {
	model.Model

	Name string `json:"name"` // 名称（全局唯一）

	ResourceCode   string                 `json:"resourceCode"`   // 资源编号
	ResourceType   int8                   `json:"resourceType"`   // 资源类型
	Type           AuthCertType           `json:"type"`           // 凭证类型
	Username       string                 `json:"username"`       // 用户名
	Ciphertext     string                 `json:"ciphertext"`     // 密文
	CiphertextType AuthCertCiphertextType `json:"ciphertextType"` // 密文类型
	Extra          model.Map[string, any] `json:"extra"`          // 账号需要的其他额外信息（如秘钥口令等）
	Remark         string                 `json:"remark"`         // 备注
}

// CiphertextEncrypt 密文加密
func (m *ResourceAuthCert) CiphertextEncrypt() error {
	// 密码替换为加密后的密码
	password, err := utils.PwdAesEncrypt(m.Ciphertext)
	if err != nil {
		return errors.New("加密密文失败")
	}
	m.Ciphertext = password

	// 加密秘钥口令
	if m.CiphertextType == AuthCertCiphertextTypePrivateKey {
		passphrase := m.GetExtraString(ExtraKeyPassphrase)
		if passphrase != "" {
			passphrase, err := utils.PwdAesEncrypt(passphrase)
			if err != nil {
				return errors.New("加密秘钥口令失败")
			}
			m.SetExtra(ExtraKeyPassphrase, passphrase)
		}
	}

	return nil
}

// CiphertextDecrypt 密文解密
func (m *ResourceAuthCert) CiphertextDecrypt() error {
	// 密码替换为解密后的密码
	password, err := utils.PwdAesDecrypt(m.Ciphertext)
	if err != nil {
		return errors.New("解密密文失败")
	}
	m.Ciphertext = password

	// 加密秘钥口令
	if m.CiphertextType == AuthCertCiphertextTypePrivateKey {
		passphrase := m.GetExtraString(ExtraKeyPassphrase)
		if passphrase != "" {
			passphrase, err := utils.PwdAesDecrypt(passphrase)
			if err != nil {
				return errors.New("解密秘钥口令失败")
			}
			m.SetExtra(ExtraKeyPassphrase, passphrase)
		}
	}
	return nil
}

// CiphertextClear 密文清楚
func (m *ResourceAuthCert) CiphertextClear() {
	// 如果密文类型非公共授权凭证，则清空
	if m.CiphertextType != AuthCertCiphertextTypePublic {
		m.Ciphertext = ""
	}
	m.SetExtra(ExtraKeyPassphrase, "")
}

func (m *ResourceAuthCert) SetExtra(key string, val any) {
	if m.Extra != nil {
		m.Extra[key] = val
	}
}

func (m *ResourceAuthCert) GetExtraString(key string) string {
	if m.Extra == nil {
		return ""
	}
	return cast.ToString(m.Extra[key])
}

// HasChanged 与指定授权凭证比较是否有变更
func (m *ResourceAuthCert) HasChanged(rac *ResourceAuthCert) bool {
	if rac == nil {
		return true
	}
	return m.Username != rac.Username ||
		(m.Ciphertext != "" && rac.Ciphertext != "" && m.Ciphertext != rac.Ciphertext) ||
		m.CiphertextType != rac.CiphertextType ||
		m.Remark != rac.Remark ||
		m.Type != rac.Type ||
		m.GetExtraString(ExtraKeyPassphrase) != rac.GetExtraString(ExtraKeyPassphrase)
}

// 密文类型
type AuthCertCiphertextType int8

// 凭证类型
type AuthCertType int8

const (
	AuthCertCiphertextTypePublic     AuthCertCiphertextType = -1 // 公共授权凭证密文
	AuthCertCiphertextTypePassword   AuthCertCiphertextType = 1  // 密码
	AuthCertCiphertextTypePrivateKey AuthCertCiphertextType = 2  // 私钥

	AuthCertTypePrivate        AuthCertType = 1  // 普通私有凭证
	AuthCertTypePrivileged     AuthCertType = 11 // 特权私有凭证
	AuthCertTypePrivateDefault AuthCertType = 12 // 默认私有凭证
	AuthCertTypePublic         AuthCertType = 2  // 公共凭证（可多个资源共享该授权凭证）
)

// 授权凭证接口，填充资源授权凭证信息
type IAuthCert interface {
	// 获取资源code
	GetCode() string

	// 设置授权信息
	SetAuthCert(ac AuthCert)
}

// 资源关联的标签信息
type AuthCert struct {
	Name           string                 `json:"name" gorm:"-"`           // 名称
	Username       string                 `json:"username" gorm:"-"`       // 用户名
	CiphertextType AuthCertCiphertextType `json:"ciphertextType" gorm:"-"` // 密文类型
	Type           AuthCertType           `json:"type" gorm:"-"`           // 凭证类型
}

func (r *AuthCert) SetAuthCert(ac AuthCert) {
	r.Name = ac.Name
	r.Username = ac.Username
	r.Type = ac.Type
	r.CiphertextType = ac.CiphertextType
}

// 资源标签列表
type AuthCerts struct {
	AuthCerts []AuthCert `json:"authCerts" gorm:"-"`
}

func (r *AuthCerts) SetAuthCert(rt AuthCert) {
	if r.AuthCerts == nil {
		r.AuthCerts = make([]AuthCert, 0)
	}
	r.AuthCerts = append(r.AuthCerts, rt)
}
