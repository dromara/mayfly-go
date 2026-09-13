/**
 * Schema Diff 引擎 - SchemaDiffService
 * 
 * 对比两个数据库 Schema 快照，生成差异报告
 */

import type {
  SchemaSnapshot,
  SchemaDiff,
  TableDefinition,
  TableModification,
  TableChange,
  ColumnDiff,
  ColumnModification,
  IndexDiff,
  IndexModification,
  ConstraintDiff,
  ConstraintModification,
  ColumnDefinition,
  TableIndexDefinition,
  ConstraintDefinition,
} from '../../types/schema';
import { i18n } from '@/i18n';

class SchemaDiffService {
  /**
   * 对比两个 Schema 快照
   */
  diff(source: SchemaSnapshot, target: SchemaSnapshot): SchemaDiff {
    const result: SchemaDiff = {
      tableLevel: {
        added: [],
        removed: [],
        modified: [],
      },
      columnLevel: [],
      indexLevel: [],
      constraintLevel: [],
    };
    
    const sourceTables = new Map(source.tables.map(t => [t.name, t]));
    const targetTables = new Map(target.tables.map(t => [t.name, t]));
    
    // 查找新增的表
    target.tables.forEach(table => {
      if (!sourceTables.has(table.name)) {
        result.tableLevel.added.push(table);
      }
    });
    
    // 查找删除的表
    source.tables.forEach(table => {
      if (!targetTables.has(table.name)) {
        result.tableLevel.removed.push(table);
      }
    });
    
    // 查找修改的表
    target.tables.forEach(targetTable => {
      const sourceTable = sourceTables.get(targetTable.name);
      if (sourceTable) {
        const modification = this.diffTable(sourceTable, targetTable);
        if (modification.changes.length > 0) {
          result.tableLevel.modified.push(modification);
        }
        
        // 对比列
        const columnDiff = this.diffColumns(targetTable.name, sourceTable.columns, targetTable.columns);
        if (columnDiff.added.length > 0 || columnDiff.removed.length > 0 || columnDiff.modified.length > 0) {
          result.columnLevel.push(columnDiff);
        }
        
        // 对比索引
        const indexDiff = this.diffIndexes(targetTable.name, sourceTable.indexes, targetTable.indexes);
        if (indexDiff.added.length > 0 || indexDiff.removed.length > 0 || indexDiff.modified.length > 0) {
          result.indexLevel.push(indexDiff);
        }
        
        // 对比约束
        const constraintDiff = this.diffConstraints(targetTable.name, sourceTable.constraints, targetTable.constraints);
        if (constraintDiff.added.length > 0 || constraintDiff.removed.length > 0 || constraintDiff.modified.length > 0) {
          result.constraintLevel.push(constraintDiff);
        }
      }
    });
    
    return result;
  }
  
  /**
   * 对比单个表
   */
  private diffTable(source: TableDefinition, target: TableDefinition): TableModification {
    const changes: TableChange[] = [];
    
    // 对比表注释
    if (source.comment !== target.comment) {
      changes.push({
        type: 'COMMENT',
        before: source.comment,
        after: target.comment,
        ddl: `-- ${i18n.global.t('db.diffChangeComment')}: ${source.comment || i18n.global.t('db.diffNone')} -> ${target.comment || i18n.global.t('db.diffNone')}`,
      });
    }
    
    // 对比表引擎（MySQL）
    if (source.engine && target.engine && source.engine !== target.engine) {
      changes.push({
        type: 'ENGINE',
        before: source.engine,
        after: target.engine,
        ddl: `ALTER TABLE ${target.name} ENGINE = ${target.engine}`,
      });
    }
    
    // 对比字符集
    if (source.charset && target.charset && source.charset !== target.charset) {
      changes.push({
        type: 'CHARSET',
        before: source.charset,
        after: target.charset,
        ddl: `ALTER TABLE ${target.name} CONVERT TO CHARACTER SET ${target.charset}`,
      });
    }
    
    return {
      tableName: target.name,
      changes,
    };
  }
  
