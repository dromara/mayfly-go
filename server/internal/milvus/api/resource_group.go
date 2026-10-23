package api

import (
	"mayfly-go/internal/milvus/api/form"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

func (c *Collection) ListResourceGroups(rc *req.Ctx) {
	client := c.getClient(rc)
	rgs, err := client.ListResourceGroups()
	biz.ErrIsNil(err)
	rc.ResData = rgs
}

func (c *Collection) CreateResourceGroup(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.CreateResourceGroupForm]()
	err := client.CreateResourceGroup(param.Name)
	biz.ErrIsNil(err)
}

func (c *Collection) DropResourceGroup(rc *req.Ctx) {
	client := c.getClient(rc)
	name := rc.PathParam("name")
	err := client.DropResourceGroup(name)
	biz.ErrIsNil(err)
}

func (c *Collection) DescribeResourceGroup(rc *req.Ctx) {
	client := c.getClient(rc)
	name := rc.PathParam("name")
	rg, err := client.DescribeResourceGroup(name)
	biz.ErrIsNil(err)
	rc.ResData = rg
}
