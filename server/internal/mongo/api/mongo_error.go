package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"mayfly-go/internal/mongo/application"
	"mayfly-go/internal/mongo/application/mongodoc"
	"mayfly-go/internal/mongo/imsg"
	"mayfly-go/internal/mongo/mgm"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/req"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// mongoError 把数据面与编解码错误翻译成带上下文的国际化业务错误。
//
// 应用层只返回 error（分层禁止在 application 断言），面向用户的措辞在 API 层收口；
// 每个分支都对应一种用户能采取的下一步动作，所以不允许统统并成「执行失败」。
func mongoError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, application.ErrDocNotFound):
		return errorx.NewBizI(ctx, imsg.ErrMongoDocNotMatched)
	case errors.Is(err, application.ErrDocConflict):
		return errorx.NewBizI(ctx, imsg.ErrMongoDocConflict)
	case errors.Is(err, application.ErrNothingToWrite):
		return errorx.NewBizI(ctx, imsg.ErrMongoDocsEmpty)
	case errors.Is(err, mongodoc.ErrIDTokenInvalid):
		return errorx.NewBizI(ctx, imsg.ErrMongoIdTokenInvalid)
	case errors.Is(err, mongodoc.ErrIDTokenMissing):
		return errorx.NewBizI(ctx, imsg.ErrMongoIdTokenMissing)
	case errors.Is(err, mongodoc.ErrEmptyCommand):
		return errorx.NewBizI(ctx, imsg.ErrMongoCommandEmpty)
	case errors.Is(err, mongodoc.ErrNoUpdateOperators):
		return errorx.NewBizI(ctx, imsg.ErrMongoUpdateOperatorForbidden)
	case errors.Is(err, mongodoc.ErrUpdateSpecInvalid):
		return errorx.NewBizI(ctx, imsg.ErrMongoUpdateOperatorForbidden)
	case errors.Is(err, mongodoc.ErrPipelineInvalid), errors.Is(err, application.ErrEmptyPipeline):
		return errorx.NewBizI(ctx, imsg.ErrMongoPipelineInvalid)
	case errors.Is(err, application.ErrEmptyIndexSpecs):
		return errorx.NewBizI(ctx, imsg.ErrMongoIndexSpecsInvalid)
	}

	// 认证/权限拒绝单独成一条：它的修法是「补连接串凭证或换账号」，
	// 混进通用「执行失败」里，用户会去反复改命令文本而不是改配置。
	if errors.Is(err, mgm.ErrNoCredentials) || mgm.IsUnauthorized(err) {
		return errorx.NewBizI(ctx, imsg.ErrMongoUnauthorized, "detail", err.Error())
	}

	// 批量操作的命中数不一致需要把两个数字都告知用户，否则他只看到「失败」而不知该确认多少条
	var mismatch *application.BatchCountMismatchError
	if errors.As(err, &mismatch) {
		return errorx.NewBizI(ctx, imsg.ErrMongoBatchCountMismatch,
			"expect", mismatch.Expect, "actual", mismatch.Actual)
	}

	var timeoutErr *application.ExecTimeoutError
	if errors.As(err, &timeoutErr) {
		return errorx.NewBizI(ctx, imsg.ErrMongoExecTimeout, "limit", timeoutErr.Seconds)
	}

	return errorx.NewBizI(ctx, imsg.ErrMongoExecFailed, "detail", err.Error())
}

// decodeJSONField 解码请求里的 JSON 文档字段（filter/sort/projection/doc）。
//
// 解码失败必须把出错的具体字段名与原因带出来：旧实现只回一句「filter或sort字段json字符串值错误」，
// 用户无法判断是哪个字段、错在哪一行。
func decodeJSONField(rc *req.Ctx, raw json.RawMessage, field string) bson.D {
	doc, err := mongodoc.Decode(raw)
	biz.ErrIsNil(decodeError(rc.MetaCtx, field, err))
	return doc
}

// decodeJSONDocs 解码「文档或文档数组」。
func decodeJSONDocs(rc *req.Ctx, raw json.RawMessage, field string) []bson.D {
	docs, err := mongodoc.DecodeDocuments(raw)
	biz.ErrIsNil(decodeError(rc.MetaCtx, field, err))
	return docs
}

func decodeError(ctx context.Context, field string, err error) error {
	if err == nil {
		return nil
	}
	if !errors.Is(err, mongodoc.ErrDecodeDocument) {
		return mongoError(ctx, err)
	}

	// ErrDecodeDocument 包装文本里的「mongodoc: ... : 」前缀对用户没有信息量，只取原因部分
	return errorx.NewBizI(ctx, imsg.ErrMongoJsonInvalid, "field", field, "detail", decodeDetail(err))
}

// decodeDetail 提取解析失败的原因部分，去掉包装层的包名前缀。
func decodeDetail(err error) string {
	if _, cause, found := strings.Cut(err.Error(), mongodoc.ErrDecodeDocument.Error()+": "); found {
		return cause
	}
	return strings.TrimSpace(err.Error())
}
