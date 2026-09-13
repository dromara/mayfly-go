/**
 * 验证服务 - ValidationService
 * 
 * 提供完整的表结构验证功能
 */

import { i18n } from '@/i18n';
import type { 
  TableDefinition, 
  ColumnDefinition, 
  TableIndexDefinition,
  ConstraintDefinition,
  ValidationResult,
  ValidationError,
} from '../../types/schema';

class ValidationService {
  private t(key: string, params?: Record<string, any>): string {
    return i18n.global.t(key, params) as string;
  }

  /**
   * 验证表定义
   */
  validateTable(table: TableDefinition): ValidationResult {
    const errors: ValidationError[] = [];
    
    this.validateTableName(table.name, errors);
    
    if (table.columns.length === 0) {
      errors.push({
        code: 'NO_COLUMNS',
        message: this.t('db.valNoColumns'),
        severity: 'error',
        location: { type: 'table', name: table.name },
      });
    } else {
      table.columns.forEach((column) => {
        errors.push(...this.validateColumn(column, table));
      });
    }
    
    const primaryKeys = table.columns.filter(c => c.pri);
    if (primaryKeys.length === 0) {
      errors.push({
        code: 'NO_PRIMARY_KEY',
        message: this.t('db.valNoPriKey'),
        severity: 'warning',
        location: { type: 'table', name: table.name },
        suggestion: this.t('db.valNoPriKeySuggestion'),
      });
    }
    
    table.indexes.forEach((index) => {
      errors.push(...this.validateIndex(index, table));
    });
    
    table.constraints.forEach((constraint) => {
      errors.push(...this.validateConstraint(constraint, table));
    });
    
    this.checkDuplicateColumns(table.columns, table.name, errors);
    this.checkDuplicateIndexes(table.indexes, table.name, errors);
    
    return {
      valid: errors.filter(e => e.severity === 'error').length === 0,
      errors,
      warnings: errors.filter(e => e.severity === 'warning'),
    };
  }
  
  private validateTableName(name: string, errors: ValidationError[]) {
    const unnamed = this.t('db.valTableNameUnnamed');
    if (!name || name.trim() === '') {
      errors.push({
        code: 'TABLE_NAME_REQUIRED',
        message: this.t('db.valTableNameRequired'),
        severity: 'error',
        location: { type: 'table', name: name || unnamed },
      });
      return;
    }
    
    if (name.length > 64) {
      errors.push({
        code: 'TABLE_NAME_TOO_LONG',
        message: this.t('db.valTableNameTooLong'),
        severity: 'error',
        location: { type: 'table', name },
      });
    }
    
    if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(name)) {
      errors.push({
        code: 'TABLE_NAME_INVALID',
        message: this.t('db.valTableNameInvalid'),
        severity: 'error',
        location: { type: 'table', name },
        suggestion: this.t('db.valTableNameSuggestion'),
      });
    }
    
