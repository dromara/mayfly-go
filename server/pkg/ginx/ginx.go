package ginx

import (
	"io"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/structx"
	"mayfly-go/pkg/validatorx"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// 绑定并校验请求结构体参数
func BindJsonAndValid[T any](g *gin.Context, data T) T {
	if err := g.ShouldBindJSON(data); err != nil {
		panic(ConvBindValidationError(data, err))
	} else {
		return data
	}
}

// 绑定请求体中的json至form结构体，并拷贝至另一结构体
func BindJsonAndCopyTo[T any](g *gin.Context, form any, toStruct T) T {
	BindJsonAndValid(g, form)
	structx.Copy(toStruct, form)
	return toStruct
}

// 绑定查询字符串到指定结构体
func BindQuery[T any](g *gin.Context, data T) T {
	if err := g.BindQuery(data); err != nil {
		panic(ConvBindValidationError(data, err))
	} else {
		return data
	}
}

// 绑定查询字符串到指定结构体，并将分页信息也返回
func BindQueryAndPage[T any](g *gin.Context, data T) (T, *model.PageParam) {
	if err := g.BindQuery(data); err != nil {
		panic(ConvBindValidationError(data, err))
	} else {
		return data, GetPageParam(g)
	}
}

// 获取分页参数
func GetPageParam(g *gin.Context) *model.PageParam {
	return &model.PageParam{PageNum: QueryInt(g, "pageNum", 1), PageSize: QueryInt(g, "pageSize", 10)}
}

// 获取查询参数，不存在则返回默认值
func Query(g *gin.Context, qm string, defaultStr string) string {
	qv := g.Query(qm)
	if qv == "" {
		return defaultStr
	}
	return qv
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

// 获取路径参数
func PathParamInt(g *gin.Context, pm string) int {
	value, err := strconv.Atoi(g.Param(pm))
	biz.ErrIsNilAppendErr(err, "string类型转换int异常: %s")
	return value
}

// 获取路径参数
func PathParam(g *gin.Context, pm string) string {
	return g.Param(pm)
}

// 文件下载
func Download(g *gin.Context, reader io.Reader, filename string) {
	g.Header("Content-Type", "application/octet-stream")
	g.Header("Content-Disposition", "attachment; filename="+filename)
	io.Copy(g.Writer, reader)
}

// 返回统一成功结果
func SuccessRes(g *gin.Context, data any) {
	g.JSON(http.StatusOK, model.Success(data))
}

// 返回失败结果集
func ErrorRes(g *gin.Context, err any) {
	switch t := err.(type) {
	case errorx.BizError:
		g.JSON(http.StatusOK, model.Error(t))
	default:
		logx.ErrorTrace("服务器错误", t)
		g.JSON(http.StatusOK, model.ServerError())
	}
}

// 转换参数校验错误为业务异常错误
func ConvBindValidationError(data any, err error) error {
	if e, ok := err.(validator.ValidationErrors); ok {
		return errorx.NewBizCode(403, validatorx.Translate2Str(data, e))
	}
	return err
}
