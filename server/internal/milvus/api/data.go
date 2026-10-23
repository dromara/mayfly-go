package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mayfly-go/internal/milvus/api/form"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"strconv"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

func (c *Collection) Insert(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.InsertForm]()
	collection := rc.PathParam("collection")

	// data 格式: {"data": [{"field1": value1, "field2": value2, ...}, ...]}
	dataSlice, ok := param.Data["data"].([]any)
	if !ok {
		panic("data 字段必须是数组")
	}

	// 转换为 rows 格式用于插入
	rows := make([]any, 0, len(dataSlice))
	for _, item := range dataSlice {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, row)
		}
	}

	// 使用 Milvus SDK 的 row-based 插入
	result, err := client.GetClient().Insert(context.Background(), milvusclient.NewRowBasedInsertOption(collection, rows...))
	biz.ErrIsNil(err)

	rc.ResData = map[string]any{
		"insertCount": result.InsertCount,
	}
}

func (c *Collection) Delete(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.DeleteForm]()
	collection := rc.PathParam("collection")

	var err error
	rc.ResData, err = client.Delete(collection, param.Expr)
	biz.ErrIsNil(err)
}

func (c *Collection) Query(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.QueryForm]()
	collection := rc.PathParam("collection")
	results, err := client.Query(collection, param)
	biz.ErrIsNil(err)
	rc.ResData = results
}

func (c *Collection) Search(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.SearchForm]()
	collection := rc.PathParam("collection")

	var err error
	rc.ResData, err = client.Search(collection, param)
	biz.ErrIsNil(err)
}

// GenerateMockData 生成样本数据
func (c *Collection) GenerateMockData(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.GenerateMockDataForm]()
	collection := rc.PathParam("collection")

	// 获取 Collection 的 Schema 信息
	coll, err := client.DescribeCollection(collection)
	biz.ErrIsNil(err)

	// 生成 Mock 数据
	mockData := generateMockData(coll.Schema.Fields, param.Count)

	rc.ResData = map[string]interface{}{
		"data": mockData,
	}
}

// InsertSampleData 插入样本数据（后端直接 mock 数据并入库）
func (c *Collection) InsertSampleData(rc *req.Ctx) {
	client := c.getClient(rc)
	param := rc.BindJson[form.GenerateMockDataForm]()
	collection := rc.PathParam("collection")

	// 获取 Collection 的 Schema 信息
	coll, err := client.DescribeCollection(collection)
	biz.ErrIsNil(err)

	// 生成 Mock 数据
	mockData := generateMockData(coll.Schema.Fields, param.Count)

	// 直接入库
	var insertResult milvusclient.InsertResult
	ops := milvusclient.NewRowBasedInsertOption(collection, mockData...)
	if param.PartitionName != "" {
		ops.WithPartition(param.PartitionName)
	}
	insertResult, err = client.GetClient().Insert(context.Background(), ops)

	biz.ErrIsNil(err)

	rc.ResData = map[string]interface{}{
		"insertCount": insertResult.InsertCount,
	}
}

