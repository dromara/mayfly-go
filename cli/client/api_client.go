package client

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mayfly-go/cli/i18n"
	"net/http"
	"net/url"
	"os"
	"time"
)

// ErrTokenExpired 表示 token 已过期且刷新失败，需要重新登录
var ErrTokenExpired = errors.New("token expired, please re-login")

// ConnectionError 表示连接/认证类错误（结构化，便于上层用 errors.As 判断）
type ConnectionError struct {
	Code int
	Msg  string
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("%s (code=%d)", e.Msg, e.Code)
}

// TokenRefreshedFunc token 刷新成功后的回调，用于持久化新 token
// 参数为新的 accessToken 和 refreshToken
type TokenRefreshedFunc func(accessToken, refreshToken string)

type ApiClient struct {
	baseURL          string
	token            string
	refreshToken     string
	client           *http.Client
	rawClient        *http.Client // 禁用压缩的客户端，用于二进制流下载
	onTokenRefreshed TokenRefreshedFunc
	ctx              context.Context // 请求上下文，支持 Ctrl+C 取消
	verbose          bool            // 详细输出模式
}

func NewApiClient(baseURL, token string) *ApiClient {
	return &ApiClient{
		baseURL: baseURL,
		token:   token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		rawClient: &http.Client{
			Transport: &http.Transport{DisableCompression: true},
			Timeout:   30 * time.Second,
		},
		ctx: context.Background(),
	}
}

// WithContext 设置请求上下文（支持 Ctrl+C 取消）
func (c *ApiClient) WithContext(ctx context.Context) *ApiClient {
	c.ctx = ctx
	return c
}

// WithVerbose 设置详细输出模式
func (c *ApiClient) WithVerbose(verbose bool) *ApiClient {
	c.verbose = verbose
	return c
}

// WithTimeout 设置请求超时时间（秒）
func (c *ApiClient) WithTimeout(seconds int) *ApiClient {
	if seconds > 0 {
		timeout := time.Duration(seconds) * time.Second
		c.client.Timeout = timeout
		c.rawClient.Timeout = timeout
	}
	return c
}

// SetRefreshToken 设置 refreshToken 及刷新成功回调
func (c *ApiClient) SetRefreshToken(refreshToken string, onRefreshed TokenRefreshedFunc) {
	c.refreshToken = refreshToken
	c.onTokenRefreshed = onRefreshed
}

// 通用请求方法（内置 token 过期自动刷新重试）
func (c *ApiClient) request(method, path string, body interface{}, result interface{}) error {
	err := c.doRequest(method, path, body, result)
	if err == nil {
		return nil
	}

	// 检测业务码 502（token 无效/过期），尝试刷新后重试一次
	var bizErr *BizError
	if errors.As(err, &bizErr) && bizErr.Code == CodeTokenInvalid {
		if refreshErr := c.refreshAccessToken(); refreshErr != nil {
			return fmt.Errorf("%w: %v", ErrTokenExpired, refreshErr)
		}
		// 刷新成功，用新 token 重试原请求
		return c.doRequest(method, path, body, result)
	}

	return err
}

// doRequest 执行单次 HTTP 请求
func (c *ApiClient) doRequest(method, path string, body interface{}, result interface{}) error {
	reqURL := fmt.Sprintf("%s/api%s", c.baseURL, path)

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("%s: %w", i18n.T("client.serialize_failed"), err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(c.ctx, method, reqURL, reqBody)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("client.create_request_failed"), err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	if c.verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] %s %s\n", method, reqURL)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("client.send_request_failed"), err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("client.read_response_failed"), err)
	}

	if c.verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] <- HTTP %d (%d bytes)\n", resp.StatusCode, len(respBody))
	}

	// 检查 HTTP 状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s (HTTP %d): %s", i18n.T("client.request_failed"), resp.StatusCode, string(respBody))
	}

	// 解析业务响应，检查 code 字段
	var envelope struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		// 不是标准信封格式，直接解析到 result
		if result != nil {
			return json.Unmarshal(respBody, result)
		}
		return nil
	}

	// 检查业务状态码
	if envelope.Code != 200 {
		bizErr := &BizError{Code: envelope.Code, Msg: envelope.Msg}
		// 认证类错误包装为 ConnectionError
		if envelope.Code == CodeTokenInvalid || envelope.Code == CodeTokenMissing {
			return &ConnectionError{Code: envelope.Code, Msg: envelope.Msg}
		}
		return bizErr
	}

	// 提取 data 字段到 result
	if result != nil && envelope.Data != nil {
		if err := json.Unmarshal(envelope.Data, result); err != nil {
			return fmt.Errorf("%s: %w", i18n.T("client.parse_response_failed"), err)
		}
	}

	return nil
}

