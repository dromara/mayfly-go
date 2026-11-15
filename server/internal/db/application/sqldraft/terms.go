package sqldraft

// 业务词典（语义层）：术语 → 表 + 口径说明的登记与读取。
//
// 读侧有两个消费者，都在本包内：glossaryLinker（描述命中术语就把登记的表带进候选）与
// glossaryFragment（把口径交给模型）。写侧是管理界面。三者共用同一份实体与同一套
// 「该库可见」的过滤口径，因此收在同一个服务里，避免出现第二份词典语义。

import (
	"context"
	"strings"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/errorx"
)

// maxTermsPerDb 单个库资源的词典条目上限。
//
// 上限不是防「用户太勤快」：条目会整段进上下文，无界登记等于让一次生成的 token 消耗与
// 上下文预算由管理员的手速决定。超限直接报错让他精简，而不是静默丢弃后排在后面的条目
// 永远读不到（静默丢弃会让用户以为登记生效了）。
const maxTermsPerDb = 50

// termAuditColumns 显式列更新必须一并写入的审计列。
//
// UpdateById 走 Select(columns).Updates()，只写列出的列：漏了审计列的表现是「改了词典
// 但查不到是谁改的」，而这条路径低频、编译与单测都不会报错。
var termAuditColumns = []string{"modifier", "modifier_id", "update_time"}

// Terms 业务词典服务
type Terms interface {
	// ListByDb 某库资源下全部条目（管理界面，含其它库的条目）
	ListByDb(ctx context.Context, dbId uint64) ([]*entity.DbAiTerm, error)

	// ListForSchema 本次生成可用的条目：该库登记的 + 声明全库共用的（Schema 为空）
	ListForSchema(ctx context.Context, dbId uint64, schema string) ([]*entity.DbAiTerm, error)

	// GetTerm 按 id 取条目（删除前先拿到它自己的库，再按那个库判可达）
	GetTerm(ctx context.Context, id uint64) (*entity.DbAiTerm, error)

	// SaveTerm 保存条目：按「库 + 术语忽略大小写」去重后更新，等价于 upsert
	SaveTerm(ctx context.Context, term *entity.DbAiTerm) error

	// DeleteTerm 删除条目：id 必须属于给定库，否则按「不存在」处理
	DeleteTerm(ctx context.Context, dbId, id uint64) error
}

type TermsAppImpl struct {
	termRepo repository.DbAiTerm `inject:"T"`
}

var _ Terms = (*TermsAppImpl)(nil)

func (t *TermsAppImpl) ListByDb(_ context.Context, dbId uint64) ([]*entity.DbAiTerm, error) {
	if dbId == 0 {
		return []*entity.DbAiTerm{}, nil
	}
	return t.termRepo.ListByDb(dbId)
}

func (t *TermsAppImpl) ListForSchema(ctx context.Context, dbId uint64, schema string) ([]*entity.DbAiTerm, error) {
	terms, err := t.ListByDb(ctx, dbId)
	if err != nil {
		return nil, err
	}
	want := strings.TrimSpace(schema)
	out := make([]*entity.DbAiTerm, 0, len(terms))
	for _, term := range terms {
		// 登记的库名为空 = 该资源下所有库共用（全库型资源一次登记即可）
		if term.Schema == "" || want == "" || strings.EqualFold(term.Schema, want) {
			out = append(out, term)
		}
	}
	return out, nil
}

func (t *TermsAppImpl) GetTerm(ctx context.Context, id uint64) (*entity.DbAiTerm, error) {
	if id == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrAiTermNotFound)
	}
	exist, err := t.termRepo.GetById(id)
	if err != nil {
		return nil, err
	}
	if exist == nil || exist.Id == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrAiTermNotFound)
	}
	return exist, nil
}

func (t *TermsAppImpl) SaveTerm(ctx context.Context, term *entity.DbAiTerm) error {
	if term == nil {
		return errorx.NewBizI(ctx, imsg.ErrAiTermRequired)
	}
	term.Term = strings.TrimSpace(term.Term)
	term.Schema = strings.TrimSpace(term.Schema)
	term.Details = strings.TrimSpace(term.Details)
	term.Tables = normalizeTermTables(term.Tables)
	if term.DbId == 0 || term.Term == "" {
		return errorx.NewBizI(ctx, imsg.ErrAiTermRequired)
	}
	if term.Id > 0 {
		return t.termRepo.UpdateById(ctx, term, append([]string{"schema_name", "term", "table_names", "details"}, termAuditColumns...)...)
	}
	terms, err := t.termRepo.ListByDb(term.DbId)
	if err != nil {
		return err
	}
	if len(terms) >= maxTermsPerDb {
		return errorx.NewBizI(ctx, imsg.ErrAiTermLimitExceeded, "max", maxTermsPerDb)
	}
	for _, exist := range terms {
		if !strings.EqualFold(exist.Term, term.Term) || !strings.EqualFold(exist.Schema, term.Schema) {
			continue
		}
		// 同库同名术语直接覆盖已有行：插两行会让「哪条生效」取决于查询顺序，
		// 而词典是喂给模型的材料，两份互相矛盾的口径比没有口径更糟
		exist.Tables = term.Tables
		exist.Details = term.Details
		return t.termRepo.UpdateById(ctx, exist, append([]string{"table_names", "details"}, termAuditColumns...)...)
	}
	return t.termRepo.Insert(ctx, term)
}

func (t *TermsAppImpl) DeleteTerm(ctx context.Context, dbId, id uint64) error {
	exist, err := t.termRepo.GetById(id)
	if err != nil {
		return err
	}
	// 查不到、已删除、或属于别的库，对外都是「不存在」：
	// 借 id 删他库条目必须失败，但也不能用不同文案告诉调用方「那条存在、只是不是你的」
	if exist == nil || exist.Id == 0 || (dbId != 0 && exist.DbId != dbId) {
		return errorx.NewBizI(ctx, imsg.ErrAiTermNotFound)
	}
	return t.termRepo.DeleteById(ctx, id)
}

// normalizeTermTables 去掉空白项并按忽略大小写去重（保留首次出现的原样写法）
func normalizeTermTables(tables []string) []string {
	if len(tables) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tables))
	out := make([]string, 0, len(tables))
	for _, table := range tables {
		name := strings.TrimSpace(table)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