// ImportFile 从文件导入数据（支持 CSV 和 JSON）
func (c *Collection) ImportFile(rc *req.Ctx) {
	client := c.getClient(rc)
	collection := rc.PathParam("collection")

	// 获取上传的文件
	fileheader, err := rc.FormFile("file")
	biz.ErrIsNilAppendErr(err, "read form file error: %s")

	file, err := fileheader.Open()
	biz.ErrIsNil(err)
	defer file.Close()

	// 获取分区名称（可选）
	partitionName := rc.PostForm("partitionName")

	// 读取文件内容
	data, err := io.ReadAll(file)
	biz.ErrIsNil(err)

	// 获取集合的 Schema 信息，用于字段类型转换
	coll, err := client.DescribeCollection(collection)
	biz.ErrIsNil(err)

	// 构建字段名到类型的映射
	fieldTypeMap := make(map[string]entity.FieldType)
	for _, field := range coll.Schema.Fields {
		fieldTypeMap[field.Name] = field.DataType
	}

	// 根据文件扩展名解析数据
	fileName := strings.ToLower(fileheader.Filename)
	var rows []any

	switch {
	case strings.HasSuffix(fileName, ".csv"):
		rows = parseCSVData(data, fieldTypeMap)
	case strings.HasSuffix(fileName, ".json"):
		rows = parseJSONData(data, fieldTypeMap)
	default:
		panic("不支持的文件格式，仅支持 CSV 和 JSON")
	}

	if len(rows) == 0 {
		panic("文件中没有有效数据")
	}

	// 插入数据
	ops := milvusclient.NewRowBasedInsertOption(collection, rows...)
	if partitionName != "" {
		ops.WithPartition(partitionName)
	}
	insertResult, err := client.GetClient().Insert(context.Background(), ops)
	biz.ErrIsNil(err)

	rc.ResData = map[string]any{
		"insertCount": insertResult.InsertCount,
	}
}

// parseCSVData 解析 CSV 数据
func parseCSVData(data []byte, fieldTypeMap map[string]entity.FieldType) []any {
	reader := csv.NewReader(bytes.NewReader(data))
	// 允许可变字段数
	reader.FieldsPerRecord = -1

	// 读取表头
	headers, err := reader.Read()
	if err != nil {
		panic("读取 CSV 表头失败: " + err.Error())
	}

	var rows []any
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // 跳过错误行
		}

		row := make(map[string]interface{})
		for i, value := range record {
			if i >= len(headers) {
				break
			}
			fieldName := headers[i]
			fieldType := fieldTypeMap[fieldName]

			// 根据字段类型进行转换
			switch fieldType {
			case entity.FieldTypeInt64:
				if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
					row[fieldName] = intVal
				} else {
					row[fieldName] = value
				}
			case entity.FieldTypeFloat:
				if floatVal, err := strconv.ParseFloat(value, 32); err == nil {
					row[fieldName] = float32(floatVal)
				} else {
					row[fieldName] = value
				}
			case entity.FieldTypeDouble:
				if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
					row[fieldName] = floatVal
				} else {
					row[fieldName] = value
				}
			case entity.FieldTypeBool:
				if boolVal, err := strconv.ParseBool(value); err == nil {
					row[fieldName] = boolVal
				} else {
					row[fieldName] = value
				}
			case entity.FieldTypeFloatVector:
				// 尝试解析为 JSON 数组（用于向量）
				var arr []float64
				if err := json.Unmarshal([]byte(value), &arr); err == nil {
					// 转换为 float32 切片
					float32Arr := make([]float32, len(arr))
					for j, v := range arr {
						float32Arr[j] = float32(v)
					}
					row[fieldName] = float32Arr
				} else {
					row[fieldName] = value
				}
			default:
				// 其他类型（如 VarChar）保持字符串
				row[fieldName] = value
			}
		}
		rows = append(rows, row)
	}

	return rows
}

// parseJSONData 解析 JSON 数据
func parseJSONData(data []byte, fieldTypeMap map[string]entity.FieldType) []any {
	// 尝试解析为对象数组
	var arrayData []map[string]interface{}
	if err := json.Unmarshal(data, &arrayData); err == nil {
		rows := make([]any, len(arrayData))
		for i, row := range arrayData {
			rows[i] = convertJSONRow(row, fieldTypeMap)
		}
		return rows
	}

	// 尝试解析为单行对象
	var singleData map[string]interface{}
	if err := json.Unmarshal(data, &singleData); err == nil {
		return []any{convertJSONRow(singleData, fieldTypeMap)}
	}

	panic("JSON 格式错误，期望对象数组或单个对象")
}

