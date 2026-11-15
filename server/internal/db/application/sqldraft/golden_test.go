package sqldraft

// golden 用例集：把「一句话需求 → 生成的 SQL 应该长什么样、结论应该是什么」固化下来。
//
// 为什么需要它：模型是概率系统，单测只能钉住编排逻辑；真实准确率要靠一批固定用例反复跑。
// 这里的分工是：
//   - 本文件（默认门禁，离线）：用**录制下来的真实模型输出**跑完整管线，断言召回、校验结论、
//     自纠轮数、取值核对——它保证「同一段模型输出，我们的判定不会悄悄变松/变严」；
//   - golden_live_test.go（-tags golden，需真机）：用真实模型 + 真实库重跑，断言
//     「能执行且结果形状符合预期」，这才是准确率本身。
//
// 用例来源全部是本轮真机实测过的形态（含踩过的坑），不是编造的样例。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/dbi"
)

// goldenCase golden 用例（字段与 testdata/golden_cases.json 对应）
type goldenCase struct {
	ID     string `json:"id"`
	Intent string `json:"intent"`
	Prompt string `json:"prompt"`
	Tables []struct {
		Name    string `json:"name"`
		Comment string `json:"comment"`
		Columns []struct {
			Name string `json:"name"`
			Type string `json:"type"`
			PK   bool   `json:"pk"`
		} `json:"columns"`
	} `json:"tables"`
	// SampledValues 该用例下「列取值采样」应返回的值（键为 table.column）
	SampledValues map[string][]string `json:"sampledValues"`
	// Chooser 语义召回（表选择题）的回复
	Chooser []string `json:"chooser"`
	// Model 依次返回的模型输出（真实录制）：第一条是首次生成，第二条起是自纠回复
	Model []string `json:"model"`
	// GateBlock 审批闸门的结论："" 放行 / "approval" 需工单 / "warn_ack" 需确认
	GateBlock string `json:"gateBlock"`
	Expect    struct {
		PickedContains []string `json:"pickedContains"`
		Verification   string   `json:"verification"`
		FindingCodes   []string `json:"findingCodes"`
		Repairs        *int     `json:"repairs"`
	} `json:"expect"`
}

// fakeSampler 列取值采样替身（按用例给定的值返回）
type fakeSampler struct {
	values map[string][]string
}

func (f *fakeSampler) ColumnValues(_ context.Context, _ uint64, _ string, table, column string, _ int) ([]string, error) {
	return f.values[strings.ToLower(table+"."+column)], nil
}

// loadGoldenCases 读取 golden 用例
func loadGoldenCases(t *testing.T) []goldenCase {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "golden_cases.json"))
	if err != nil {
		t.Fatalf("read golden cases: %v", err)
	}
	var cases []goldenCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("parse golden cases: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("golden cases are empty: 用例为空等于没有评测")
	}
	return cases
}

// runGoldenCase 用用例夹具跑一次完整管线
func runGoldenCase(t *testing.T, one goldenCase) *Draft {
	t.Helper()
	meta := &fakeMeta{tables: make([]dbi.Table, 0, len(one.Tables)), columns: make(map[string][]dbi.Column, len(one.Tables))}
	for _, table := range one.Tables {
		meta.tables = append(meta.tables, dbi.Table{TableName: table.Name, TableComment: table.Comment})
		columns := make([]dbi.Column, 0, len(table.Columns))
		for _, column := range table.Columns {
			columns = append(columns, dbi.Column{TableName: table.Name, ColumnName: column.Name, ColumnType: column.Type, IsPrimaryKey: column.PK})
		}
		meta.columns[strings.ToLower(table.Name)] = columns
	}
	dialect := dbi.GetDialect(dbi.ToDbType("mysql"))
	snap, err := buildSnapshot(context.Background(), meta, dialect, dbi.ToDbType("mysql"), &fakeAcl{}, 1, "golden")
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	spec, ok := specOf(one.Intent)
	if !ok {
		t.Fatalf("intent %s not registered", one.Intent)
	}
	gate := &fakeGate{kind: one.GateBlock}
	if one.GateBlock != "" {
		gate.err = &blockStub{}
	}
	sampler := &fakeSampler{values: make(map[string][]string, len(one.SampledValues))}
	for key, values := range one.SampledValues {
		sampler.values[strings.ToLower(key)] = values
	}
	run := &draftRun{
		req:     &Req{DbId: 1, Db: "golden", Intent: one.Intent, Prompt: one.Prompt, AnchorStatement: "select 1"},
		spec:    spec,
		snap:    snap,
		meta:    meta,
		dialect: dialect,
		acl:     &fakeAcl{},
		gate:    gate,
		// 闸门判定方法需要连接入参（真实实现要用它读库定位）；用例里闸门是替身，
		// 给一个零值连接即可——不传会让 policyChecker 按「无法判定」短路，用例就测不到分流
		conn:        &dbi.DbConn{},
		sampler:     sampler,
		sampleCache: make(map[string][]string, 4),
		gen:         newGenerator(&fakePort{replies: one.Model, chooseTables: one.Chooser}),
	}
	draft, err := run.execute(context.Background())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	// 召回断言用的 picked 存在 draft.ContextTables 里（对外只暴露依据表）
	_ = snap
	return draft
}

// blockStub 一个非 nil 的错误替身（fakeGate 只在 err != nil 时走结论分支）
type blockStub struct{}

func (*blockStub) Error() string { return "blocked by policy" }

// TestGoldenCases 默认门禁：录制回放全链路断言
func TestGoldenCases(t *testing.T) {
	for _, one := range loadGoldenCases(t) {
		one := one
		t.Run(one.ID, func(t *testing.T) {
			draft := runGoldenCase(t, one)

			picked := make(map[string]struct{}, len(draft.ContextTables))
			for _, table := range draft.ContextTables {
				picked[strings.ToLower(table.Name)] = struct{}{}
			}
			for _, want := range one.Expect.PickedContains {
				if _, ok := picked[strings.ToLower(want)]; !ok {
					t.Fatalf("expect table %s in context, got %v", want, draft.ContextTables)
				}
			}

			if draft.Verification != one.Expect.Verification {
				t.Fatalf("verification: want %q got %q (checks=%+v)", one.Expect.Verification, draft.Verification, draft.Checks)
			}

			codes := make(map[string]struct{}, len(draft.Checks))
			for _, finding := range draft.Checks {
				codes[finding.Code] = struct{}{}
			}
			for _, want := range one.Expect.FindingCodes {
				if _, ok := codes[want]; !ok {
					t.Fatalf("expect finding %q, got %+v", want, draft.Checks)
				}
			}
			if one.Expect.Repairs != nil && draft.Repairs != *one.Expect.Repairs {
				t.Fatalf("repairs: want %d got %d", *one.Expect.Repairs, draft.Repairs)
			}
			// 散文意图必须产出解释且不带语句
			if one.Intent == "explain" {
				if draft.Explanation == "" || len(draft.Statements) != 0 {
					t.Fatalf("explain must return prose only, got explanation=%q statements=%d", draft.Explanation, len(draft.Statements))
				}
			}
		})
	}
}
