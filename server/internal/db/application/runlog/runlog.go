// Package runlog 为「任务运行日志」提供尺寸守卫：数据同步与数据迁移的执行过程日志
// 都落在 run_log 这一 text 列上（MySQL 上限 65535 字节），而日志行数会随表数、分片数、
// 批次数线性增长，校验类任务还会一次性写入整段报告 JSON。
//
// 为什么独立成包：两侧超限的后果都不是「日志变短」而是「任务出事」——
//   - 同步侧：日志在分片中途写失败，MySQL 严格模式下整轮任务直接失败（非严格模式静默截断，
//     此后日志不再更新，看起来像任务卡住）；
//   - 迁移侧：日志只在收尾一次性落库，超限会让收尾 UPDATE 整行失败，日志永远停在「执行中」
//     且行数/耗时指标全部丢失，排查现场时被误导。
//
// 两处必须共用同一条不变式（结果长度恒 <= MaxBytes），各写一份迟早漂移，故收在此处。
package runlog

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// MaxBytes 运行日志软上限：低于 text 的 65535 字节，给列包装与多字节字符留出余量。
const MaxBytes = 60000

// 折叠后的保留额度：头部留住一次性上下文（任务/模式/字段映射/完整 SQL），尾部留住最近的
// 进度——排障时最关心的就是「最后跑到哪了」。两者之和须明显小于 MaxBytes，使 Fold 的
// 出口长度可控（见 Fold 的长度不变式说明）。
const (
	keepHeadBytes = 12000
	keepTailBytes = 40000
)

// Append 追加一行，并在累计超出上限时折叠中段。
//
// 调用方只负责产出整行内容，长度治理由本包负责，避免各处 append 点各写一份判断。
func Append(runLog, line string) string {
	joined := runLog + line
	if len(joined) <= MaxBytes {
		return joined
	}
	return Fold(joined)
}

// Fold 保留头尾、省略中段，返回值长度恒不超过 MaxBytes。
//
// 头尾优先取整行边界：日志含中文，按字节硬切会切碎 UTF-8 字符，写库时报 Incorrect string
// value，那比日志变短更硬。但当单行本身比保留窗口还长（校验报告整段 JSON 就是一行）时整行
// 边界无从下手，退化为在窗口边界按字符回退/前进切——宁可少留内容，也不能交出超限的值。
//
// 长度不变式：head <= keepHeadBytes、tail <= keepTailBytes、标记行不足百字节，
// 故返回值恒 < MaxBytes；重复 Fold 也只会更短（幂等），不会反弹。
func Fold(runLog string) string {
	if len(runLog) <= MaxBytes {
		return runLog
	}
	headEnd := headCut(runLog, keepHeadBytes)
	tailStart := tailCut(runLog, keepTailBytes)
	if tailStart <= headEnd {
		// 按当前保留额度（head+tail 远小于 MaxBytes）这一分支不可达；保留它是防线：
		// 有人把额度调到接近上限时，也不能交出超限内容（宁可只留头部窗口）
		return runLog[:headEnd] + omitted(runLog[headEnd:])
	}
	return runLog[:headEnd] + omitted(runLog[headEnd:tailStart]) + runLog[tailStart:]
}

// omitted 生成省略标记：同时给行数与字节数。只给行数的话，单行超长（无换行）时会写出
// 「省略 0 行」这种看起来像没丢内容的误导文案。
func omitted(dropped string) string {
	return fmt.Sprintf("… 日志中段已省略（%d 行 / %d 字节，累计超出 %d 字节上限）…\n",
		strings.Count(dropped, "\n"), len(dropped), MaxBytes)
}

// headCut 返回头部保留的截止偏移：不超过 n，且尽量停在完整行之后
func headCut(s string, n int) int {
	if n >= len(s) {
		return len(s)
	}
	if i := strings.LastIndexByte(s[:n], '\n'); i >= 0 {
		return i + 1
	}
	// n 之内没有换行：这一行比窗口还长，按字符边界硬切（向前回退，避免切碎多字节字符）
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return n
}

// tailCut 返回尾部保留的起始偏移：尾部不长于 n，且尽量从整行开头开始
func tailCut(s string, n int) int {
	start := len(s) - n
	if start <= 0 {
		return 0
	}
	if i := strings.IndexByte(s[start:], '\n'); i >= 0 {
		return start + i + 1
	}
	// n 之内没有换行：按字符边界起留（向后前进，切掉的半个字符不属于尾部）
	for start < len(s) && !utf8.RuneStart(s[start]) {
		start++
	}
	return start
}
