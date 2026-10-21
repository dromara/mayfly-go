import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

/**
 * 渠道凭据的读写接缝守卫。
 *
 * 存在理由（真机抓到）：`GET /msg/channels` 把整个 `extra` 原样回吐，而 `extra.secret` 是
 * 钉钉/飞书机器人的 HMAC 签名密钥——任何拿得到菜单权限的账号都能读到全部群的签名凭据。
 * 后端已改为「列表脱敏 + 受 msg:channel:save 把守的详情接口」，于是前端出现一条新的硬依赖：
 * **编辑回填必须走详情接口**。若有人把它改回直接用列表行对象，症状不是报错，而是用户编辑一次
 * 渠道就把密钥抹空（界面全程显示「保存成功」），此后该渠道发出的消息全部加签失败。
 *
 * 这类「静态门禁全绿、界面也看不出问题、只在下次真发消息时炸」的接线，只能钉住源码。
 */
const readSource = (path: string) => readFileSync(resolve(__dirname, '..', path), 'utf8');

const readI18n = (lang: string) => readFileSync(resolve(__dirname, '..', '..', '..', '..', 'i18n', lang, 'msg.ts'), 'utf8');

/**
 * 去掉注释后的源码。
 *
 * 不剔不行：守卫自己的注释里会引用被禁的旧写法作为反例（下面那条 not.toContain 就是这个情形），
 * 直接对整文件做包含判定会把注释里的引用当成残留缺陷，红得毫无意义
 */
const codeOnly = (source: string) => source.replace(/<!--[^]*?-->/g, '').replace(/\/\*[^]*?\*\//g, '').replace(/^\s*\/\/.*$/gm, '');

it('编辑回填必须走详情接口，而不是复用已脱敏的列表行', () => {
    const source = readSource('ChannelList.vue');
    const fn = source.match(/const editChannel = [\s\S]*?\n\};/);
    expect(fn, '未找到 editChannel（改名/删除必须同步本守卫）').toBeDefined();
    const body = fn![0];

    expect(body, '编辑必须按 id 拉详情回填：列表里的 secret 已被后端脱敏').toContain('channelApi.detail.request({ id: data.id })');
    // 直接赋行对象是本轮修掉的缺陷形态（Extra 为脱敏值 → 保存即抹密钥）
    expect(body, '不得再把列表行对象直接当回填数据').not.toMatch(/state\.editDialog\.data = data;/);
});

it('api 层必须声明那个受保存权限把守的详情端点', () => {
    const source = readSource('../api.ts');
    expect(source, '缺 /msg/channels/{id} 详情端点').toMatch(/detail: Api\.newGet<MsgChannel>\('\/msg\/channels\/\{id\}'\)/);
});

describe('加签密钥字段的文案与真源', () => {
    const source = codeOnly(readSource('ChannelDing.vue'));

    it('标签与必填提示共用同一个 i18n 键', () => {
        expect(source).toContain(`:label="t('msg.secret')"`);
        expect(source, '提示文案不得再传裸英文串（渲出来是「请输入secret」中英混排）').toContain(`Rules.requiredInput(t('msg.secret'))`);
        expect(source, "字典里没有 'Secret' 这个键，裸用等于界面直出英文单词").not.toContain("$t('Secret')");
        expect(source).not.toContain("requiredInput('secret')");
    });

    it('凭据输入框默认遮文', () => {
        expect(source, '签名字段不应常驻明文展示').toContain('show-password');
    });

    it('msg.secret 中英成对登记', () => {
        for (const lang of ['zh-cn', 'en']) {
            const dict = readI18n(lang);
            expect(dict, `${lang}/msg.ts 缺少 secret 键`).toContain('secret:');
        }
    });
});