  /**
   * 对比列
   */
  private diffColumns(
    tableName: string,
    sourceColumns: ColumnDefinition[],
    targetColumns: ColumnDefinition[]
  ): ColumnDiff {
    const result: ColumnDiff = {
      tableName,
      added: [],
      removed: [],
      modified: [],
    };
    
    const sourceMap = new Map(sourceColumns.map(c => [c.name, c]));
    const targetMap = new Map(targetColumns.map(c => [c.name, c]));
    
    // 查找新增的列
    targetColumns.forEach(col => {
      if (!sourceMap.has(col.name)) {
        result.added.push(col);
      }
    });
    
    // 查找删除的列
    sourceColumns.forEach(col => {
      if (!targetMap.has(col.name)) {
        result.removed.push(col);
      }
    });
    
    // 查找修改的列
    targetColumns.forEach(targetCol => {
      const sourceCol = sourceMap.get(targetCol.name);
      if (sourceCol) {
        const changes = this.diffColumn(sourceCol, targetCol);
        if (changes.length > 0) {
          result.modified.push({
            columnName: targetCol.name,
            before: sourceCol,
            after: targetCol,
            changes,
          });
        }
      }
    });
    
    return result;
  }
  
  /**
   * 对比单个列
   */
  private diffColumn(source: ColumnDefinition, target: ColumnDefinition): string[] {
    const changes: string[] = [];
    const t = i18n.global.t;
    const none = t('db.diffNone');
    
    // 对比类型
    if (source.type !== target.type) {
      changes.push(`${t('db.diffChangeType')}: ${source.type} -> ${target.type}`);
    }
    
    // 对比长度
    if (source.length !== target.length) {
      changes.push(`${t('db.diffChangeLength')}: ${source.length || none} -> ${target.length || none}`);
    }
    
    // 对比小数位
    if (source.numScale !== target.numScale) {
      changes.push(`${t('db.diffChangeScale')}: ${source.numScale || none} -> ${target.numScale || none}`);
    }
    
    // 对比非空约束
    if (source.notNull !== target.notNull) {
      changes.push(`${t('db.diffChangeNotNull')}: ${source.notNull} -> ${target.notNull}`);
    }
    
    // 对比主键
    if (source.pri !== target.pri) {
      changes.push(`${t('db.diffChangePri')}: ${source.pri} -> ${target.pri}`);
    }
    
    // 对比自增
    if (source.auto_increment !== target.auto_increment) {
      changes.push(`${t('db.diffChangeAutoInc')}: ${source.auto_increment} -> ${target.auto_increment}`);
    }
    
    // 对比默认值
    const sourceDefault = source.value || source.defaultValue;
    const targetDefault = target.value || target.defaultValue;
    if (sourceDefault !== targetDefault) {
      changes.push(`${t('db.diffChangeDefault')}: ${sourceDefault || none} -> ${targetDefault || none}`);
    }
    
    // 对比注释
    const sourceComment = source.remark || source.comment;
    const targetComment = target.remark || target.comment;
    if (sourceComment !== targetComment) {
      changes.push(`${t('db.diffChangeComment')}: ${sourceComment || none} -> ${targetComment || none}`);
    }
    
    return changes;
  }
  
  /**
   * 对比索引
   */
  private diffIndexes(
    tableName: string,
    sourceIndexes: TableIndexDefinition[],
    targetIndexes: TableIndexDefinition[]
  ): IndexDiff {
    const result: IndexDiff = {
      tableName,
      added: [],
      removed: [],
      modified: [],
    };
    
    const sourceMap = new Map(sourceIndexes.map(i => [i.name, i]));
    const targetMap = new Map(targetIndexes.map(i => [i.name, i]));
    
    // 查找新增的索引
    targetIndexes.forEach(idx => {
      if (!sourceMap.has(idx.name)) {
        result.added.push(idx);
      }
    });
    
    // 查找删除的索引
    sourceIndexes.forEach(idx => {
      if (!targetMap.has(idx.name)) {
        result.removed.push(idx);
      }
    });
    
    // 查找修改的索引
    targetIndexes.forEach(targetIdx => {
      const sourceIdx = sourceMap.get(targetIdx.name);
      if (sourceIdx) {
        const changes = this.diffIndex(sourceIdx, targetIdx);
        if (changes.length > 0) {
          result.modified.push({
            indexName: targetIdx.name,
            before: sourceIdx,
            after: targetIdx,
            changes,
          });
        }
      }
    });
    
    return result;
  }
  
