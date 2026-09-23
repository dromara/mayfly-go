import { EnumValue } from '@/common/Enum';

export const DbTransferRunningStateEnum = {
    Success: EnumValue.of(2, 'common.success').setTagType('success'),
    Running: EnumValue.of(1, 'db.running').setTagType('primary'),
    Fail: EnumValue.of(-1, 'common.fail').setTagType('danger'),
    Stop: EnumValue.of(-2, 'db.stop').setTagType('warning'),
};

export const DbTransferFileStatusEnum = {
    Running: EnumValue.of(1, 'db.running').setTagType('primary'),
    Success: EnumValue.of(2, 'common.success').setTagType('success'),
    Fail: EnumValue.of(-1, 'common.fail').setTagType('danger'),
};

export const DbTransferLogStatusEnum = {
    Running: EnumValue.of(2, 'db.running').setTagType('primary'),
    Success: EnumValue.of(1, 'common.success').setTagType('success'),
    Fail: EnumValue.of(-1, 'common.fail').setTagType('danger'),
};

// 执行用途：区分同一任务下的迁移 / 导出文件 / 数据校验记录
export const DbTransferLogPurposeEnum = {
    Transfer: EnumValue.of(1, 'db.transferPurposeTransfer').setTagType('primary'),
    Export: EnumValue.of(2, 'db.transferPurposeExport').setTagType('success'),
    Verify: EnumValue.of(3, 'db.transferPurposeVerify').setTagType('info'),
};
