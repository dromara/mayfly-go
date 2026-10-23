package api

import (
	"mayfly-go/internal/milvus/api/form"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

func (c *Collection) ListPartitions(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	rc.ResData, rc.Error = client.ShowPartitions(collection)
}

func (c *Collection) LoadPartitions(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	param := rc.BindJson[form.LoadPartitionForm]()
	rc.Error = client.LoadPartitions(collection, param.PartitionNames)
}
func (c *Collection) ReleasePartitions(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	param := rc.BindJson[form.ReleasePartitionForm]()
	rc.Error = client.ReleasePartitions(collection, param.PartitionNames)
}

func (c *Collection) CreatePartition(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.CreatePartitionForm]()
	collection := rc.PathParam("collection")
	err := client.CreatePartition(collection, param.Name)
	biz.ErrIsNil(err)
}

func (c *Collection) DropPartition(rc *req.Ctx) {
	client := c.getClient(rc)
	partition := rc.PathParam("partition")
	collection := rc.PathParam("collection")
	err := client.DropPartition(collection, partition)
	biz.ErrIsNil(err)
}

func (c *Collection) HasPartition(rc *req.Ctx) {
	client := c.getClient(rc)
	partition := rc.PathParam("partition")
	collection := rc.PathParam("collection")
	exists, err := client.HasPartition(collection, partition)
	biz.ErrIsNil(err)
	rc.ResData = exists
}
