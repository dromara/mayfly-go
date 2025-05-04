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
 * @param formRef 表单ref
 * @param callback 校验通过回调
 * @returns
 */
export async function useI18nFormValidate(formRef: any) {
    const t = i18n.global.t;

    try {
        await formRef.value.validate();
        return true;
    } catch (e: any) {
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

export function useI18nOperateSuccessMsg() {
    MsgSuccess('common.operateSuccess');
}

export function useI18nSaveSuccessMsg() {
    MsgSuccess('common.saveSuccess');
}

export function useI18nDeleteSuccessMsg() {
    MsgSuccess('common.deleteSuccess');
}

/**
 * error msg
 * @param msg msg(支持i8n msgkey)
 */
export function MsgError(msg: string) {
    ElMessage.error(i18n.global.t(msg));
}

/**
 * success msg
 * @param msg msg(支持i8n msgkey)
 */
export function MsgSuccess(msg: string) {
    ElMessage.success(i18n.global.t(msg));
}
