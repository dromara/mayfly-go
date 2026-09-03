package application

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	"gorm.io/gorm"

	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/domain/repository"
	"mayfly-go/internal/ai/skill"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
)

// zip 导入安全上限（对齐 zip 传输格式的防御性约束）
const (
	// maxZipBytes zip 压缩包大小上限
	maxZipBytes = 10 << 20
	// maxZipResourceSize 单个资源解压后大小上限
	maxZipResourceSize = 2 << 20
)

// skillCodeRegexp 技能 code 规则（暴露给 LLM 与 $code 提及，字符集受限）
var skillCodeRegexp = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

// SkillPlugin 技能插件管理服务（对齐 tokhub SkillService，剪裁多租户）
type SkillPlugin interface {
	base.App[*entity.Skill]

	// ListSkills 技能列表（code 排序）
	ListSkills(ctx context.Context) ([]*entity.Skill, error)

	// GetInstructions 读取 SKILL.md 原文（frontmatter + 正文）
	GetInstructions(ctx context.Context, id uint64) (string, error)

	// CreateSkill 在线创建技能（instructions 为 SKILL.md 正文）
	CreateSkill(ctx context.Context, req *SkillSaveReq) (*entity.Skill, error)

	// UpdateSkill 更新技能元数据与正文（code 不可变；instructions 变更时重写 SKILL.md）
	UpdateSkill(ctx context.Context, id uint64, req *SkillSaveReq) error

	// DeleteSkill 删除技能（级联删除资源）
	DeleteSkill(ctx context.Context, id uint64) error

	// Publish / Unpublish 发布与转草稿（仅 published 注入 LLM）
	Publish(ctx context.Context, id uint64) error
	Unpublish(ctx context.Context, id uint64) error

	// ListResources 资源列表（不含 SKILL.md 主文件）
	ListResources(ctx context.Context, skillId uint64) ([]*entity.SkillResource, error)

	// UpsertResource 新增/更新资源（SKILL.md 主文件走 UpdateSkill）
	UpsertResource(ctx context.Context, skillId uint64, resPath, content string) (*entity.SkillResource, error)

	// DeleteResource 删除资源
	DeleteResource(ctx context.Context, skillId uint64, resPath string) error

	// ImportZip zip 导入（同 code upsert 覆盖 + SemVer patch 递增）
	ImportZip(ctx context.Context, data []byte) (*entity.Skill, error)

	// ExportZip zip 导出，返回文件名（{code}.zip）与内容
	ExportZip(ctx context.Context, id uint64) (string, []byte, error)

	// ListPublishedSkills 运行时已发布技能（skill.Registry DB provider 数据源）
	ListPublishedSkills(ctx context.Context) ([]*skill.Skill, error)
}

type skillPluginAppImpl struct {
	base.AppImpl[*entity.Skill, repository.Skill]

	resourceRepo repository.SkillResource `inject:"T"`
}

var _ SkillPlugin = (*skillPluginAppImpl)(nil)

// SkillSaveReq 技能创建/更新请求
type SkillSaveReq struct {
	// Code 技能唯一标识（编辑态不可变）
	Code string `json:"code"`
	// Description 技能描述
	Description string `json:"description"`
	// AllowedTools 预批准工具列表（空格分隔）
	AllowedTools string `json:"allowedTools"`
	// Instructions SKILL.md 正文
	Instructions string `json:"instructions"`
}

// ============== 元数据管理 ==============

func (a *skillPluginAppImpl) ListSkills(ctx context.Context) ([]*entity.Skill, error) {
	return a.ListByCond(model.NewCond().OrderByAsc("code"))
}

func (a *skillPluginAppImpl) GetInstructions(ctx context.Context, id uint64) (string, error) {
	res, err := a.resourceRepo.SelectBySkillIdAndPath(ctx, id, entity.SkillMdPath)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil // 无 SKILL.md 视为空正文（对齐 tokhub read_instructions 降级）
		}
		return "", err // 非“不存在”类错误（连接/权限等）必须上抛，避免误导前端覆盖真文
	}
	return res.Content, nil
}