// 服务端业务错误码
const (
	CodeTokenInvalid = 502 // access token invalid
	CodeTokenMissing = 501 // token error (missing)
)

// BizError 服务端业务错误（携带业务码，便于区分 token 过期等场景）
type BizError struct {
	Code int
	Msg  string
}

func (e *BizError) Error() string {
	return fmt.Sprintf("%s (code=%d)", e.Msg, e.Code)
}

// refreshAccessToken 使用 refreshToken 获取新的 token 对
func (c *ApiClient) refreshAccessToken() error {
	if c.refreshToken == "" {
		return errors.New("no refresh token available")
	}

	reqURL := fmt.Sprintf("%s/api/auth/accounts/refreshToken?refresh_token=%s",
		c.baseURL, url.QueryEscape(c.refreshToken))

	resp, err := c.client.Get(reqURL)
	if err != nil {
		return fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read refresh response failed: %w", err)
	}

	var envelope struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("parse refresh response failed: %w", err)
	}
	if envelope.Code != 200 {
		return fmt.Errorf("refresh token rejected: %s", envelope.Msg)
	}

	var tokens struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(envelope.Data, &tokens); err != nil {
		return fmt.Errorf("parse refresh tokens failed: %w", err)
	}
	if tokens.Token == "" {
		return errors.New("empty token in refresh response")
	}

	// 更新内存中的 token
	c.token = tokens.Token
	c.refreshToken = tokens.RefreshToken

	// 回调持久化（保存配置文件）
	if c.onTokenRefreshed != nil {
		c.onTokenRefreshed(tokens.Token, tokens.RefreshToken)
	}

	return nil
}

// Get 发送 GET 请求
func (c *ApiClient) Get(path string, result interface{}) error {
	return c.request("GET", path, nil, result)
}

// Post 发送 POST 请求
func (c *ApiClient) Post(path string, body interface{}, result interface{}) error {
	return c.request("POST", path, body, result)
}

// getRaw 发送 GET 请求并返回原始响应字节（用于文件下载/SQL 导出等二进制流）
// 自动检测并处理 gzip 压缩的响应体
func (c *ApiClient) getRaw(path string) ([]byte, error) {
	reqURL := fmt.Sprintf("%s/api%s", c.baseURL, path)

	req, err := http.NewRequestWithContext(c.ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.rawClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}

	// 检测 gzip 魔数 (0x1f 0x8b)，若为 gzip 则解压
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		gzReader, gzErr := gzip.NewReader(bytes.NewReader(data))
		if gzErr != nil {
			return nil, fmt.Errorf("gzip decompress: %w", gzErr)
		}
		defer gzReader.Close()
		decompressed, readErr := io.ReadAll(gzReader)
		// 服务端流式写入可能未正确关闭 gzip 尾，已读到内容即可
		if readErr != nil && len(decompressed) == 0 {
			return nil, fmt.Errorf("gzip read: %w", readErr)
		}
		data = decompressed
	}

	return data, nil
}

// PageResult 分页结果通用结构
type PageResult struct {
	List     []map[string]interface{} `json:"list"`
	Total    int64                    `json:"total"`
	PageNum  int                      `json:"pageNum"`
	PageSize int                      `json:"pageSize"`
}
