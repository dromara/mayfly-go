package httpclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

// 默认超时
const DefTimeout = 60

type RequestWrapper struct {
	client  http.Client
	url     string
	method  string
	timeout int
	body    io.Reader
	header  map[string]string
}

type MultipartFile struct {
	FieldName string // 字段名
	FileName  string // 文件名
	FilePath  string // 文件路径，文件路径不为空，则优先读取文件路径的内容
	Bytes     []byte // 文件内容
}

// 创建一个请求
func NewRequest(url string) *RequestWrapper {
	return &RequestWrapper{url: url, client: http.Client{}}
}

func (r *RequestWrapper) Url(url string) *RequestWrapper {
	r.url = url
	return r
}

func (r *RequestWrapper) Header(name, value string) *RequestWrapper {
	if r.header == nil {
		r.header = make(map[string]string)
	}
	r.header[name] = value
	return r
}

func (r *RequestWrapper) Timeout(timeout int) *RequestWrapper {
	r.timeout = timeout
	return r
}

func (r *RequestWrapper) GetByQuery(queryMap collx.M) *ResponseWrapper {
	var params string
	for k, v := range queryMap {
		if params != "" {
			params += "&"
		}
		params += k + "=" + anyx.ToString(v)
	}
	r.url += "?" + params
	return r.Get()
}

func (r *RequestWrapper) Get() *ResponseWrapper {
	r.method = "GET"
	r.body = nil
	return sendRequest(r)
}

func (r *RequestWrapper) PostJson(body string) *ResponseWrapper {
	buf := bytes.NewBufferString(body)
	r.method = "POST"
	r.body = buf
	if r.header == nil {
		r.header = make(map[string]string)
	}
	r.header["Content-type"] = "application/json"
	return sendRequest(r)
}

func (r *RequestWrapper) PostObj(body any) *ResponseWrapper {
	marshal, err := json.Marshal(body)
	if err != nil {
		return &ResponseWrapper{err: errors.New("解析json obj错误")}
	}
	return r.PostJson(string(marshal))
}

func (r *RequestWrapper) PostForm(params string) *ResponseWrapper {
	buf := bytes.NewBufferString(params)
	r.method = "POST"
	r.body = buf
	if r.header == nil {
		r.header = make(map[string]string)
	}
	r.header["Content-type"] = "application/x-www-form-urlencoded"
	return sendRequest(r)
}

func (r *RequestWrapper) PostMulipart(files []MultipartFile, reqParams collx.M) *ResponseWrapper {
	buf := &bytes.Buffer{}
	// 文件写入 buf
	writer := multipart.NewWriter(buf)
	for _, uploadFile := range files {
		var reader io.Reader
		// 如果文件路径不为空，则读取该路径文件，否则使用bytes
		if uploadFile.FilePath != "" {
			file, err := os.Open(uploadFile.FilePath)
			if err != nil {
				return &ResponseWrapper{err: err}
			}
			defer file.Close()
			reader = file
		} else {
			reader = bytes.NewBuffer(uploadFile.Bytes)
		}

		part, err := writer.CreateFormFile(uploadFile.FieldName, uploadFile.FileName)
		if err != nil {
			return &ResponseWrapper{err: err}
		}
		io.Copy(part, reader)
	}
	// 如果有其他参数，则写入body
	for k, v := range reqParams {
		if err := writer.WriteField(k, anyx.ConvString(v)); err != nil {
			return &ResponseWrapper{err: err}
		}
	}
	if err := writer.Close(); err != nil {
		return &ResponseWrapper{err: err}
	}

	r.method = "POST"
	r.body = buf
	if r.header == nil {
		r.header = make(map[string]string)
	}
	r.header["Content-type"] = writer.FormDataContentType()
	return sendRequest(r)
}

func sendRequest(rw *RequestWrapper) *ResponseWrapper {
	respWrapper := &ResponseWrapper{}
	timeout := rw.timeout
	if timeout > 0 {
		rw.client.Timeout = time.Duration(timeout) * time.Second
	} else {
		timeout = DefTimeout
	}

	req, err := http.NewRequest(rw.method, rw.url, rw.body)
	if err != nil {
		respWrapper.err = fmt.Errorf("创建请求错误-%s", err.Error())
		return respWrapper
	}
	setRequestHeader(req, rw.header)
	resp, err := rw.client.Do(req)
	return &ResponseWrapper{resp: resp, err: err}
}

func setRequestHeader(req *http.Request, header map[string]string) {
	req.Header.Set("User-Agent", "golang/mayfly")
	for k, v := range header {
		req.Header.Set(k, v)
	}
}

type ResponseWrapper struct {
	resp *http.Response
	err  error
}

// 将响应体通过json解析转为指定结构体
func (r *ResponseWrapper) BodyToObj(objPtr any) error {
	bodyBytes, err := r.BodyBytes()
	if err != nil {
		return err
	}

	err = json.Unmarshal(bodyBytes, &objPtr)
	if err != nil {
		return fmt.Errorf("解析响应体-json解析失败-%s", err.Error())
	}
	return nil
}

// 将响应体转为strings
func (r *ResponseWrapper) BodyToString() (string, error) {
	bodyBytes, err := r.BodyBytes()
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}

// 将响应体通过json解析转为map
func (r *ResponseWrapper) BodyToMap() (map[string]any, error) {
	var res map[string]any
	return res, r.BodyToObj(&res)
}

// 获取响应体的字节数组
func (r *ResponseWrapper) BodyBytes() ([]byte, error) {
	resp, err := r.GetHttpResp()
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		return nil, fmt.Errorf("读取响应体数据失败-%s", err.Error())
	}
	return body, err
}

// 获取http响应结果结构体
func (r *ResponseWrapper) GetHttpResp() (*http.Response, error) {
	if r.err != nil {
		return nil, fmt.Errorf("请求失败-%s", r.err.Error())
	}
	if r.resp == nil {
		return nil, errors.New("请求失败-响应结构体为空,请检查请求url等信息")
	}

	statusCode := r.resp.StatusCode
	if isFailureStatusCode(statusCode) {
		logx.Warnf("请求响应状态码为为失败状态: %v", statusCode)
	}

	return r.resp, nil
}

func isFailureStatusCode(statusCode int) bool {
	return statusCode < http.StatusOK || statusCode >= http.StatusBadRequest
}