func (a *skillPluginAppImpl) CreateSkill(ctx context.Context, req *SkillSaveReq) (*entity.Skill, error) {
	code, err := normalizeSkillCode(req.Code)
	if err != nil {
		return nil, err
	}
	if a.CountByCond(&entity.Skill{Code: code}) > 0 {
		return nil, errorx.NewBizf("skill code already exists: %s", code)
	}
	skillRow := &entity.Skill{
		Code:         code,
		Name:         code,
		Description:  req.Description,
		AllowedTools: req.AllowedTools,
		Version:      defaultSemver(),
		Source:       entity.SkillSourceCustom,
		Status:       entity.SkillStatusDraft,
	}
	if err := a.Insert(ctx, skillRow); err != nil {
		return nil, err
	}
	if err := a.writeInstructions(ctx, skillRow, req.Instructions); err != nil {
		return nil, err
	}
	return skillRow, nil
}

func (a *skillPluginAppImpl) UpdateSkill(ctx context.Context, id uint64, req *SkillSaveReq) error {
	skillRow, err := a.GetById(id)
	if err != nil {
		return err
	}
	updateRow := &entity.Skill{Description: req.Description, AllowedTools: req.AllowedTools}
	updateRow.Id = skillRow.Id
	if err := a.GetRepo().UpdateById(ctx, updateRow, "description", "allowed_tools"); err != nil {
		return err
	}
	return a.writeInstructions(ctx, skillRow, req.Instructions)
}

func (a *skillPluginAppImpl) DeleteSkill(ctx context.Context, id uint64) error {
	if err := a.DeleteById(ctx, id); err != nil {
		return err
	}
	return a.resourceRepo.DeleteBySkillId(ctx, id)
}

func (a *skillPluginAppImpl) Publish(ctx context.Context, id uint64) error {
	return a.updateStatus(ctx, id, entity.SkillStatusPublished)
}

func (a *skillPluginAppImpl) Unpublish(ctx context.Context, id uint64) error {
	return a.updateStatus(ctx, id, entity.SkillStatusDraft)
}

func (a *skillPluginAppImpl) updateStatus(ctx context.Context, id uint64, status string) error {
	return a.GetRepo().UpdateByCond(ctx,
		map[string]any{"status": status},
		model.NewCond().Eq("id", id))
}

// writeInstructions 写入 SKILL.md（存在则覆盖）：
// 纯正文（无 frontmatter）按当前元数据补全，保证导出 roundtrip 结构一致
func (a *skillPluginAppImpl) writeInstructions(ctx context.Context, skillRow *entity.Skill, instructions string) error {
	content := instructions
	if !strings.HasPrefix(strings.TrimSpace(content), "---") {
		content = skill.BuildSkillMd(skillRow.Name, skillRow.Description, skillRow.AllowedTools, instructions)
	}
	return a.upsertResourceRow(ctx, skillRow.Id, entity.SkillMdPath, content)
}

// ============== 资源管理（对齐 resource.rs）==============

func (a *skillPluginAppImpl) ListResources(ctx context.Context, skillId uint64) ([]*entity.SkillResource, error) {
	resources, err := a.resourceRepo.SelectBySkillId(ctx, skillId)
	if err != nil {
		return nil, err
	}
	// 排除 SKILL.md 主文件（正文走 instructions 接口，对齐 tokhub list_resources）
	result := make([]*entity.SkillResource, 0, len(resources))
	for _, r := range resources {
		if r.Path == entity.SkillMdPath {
			continue
		}
		result = append(result, r)
	}
	return result, nil
}

func (a *skillPluginAppImpl) UpsertResource(ctx context.Context, skillId uint64, resPath, content string) (*entity.SkillResource, error) {
	if err := validateResourcePath(resPath); err != nil {
		return nil, err
	}
	if _, err := a.GetById(skillId); err != nil {
		return nil, err
	}
	if err := a.upsertResourceRow(ctx, skillId, resPath, content); err != nil {
		return nil, err
	}
	return a.resourceRepo.SelectBySkillIdAndPath(ctx, skillId, resPath)
}

func (a *skillPluginAppImpl) DeleteResource(ctx context.Context, skillId uint64, resPath string) error {
	if resPath == entity.SkillMdPath {
		return errorx.NewBiz("SKILL.md is the skill main file, use skill update API")
	}
	if err := validateResourcePath(resPath); err != nil {
		return err
	}
	return a.resourceRepo.DeleteByCond(ctx, &entity.SkillResource{SkillId: skillId, Path: resPath})
}

