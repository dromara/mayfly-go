/**
 * 按 language 全局唯一的补全 provider 注册表
 *
 * monaco 的 `registerCompletionItemProvider` 是全局生效的（同一 language 后注册者会被一并返回），
 * 所以这里的注册互斥：同 language 再注册会先 dispose 掉前一个。写入方必须成对申领/释放，
 * 目前唯一的使用方是 db/completion（经 completion/lazy.ts 动态引入，见 monacoBoundary 守卫）。
 *
 * 不要给编辑器弹窗开「注入自定义补全」的口子：那会让弹窗在用户看不见的地方顶掉 db 的 SQL 联想，
 * 而弹窗自己既不感知也不释放。真要按实例隔离补全，应绑到编辑器的 model 上而非这张全局表。
 */
import { languages } from './setup';

/**
 * key: language, value: CompletionItemProvider
 */
const completionItemProviders: Map<string, { dispose: () => void }> = new Map();

export function registerCompletionItemProvider(language: string, completionItemProvider: any, replace: boolean = true) {
    const exist = completionItemProviders.get(language);
    if (exist) {
        if (!replace) {
            return;
        }
        exist.dispose();
    }
    completionItemProviders.set(language, languages.registerCompletionItemProvider(language, completionItemProvider));
}

export function disposeCompletionItemProvider(language: string) {
    const exist = completionItemProviders.get(language);
    if (exist) {
        exist.dispose();
        completionItemProviders.delete(language);
    }
}
