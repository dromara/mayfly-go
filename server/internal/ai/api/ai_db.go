package api

import (
	"fmt"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/req"
	"net/http"
	"strings"
	"time"
)

// AiDB API结构体，用于处理AI DB相关请求
type AiDB struct{}

// ReqConfs 获取AI DB相关的请求配置
func (a *AiDB) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 文生SQL接口
		req.NewPost("/sql-gen", a.GenerateSql),
	}

	return req.NewConfs("/ai/db", reqs[:]...)
}

// GenerateSqlResponse 生成SQL响应结果
type GenerateSqlResponse struct {
	Sql string `json:"sql"` // 生成的SQL语句
}

// GenerateSql 根据自然语言生成SQL语句
func (a *AiDB) GenerateSql(rc *req.Ctx) {

	req := struct {
		DbType          string   `json:"dbType" binding:"omitempty"`         // 数据库类型
		NaturalLanguage string   `json:"naturalLanguage" binding:"required"` // 自然语言描述
		Tables          []string `json:"tables"`                             // 相关表名
	}{}
	biz.ErrIsNil(rc.BindJSON(&req))

	// 默认数据库类型
	dbType := req.DbType
	if dbType == "" {
		dbType = "MySQL"
	}

	// 生成提示词
	promptText := generateSqlPrompt(dbType, req.NaturalLanguage, req.Tables)
	logx.Infof("生成的SQL提示词: %s", promptText)

	// 调用AI生成SQL - 这里提供一个模拟实现
	// 在实际项目中，需要调用真实的AI模型API进行SQL生成
	generatedSql := fmt.Sprintf("-- 根据您的需求生成的SQL:\n-- 自然语言描述: %s\n-- 数据库类型: %s\n-- 相关表名: %s\nSELECT * FROM %s WHERE 1=1",
		req.NaturalLanguage, dbType, strings.Join(req.Tables, ", "),
		strings.Join(req.Tables, ", "))
	if len(req.Tables) == 0 {
		generatedSql = fmt.Sprintf("-- 根据您的需求生成的SQL:\n-- 自然语言描述: %s\n-- 数据库类型: %s\nSELECT * FROM users WHERE 1=1",
			req.NaturalLanguage, dbType)
	}

	// 检查是否需要流式输出
	stream := rc.Query("stream")
	logx.Infof("Stream parameter value: '%s'", stream)
	if stream == "true" {
		// 直接使用标准的http.ResponseWriter来处理流式响应
		w := rc.GetWriter()

		// 设置SSE响应头
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		// 确保响应立即发送
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		// 模拟流式输出，逐字符发送SQL内容
		var sqlContent string
		for _, char := range generatedSql {
			sqlContent += string(char)
			// 构建SSE消息
			message := fmt.Sprintf("data: {\"sql\": \"%s\"}\n\n", strings.ReplaceAll(sqlContent, "\"", "\\\""))
			w.Write([]byte(message))

			// 确保响应立即发送
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

			// 模拟AI生成延迟
			time.Sleep(20 * time.Millisecond)
		}

		// 发送结束消息
		endMessage := "data: {\"sql\": null, \"done\": true}\n\n"
		w.Write([]byte(endMessage))

		// 确保响应立即发送
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		rc.Conf.NoRes()

		return
	}

	// 非流式输出，使用标准响应格式
	rc.ResData = &GenerateSqlResponse{Sql: generatedSql}
}

// generateSqlPrompt 生成SQL提示词
func generateSqlPrompt(dbType, text string, tables []string) string {
	// 使用prompt包中的GetPrompt函数获取提示词模板
	// 如果没有找到模板，则使用默认模板
	return ""
}
