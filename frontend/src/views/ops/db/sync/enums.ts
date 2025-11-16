import { EnumValue } from '@/common/Enum';

export const DbDataSyncDuplicateStrategyEnum = {
    None: EnumValue.of(-1, 'db.none'),
    Ignore: EnumValue.of(1, 'db.ignore'),
    Replace: EnumValue.of(2, 'db.replace'),
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
