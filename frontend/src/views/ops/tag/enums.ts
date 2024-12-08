import { EnumValue } from '@/common/Enum';

// 授权凭证类型
export const AuthCertTypeEnum = {
    Public: EnumValue.of(2, 'ac.acTypeEnumPublic').tagTypeSuccess().tagTypeSuccess(),
    Private: EnumValue.of(1, 'ac.acTypeEnumPrivate'),
    Privileged: EnumValue.of(11, 'ac.acTypeEnumPrivileged').tagTypeDanger(),
    PrivateDefault: EnumValue.of(12, 'ac.acTypeEnumPrivateDefault').tagTypeWarning(),
};

// 授权凭证密文类型
export const AuthCertCiphertextTypeEnum = {
    Password: EnumValue.of(1, 'ac.ciphertextTypeEnumPassword').tagTypeSuccess(),
    PrivateKey: EnumValue.of(2, 'ac.ciphertextTypeEnumPrivateKey').tagTypeSuccess(),
    Public: EnumValue.of(-1, 'ac.ciphertextTypeEnumPublic').tagTypeSuccess(),
};

export const TagTreeRelateTypeEnum = {
    Team: EnumValue.of(1, '团队'),
};
