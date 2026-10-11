package mssql

// mssql 执行计划分析（可选能力 ExplainAnalyzer 的 mssql 实现）。
//
// 用 SET SHOWPLAN_XML —— SQL Server 提供的机器可读执行计划格式。SHOWPLAN 系列
// 只编译并返回计划、**绝不执行**被分析语句（对 UPDATE/DELETE 尤为关键，等价于 mysql 用 EXPLAIN 而非 ANALYZE）。
// 且 SET SHOWPLAN 必须是所在批次的唯一语句，故钉住一条连接分批次执行：开 → 取计划 → 无论成败都复位，
// 避免连接带着 SHOWPLAN 会话状态回到连接池污染后续查询。
//
// 解析用 RelOp 出入栈的流式遍历：showplan 的子算子既可能嵌在 <Inputs>（多数算子），也可能直接嵌在算子专属
// 元素里（如 <Update>/<ComputeScalar> 内层就是其输入 RelOp），声明式 struct 难以覆盖所有嵌套形态，栈遍历则与
// 嵌套位置无关，能可靠还原父子、判全表扫描、收集所用索引。
//
// Rows 取根 RelOp 的 EstimateRows（优化器估算），会因统计信息陈旧偏离真实值，故 EstimateKind 固定 explain。

import (
	"context"
	"encoding/xml"
	"io"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"

	"github.com/spf13/cast"
)

var _ dbi.ExplainAnalyzer = (*MssqlMetadata)(nil)

// mssqlIsFullScan 物理算子名含 "Scan" 即整表/整索引顺序扫描（Table Scan / Clustered Index Scan / Index Scan）；
// 含 "Seek" 的走定位不算。判据只在这里出现一次，上层不再按文本猜第二遍。
func mssqlIsFullScan(physicalOp string) bool {
	return strings.Contains(strings.ToLower(physicalOp), "scan")
}

func (md *MssqlMetadata) Analyze(ctx context.Context, sql string) (*dbi.ExplainResult, error) {
	stmt := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(sql), ";"))
	if stmt == "" {
		return nil, errorx.NewBiz("sql to analyze is empty")
	}
	upper := strings.ToUpper(stmt)
	if strings.HasPrefix(upper, "EXPLAIN") || strings.HasPrefix(upper, "DESC") || strings.Contains(upper, "SHOWPLAN") {
		return nil, errorx.NewBiz("the statement to analyze must not be an EXPLAIN/SHOWPLAN itself")
	}

	planText, err := md.showplanXml(ctx, stmt)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(planText, "ShowPlanXML") {
		return nil, errorx.NewBiz("explain returned no showplan xml")
	}
	return parseMssqlShowplan(planText)
}

// showplanXml 在钉住的单连接上分批次开启 SHOWPLAN_XML、取被分析语句（不执行）的计划 XML、结束复位。
func (md *MssqlMetadata) showplanXml(ctx context.Context, stmt string) (string, error) {
	db := md.di.GetDb()
	if db == nil {
		return "", errorx.NewBiz("no underlying connection to analyze against")
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	// defer 用独立 context 复位：即便调用方 ctx 已取消/超时，也要保证 SHOWPLAN 关掉，不泄漏给连接池
	defer func() { _, _ = conn.ExecContext(context.Background(), "SET SHOWPLAN_XML OFF") }()

	if _, err := conn.ExecContext(ctx, "SET SHOWPLAN_XML ON"); err != nil {
		return "", err
	}
	rows, err := conn.QueryContext(ctx, stmt)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	// 计划以单行单列 SHOWPLAN_XML 返回（驱动给 []byte，Scan 进 string 自动转换）
	var planText string
	if rows.Next() {
		if err := rows.Scan(&planText); err != nil {
			return "", err
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return planText, nil
}

// parseMssqlShowplan 遍历 showplan XML：每个 <RelOp> 依其嵌套栈建立父子边；顺带在 <Object> 上取表名与索引名。
func parseMssqlShowplan(planText string) (*dbi.ExplainResult, error) {
	result := &dbi.ExplainResult{EstimateKind: dbi.EstimateKindExplain}
	seenIndex := make(map[string]bool)
	stack := make([]int, 0, 8) // 当前 RelOp 祖先链在 result.Plan 中的下标
	rowsSet := false
	// cur：栈顶节点在 result.Plan 中的下标；无 RelOp 上下文时为 -1
	cur := -1

	dec := xml.NewDecoder(strings.NewReader(planText))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errorx.NewBizf("failed to parse the explain plan: %s", err.Error())
		}
		switch el := tok.(type) {
		case xml.StartElement:
			switch el.Name.Local {
			case "RelOp":
				physicalOp, estRows := relOpAttrs(el.Attr)
				fullScan := mssqlIsFullScan(physicalOp)
				if fullScan {
					result.FullScan = true
				}
				idx := len(result.Plan)
				result.Plan = append(result.Plan, dbi.PlanNode{
					NodeKind:     physicalOp,
					AccessMethod: physicalOp,
					EstRows:      estRows,
					FullScan:     fullScan,
				})
				if len(stack) > 0 {
					parent := stack[len(stack)-1]
					result.Plan[parent].Children = append(result.Plan[parent].Children, idx)
				} else if !rowsSet {
					// 根 RelOp 的行数估算即整条语句的估算
					result.Rows = estRows
					rowsSet = true
				}
				stack = append(stack, idx)
				cur = idx
			case "Object":
				if cur >= 0 {
					table, index := objectAttrs(el.Attr)
					if table != "" && result.Plan[cur].Table == "" {
						result.Plan[cur].Table = table
					}
					if index != "" && !seenIndex[index] {
						seenIndex[index] = true
						result.UsedIndexes = append(result.UsedIndexes, index)
						result.Plan[cur].IndexName = index
					}
				}
			}
		case xml.EndElement:
			if el.Name.Local == "RelOp" && len(stack) > 0 {
				stack = stack[:len(stack)-1]
				cur = -1
				if len(stack) > 0 {
					cur = stack[len(stack)-1]
				}
			}
		}
	}

	if len(result.Plan) == 0 {
		return nil, errorx.NewBiz("explain returned no plan node")
	}
	return result, nil
}

// relOpAttrs 取 RelOp 的 PhysicalOp 与 EstimateRows 属性（无命名空间，按本地名匹配）。
func relOpAttrs(attrs []xml.Attr) (physicalOp string, estRows float64) {
	for _, a := range attrs {
		switch a.Name.Local {
		case "PhysicalOp":
			physicalOp = a.Value
		case "EstimateRows":
			estRows = cast.ToFloat64(a.Value)
		}
	}
	return
}

// objectAttrs 取 <Object> 的 Table 与 Index 属性；方括号是 mssql 标识符引用，剥掉以保持干净事实。
func objectAttrs(attrs []xml.Attr) (table, index string) {
	for _, a := range attrs {
		switch a.Name.Local {
		case "Table":
			table = strings.Trim(a.Value, "[]")
		case "Index":
			index = strings.Trim(a.Value, "[]")
		}
	}
	return
}
