package cryptox

import (
	"encoding/base64"
	"testing"
)

func TestAesEncrypt(t *testing.T) {
	key := []byte("eyJhbGciOiJIUzI1NiIsInR5")
	data := []byte("SELECT * FROM \"instruct\" OFFSET 0 LIMIT 25;")
	encrypt, err := AesEncrypt(data, key)
	if err != nil {
		t.Error(err)
	}
	toString := base64.StdEncoding.EncodeToString(encrypt)
	t.Log(toString)
	decrypt, err := AesDecrypt(encrypt, key)

	t.Log(string(decrypt))
}

func TestDes(t *testing.T) {
	key := []byte("eyJhbGciOiJIUzI1NiIsInR5")
	data := []byte("SELECT * FROM \"instruct\" OFFSET 0 LIMIT 25;")
	encrypt, err := DesEncrypt(data, key)
	if err != nil {
		t.Error(err)
	}
	t.Log("encrypt", encrypt)
	decrypt, err := DesDecrypt(encrypt, key)
	if err != nil {
		t.Error(err)
	}
	t.Log("decrypt", decrypt)
}