  /**
   * 对比单个索引
   */
  private diffIndex(source: TableIndexDefinition, target: TableIndexDefinition): string[] {
    const changes: string[] = [];
    const t = i18n.global.t;
    const none = t('db.diffNone');
    
    // 对比唯一性
    if (source.unique !== target.unique) {
      changes.push(`${t('db.diffChangeUnique')}: ${source.unique} -> ${target.unique}`);
    }
    
    // 对比类型
    if (source.type !== target.type) {
      changes.push(`${t('db.diffChangeType')}: ${source.type} -> ${target.type}`);
    }
    
    // 对比索引列
    const sourceCols = source.columns.map(c => c.name).join(',');
    const targetCols = target.columns.map(c => c.name).join(',');
    if (sourceCols !== targetCols) {
      changes.push(`${t('db.diffChangeColumns')}: ${sourceCols} -> ${targetCols}`);
    }
    
    // 对比注释：本模型只有 comment 一个注释字段（方言层的 indexComment 由 buildTableDefinition 转换时填入）
    if (source.comment !== target.comment) {
      changes.push(`${t('db.diffChangeComment')}: ${source.comment || none} -> ${target.comment || none}`);
    }
    
    return changes;
  }
  
  /**
   * 对比约束
   */
  private diffConstraints(
    tableName: string,
    sourceConstraints: ConstraintDefinition[],
    targetConstraints: ConstraintDefinition[]
  ): ConstraintDiff {
    const result: ConstraintDiff = {
      tableName,
      added: [],
      removed: [],
      modified: [],
    };
    
    const sourceMap = new Map(sourceConstraints.map(c => [c.name, c]));
    const targetMap = new Map(targetConstraints.map(c => [c.name, c]));
    
    // 查找新增的约束
    targetConstraints.forEach(constraint => {
      if (!sourceMap.has(constraint.name)) {
        result.added.push(constraint);
      }
    });
    
    // 查找删除的约束
    sourceConstraints.forEach(constraint => {
      if (!targetMap.has(constraint.name)) {
        result.removed.push(constraint);
      }
    });
    
    // 查找修改的约束
    targetConstraints.forEach(targetConstraint => {
      const sourceConstraint = sourceMap.get(targetConstraint.name);
      if (sourceConstraint) {
        const changes = this.diffConstraint(sourceConstraint, targetConstraint);
        if (changes.length > 0) {
          result.modified.push({
            constraintName: targetConstraint.name,
            before: sourceConstraint,
            after: targetConstraint,
            changes,
          });
        }
      }
    });
    
    return result;
  }
  
