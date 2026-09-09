package api

import (
	"archive/zip"
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mayfly-go/internal/es/api/form"
	"mayfly-go/internal/es/api/vo"
	"mayfly-go/internal/es/application"
	"mayfly-go/internal/es/application/dto"
	"mayfly-go/internal/es/domain/entity"
	"mayfly-go/internal/es/esm/esi"
	"mayfly-go/internal/es/imsg"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cast"
	"github.com/xuri/excelize/v2"
)

type Instance struct {
	inst                application.Instance    `inject:"T"`
	tagApp              tagapp.TagTreeReader    `inject:"T"`
	resourceAuthCertApp tagapp.ResourceAuthCert `inject:"T"`
}

func (d *Instance) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{

		// /es/instance 获取实例列表
		req.NewGet("", d.Instances),

		// /es/instance/test-conn 测试连接
		req.NewPost("/test-conn", d.TestConn),

		// /es/instance 添加实例
		req.NewPost("", d.SaveInstance).Log(req.NewLogSaveI(imsg.LogEsInstSave)),

		// /es/instance/:id 删除实例
		req.NewDelete(":instanceId", d.DeleteInstance).Log(req.NewLogSaveI(imsg.LogEsInstDelete)),

		// /es/instance/proxy 反向代理es接口请求
		req.NewAny("/proxy/:instanceId/*path", d.Proxy),

		// /es/instance/export/:instanceId 导出索引数据
		req.NewPost("/export/:instanceId", d.ExportData).NoRes(),

		// /es/instance/export/progress/:exportId 查询导出进度
		req.NewGet("/export/progress/:exportId", d.ExportProgress),
	}

	return req.NewConfs("/es/instance", reqs[:]...)
}

func (d *Instance) Instances(rc *req.Ctx) {
	queryCond := rc.BindQuery[entity.InstanceQuery]()

	// 只查询实例，兼容没有录入密码的实例
	instTags := d.tagApp.GetAccountTags(rc.GetLoginAccount().Id, &tagentity.TagTreeQuery{
		TypePaths:     collx.AsArray(tagentity.NewTypePaths(tagentity.TagTypeEsInstance)),
		CodePathLikes: collx.AsArray(queryCond.TagPath),
	})

	// 不存在可操作的数据库，即没有可操作数据
	if len(instTags) == 0 {
		rc.ResData = model.NewEmptyPageResult[any]()
		return
	}
	dbInstCodes := tagentity.GetCodesByCodePaths(tagentity.TagTypeEsInstance, instTags.GetCodePaths()...)
	queryCond.Codes = dbInstCodes

	res, err := d.inst.GetPageList(queryCond)
	biz.ErrIsNil(err)
	resVo := model.PageResultConv[*entity.EsInstance, *vo.InstanceListVO](res)
	instvos := resVo.List

	// 只查询标签
	certTags := d.tagApp.GetAccountTags(rc.GetLoginAccount().Id, &tagentity.TagTreeQuery{
		TypePaths:     collx.AsArray(tagentity.NewTypePaths(tagentity.TagTypeEsInstance, tagentity.TagTypeAuthCert)),
		CodePathLikes: collx.AsArray(queryCond.TagPath),
	})

	// 填充授权凭证信息
	d.resourceAuthCertApp.FillAuthCertByAcNames(tagentity.GetCodesByCodePaths(tagentity.TagTypeAuthCert, certTags.GetCodePaths()...), collx.ArrayMap(instvos, func(vos *vo.InstanceListVO) tagentity.IAuthCert {
		return vos
	})...)

	rc.ResData = resVo
}

