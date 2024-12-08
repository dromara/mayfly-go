import type { App } from 'vue';
import { authDirective } from './auth';
import { wavesDirective } from './waves';

// 导出指令方法
export function directive(app: App) {
    // 用户权限指令
    authDirective(app);
    // 按钮波浪指令
    wavesDirective(app);
}
