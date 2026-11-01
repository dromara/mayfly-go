/**
 * 诊断面板各页签的取数样板（loading / error / 结果三件套）。
 *
 * 面板有六个页签，每个都要「防重入 + 清错 + 取数 + 落错 + 关 loading」。
 * 这段样板抄六遍的代价不是行数，而是**改动会漏抄**：新增页签时忘了清 error，
 * 上一页签的报错就会挂在新页签上，看起来像这次查询失败了。
 */

/** 统一的错误文案提取：Error 用 message，其余 String()，避免把 [object Object] 显示给用户 */
export function toErrorMessage(error: unknown): string {
    return error instanceof Error ? error.message : String(error);
}

export interface PanelLoaderOptions<R> {
    /** 是否正在取数（用于防重入） */
    isLoading: () => boolean;
    setLoading: (loading: boolean) => void;
    /** 清空/写入错误文案 */
    setError: (message: string) => void;
    fetch: () => Promise<R>;
    onSuccess: (result: R) => void;
    /** 自定义失败处理（如会话页签要区分「能力缺失」与真实错误）；不传则把原因写进 error */
    onError?: (error: unknown) => void;
}

/** 生成一个可取数的函数：同一时刻只跑一次，失败不清空既有数据的判据交给调用方 */
export function createPanelLoader<R>(options: PanelLoaderOptions<R>): () => Promise<void> {
    return async () => {
        if (options.isLoading()) {
            return;
        }
        options.setLoading(true);
        options.setError('');
        try {
            options.onSuccess(await options.fetch());
        } catch (error) {
            if (options.onError) {
                options.onError(error);
            } else {
                options.setError(toErrorMessage(error));
            }
        } finally {
            options.setLoading(false);
        }
    };
}
