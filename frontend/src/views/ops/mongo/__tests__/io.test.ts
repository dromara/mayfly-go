/**
 * 数据进出口的解析层测试。
 *
 * 守护两件事：坏行只影响它自己（其余文档照样可导入），
 * 以及提交前的形状判据不会把合法写法误拦（更新管道、含 options 的索引定义）。
 */
import { describe, expect, it } from 'vitest';

import { chunk, IMPORT_BATCH_SIZE, parseImportText } from '../io/parse';
import { indexSpecTemplate, parseIndexSpecs, parseJsonArray, parseJsonObject, parseUpdateSpec } from '../docview/json';

describe('parseImportText', () => {
    it('JSON 数组形态：非文档元素被记下但不牵连其他', () => {
        const { docs, issues } = parseImportText('[{"a":1}, "nope", {"b":2}]');
        expect(docs).toEqual([{ a: 1 }, { b: 2 }]);
        expect(issues).toEqual([{ at: 2, reason: 'notObject' }]);
    });

    it('单个文档对象视为一条', () => {
        expect(parseImportText('{"a":1}').docs).toEqual([{ a: 1 }]);
    });

    it('整体不是合法 JSON 时按 NDJSON 逐行解析并给出行号', () => {
        const { docs, issues } = parseImportText('{"a":1}\nnot json\n{"b":2}\n\n3');
        expect(docs).toEqual([{ a: 1 }, { b: 2 }]);
        expect(issues).toEqual([
            { at: 2, reason: 'invalidJson' },
            { at: 5, reason: 'notObject' },
        ]);
    });

    it('空内容与空行都不产出文档', () => {
        expect(parseImportText('')).toEqual({ docs: [], issues: [] });
        expect(parseImportText('\n  \n')).toEqual({ docs: [], issues: [] });
    });

    it('CRLF 换行的 NDJSON 行号与内容都按行算', () => {
        const { docs, issues } = parseImportText('{"a":1}\r\nbad\r\n');
        expect(docs).toEqual([{ a: 1 }]);
        expect(issues[0].at).toBe(2);
    });
});

describe('chunk', () => {
    it('末批允许不满', () => {
        const batches = chunk([1, 2, 3], 2);
        expect(batches).toEqual([[1, 2], [3]]);
    });

    it('按默认批大小切分且不丢元素', () => {
        const items = Array.from({ length: IMPORT_BATCH_SIZE * 2 + 5 }, (_, i) => i);
        const flat = chunk(items).flat();
        expect(flat.length).toBe(items.length);
        expect(chunk(items).length).toBe(3);
    });

    it('空列表切出空数组，非法批大小退回一次性提交', () => {
        expect(chunk([])).toEqual([]);
        expect(chunk([1, 2], 0)).toEqual([[1, 2]]);
    });
});

describe('写形状判据', () => {
    it('必须是对象的地方拒绝数组与标量', () => {
        expect(parseJsonObject('{}').ok).toBe(true);
        expect(parseJsonObject('[]').ok).toBe(false);
        expect(parseJsonObject('').ok).toBe(false);
    });

    it('数组形状要求非空', () => {
        expect(parseJsonArray('[{}]', 'mongo.pipelineInvalid').ok).toBe(true);
        expect(parseJsonArray('[]', 'mongo.pipelineInvalid').ok).toBe(false);
    });

    it('索引定义每项都要含 key，其他选项不拦', () => {
        expect(parseIndexSpecs('[{"key":{"a":1},"name":"a_1","expireAfterSeconds":60}]').ok).toBe(true);
        expect(parseIndexSpecs('[{"name":"a_1"}]').ok).toBe(false);
        expect(parseIndexSpecs('{"key":{"a":1}}').ok).toBe(false);
    });

    it('索引模板本身就是合法定义（复制出去的形态能被再次提交）', () => {
        const parsed = parseIndexSpecs(indexSpecTemplate('createdAt'));
        expect(parsed.ok).toBe(true);
        if (parsed.ok) {
            expect(parsed.value[0]).toHaveProperty('key');
        }
    });

    it('更新内容接受操作符文档与更新管道，拒绝裸替换文档', () => {
        expect(parseUpdateSpec('{"$set":{"a":1}}').ok).toBe(true);
        expect(parseUpdateSpec('[{"$set":{"a":1}}]').ok).toBe(true);
        // 裸替换文档会整条覆盖文档（丢掉所有其他字段），必须在提交前拦住
        expect(parseUpdateSpec('{"status":"paid"}').ok).toBe(false);
        expect(parseUpdateSpec('{"$set":{"a":1},"status":"paid"}').ok).toBe(false);
    });
});
