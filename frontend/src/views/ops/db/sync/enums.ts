import { EnumValue } from '@/common/Enum';

export const DbDataSyncDuplicateStrategyEnum = {
    // None(-1) 语义是"该模式不做冲突处理"（如增量追加/校验模式），显示"不适用"而非"无"，避免用户以为配错了
    None: EnumValue.of(-1, 'db.notApplicable'),
    Ignore: EnumValue.of(1, 'db.ignore'),
    Replace: EnumValue.of(2, 'db.replace'),
};

export const DbDataSyncModeEnum = {
    IncrementalAppend: EnumValue.of(1, 'db.syncModeIncrementalAppend'),
    IncrementalMerge: EnumValue.of(2, 'db.syncModeIncrementalMerge'),
    FullRefresh: EnumValue.of(3, 'db.syncModeFullRefresh'),
    IncrementalSoftDel: EnumValue.of(4, 'db.syncModeIncrementalSoftDel'),
    IncrementalHardDel: EnumValue.of(5, 'db.syncModeIncrementalHardDel'),
    Validation: EnumValue.of(6, 'db.syncModeValidation'),
};

export const DbDataSyncRecentStateEnum = {
    // 任务创建后从未跑完过时，后端该字段为零值，不能归到成功/失败里展示
    NotRun: EnumValue.of(0, 'db.notRun').setTagType('info'),
    Success: EnumValue.of(1, 'common.success').setTagType('success'),
    Fail: EnumValue.of(-1, 'common.fail').setTagType('danger'),
};

export const DbDataSyncLogStatusEnum = {
    Success: EnumValue.of(1, 'common.success').setTagType('success'),
    Running: EnumValue.of(2, 'db.running').setTagType('primary'),
    Fail: EnumValue.of(-1, 'common.fail').setTagType('danger'),
};

export const DbDataSyncRunningStateEnum = {
    Running: EnumValue.of(1, 'db.running').setTagType('success'),
    WaitRun: EnumValue.of(2, 'db.waitRun').setTagType('primary'),
    Stop: EnumValue.of(3, 'db.stop').setTagType('danger'),
};

export const DbNullStrategyEnum = {
    Pass: EnumValue.of(0, 'db.nullStrategyPass'),
    Default: EnumValue.of(1, 'db.nullStrategyDefault'),
    SkipRow: EnumValue.of(2, 'db.nullStrategySkipRow'),
};

export const DbSchemaEvolveModeEnum = {
    Off: EnumValue.of(0, 'db.schemaEvolveOff'),
    Warn: EnumValue.of(1, 'db.schemaEvolveWarn'),
    Auto: EnumValue.of(2, 'db.schemaEvolveAuto'),
};

export const DbConflictStrategyEnum = {
    SourceWins: EnumValue.of(1, 'db.conflictSourceWins'),
    TargetWins: EnumValue.of(2, 'db.conflictTargetWins'),
    Skip: EnumValue.of(3, 'db.conflictSkip'),
};

// 与 entity.CursorInclusivity 对齐（后端 Auto=0/Exclusive=1/Inclusive=2），
// 实际存于 Extra（非查询维度不占列）。
export const DbCursorInclusivityEnum = {
    Auto: EnumValue.of(0, 'db.cursorInclusivityAuto'),
    Exclusive: EnumValue.of(1, 'db.cursorInclusivityExclusive'),
    Inclusive: EnumValue.of(2, 'db.cursorInclusivityInclusive'),
};
