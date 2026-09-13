/**
 * 数据库表编辑系统 - 领域模型类型定义
 * 
 * 参考国际先进产品（DataGrip、DBeaver、TablePlus、Navicat）设计
 */

// ==================== 基础类型 ====================

/**
 * 表定义聚合根
 */
export interface TableDefinition {
  // 基础信息
  name: string;
  comment?: string;
  schema?: string;
  database?: string;
  
  // 组成元素
  columns: ColumnDefinition[];
  indexes: TableIndexDefinition[];
  constraints: ConstraintDefinition[];
  
  // 表选项
  engine?: string;
  charset?: string;
  collation?: string;
  rowFormat?: string;
  
  // 元数据
  createdAt?: number;
  updatedAt?: number;
  version?: number;
}

/**
 * 列定义
 */
export interface ColumnDefinition {
  name: string;
  oldName?: string; // 用于重命名追踪
  
  // 类型信息
  type: string;
  length?: string | number;
  numScale?: string | number;
  
  // 约束
  notNull: boolean;
  pri: boolean;
  unique?: boolean;
  auto_increment: boolean;
  
  // 默认值
  value?: string;
  defaultValue?: string;
  
  // 注释
  remark?: string;
  comment?: string;
  
  // 位置（MySQL）
  after?: string;
  first?: boolean;
  
  // 元数据
  ordinalPosition?: number;
  isNullable?: boolean;
  columnDefault?: string;
  extra?: string;
}

/**
 * 表索引定义（表编辑子系统的领域模型）
 *
 * 带 Table 前缀是为了与方言层的 IndexDefinition（dialect/types.ts）区分开：
 * 后者是生成 DDL 用的扁平契约（indexName / columnNames: string[] / indexType / indexComment），
 * 本模型是表编辑器用的富领域模型（列明细、算法、锁选项、重命名追踪），二者在 DbTableOp 里双向转换。
 *
 * 曾经两者同名，导致 DbTableOp 只能导入一侧、另一侧被遮蔽，转换处还混用了两侧的字段名
 * （本模型同时留有 indexType / indexComment）——结果是索引注释从未被填进快照，
 * SchemaDiffView 永远比不出注释变更。故此处不再保留方言层字段名，索引类型一律用 type、注释一律用 comment。
 */
export interface TableIndexDefinition {
  name: string;
  oldName?: string;
  
  // 索引列
  columns: IndexColumnDefinition[];
  
  // 索引属性
  unique: boolean;
  type: 'BTREE' | 'HASH' | 'FULLTEXT' | 'SPATIAL' | 'NORMAL';
  
  // 索引选项
  comment?: string;
  algorithm?: string;
  lock?: string;
  
  // 元数据
  cardinality?: number;
}

/**
 * 索引列定义
 */
export interface IndexColumnDefinition {
  name: string;
  length?: number;
  order?: 'ASC' | 'DESC';
}

/**
 * 约束定义
 */
export interface ConstraintDefinition {
  name: string;
  type: 'PRIMARY KEY' | 'FOREIGN KEY' | 'UNIQUE' | 'CHECK';
  
  // 主键/唯一约束
  columns?: string[];
  
  // 外键约束
  referencedTable?: string;
  referencedColumns?: string[];
  onDelete?: 'CASCADE' | 'SET NULL' | 'RESTRICT' | 'NO ACTION';
  onUpdate?: 'CASCADE' | 'SET NULL' | 'RESTRICT' | 'NO ACTION';
  
  // CHECK 约束
  checkExpression?: string;
  
  // 元数据
  comment?: string;
}

// ==================== 验证类型 ====================

/**
 * 验证结果
 */
export interface ValidationResult {
  valid: boolean;
  errors: ValidationError[];
  warnings: ValidationWarning[];
}

/**
 * 验证错误
 */
export interface ValidationError {
  code: string;
  message: string;
  severity: 'error' | 'warning' | 'info';
  location: {
    type: 'table' | 'column' | 'index' | 'constraint' | 'trigger' | 'partition';
    name: string;
    field?: string;
  };
  suggestion?: string;
}

/**
 * 验证警告（继承自 ValidationError）
 */
type ValidationWarning = ValidationError;

// ==================== 模板类型 ====================

/**
 * 表模板
 */
export interface TableTemplate {
  id: string;
  name: string;
  description: string;
  category: 'user' | 'log' | 'config' | 'dictionary' | 'custom';
  
  // 模板内容
  columns: ColumnTemplate[];
  indexes?: IndexTemplate[];
  constraints?: ConstraintTemplate[];
  
  // 元数据
  tags: string[];
  dialect?: string;
  usageCount: number;
  createdAt?: number;
}

/**
 * 列模板
 */
export interface ColumnTemplate {
  name: string;
  type: string;
  length?: string;
  notNull: boolean;
  defaultValue?: string;
  comment: string;
  isPrimaryKey?: boolean;
  autoIncrement?: boolean;
  unique?: boolean;
}

/**
 * 索引模板
 */
export interface IndexTemplate {
  name: string;
  columns: { name: string; length?: number }[];
  unique: boolean;
  type: 'BTREE' | 'HASH' | 'FULLTEXT' | 'SPATIAL' | 'NORMAL';
  comment?: string;
}

/**
 * 约束模板
 */
export interface ConstraintTemplate {
  name: string;
  type: 'PRIMARY KEY' | 'FOREIGN KEY' | 'UNIQUE' | 'CHECK';
  columns?: string[];
  referencedTable?: string;
  referencedColumns?: string[];
  checkExpression?: string;
  comment?: string;
}

// ==================== Schema Diff 类型 ====================

/**
 * Schema 快照
 */
export interface SchemaSnapshot {
  database: string;
  schema?: string;
  tables: TableDefinition[];
  timestamp: number;
}

/**
 * Schema 差异
 */
export interface SchemaDiff {
  tableLevel: {
    added: TableDefinition[];
    removed: TableDefinition[];
    modified: TableModification[];
  };
  columnLevel: ColumnDiff[];
  indexLevel: IndexDiff[];
  constraintLevel: ConstraintDiff[];
}

/**
 * 表修改详情
 */
export interface TableModification {
  tableName: string;
  changes: TableChange[];
}

/**
 * 表变更
 */
export interface TableChange {
  type: 'ALTER' | 'RENAME' | 'COMMENT' | 'ENGINE' | 'CHARSET';
  before: any;
  after: any;
  ddl: string;
}

/**
 * 列差异
 */
export interface ColumnDiff {
  tableName: string;
  added: ColumnDefinition[];
  removed: ColumnDefinition[];
  modified: ColumnModification[];
}

/**
 * 列修改
 */
export interface ColumnModification {
  columnName: string;
  before: ColumnDefinition;
  after: ColumnDefinition;
  changes: string[];
}

/**
 * 索引差异
 */
export interface IndexDiff {
  tableName: string;
  added: TableIndexDefinition[];
  removed: TableIndexDefinition[];
  modified: IndexModification[];
}

/**
 * 索引修改
 */
export interface IndexModification {
  indexName: string;
  before: TableIndexDefinition;
  after: TableIndexDefinition;
  changes: string[];
}

/**
 * 约束差异
 */
export interface ConstraintDiff {
  tableName: string;
  added: ConstraintDefinition[];
  removed: ConstraintDefinition[];
  modified: ConstraintModification[];
}

/**
 * 约束修改
 */
export interface ConstraintModification {
  constraintName: string;
  before: ConstraintDefinition;
  after: ConstraintDefinition;
  changes: string[];
}
