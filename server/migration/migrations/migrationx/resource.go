package migrationx

import (
	"time"

	"gorm.io/gorm"
)

// ResourceDef 系统资源定义（菜单 / 按钮权限通用）。
// 字段必须导出：各版本文件以位置初始化（{id, pid, uiPath, ...}）声明大段资源数据，
// 位初始化对非导出字段是编译错误（implicit assignment to unexported field）
type ResourceDef struct {
	Id     int64
	Pid    int64
	UiPath string
	Typ    int8
	Name   string
	Code   string
	Weight int
	Meta   string
}

// Resource 按位置参数构造资源定义，供各批次的菜单 / 按钮权限 seed 表使用。
//
// 不直接让各批次写 struct 字面量：跳包位置初始化会被 go vet 判为 unkeyed fields，
// 更实际的风险是字段顺序一旦调整，name / code 这类同类型字段会静默错位写库；
// 走函数签名后参数错位直接编译失败
func Resource(id, pid int64, uiPath string, typ int8, name, code string, weight int, meta string) ResourceDef {
	return ResourceDef{Id: id, Pid: pid, UiPath: uiPath, Typ: typ, Name: name, Code: code, Weight: weight, Meta: meta}
}

// InsertResource 按 id 幂等写入单条系统资源：不存在则插入，未删除则跳过，软删则复活并刷新字段。
//
// 存在性判定必须含软删行：t_sys_resource 的主键唯一性不含 is_deleted，只按「未删除」判存在，
// 会在「用户曾删掉该菜单」的存量库上走 INSERT 分支撞 1062，让整条迁移链在该版本永久卡住——
// 全新库跑不出来，只在老库炸（只能靠真实升级路径复现）
func InsertResource(tx *gorm.DB, r ResourceDef) error {
	var deletedFlags []int8
	if err := tx.Table("t_sys_resource").Where("id = ?", r.Id).Limit(1).Pluck("is_deleted", &deletedFlags).Error; err != nil {
		return err
	}
	now := time.Now()
	switch {
	case len(deletedFlags) == 0:
		return tx.Exec(
			"INSERT INTO t_sys_resource (id, pid, ui_path, type, status, name, code, weight, meta, creator_id, creator, modifier_id, modifier, create_time, update_time, is_deleted, delete_time) "+
				"VALUES (?, ?, ?, ?, 1, ?, ?, ?, ?, 1, 'admin', 1, 'admin', ?, ?, 0, NULL)",
			r.Id, r.Pid, r.UiPath, r.Typ, r.Name, r.Code, r.Weight, r.Meta, now, now,
		).Error
	case deletedFlags[0] == 0:
		return nil // 已存在且未删除：幂等跳过
	}
	// 软删行：复活并刷新为本次声明的形态。直接跳过会让该资源升级后仍然不可见（菜单被静默吞掉）
	return tx.Table("t_sys_resource").Where("id = ?", r.Id).UpdateColumns(map[string]any{
		"pid": r.Pid, "ui_path": r.UiPath, "type": r.Typ, "status": 1,
		"name": r.Name, "code": r.Code, "weight": r.Weight, "meta": r.Meta,
		"is_deleted": 0, "delete_time": nil,
		"modifier_id": 1, "modifier": "admin", "update_time": now,
	}).Error
}

// InsertResourcesWithRole 批量插入资源并授予 COMMON 角色（role_id=1）
func InsertResourcesWithRole(tx *gorm.DB, resources []ResourceDef) error {
	for _, r := range resources {
		if err := InsertResource(tx, r); err != nil {
			return err
		}
		if err := GrantRoleResource(tx, 1, r.Id); err != nil {
			return err
		}
	}
	return nil
}

// GrantRoleResource 按 (role_id, resource_id) 幂等授予角色资源权限。
//
// 同样含软删行判定：已有软删授权时复活它，而不是再插一条——同一授权留多行会让
// 「按行判定是否已授权」的后续逻辑口径不定
func GrantRoleResource(tx *gorm.DB, roleId, resourceId int64) error {
	var deletedFlags []int8
	if err := tx.Table("t_sys_role_resource").
		Where("role_id = ? AND resource_id = ?", roleId, resourceId).Limit(1).Pluck("is_deleted", &deletedFlags).Error; err != nil {
		return err
	}
	if len(deletedFlags) == 0 {
		return tx.Exec(
			"INSERT INTO t_sys_role_resource (role_id, resource_id, creator_id, creator, create_time, is_deleted, delete_time) VALUES (?, ?, 1, 'admin', ?, 0, NULL)",
			roleId, resourceId, time.Now(),
		).Error
	}
	if deletedFlags[0] == 0 {
		return nil // 已授权且未删除：幂等跳过
	}
	// 只写该表确实存在的列（本表无 modifier/update_time，多写会让迁移直接报未知列）
	return tx.Table("t_sys_role_resource").Where("role_id = ? AND resource_id = ?", roleId, resourceId).
		UpdateColumns(map[string]any{"is_deleted": 0, "delete_time": nil}).Error
}

// InsertResourceWithRole 插入单条资源并授予 COMMON 角色。
//
// 用命名字段而不是位置初始化：本函数签名里 id/pid/weight/typ 全是整数、
// name/code/meta 全是字符串，参数顺序写错没有任何编译期或运行期信号（只会把名字写进 code 列）
func InsertResourceWithRole(tx *gorm.DB, id, pid int64, uiPath string, typ int8, name, code string, weight int, meta string) error {
	return InsertResourcesWithRole(tx, []ResourceDef{{
		Id: id, Pid: pid, UiPath: uiPath, Typ: typ, Name: name, Code: code, Weight: weight, Meta: meta,
	}})
}

// GrantResourceToParentRoles 把新资源授予所有已拥有指定父资源的角色。
//
// InsertResourcesWithRole 只会授予 COMMON 角色（role_id=1）；后续给已有菜单补基础权限码时，
// 自定义角色如果只靠它就会缺失新码，表现为升级后入口凭空消失。所以按父菜单反查已授权角色逐个补授
func GrantResourceToParentRoles(tx *gorm.DB, parentResourceId, resourceId int64) error {
	var roleIds []int64
	if err := tx.Table("t_sys_role_resource").
		Where("resource_id = ? AND is_deleted = 0", parentResourceId).
		Pluck("role_id", &roleIds).Error; err != nil {
		return err
	}
	for _, roleId := range roleIds {
		if err := GrantRoleResource(tx, roleId, resourceId); err != nil {
			return err
		}
	}
	return nil
}
