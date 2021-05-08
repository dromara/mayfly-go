package ginx

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/mlog"
	"mayfly-go/base/model"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// 绑定并校验请求结构体参数
func BindJsonAndValid(g *gin.Context, data interface{}) {
	if err := g.BindJSON(data); err != nil {
		panic(biz.NewBizErr(err.Error()))
	}
}

// 绑定查询字符串到
func BindQuery(g *gin.Context, data interface{}) {
	if err := g.BindQuery(data); err != nil {
		panic(biz.NewBizErr(err.Error()))
	}
}

// 获取分页参数
func GetPageParam(g *gin.Context) *model.PageParam {
	return &model.PageParam{PageNum: QueryInt(g, "pageNum", 1), PageSize: QueryInt(g, "pageSize", 10)}
}

// 获取查询参数中指定参数值，并转为int
func QueryInt(g *gin.Context, qm string, defaultInt int) int {
	qv := g.Query(qm)
	if qv == "" {
		return defaultInt
	}
	qvi, err := strconv.Atoi(qv)
	biz.ErrIsNil(err, "query param not int")
	return qvi
}

// 文件下载
func Download(g *gin.Context, data []byte, filename string) {
	g.Header("Content-Type", "application/octet-stream")
	g.Header("Content-Disposition", "attachment; filename="+filename)
	g.Data(http.StatusOK, "application/octet-stream", data)
}

// 返回统一成功结果
func SuccessRes(g *gin.Context, data interface{}) {
	g.JSON(http.StatusOK, model.Success(data))
}

// 返回失败结果集
func ErrorRes(g *gin.Context, err interface{}) {
	switch t := err.(type) {
	case *biz.BizError:
		g.JSON(http.StatusOK, model.Error(t.Code(), t.Error()))
		break
	case error:
		g.JSON(http.StatusOK, model.ServerError())
		mlog.Log.Error(t)
		// panic(err)
		break
	case string:
		g.JSON(http.StatusOK, model.ServerError())
		mlog.Log.Error(t)
		// panic(err)
		break
	default:
		mlog.Log.Error(t)
	}
}
