package api

import (
	"mayfly-go/internal/milvus/api/form"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

func (c *Collection) CreateIndex(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.CreateIndexForm]()
	collection := rc.PathParam("collection")
	field := rc.PathParam("field")
	// TODO: 实现索引创建
	_ = param
	_ = collection
	_ = field
	_ = client
}

func (c *Collection) DescribeIndex(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	index := rc.PathParam("index")
	indexes, err := client.DescribeIndex(collection, index)
	biz.ErrIsNil(err)
	rc.ResData = indexes
}

func (c *Collection) DropIndex(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	field := rc.PathParam("field")
	err := client.DropIndex(collection, field)
	biz.ErrIsNil(err)
}
