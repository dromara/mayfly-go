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
 * @returns
 */
export async function useI18nDeleteConfirm(name: string = '') {
    return useI18nConfirm('common.deleteConfirm2', { name });
}

/**
 * 提示确认信息
 * @param i18nKey i18n msg key
 * @param value i18n msg value
 * @returns
 */
export async function useI18nConfirm(i18nKey: string = '', value = {}) {
    const t = i18n.global.t;
    return ElMessageBox.confirm(t(i18nKey, value), t('common.hint'), {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning',
    });
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
