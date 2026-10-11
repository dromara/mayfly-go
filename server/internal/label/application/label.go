package application

import (
	"context"
	"mayfly-go/internal/label/domain/entity"
	"mayfly-go/internal/label/domain/repository"
	"mayfly-go/internal/label/imsg"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
)

// Label 标签应用接口
type Label interface {
	base.App[*entity.Label]

	// SaveLabel 保存标签（新增或更新）
	SaveLabel(ctx context.Context, label *entity.Label) error

	// DeleteLabel 删除标签（级联删除绑定关系）
	DeleteLabel(ctx context.Context, id uint64) error

	// GetLabelList 分页查询标签
	GetLabelList(condition *entity.LabelQuery) (*model.PageResult[*entity.Label], error)

	// ListKeys 获取所有标签键
	ListKeys(ctx context.Context) ([]string, error)

	// ListValuesByKey 获取指定 key 的所有值
	ListValuesByKey(ctx context.Context, labelKey string) ([]string, error)

	// GetAutocomplete 自动补全数据
	GetAutocomplete(labelKey string) ([]LabelAutocompleteItem, error)
}

// LabelValueDetail 标签值明细
//
// 颜色与描述归属于「key + value」这一具体标签（t_label 的一行），而非 key 本身。
// 同一 key 下的不同 value 允许各有颜色与描述，因此自动补全必须把值级别的元信息一并下发，
// 否则前端只能取 key 级别的首个非空值，导致同键不同值的标签渲染成同一个颜色
type LabelValueDetail struct {
	Value       string `json:"value"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

// LabelAutocompleteItem 自动补全项
type LabelAutocompleteItem struct {
	Key         string   `json:"key"`
	Values      []string `json:"values"`
	Description string   `json:"description"`
	Color       string   `json:"color"`

	// ValueDetails 值级别明细，与 Values 同序
	ValueDetails []LabelValueDetail `json:"valueDetails"`
}

type labelAppImpl struct {
	base.AppImpl[*entity.Label, repository.Label]

	labelBindingRepo repository.LabelBinding `inject:"T"`
}

var _ Label = (*labelAppImpl)(nil)

func (a *labelAppImpl) SaveLabel(ctx context.Context, label *entity.Label) error {
	// 1. 参数校验
	if label.LabelKey == "" {
		return errorx.NewBizI(ctx, imsg.ErrLabelKeyRequired)
	}
	if label.LabelValue == "" {
		return errorx.NewBizI(ctx, imsg.ErrLabelValueRequired)
	}

	// 2. 编辑时禁止修改 key 和 value（只允许修改颜色/描述）
	if label.Id > 0 {
		original, err := a.GetById(label.Id)
		if err != nil {
			return err
		}
		if original != nil && (original.LabelKey != label.LabelKey || original.LabelValue != label.LabelValue) {
			return errorx.NewBizI(ctx, imsg.ErrLabelKeyValueChanged)
		}
	}

	// 3. 重复检查（同 key+value 不允许重复）
	//
	// 查询必须含已逻辑删除的行：唯一索引对墓碑行同样生效，只看未删除数据会让校验通过、
	// 随后 INSERT 撞 idx_label_kv 变成一条数据库错误
	existing, err := a.GetRepo().GetByKeyIncludingDeleted(label.LabelKey, label.LabelValue)
	if err != nil {
		return err
	}
	if existing != nil && existing.Id != label.Id {
		if existing.IsDeleted == model.ModelUndeleted {
			return errorx.NewBizI(ctx, imsg.ErrLabelDuplicate, "key", label.LabelKey, "value", label.LabelValue)
		}
		// 命中墓碑行：恢复它并沿用其主键，而不是报错或插新行。
		// 用户视角下「重新添加刚删掉的标签」就是把它恢复回来，同时保留 id 与创建时间等历史
		if err := a.RestoreByCond(ctx, model.NewCond().IncludeDeleted().Eq(model.IdColumn, existing.Id)); err != nil {
			return err
		}
		label.Id = existing.Id
	}

	// 4. 持久化
	return a.Save(ctx, label)
}

func (a *labelAppImpl) DeleteLabel(ctx context.Context, id uint64) error {
	label, err := a.GetById(id)
	if err != nil {
		return err
	}
	if label == nil {
		return nil
	}
	// 引用校验：标签仍被资源绑定时不允许删除。
	// 告警等实体的冗余 labels JSON 字段与 t_label_binding 需保持一致，若直接级联删绑定会残留
	// 实体 JSON 中的"幽灵标签"（告警引擎按 JSON 匹配仍会命中），故在源头阻止删除。
	//
	// 统计失败时一律不得放行删除：把 SQL 错误当成「0 条绑定」会删掉仍在被使用的标签，
	// 留下指向已删标签的孤儿绑定，而这种脏数据只能等告警引擎误匹配时才被发现
	bindCount, err := a.labelBindingRepo.CountByLabelId(ctx, id)
	if err != nil {
		return err
	}
	if bindCount > 0 {
		return errorx.NewBizI(ctx, imsg.ErrLabelInUse, "key", label.LabelKey, "value", label.LabelValue, "count", bindCount)
	}
	return a.DeleteById(ctx, id)
}

func (a *labelAppImpl) GetLabelList(condition *entity.LabelQuery) (*model.PageResult[*entity.Label], error) {
	qd := model.NewCond().
		Like("label_key", condition.LabelKey)

	return a.PageByCond(qd, condition.PageParam)
}

func (a *labelAppImpl) ListKeys(ctx context.Context) ([]string, error) {
	return a.GetRepo().ListKeys(ctx)
}

func (a *labelAppImpl) ListValuesByKey(ctx context.Context, labelKey string) ([]string, error) {
	return a.GetRepo().ListValuesByKey(ctx, labelKey)
}

func (a *labelAppImpl) GetAutocomplete(labelKey string) ([]LabelAutocompleteItem, error) {
	// 单次查询所有 key+value 对，在内存中聚合，避免 N+1
	allLabels, err := a.GetRepo().SelectByCond(model.NewCond())
	if err != nil {
		return nil, err
	}

	type keyInfo struct {
		values       []string
		description  string
		color        string
		valueDetails []LabelValueDetail
	}
	keyMap := make(map[string]*keyInfo)
	keyOrder := make([]string, 0)
	for _, l := range allLabels {
		if labelKey != "" && l.LabelKey != labelKey {
			continue
		}
		info, ok := keyMap[l.LabelKey]
		if !ok {
			info = &keyInfo{}
			keyMap[l.LabelKey] = info
			keyOrder = append(keyOrder, l.LabelKey)
		}
		valueColor := l.GetExtraString(entity.LabelExtraKeyColor)
		valueDescription := l.GetExtraString(entity.LabelExtraKeyDescription)
		info.values = append(info.values, l.LabelValue)
		info.valueDetails = append(info.valueDetails, LabelValueDetail{
			Value:       l.LabelValue,
			Description: valueDescription,
			Color:       valueColor,
		})
		// key 级别取首个非空描述和颜色，供只按 key 着色的场景兜底
		if info.description == "" {
			info.description = valueDescription
		}
		if info.color == "" {
			info.color = valueColor
		}
	}

	items := make([]LabelAutocompleteItem, 0, len(keyOrder))
	for _, key := range keyOrder {
		info := keyMap[key]
		items = append(items, LabelAutocompleteItem{
			Key:          key,
			Values:       info.values,
			Description:  info.description,
			Color:        info.color,
			ValueDetails: info.valueDetails,
		})
	}
	return items, nil
}

// LabelBinding 标签绑定应用接口
type LabelBinding interface {
	base.App[*entity.LabelBinding]

	// SaveBindings 批量保存绑定
	SaveBindings(ctx context.Context, bindings []*entity.LabelBinding) error

	// SaveBindingsByLabels 根据 key-value 对保存绑定（自动查找或创建标签）
	SaveBindingsByLabels(ctx context.Context, targetType string, targetId uint64, labels map[string]string) error

	// DeleteByTarget 删除目标的所有绑定
	DeleteByTarget(ctx context.Context, targetType string, targetId uint64) error

	// DeleteBinding 删除单个绑定
	DeleteBinding(ctx context.Context, labelId uint64, targetType string, targetId uint64) error

	// ListByTarget 获取目标的绑定标签（返回 key+value）
	ListByTarget(ctx context.Context, targetType string, targetId uint64) ([]LabelBindingVO, error)

	// ListByTargets 批量获取多个目标的绑定标签（一次查询，消除 N+1）
	ListByTargets(ctx context.Context, targetType string, targetIds []uint64) (map[uint64][]LabelBindingVO, error)
}

// LabelBindingVO 绑定标签视图对象
type LabelBindingVO struct {
	LabelId    uint64 `json:"labelId"`
	LabelKey   string `json:"labelKey"`
	LabelValue string `json:"labelValue"`
}

type labelBindingAppImpl struct {
	base.AppImpl[*entity.LabelBinding, repository.LabelBinding]

	labelRepo repository.Label `inject:"T"`
}

var _ LabelBinding = (*labelBindingAppImpl)(nil)

func (a *labelBindingAppImpl) SaveBindings(ctx context.Context, bindings []*entity.LabelBinding) error {
	if len(bindings) == 0 {
		return nil
	}

	targetType := bindings[0].TargetType
	targetId := bindings[0].TargetId

	// 校验批次一致性（P1 修复）
	for i := 1; i < len(bindings); i++ {
		if bindings[i].TargetType != targetType || bindings[i].TargetId != targetId {
			return errorx.NewBiz("all bindings must share the same targetType and targetId")
		}
	}

	// 查询当前已有的绑定
	oldBindings, err := a.GetRepo().ListByTarget(targetType, targetId)
	if err != nil {
		return err
	}

	// 提取 label_id 用于比较
	oldLabelIds := collx.ArrayMap(oldBindings, func(b *entity.LabelBinding) uint64 {
		return b.LabelId
	})
	newLabelIds := collx.ArrayMap(bindings, func(b *entity.LabelBinding) uint64 {
		return b.LabelId
	})

	// 差异比较：新增、删除、不变
	addedLabelIds, deletedLabelIds, _ := collx.ArrayCompare(newLabelIds, oldLabelIds)

	if len(addedLabelIds) == 0 && len(deletedLabelIds) == 0 {
		return nil // 无变化
	}

	// 在事务中执行增删
	//
	// 墓碑绑定必须在事务**之前**读：基座读入口尚未接 ctx（这是未完成的那部分改造），它走连接池
	// 而不是事务所在的那条连接 —— 事务内的读既看不到本事务刚写入的绑定，也会在事务占满连接池时
	// 等不到空闲连接（默认方言 SQLite 是文件库，把 max-open-conns 调小后就是长时间不返回）。
	// 先读与在事务内读结果等价：删除集与新增集来自 ArrayCompare 的两侧，互不相交，本事务内的
	// 删除不会新增出属于新增集的墓碑
	var tombstones []uint64
	if len(addedLabelIds) > 0 {
		tombstones, err = a.GetRepo().ListDeletedLabelIds(ctx, targetType, targetId, addedLabelIds)
		if err != nil {
			return err
		}
	}

	return a.Tx(ctx, func(ctx context.Context) error {
		// 删除不再需要的绑定
		if len(deletedLabelIds) > 0 {
			if err := a.GetRepo().DeleteByCond(ctx, model.NewCond().
				Eq("target_type", targetType).
				Eq("target_id", targetId).
				In("label_id", deletedLabelIds)); err != nil {
				return err
			}
		}
		return nil
	}, func(ctx context.Context) error {
		if len(addedLabelIds) == 0 {
			return nil
		}
		// 先恢复该目标下的墓碑绑定，只为真正没有记录的标签插入新行：idx_lb_unique 唯一索引覆盖
		// 已软删的行，直接插入会让「解绑再绑同一标签」变成数据库唯一键错误
		if len(tombstones) > 0 {
			if err := a.RestoreByCond(ctx, model.NewCond().IncludeDeleted().
				Eq("target_type", targetType).
				Eq("target_id", targetId).
				In("label_id", tombstones)); err != nil {
				return err
			}
		}
		tombstoneSet := collx.ArrayToMap(tombstones, func(id uint64) uint64 { return id })

		addedBindings := make([]*entity.LabelBinding, 0, len(addedLabelIds))
		for _, b := range bindings {
			if _, isTombstone := tombstoneSet[b.LabelId]; !isTombstone {
				addedBindings = append(addedBindings, b)
			}
		}
		if len(addedBindings) == 0 {
			return nil
		}
		return a.BatchInsert(ctx, addedBindings)
	})
}

// SaveBindingsByLabels 根据 key-value 对保存绑定（自动查找或创建标签）
func (a *labelBindingAppImpl) SaveBindingsByLabels(ctx context.Context, targetType string, targetId uint64, labels map[string]string) error {
	if len(labels) == 0 {
		// 清空绑定
		return a.GetRepo().DeleteByCond(ctx, model.NewCond().Eq("target_type", targetType).Eq("target_id", targetId))
	}

	// 收集所有 key-value 对
	type kv struct {
		key   string
		value string
	}
	var kvs []kv
	for k, v := range labels {
		kvs = append(kvs, kv{key: k, value: v})
	}

	// 批量查找或创建标签
	bindings := make([]*entity.LabelBinding, 0, len(kvs))
	for _, kv := range kvs {
		label, err := a.getOrCreateLabelIncludingTombstone(ctx, kv.key, kv.value)
		if err != nil {
			return err
		}
		bindings = append(bindings, &entity.LabelBinding{
			LabelId:    label.Id,
			TargetType: targetType,
			TargetId:   targetId,
		})
	}

	return a.SaveBindings(ctx, bindings)
}

// DeleteByTarget 删除目标的所有绑定
func (a *labelBindingAppImpl) DeleteByTarget(ctx context.Context, targetType string, targetId uint64) error {
	return a.GetRepo().DeleteByCond(ctx, model.NewCond().Eq("target_type", targetType).Eq("target_id", targetId))
}

func (a *labelBindingAppImpl) DeleteBinding(ctx context.Context, labelId uint64, targetType string, targetId uint64) error {
	return a.GetRepo().DeleteByCond(ctx, model.NewCond().Eq("label_id", labelId).Eq("target_type", targetType).Eq("target_id", targetId))
}

func (a *labelBindingAppImpl) ListByTarget(ctx context.Context, targetType string, targetId uint64) ([]LabelBindingVO, error) {
	bindings, err := a.GetRepo().ListByTarget(targetType, targetId)
	if err != nil {
		return nil, err
	}

	if len(bindings) == 0 {
		return []LabelBindingVO{}, nil
	}

	// 批量查询标签信息，避免逐个 GetById 的 N+1 问题
	labelIds := make([]uint64, 0, len(bindings))
	for _, b := range bindings {
		labelIds = append(labelIds, b.LabelId)
	}
	labels, err := a.labelRepo.GetByIds(labelIds)
	if err != nil {
		return nil, err
	}
	labelMap := make(map[uint64]*entity.Label, len(labels))
	for _, l := range labels {
		labelMap[l.Id] = l
	}

	result := make([]LabelBindingVO, 0, len(bindings))
	for _, b := range bindings {
		label, ok := labelMap[b.LabelId]
		if !ok {
			continue
		}
		result = append(result, LabelBindingVO{
			LabelId:    b.LabelId,
			LabelKey:   label.LabelKey,
			LabelValue: label.LabelValue,
		})
	}
	return result, nil
}

// ListByTargets 批量查询多个目标的绑定标签（一次绑定查询 + 一次标签查询）
func (a *labelBindingAppImpl) ListByTargets(ctx context.Context, targetType string, targetIds []uint64) (map[uint64][]LabelBindingVO, error) {
	result := make(map[uint64][]LabelBindingVO, len(targetIds))
	if len(targetIds) == 0 {
		return result, nil
	}

	// 一次查询所有绑定
	allBindings, err := a.GetRepo().ListByTargetIds(targetType, targetIds)
	if err != nil {
		return nil, err
	}
	if len(allBindings) == 0 {
		return result, nil
	}

	// 一次查询所有关联的标签
	labelIdSet := make(map[uint64]struct{})
	for _, b := range allBindings {
		labelIdSet[b.LabelId] = struct{}{}
	}
	labelIds := make([]uint64, 0, len(labelIdSet))
	for id := range labelIdSet {
		labelIds = append(labelIds, id)
	}
	labels, err := a.labelRepo.GetByIds(labelIds)
	if err != nil {
		return nil, err
	}
	labelMap := make(map[uint64]*entity.Label, len(labels))
	for _, l := range labels {
		labelMap[l.Id] = l
	}

	// 按 targetId 分组
	for _, b := range allBindings {
		label, ok := labelMap[b.LabelId]
		if !ok {
			continue
		}
		result[b.TargetId] = append(result[b.TargetId], LabelBindingVO{
			LabelId:    b.LabelId,
			LabelKey:   label.LabelKey,
			LabelValue: label.LabelValue,
		})
	}
	return result, nil
}

// getOrCreateLabelIncludingTombstone 按 key+value 取标签，缺失则创建，命中墓碑则恢复后复用其主键。
//
// 查询必须含已逻辑删除的行：idx_label_kv 唯一索引对墓碑行同样生效，只看未删除数据会让这里
// 判断成「不存在」，随后的 INSERT 撞数据库唯一键，而回退查询如果用同样的口径依然查不到东西，
// 最终只能把数据库错误原样抛成一条 500 —— 「删掉标签 env=prod，再在告警规则里填 env=prod」
// 就是这条路径。恢复墓碑而不是报错，也符合用户视角：重新添加刚删掉的标签就是把它取回来
func (a *labelBindingAppImpl) getOrCreateLabelIncludingTombstone(ctx context.Context, labelKey, labelValue string) (*entity.Label, error) {
	existing, err := a.labelRepo.GetByKeyIncludingDeleted(labelKey, labelValue)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.IsDeleted != model.ModelDeleted {
			return existing, nil
		}
		rows, err := a.labelRepo.RestoreByCond(ctx, model.NewCond().IncludeDeleted().Eq(model.IdColumn, existing.Id))
		if err != nil {
			return nil, err
		}
		if rows == 0 {
			// 刚读到墓碑、恢复却没命中：并发下已被他人改动，不能拿一个不确定状态的标签去绑定
			return nil, errorx.NewBizI(ctx, imsg.ErrLabelDuplicate, "key", labelKey, "value", labelValue)
		}
		existing.IsDeleted = model.ModelUndeleted
		return existing, nil
	}

	label := &entity.Label{LabelKey: labelKey, LabelValue: labelValue}
	if err := a.labelRepo.Insert(ctx, label); err != nil {
		// 并发下另一个请求可能刚插入同一标签：按同一口径（含墓碑）再查一次，查到就复用
		concurrent, queryErr := a.labelRepo.GetByKeyIncludingDeleted(labelKey, labelValue)
		if queryErr != nil || concurrent == nil {
			return nil, err
		}
		return concurrent, nil
	}
	return label, nil
}
