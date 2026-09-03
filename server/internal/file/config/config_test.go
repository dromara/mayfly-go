package config

import (
	"mayfly-go/pkg/utils/collx"
	"testing"
)

func TestGetS3Config(t *testing.T) {
	cases := []struct {
		name   string
		m      collx.M
		expect bool // 是否启用s3
	}{
		{"未配置", collx.M{}, false},
		{"仅配置部分项", collx.M{"s3Endpoint": "http://127.0.0.1:9000", "s3Bucket": "test"}, false},
		{"完整配置", collx.M{
			"s3Endpoint":  "http://127.0.0.1:9000",
			"s3Bucket":    "test",
			"s3AccessKey": "ak",
			"s3SecretKey": "sk",
		}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s3 := getS3Config(c.m)
			if (s3 != nil) != c.expect {
				t.Fatalf("expect enabled=%v, got %+v", c.expect, s3)
			}
			if s3 != nil && s3.Region != "us-east-1" {
				t.Fatalf("default region expect us-east-1, got %s", s3.Region)
			}
		})
	}
}
