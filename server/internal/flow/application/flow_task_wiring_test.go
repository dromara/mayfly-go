package application

import (
	"context"
	"os"
	"strings"
	"testing"

	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
	"mayfly-go/pkg/utils/collx"

	"github.com/stretchr/testify/require"
)

// TestApprovalResultReadableForCompletionCondition 审批结果必须在完成条件求值之前就可读。
//
// 完成条件与连线条件共用同一份字段字典，`approvalResult` 因此可以配进完成条件；
// 但它原先只在推进执行流时写进 OpExtra，配了这种条件的节点会永远判不出完成——
// 不报错、不提示，流程静默停在待审批。顺序写反了编译器不会报错，只能这样守
func TestApprovalResultReadableForCompletionCondition(t *testing.T) {
	content, err := os.ReadFile("procinst_task.go")
	if err != nil {
		t.Fatalf("read procinst_task.go failed: %v", err)
	}
	source := string(content)

	start := strings.Index(source, "func (p *procinstTaskAppImpl) CompleteTask(")
	if start < 0 {
		t.Fatal("CompleteTask not found")
	}
	end := strings.Index(source[start+len("func (p *procinstTaskAppImpl) CompleteTask("):], "\nfunc ")
	body := source[start : start+len("func (p *procinstTaskAppImpl) CompleteTask(")+end]

	set := strings.Index(body, "vars.Set(flowFieldApprovalResult")
	check := strings.Index(body, "IsUserTaskComplete(")
	if set < 0 || check < 0 {
		t.Fatalf("the completion path must both publish and consume the approval result, set=%d check=%d", set, check)
	}
	if set > check {
		t.Fatal("approvalResult must be written into the vars BEFORE the completion condition is evaluated, otherwise such a condition can never be satisfied")
	}
}

// TestUnknownConditionFieldsAreLogged 条件引用了取不到的字段时必须留痕。
//
// 引擎按「不成立」处理是刻意的（猜一个值会让流程走错分支，代价是跳过审批），
// 但一句都不留就等于让排查者对着一个不动的节点猜
func TestUnknownConditionFieldsAreLogged(t *testing.T) {
	content, err := os.ReadFile("node_usertask.go")
	if err != nil {
		t.Fatalf("read node_usertask.go failed: %v", err)
	}
	if !strings.Contains(string(content), "tc.Unknown") {
		t.Fatal("IsUserTaskComplete must report the fields it could not resolve")
	}
}

// TestCompletionConditionOnApprovalResult 以「审批结果=通过」为完成条件时必须判得出结果。
//
// 这条条件在旧实现里恒判不出（事实写在求值之后），节点会静默停在待审批；
// 这里直接守求值面，确保修的不是只是接线而是真的能用
func TestCompletionConditionOnApprovalResult(t *testing.T) {
	condition := &entity.RuleNode{Kind: entity.NodeKindCondition, Field: flowFieldApprovalResult, Op: trigger.OpEq, Value: ApprovalResultCompleted}

	complete, err := IsUserTaskComplete(context.Background(), condition, collx.M{flowFieldApprovalResult: ApprovalResultCompleted})
	require.NoError(t, err)
	require.True(t, complete, "a completion condition on the approval result must be satisfiable")

	// 结果还不是「通过」时不能算完成：否则第一个审批人就把会签节点整个放过
	notYet, err := IsUserTaskComplete(context.Background(), condition, collx.M{flowFieldApprovalResult: ApprovalResultProcessing})
	require.NoError(t, err)
	require.False(t, notYet)
}
