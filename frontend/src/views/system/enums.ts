import { EnumValue } from '@/common/Enum';

export const ResourceTypeEnum = {
    Menu: EnumValue.of(1, 'system.menu.menu').tagTypeSuccess(),
    Permission: EnumValue.of(2, 'system.menu.permission'),
};

export const AccountStatusEnum = {
    Enable: EnumValue.of(1, 'system.account.statusEnable').tagTypeSuccess(),
    Disable: EnumValue.of(-1, 'system.account.statusDisable').tagTypeDanger(),
};

export const RoleStatusEnum = {
    Enable: EnumValue.of(1, 'system.role.statusEnable').tagTypeSuccess(),
    Disable: EnumValue.of(-1, 'system.role.statusDisable').tagTypeDanger(),
};

export const LogTypeEnum = {
    Success: EnumValue.of(1, 'system.syslog.resultSuccess').tagTypeSuccess(),
    Error: EnumValue.of(2, 'system.syslog.resultFail').tagTypeDanger(),
    Running: EnumValue.of(-1, 'system.syslog.resultRunning'),
};
