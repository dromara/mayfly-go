import { EnumValue } from '@/common/Enum';

// 授权凭证类型
export const AuthCertTypeEnum = {
    Public: EnumValue.of(2, '公共凭证').tagTypeSuccess().tagTypeSuccess(),
    Private: EnumValue.of(1, '普通账号'),
    Privileged: EnumValue.of(11, '特权账号').tagTypeDanger(),
    PrivateDefault: EnumValue.of(12, '默认账号').tagTypeWarning(),
};

// 授权凭证密文类型
export const AuthCertCiphertextTypeEnum = {
    Password: EnumValue.of(1, '密码').tagTypeSuccess(),
    PrivateKey: EnumValue.of(2, '秘钥').tagTypeSuccess(),
    Public: EnumValue.of(-1, '公共凭证').tagTypeSuccess(),
};

export const TagTreeRelateTypeEnum = {
    Team: EnumValue.of(1, '团队'),
};
