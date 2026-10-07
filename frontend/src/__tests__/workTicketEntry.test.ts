/**
 * 「提交工单」入口的判定：只有需要审批的失败才给入口。
 *
 * 「需提交工单审批」与「已被管理员禁止执行」在界面上同为红色失败提示，
 * 但前者提单后会执行、后者提单也不会执行。判错一次就等于把用户引向一条走不通的路，
 * 所以入口必须认后端错误码，而不是认提示文案
 */
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import { NEED_WORK_TICKET_CODE, isNeedWorkTicketError, newBizFailureError } from '@/common/request';

/** 请求层在业务失败时抛出的错误形态：直接用真实工厂，漏挂 code 就会被测出来 */
function bizError(code: number, msg = '该操作需要提交工单审批执行') {
    return newBizFailureError(msg, code);
}

describe('提单入口按错误码判定', () => {
    it('需提单的错误码给出入口', () => {
        expect(isNeedWorkTicketError(bizError(NEED_WORK_TICKET_CODE))).toBe(true);
    });

    it('其它业务失败不给入口', () => {
        // 400 普通业务失败、501 无权限、以及「禁止执行」都走默认码：提单不会执行，入口是误导
        for (const code of [400, 405, 500, 501, 502]) {
            expect(isNeedWorkTicketError(bizError(code))).toBe(false);
        }
    });

    it('没有码或不是错误对象时判假', () => {
        for (const value of [undefined, null, 'boom', {}, new Error('普通错误')]) {
            expect(isNeedWorkTicketError(value)).toBe(false);
        }
    });

    it('业务失败错误必须带上响应码', () => {
        // 提单入口依赖这个码：请求层漏挂时判断恒假，入口静默消失
        expect(newBizFailureError('boom', 4001).code).toBe(4001);
    });

    it('码与后端约定一致', () => {
        // 后端 flow.application.CodeNeedWorkTicket = 4001（int16 范围内且不与 200/400/405/500/501/502 冲突）
        expect(NEED_WORK_TICKET_CODE).toBe(4001);
    });

    it('改文案不影响判定', () => {
        // 文案换成英文或调整措辞后入口必须照旧出现：这正是当初按字符串匹配的失效方式
        expect(isNeedWorkTicketError(bizError(NEED_WORK_TICKET_CODE, 'this operation needs approval via a work ticket'))).toBe(true);
    });
});

// 请求层要真的用该工厂抛错，否则调用方永远拿不到码。
// execCustomFetch 依赖真实 fetch，单元里不便 mock，这里按源码接线断言（同 layering.test.ts 的做法）
describe('请求层的接线', () => {
    it('业务失败必须以带码的错误抛出', () => {
        const source = readFileSync(join(import.meta.dirname, '../hooks/useRequest.ts'), 'utf-8');
        expect(source).toMatch(/newBizFailureError\(errMsg, resultCode\)/);
    });
});
