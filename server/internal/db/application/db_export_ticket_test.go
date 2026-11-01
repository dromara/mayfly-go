package application

// 导出工单作为「下载凭据」的判定：类型对、已通过、未过期、申请人本人、范围非空。
// 每一条都对应一种真实的越权或误放行，缺一条就会把工单变成通用下载通道。

import (
	"context"
	"testing"
	"time"

	flowapp "mayfly-go/internal/flow/application"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/model"
)

// fakeProcinstApp 只提供取实例的能力，其余接口方法由嵌入的接口占位
type fakeProcinstApp struct {
	flowapp.Procinst
	procinst *flowentity.Procinst
	err      error
}

func (f *fakeProcinstApp) GetById(id uint64, cols ...string) (*flowentity.Procinst, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.procinst, nil
}

func exportTicketAppWith(procinst *flowentity.Procinst) *dbExportTicketAppImpl {
	return &dbExportTicketAppImpl{procinstApp: &fakeProcinstApp{procinst: procinst}}
}

func approvedExportTicket(endTime time.Time) *flowentity.Procinst {
	return &flowentity.Procinst{
		BizType:   DbExportFlowBizType,
		Status:    flowentity.ProcinstStatusCompleted,
		BizForm:   `{"dbId":1,"dbName":"mayfly_go","tables":["t_a"],"dumpType":"3","reason":"排查"}`,
		EndTime:   &endTime,
		CreatorId: 7,
	}
}

func TestValidateExportTicket(t *testing.T) {
	ctx := contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: 7})
	now := time.Now()

	// 正常路径
	app := exportTicketAppWith(approvedExportTicket(now.Add(-time.Hour)))
	form, err := app.ValidateForDownload(ctx, 1)
	if err != nil || form == nil || form.DbName != "mayfly_go" || len(form.Tables) != 1 {
		t.Fatalf("a valid ticket must pass: form=%+v err=%v", form, err)
	}

	cases := []struct {
		name     string
		procinst *flowentity.Procinst
		wantErr  string
	}{
		{"missing", nil, "找不到"},
		{"wrong biz type", &flowentity.Procinst{BizType: "db_sql_exec_flow", Status: flowentity.ProcinstStatusCompleted, EndTime: &now, CreatorId: 7}, "不是导出工单"},
		{"not approved", &flowentity.Procinst{BizType: DbExportFlowBizType, Status: flowentity.ProcinstStatusActive, CreatorId: 7}, "尚未审批通过"},
		{"expired", approvedExportTicket(now.Add(-25 * time.Hour)), "超出可下载时限"},
		{"no end time", &flowentity.Procinst{BizType: DbExportFlowBizType, Status: flowentity.ProcinstStatusCompleted, CreatorId: 7}, "超出可下载时限"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := exportTicketAppWith(tc.procinst).ValidateForDownload(ctx, 1)
			if err == nil {
				t.Fatalf("%s: must be rejected", tc.name)
			}
			if !contains(err.Error(), tc.wantErr) {
				t.Fatalf("%s: unexpected message: %s", tc.name, err.Error())
			}
		})
	}

	// 别人的工单不能借用
	other := contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: 8})
	if _, err := exportTicketAppWith(approvedExportTicket(now)).ValidateForDownload(other, 1); err == nil || !contains(err.Error(), "申请人") {
		t.Fatalf("another account must be rejected, got %v", err)
	}

	// 没有登录态一律拒绝
	if _, err := exportTicketAppWith(approvedExportTicket(now)).ValidateForDownload(context.Background(), 1); err == nil {
		t.Fatal("a request without login account must be rejected")
	}

	// 工单没写清范围（表清单为空）不构成授权
	empty := approvedExportTicket(now)
	empty.BizForm = `{"dbId":1,"dbName":"mayfly_go","tables":[],"dumpType":"3"}`
	if _, err := exportTicketAppWith(empty).ValidateForDownload(ctx, 1); err == nil || !contains(err.Error(), "导出范围") {
		t.Fatalf("an empty scope must be rejected, got %v", err)
	}

	// 工单 id 为 0 直接拒绝，不去查库
	if _, err := exportTicketAppWith(nil).ValidateForDownload(ctx, 0); err == nil {
		t.Fatal("a zero procinst id must be rejected")
	}
}

func contains(text, sub string) bool {
	return len(sub) == 0 || (len(text) >= len(sub) && indexOf(text, sub) >= 0)
}

func indexOf(text, sub string) int {
	for i := 0; i+len(sub) <= len(text); i++ {
		if text[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