func (d *Instance) TestConn(rc *req.Ctx) {
	fm, instance := rc.BindJsonAndCopyTo[form.InstanceForm, entity.EsInstance]()

	var ac *tagentity.ResourceAuthCert
	if len(fm.AuthCerts) > 0 {
		ac = fm.AuthCerts[0]
	}

	res, err := d.inst.TestConn(rc.MetaCtx, instance, ac)
	biz.ErrIsNil(err)
	rc.ResData = res
}
func (d *Instance) SaveInstance(rc *req.Ctx) {
	fm, instance := rc.BindJsonAndCopyTo[form.InstanceForm, entity.EsInstance]()

	rc.ReqParam = fm
	id, err := d.inst.SaveInst(rc.MetaCtx, &dto.SaveEsInstance{
		EsInstance:   instance,
		AuthCerts:    fm.AuthCerts,
		TagCodePaths: fm.TagCodePaths,
	})
	biz.ErrIsNil(err)
	rc.ResData = id
}
func (d *Instance) DeleteInstance(rc *req.Ctx) {
	idsStr := rc.PathParam("instanceId")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		biz.ErrIsNilAppendErr(d.inst.Delete(rc.MetaCtx, cast.ToUint64(v)), "delete db instance failed: %s")
	}
}
func (d *Instance) Proxy(rc *req.Ctx) {
	path := rc.PathParam("path")
	instanceId := getInstanceId(rc)
	// 去掉request中的 id 和 path参数，否则es会报错

	r := rc.GetRequest()
	_ = RemoveQueryParam(r, "id", "path")

	err := d.inst.DoConn(rc.MetaCtx, instanceId, func(conn *esi.EsConn) error {
		conn.Proxy(rc.GetWriter(), r, path)
		return nil
	})

	biz.ErrIsNil(err)
}

func RemoveQueryParam(req *http.Request, paramNames ...string) error {
	parsedURL, err := url.ParseRequestURI(req.RequestURI)
	if err != nil {
		return err
	}
	// Get the query parameters
	queryParams, err := url.ParseQuery(parsedURL.RawQuery)
	if err != nil {
		return err
	}
	// Remove the specified query parameter
	for i := range paramNames {
		delete(queryParams, paramNames[i])
	}
	// Reconstruct the query string
	parsedURL.RawQuery = queryParams.Encode()
	// Update the request URL
	req.URL = parsedURL
	req.RequestURI = parsedURL.String()
	return nil
}

func getInstanceId(rc *req.Ctx) uint64 {
	instanceId := rc.PathParamInt("instanceId")
	biz.IsTrue(instanceId > 0, "instanceId error")
	return uint64(instanceId)
}

// ---- Export progress tracking ----

type exportProgress struct {
	Total     int64  `json:"total"`
	Processed int64  `json:"processed"`
	Phase     string `json:"phase"`
	Done      bool   `json:"done"`
	Error     string `json:"error,omitempty"`
	UpdatedAt int64  `json:"-"`
}

var exportProgressMap sync.Map

func setProgress(id string, p *exportProgress) {
	p.UpdatedAt = time.Now().Unix()
	exportProgressMap.Store(id, p)
}

func getProgress(id string) *exportProgress {
	if v, ok := exportProgressMap.Load(id); ok {
		return v.(*exportProgress)
	}
	return nil
}

func deleteProgress(id string) {
	exportProgressMap.Delete(id)
}

// ExportData 导出索引数据（scroll 全量数据 -> 文件 -> zip -> 下载）
func (d *Instance) ExportData(rc *req.Ctx) {
	instanceId := getInstanceId(rc)
	exportForm := rc.BindJson[form.EsExportForm]()

	rc.ReqParam = exportForm

	err := d.inst.DoConn(rc.MetaCtx, instanceId, func(conn *esi.EsConn) error {
		return doExportData(rc, conn, exportForm)
	})

	biz.ErrIsNil(err)
}

// ExportProgress 查询导出进度
func (d *Instance) ExportProgress(rc *req.Ctx) {
	exportId := rc.PathParam("exportId")
	biz.IsTrue(exportId != "", "exportId is required")

	p := getProgress(exportId)
	if p == nil {
		rc.ResData = &exportProgress{Phase: "unknown", Done: true}
		return
	}
	// Clean up completed/old entries
	if p.Done || time.Now().Unix()-p.UpdatedAt > 600 {
		deleteProgress(exportId)
	}
	rc.ResData = p
}

const scrollSize = 10000
const scrollTimeout = "2m"

