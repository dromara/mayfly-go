/**
 * 验证服务单元测试
 */
import { describe, it, expect } from 'vitest';
import { validationService } from '../validationService';
import type { TableDefinition, TableColumnDefinition } from '../../../types/schema';

describe('ValidationService', () => {
    describe('validateTable', () => {
        it('应该通过有效的表定义', () => {
            const table: TableDefinition = {
                name: 'user_info',
                columns: [
                    { name: 'id', type: 'bigint', nullable: false, isPrimaryKey: true, autoIncrement: true, comment: '主键' },
                    { name: 'username', type: 'varchar', length: '50', nullable: false, isPrimaryKey: false, autoIncrement: false, comment: '用户名' },
                ],
                indexes: [],
                constraints: [],
            };

            const result = validationService.validateTable(table);
            expect(result.valid).toBe(true);
            expect(result.errors.filter((e) => e.severity === 'error')).toHaveLength(0);
        });

        it('应该检测空表名', () => {
            const table: TableDefinition = {
                name: '',
                columns: [{ name: 'id', type: 'bigint', nullable: false, isPrimaryKey: false, autoIncrement: false }],
                indexes: [],
                constraints: [],
            };

            const result = validationService.validateTable(table);
            expect(result.valid).toBe(false);
            expect(result.errors.some((e) => e.code === 'TABLE_NAME_REQUIRED')).toBe(true);
        });

        it('应该检测非法表名', () => {
            const table: TableDefinition = {
                name: '123table',
                columns: [{ name: 'id', type: 'bigint', nullable: false, isPrimaryKey: false, autoIncrement: false }],
                indexes: [],
                constraints: [],
            };

            const result = validationService.validateTable(table);
            expect(result.valid).toBe(false);
            expect(result.errors.some((e) => e.code === 'TABLE_NAME_INVALID')).toBe(true);
        });

        it('应该检测空列', () => {
            const table: TableDefinition = {
                name: 'test_table',
                columns: [],
                indexes: [],
                constraints: [],
            };

            const result = validationService.validateTable(table);
            expect(result.valid).toBe(false);
            expect(result.errors.some((e) => e.code === 'NO_COLUMNS')).toBe(true);
        });

        it('应该警告缺少主键', () => {
            const table: TableDefinition = {
                name: 'test_table',
                columns: [{ name: 'id', type: 'bigint', nullable: false, isPrimaryKey: false, autoIncrement: false }],
                indexes: [],
                constraints: [],
            };

            const result = validationService.validateTable(table);
            expect(result.warnings.some((w) => w.code === 'NO_PRIMARY_KEY')).toBe(true);
        });

        it('应该检测重复列名', () => {
            const table: TableDefinition = {
                name: 'test_table',
                columns: [
                    { name: 'id', type: 'bigint', nullable: false, isPrimaryKey: true, autoIncrement: false },
                    { name: 'id', type: 'varchar', nullable: true, isPrimaryKey: false, autoIncrement: false },
                ],
                indexes: [],
                constraints: [],
            };

            const result = validationService.validateTable(table);
            expect(result.valid).toBe(false);
            expect(result.errors.some((e) => e.code === 'DUPLICATE_COLUMN')).toBe(true);
        });
    });

    describe('validateColumn', () => {
        it('应该检测空列名', () => {
            const column: TableColumnDefinition = {
                name: '',
                type: 'varchar',
                nullable: true,
                isPrimaryKey: false,
                autoIncrement: false,
            };

            const errors = validationService.validateColumn(column, { name: 'test', columns: [column], indexes: [], constraints: [] });
            expect(errors.some((e) => e.code === 'COLUMN_NAME_REQUIRED')).toBe(true);
        });

        it('应该检测空列类型', () => {
            const column: TableColumnDefinition = {
                name: 'test_col',
                type: '',
                nullable: true,
                isPrimaryKey: false,
                autoIncrement: false,
            };

            const errors = validationService.validateColumn(column, { name: 'test', columns: [column], indexes: [], constraints: [] });
            expect(errors.some((e) => e.code === 'COLUMN_TYPE_REQUIRED')).toBe(true);
        });

        it('应该检测无效的自增列类型', () => {
            const column: TableColumnDefinition = {
                name: 'id',
                type: 'varchar',
                nullable: false,
                isPrimaryKey: true,
                autoIncrement: true,
            };

            const errors = validationService.validateColumn(column, { name: 'test', columns: [column], indexes: [], constraints: [] });
            expect(errors.some((e) => e.code === 'AUTO_INCREMENT_TYPE_INVALID')).toBe(true);
        });

        it('应该通过有效的自增列', () => {
            const column: TableColumnDefinition = {
                name: 'id',
                type: 'bigint',
                nullable: false,
                isPrimaryKey: true,
                autoIncrement: true,
            };

            const errors = validationService.validateColumn(column, { name: 'test', columns: [column], indexes: [], constraints: [] });
            expect(errors.filter((e) => e.severity === 'error')).toHaveLength(0);
        });
    });

    describe('quickValidate', () => {
        it('应该返回true对于有效表', () => {
            const table: TableDefinition = {
                name: 'user',
                columns: [{ name: 'id', type: 'bigint', nullable: false, isPrimaryKey: true, autoIncrement: false }],
                indexes: [],
                constraints: [],
            };

            expect(validationService.quickValidate(table)).toBe(true);
        });

        it('应该返回false对于无效表', () => {
            const table: TableDefinition = {
                name: '',
                columns: [],
                indexes: [],
                constraints: [],
            };

            expect(validationService.quickValidate(table)).toBe(false);
        });
    });

    describe('getValidationSummary', () => {
        it('应该返回验证通过', () => {
            const result = validationService.validateTable({
                name: 'user',
                columns: [{ name: 'id', type: 'bigint', nullable: false, isPrimaryKey: true, autoIncrement: false }],
                indexes: [],
                constraints: [],
            });

            expect(validationService.getValidationSummary(result)).toBe('✓ 验证通过');
        });
    });
});
