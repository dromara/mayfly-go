import { describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { WARN_ACK_CODE, isWarnAckError, newBizFailureError, NEED_WORK_TICKET_CODE } from '@/common/request';

vi.mock('element-plus', () => ({
    ElMessageBox: { confirm: vi.fn() },
}));

const { ElMessageBox } = await import('element-plus');
type MessageBoxData = Awaited<ReturnType<typeof ElMessageBox.confirm>>;
const { confirmWarnAck } = await import('@/views/flow/warnAck');

function read(...segments: string[]) {
    return readFileSync(join(process.cwd(), ...segments), 'utf8');
}

describe('提醒确认码与提单码互不混用', () => {
    it('4002 判为需确认，4001 不是', () => {
        expect(isWarnAckError(newBizFailureError('命中提醒', WARN_ACK_CODE))).toBe(true);
        expect(isWarnAckError(newBizFailureError('需审批', NEED_WORK_TICKET_CODE))).toBe(false);
        expect(WARN_ACK_CODE).not.toBe(NEED_WORK_TICKET_CODE);
    });

    it('没有码或不是错误对象时判假', () => {
        for (const value of [undefined, null, 'boom', {}, new Error('普通错误')]) {
            expect(isWarnAckError(value)).toBe(false);
        }
    });

    it('改文案不影响判定', () => {
        expect(isWarnAckError(newBizFailureError('the operation matched a warning-only rule', WARN_ACK_CODE))).toBe(true);
    });
});

describe('确认框的三态映射', () => {
    it('点「直接执行」返回 run', async () => {
        vi.mocked(ElMessageBox.confirm).mockResolvedValue('confirm' as MessageBoxData);
        expect(await confirmWarnAck('命中提醒')).toBe('run');
    });

    it('点「提交工单审批」返回 ticket，关掉弹窗返回 skip', async () => {
        // 两者的差别是「要不要给出提单入口」，判反会让人对着一条没执行的语句找不到提单按钮，
        // 或者关掉弹窗却冒出一个提单入口
        vi.mocked(ElMessageBox.confirm).mockRejectedValue('cancel');
        expect(await confirmWarnAck('命中提醒')).toBe('ticket');

        vi.mocked(ElMessageBox.confirm).mockRejectedValue('close');
        expect(await confirmWarnAck('命中提醒')).toBe('skip');
    });

    it('必须区分取消按钮与关闭弹窗', async () => {
        await confirmWarnAck('命中提醒');
        expect(vi.mocked(ElMessageBox.confirm).mock.calls[0][2]).toMatchObject({ distinguishCancelAndClose: true });
    });
});

describe('两个入口都接了同一套确认流', () => {
    const redisConsole = read('src', 'views', 'ops', 'redis', 'keyview', 'KeyConsole.vue');
    const sqlExec = read('src', 'views', 'ops', 'db', 'sql-editor', 'composables', 'useSqlExec.ts');
    const request = read('src', 'hooks', 'useRequest.ts');
    const resultTabs = read('src', 'views', 'ops', 'db', 'sql-editor', 'SqlExecResultTabs.vue');

    it('Redis 命令台按码分流并带确认重试', () => {
        expect(redisConsole).toContain('isWarnAckError');
        expect(redisConsole).toContain('confirmWarnAck');
        // 重试必须带确认位：不带就会原地再弹一次同样的确认框
        expect(redisConsole).toMatch(/runAndRecord\(line, args, \{ ackWarn: true/);
        // 历史条目只由一个函数落，避免「主分支补了标记、重试分支漏了」
        expect(redisConsole.match(/history\.value\.unshift/g)?.length).toBe(3);
        expect(redisConsole).toMatch(/needApproval: isNeedWorkTicketError\(error\)/);
    });

    it('批量选区明确声明「不能弹确认」，否则每条都会被追问而无人处理', () => {
        // 批量在客户端也是一条一条发请求的，服务端看不出「这是一批」，
        // 只能由调用方声明 ask=false；漏掉就会变成「提示写着可直接执行、却一条也没跑」
        expect(sqlExec).toMatch(/execSql\(dbName, sql, '', \{ ask: false \}\)/);
        // 单条执行则要确认
        expect(sqlExec).toMatch(/\{ ask: true, acknowledged \}/);
    });

    it('SQL 单条执行按 warnAck 位分流并带确认重试', () => {
        expect(sqlExec).toContain('colAndData.warnAck');
        expect(sqlExec).toContain('confirmWarnAck');
        expect(sqlExec).toContain('retry(true)');
        // 提单入口靠 needApprovalSqls 出现，确认框的 ticket 分支必须把语句交给它
        expect(sqlExec).toMatch(/needApprovalSqls = choice === 'ticket' \? \[sql\] : \[\]/);
    });

    it('未执行的语句按提醒语义呈现，不能渲染成红色的执行失败', () => {
        // 语句根本没跑，报「执行失败」会让人以为语法或权限出了错
        expect(sqlExec).toMatch(/if \(err\.warnAck\) \{[\s\S]{0,160}state\.execResTabs\[i\]\.notice = err\.msg/);
        expect(resultTabs).toContain('v-else-if="dt.notice"');
        expect(resultTabs).toContain('icon="warning"');
        // 但真正的失败仍要走红色分支
        expect(resultTabs).toContain(':title="$t(\'db.execFail\')"');
    });

    it('确认码不再叠一条红色报错', () => {
        // 这句话就是操作者要看的提示，弹两遍会让他以为执行失败了
        expect(request).toMatch(/if \(resultCode != WARN_ACK_CODE\) \{\n\s+Msg\.error\(result\.msg\)/);
        // 但提示文本必须照样带出去，确认框正文用的就是它
        expect(request).toMatch(/errMsg = result\.msg;/);
    });
});

const { execWithWarnAck } = await import('@/views/ops/redis/warnAckOps');

describe('面板类型化操作也参与提醒确认', () => {
    it('没有提单通道的入口：确认框取消按钮不再是「提交工单」，且取消与关闭都算先不执行', async () => {
        const confirm = ElMessageBox.confirm as ReturnType<typeof vi.fn>;
        for (const reason of ['cancel', 'close', new Error('cancel')]) {
            confirm.mockRejectedValueOnce(reason);
            expect(await confirmWarnAck('命中提醒', { ticket: false })).toBe('skip');
        }
        const options = confirm.mock.calls.at(-1)?.[2];
        expect(options.distinguishCancelAndClose).toBe(false);
        // 取消按钮文案不该再是提单：这个入口给不出提单表单
        expect(options.cancelButtonText).not.toBe('提交工单审批');
    });

    it('命中确认码时带确认位重试一次，选择先不执行则不再发第二次请求', async () => {
        const confirm = ElMessageBox.confirm as ReturnType<typeof vi.fn>;
        const calls: boolean[] = [];
        const run = async (ackWarn: boolean) => {
            calls.push(ackWarn);
            if (!ackWarn) {
                throw newBizFailureError('命中提醒', WARN_ACK_CODE);
            }
            return 'ok';
        };

        // 确认框 resolve 即代表点了「直接执行」，返回值本身对本用例无意义
        confirm.mockResolvedValueOnce({});
        expect(await execWithWarnAck(run)).toEqual({ executed: true, result: 'ok' });
        expect(calls).toEqual([false, true]);

        calls.length = 0;
        confirm.mockRejectedValueOnce('cancel');
        expect(await execWithWarnAck(run)).toEqual({ executed: false, ticket: false });
        expect(calls).toEqual([false]); // 先不执行就不能带确认位再发一次请求
    });

    it('入口可提单时，「提交工单审批」这个选择要能传回调用方', async () => {
        const confirm = ElMessageBox.confirm as ReturnType<typeof vi.fn>;
        const calls: boolean[] = [];
        const run = async (ackWarn: boolean) => {
            calls.push(ackWarn);
            throw ackWarn ? 'unexpected' : newBizFailureError('命中提醒', WARN_ACK_CODE);
        };

        // 默认（面板写操作）不给出提单按钮：取消与关闭都只是暂缓
        confirm.mockRejectedValueOnce('cancel');
        expect(await execWithWarnAck(run)).toEqual({ executed: false, ticket: false });

        // 读内容给得出「申请查看」，因此必须区分「去提单」与「先不动」
        confirm.mockRejectedValueOnce('cancel');
        expect(await execWithWarnAck(run, { ticket: true })).toEqual({ executed: false, ticket: true });
        expect(calls).toEqual([false, false]);
    });

    it('普通失败照常上抛，不会被误当成提醒', async () => {
        await expect(execWithWarnAck(async () => Promise.reject(newBizFailureError('连接失败', 400)))).rejects.toThrow();
    });

    it('每次面板写请求都带确认位（漏一处就等于那个入口仍在悄悄执行）', () => {
        const files = ['src/views/ops/redis/keyview/useKeyView.ts', 'src/views/ops/redis/KeyDetail.vue', 'src/views/ops/redis/resource/RedisDataOp.vue'];
        const writeOps = 'delKeys|setKeyTtl|renameKey|copyKey|putKeyValue|runKeyOp';
        let total = 0;
        for (const file of files) {
            const source = read(file);
            const lines = source.split('\n').filter((line) => new RegExp(`redisApi\\.(${writeOps})\\.request\\(`).test(line));
            total += lines.length;
            // 漏掉确认位就等于该入口仍在悄悄执行，报出文件名才找得到是哪一处
            expect(lines.filter((line) => !line.includes('ackWarn'))).toEqual([]);
        }
        // 4 个视角出口 + 3 个 key 弹层操作 + 2 个删除入口：数量变化说明有入口漏接或多加
        expect(total).toBe(9);
    });

    it('操作者选了先不执行时，不得再走成功提示与刷新', () => {
        const dialog = read('src/views/ops/redis/keyview/useKeyFormDialog.ts');
        // confirmApi 只有抛错才是取消，正常 return 会被宿主当成保存成功
        expect(dialog).toMatch(/if \(!\(await store\.saveMember\([\s\S]{0,120}?\)\) \{\s*\n\s*throw new Error/);
        const area = read('src/views/ops/redis/keyview/KeyMemberArea.vue');
        expect(area).toMatch(/if \(!\(await props\.store\.deleteMembers\(\[row\]\)\)\) \{/);
        expect(area).toMatch(/if \(!\(await props\.store\.saveMember\(/);
    });
});

describe('key 面板读内容也过策略，且被拦时给得出路', () => {
    const store = read('src/views/ops/redis/keyview/useKeyView.ts');
    const detail = read('src/views/ops/redis/KeyDetail.vue');

    it('读内容请求带确认位并走同一套确认流', () => {
        // 读请求必须把确认位带到后端：不带就退化成「提醒也不问」的静默读取
        // [^}]* 限定在同一个请求对象内：跨到别处的 ackWarn 不算数
        expect(store).toMatch(/redisApi\.keyValues\.request\(\{[^}]*ackWarn/);
        expect(store).toContain('execWithWarnAck((ackWarn) =>');
    });

    it('需审批时不给数据，并给出「申请查看」提单入口', () => {
        expect(store).toContain('isNeedWorkTicketError(error)');
        expect(store).toContain('contentLocked.value = true');
        // 内容被拦下时面板只给等价命令与提单入口，不能把 rows/total 照常渲染出去
        // 锁定面板与内容区必须互斥：只判「有锁定分支」测不出「锁了但照常渲染数据」
        const lockAt = detail.indexOf('store.contentLocked.value');
        const areaAt = detail.indexOf('<KeyMemberArea');
        expect(lockAt).toBeGreaterThan(-1);
        expect(areaAt).toBeGreaterThan(lockAt);
        expect(detail.slice(areaAt, areaAt + 80)).toContain('v-else');
        // 按钮与处理器必须在同一个元素上，否则「有文案没接线」这种断线测不出来
        expect(detail).toMatch(/@click="onRequestView"[^>]*>\s*\{\{\s*\$t\('redis\.applyToView'\)/);
        // 在确认框里选了「提交工单审批」就必须真的开抽屉，不能只把面板变灰
        expect(store).toContain('ticketRequested.value = true');
        // 只是「暂缓」时若面板没有内容，必须把锁定态与「申请查看」入口留着，不能留一片空白
        expect(store).toMatch(/if \(!members\.value\.length\) \{\s*\n\s*lockContent\(key\);/);
        expect(detail).toMatch(/store\.ticketRequested\.value/);
        expect(detail).toMatch(/store\.ticketRequested\.value[\s\S]{0,220}onRequestView\(\)/);
        // 与命令台同一条提单链路（同一个业务类型），否则两张单在流程侧不是同一回事
        expect(detail).toContain('FlowBizType.RedisRunWriteCmd.value');
        expect(detail).toContain('cmd: store.viewCommand.value');
    });

    it('判定命令名来自后端视角声明，前端不再猜一份', () => {
        expect(store).toContain('descriptor.value?.readCmd');
        expect(read('src/views/ops/redis/types.ts')).toContain('readCmd: string;');
    });
});

/**
 * 「清空整库」这类按钮的策略出口守卫。
 *
 * 按钮发出的是一条真实命令，被策略判「需审批」时给得出等价工单：少了提单宿主或漏判提单码，
 * 用户只会看到一句「请提交工单」却无处可提交，而这类缺失编译与类型检查都发现不了
 */
describe('数据操作页的策略出口', () => {
    const source = read('src/views/ops/redis/resource/RedisDataOp.vue');

    it('清空整库必须走提醒确认，并能就地转工单', () => {
        const flush = source.slice(source.indexOf('const onFlushDb'), source.indexOf('const onFlushDb') + 1200);
        expect(flush, 'FLUSHDB 未走确认流：命中「仅提醒」会直接执行').toContain('execWithWarnAck');
        expect(flush, 'FLUSHDB 请求未带确认位').toContain('ackWarn');
        expect(flush, 'FLUSHDB 被要求审批时没有提单出口').toContain('openFlushTicket(');
        expect(flush).toContain("'FLUSHDB'");
    });

    it('提单宿主与需审批分流码要成对存在', () => {
        expect(source).toContain('isNeedWorkTicketError');
        expect(source).toContain('<WorkTicketSubmit');
        expect(source).toContain('FlowBizType.RedisRunWriteCmd.value');
    });
});
