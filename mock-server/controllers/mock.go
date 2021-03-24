package controllers

import (
	"encoding/json"
	"mayfly-go/base"
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/rediscli"
	"mayfly-go/base/utils"
	"mayfly-go/mock-server/controllers/form"
)

const key = "mock:data"

type MockController struct {
	base.Controller
}

// @router /api/mock-datas/:method [get]
func (c *MockController) GetMockData() {
	c.ReturnData(ctx.NewNoLogReqCtx(false), func(account *ctx.LoginAccount) interface{} {
		val := rediscli.HGet(key, c.Ctx.Input.Param(":method"))
		mockData := &form.MockData{}
		json.Unmarshal([]byte(val), mockData)
		biz.IsTrue(mockData.Enable == 1, "无该mock数据")

		eu := mockData.EffectiveUser
		// 如果设置的生效用户为空，则表示所有用户都生效
		if len(eu) == 0 {
			return mockData.Data
		}

		// 该mock数据需要指定的生效用户才可访问
		username := c.GetString("username")
		biz.IsTrue(utils.StrLen(username) != 0, "该用户无法访问该mock数据")
		for _, e := range eu {
			if username == e {
				return mockData.Data
			}
		}
		panic(biz.NewBizErr("该用户无法访问该mock数据"))
	})
}

// @router /api/mock-datas [put]
func (c *MockController) UpdateMockData() {
	c.Operation(ctx.NewReqCtx(true, "修改mock数据"), func(account *ctx.LoginAccount) {
		mockData := &form.MockData{}
		c.UnmarshalBodyAndValid(mockData)
		val, _ := json.Marshal(mockData)
		rediscli.HSet(key, mockData.Method, val)
	})
}

// @router /api/mock-datas [post]
func (c *MockController) CreateMockData() {
	c.Operation(ctx.NewReqCtx(true, "保存mock数据"), func(account *ctx.LoginAccount) {
		mockData := &form.MockData{}
		c.UnmarshalBodyAndValid(mockData)
		biz.IsTrue(!rediscli.HExist(key, mockData.Method), "该方法已存在")
		val, _ := json.Marshal(mockData)
		rediscli.HSet(key, mockData.Method, val)
	})
}

// @router /api/mock-datas [get]
func (c *MockController) GetAllData() {
	c.ReturnData(ctx.NewNoLogReqCtx(false), func(account *ctx.LoginAccount) interface{} {
		return rediscli.HGetAll(key)
	})
}

// @router /api/mock-datas/:method [delete]
func (c *MockController) DeleteMockData() {
	c.Operation(ctx.NewReqCtx(false, "删除mock数据"), func(account *ctx.LoginAccount) {
		rediscli.HDel(key, c.Ctx.Input.Param(":method"))
	})
}
