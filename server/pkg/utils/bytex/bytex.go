package bytex

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	KB = 1024
	MB = KB * 1024
	GB = MB * 1024
)

// 解析字符串byte size
//
// 1kb -> 1024
// 1mb -> 1024 * 1024
func ParseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	unit := sizeStr[len(sizeStr)-2:]

	valueStr := sizeStr[:len(sizeStr)-2]
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return 0, err
	}

	var bytes int64

	switch strings.ToUpper(unit) {
	case "KB":
		bytes = value * KB
	case "MB":
		bytes = value * MB
	case "GB":
		bytes = value * GB
	default:
		return 0, fmt.Errorf("invalid size unit")
	}

	return bytes, nil
}