// upsertResourceRow 按 (skill_id, path) 新增或覆盖资源
func (a *skillPluginAppImpl) upsertResourceRow(ctx context.Context, skillId uint64, resPath, content string) error {
	existing, err := a.resourceRepo.SelectBySkillIdAndPath(ctx, skillId, resPath)
	if err != nil {
		// 不存在则新增
		return a.resourceRepo.Insert(ctx, &entity.SkillResource{
			SkillId: skillId,
			Path:    resPath,
			Content: content,
			Size:    len(content),
		})
	}
	existing.Content = content
	existing.Size = len(content)
	return a.resourceRepo.UpdateById(ctx, existing, "content", "size")
}

// ============== zip 导入导出（对齐 zip.rs）==============

// ImportZip 从 zip 导入技能：
//   - 根目录或单层子目录下包含 SKILL.md
//   - SKILL.md frontmatter name 作为技能 code
//   - 其余文件作为资源（路径去除顶层目录前缀）
//   - 同 code 重复导入时 upsert 覆盖 + SemVer patch 递增
func (a *skillPluginAppImpl) ImportZip(ctx context.Context, data []byte) (*entity.Skill, error) {
	if len(data) == 0 || len(data) > maxZipBytes {
		return nil, errorx.NewBiz("invalid zip size")
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, errorx.NewBizf("invalid zip: %v", err)
	}
	skillMd, resources, err := extractZipContents(archive)
	if err != nil {
		return nil, err
	}

	fmName, fmDesc, _ := skill.ParseFrontmatter(skillMd)
	code, err := normalizeSkillCode(fmName)
	if err != nil {
		return nil, errorx.NewBizf("invalid frontmatter name: %v", err)
	}

	existing, err := a.ListByCond(&entity.Skill{Code: code})
	if err != nil {
		return nil, err
	}

	var skillRow *entity.Skill
	if len(existing) > 0 {
		// 已存在：覆盖更新 + patch 递增（upsert 语义，不报"编号已存在"）
		old := existing[0]
		version, err := bumpSemver(old.Version)
		if err != nil {
			return nil, err
		}
		updateRow := &entity.Skill{
			Name:        code,
			Description: fmDesc,
			Version:     version,
			Source:      entity.SkillSourceImported,
		}
		updateRow.Id = old.Id
		if err := a.GetRepo().UpdateById(ctx, updateRow, "name", "description", "version", "source"); err != nil {
			return nil, err
		}
		skillRow = old
		skillRow.Version = version
	} else {
		skillRow = &entity.Skill{
			Code:        code,
			Name:        code,
			Description: fmDesc,
			Version:     defaultSemver(),
			Source:      entity.SkillSourceImported,
			Status:      entity.SkillStatusDraft,
		}
		if err := a.Insert(ctx, skillRow); err != nil {
			return nil, err
		}
	}

	// 全量重建：先清空旧资源再写入（导入为覆盖语义）
	if err := a.resourceRepo.DeleteBySkillId(ctx, skillRow.Id); err != nil {
		return nil, err
	}
	if err := a.upsertResourceRow(ctx, skillRow.Id, entity.SkillMdPath, skillMd); err != nil {
		return nil, err
	}
	for _, res := range resources {
		if res.path == entity.SkillMdPath {
			continue
		}
		if err := validateResourcePath(res.path); err != nil {
			continue // 跳过非法路径（对齐 tokhub insert_resources）
		}
		if err := a.upsertResourceRow(ctx, skillRow.Id, res.path, res.content); err != nil {
			return nil, err
		}
	}
	return skillRow, nil
}

