import { useI18nPleaseInput, useI18nPleaseSelect } from '@/hooks/useI18n';
import { i18n } from '@/i18n';

/**
 * 表单验证规则
 * label: 支持 i18n key
 */
export const Rules = {
    requiredInput: (label: string = '', trigger: string[] = ['change', 'blur']) => {
        return {
            required: true,
            message: useI18nPleaseInput(label),
            trigger: trigger,
        };
    },

    requiredSelect: (label: string = '', trigger: string[] = ['change', 'blur']) => {
        return {
            required: true,
            message: useI18nPleaseSelect(label),
            trigger: trigger,
        };
    },

    accountUsername: {
        pattern: /^[a-zA-Z0-9_]{5,16}$/g,
        message: i18n.global.t('system.account.usernamePatternErrMsg'),
        trigger: 'blur',
    },
    accountPassword: {
        pattern: /^(?=.*[A-Za-z])(?=.*\d)(?=.*[`~!@#$%^&*()_+<>?:"{},.\/\\;'[\]])[A-Za-z\d`~!@#$%^&*()_+<>?:"{},.\/\\;'[\]]{8,}$/,
        message: i18n.global.t('login.passwordRuleTip'),
        trigger: 'blur',
    },

    resourceCode: {
        pattern: /^[a-zA-Z0-9_\-.:]{1,32}$/g,
        message: i18n.global.t('system.menu.resourceCodePatternErrMsg'),
        trigger: 'blur',
    },
};
