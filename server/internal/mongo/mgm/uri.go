package mgm

import "strings"

// maskedPassword 连接串中密码的占位替代。
const maskedPassword = "****"

// MaskUri 掩去 Mongo 连接串里的密码。
//
// 覆盖 `mongodb://` 与 `mongodb+srv://` 两种 scheme：旧实现用一条只匹配 `^mongodb://` 的正则，
// `+srv` 场景下掩码空跑，日志里落下的就是明文凭证。
//
// 定位规则：先截出 authority（第一个 / ? # 之前的部分），密码是其中最后一个 `:` 与 `@` 之间的部分。
// 必须先在 authority 边界内找 `@`：直接取全文最后一个 `@` 会被「查询参数里含 @」的连接串
// 误导（如 mongodb://u:p@host/db?x=a@b），那种情况下反而什么都不掩，等于泄露凭证。
// 无 userinfo 或无密码时原样返回。
// userInfoSpan 定位连接串 authority 内的 userinfo 段（`user:password@` 去掉尾部 `@` 的部分）。
//
// 必须先在 authority 边界（第一个 `/ ? #` 之前）内找最后一个 `@`：拿全文最后一个 `@` 会被
// 查询参数里含 `@` 的连接串误导（如 mongodb://u:p@host/db?x=a@b），那样既掩不掉密码也判错凭证。
// 返回的区间是 uri 上的左闭右开下标；没有 userinfo（匿名连接串）时 ok 为 false。
func userInfoSpan(uri string) (start, end int, ok bool) {
	schemeIdx := strings.Index(uri, "://")
	if schemeIdx < 0 {
		return 0, 0, false
	}

	rest := uri[schemeIdx+3:]
	authority := rest
	if idx := strings.IndexAny(rest, "/?#"); idx >= 0 {
		authority = rest[:idx]
	}

	atIdx := strings.LastIndex(authority, "@")
	if atIdx < 0 {
		return 0, 0, false
	}
	return schemeIdx + 3, schemeIdx + 3 + atIdx, true
}

// HasCredentials 报告连接串里是否写了账号密码。
//
// MongoDB 的 `ping` 允许匿名执行，所以「连得上」不代表「有权限」：没配凭证的连接串在要求认证的
// 服务器上照样 ping 得通，却要等到执行 listDatabases 才报 Unauthorized。要区分这两种情况只能看凭证。
func HasCredentials(uri string) bool {
	start, end, ok := userInfoSpan(uri)
	return ok && end > start
}

// MaskUri 掩去 Mongo 连接串里的密码。
//
// 覆盖 `mongodb://` 与 `mongodb+srv://` 两种 scheme：旧实现用一条只匹配 `^mongodb://` 的正则，
// `+srv` 场景下掩码空跑，日志里落下的就是明文凭证。
// 密码是 userinfo 里最后一个 `:` 与结尾之间的部分；只有用户名没有密码时原样返回。
func MaskUri(uri string) string {
	start, end, ok := userInfoSpan(uri)
	if !ok {
		return uri
	}

	userInfo := uri[start:end]
	colonIdx := strings.LastIndex(userInfo, ":")
	if colonIdx < 0 {
		// 只有用户名没有密码，无需掩码
		return uri
	}

	schemeEnd := strings.Index(uri, "://") + 3
	atIdx := end
	var builder strings.Builder
	builder.Grow(len(uri))
	builder.WriteString(uri[:schemeEnd])
	builder.WriteString(userInfo[:colonIdx])
	builder.WriteString(":" + maskedPassword + "@")
	builder.WriteString(uri[atIdx+1:])
	return builder.String()
}