func (a *skillPluginAppImpl) ExportZip(ctx context.Context, id uint64) (string, []byte, error) {
	skillRow, err := a.GetById(id)
	if err != nil {
		return "", nil, err
	}
	resources, err := a.resourceRepo.SelectBySkillId(ctx, skillRow.Id)
	if err != nil {
		return "", nil, err
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, res := range resources {
		w, err := zw.Create(fmt.Sprintf("%s/%s", skillRow.Code, res.Path))
		if err != nil {
			return "", nil, err
		}
		if _, err := w.Write([]byte(res.Content)); err != nil {
			return "", nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("%s.zip", skillRow.Code), buf.Bytes(), nil
}

// ============== 运行时注入 ==============

// ListPublishedSkills 已发布技能转运行时结构（skill.Registry DB provider 数据源）：
// SKILL.md 解析出 body 作为 L3 全文，目录层仅用 code/name/description
func (a *skillPluginAppImpl) ListPublishedSkills(ctx context.Context) ([]*skill.Skill, error) {
	skills, err := a.ListByCond(model.NewCond().Eq("status", entity.SkillStatusPublished).OrderByAsc("code"))
	if err != nil {
		return nil, err
	}
	result := make([]*skill.Skill, 0, len(skills))
	for _, s := range skills {
		res, err := a.resourceRepo.SelectBySkillIdAndPath(ctx, s.Id, entity.SkillMdPath)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue // 无 SKILL.md 的技能跳过（不注入）
			}
			return nil, err // 真实 DB 错误上抛，由 Registry 层退回缓存兜底
		}
		_, _, body := skill.ParseFrontmatter(res.Content)
		if body == "" {
			body = res.Content
		}
		result = append(result, &skill.Skill{
			Code:        s.Code,
			Name:        s.Name,
			Description: s.Description,
			Content:     body,
		})
	}
	return result, nil
}

// ============== 内部辅助 ==============

// zipResource zip 内资源文件
type zipResource struct {
	path    string
	content string
}

// extractZipContents 定位 SKILL.md 并收集资源（对齐 zip.rs extract_zip_contents）：
// SKILL.md 位于根目录或单层子目录，资源路径已去除顶层目录前缀；
// 跳过目录、隐藏段（.开头）与超大文件
func extractZipContents(archive *zip.Reader) (string, []zipResource, error) {
	contents := make(map[string]string)
	var order []string
	for _, file := range archive.File {
		if file.FileInfo().IsDir() || file.UncompressedSize64 > uint64(maxZipResourceSize) {
			continue
		}
		name := file.Name
		if strings.Contains(name, "\\") || strings.Contains(name, "..") {
			continue
		}
		hidden := false
		for _, seg := range strings.Split(name, "/") {
			if strings.HasPrefix(seg, ".") {
				hidden = true
				break
			}
		}
		if hidden {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return "", nil, errorx.NewBizf("open zip entry failed: %v", err)
		}
		content, err := readZipContent(rc)
		rc.Close()
		if err != nil {
			return "", nil, err
		}
		contents[name] = content
		order = append(order, name)
	}

	// 定位 SKILL.md（根目录优先，其次单层子目录）
	prefix := ""
	skillMd, ok := contents[entity.SkillMdPath]
	if !ok {
		for _, name := range order {
			if top, rest, _ := strings.Cut(name, "/"); rest == entity.SkillMdPath {
				prefix = top + "/"
				skillMd = contents[name]
				break
			}
		}
	}
	if skillMd == "" {
		return "", nil, errorx.NewBiz("ZIP does not contain SKILL.md")
	}

	resources := make([]zipResource, 0, len(contents))
	for _, name := range order {
		rel := strings.TrimPrefix(name, prefix)
		if rel == "" || rel == name || rel == entity.SkillMdPath {
			continue
		}
		resources = append(resources, zipResource{path: rel, content: contents[name]})
	}
	return skillMd, resources, nil
}

// readZipContent 读取 zip entry 内容（限制大小，防解压炸弹）
func readZipContent(rc io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(rc, maxZipResourceSize+1))
	if err != nil {
		return "", errorx.NewBizf("read zip entry failed: %v", err)
	}
	if len(data) > maxZipResourceSize {
		return "", errorx.NewBiz("zip entry too large")
	}
	return string(data), nil
}

// validateResourcePath 资源路径校验：禁空、禁绝对路径、禁反斜杠、禁 .. 逃逸
func validateResourcePath(resPath string) error {
	resPath = strings.TrimSpace(resPath)
	if resPath == "" {
		return errorx.NewBiz("resource path is required")
	}
	if strings.HasPrefix(resPath, "/") || strings.Contains(resPath, "\\") {
		return errorx.NewBizf("invalid resource path: %s", resPath)
	}
	if path.Clean(resPath) != resPath {
		return errorx.NewBizf("invalid resource path: %s", resPath)
	}
	return nil
}

// normalizeSkillCode 规范化并校验 code
func normalizeSkillCode(code string) (string, error) {
	code = strings.TrimSpace(code)
	if !skillCodeRegexp.MatchString(code) {
		return "", errorx.NewBizf("invalid skill code: %s (allowed: letters, digits, '-', '_')", code)
	}
	return code, nil
}

// defaultSemver 新建技能默认版本号
func defaultSemver() string { return "1.0.0" }

// bumpSemver SemVer patch 递增（对齐 tokhub semver::bump_version("patch")）
func bumpSemver(version string) (string, error) {
	var major, minor, patch int
	if _, err := fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &patch); err != nil {
		return "", errorx.NewBizf("invalid semver format: '%s', expected 'MAJOR.MINOR.PATCH'", version)
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch+1), nil
}
