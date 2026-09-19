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
    Fail: EnumValue.of(0, 'common.fail').setTagType('danger'),
};
