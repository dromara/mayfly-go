package migrations

// 内置技能内容（对齐菜单内置进 SQL 的方式：内容随迁移硬编码落库 t_ai_skill，
// 运行时不依赖 embed，技能统一走 DB 管理，用户可在插件管理中编辑/删除）
//
// Body 为 SKILL.md 正文；frontmatter 由 skill.BuildSkillMd 按元数据生成。

type builtinSkill struct {
	Code        string
	Name        string
	Description string
	Body        []string // 正文行（含反引号故逐行存储，避免 raw string 限制）
}

var builtinSkills = []builtinSkill{
	{
		Code:        "db-ops-guide",
		Name:        "数据库操作手册",
		Description: "数据库查询与 SQL 执行规范：只读优先、分页查询、索引意识、危险语句防护与执行前确认",
		Body: []string{
			"# 数据库操作手册",
			"",
			"## 核心原则",
			"",
			"1. **只读优先**：优先使用 `DbQueryData`、`DbQueryTables`、`DbQueryTableDDL` 等只读工具获取信息，仅在用户明确要求修改数据时才使用 `ExecSql`。",
			"2. **先探查后执行**：执行任何写操作前，先用只读查询确认目标数据的存在与状态（如先 SELECT 再 UPDATE/DELETE）。",
			"3. **分页查询**：查询大表时必须带 LIMIT（建议 100-500 行），禁止无 WHERE 条件的全表扫描式查询。",
			"",
			"## SQL 编写规范",
			"",
			"- 使用显式列名，避免 `SELECT *`（除非用户明确要求全部列）",
			"- 判断表是否存在/查看结构时使用 `DbQueryTableDDL`，不要凭空猜测列名",
			"- MySQL 环境注意 `ONLY_FULL_GROUP_BY` 模式：`SELECT DISTINCT` 与聚合函数并用时注意列一致性",
			"- 时间条件优先使用数据库兼容的格式（如 `2024-01-01 00:00:00`）",
			"- 更新与删除语句必须带精确的 WHERE 条件，并先用 SELECT 验证影响范围",
			"",
			"## 危险操作防护",
			"",
			"以下操作会触发执行确认（中断等待用户审批），属于正常安全机制，不要试图绕过：",
			"",
			"- `DROP` / `TRUNCATE` 表",
			"- 无 WHERE 条件的 `UPDATE` / `DELETE`",
			"- 大批量数据变更（预估影响行数较大时）",
			"",
			"执行确认被用户拒绝后，不要重复尝试相同操作；应询问用户的顾虑并给出替代方案。",
			"",
			"## 结果解读",
			"",
			"- 查询结果为空时，先检查 dbName/dbId 参数是否正确（参数缺失会触发参数补全中断，等待用户选择资产）",
			"- 报告行数与数据摘要即可，不要向用户倾倒大量原始数据",
		},
	},
	{
		Code:        "machine-ops-guide",
		Name:        "机器操作手册",
		Description: "机器命令执行规范：只读探查优先、危险命令确认、文件与日志操作技巧",
		Body: []string{
			"# 机器操作手册",
			"",
			"## 核心原则",
			"",
			"1. **只读探查优先**：排查问题时优先使用只读命令（`ps`、`top`、`df`、`free`、`netstat`、`cat`、`grep`、`tail`），避免直接执行变更类命令。",
			"2. **单步执行**：复杂操作拆分为多步，每步确认结果后再进行下一步，不要拼接过长的命令链。",
			"3. **明确路径**：使用绝对路径操作文件；不确定路径时先用 `ls` / `find` 探查，禁止对不确定的目标执行删除。",
			"",
			"## 常用排查命令",
			"",
			"| 场景 | 命令 |",
			"|------|------|",
			"| CPU/内存 | `top -bn1`、`free -h` |",
			"| 磁盘 | `df -h`、`du -sh <dir>` |",
			"| 端口占用 | `netstat -tlnp \\| grep <port>` 或 `lsof -i:<port>` |",
			"| 进程 | `ps aux \\| grep <name>` |",
			"| 日志 | `tail -n 200 <log>`、`grep -C 5 \"ERROR\" <log>` |",
			"",
			"## 危险命令防护",
			"",
			"以下命令会触发执行确认（中断等待用户审批），属于正常安全机制：",
			"",
			"- `rm` / `rmdir` / `mv`（覆盖目标）/ `chmod -R` / `chown -R`",
			"- `kill` / `pkill` / `killall`",
			"- 服务启停（`systemctl start/stop/restart`）",
			"- 任何涉及覆盖、清空文件内容的重定向（`>`、`>!`）",
			"",
			"确认被拒绝后不要重复尝试；应与用户确认替代方案。",
			"",
			"## 日志查看技巧",
			"",
			"- 优先 `tail -n` 取最近日志，再按需扩大范围",
			"- 大文件用 `grep` 过滤而非全量输出",
			"- 输出内容做摘要归纳（错误关键词、时间点、频次），不要原样倾倒给用户",
		},
	},
}
