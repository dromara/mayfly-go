package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mayfly-go/internal/file/config"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// 记录一次请求的关键信息，用于断言签名/校验和头对s3兼容存储的兼容性
type reqRecord struct {
	method       string
	query        string
	payloadHash  string
	bodySHA256   string
	bodyLen      int64
	checksumHead []string
}

// newCompatTestServer 启动一个最小s3协议假服务，支持单请求上传与分片上传，并记录每个请求的头信息
func newCompatTestServer(t *testing.T, records *[]reqRecord) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	uploadID := "mock-upload-id"

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body error: %v", err)
		}

		sum := sha256.Sum256(body)
		rec := reqRecord{
			method:      r.Method,
			query:       r.URL.RawQuery,
			payloadHash: r.Header.Get("x-amz-content-sha256"),
			bodySHA256:  hex.EncodeToString(sum[:]),
			bodyLen:     int64(len(body)),
		}
		// 收集所有flexible checksum相关头
		for k := range r.Header {
			kl := strings.ToLower(k)
			if strings.HasPrefix(kl, "x-amz-checksum") || kl == "x-amz-sdk-checksum-algorithm" || kl == "x-amz-decoded-content-length" {
				rec.checksumHead = append(rec.checksumHead, kl+"="+r.Header.Get(k))
			}
		}
		mu.Lock()
		*records = append(*records, rec)
		mu.Unlock()

		q := r.URL.Query()
		w.Header().Set("Content-Type", "application/xml")
		switch {
		case r.Method == http.MethodPost && q.Has("uploads"):
			fmt.Fprint(w, `<InitiateMultipartUploadResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Bucket>bkt</Bucket><Key>k</Key><UploadId>`+uploadID+`</UploadId></InitiateMultipartUploadResult>`)
		case r.Method == http.MethodPut && q.Has("uploadId"):
			w.Header().Set("ETag", `"etag-part"`)
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && q.Has("uploadId"):
			fmt.Fprint(w, `<CompleteMultipartUploadResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Bucket>bkt</Bucket><Key>k</Key><ETag>"etag-obj"</ETag></CompleteMultipartUploadResult>`)
		case r.Method == http.MethodPut:
			w.Header().Set("ETag", `"etag-single"`)
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusOK)
		}
	}))
}

// TestS3CompatPayloadHeaders 校验上传请求对第三方s3兼容存储（华为OBS等）的兼容性：
//  1. 不允许出现flexible checksum头（x-amz-checksum-*/x-amz-sdk-checksum-algorithm），否则部分实现会校验失败；
//  2. x-amz-content-sha256必须是载荷的真实sha256，不能是STREAMING-*流式签名标记；
//  3. 声明的hash必须与服务端实收载荷的hash一致（覆盖单请求PutObject与分片UploadPart）。
func TestS3CompatPayloadHeaders(t *testing.T) {
	var records []reqRecord
	srv := newCompatTestServer(t, &records)
	defer srv.Close()

	st, err := newS3Storage(&config.S3Config{
		Endpoint: srv.URL, Region: "us-east-1", Bucket: "bkt",
		AccessKey: "ak", SecretKey: "sk", UsePathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 小文件：单请求PutObject；大文件：超过默认5MB分片，走multipart UploadPart
	for _, tc := range []struct {
		name  string
		write func(io.Writer) error
	}{
		{"small", func(w io.Writer) error { _, err := io.WriteString(w, strings.Repeat("a", 1024)); return err }},
		{"big", func(w io.Writer) error {
			chunk := strings.Repeat("b", 1024*1024)
			for i := 0; i < 7; i++ {
				if _, err := io.WriteString(w, chunk); err != nil {
					return err
				}
			}
			return nil
		}},
	} {
		w, err := st.OpenWriter(context.Background(), tc.name+".txt")
		if err != nil {
			t.Fatalf("[%s] OpenWriter error: %v", tc.name, err)
		}
		if err := tc.write(w); err != nil {
			t.Fatalf("[%s] write error: %v", tc.name, err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("[%s] close(upload) error: %v", tc.name, err)
		}
	}

	if err := st.Remove(context.Background(), "small.txt"); err != nil {
		t.Fatalf("Remove error: %v", err)
	}

	hexRe := regexp.MustCompile(`^[0-9a-f]{64}$`)
	partCount := 0
	for _, rec := range records {
		desc := fmt.Sprintf("%s ?%s", rec.method, rec.query)
		// 空载荷请求（Initiate/Complete/Delete）hash为空串的sha256，同样要求是hex而非STREAMING-*
		if len(rec.checksumHead) > 0 {
			t.Errorf("[%s] 出现flexible checksum头，s3兼容存储可能拒绝: %v", desc, rec.checksumHead)
		}
		if !hexRe.MatchString(rec.payloadHash) {
			t.Errorf("[%s] x-amz-content-sha256非真实hash（流式签名标记）: %s", desc, rec.payloadHash)
		}
		if rec.payloadHash != rec.bodySHA256 {
			t.Errorf("[%s] 声明hash与实收载荷不一致: declared=%s actual=%s bytes=%d", desc, rec.payloadHash, rec.bodySHA256, rec.bodyLen)
		}
		if strings.Contains(rec.query, "partNumber=") {
			partCount++
		}
	}
	if partCount < 2 {
		t.Fatalf("大文件未按预期分片上传，partCount=%d，records=%d", partCount, len(records))
	}
	t.Logf("共%d个请求，其中UploadPart %d个，全部为真实sha256签名且无flexible checksum头", len(records), partCount)
}