func doExportData(rc *req.Ctx, conn *esi.EsConn, exportForm *form.EsExportForm) error {
	idxName := exportForm.IdxName
	exportType := exportForm.ExportType
	exportId := exportForm.ExportId

	// Progress tracking helper
	var progress *exportProgress
	if exportId != "" {
		progress = &exportProgress{Phase: "querying"}
		setProgress(exportId, progress)
		defer func() {
			if p := getProgress(exportId); p != nil && !p.Done {
				p.Done = true
				if r := recover(); r != nil {
					p.Error = fmt.Sprintf("%v", r)
				}
				setProgress(exportId, p)
			}
		}()
	}
	updateProgress := func(phase string, total, processed int64) {
		if progress != nil {
			progress.Phase = phase
			progress.Total = total
			progress.Processed = processed
			setProgress(exportId, progress)
		}
	}

	// Build search body
	searchBody := map[string]any{
		"size": scrollSize,
	}
	if exportForm.SearchQuery != nil {
		if sq, ok := exportForm.SearchQuery.(map[string]any); ok {
			for k, v := range sq {
				searchBody[k] = v
			}
		}
	}
	if _, hasQuery := searchBody["query"]; !hasQuery {
		searchBody["query"] = map[string]any{"match_all": map[string]any{}}
	}

	// When specific fields are requested, use _source filtering to reduce ES data transfer
	predefinedFields := exportForm.Fields
	if len(predefinedFields) > 0 {
		var sourceFields []string
		hasIdField := false
		for _, f := range predefinedFields {
			if f == "_id" {
				hasIdField = true
				continue
			}
			sourceFields = append(sourceFields, f)
		}
		if len(sourceFields) > 0 {
			searchBody["_source"] = sourceFields
		}
		// Build fields list directly (_id first if included)
		var finalFields []string
		if hasIdField {
			finalFields = append(finalFields, "_id")
		}
		sort.Strings(sourceFields)
		finalFields = append(finalFields, sourceFields...)
		if len(finalFields) == 0 {
			return fmt.Errorf("no valid fields selected")
		}
		predefinedFields = finalFields
	}

	// extractHits extracts hit documents from scroll response, adding _id to _source
	fieldSet := make(map[string]struct{})
	extractHits := func(res map[string]any) []map[string]any {
		hitsObj, ok := res["hits"].(map[string]any)
		if !ok {
			return nil
		}
		hitsArr, ok := hitsObj["hits"].([]any)
		if !ok {
			return nil
		}
		result := make([]map[string]any, 0, len(hitsArr))
		for _, h := range hitsArr {
			hit, ok := h.(map[string]any)
			if !ok {
				continue
			}
			source, _ := hit["_source"].(map[string]any)
			if source == nil {
				source = map[string]any{}
			}
			if id, hasId := hit["_id"]; hasId {
				source["_id"] = id
			}
			if predefinedFields == nil {
				for k := range source {
					fieldSet[k] = struct{}{}
				}
			}
			result = append(result, source)
		}
		return result
	}

	// Get accurate total via _count API for progress tracking (scroll's total.value may be capped at 10000)
	var totalHits int64
	countBody := map[string]any{}
	if q, ok := searchBody["query"]; ok {
		countBody["query"] = q
	}
	countPath := fmt.Sprintf("/%s/_count", idxName)
	if countRes, err := conn.Info.ExecApi("post", countPath, countBody, 30); err == nil {
		if v, ok := countRes["count"].(float64); ok {
			totalHits = int64(v)
		}
	}
	updateProgress("querying", totalHits, 0)

	// First scroll request - used to discover field names
	scrollPath := fmt.Sprintf("/%s/_search?scroll=%s", idxName, scrollTimeout)
	res, err := conn.Info.ExecApi("post", scrollPath, searchBody, 120)
	if err != nil {
		return fmt.Errorf("es search failed: %w", err)
	}

	firstHits := extractHits(res)
	if len(firstHits) == 0 {
		return fmt.Errorf("no data to export")
	}

	// Fallback: use scroll response total if _count didn't return a value
	if totalHits == 0 {
		if hitsObj, ok := res["hits"].(map[string]any); ok {
			if totalObj, ok := hitsObj["total"].(map[string]any); ok {
				if v, ok := totalObj["value"].(float64); ok {
					totalHits = int64(v)
				}
			}
		}
	}
	processed := int64(len(firstHits))
	updateProgress("exporting", totalHits, processed)

	scrollId, _ := res["_scroll_id"].(string)
	defer func() {
		// Clean up scroll context on exit
		if scrollId != "" {
			conn.Info.ExecApi("delete", "/_search/scroll", map[string]any{"scroll_id": scrollId})
		}
	}()

	// Collect and sort field names (_id always first)
	var fields []string
	if predefinedFields != nil {
		fields = predefinedFields
	} else {
		if _, hasId := fieldSet["_id"]; hasId {
			fields = append(fields, "_id")
			delete(fieldSet, "_id")
		}
		sortedFields := make([]string, 0, len(fieldSet))
		for f := range fieldSet {
			sortedFields = append(sortedFields, f)
		}
		sort.Strings(sortedFields)
		fields = append(fields, sortedFields...)
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "es-export-*")
	if err != nil {
		return fmt.Errorf("create temp dir failed: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create stream writer for the export type
	dataFileName := idxName + "." + exportTypeExt(exportType)
	dataFilePath := filepath.Join(tmpDir, dataFileName)

	sw, err := newStreamWriter(exportType, dataFilePath, fields)
	if err != nil {
		return err
	}

	// Stream write first batch (no memory accumulation)
	if err := sw.WriteBatch(firstHits); err != nil {
		sw.Close()
		return err
	}

	// Stream write remaining scroll batches
	for scrollId != "" {
		scrollReqBody := map[string]any{
			"scroll":    scrollTimeout,
			"scroll_id": scrollId,
		}
		scrollRes, err := conn.Info.ExecApi("post", "/_search/scroll", scrollReqBody, 120)
		if err != nil {
			break
		}
		hits := extractHits(scrollRes)
		if len(hits) == 0 {
			break
		}
		if err := sw.WriteBatch(hits); err != nil {
			sw.Close()
			return err
		}
		processed += int64(len(hits))
		updateProgress("exporting", totalHits, processed)
		scrollId, _ = scrollRes["_scroll_id"].(string)
	}

	if err := sw.Close(); err != nil {
		return err
	}
	updateProgress("compressing", totalHits, processed)

	// Collect all output files (CSV may produce multiple when exceeding row limit)
	outputFiles := sw.Files()

	// Stream zip directly to HTTP response via pipe (no intermediate zip file on disk)
	pr, pw := io.Pipe()
	pipeErr := make(chan error, 1)
	gox.Go(func() {
		defer pw.Close()
		zw := zip.NewWriter(pw)
		for _, outFile := range outputFiles {
			entryName := filepath.Base(outFile)
			fw, err := zw.Create(entryName)
			if err != nil {
				pipeErr <- err
				return
			}
			f, err := os.Open(outFile)
			if err != nil {
				zw.Close()
				pipeErr <- err
				return
			}
			if _, err := io.Copy(fw, f); err != nil {
				f.Close()
				zw.Close()
				pipeErr <- err
				return
			}
			f.Close()
		}
		pipeErr <- zw.Close()
	})

	rc.Download(pr, idxName+".zip")
	if err := <-pipeErr; err != nil {
		logx.Errorf("es export zip streaming failed: %v", err)
		if progress != nil {
			progress.Error = err.Error()
		}
	}
	if progress != nil {
		progress.Done = true
		progress.Phase = "completed"
		setProgress(exportId, progress)
	}
	return nil
}

// streamWriter defines the interface for streaming export data to disk.
// Each batch of hits is written immediately without accumulating in memory.
type streamWriter interface {
	WriteBatch(hits []map[string]any) error
	Close() error
	// Files returns all output file paths created by the writer.
	// CSV may produce multiple files when row count exceeds maxRowsPerFile.
	Files() []string
}

// ---- CSV stream writer ----

type csvStreamWriter struct {
	file        *os.File
	writer      *csv.Writer
	fields      []string
	filePath    string // original file path (first output file)
	dir         string // directory for output files
	baseName    string // file name without extension (e.g. "myindex")
	ext         string // file extension (e.g. "csv")
	rowNum      int    // current row number (1 = header, 2+ = data)
	fileIndex   int    // current file index (1-based)
	outputFiles []string
}

func newCsvStreamWriter(filePath string, fields []string) (*csvStreamWriter, error) {
	dir := filepath.Dir(filePath)
	ext := filepath.Ext(filePath)
	baseName := strings.TrimSuffix(filepath.Base(filePath), ext)

	w := &csvStreamWriter{
		fields:   fields,
		filePath: filePath,
		dir:      dir,
		baseName: baseName,
		ext:      ext,
	}
	if err := w.addNewFile(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *csvStreamWriter) currentFileName() string {
	if w.fileIndex == 1 {
		return w.baseName + w.ext
	}
	return fmt.Sprintf("%s_%d%s", w.baseName, w.fileIndex, w.ext)
}

func (w *csvStreamWriter) addNewFile() error {
	w.fileIndex++
	newPath := filepath.Join(w.dir, w.currentFileName())
	f, err := os.Create(newPath)
	if err != nil {
		return fmt.Errorf("create csv file failed: %w", err)
	}
	// Write BOM for Excel compatibility
	f.WriteString("\xef\xbb\xbf")
	cw := csv.NewWriter(f)
	// Write header
	if err := cw.Write(w.fields); err != nil {
		f.Close()
		return err
	}
	w.file = f
	w.writer = cw
	w.rowNum = 2 // start from row 2 (after header)
	w.outputFiles = append(w.outputFiles, newPath)
	return nil
}

func (w *csvStreamWriter) WriteBatch(hits []map[string]any) error {
	for _, hit := range hits {
		// Check if current file is full, switch to a new file
		if w.rowNum > maxRowsPerFile {
			w.writer.Flush()
			if err := w.writer.Error(); err != nil {
				w.file.Close()
				return err
			}
			w.file.Close()
			if err := w.addNewFile(); err != nil {
				return err
			}
		}
		row := make([]string, len(w.fields))
		for j, field := range w.fields {
			row[j] = formatValue(hit[field])
		}
		if err := w.writer.Write(row); err != nil {
			return err
		}
		w.rowNum++
	}
	return nil
}

func (w *csvStreamWriter) Close() error {
	w.writer.Flush()
	if err := w.writer.Error(); err != nil {
		w.file.Close()
		return err
	}
	return w.file.Close()
}

func (w *csvStreamWriter) Files() []string {
	return w.outputFiles
}

// ---- JSON stream writer ----

type jsonStreamWriter struct {
	file     *os.File
	buf      *bufio.Writer
	filePath string
	first    bool
}

func newJsonStreamWriter(filePath string) (*jsonStreamWriter, error) {
	f, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("create json file failed: %w", err)
	}
	buf := bufio.NewWriterSize(f, 64*1024)
	if _, err := buf.WriteString("[\n"); err != nil {
		f.Close()
		return nil, err
	}
	return &jsonStreamWriter{file: f, buf: buf, filePath: filePath, first: true}, nil
}

func (w *jsonStreamWriter) WriteBatch(hits []map[string]any) error {
	// Build entire batch into buffer, then flush once
	for _, hit := range hits {
		if !w.first {
			w.buf.WriteString(",\n")
		}
		w.first = false
		w.buf.WriteString("  ")
		b, err := json.Marshal(hit)
		if err != nil {
			return err
		}
		w.buf.Write(b)
	}
	return w.buf.Flush()
}

func (w *jsonStreamWriter) Close() error {
	if _, err := w.buf.WriteString("\n]\n"); err != nil {
		w.file.Close()
		return err
	}
	if err := w.buf.Flush(); err != nil {
		w.file.Close()
		return err
	}
	return w.file.Close()
}

func (w *jsonStreamWriter) Files() []string {
	return []string{w.filePath}
}

// ---- Excel stream writer (excelize StreamWriter) ----

// Max rows per file/sheet (including header row).
// Applies to both Excel (new sheet) and CSV (new file).
const maxRowsPerFile = 1000000

type excelStreamWriter struct {
	file         *excelize.File
	streamWriter *excelize.StreamWriter
	sheetIndex   int
	fields       []string
	rowNum       int
	headerStyle  int
}

func newExcelStreamWriter(filePath string, fields []string) (*excelStreamWriter, error) {
	f := excelize.NewFile()

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})

	w := &excelStreamWriter{file: f, sheetIndex: 0, fields: fields, headerStyle: headerStyle}
	if err := w.addNewSheet(); err != nil {
		f.Close()
		return nil, err
	}
	// Delete the default "Sheet1" AFTER creating "Data" sheet.
	// DeleteSheet is a no-op when SheetCount==1, so we must create
	// a new sheet first to bring the count to 2.
	f.DeleteSheet("Sheet1")
	return w, nil
}

