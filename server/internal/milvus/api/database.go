package api

import (
	"mayfly-go/internal/milvus/api/form"
	"mayfly-go/internal/milvus/mvm"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

func (c *Collection) ListDatabases(rc *req.Ctx) {
	client := c.getClient(rc)
	dbs, err := client.ListDatabases()
	biz.ErrIsNil(err)
	rc.ResData = dbs
}

func (c *Collection) CreateDatabase(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.CreateDatabaseForm]()
	err := client.CreateDatabase(param)
	biz.ErrIsNil(err)
}

func (c *Collection) DropDatabase(rc *req.Ctx) {
	client := c.getClient(rc)
	database := rc.PathParam("database")
	err := client.DropDatabase(database)
	biz.ErrIsNil(err)
	mvm.CloseConn(client.Id, database, rc.Query("ac"))
}

func (c *Collection) DescribeDatabase(rc *req.Ctx) {
	client := c.getClient(rc)
	database := rc.PathParam("database")
	var err error
	rc.ResData, err = client.DescribeDatabase(database)
	biz.ErrIsNil(err)
}

func (c *Collection) Properties(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.CreateDatabaseForm]()
	err := client.AlterDatabase(param)
	biz.ErrIsNil(err)
}
