import { EnumValue } from '@/common/Enum';

export const ResourceTypeEnum = {
    Menu: EnumValue.of(1, '菜单'),
    Permission: EnumValue.of(2, '权限'),
};

export const AccountStatusEnum = {
    Enable: EnumValue.of(1, '正常').tagTypeSuccess(),
    Disable: EnumValue.of(-1, '禁用').tagTypeDanger(),
};

export const LogTypeEnum = {
    Success: EnumValue.of(1, '成功').tagTypeSuccess(),
    Error: EnumValue.of(2, '失败').tagTypeDanger(),
};
