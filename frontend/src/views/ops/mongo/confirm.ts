/**
 * 破坏性结构操作的用户侧确认。
 *
 * 与后端的 confirmName 校验配对：服务端负责真正拦住不一致，
 * 前端负责让「点错行」在提交前就说不通，而不是等一次 400。
 */
import { ElMessageBox } from 'element-plus';

import { i18n } from '@/i18n';

/**
 * 要求把目标名原样输入后才能继续。
 *
 * 删集合、删库会连带销毁索引与集合选项且不可逆，一次「确定删除?」气泡拦不住误点；
 * 返回 false 表示用户取消，调用方据此中止操作。
 */
export async function confirmByName(targetName: string): Promise<boolean> {
    const t = i18n.global.t;
    try {
        await ElMessageBox.prompt(t('mongo.confirmByNameTip', { name: targetName }), t('mongo.confirmByNameTitle'), {
            confirmButtonText: t('common.confirm'),
            cancelButtonText: t('common.cancel'),
            type: 'warning',
            inputValidator: (value: string) => (value === targetName ? true : t('mongo.confirmByNameMismatch', { name: targetName })),
        });
        return true;
    } catch {
        // 用户取消或关闭弹窗
        return false;
    }
}
