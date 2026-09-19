import { EnumValue } from '@/common/Enum';

export const DbDataSyncDuplicateStrategyEnum = {
    None: EnumValue.of(-1, 'db.none'),
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
