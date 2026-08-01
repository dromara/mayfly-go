package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// --- 机器相关 API ---

// ListMachines 获取机器列表
func (c *ApiClient) ListMachines() (*PageResult, error) {
	var result PageResult
	if err := c.Get("/machines", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetMachineStats 获取机器状态信息
func (c *ApiClient) GetMachineStats(machineId uint64) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.Get(fmt.Sprintf("/machines/%d/stats", machineId), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMachineProcesses 获取进程列表
func (c *ApiClient) GetMachineProcesses(machineId uint64) (interface{}, error) {
	var result interface{}
	if err := c.Get(fmt.Sprintf("/machines/%d/process", machineId), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// RunMachineCmd 执行远程命令
func (c *ApiClient) RunMachineCmd(machineId uint64, authCert, cmd string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"cmd": cmd,
	}
	var result map[string]interface{}
	if err := c.Post(fmt.Sprintf("/machines/%d/%s/run-cmd", machineId, authCert), body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMachineUsers 获取用户列表
func (c *ApiClient) GetMachineUsers(machineId uint64) ([]interface{}, error) {
	var result []interface{}
	if err := c.Get(fmt.Sprintf("/machines/%d/users", machineId), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// KillProcess 终止指定进程
func (c *ApiClient) KillProcess(machineId uint64, pid int) error {
	path := fmt.Sprintf("/machines/%d/process?pid=%d", machineId, pid)
	return c.request("DELETE", path, nil, nil)
}

// --- 机器文件操作 API ---

// fileOpPath 构建文件操作的 URL 路径
func fileOpPath(machineId uint64, action string) string {
	return fmt.Sprintf("/machines/%d/files/0/%s", machineId, action)
}

// fileOpQuery 构建文件操作的查询参数
func fileOpQuery(machineId uint64, authCert, filePath string) string {
	return fmt.Sprintf("machineId=%d&protocol=1&authCertName=%s&path=%s",
		machineId, url.QueryEscape(authCert), url.QueryEscape(filePath))
}

// ListDir 列出远程目录内容
func (c *ApiClient) ListDir(machineId uint64, authCert, dirPath string) (interface{}, error) {
	path := fmt.Sprintf("%s?%s", fileOpPath(machineId, "read-dir"), fileOpQuery(machineId, authCert, dirPath))
	var result interface{}
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ReadRemoteFile 读取远程文件内容（文件需小于 1MB）
func (c *ApiClient) ReadRemoteFile(machineId uint64, authCert, filePath string) (string, error) {
	path := fmt.Sprintf("%s?%s", fileOpPath(machineId, "read"), fileOpQuery(machineId, authCert, filePath))
	var result string
	if err := c.Get(path, &result); err != nil {
		return "", err
	}
	return result, nil
}

// DownloadFile 下载远程文件（返回原始字节）
func (c *ApiClient) DownloadFile(machineId uint64, authCert, filePath string) ([]byte, error) {
	path := fmt.Sprintf("%s?%s", fileOpPath(machineId, "download"), fileOpQuery(machineId, authCert, filePath))
	return c.getRaw(path)
}

// UploadFile 上传文件到远程机器
func (c *ApiClient) UploadFile(machineId uint64, authCert, dirPath, filename string, content []byte) error {
	reqURL := fmt.Sprintf("%s/api%s?%s&filename=%s", c.baseURL,
		fileOpPath(machineId, "upload"),
		fileOpQuery(machineId, authCert, dirPath),
		url.QueryEscape(filename))

	req, err := http.NewRequest("POST", reqURL, bytes.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var envelope struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &envelope); err == nil && envelope.Code != 200 {
		return &BizError{Code: envelope.Code, Msg: envelope.Msg}
	}
	return nil
}

// WriteRemoteFile 写入远程文件内容
func (c *ApiClient) WriteRemoteFile(machineId uint64, authCert, filePath, content string) error {
	body := map[string]interface{}{
		"machineId":    machineId,
		"protocol":     1,
		"authCertName": authCert,
		"path":         filePath,
		"content":      content,
	}
	return c.Post(fileOpPath(machineId, "write"), body, nil)
}

// RemoveRemoteFile 删除远程文件/目录
func (c *ApiClient) RemoveRemoteFile(machineId uint64, authCert string, paths []string) error {
	body := map[string]interface{}{
		"machineId":    machineId,
		"protocol":     1,
		"authCertName": authCert,
		"paths":        paths,
	}
	return c.Post(fileOpPath(machineId, "remove"), body, nil)
}

// CopyRemoteFile 复制远程文件
func (c *ApiClient) CopyRemoteFile(machineId uint64, authCert, src, dst string) error {
	body := map[string]interface{}{
		"machineId":    machineId,
		"protocol":     1,
		"authCertName": authCert,
		"paths":        []string{src},
		"toPath":       dst,
	}
	return c.Post(fileOpPath(machineId, "cp"), body, nil)
}

// MoveRemoteFile 移动远程文件
func (c *ApiClient) MoveRemoteFile(machineId uint64, authCert, src, dst string) error {
	body := map[string]interface{}{
		"machineId":    machineId,
		"protocol":     1,
		"authCertName": authCert,
		"paths":        []string{src},
		"toPath":       dst,
	}
	return c.Post(fileOpPath(machineId, "mv"), body, nil)
}

// RenameRemoteFile 重命名远程文件
func (c *ApiClient) RenameRemoteFile(machineId uint64, authCert, path, newName string) error {
	body := map[string]interface{}{
		"machineId":    machineId,
		"protocol":     1,
		"authCertName": authCert,
		"path":         path,
		"newname":      newName,
	}
	return c.Post(fileOpPath(machineId, "rename"), body, nil)
}

// CreateRemoteFile 创建远程文件或目录
func (c *ApiClient) CreateRemoteFile(machineId uint64, authCert, path string, isDir bool) error {
	fileType := "file"
	if isDir {
		fileType = "dir"
	}
	body := map[string]interface{}{
		"machineId":    machineId,
		"protocol":     1,
		"authCertName": authCert,
		"path":         path,
		"type":         fileType,
	}
	return c.Post(fileOpPath(machineId, "create-file"), body, nil)
}

// GetFileStat 获取远程文件状态信息
func (c *ApiClient) GetFileStat(machineId uint64, authCert, filePath string) (interface{}, error) {
	path := fmt.Sprintf("%s?%s", fileOpPath(machineId, "file-stat"), fileOpQuery(machineId, authCert, filePath))
	var result interface{}
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetDirSize 获取远程目录大小
func (c *ApiClient) GetDirSize(machineId uint64, authCert, dirPath string) (interface{}, error) {
	path := fmt.Sprintf("%s?%s", fileOpPath(machineId, "dir-size"), fileOpQuery(machineId, authCert, dirPath))
	var result interface{}
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}