// convertJSONRow 转换 JSON 行数据，根据字段类型进行处理
func convertJSONRow(row map[string]interface{}, fieldTypeMap map[string]entity.FieldType) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range row {
		fieldType := fieldTypeMap[key]

		switch v := value.(type) {
		case float64:
			// 根据字段类型决定是整数还是浮点数
			switch fieldType {
			case entity.FieldTypeInt64:
				result[key] = int64(v)
			case entity.FieldTypeFloat:
				result[key] = float32(v)
			case entity.FieldTypeDouble:
				result[key] = v
			default:
				// 默认保持原样
				if v == float64(int64(v)) {
					result[key] = int64(v)
				} else {
					result[key] = v
				}
			}
		case string:
			// 只有字段类型是 Int64 时才尝试转换
			if fieldType == entity.FieldTypeInt64 {
				if intVal, err := strconv.ParseInt(v, 10, 64); err == nil {
					result[key] = intVal
				} else {
					result[key] = v
				}
			} else {
				result[key] = v
			}
		case []interface{}:
			// 处理数组（可能是向量）
			if len(v) > 0 {
				switch v[0].(type) {
				case float64:
					float32Arr := make([]float32, len(v))
					for i, item := range v {
						if f, ok := item.(float64); ok {
							float32Arr[i] = float32(f)
						}
					}
					result[key] = float32Arr
				default:
					result[key] = v
				}
			} else {
				result[key] = v
			}
		default:
			result[key] = value
		}
	}
	return result
}

// generateMockData 根据字段信息生成 Mock 数据
// 使用 gofakeit 生成更真实的假数据
func generateMockData(fields []*entity.Field, count int) []any {
	var data []any

	for i := 0; i < count; i++ {
		row := make(map[string]interface{})

		for _, field := range fields {
			// 跳过动态字段
			if field.IsDynamic {
				continue
			}

			// 跳过自增字段
			if field.AutoID {
				continue
			}

			// 根据字段类型和名称特征生成数据
			row[field.Name] = generateFieldValue(field, i)
		}

		data = append(data, row)
	}

	return data
}

// generateFieldValue 根据字段类型和名称生成对应的值
func generateFieldValue(field *entity.Field, index int) interface{} {
	fieldNameLower := strings.ToLower(field.Name)

	switch field.DataType {
	case entity.FieldTypeInt64:
		return generateInt64Value(fieldNameLower, index)
	case entity.FieldTypeInt32:
		return generateInt32Value(fieldNameLower)
	case entity.FieldTypeInt16:
		return generateInt16Value(fieldNameLower)
	case entity.FieldTypeInt8:
		return generateInt8Value(fieldNameLower)
	case entity.FieldTypeFloat:
		return generateFloat32Value(fieldNameLower)
	case entity.FieldTypeDouble:
		return generateFloat64Value(fieldNameLower)
	case entity.FieldTypeVarChar:
		return generateStringValue(fieldNameLower, index)
	case entity.FieldTypeBool:
		return gofakeit.Bool()
	case entity.FieldTypeFloatVector:
		return generateFloatVector(field)
	case entity.FieldTypeBinaryVector:
		return generateBinaryVector(field)
	case entity.FieldTypeFloat16Vector:
		return generateFloat16Vector(field)
	case entity.FieldTypeBFloat16Vector:
		return generateBFloat16Vector(field)
	case entity.FieldTypeInt8Vector:
		return generateInt8Vector(field)
	case entity.FieldTypeJSON:
		return generateJSONValue()
	case entity.FieldTypeArray:
		return generateArrayValue(field)
	default:
		return fmt.Sprintf("value_%d", index)
	}
}

// generateInt64Value 生成 Int64 类型的值
func generateInt64Value(fieldName string, index int) int64 {
	switch {
	case strings.Contains(fieldName, "id"):
		return gofakeit.Int64()
	case strings.Contains(fieldName, "time"), strings.Contains(fieldName, "timestamp"):
		return time.Now().Add(-time.Duration(gofakeit.Number(0, 365*24)) * time.Hour).UnixMilli()
	case strings.Contains(fieldName, "count"), strings.Contains(fieldName, "num"):
		return int64(gofakeit.Number(0, 10000))
	default:
		return gofakeit.Int64()
	}
}

