/**
 * webhook 推送地址的唯一拼装点（外部监控 Prometheus/Zabbix 会直接粘贴使用）。
 *
 * 为什么单独成函数：`baseApiUrl` 既可能是**绝对地址**（本仓默认形态 `protocol//host:port/api`），
 * 也可能被部署配置成相对前缀（`/api`，同源反代场景）。旧写法 `${window.location.origin}${baseUrl}`
 * 无条件前置 origin，在默认形态下会拼出 `http://hosthttp://host/api/...`——
 * 复制出来的地址粘进外部监控直接不可用，而且它不会报错，只在真的配监控那天才暴露
 * （本轮由「回读剪贴板」发现；此前只断言「复制成功 toast 出现」，等于没验）。
 *
 * 用 `new URL(相对或绝对, 基准)` 一次覆盖两种形态：第一段是绝对地址时基准被忽略，
 * 是相对前缀时按同源补全。尾部多余斜杠先剥掉，避免拼出 `//agentops`。
 */
export function buildWebhookUrl(baseApiUrl: string, token: string, origin: string): string {
    const base = baseApiUrl.replace(/\/+$/, '');
    return new URL(`${base}/agentops/triggers/webhook/${token}`, origin).href;
}
