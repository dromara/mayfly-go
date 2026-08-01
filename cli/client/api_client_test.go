package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewApiClient(t *testing.T) {
	client := NewApiClient("http://localhost:8888", "test-token")
	if client == nil {
		t.Fatal("NewApiClient() returned nil")
	}
	if client.baseURL != "http://localhost:8888" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "http://localhost:8888")
	}
	if client.token != "test-token" {
		t.Errorf("token = %q, want %q", client.token, "test-token")
	}
}

func TestApiClient_WithContext(t *testing.T) {
	client := NewApiClient("http://localhost:8888", "")
	ctx := context.Background()
	returned := client.WithContext(ctx)
	if returned != client {
		t.Error("WithContext() should return the same client for chaining")
	}
}

func TestApiClient_WithVerbose(t *testing.T) {
	client := NewApiClient("http://localhost:8888", "")
	returned := client.WithVerbose(true)
	if returned != client {
		t.Error("WithVerbose() should return the same client for chaining")
	}
	if !client.verbose {
		t.Error("verbose should be true after WithVerbose(true)")
	}
}

func TestConnectionError(t *testing.T) {
	err := &ConnectionError{Code: 501, Msg: "token invalid"}
	expected := "token invalid (code=501)"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestBizError(t *testing.T) {
	err := &BizError{Code: 400, Msg: "bad request"}
	expected := "bad request (code=400)"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestApiClient_Get_Success(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/test" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 200,
			"msg":  "success",
			"data": map[string]interface{}{"id": 1, "name": "test"},
		})
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "test-token")
	var result map[string]interface{}
	err := client.Get("/test", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if result["name"] != "test" {
		t.Errorf("result[name] = %v, want %q", result["name"], "test")
	}
}

func TestApiClient_Get_BizError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 400,
			"msg":  "bad request",
		})
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "")
	var result map[string]interface{}
	err := client.Get("/test", &result)
	if err == nil {
		t.Fatal("Get() should return error for biz error")
	}
	var bizErr *BizError
	if !errors.As(err, &bizErr) {
		t.Errorf("error should be BizError, got %T", err)
	}
}

func TestApiClient_Get_ConnectionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 501,
			"msg":  "token invalid",
		})
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "invalid-token")
	var result map[string]interface{}
	err := client.Get("/test", &result)
	if err == nil {
		t.Fatal("Get() should return error for connection error")
	}
	var connErr *ConnectionError
	if !errors.As(err, &connErr) {
		t.Errorf("error should be ConnectionError, got %T", err)
	}
}
