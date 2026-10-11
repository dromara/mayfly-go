import { reactive } from 'vue';
import { dbApi } from '../api';
import type { SqlRollbackConflict } from '../types';

/**
 * 回滚 SQL 弹窗的状态与取数。
 *
 * 回滚 SQL 由服务端生成而不是在前端拼：服务端手里有列类型、主键清单与旧值采集当时的上下文，
 * 而前端只拿到一份 JSON——联合主键、字符串里的引号、超过 2^53 的 BIGINT 主键、base64 之后
 * 的二进制列，在客户端每一项都只能猜。猜错了生成的语句照样能「执行成功」，只是把数据改坏了。
 *
 * 失败时不开弹窗：请求中心层已经把服务端的拒绝原因 toast 过（如「该表没有主键，无法定位行」），
 * 再弹一个空框会让人以为「回滚 SQL 就是空的」，而实际上一条都没生成
 */
export function useRollbackSqlDialog() {
    const dialog = reactive({
        visible: false,
        sql: '',
        /** 生成时的如实说明（如行数据采集被截断、未做并发校验），非空才展示告警条 */
        note: '',
        /** 本次生成的语句条数，不等于原执行的影响行数 */
        rows: 0,
        /** 未生成回滚的行（被并发修改、删除或无法定位）：只给条数用户没法对账，逐行带原因展示 */
        conflicts: [] as SqlRollbackConflict[],
    });

    async function show(execId: number) {
        dialog.sql = '';
        dialog.note = '';
        dialog.rows = 0;
        dialog.conflicts = [];
        try {
            const res = await dbApi.rollbackSql.request({ id: execId });
            if (!res) {
                return false;
            }
            dialog.sql = (res.sqls ?? []).join('\n');
            dialog.rows = res.rows ?? 0;
            dialog.note = res.note ?? '';
            dialog.conflicts = res.conflicts ?? [];
            dialog.visible = true;
            return true;
        } catch {
            return false;
        }
    }

    function close() {
        dialog.visible = false;
    }

    return { dialog, show, close };
}
