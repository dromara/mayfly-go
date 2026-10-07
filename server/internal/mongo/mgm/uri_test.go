package mgm

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestMaskUri(t *testing.T) {
	cases := []struct {
		name string
		uri  string
		want string
	}{
		{"单主机带账号密码", "mongodb://admin:p@ssw0rd@127.0.0.1:27017", "mongodb://admin:****@127.0.0.1:27017"},
		{"srv 形式（旧实现的正则漏掩）", "mongodb+srv://admin:secret@cluster0.example.net/", "mongodb+srv://admin:****@cluster0.example.net/"},
		{"多主机与库名", "mongodb://u:pwd@h1:27017,h2:27017/mydb?replicaSet=rs0", "mongodb://u:****@h1:27017,h2:27017/mydb?replicaSet=rs0"},
		{"查询参数含 @ 时仍须掩码", "mongodb://u:pwd@h1:27017/db?x=a@b", "mongodb://u:****@h1:27017/db?x=a@b"},
		{"路径含 @ 时不得误切", "mongodb://u:pwd@h1:27017/a@b", "mongodb://u:****@h1:27017/a@b"},
		{"密码含冒号取最后一个冒号", "mongodb://u:a:b@h1:27017", "mongodb://u:a:****@h1:27017"},
		{"无密码只用户名", "mongodb://readonly@h1:27017", "mongodb://readonly@h1:27017"},
		{"无凭证", "mongodb://127.0.0.1:27017/test", "mongodb://127.0.0.1:27017/test"},
		{"非连接串原样返回", "not-a-uri", "not-a-uri"},
		{"空串", "", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MaskUri(c.uri); got != c.want {
				t.Fatalf("MaskUri(%q) = %q, want %q", c.uri, got, c.want)
			}
		})
	}
}

// TestMaskUriNeverLeaksPassword 脱敏结果里不得残留原密码子串，覆盖上面逐条断言之外的组合。
func TestMaskUriNeverLeaksPassword(t *testing.T) {
	const password = "S3cr3t!x"
	uris := []string{
		"mongodb://admin:" + password + "@127.0.0.1:27017",
		"mongodb://admin:" + password + "@127.0.0.1:27017,127.0.0.1:27018/app?authSource=admin",
		"mongodb+srv://admin:" + password + "@cluster0.mongodb.net/app",
	}
	for _, uri := range uris {
		masked := MaskUri(uri)
		if strings.Contains(masked, password) {
			t.Fatalf("password leaked after masking: %q", masked)
		}
	}
}

// TestHasCredentials 凭证判定必须与掩码用同一份 userinfo 定位规则。
//
// 这个判定决定「测试连接」要不要补做一次权限探测：把匿名串误判成有凭证，就会放过一个
// 点开即报 Unauthorized 的实例；把有凭证串误判成匿名，则会对权限受限的账号误报连接失败。
func TestHasCredentials(t *testing.T) {
	cases := []struct {
		uri  string
		want bool
	}{
		{"mongodb://localhost:27017", false},
		{"mongodb://127.0.0.1:27017,test:27017/?replicaSet=rs0", false},
		{"mongodb://mayfly:pwd@127.0.0.1:27017/?authSource=admin", true},
		{"mongodb://mayfly@127.0.0.1:27017", true},
		{"mongodb+srv://u:p@cluster0.example.net/", true},
		// 查询参数里有 @：凭证判定只能看 authority 段
		{"mongodb://u:p@h1:27017/db?x=a@b", true},
		{"mongodb://h1:27017/db?mail=a@b", false},
		{"not-a-uri", false},
	}
	for _, c := range cases {
		if got := HasCredentials(c.uri); got != c.want {
			t.Fatalf("HasCredentials(%q) = %v, want %v", c.uri, got, c.want)
		}
	}
}

// TestIsUnauthorized 只认服务端错误码 13，不能靠文案匹配。
//
// 文案随语言/版本变化，而这条判据决定用户看到的是「补凭证」提示还是通用执行失败。
func TestIsUnauthorized(t *testing.T) {
	if !IsUnauthorized(mongo.CommandError{Code: CodeUnauthorized, Message: "Command listDatabases requires authentication"}) {
		t.Fatal("code 13 must be treated as unauthorized")
	}
	if IsUnauthorized(mongo.CommandError{Code: 26, Message: "namespace not found"}) {
		t.Fatal("other command errors must not be reported as unauthorized")
	}
	if IsUnauthorized(fmt.Errorf("wrapped: %w", mongo.CommandError{Code: CodeUnauthorized})) == false {
		t.Fatal("wrapped unauthorized must still be recognized")
	}
	if IsUnauthorized(nil) || IsUnauthorized(errors.New("boom")) {
		t.Fatal("nil and plain errors are not unauthorized")
	}
}
