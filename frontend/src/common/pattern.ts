import { i18n } from '@/i18n';

export const AccountUsernamePattern = {
    pattern: /^[a-zA-Z0-9_]{5,16}$/g,
    message: i18n.global.t('system.account.usernamePatternErrMsg'),
};

export const ResourceCodePattern = {
    pattern: /^[a-zA-Z0-9_\-.:]{1,32}$/g,
    message: i18n.global.t('system.menu.resourceCodePatternErrMsg'),
};