  /**
   * 对比单个约束
   */
  private diffConstraint(source: ConstraintDefinition, target: ConstraintDefinition): string[] {
    const changes: string[] = [];
    const t = i18n.global.t;
    const none = t('db.diffNone');
    
    // 对比类型
    if (source.type !== target.type) {
      changes.push(`${t('db.diffChangeType')}: ${source.type} -> ${target.type}`);
    }
    
    // 对比列
    const sourceCols = (source.columns || []).join(',');
    const targetCols = (target.columns || []).join(',');
    if (sourceCols !== targetCols) {
      changes.push(`${t('db.diffChangeColumns')}: ${sourceCols || none} -> ${targetCols || none}`);
    }
    
    // 对比引用表（外键）
    if (source.referencedTable !== target.referencedTable) {
      changes.push(`${t('db.diffChangeRefTable')}: ${source.referencedTable || none} -> ${target.referencedTable || none}`);
    }
    
    // 对比引用列（外键）
    const sourceRefCols = (source.referencedColumns || []).join(',');
    const targetRefCols = (target.referencedColumns || []).join(',');
    if (sourceRefCols !== targetRefCols) {
      changes.push(`${t('db.diffChangeRefColumns')}: ${sourceRefCols || none} -> ${targetRefCols || none}`);
    }
    
    // 对比 ON DELETE
    if (source.onDelete !== target.onDelete) {
      changes.push(`ON DELETE: ${source.onDelete || none} -> ${target.onDelete || none}`);
    }
    
    // 对比 ON UPDATE
    if (source.onUpdate !== target.onUpdate) {
      changes.push(`ON UPDATE: ${source.onUpdate || none} -> ${target.onUpdate || none}`);
    }
    
    // 对比检查表达式（CHECK）
    if (source.checkExpression !== target.checkExpression) {
      changes.push(`${t('db.diffChangeCheckExpr')}: ${source.checkExpression || none} -> ${target.checkExpression || none}`);
    }
    
    return changes;
  }
  
  /**
   * 生成差异摘要
   */
  generateSummary(diff: SchemaDiff): string {
    const parts: string[] = [];
    const t = i18n.global.t;
    
    // 表级别变更
    if (diff.tableLevel.added.length > 0) {
      parts.push(t('db.diffSummaryAddedTables', { count: diff.tableLevel.added.length }));
    }
    if (diff.tableLevel.removed.length > 0) {
      parts.push(t('db.diffSummaryRemovedTables', { count: diff.tableLevel.removed.length }));
    }
    if (diff.tableLevel.modified.length > 0) {
      parts.push(t('db.diffSummaryModifiedTables', { count: diff.tableLevel.modified.length }));
    }
    
    // 列级别变更
    const columnChanges = diff.columnLevel.reduce((sum, d) => 
      sum + d.added.length + d.removed.length + d.modified.length, 0);
    if (columnChanges > 0) {
      parts.push(t('db.diffSummaryColumnChanges', { count: columnChanges }));
    }
    
    // 索引级别变更
    const indexChanges = diff.indexLevel.reduce((sum, d) => 
      sum + d.added.length + d.removed.length + d.modified.length, 0);
    if (indexChanges > 0) {
      parts.push(t('db.diffSummaryIndexChanges', { count: indexChanges }));
    }
    
    // 约束级别变更
    const constraintChanges = diff.constraintLevel.reduce((sum, d) => 
      sum + d.added.length + d.removed.length + d.modified.length, 0);
    if (constraintChanges > 0) {
      parts.push(t('db.diffSummaryConstraintChanges', { count: constraintChanges }));
    }
    
    return parts.length > 0 ? parts.join(', ') : t('db.diffSummaryNoChanges');
  }
  
  /**
   * 可视化差异（用于 UI 展示）
   */
  visualize(diff: SchemaDiff): any {
    return {
      summary: this.generateSummary(diff),
      tables: {
        added: diff.tableLevel.added.map(t => ({ name: t.name, columns: t.columns.length })),
        removed: diff.tableLevel.removed.map(t => ({ name: t.name, columns: t.columns.length })),
        modified: diff.tableLevel.modified.map(m => ({
          name: m.tableName,
          changes: m.changes.length,
        })),
      },
      columns: diff.columnLevel.map(d => ({
        tableName: d.tableName,
        added: d.added.map(c => c.name),
        removed: d.removed.map(c => c.name),
        modified: d.modified.map(m => m.columnName),
      })),
      indexes: diff.indexLevel.map(d => ({
        tableName: d.tableName,
        added: d.added.map(i => i.name),
        removed: d.removed.map(i => i.name),
        modified: d.modified.map(m => m.indexName),
      })),
      constraints: diff.constraintLevel.map(d => ({
        tableName: d.tableName,
        added: d.added.map(c => c.name),
        removed: d.removed.map(c => c.name),
        modified: d.modified.map(m => m.constraintName),
      })),
    };
  }
}

// 导出单例
export const schemaDiffService = new SchemaDiffService();