// generateInt32Value 生成 Int32 类型的值
func generateInt32Value(fieldName string) int32 {
	switch {
	case strings.Contains(fieldName, "age"):
		return int32(gofakeit.Number(18, 80))
	case strings.Contains(fieldName, "score"), strings.Contains(fieldName, "rating"):
		return int32(gofakeit.Number(1, 100))
	case strings.Contains(fieldName, "level"), strings.Contains(fieldName, "grade"):
		return int32(gofakeit.Number(1, 10))
	default:
		return gofakeit.Int32()
	}
}

// generateInt16Value 生成 Int16 类型的值
func generateInt16Value(fieldName string) int16 {
	return int16(gofakeit.Number(-32768, 32767))
}

// generateInt8Value 生成 Int8 类型的值
func generateInt8Value(fieldName string) int8 {
	return int8(gofakeit.Number(-128, 127))
}

// generateFloat32Value 生成 Float32 类型的值
func generateFloat32Value(fieldName string) float32 {
	switch {
	case strings.Contains(fieldName, "price"):
		return float32(gofakeit.Price(1, 10000))
	case strings.Contains(fieldName, "rate"), strings.Contains(fieldName, "ratio"):
		return float32(gofakeit.Float64Range(0, 1))
	case strings.Contains(fieldName, "percent"):
		return float32(gofakeit.Float64Range(0, 100))
	case strings.Contains(fieldName, "score"):
		return float32(gofakeit.Float64Range(0, 100))
	default:
		return gofakeit.Float32()
	}
}

// generateFloat64Value 生成 Float64 类型的值
func generateFloat64Value(fieldName string) float64 {
	switch {
	case strings.Contains(fieldName, "price"), strings.Contains(fieldName, "amount"), strings.Contains(fieldName, "salary"):
		return gofakeit.Price(1, 100000)
	case strings.Contains(fieldName, "latitude"):
		return gofakeit.Latitude()
	case strings.Contains(fieldName, "longitude"):
		return gofakeit.Longitude()
	default:
		return gofakeit.Float64()
	}
}

// generateStringValue 生成字符串类型的值
func generateStringValue(fieldName string, index int) string {
	switch {
	case strings.Contains(fieldName, "name"):
		if strings.Contains(fieldName, "first") {
			return gofakeit.FirstName()
		}
		if strings.Contains(fieldName, "last") {
			return gofakeit.LastName()
		}
		return gofakeit.Name()
	case strings.Contains(fieldName, "email"):
		return gofakeit.Email()
	case strings.Contains(fieldName, "phone"):
		return gofakeit.Phone()
	case strings.Contains(fieldName, "address"):
		return gofakeit.Address().Address
	case strings.Contains(fieldName, "city"):
		return gofakeit.City()
	case strings.Contains(fieldName, "country"):
		return gofakeit.Country()
	case strings.Contains(fieldName, "company"):
		return gofakeit.Company()
	case strings.Contains(fieldName, "job"), strings.Contains(fieldName, "title"):
		return gofakeit.JobTitle()
	case strings.Contains(fieldName, "description"), strings.Contains(fieldName, "content"):
		return gofakeit.Sentence(gofakeit.Number(5, 20))
	case strings.Contains(fieldName, "url"), strings.Contains(fieldName, "link"):
		return gofakeit.URL()
	case strings.Contains(fieldName, "ip"):
		return gofakeit.IPv4Address()
	case strings.Contains(fieldName, "uuid"), strings.Contains(fieldName, "guid"):
		return gofakeit.UUID()
	case strings.Contains(fieldName, "color"):
		return gofakeit.Color()
	case strings.Contains(fieldName, "status"):
		statuses := []string{"active", "inactive", "pending", "completed", "cancelled"}
		return statuses[gofakeit.Number(0, len(statuses)-1)]
	case strings.Contains(fieldName, "category"), strings.Contains(fieldName, "type"):
		categories := []string{"electronics", "clothing", "food", "books", "sports", "home"}
		return categories[gofakeit.Number(0, len(categories)-1)]
	default:
		return gofakeit.Word()
	}
}

