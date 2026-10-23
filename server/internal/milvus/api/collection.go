package api

import (
	"cmp"
	"mayfly-go/internal/milvus/api/form"
	"mayfly-go/internal/milvus/application"
	"mayfly-go/internal/milvus/mvm"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

// Collection Collection 相关 API
type Collection struct {
	milvusApp application.Milvus `inject:"T"`
}

// ReqConfs 注册路由
func (c *Collection) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 数据库操作
		req.NewGet(":id/databases", c.ListDatabases),
		req.NewPost(":id/databases", c.CreateDatabase),
		req.NewDelete(":id/databases/:database", c.DropDatabase),
		req.NewGet(":id/databases/:database/describe", c.DescribeDatabase),
		req.NewPost(":id/databases/:database/properties", c.Properties),

		// Collection 管理
		req.NewGet(":id/collections", c.ListCollections),
		req.NewPost(":id/collections", c.CreateCollection),
		req.NewPost(":id/collections/:collection/alter", c.AlterCollection),
		req.NewDelete(":id/collections/:collection", c.DropCollection),
		req.NewGet(":id/collections/:collection/describe", c.DescribeCollection),
		req.NewGet(":id/collections/:collection/statistics", c.GetCollectionStatistics),
		req.NewPost(":id/collections/:collection/load", c.LoadCollection),
		req.NewPost(":id/collections/:collection/release", c.ReleaseCollection),
		req.NewGet(":id/collections/:collection/has", c.HasCollection),
		req.NewGet(":id/collections/:collection/load-state", c.GetLoadState),

		// 别名管理
		req.NewGet(":id/collections/:collection/aliases", c.ListAliases),
		req.NewPost(":id/collections/:collection/aliases", c.CreateAlias),
		req.NewDelete(":id/aliases/:alias", c.DropAlias),

		// 字段管理
		req.NewPost(":id/collections/:collection/fields", c.AddCollectionField),
		req.NewDelete(":id/collections/:collection/fields/:field", c.DropCollectionField),
		req.NewPost(":id/collections/:collection/fields/:field/alter", c.AlterCollectionFieldProperty),

		// 分区管理
		req.NewGet(":id/collections/:collection/partitions", c.ListPartitions),
		req.NewPost(":id/collections/:collection/partitions", c.CreatePartition),
		req.NewGet(":id/collections/:collection/partitions/load", c.LoadPartitions),
		req.NewGet(":id/collections/:collection/partitions/release", c.ReleasePartitions),
		req.NewDelete(":id/collections/:collection/partitions/:partition", c.DropPartition),
		req.NewGet(":id/collections/:collection/partitions/:partition/has", c.HasPartition),

		// 索引管理
		req.NewGet(":id/collections/:collection/:index/describe", c.DescribeIndex),

		// 数据操作
		req.NewPost(":id/collections/:collection/insert", c.Insert),
		req.NewPost(":id/collections/:collection/delete", c.Delete),
		req.NewPost(":id/collections/:collection/query", c.Query),
		req.NewPost(":id/collections/:collection/search", c.Search),
		req.NewPost(":id/collections/:collection/generate-mock-data", c.GenerateMockData),
		req.NewPost(":id/collections/:collection/insert-sample-data", c.InsertSampleData),
		req.NewPost(":id/collections/:collection/import-file", c.ImportFile),

		// 用户权限
		req.NewGet(":id/users", c.ListUsers),
		req.NewPost(":id/users", c.CreateUser),
		req.NewDelete(":id/users/:username", c.DeleteUser),
		req.NewPost(":id/users/:username/password", c.UpdatePassword),
		req.NewPost(":id/users/:username/grantRole", c.GrantRoleToUser),
		req.NewPost(":id/users/:username/revokeRole", c.RevokeRoleFromUser),

		// 角色管理
		req.NewGet(":id/roles", c.ListRoles),
		req.NewPost(":id/roles", c.UpdateRole),
		req.NewDelete(":id/roles/:role", c.DropRole),
		req.NewGet(":id/roles/:role", c.SelectRole),

		req.NewGet(":id/privilege-group", c.PrivilegeGroup),
		req.NewPost(":id/privilege-group", c.SavePrivilegeGroup),
		req.NewDelete(":id/privilege-group/:name", c.DropPrivilegeGroup),

		// 资源组
		req.NewGet(":id/resource-groups", c.ListResourceGroups),
		req.NewPost(":id/resource-groups", c.CreateResourceGroup),
		req.NewDelete(":id/resource-groups/:name", c.DropResourceGroup),
		req.NewGet(":id/resource-groups/:name/describe", c.DescribeResourceGroup),

		// 系统信息
		req.NewGet(":id/version", c.GetVersion),
		req.NewGet(":id/health", c.CheckHealth),
	}

	return req.NewConfs("milvus", reqs[:]...)
}

func (c *Collection) getClient(rc *req.Ctx) *mvm.MilvusConn {
	client, err := c.milvusApp.GetMilvusConn(rc)
	biz.ErrIsNil(err)
	return client
}

// ============ 数据库操作 ============

func (c *Collection) ListCollections(rc *req.Ctx) {
	client := c.getClient(rc)
	collections, err := client.ListCollections()
	biz.ErrIsNil(err)
	rc.ResData = collections
}

func (c *Collection) CreateCollection(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.CreateCollectionForm]()
	err := client.CreateCollection(param.ToSchema(), cmp.Or(param.ShardsNum, 1))
	biz.ErrIsNil(err)
}

func (c *Collection) AlterCollection(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	param := rc.BindJson[form.AlterCollectionForm]()
	err := client.AlterCollection(collection, param)
	biz.ErrIsNil(err)
}

func (c *Collection) DropCollection(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	err := client.DropCollection(collection)
	biz.ErrIsNil(err)
}

func (c *Collection) DescribeCollection(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	coll, err := client.DescribeCollection(collection)
	biz.ErrIsNil(err)
	rc.ResData = coll
}

func (c *Collection) GetCollectionStatistics(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	stats, err := client.GetCollectionStatistics(collection)
	biz.ErrIsNil(err)
	rc.ResData = stats
}

func (c *Collection) LoadCollection(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	err := client.LoadCollection(collection)
	biz.ErrIsNil(err)
}

func (c *Collection) ReleaseCollection(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	err := client.ReleaseCollection(collection)
	biz.ErrIsNil(err)
}

func (c *Collection) HasCollection(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	exists, err := client.HasCollection(collection)
	biz.ErrIsNil(err)
	rc.ResData = exists
}

func (c *Collection) GetLoadState(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")

	// 通过 describe collection 获取加载状态
	coll, err := client.DescribeCollection(collection)
	biz.ErrIsNil(err)

	rc.ResData = map[string]interface{}{
		"loaded": coll.Loaded,
	}
}
