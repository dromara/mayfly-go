package api

import (
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

// ============ 系统信息 ============

func (c *Collection) GetVersion(rc *req.Ctx) {
	client := c.getClient(rc)
	version, err := client.GetVersion()
	biz.ErrIsNil(err)
	rc.ResData = version
}

func (c *Collection) CheckHealth(rc *req.Ctx) {
	client := c.getClient(rc)
	health, err := client.CheckHealth()
	biz.ErrIsNil(err)
	rc.ResData = health
}