// generateFloatVector 生成 Float Vector
func generateFloatVector(field *entity.Field) []float32 {
	dim := getDimension(field)
	if dim == 0 {
		dim = 128
	}
	vector := make([]float32, dim)
	for i := range vector {
		vector[i] = gofakeit.Float32Range(-1, 1)
	}
	return vector
}

// generateBinaryVector 生成 Binary Vector
func generateBinaryVector(field *entity.Field) []byte {
	dim := getDimension(field)
	if dim == 0 {
		dim = 128
	}
	vector := make([]byte, dim/8)
	for i := range vector {
		vector[i] = byte(gofakeit.Number(0, 255))
	}
	return vector
}

// generateFloat16Vector 生成 Float16 Vector
func generateFloat16Vector(field *entity.Field) []byte {
	dim := getDimension(field)
	if dim == 0 {
		dim = 128
	}
	vector := make([]byte, dim*2)
	for i := range vector {
		vector[i] = byte(gofakeit.Number(0, 255))
	}
	return vector
}

// generateBFloat16Vector 生成 BFloat16 Vector
func generateBFloat16Vector(field *entity.Field) []byte {
	dim := getDimension(field)
	if dim == 0 {
		dim = 128
	}
	vector := make([]byte, dim*2)
	for i := range vector {
		vector[i] = byte(gofakeit.Number(0, 255))
	}
	return vector
}

// generateInt8Vector 生成 Int8 Vector
func generateInt8Vector(field *entity.Field) []int8 {
	dim := getDimension(field)
	if dim == 0 {
		dim = 128
	}
	vector := make([]int8, dim)
	for i := range vector {
		vector[i] = int8(gofakeit.Number(-128, 127))
	}
	return vector
}

// generateJSONValue 生成 JSON 类型的值
func generateJSONValue() map[string]interface{} {
	return map[string]interface{}{
		"key":   gofakeit.Word(),
		"value": gofakeit.Int32(),
		"tags":  []string{gofakeit.Word(), gofakeit.Word(), gofakeit.Word()},
	}
}

// generateArrayValue 生成 Array 类型的值
func generateArrayValue(field *entity.Field) interface{} {
	elementType := field.ElementType
	switch elementType {
	case entity.FieldTypeInt64:
		arr := make([]int64, gofakeit.Number(2, 5))
		for i := range arr {
			arr[i] = gofakeit.Int64()
		}
		return arr
	case entity.FieldTypeInt32:
		arr := make([]int32, gofakeit.Number(2, 5))
		for i := range arr {
			arr[i] = gofakeit.Int32()
		}
		return arr
	case entity.FieldTypeFloat:
		arr := make([]float32, gofakeit.Number(2, 5))
		for i := range arr {
			arr[i] = gofakeit.Float32()
		}
		return arr
	case entity.FieldTypeVarChar:
		arr := make([]string, gofakeit.Number(2, 5))
		for i := range arr {
			arr[i] = gofakeit.Word()
		}
		return arr
	default:
		return []string{gofakeit.Word(), gofakeit.Word()}
	}
}

// getDimension 从 TypeParams 中获取维度信息
func getDimension(field *entity.Field) int64 {
	if field.TypeParams != nil {
		if dimStr, ok := field.TypeParams["dim"]; ok {
			var dim int64
			fmt.Sscanf(dimStr, "%d", &dim)
			return dim
		}
	}
	return 0
}
