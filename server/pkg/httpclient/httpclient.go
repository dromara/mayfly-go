package httpclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

var client = &http.Client{}

// 默认超时
const DefTimeout = 60

type RequestWrapper struct {
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
	return &RequestWrapper{url: url}
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

func (r *RequestWrapper) GetByParam(paramMap map[string]string) *ResponseWrapper {
	var params string
	for k, v := range paramMap {
		if params != "" {
			params += "&"
		} else {
			params += "?"
		}
		params += k + "=" + v
	}
	r.url += "?" + params
	return r.Get()
}

func (r *RequestWrapper) Get() *ResponseWrapper {
	r.method = "GET"
	r.body = nil
	return request(r)
}

func (r *RequestWrapper) PostJson(body string) *ResponseWrapper {
	buf := bytes.NewBufferString(body)
	r.method = "POST"
	r.body = buf
	if r.header == nil {
		r.header = make(map[string]string)
	}
	r.header["Content-type"] = "application/json"
	return request(r)
}

func (r *RequestWrapper) PostObj(body interface{}) *ResponseWrapper {
	marshal, err := json.Marshal(body)
	if err != nil {
		return createRequestError(errors.New("解析json obj错误"))
	}
	return r.PostJson(string(marshal))
}

func (r *RequestWrapper) PostParams(params string) *ResponseWrapper {
	buf := bytes.NewBufferString(params)
	r.method = "POST"
	r.body = buf
	if r.header == nil {
		r.header = make(map[string]string)
	}
	r.header["Content-type"] = "application/x-www-form-urlencoded"
	return request(r)
}

func (r *RequestWrapper) PostMulipart(files []MultipartFile, reqParams map[string]string) *ResponseWrapper {
	buf := &bytes.Buffer{}
	// 文件写入 buf
	writer := multipart.NewWriter(buf)
	for _, uploadFile := range files {
		var reader io.Reader
		// 如果文件路径不为空，则读取该路径文件，否则使用bytes
		if uploadFile.FilePath != "" {
			file, err := os.Open(uploadFile.FilePath)
			if err != nil {
				return createRequestError(err)
			}
			defer file.Close()
			reader = file
		} else {
			reader = bytes.NewBuffer(uploadFile.Bytes)
		}

		part, err := writer.CreateFormFile(uploadFile.FieldName, uploadFile.FileName)
		if err != nil {
			return createRequestError(err)
		}
		_, err = io.Copy(part, reader)
	}
	// 如果有其他参数，则写入body
	for k, v := range reqParams {
		if err := writer.WriteField(k, v); err != nil {
			return createRequestError(err)
		}
	}
	if err := writer.Close(); err != nil {
		return createRequestError(err)
	}

	r.method = "POST"
	r.body = buf
	if r.header == nil {
		r.header = make(map[string]string)
	}
	r.header["Content-type"] = writer.FormDataContentType()
	return request(r)
}

type ResponseWrapper struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

func (r *ResponseWrapper) IsSuccess() bool {
	return r.StatusCode == 200
}

func (r *ResponseWrapper) BodyToObj(objPtr interface{}) error {
	_ = json.Unmarshal(r.Body, &objPtr)
	return r.getError()
}

func (r *ResponseWrapper) BodyToString() (string, error) {
	return string(r.Body), r.getError()
}

func (r *ResponseWrapper) BodyToMap() (map[string]interface{}, error) {
	var res map[string]interface{}
	err := json.Unmarshal(r.Body, &res)
	if err != nil {
		return nil, err
	}
	return res, r.getError()
}

func (r *ResponseWrapper) getError() error {
	if !r.IsSuccess() {
		return errors.New(string(r.Body))
	}
	return nil
}

func request(rw *RequestWrapper) *ResponseWrapper {
	wrapper := &ResponseWrapper{StatusCode: 0, Header: make(http.Header)}
	timeout := rw.timeout
	if timeout > 0 {
		client.Timeout = time.Duration(timeout) * time.Second
	} else {
		timeout = DefTimeout
	}

	req, err := http.NewRequest(rw.method, rw.url, rw.body)
	if err != nil {
		return createRequestError(err)
	}
	setRequestHeader(req, rw.header)
	resp, err := client.Do(req)
	if err != nil {
		wrapper.Body = []byte(fmt.Sprintf("执行HTTP请求错误-%s", err.Error()))
		return wrapper
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		wrapper.Body = []byte(fmt.Sprintf("读取HTTP请求返回值失败-%s", err.Error()))
		return wrapper
	}
	wrapper.StatusCode = resp.StatusCode
	wrapper.Body = body
	wrapper.Header = resp.Header

	return wrapper
}

func setRequestHeader(req *http.Request, header map[string]string) {
	req.Header.Set("User-Agent", "golang/mayfly")
	for k, v := range header {
		req.Header.Set(k, v)
	}
}

func createRequestError(err error) *ResponseWrapper {
	return &ResponseWrapper{0, []byte(fmt.Sprintf("创建HTTP请求错误-%s", err.Error())), make(http.Header)}
}
