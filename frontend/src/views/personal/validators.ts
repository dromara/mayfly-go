import { Rules } from '@/common/rule';

/** 修改密码表单的判定结果：ok=可提交，strength=不符合口令强度，mismatch=两次输入不一致 */
export type PasswordChangeResult = 'ok' | 'strength' | 'mismatch';

export interface PasswordChangeForm {
    password: string;
    confirmPassword: string;
}

/**
 * 修改密码的提交前判定。
 *
 * 不能只依赖 el-form 的字段校验：Element Plus 的 validate 在 validator 抛异常时会静默放行
 * （内部把异常当成空错误集合，最终 resolve true），此时不合法口令会被直接提交给后端。
 * 这里用与表单规则同一份正则再判一次，保证提交路径始终有拦截
 */
export function validatePasswordChange(form: PasswordChangeForm): PasswordChangeResult {
    if (!Rules.accountPassword.pattern.test(form.password)) {
        return 'strength';
    }
    if (form.password !== form.confirmPassword) {
        return 'mismatch';
    }
    return 'ok';
}
