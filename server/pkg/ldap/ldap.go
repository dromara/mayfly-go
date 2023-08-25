package ldap

import (
	"crypto/tls"
	"fmt"
	"mayfly-go/pkg/config"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/pkg/errors"
)

type UserInfo struct {
	UserName    string
	DisplayName string
	Email       string
}

func dial() (*ldap.Conn, error) {
	conf := config.Conf.Ldap
	addr := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	tlsConfig := &tls.Config{
		ServerName:         conf.Host,
		InsecureSkipVerify: conf.SkipTLSVerify,
	}
	if conf.SecurityProtocol == config.SecurityProtocolLDAPS {
		conn, err := ldap.DialTLS("tcp", addr, tlsConfig)
		if err != nil {
			return nil, errors.Errorf("dial TLS: %v", err)
		}
		return conn, nil
	}

	conn, err := ldap.Dial("tcp", addr)
	if err != nil {
		return nil, errors.Errorf("dial: %v", err)
	}
	if conf.SecurityProtocol == config.SecurityProtocolStartTLS {
		if err = conn.StartTLS(tlsConfig); err != nil {
			_ = conn.Close()
			return nil, errors.Errorf("start TLS: %v", err)
		}
	}
	return conn, nil
}

// Connect 创建 LDAP 连接
func Connect() (*ldap.Conn, error) {
	conn, err := dial()
	if err != nil {
		return nil, err
	}

	// Bind with a system account
	conf := config.Conf.Ldap
	err = conn.Bind(conf.BindDN, conf.BindPassword)
	if err != nil {
		_ = conn.Close()
		return nil, errors.Errorf("bind: %v", err)
	}
	return conn, nil
}

// Authenticate 通过 LDAP 验证用户名密码
func Authenticate(username, password string) (*UserInfo, error) {
	conn, err := Connect()
	if err != nil {
		return nil, errors.Errorf("connect: %v", err)
	}
	defer func() { _ = conn.Close() }()

	conf := config.Conf.Ldap
	sr, err := conn.Search(
		ldap.NewSearchRequest(
			conf.BaseDN,
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			0,
			0,
			false,
			strings.ReplaceAll(conf.UserFilter, "%s", username),
			[]string{"dn", conf.FieldMapping.Identifier, conf.FieldMapping.DisplayName, conf.FieldMapping.Email},
			nil,
		),
	)
	if err != nil {
		return nil, errors.Errorf("search user DN: %v", err)
	} else if len(sr.Entries) != 1 {
		return nil, errors.Errorf("expect 1 user DN but got %d", len(sr.Entries))
	}
	entry := sr.Entries[0]

	// Bind as the user to verify their password
	err = conn.Bind(entry.DN, password)
	if err != nil {
		return nil, errors.Errorf("bind user: %v", err)
	}

	userName := entry.GetAttributeValue(conf.FieldMapping.Identifier)
	if userName == "" {
		return nil, errors.Errorf("the attribute %q is not found or has empty value", conf.FieldMapping.Identifier)
	}
	return &UserInfo{
		UserName:    userName,
		DisplayName: entry.GetAttributeValue(conf.FieldMapping.DisplayName),
		Email:       entry.GetAttributeValue(conf.FieldMapping.Email),
	}, nil
}
