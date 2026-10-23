package api

import (
	"encoding/json"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/importer"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/req"

	"github.com/spf13/cast"
)

// 表格导入文件体积上限：
//   - CSV/TSV/TXT 走逐行流式解析，内存与文件大小无关，上限主要防止临时盘被单个上传占满；
//   - Excel 的 excelize.OpenReader 会把整个文件读入内存（行迭代只省「行」不省「字节」），且两遍导入
//     各打开一次，上限必须按内存风险定得更小。
//
// 这是字节护栏而非行数上限：512MB CSV 仍可包含数百万行。
const (
	maxImportCSVBytes   = 512 << 20 // 512MB
	maxImportExcelBytes = 100 << 20 // 100MB
)

// checkImportFileSize 按扩展名选择体积上限并预检声明大小（声明可伪造，解析器仍会真实读取，
// 但超大上传在进入解析前就会被拒，避免临时盘/内存被单个请求打满）。
func checkImportFileSize(fileHeader *multipart.FileHeader) {
	limit := int64(maxImportCSVBytes)
	switch strings.ToLower(filepath.Ext(fileHeader.Filename)) {
	case ".xlsx", ".xlsm":
		limit = maxImportExcelBytes
	}
	if fileHeader.Size > limit {
		biz.ErrIsNil(errorx.NewBizf("import file [%s] is too large: %d bytes exceeds limit %d bytes", fileHeader.Filename, fileHeader.Size, limit))
	}
}

// PreviewImportData 解析上传的表格文件，返回表头与样本行，供前端构建「文件列 → 表列」映射界面。
//
// 仅做文件解析，不连接目标库、不落库，因此不需要数据权限校验（真正的写入在 ImportData 中鉴权）。
func (d *Db) PreviewImportData(rc *req.Ctx) {
	fileHeader, err := rc.FormFile("file")
	biz.ErrIsNilAppendErr(err, "read upload file error: %s")
	checkImportFileSize(fileHeader)

	file, err := fileHeader.Open()
	biz.ErrIsNilAppendErr(err, "open upload file error: %s")
	defer file.Close()

	rc.ReqParam = "preview import file: " + fileHeader.Filename

	res, err := d.dbDataImportApp.Preview(rc.MetaCtx, &dto.ImportPreviewReq{
		Reader:   file,
		Filename: fileHeader.Filename,
		Options:  bindImportOptions(rc),
	})
	biz.ErrIsNil(err)
	rc.ResData = res
}

// ImportData 按列映射把表格文件数据批量导入目标表：解析文件 → 列映射与类型转换 → 方言批量 INSERT → 事务写入。
//
// 与「执行SQL文件」不同，本入口消费表格数据而非 SQL 文本；写库前执行与 SQL 执行一致的数据权限校验。
func (d *Db) ImportData(rc *req.Ctx) {
	dbId := getDbId(rc)
	dbName := rc.PostForm("db")
	biz.NotEmpty(dbName, "db cannot be empty")
	tableName := rc.PostForm("table")
	biz.NotEmpty(tableName, "table cannot be empty")

	dbConn, err := d.dbApp.GetDbConn(rc.MetaCtx, dbId, dbName)
	biz.ErrIsNil(err)
	biz.ErrIsNilAppendErr(d.tagApp.CanAccess(rc.GetLoginAccount().Id, dbConn.Info.CodePath...), "%s")

	fileHeader, err := rc.FormFile("file")
	biz.ErrIsNilAppendErr(err, "read upload file error: %s")
	checkImportFileSize(fileHeader)
	file, err := fileHeader.Open()
	biz.ErrIsNilAppendErr(err, "open upload file error: %s")
	defer file.Close()

	rc.ReqParam = "import file: " + fileHeader.Filename + " -> " + dbConn.Info.GetLogDesc() + "." + tableName

	res, err := d.dbDataImportApp.Import(rc.MetaCtx, &dto.DataImportReq{
		DbConn:            dbConn,
		TableName:         tableName,
		Reader:            file,
		Filename:          fileHeader.Filename,
		Options:           bindImportOptions(rc),
		Columns:           bindImportColumns(rc),
		EmptyAsNull:       parseBoolParam(rc.PostForm("emptyAsNull")),
		DuplicateStrategy: cast.ToInt(rc.PostForm("duplicateStrategy")),
		BatchSize:         cast.ToInt(rc.PostForm("batchSize")),
		ClientId:          rc.Query("clientId"),
		UploadId:          rc.PostForm("uploadId"),
	})
	biz.ErrIsNil(err)
	rc.ResData = res
}

// bindImportOptions 读取解析相关表单参数：是否有表头、CSV 分隔符、Excel 工作表名。
func bindImportOptions(rc *req.Ctx) *importer.Options {
	return &importer.Options{
		HasHeader:      parseBoolParam(rc.PostForm("hasHeader")),
		FieldSeparator: rc.PostForm("separator"),
		Sheet:          rc.PostForm("sheet"),
	}
}

// bindImportColumns 解析列映射 JSON（[{source,target}]）。target 为空表示跳过该文件列。
func bindImportColumns(rc *req.Ctx) []dto.ImportColumn {
	raw := rc.PostForm("columns")
	biz.NotEmpty(raw, "column mapping cannot be empty")
	var columns []dto.ImportColumn
	biz.ErrIsNilAppendErr(json.Unmarshal([]byte(raw), &columns), "invalid column mapping: %s")
	return columns
}

// parseBoolParam 解析布尔表单参数：前端由 String(bool) 生成，直接用标准库 strconv.ParseBool；
// 缺失/非法一律视为 false（ParseBool 报错时忽略）。不再自写「true/1/on/yes」真值表。
func parseBoolParam(s string) bool {
	v, _ := strconv.ParseBool(s)
	return v
}
