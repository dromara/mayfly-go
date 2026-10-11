import { describe, expect, it } from 'vitest';

import { buildTransferFileDownloadUrl } from '../api';

/**
 * 导出文件下载地址的契约测试。
 *
 * 钉住的是「数据类产物必须走本模块带权限码的下载入口」这条收口：一旦有人把它改回通用文件路由
 * （/sys/files/{key}），私有文件会被服务端拒读，界面上表现为下载无响应，而这条测试会先红。
 */
describe('buildTransferFileDownloadUrl', () => {
    const url = buildTransferFileDownloadUrl(123);

    it('走迁移模块的鉴权下载入口，而不是通用文件路由', () => {
        expect(url).toContain('/dbTransfer/files/content/123');
        expect(url).not.toContain('/sys/files/');
    });

    it('携带凭据查询参数（浏览器直接导航无法附带 Authorization 头）', () => {
        expect(url).toContain('token=');
        expect(url).toContain('clientId=');
    });
});
