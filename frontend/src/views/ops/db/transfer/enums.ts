import { EnumValue } from '@/common/Enum';

export const DbTransferRunningStateEnum = {
    // 任务创建后从未跑过时，后端该字段为零值，不能归到成功/失败里展示
    NotRun: EnumValue.of(0, 'db.notRun').setTagType('info'),
    Success: EnumValue.of(2, 'common.success').setTagType('success'),
    Running: EnumValue.of(1, 'db.running').setTagType('primary'),
    Fail: EnumValue.of(-1, 'common.fail').setTagType('danger'),
    Stop: EnumValue.of(-2, 'db.stop').setTagType('warning'),
};

// 迁移方式：决定「目标」是数据库还是 SQL 文件，列表里据此分派目标列的渲染
export const DbTransferModeEnum = {
    Db: EnumValue.of(1, 'db.transfer2Db').setTagType('primary'),
    File: EnumValue.of(2, 'db.transfer2File').setTagType('success'),
};

// 迁移策略
export const DbTransferStrategyEnum = {
    Full: EnumValue.of(1, 'db.transferFull').setTagType('primary'),
    Increment: EnumValue.of(2, 'db.transferIncrement').tagTypeInfo(),
};

// 建表前是否删除同名目标表：「是」会连带清空目标表数据，故用 danger 醒目标注
export const DbTransferDeleteTableEnum = {
    Yes: EnumValue.of(1, 'common.yes').setTagType('danger'),
    No: EnumValue.of(2, 'common.no').setTagType('info'),
};

// 目标表名/字段名大小写转换
export const DbTransferNameCaseEnum = {
    None: EnumValue.of(1, 'db.none').setTagType('info'),
    Upper: EnumValue.of(2, 'db.upper').setTagType('warning'),
    Lower: EnumValue.of(3, 'db.lower').setTagType('warning'),
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
