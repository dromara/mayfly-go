/**
 * SQL 补全能力（db 模块级）
 *
 * 归属说明：本目录被 sql-editor、resource 查询页、sync 任务表单、审批流 SQL 表单四处共用，
 * 是 db 模块的公共能力而非某个子功能的私有实现，故直接置于 db/ 下，不嵌进 services/ 之类的空壳层。
 *
 * 本文件是「编辑器岛」的入口：只有 lazy.ts 动态 import 它，故对外仅暴露注册/注销两个函数。
 * 目录内部模块（types / contributors / quoter / sqlContext）各自按路径直接引用，不在此转发，
 * 以免出现无人消费的兼容出口（新增贡献者的扩展点见 contributors/index.ts）。
 */
import { disposeCompletionItemProvider, registerCompletionItemProvider } from '@/components/monaco/completionItemProvider';
import type { editor, languages, Position } from 'monaco-editor';
import { getSqlSplitOptions } from '../sql-editor/utils/sqlParser';
import { buildCompletionContext } from './context';
import { createDefaultContributors } from './contributors';
import { resolveCursorZone } from './sqlContext';

/** 本子系统接管的补全语言 */
const SQL_LANGUAGE = 'sql';

/** 注销数据库 SQL 补全提供者：与注册对称，调用方无需知道底层按语言维护的注册表 */
export function disposeDbCompletionItemProvider() {
    disposeCompletionItemProvider(SQL_LANGUAGE);
}

/**
 * 注册数据库 SQL 补全提供者（库名、表、字段、关键字等联想）。
 *
 * 架构说明（对齐开闭原则）：
 * - context.ts   统一构建补全上下文（语句提取、令牌/别名解析、标识符引用状态检测）；
 * - contributors 贡献者链按序产出建议，每类建议一个贡献者，新增类型零修改核心；
 * - quoter.ts    标识符引用策略唯一出口：光标处已被方言引用符包裹时插入裸名，
 *                修复 `` `id` `` 双击补全后变为 `` ``id`` `` 的二次包裹问题。
 *
 * @param dbId 数据库实例 id
 * @param db 当前库名
 * @param dbs 该实例所有库名
 * @param dbType 数据库类型
 */
export function registerDbCompletionItemProvider(dbId: number, db: string, dbs: string[] = [], dbType: string) {
    // 方言切割语义在注册期解析一次：避免每次按键补全都重新查注册表并构造能力对象
    const splitOptions = getSqlSplitOptions(dbType);

    registerCompletionItemProvider(SQL_LANGUAGE, {
        // '.' 别名/库名限定；空格为通用触发；'(' 用于 INSERT INTO t ( 列清单等场景
        triggerCharacters: ['.', ' ', '('],
        provideCompletionItems: async (model: editor.ITextModel, position: Position): Promise<languages.CompletionList | null | undefined> => {
            // 字符串字面量与注释内不提供代码提示（含行中尾注释，比仅判断行首更精确）
            if (resolveCursorZone(model.getValue(), model.getOffsetAt(position), splitOptions) !== 'code') {
                return { suggestions: [] };
            }

            const ctx = await buildCompletionContext(model, position, dbId, db, dbs, dbType);

            // 贡献者链：命中专属场景（exclusive）时提前收敛，仅返回该结果
            // 链为模块级稳定数组（注册表维护），此处遍历不产生额外分配
            let suggestions: languages.CompletionItem[] = [];
            for (const contributor of createDefaultContributors()) {
                const result = await contributor.contribute(ctx);
                if (!result) {
                    continue;
                }
                if (result.exclusive) {
                    return { suggestions: result.suggestions };
                }
                suggestions.push(...result.suggestions);
            }
            return { suggestions };
        },
    });
}
