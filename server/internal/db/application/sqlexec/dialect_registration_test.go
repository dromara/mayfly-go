package sqlexec

// 本包的测试需要「生产同款方言注册表」：预检/执行共用判定、回滚字面量渲染都按 DbType 取已注册方言，
// 其中 rollbackLiteral 必须委托方言 Codec 渲染（否则 bigint 会丢位、BIT 会被塌缩成布尔、二进制会被当字符串）。
// 方言文件从 application 根包搬进本包时这条副作用导入未随迁，在此显式声明一次
import (
	_ "mayfly-go/internal/db/dbm"
)
