package dbi

// DbVersion 数据库兼容版本标识（如 "oracle11"、"mysql5.7"）。
type DbVersion string

// parseVersionParts 从版本字符串中提取数值段
func parseVersionParts(version string) []int {
	// 跳过非数字前缀
	start := 0
	for start < len(version) && (version[start] < '0' || version[start] > '9') {
		start++
	}
	if start >= len(version) {
		return nil
	}
	version = version[start:]

	var parts []int
	current := 0
	hasDigit := false
	for i := 0; i < len(version); i++ {
		c := version[i]
		if c >= '0' && c <= '9' {
			current = current*10 + int(c-'0')
			hasDigit = true
		} else if c == '.' && hasDigit {
			parts = append(parts, current)
			current = 0
			hasDigit = false
		} else if hasDigit {
			parts = append(parts, current)
			current = 0
			hasDigit = false
		}
	}
	if hasDigit {
		parts = append(parts, current)
	}
	return parts
}

// ParseDbVersion 从原始版本字符串解析主/次版本号，填充到 DbServer。
// 支持常见格式："8.0.32"、"16.1 (Debian 16.1-1.pgdg120+1)"、"22.0.0.0.0" 等。
// 解析失败时 MajorVersion/MinorVersion 保持零值，不影响 Version 原始字段。
func ParseDbVersion(server *DbServer) {
	if server == nil || server.Version == "" {
		return
	}
	parts := parseVersionParts(server.Version)
	if len(parts) > 0 {
		server.MajorVersion = parts[0]
	}
	if len(parts) > 1 {
		server.MinorVersion = parts[1]
	}
}
