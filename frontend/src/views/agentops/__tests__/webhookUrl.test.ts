import { describe, expect, it } from 'vitest';
import { buildWebhookUrl } from '../webhookUrl';

/**
 * webhook 推送地址拼装的回归锁。
 *
 * 缺陷形态（真机由回读剪贴板发现）：`baseApiUrl` 默认就是含 origin 的绝对地址，
 * 而旧写法又无条件前置了一次 `window.location.origin`，复制出来是
 * `http://localhost:8889http://localhost:8889/api/agentops/...` ——
 * 粘进 Prometheus/Zabbix 直接不可用，且界面上只有一句「复制成功」，看起来完全正常
 */
const ORIGIN = 'http://localhost:8889';
const TOKEN = '31be191d1add2641b47ae55e435f59db';

describe('buildWebhookUrl', () => {
    it('绝对 baseApiUrl（本仓默认形态）不得重复 origin', () => {
        expect(buildWebhookUrl('http://localhost:8889/api', TOKEN, ORIGIN)).toBe(
            `http://localhost:8889/api/agentops/triggers/webhook/${TOKEN}`,
        );
    });

    it('相对 baseApiUrl（同源反代）按 origin 补全', () => {
        expect(buildWebhookUrl('/api', TOKEN, ORIGIN)).toBe(`http://localhost:8889/api/agentops/triggers/webhook/${TOKEN}`);
    });

    it('baseApiUrl 尾部多余斜杠不产生双斜杠路径', () => {
        expect(buildWebhookUrl('http://localhost:8889/api/', TOKEN, ORIGIN)).toBe(
            `http://localhost:8889/api/agentops/triggers/webhook/${TOKEN}`,
        );
        expect(buildWebhookUrl('/api//', TOKEN, ORIGIN)).toBe(`http://localhost:8889/api/agentops/triggers/webhook/${TOKEN}`);
    });

    it('结果必须能被 URL 解析且 origin 只出现一次（旧缺陷的精确判据）', () => {
        for (const base of ['http://localhost:8889/api', '/api', 'http://localhost:18888/api']) {
            const url = buildWebhookUrl(base, TOKEN, ORIGIN);
            expect(url.match(/https?:\/\//g), `${base} 拼出的地址里 origin 重复了`).toHaveLength(1);
            expect(() => new URL(url)).not.toThrow();
        }
    });
});
