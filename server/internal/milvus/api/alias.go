package api

import (
	"mayfly-go/internal/milvus/api/form"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

func (c *Collection) ListAliases(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	aliases, err := client.ListAliases(collection)
	biz.ErrIsNil(err)
	rc.ResData = aliases
}

func (c *Collection) CreateAlias(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	param := rc.BindJson[form.CreateAliasForm]()
	err := client.CreateAlias(collection, param.Alias)
	biz.ErrIsNil(err)
}

func (c *Collection) DropAlias(rc *req.Ctx) {
	client := c.getClient(rc)
	alias := rc.PathParam("alias")
	err := client.DropAlias(alias)
	biz.ErrIsNil(err)
}