    const reservedWords = ['SELECT', 'FROM', 'WHERE', 'TABLE', 'INDEX', 'CREATE', 'DROP', 'ALTER'];
    if (reservedWords.includes(name.toUpperCase())) {
      errors.push({
        code: 'TABLE_NAME_RESERVED',
        message: this.t('db.valTableNameReserved', { name }),
        severity: 'warning',
        location: { type: 'table', name },
        suggestion: this.t('db.valTableNameReservedSuggestion'),
      });
    }
  }
  
  validateColumn(column: ColumnDefinition, table: TableDefinition): ValidationError[] {
    const errors: ValidationError[] = [];
    this.validateColumnName(column.name, column, errors);
    this.validateColumnType(column, errors);
    this.validateColumnLength(column, errors);
    this.validateAutoIncrement(column, errors);
    this.validateDefaultValue(column, errors);
    return errors;
  }
  
  private validateColumnName(name: string, column: ColumnDefinition, errors: ValidationError[]) {
    const unnamed = this.t('db.valTableNameUnnamed');
    if (!name || name.trim() === '') {
      errors.push({
        code: 'COLUMN_NAME_REQUIRED',
        message: this.t('db.valColNameRequired'),
        severity: 'error',
        location: { type: 'column', name: name || unnamed, field: 'name' },
      });
      return;
    }
    
    if (name.length > 64) {
      errors.push({
        code: 'COLUMN_NAME_TOO_LONG',
        message: this.t('db.valColNameTooLong'),
        severity: 'error',
        location: { type: 'column', name, field: 'name' },
      });
    }
    
    if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(name)) {
      errors.push({
        code: 'COLUMN_NAME_INVALID',
        message: this.t('db.valColNameInvalid'),
        severity: 'error',
        location: { type: 'column', name, field: 'name' },
      });
    }
  }
  
  private validateColumnType(column: ColumnDefinition, errors: ValidationError[]) {
    if (!column.type || column.type.trim() === '') {
      errors.push({
        code: 'COLUMN_TYPE_REQUIRED',
        message: this.t('db.valColTypeRequired'),
        severity: 'error',
        location: { type: 'column', name: column.name, field: 'type' },
      });
      return;
    }
    
    const validTypes = [
      'int', 'integer', 'bigint', 'smallint', 'tinyint', 'mediumint',
      'decimal', 'numeric', 'float', 'double', 'real',
      'varchar', 'char', 'text', 'tinytext', 'mediumtext', 'longtext',
      'date', 'time', 'datetime', 'timestamp', 'year',
      'blob', 'tinyblob', 'mediumblob', 'longblob',
      'boolean', 'bool', 'bit',
      'json', 'jsonb', 'xml',
      'uuid', 'uniqueidentifier', 'serial', 'bigserial',
      'enum', 'set',
    ];
    
    const typeLower = column.type.toLowerCase();
    const isValidType = validTypes.some(t => typeLower.includes(t));
    
    if (!isValidType) {
      errors.push({
        code: 'COLUMN_TYPE_INVALID',
        message: this.t('db.valColTypeUnknown', { type: column.type }),
        severity: 'warning',
        location: { type: 'column', name: column.name, field: 'type' },
        suggestion: this.t('db.valColTypeSuggestion'),
      });
    }
  }
  
  private validateColumnLength(column: ColumnDefinition, errors: ValidationError[]) {
    if (column.length !== undefined && column.length !== '') {
      const length = Number(column.length);
      if (isNaN(length) || length <= 0) {
        errors.push({
          code: 'COLUMN_LENGTH_INVALID',
          message: this.t('db.valColLengthInvalid'),
          severity: 'error',
          location: { type: 'column', name: column.name, field: 'length' },
        });
      }
      
      if (column.type.toLowerCase() === 'varchar' && length > 65535) {
        errors.push({
          code: 'COLUMN_LENGTH_TOO_LARGE',
          message: this.t('db.valVarcharTooLarge'),
          severity: 'error',
          location: { type: 'column', name: column.name, field: 'length' },
          suggestion: this.t('db.valVarcharTooLargeSuggestion'),
        });
      }
    }
    
    if (column.numScale !== undefined && column.numScale !== '') {
      const scale = Number(column.numScale);
      if (isNaN(scale) || scale < 0) {
        errors.push({
          code: 'COLUMN_SCALE_INVALID',
          message: this.t('db.valColScaleInvalid'),
          severity: 'error',
          location: { type: 'column', name: column.name, field: 'numScale' },
        });
      }
      
      if (column.length && scale > Number(column.length)) {
        errors.push({
          code: 'COLUMN_SCALE_TOO_LARGE',
          message: this.t('db.valColScaleTooLarge'),
          severity: 'error',
          location: { type: 'column', name: column.name, field: 'numScale' },
        });
      }
    }
  }
  
  private validateAutoIncrement(column: ColumnDefinition, errors: ValidationError[]) {
    if (!column.auto_increment) return;
    
    const intTypes = ['int', 'integer', 'bigint', 'smallint', 'tinyint', 'mediumint'];
    const isIntType = intTypes.some(t => column.type.toLowerCase().includes(t));
    
    if (!isIntType) {
      errors.push({
        code: 'AUTO_INCREMENT_TYPE_INVALID',
        message: this.t('db.valAutoIncTypeInvalid'),
        severity: 'error',
        location: { type: 'column', name: column.name, field: 'auto_increment' },
        suggestion: this.t('db.valAutoIncTypeSuggestion'),
      });
    }
    
    if (!column.pri && !column.unique) {
      errors.push({
        code: 'AUTO_INCREMENT_NOT_KEY',
        message: this.t('db.valAutoIncNotKey'),
        severity: 'warning',
        location: { type: 'column', name: column.name, field: 'auto_increment' },
        suggestion: this.t('db.valAutoIncNotKeySuggestion'),
      });
    }
    
    if (!column.notNull) {
      errors.push({
        code: 'AUTO_INCREMENT_NULLABLE',
        message: this.t('db.valAutoIncNullable'),
        severity: 'warning',
        location: { type: 'column', name: column.name, field: 'auto_increment' },
      });
    }
  }
  
  private validateDefaultValue(column: ColumnDefinition, errors: ValidationError[]) {
    const defaultValue = column.value || column.defaultValue;
    if (!defaultValue) return;
    
    // 整数类型默认值校验：必须为数字或函数调用（如 CURRENT_TIMESTAMP）
    // 注意：字符串类型默认值无需在此校验引号，各 dialect 层会在 DDL 生成时自动处理引号
    if (column.type.toLowerCase().includes('int') && isNaN(Number(defaultValue))) {
      if (!/^[A-Z_]+\(\)$/i.test(defaultValue)) {
        errors.push({
          code: 'DEFAULT_VALUE_INVALID',
          message: this.t('db.valDefaultIntInvalid'),
          severity: 'warning',
          location: { type: 'column', name: column.name, field: 'value' },
        });
      }
    }
  }
  
  validateIndex(index: TableIndexDefinition, table: TableDefinition): ValidationError[] {
    const errors: ValidationError[] = [];
    const unnamed = this.t('db.valTableNameUnnamed');
    
    if (!index.name || index.name.trim() === '') {
      errors.push({
        code: 'INDEX_NAME_REQUIRED',
        message: this.t('db.valIndexNameRequired'),
        severity: 'error',
        location: { type: 'index', name: index.name || unnamed },
      });
    }
    
    if (!index.columns || index.columns.length === 0) {
      errors.push({
        code: 'INDEX_NO_COLUMNS',
        message: this.t('db.valIndexNoColumns'),
        severity: 'error',
        location: { type: 'index', name: index.name },
      });
    } else {
      const columnNames = table.columns.map(c => c.name);
      index.columns.forEach(col => {
        if (!columnNames.includes(col.name)) {
          errors.push({
            code: 'INDEX_COLUMN_NOT_EXISTS',
            message: this.t('db.valIndexColNotExists', { name: col.name }),
            severity: 'error',
            location: { type: 'index', name: index.name },
          });
        }
      });
    }
    
    if (index.columns && index.columns.length > 16) {
      errors.push({
        code: 'INDEX_TOO_MANY_COLUMNS',
        message: this.t('db.valIndexTooManyCols'),
        severity: 'warning',
        location: { type: 'index', name: index.name },
        suggestion: this.t('db.valIndexTooManyColsSuggestion'),
      });
    }
    
    return errors;
  }
  
  validateConstraint(constraint: ConstraintDefinition, table: TableDefinition): ValidationError[] {
    const errors: ValidationError[] = [];
    const unnamed = this.t('db.valTableNameUnnamed');
    
    if (!constraint.name || constraint.name.trim() === '') {
      errors.push({
        code: 'CONSTRAINT_NAME_REQUIRED',
        message: this.t('db.valConstraintNameRequired'),
        severity: 'error',
        location: { type: 'constraint', name: constraint.name || unnamed },
      });
    }
    
    if (constraint.type === 'FOREIGN KEY') {
      this.validateForeignKey(constraint, table, errors);
    }
    
    if (constraint.type === 'CHECK') {
      if (!constraint.checkExpression || constraint.checkExpression.trim() === '') {
        errors.push({
          code: 'CHECK_NO_EXPRESSION',
          message: this.t('db.valCheckNoExpr'),
          severity: 'error',
          location: { type: 'constraint', name: constraint.name },
        });
      }
    }
    
    return errors;
  }
  
  private validateForeignKey(constraint: ConstraintDefinition, table: TableDefinition, errors: ValidationError[]) {
    if (!constraint.columns || constraint.columns.length === 0) {
      errors.push({
        code: 'FK_NO_COLUMNS',
        message: this.t('db.valFkNoColumns'),
        severity: 'error',
        location: { type: 'constraint', name: constraint.name },
      });
    }
    
    if (!constraint.referencedTable) {
      errors.push({
        code: 'FK_NO_REFERENCED_TABLE',
        message: this.t('db.valFkNoRefTable'),
        severity: 'error',
        location: { type: 'constraint', name: constraint.name },
      });
    }
    
    if (!constraint.referencedColumns || constraint.referencedColumns.length === 0) {
      errors.push({
        code: 'FK_NO_REFERENCED_COLUMNS',
        message: this.t('db.valFkNoRefColumns'),
        severity: 'error',
        location: { type: 'constraint', name: constraint.name },
      });
    }
    
    if (constraint.columns && constraint.referencedColumns &&
        constraint.columns.length !== constraint.referencedColumns.length) {
      errors.push({
        code: 'FK_COLUMN_MISMATCH',
        message: this.t('db.valFkColMismatch'),
        severity: 'error',
        location: { type: 'constraint', name: constraint.name },
      });
    }
    
    if (constraint.columns) {
      const columnNames = table.columns.map(c => c.name);
      constraint.columns.forEach(col => {
        if (!columnNames.includes(col)) {
          errors.push({
            code: 'FK_COLUMN_NOT_EXISTS',
            message: this.t('db.valFkColNotExists', { name: col }),
            severity: 'error',
            location: { type: 'constraint', name: constraint.name },
          });
        }
      });
    }
  }
  
  private checkDuplicateColumns(columns: ColumnDefinition[], tableName: string, errors: ValidationError[]) {
    const columnNames = columns.map(c => c.name);
    const duplicates = columnNames.filter((name, index) => columnNames.indexOf(name) !== index);
    
    if (duplicates.length > 0) {
      errors.push({
        code: 'DUPLICATE_COLUMN',
        message: this.t('db.valDuplicateColumns', { names: [...new Set(duplicates)].join(', ') }),
        severity: 'error',
        location: { type: 'table', name: tableName },
      });
    }
  }
  
  private checkDuplicateIndexes(indexes: TableIndexDefinition[], tableName: string, errors: ValidationError[]) {
    const indexNames = indexes.map(i => i.name);
    const duplicates = indexNames.filter((name, index) => indexNames.indexOf(name) !== index);
    
    if (duplicates.length > 0) {
      errors.push({
        code: 'DUPLICATE_INDEX',
        message: this.t('db.valDuplicateIndexes', { names: [...new Set(duplicates)].join(', ') }),
        severity: 'error',
        location: { type: 'table', name: tableName },
      });
    }
  }
  
  quickValidate(table: TableDefinition): boolean {
    return this.validateTable(table).valid;
  }
  
  getValidationSummary(result: ValidationResult): string {
    const errorCount = result.errors.filter(e => e.severity === 'error').length;
    const warningCount = result.warnings.length;
    
    if (errorCount === 0 && warningCount === 0) {
      return this.t('db.valSummaryPass');
    }
    
    const parts = [];
    if (errorCount > 0) {
      parts.push(this.t('db.valSummaryErrors', { count: errorCount }));
    }
    if (warningCount > 0) {
      parts.push(this.t('db.valSummaryWarnings', { count: warningCount }));
    }
    
    return parts.join(', ');
  }
}

export const validationService = new ValidationService();
