package persistence

import (
	"context"

	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type skillRepoImpl struct {
	base.RepoImpl[*entity.Skill]
}

var _ repository.Skill = (*skillRepoImpl)(nil)

func newSkillRepo() repository.Skill {
	return &skillRepoImpl{}
}

func (s *skillRepoImpl) SelectByCode(ctx context.Context, code string) (*entity.Skill, error) {
	// GetByCond 的条件与结果落点是同一个结构体（First(dest)），条件字段必须填进目标结构体本身
	skill := &entity.Skill{Code: code}
	if err := s.GetByCond(skill); err != nil {
		return nil, err
	}
	return skill, nil
}

func (s *skillRepoImpl) SelectByStatus(ctx context.Context, status string) ([]*entity.Skill, error) {
	cond := model.NewCond().Eq("status", status).OrderByAsc("code")
	return s.SelectByCond(cond)
}

type skillResourceRepoImpl struct {
	base.RepoImpl[*entity.SkillResource]
}

var _ repository.SkillResource = (*skillResourceRepoImpl)(nil)

func newSkillResourceRepo() repository.SkillResource {
	return &skillResourceRepoImpl{}
}

func (r *skillResourceRepoImpl) SelectBySkillId(ctx context.Context, skillId uint64) ([]*entity.SkillResource, error) {
	cond := model.NewCond().Eq("skillId", skillId).OrderByAsc("path")
	return r.SelectByCond(cond)
}

func (r *skillResourceRepoImpl) SelectBySkillIdAndPath(ctx context.Context, skillId uint64, path string) (*entity.SkillResource, error) {
	// GetByCond 的条件与结果落点是同一个结构体（First(dest)），必须把条件字段填进
	// 目标结构体本身，否则查询结果会被扫进临时对象而丢失（此前 res 恒为零值，
	// 导致 GetInstructions 恒返回空正文、upsert 更新路径恒失效）。
	res := &entity.SkillResource{SkillId: skillId, Path: path}
	if err := r.GetByCond(res); err != nil {
		return nil, err
	}
	return res, nil
}

func (r *skillResourceRepoImpl) DeleteBySkillId(ctx context.Context, skillId uint64) error {
	return r.DeleteByCond(ctx, &entity.SkillResource{SkillId: skillId})
}

type mcpServerRepoImpl struct {
	base.RepoImpl[*entity.McpServer]
}

var _ repository.McpServer = (*mcpServerRepoImpl)(nil)

func newMcpServerRepo() repository.McpServer {
	return &mcpServerRepoImpl{}
}

func (m *mcpServerRepoImpl) SelectByCode(ctx context.Context, code string) (*entity.McpServer, error) {
	// 同 SelectBySkillIdAndPath：条件与落点同一结构体
	server := &entity.McpServer{Code: code}
	if err := m.GetByCond(server); err != nil {
		return nil, err
	}
	return server, nil
}

func (m *mcpServerRepoImpl) SelectEnabled(ctx context.Context) ([]*entity.McpServer, error) {
	cond := model.NewCond().Eq("enabled", 1).OrderByAsc("id")
	return m.SelectByCond(cond)
}
