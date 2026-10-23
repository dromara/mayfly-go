package api

import (
	"mayfly-go/internal/milvus/api/form"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"

	"github.com/milvus-io/milvus/client/v2/entity"
)

// AddCollectionField 添加 Collection 字段
func (c *Collection) AddCollectionField(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	param := rc.BindJson[form.AddCollectionFieldRequest]()

	// 将表单字段转换为 entity.Field
	fieldEntity := param.Field
	field := entity.NewField().
		WithName(fieldEntity.Name).
		WithDataType(fieldEntity.DataType).
		WithIsPrimaryKey(fieldEntity.IsPrimaryKey).
		WithIsAutoID(fieldEntity.AutoID).
		WithDescription(fieldEntity.Description).
		WithDim(fieldEntity.Dim).
		WithElementType(fieldEntity.ElementType).
		WithIsDynamic(fieldEntity.IsDynamic).
		WithIsPartitionKey(fieldEntity.IsPartitionKey).
		WithIsClusteringKey(fieldEntity.IsClusteringKey).
		WithMaxLength(fieldEntity.MaxLength).
		WithMaxCapacity(fieldEntity.MaxCapacity)

	field.TypeParams = fieldEntity.TypeParams
	field.IndexParams = fieldEntity.IndexParams

	err := client.AddCollectionField(collection, field)
	biz.ErrIsNil(err)
}

// DropCollectionField 删除 Collection 字段
func (c *Collection) DropCollectionField(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	fieldName := rc.PathParam("field")

	err := client.DropCollectionField(collection, fieldName)
	biz.ErrIsNil(err)
}
func (c *Collection) AlterCollectionFieldProperty(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")
	param := rc.BindJson[form.AlterCollectionFieldForm]()
	err := client.AlterCollectionProperty(collection, param)
	biz.ErrIsNil(err)
}
