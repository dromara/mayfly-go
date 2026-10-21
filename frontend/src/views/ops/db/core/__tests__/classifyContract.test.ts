/**
 * SQL 读写分类跨语言契约（前端侧）：与服务端 internal/db/dbm/classify_contract_test.go 读取同一份
 * server/internal/db/dbm/sqlparser/testdata/classify_cases.json。
 *
 * 断言前端 isQueryStatement 的读/写判定与契约 isQuery 一致。任一侧改动分类规则而未同步 fixture，
 * 另一侧就会变红——防止「后端判读、前端判写」这类漂移（历史上 SELECT INTO 就漂移过一次）。
 */
import { existsSync, readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { describe, expect, it } from 'vitest';

import { isQueryStatement } from '../sqlKind';
import { getDbDialect, getDialectCapabilities } from '../../dialect';

const findRepoRoot = (start: string): string => {
    let dir = start;
    for (let i = 0; i < 12; i++) {
        if (existsSync(join(dir, 'server')) && existsSync(join(dir, 'frontend'))) {
            return dir;
        }
        const parent = dirname(dir);
        if (parent === dir) {
            break;
        }
        dir = parent;
    }
    throw new Error('未定位到仓库根，无法读取 SQL 分类契约文件');
};

const CLASSIFY_CONTRACT_FILE = join(findRepoRoot(import.meta.dirname), 'server', 'internal/db/dbm/sqlparser/testdata/classify_cases.json');

type ClassifyCase = { name: string; dbType: string; sql: string; stmtType: string; isQuery: boolean };

describe('SQL 读写分类跨语言契约（前端 isQueryStatement）', () => {
    const { cases } = JSON.parse(readFileSync(CLASSIFY_CONTRACT_FILE, 'utf-8')) as { cases: ClassifyCase[] };

    it('契约存在且非空', () => {
        expect(cases.length).toBeGreaterThanOrEqual(20);
    });

    it.each(cases)('$name', (c) => {
        const caps = getDialectCapabilities(getDbDialect(c.dbType));
        const got = isQueryStatement(c.sql, { extraReadKeywords: caps.queryStatementKeywords, splitOptions: caps.sqlSplitOptions });
        expect(got, `[${c.dbType}] ${c.sql}`).toBe(c.isQuery);
    });
});