func (w *excelStreamWriter) sheetName() string {
	if w.sheetIndex == 1 {
		return "Data"
	}
	return fmt.Sprintf("Data_%d", w.sheetIndex)
}

func (w *excelStreamWriter) addNewSheet() error {
	w.sheetIndex++
	name := w.sheetName()
	index, _ := w.file.NewSheet(name)
	w.file.SetActiveSheet(index)

	sw, err := w.file.NewStreamWriter(name)
	if err != nil {
		return fmt.Errorf("create excel stream writer failed: %w", err)
	}

	// Write header row with bold style via StreamWriter's StyledCell
	headerRow := make([]any, len(w.fields))
	for i, field := range w.fields {
		headerRow[i] = excelize.Cell{StyleID: w.headerStyle, Value: field}
	}
	if err := sw.SetRow("A1", headerRow); err != nil {
		sw.Flush()
		return err
	}
	w.streamWriter = sw
	w.rowNum = 2 // start from row 2 (after header)
	return nil
}

func (w *excelStreamWriter) WriteBatch(hits []map[string]any) error {
	for _, hit := range hits {
		// Check if current sheet is full, switch to a new sheet
		if w.rowNum > maxRowsPerFile {
			if err := w.streamWriter.Flush(); err != nil {
				return err
			}
			if err := w.addNewSheet(); err != nil {
				return err
			}
		}
		row := make([]any, len(w.fields))
		for j, field := range w.fields {
			row[j] = formatValue(hit[field])
		}
		cellName, _ := excelize.CoordinatesToCellName(1, w.rowNum)
		if err := w.streamWriter.SetRow(cellName, row); err != nil {
			return err
		}
		w.rowNum++
	}
	return nil
}

