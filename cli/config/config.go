package config

import (
	"fmt"
	"mayfly-go/cli/i18n"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultFileName 默认配置文件名（位于用户主目录下）
const DefaultFileName = ".mayfly-cli.yaml"

// DefaultPath 返回默认配置文件的完整路径
func DefaultPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("%s: %w", i18n.T(i18n.MsgCfgHomeDirFailed), err)
	}
	return filepath.Join(homeDir, DefaultFileName), nil
}

type Config struct {
	Server       string `yaml:"server"`
	Token        string `yaml:"token"`
	RefreshToken string `yaml:"refresh_token,omitempty"`

	// 数据库默认配置
	DefaultDB string `yaml:"default_db,omitempty"`

	// SSH 默认配置
	DefaultSSH string `yaml:"default_ssh,omitempty"`

	// 当前使用的 profile
	CurrentProfile string `yaml:"current_profile,omitempty"`

	// 多服务器配置
	Profiles map[string]*Profile `yaml:"profiles,omitempty"`
}

// Profile 服务器配置档案
type Profile struct {
	Server       string `yaml:"server"`
	Token        string `yaml:"token,omitempty"`
	RefreshToken string `yaml:"refresh_token,omitempty"`
}

// GetActiveProfile 获取当前活动的 profile 配置
func (c *Config) GetActiveProfile() *Profile {
	if c.CurrentProfile == "" || c.Profiles == nil {
		return nil
	}
	return c.Profiles[c.CurrentProfile]
}

// SetProfile 设置指定名称的 profile
func (c *Config) SetProfile(name string, p *Profile) {
	if c.Profiles == nil {
		c.Profiles = make(map[string]*Profile)
	}
	c.Profiles[name] = p
}

// ListProfiles 列出所有 profile 名称
func (c *Config) ListProfiles() []string {
	if c.Profiles == nil {
		return nil
	}
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	return names
}

func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		var err error
		if configPath, err = DefaultPath(); err != nil {
			return nil, err
		}
	}

	// 如果配置文件不存在，返回空配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T(i18n.MsgCfgReadFailed), err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T(i18n.MsgCfgParseFailed), err)
	}

	return &cfg, nil
}

func SaveConfig(cfg *Config, configPath string) error {
	if configPath == "" {
		var err error
		if configPath, err = DefaultPath(); err != nil {
			return err
		}
	}

	// 校验服务器地址格式
	if cfg.Server != "" {
		if err := ValidateServerURL(cfg.Server); err != nil {
			return err
		}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T(i18n.MsgCfgSerializeFailed), err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("%s: %w", i18n.T(i18n.MsgCfgWriteFailed), err)
	}

	return nil
}

// ValidateServerURL 校验服务器地址格式
func ValidateServerURL(server string) error {
	if !strings.HasPrefix(server, "http://") && !strings.HasPrefix(server, "https://") {
		return fmt.Errorf("%s", i18n.T(i18n.MsgCfgInvalidURL))
	}
	if _, err := url.ParseRequestURI(server); err != nil {
		return fmt.Errorf("%s: %w", i18n.T(i18n.MsgCfgInvalidURL), err)
	}
	return nil
}
