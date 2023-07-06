import { EnumValue } from '@/common/Enum';

// 脚本执行结果类型
export const ScriptResultEnum = {
    Result: EnumValue.of(1, '有结果'),
    NoResult: EnumValue.of(2, '无结果'),
    RealTime: EnumValue.of(3, '实时交互'),
};

// 脚本类型
export const ScriptTypeEnum = {
    Private: EnumValue.of(1, '私有'),
    Public: EnumValue.of(2, '公共'),
};

// 文件类型枚举
export const FileTypeEnum = {
    Directory: EnumValue.of(1, '目录'),
    File: EnumValue.of(2, '文件'),
};

// 授权凭证认证方式枚举
export const AuthMethodEnum = {
    Password: EnumValue.of(1, '密码').tagTypeSuccess(),
    PrivateKey: EnumValue.of(2, '秘钥'),
};
