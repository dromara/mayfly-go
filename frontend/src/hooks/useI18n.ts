import { i18n } from '@/i18n';
import { ElMessage, ElMessageBox } from 'element-plus';

/**
 *  rule message 提示输入字段名
 * @param label 字段名称key
 * @returns
 */
export function useI18nPleaseInput(labelI18nKey: string) {
    const t = i18n.global.t;
    return t('common.pleaseInput', { label: t(labelI18nKey) });
}

/**
 *  rule message 提示选择字段名
 * @param label 字段名称key
 * @returns
 */
export function useI18nPleaseSelect(labelI18nKey: string) {
    const t = i18n.global.t;
    return t('common.pleaseSelect', { label: t(labelI18nKey) });
}

/**
 * 提示确认删除
 * @param name 删除对象名称
 * @returns 用户是否确认
 */
export async function useI18nDeleteConfirm(name: string = ''): Promise<boolean> {
    return useI18nConfirm('common.deleteConfirm2', { name });
}

/**
 * 弹确认框，**用返回值表达用户的选择，不抛异常**
 * @param i18nKey i18n msg key
 * @param value i18n msg value
 * @returns true = 点了确认；false = 取消或关掉弹窗（都代表「别执行后续动作」）
 *
 * 「取消」是正常交互结果，不是失败。用 reject 表达它，等于把 try/catch 强制分发给每一个调用点：
 * 全仓几十处确认里多数是裸 await，用户点一下就留一条未捕获异常，个别调用点还会因此跳过后面的
 * 清理逻辑。收敛成布尔值后，调用点只需 `if (!(await useI18nConfirm(...))) return;`。
 *
 * 需要区分「确认 / 取消按钮 / 关掉弹窗」三种结果时（如策略提醒要分别对应执行、提单、暂缓），
 * 直接按 element-plus 的 reject 值判定，别把这个原语改回抛异常
 */
export async function useI18nConfirm(i18nKey: string = '', value = {}): Promise<boolean> {
    const t = i18n.global.t;
    try {
        await ElMessageBox.confirm(t(i18nKey, value), t('common.hint'), {
            confirmButtonText: t('common.confirm'),
            cancelButtonText: t('common.cancel'),
            type: 'warning',
        });
        return true;
    } catch {
        // element-plus 的取消与关闭都走 reject，且 reject 值形态不一（'cancel' / 'close' / Error），
        // 二者对本原语是同一件事：不继续
        return false;
    }
}

/**
 * 表单校验
 * @param formRef 表单ref（支持 useTemplateRef / ref 获取的 el-form 实例）
 * @returns
 */
export async function useI18nFormValidate(formRef: { value: { validate: (...args: any[]) => any } | null | undefined }) {
    const t = i18n.global.t;

    try {
        if (!formRef.value) return false;
        await formRef.value.validate();
        return true;
    } catch (e: unknown) {
        ElMessage.error(t('common.formValidationError'));
        throw e;
    }
}

export function useI18nCreateTitle(i18nKey: string) {
    const t = i18n.global.t;
    return t('common.createTitle', { name: t(i18nKey) });
}

export function useI18nEditTitle(i18nKey: string) {
    const t = i18n.global.t;
    return t('common.editTitle', { name: t(i18nKey) });
}

export function useI18nDetailTitle(i18nKey: string) {
    const t = i18n.global.t;
    return t('common.detailTitle', { name: t(i18nKey) });
}

/**
 * 国际化消息提示（基于 ElMessage）
 */
export const Msg = {
    /**
     * 成功消息
     * @param msg 消息内容（支持 i18n key）
     * @param params 国际化参数
     */
    success(msg: string, params?: Record<string, unknown>) {
        ElMessage.success(i18n.global.t(msg, params));
    },

    /**
     * 错误消息
     * @param msg 消息内容（支持 i18n key）
     * @param params 国际化参数
     */
    error(msg: string, params?: Record<string, unknown>) {
        ElMessage.error(i18n.global.t(msg, params));
    },

    /**
     * 警告消息
     * @param msg 消息内容（支持 i18n key）
     * @param params 国际化参数
     */
    warning(msg: string, params?: Record<string, unknown>) {
        ElMessage.warning(i18n.global.t(msg, params));
    },

    /**
     * 信息消息
     * @param msg 消息内容（支持 i18n key）
     * @param params 国际化参数
     */
    info(msg: string, params?: Record<string, unknown>) {
        ElMessage.info(i18n.global.t(msg, params));
    },

    /**
     * 保存成功消息
     */
    saveSuccess() {
        Msg.success('common.saveSuccess');
    },

    /**
     * 删除成功消息
     */
    deleteSuccess() {
        Msg.success('common.deleteSuccess');
    },

    /**
     * 操作成功消息
     */
    operateSuccess() {
        Msg.success('common.operateSuccess');
    },
};
