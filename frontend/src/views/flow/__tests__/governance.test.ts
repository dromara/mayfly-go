/**
 * 「生效资源」可选节点由后端下发的治理路径驱动，前端只做拼接。
 *
 * 这里曾经写过「数据库实例 → 换成库那条路径」「库名类型 → 跳过」两条分支，
 * 那份「谁在哪一层被治理」的知识属于后端注册表：复制过来就是第二真源，
 * 新接入一个粒度不同的资源（ES 索引、Mongo 集合）必须回来改这里。
 */
import { describe, expect, it } from 'vitest';

import { governTagTypesOf } from '../governance';

const MACHINE = 1;
const DB_INSTANCE = 2;
const AUTH_CERT = 5;
const DB_NAME = 22;
const REDIS = 3;

describe('治理范围的可选节点类型', () => {
    it('机器场景声明了治理路径就必须能被选到', () => {
        expect(governTagTypesOf([[MACHINE], [DB_INSTANCE, AUTH_CERT, DB_NAME], [REDIS]])).toContain(MACHINE);
    });

    it('多段路径拼成类型路径，单段路径保持数字', () => {
        expect(governTagTypesOf([[DB_INSTANCE, AUTH_CERT, DB_NAME], [REDIS]])).toEqual([`${DB_INSTANCE}/${AUTH_CERT}/${DB_NAME}`, REDIS]);
    });

    it('多个场景共用同一条路径时不产生重复节点', () => {
        expect(governTagTypesOf([[MACHINE], [MACHINE]])).toEqual([MACHINE]);
        expect(
            governTagTypesOf([
                [DB_INSTANCE, AUTH_CERT, DB_NAME],
                [DB_INSTANCE, AUTH_CERT, DB_NAME],
            ])
        ).toEqual([`${DB_INSTANCE}/${AUTH_CERT}/${DB_NAME}`]);
    });

    it('schema 未加载完时返回 null，让调用方等 schema 而不是先用保底清单', () => {
        expect(governTagTypesOf(undefined)).toBeNull();
        expect(governTagTypesOf([])).toBeNull();
        // 只有空段的路径等同于还没下发，不能渲染出一个 type= 的查询
        expect(governTagTypesOf([[], null as unknown as number[]])).toBeNull();
    });

    it('新场景只要声明治理路径就无需改前端（开闭原则）', () => {
        // 拿一个尚未出现在任何前端常量表里的资源类型与层级模拟「es 索引层接入」
        expect(governTagTypesOf([[6, 5, 99]])).toEqual(['6/5/99']);
    });
});