func (w *excelStreamWriter) Close() error {
	if err := w.streamWriter.Flush(); err != nil {
		w.file.Close()
		return err
	}
	return w.file.Close()
}

func newStreamWriter(exportType, filePath string, fields []string) (streamWriter, error) {
	switch exportType {
	case "csv":
		return newCsvStreamWriter(filePath, fields)
	case "excel":
		sw, err := newExcelStreamWriter(filePath, fields)
		if err != nil {
			return nil, err
		}
		// Override close to save to specific path
		return &excelStreamWriterCloser{sw: sw, filePath: filePath}, nil
	case "json":
		return newJsonStreamWriter(filePath)
	default:
		return nil, fmt.Errorf("unsupported export type: %s", exportType)
	}
}

// excelStreamWriterCloser wraps excelStreamWriter to save to the correct path
type excelStreamWriterCloser struct {
	sw       *excelStreamWriter
	filePath string
}

func (c *excelStreamWriterCloser) WriteBatch(hits []map[string]any) error {
	return c.sw.WriteBatch(hits)
}

func (c *excelStreamWriterCloser) Close() error {
	if err := c.sw.streamWriter.Flush(); err != nil {
		c.sw.file.Close()
		return err
	}
	saveErr := c.sw.file.SaveAs(c.filePath)
	// Always close to release temp files created by bufferedWriter
	c.sw.file.Close()
	return saveErr
}

func (c *excelStreamWriterCloser) Files() []string {
	return []string{c.filePath}
}

func exportTypeExt(exportType string) string {
	switch exportType {
	case "excel":
		return "xlsx"
	default:
		return exportType
	}
}

// formatValue converts any ES field value to its string representation.
// Uses strconv for numbers which is significantly faster than fmt.Sprintf.
func formatValue(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'g', -1, 64)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case json.Number:
		return v.String()
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}
