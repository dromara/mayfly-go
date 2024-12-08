// Code generated from MySqlParser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package parser // MySqlParser
import "github.com/antlr4-go/antlr/v4"

type BaseMySqlParserVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseMySqlParserVisitor) VisitRoot(ctx *RootContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSqlStatements(ctx *SqlStatementsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSqlStatement(ctx *SqlStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitEmptyStatement_(ctx *EmptyStatement_Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDdlStatement(ctx *DdlStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDmlStatement(ctx *DmlStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTransactionStatement(ctx *TransactionStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitReplicationStatement(ctx *ReplicationStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPreparedStatement(ctx *PreparedStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCompoundStatement(ctx *CompoundStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAdministrationStatement(ctx *AdministrationStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUtilityStatement(ctx *UtilityStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateDatabase(ctx *CreateDatabaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateEvent(ctx *CreateEventContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateIndex(ctx *CreateIndexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateLogfileGroup(ctx *CreateLogfileGroupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateProcedure(ctx *CreateProcedureContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateFunction(ctx *CreateFunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateRole(ctx *CreateRoleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateServer(ctx *CreateServerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCopyCreateTable(ctx *CopyCreateTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitQueryCreateTable(ctx *QueryCreateTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitColumnCreateTable(ctx *ColumnCreateTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateTablespaceInnodb(ctx *CreateTablespaceInnodbContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateTablespaceNdb(ctx *CreateTablespaceNdbContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateTrigger(ctx *CreateTriggerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitWithClause(ctx *WithClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCommonTableExpressions(ctx *CommonTableExpressionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCteName(ctx *CteNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCteColumnName(ctx *CteColumnNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateView(ctx *CreateViewContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateDatabaseOption(ctx *CreateDatabaseOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCharSet(ctx *CharSetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCurrentUserExpression(ctx *CurrentUserExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitOwnerStatement(ctx *OwnerStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPreciseSchedule(ctx *PreciseScheduleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIntervalSchedule(ctx *IntervalScheduleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTimestampValue(ctx *TimestampValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIntervalExpr(ctx *IntervalExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIntervalType(ctx *IntervalTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitEnableType(ctx *EnableTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIndexType(ctx *IndexTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIndexOption(ctx *IndexOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitProcedureParameter(ctx *ProcedureParameterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFunctionParameter(ctx *FunctionParameterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRoutineComment(ctx *RoutineCommentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRoutineLanguage(ctx *RoutineLanguageContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRoutineBehavior(ctx *RoutineBehaviorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRoutineData(ctx *RoutineDataContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRoutineSecurity(ctx *RoutineSecurityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitServerOption(ctx *ServerOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateDefinitions(ctx *CreateDefinitionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitColumnDeclaration(ctx *ColumnDeclarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitConstraintDeclaration(ctx *ConstraintDeclarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIndexDeclaration(ctx *IndexDeclarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitColumnDefinition(ctx *ColumnDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitNullColumnConstraint(ctx *NullColumnConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDefaultColumnConstraint(ctx *DefaultColumnConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitVisibilityColumnConstraint(ctx *VisibilityColumnConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitInvisibilityColumnConstraint(ctx *InvisibilityColumnConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAutoIncrementColumnConstraint(ctx *AutoIncrementColumnConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPrimaryKeyColumnConstraint(ctx *PrimaryKeyColumnConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUniqueKeyColumnConstraint(ctx *UniqueKeyColumnConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCommentColumnConstraint(ctx *CommentColumnConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFormatColumnConstraint(ctx *FormatColumnConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitStorageColumnConstraint(ctx *StorageColumnConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitReferenceColumnConstraint(ctx *ReferenceColumnConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCollateColumnConstraint(ctx *CollateColumnConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitGeneratedColumnConstraint(ctx *GeneratedColumnConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSerialDefaultColumnConstraint(ctx *SerialDefaultColumnConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCheckColumnConstraint(ctx *CheckColumnConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPrimaryKeyTableConstraint(ctx *PrimaryKeyTableConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUniqueKeyTableConstraint(ctx *UniqueKeyTableConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitForeignKeyTableConstraint(ctx *ForeignKeyTableConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCheckTableConstraint(ctx *CheckTableConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitReferenceDefinition(ctx *ReferenceDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitReferenceAction(ctx *ReferenceActionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitReferenceControlType(ctx *ReferenceControlTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSimpleIndexDeclaration(ctx *SimpleIndexDeclarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSpecialIndexDeclaration(ctx *SpecialIndexDeclarationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionEngine(ctx *TableOptionEngineContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionEngineAttribute(ctx *TableOptionEngineAttributeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionAutoextendSize(ctx *TableOptionAutoextendSizeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionAutoIncrement(ctx *TableOptionAutoIncrementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionAverage(ctx *TableOptionAverageContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionCharset(ctx *TableOptionCharsetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionChecksum(ctx *TableOptionChecksumContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionCollate(ctx *TableOptionCollateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionComment(ctx *TableOptionCommentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionCompression(ctx *TableOptionCompressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionConnection(ctx *TableOptionConnectionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionDataDirectory(ctx *TableOptionDataDirectoryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionDelay(ctx *TableOptionDelayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionEncryption(ctx *TableOptionEncryptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionPageCompressed(ctx *TableOptionPageCompressedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionPageCompressionLevel(ctx *TableOptionPageCompressionLevelContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionEncryptionKeyId(ctx *TableOptionEncryptionKeyIdContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionIndexDirectory(ctx *TableOptionIndexDirectoryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionInsertMethod(ctx *TableOptionInsertMethodContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionKeyBlockSize(ctx *TableOptionKeyBlockSizeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionMaxRows(ctx *TableOptionMaxRowsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionMinRows(ctx *TableOptionMinRowsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionPackKeys(ctx *TableOptionPackKeysContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionPassword(ctx *TableOptionPasswordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionRowFormat(ctx *TableOptionRowFormatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionStartTransaction(ctx *TableOptionStartTransactionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionSecondaryEngineAttribute(ctx *TableOptionSecondaryEngineAttributeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionRecalculation(ctx *TableOptionRecalculationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionPersistent(ctx *TableOptionPersistentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionSamplePage(ctx *TableOptionSamplePageContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionTablespace(ctx *TableOptionTablespaceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionTableType(ctx *TableOptionTableTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionTransactional(ctx *TableOptionTransactionalContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableOptionUnion(ctx *TableOptionUnionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableType(ctx *TableTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTablespaceStorage(ctx *TablespaceStorageContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionDefinitions(ctx *PartitionDefinitionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionFunctionHash(ctx *PartitionFunctionHashContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionFunctionKey(ctx *PartitionFunctionKeyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionFunctionRange(ctx *PartitionFunctionRangeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionFunctionList(ctx *PartitionFunctionListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSubPartitionFunctionHash(ctx *SubPartitionFunctionHashContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSubPartitionFunctionKey(ctx *SubPartitionFunctionKeyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionComparison(ctx *PartitionComparisonContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionListAtom(ctx *PartitionListAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionListVector(ctx *PartitionListVectorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionSimple(ctx *PartitionSimpleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionDefinerAtom(ctx *PartitionDefinerAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionDefinerVector(ctx *PartitionDefinerVectorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSubpartitionDefinition(ctx *SubpartitionDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionOptionEngine(ctx *PartitionOptionEngineContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionOptionComment(ctx *PartitionOptionCommentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionOptionDataDirectory(ctx *PartitionOptionDataDirectoryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionOptionIndexDirectory(ctx *PartitionOptionIndexDirectoryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionOptionMaxRows(ctx *PartitionOptionMaxRowsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionOptionMinRows(ctx *PartitionOptionMinRowsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionOptionTablespace(ctx *PartitionOptionTablespaceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionOptionNodeGroup(ctx *PartitionOptionNodeGroupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterSimpleDatabase(ctx *AlterSimpleDatabaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterUpgradeName(ctx *AlterUpgradeNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterEvent(ctx *AlterEventContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterFunction(ctx *AlterFunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterInstance(ctx *AlterInstanceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterLogfileGroup(ctx *AlterLogfileGroupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterProcedure(ctx *AlterProcedureContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterServer(ctx *AlterServerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterTable(ctx *AlterTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterTablespace(ctx *AlterTablespaceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterView(ctx *AlterViewContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByTableOption(ctx *AlterByTableOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByAddColumn(ctx *AlterByAddColumnContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByAddColumns(ctx *AlterByAddColumnsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByAddIndex(ctx *AlterByAddIndexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByAddPrimaryKey(ctx *AlterByAddPrimaryKeyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByAddUniqueKey(ctx *AlterByAddUniqueKeyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByAddSpecialIndex(ctx *AlterByAddSpecialIndexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByAddForeignKey(ctx *AlterByAddForeignKeyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByAddCheckTableConstraint(ctx *AlterByAddCheckTableConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByAlterCheckTableConstraint(ctx *AlterByAlterCheckTableConstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterBySetAlgorithm(ctx *AlterBySetAlgorithmContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByChangeDefault(ctx *AlterByChangeDefaultContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByChangeColumn(ctx *AlterByChangeColumnContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByRenameColumn(ctx *AlterByRenameColumnContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByLock(ctx *AlterByLockContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByModifyColumn(ctx *AlterByModifyColumnContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByDropColumn(ctx *AlterByDropColumnContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByDropConstraintCheck(ctx *AlterByDropConstraintCheckContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByDropPrimaryKey(ctx *AlterByDropPrimaryKeyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByDropIndex(ctx *AlterByDropIndexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByRenameIndex(ctx *AlterByRenameIndexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByAlterColumnDefault(ctx *AlterByAlterColumnDefaultContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByAlterIndexVisibility(ctx *AlterByAlterIndexVisibilityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByDropForeignKey(ctx *AlterByDropForeignKeyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByDisableKeys(ctx *AlterByDisableKeysContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByEnableKeys(ctx *AlterByEnableKeysContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByRename(ctx *AlterByRenameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByOrder(ctx *AlterByOrderContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByConvertCharset(ctx *AlterByConvertCharsetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByDefaultCharset(ctx *AlterByDefaultCharsetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByDiscardTablespace(ctx *AlterByDiscardTablespaceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByImportTablespace(ctx *AlterByImportTablespaceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByForce(ctx *AlterByForceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByValidate(ctx *AlterByValidateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByAddDefinitions(ctx *AlterByAddDefinitionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterPartition(ctx *AlterPartitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByAddPartition(ctx *AlterByAddPartitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByDropPartition(ctx *AlterByDropPartitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByDiscardPartition(ctx *AlterByDiscardPartitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByImportPartition(ctx *AlterByImportPartitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByTruncatePartition(ctx *AlterByTruncatePartitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByCoalescePartition(ctx *AlterByCoalescePartitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByReorganizePartition(ctx *AlterByReorganizePartitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByExchangePartition(ctx *AlterByExchangePartitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByAnalyzePartition(ctx *AlterByAnalyzePartitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByCheckPartition(ctx *AlterByCheckPartitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByOptimizePartition(ctx *AlterByOptimizePartitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByRebuildPartition(ctx *AlterByRebuildPartitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByRepairPartition(ctx *AlterByRepairPartitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByRemovePartitioning(ctx *AlterByRemovePartitioningContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterByUpgradePartitioning(ctx *AlterByUpgradePartitioningContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDropDatabase(ctx *DropDatabaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDropEvent(ctx *DropEventContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDropIndex(ctx *DropIndexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDropLogfileGroup(ctx *DropLogfileGroupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDropProcedure(ctx *DropProcedureContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDropFunction(ctx *DropFunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDropServer(ctx *DropServerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDropTable(ctx *DropTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDropTablespace(ctx *DropTablespaceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDropTrigger(ctx *DropTriggerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDropView(ctx *DropViewContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDropRole(ctx *DropRoleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSetRole(ctx *SetRoleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRenameTable(ctx *RenameTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRenameTableClause(ctx *RenameTableClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTruncateTable(ctx *TruncateTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCallStatement(ctx *CallStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDeleteStatement(ctx *DeleteStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDoStatement(ctx *DoStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitHandlerStatement(ctx *HandlerStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitInsertStatement(ctx *InsertStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLoadDataStatement(ctx *LoadDataStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLoadXmlStatement(ctx *LoadXmlStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitReplaceStatement(ctx *ReplaceStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSimpleSelect(ctx *SimpleSelectContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitParenthesisSelect(ctx *ParenthesisSelectContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUnionSelect(ctx *UnionSelectContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUnionParenthesisSelect(ctx *UnionParenthesisSelectContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitWithLateralStatement(ctx *WithLateralStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUpdateStatement(ctx *UpdateStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitValuesStatement(ctx *ValuesStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitInsertStatementValue(ctx *InsertStatementValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUpdatedElement(ctx *UpdatedElementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAssignmentField(ctx *AssignmentFieldContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLockClause(ctx *LockClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSingleDeleteStatement(ctx *SingleDeleteStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitMultipleDeleteStatement(ctx *MultipleDeleteStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitHandlerOpenStatement(ctx *HandlerOpenStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitHandlerReadIndexStatement(ctx *HandlerReadIndexStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitHandlerReadStatement(ctx *HandlerReadStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitHandlerCloseStatement(ctx *HandlerCloseStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSingleUpdateStatement(ctx *SingleUpdateStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitMultipleUpdateStatement(ctx *MultipleUpdateStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitOrderByClause(ctx *OrderByClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitOrderByExpression(ctx *OrderByExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableSources(ctx *TableSourcesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableSourceBase(ctx *TableSourceBaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableSourceNested(ctx *TableSourceNestedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableJson(ctx *TableJsonContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAtomTableItem(ctx *AtomTableItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSubqueryTableItem(ctx *SubqueryTableItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableSourcesItem(ctx *TableSourcesItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIndexHint(ctx *IndexHintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIndexHintType(ctx *IndexHintTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitInnerJoin(ctx *InnerJoinContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitStraightJoin(ctx *StraightJoinContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitOuterJoin(ctx *OuterJoinContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitNaturalJoin(ctx *NaturalJoinContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitJoinSpec(ctx *JoinSpecContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitQueryExpression(ctx *QueryExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitQueryExpressionNointo(ctx *QueryExpressionNointoContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitQuerySpecification(ctx *QuerySpecificationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitQuerySpecificationNointo(ctx *QuerySpecificationNointoContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUnionParenthesis(ctx *UnionParenthesisContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUnionStatement(ctx *UnionStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLateralStatement(ctx *LateralStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitJsonTable(ctx *JsonTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitJsonColumnList(ctx *JsonColumnListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitJsonColumn(ctx *JsonColumnContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitJsonOnEmpty(ctx *JsonOnEmptyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitJsonOnError(ctx *JsonOnErrorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSelectSpec(ctx *SelectSpecContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSelectElements(ctx *SelectElementsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSelectStarElement(ctx *SelectStarElementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSelectColumnElement(ctx *SelectColumnElementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSelectFunctionElement(ctx *SelectFunctionElementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSelectExpressionElement(ctx *SelectExpressionElementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSelectIntoVariables(ctx *SelectIntoVariablesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSelectIntoDumpFile(ctx *SelectIntoDumpFileContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSelectIntoTextFile(ctx *SelectIntoTextFileContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSelectFieldsInto(ctx *SelectFieldsIntoContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSelectLinesInto(ctx *SelectLinesIntoContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFromClause(ctx *FromClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitGroupByClause(ctx *GroupByClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitHavingClause(ctx *HavingClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitWindowClause(ctx *WindowClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitGroupByItem(ctx *GroupByItemContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLimitClause(ctx *LimitClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLimitClauseAtom(ctx *LimitClauseAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitStartTransaction(ctx *StartTransactionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitBeginWork(ctx *BeginWorkContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCommitWork(ctx *CommitWorkContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRollbackWork(ctx *RollbackWorkContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSavepointStatement(ctx *SavepointStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRollbackStatement(ctx *RollbackStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitReleaseStatement(ctx *ReleaseStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLockTables(ctx *LockTablesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUnlockTables(ctx *UnlockTablesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSetAutocommitStatement(ctx *SetAutocommitStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSetTransactionStatement(ctx *SetTransactionStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTransactionMode(ctx *TransactionModeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLockTableElement(ctx *LockTableElementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLockAction(ctx *LockActionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTransactionOption(ctx *TransactionOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTransactionLevel(ctx *TransactionLevelContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitChangeMaster(ctx *ChangeMasterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitChangeReplicationFilter(ctx *ChangeReplicationFilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPurgeBinaryLogs(ctx *PurgeBinaryLogsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitResetMaster(ctx *ResetMasterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitResetSlave(ctx *ResetSlaveContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitStartSlave(ctx *StartSlaveContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitStopSlave(ctx *StopSlaveContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitStartGroupReplication(ctx *StartGroupReplicationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitStopGroupReplication(ctx *StopGroupReplicationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitMasterStringOption(ctx *MasterStringOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitMasterDecimalOption(ctx *MasterDecimalOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitMasterBoolOption(ctx *MasterBoolOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitMasterRealOption(ctx *MasterRealOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitMasterUidListOption(ctx *MasterUidListOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitStringMasterOption(ctx *StringMasterOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDecimalMasterOption(ctx *DecimalMasterOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitBoolMasterOption(ctx *BoolMasterOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitChannelOption(ctx *ChannelOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDoDbReplication(ctx *DoDbReplicationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIgnoreDbReplication(ctx *IgnoreDbReplicationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDoTableReplication(ctx *DoTableReplicationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIgnoreTableReplication(ctx *IgnoreTableReplicationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitWildDoTableReplication(ctx *WildDoTableReplicationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitWildIgnoreTableReplication(ctx *WildIgnoreTableReplicationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRewriteDbReplication(ctx *RewriteDbReplicationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTablePair(ctx *TablePairContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitThreadType(ctx *ThreadTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitGtidsUntilOption(ctx *GtidsUntilOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitMasterLogUntilOption(ctx *MasterLogUntilOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRelayLogUntilOption(ctx *RelayLogUntilOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSqlGapsUntilOption(ctx *SqlGapsUntilOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUserConnectionOption(ctx *UserConnectionOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPasswordConnectionOption(ctx *PasswordConnectionOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDefaultAuthConnectionOption(ctx *DefaultAuthConnectionOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPluginDirConnectionOption(ctx *PluginDirConnectionOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitGtuidSet(ctx *GtuidSetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitXaStartTransaction(ctx *XaStartTransactionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitXaEndTransaction(ctx *XaEndTransactionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitXaPrepareStatement(ctx *XaPrepareStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitXaCommitWork(ctx *XaCommitWorkContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitXaRollbackWork(ctx *XaRollbackWorkContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitXaRecoverWork(ctx *XaRecoverWorkContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPrepareStatement(ctx *PrepareStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitExecuteStatement(ctx *ExecuteStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDeallocatePrepare(ctx *DeallocatePrepareContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRoutineBody(ctx *RoutineBodyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitBlockStatement(ctx *BlockStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCaseStatement(ctx *CaseStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIfStatement(ctx *IfStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIterateStatement(ctx *IterateStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLeaveStatement(ctx *LeaveStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLoopStatement(ctx *LoopStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRepeatStatement(ctx *RepeatStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitReturnStatement(ctx *ReturnStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitWhileStatement(ctx *WhileStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCloseCursor(ctx *CloseCursorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFetchCursor(ctx *FetchCursorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitOpenCursor(ctx *OpenCursorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDeclareVariable(ctx *DeclareVariableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDeclareCondition(ctx *DeclareConditionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDeclareCursor(ctx *DeclareCursorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDeclareHandler(ctx *DeclareHandlerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitHandlerConditionCode(ctx *HandlerConditionCodeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitHandlerConditionState(ctx *HandlerConditionStateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitHandlerConditionName(ctx *HandlerConditionNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitHandlerConditionWarning(ctx *HandlerConditionWarningContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitHandlerConditionNotfound(ctx *HandlerConditionNotfoundContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitHandlerConditionException(ctx *HandlerConditionExceptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitProcedureSqlStatement(ctx *ProcedureSqlStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCaseAlternative(ctx *CaseAlternativeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitElifAlternative(ctx *ElifAlternativeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterUserMysqlV56(ctx *AlterUserMysqlV56Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAlterUserMysqlV80(ctx *AlterUserMysqlV80Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateUserMysqlV56(ctx *CreateUserMysqlV56Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateUserMysqlV80(ctx *CreateUserMysqlV80Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDropUser(ctx *DropUserContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitGrantStatement(ctx *GrantStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRoleOption(ctx *RoleOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitGrantProxy(ctx *GrantProxyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRenameUser(ctx *RenameUserContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDetailRevoke(ctx *DetailRevokeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShortRevoke(ctx *ShortRevokeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRoleRevoke(ctx *RoleRevokeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRevokeProxy(ctx *RevokeProxyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSetPasswordStatement(ctx *SetPasswordStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUserSpecification(ctx *UserSpecificationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitHashAuthOption(ctx *HashAuthOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRandomAuthOption(ctx *RandomAuthOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitStringAuthOption(ctx *StringAuthOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitModuleAuthOption(ctx *ModuleAuthOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSimpleAuthOption(ctx *SimpleAuthOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAuthOptionClause(ctx *AuthOptionClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitModule(ctx *ModuleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPasswordModuleOption(ctx *PasswordModuleOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTlsOption(ctx *TlsOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUserResourceOption(ctx *UserResourceOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUserPasswordOption(ctx *UserPasswordOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUserLockOption(ctx *UserLockOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPrivelegeClause(ctx *PrivelegeClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPrivilege(ctx *PrivilegeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCurrentSchemaPriviLevel(ctx *CurrentSchemaPriviLevelContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitGlobalPrivLevel(ctx *GlobalPrivLevelContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDefiniteSchemaPrivLevel(ctx *DefiniteSchemaPrivLevelContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDefiniteFullTablePrivLevel(ctx *DefiniteFullTablePrivLevelContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDefiniteFullTablePrivLevel2(ctx *DefiniteFullTablePrivLevel2Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDefiniteTablePrivLevel(ctx *DefiniteTablePrivLevelContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRenameUserClause(ctx *RenameUserClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAnalyzeTable(ctx *AnalyzeTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCheckTable(ctx *CheckTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitChecksumTable(ctx *ChecksumTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitOptimizeTable(ctx *OptimizeTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRepairTable(ctx *RepairTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCheckTableOption(ctx *CheckTableOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCreateUdfunction(ctx *CreateUdfunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitInstallPlugin(ctx *InstallPluginContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUninstallPlugin(ctx *UninstallPluginContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSetVariable(ctx *SetVariableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSetCharset(ctx *SetCharsetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSetNames(ctx *SetNamesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSetPassword(ctx *SetPasswordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSetTransaction(ctx *SetTransactionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSetAutocommit(ctx *SetAutocommitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSetNewValueInsideTrigger(ctx *SetNewValueInsideTriggerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowMasterLogs(ctx *ShowMasterLogsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowLogEvents(ctx *ShowLogEventsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowObjectFilter(ctx *ShowObjectFilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowColumns(ctx *ShowColumnsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowCreateDb(ctx *ShowCreateDbContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowCreateFullIdObject(ctx *ShowCreateFullIdObjectContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowCreateUser(ctx *ShowCreateUserContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowEngine(ctx *ShowEngineContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowGlobalInfo(ctx *ShowGlobalInfoContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowErrors(ctx *ShowErrorsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowCountErrors(ctx *ShowCountErrorsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowSchemaFilter(ctx *ShowSchemaFilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowRoutine(ctx *ShowRoutineContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowGrants(ctx *ShowGrantsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowIndexes(ctx *ShowIndexesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowOpenTables(ctx *ShowOpenTablesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowProfile(ctx *ShowProfileContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowSlaveStatus(ctx *ShowSlaveStatusContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitVariableClause(ctx *VariableClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowCommonEntity(ctx *ShowCommonEntityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowFilter(ctx *ShowFilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowGlobalInfoClause(ctx *ShowGlobalInfoClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowSchemaEntity(ctx *ShowSchemaEntityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShowProfileType(ctx *ShowProfileTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitBinlogStatement(ctx *BinlogStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCacheIndexStatement(ctx *CacheIndexStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFlushStatement(ctx *FlushStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitKillStatement(ctx *KillStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLoadIndexIntoCache(ctx *LoadIndexIntoCacheContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitResetStatement(ctx *ResetStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitShutdownStatement(ctx *ShutdownStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableIndexes(ctx *TableIndexesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSimpleFlushOption(ctx *SimpleFlushOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitChannelFlushOption(ctx *ChannelFlushOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableFlushOption(ctx *TableFlushOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFlushTableOption(ctx *FlushTableOptionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLoadedTableIndexes(ctx *LoadedTableIndexesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSimpleDescribeStatement(ctx *SimpleDescribeStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFullDescribeStatement(ctx *FullDescribeStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitHelpStatement(ctx *HelpStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUseStatement(ctx *UseStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSignalStatement(ctx *SignalStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitResignalStatement(ctx *ResignalStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSignalConditionInformation(ctx *SignalConditionInformationContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitWithStatement(ctx *WithStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableStatement(ctx *TableStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDiagnosticsStatement(ctx *DiagnosticsStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDiagnosticsConditionInformationName(ctx *DiagnosticsConditionInformationNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDescribeStatements(ctx *DescribeStatementsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDescribeConnection(ctx *DescribeConnectionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFullId(ctx *FullIdContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTableName(ctx *TableNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRoleName(ctx *RoleNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFullColumnName(ctx *FullColumnNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIndexColumnName(ctx *IndexColumnNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSimpleUserName(ctx *SimpleUserNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitHostName(ctx *HostNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUserName(ctx *UserNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitMysqlVariable(ctx *MysqlVariableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCharsetName(ctx *CharsetNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCollationName(ctx *CollationNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitEngineName(ctx *EngineNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitEngineNameBase(ctx *EngineNameBaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUuidSet(ctx *UuidSetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitXid(ctx *XidContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitXuidStringId(ctx *XuidStringIdContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAuthPlugin(ctx *AuthPluginContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUid(ctx *UidContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSimpleId(ctx *SimpleIdContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDottedId(ctx *DottedIdContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDecimalLiteral(ctx *DecimalLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFileSizeLiteral(ctx *FileSizeLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitStringLiteral(ctx *StringLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitBooleanLiteral(ctx *BooleanLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitHexadecimalLiteral(ctx *HexadecimalLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitNullNotnull(ctx *NullNotnullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitConstant(ctx *ConstantContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitStringDataType(ctx *StringDataTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitNationalVaryingStringDataType(ctx *NationalVaryingStringDataTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitNationalStringDataType(ctx *NationalStringDataTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDimensionDataType(ctx *DimensionDataTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSimpleDataType(ctx *SimpleDataTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCollectionDataType(ctx *CollectionDataTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSpatialDataType(ctx *SpatialDataTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLongVarcharDataType(ctx *LongVarcharDataTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLongVarbinaryDataType(ctx *LongVarbinaryDataTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCollectionOptions(ctx *CollectionOptionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitConvertedDataType(ctx *ConvertedDataTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLengthOneDimension(ctx *LengthOneDimensionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLengthTwoDimension(ctx *LengthTwoDimensionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLengthTwoOptionalDimension(ctx *LengthTwoOptionalDimensionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUidList(ctx *UidListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFullColumnNameList(ctx *FullColumnNameListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTables(ctx *TablesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIndexColumnNames(ctx *IndexColumnNamesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitExpressions(ctx *ExpressionsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitExpressionsWithDefaults(ctx *ExpressionsWithDefaultsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitConstants(ctx *ConstantsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSimpleStrings(ctx *SimpleStringsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUserVariables(ctx *UserVariablesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDefaultValue(ctx *DefaultValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCurrentTimestamp(ctx *CurrentTimestampContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitExpressionOrDefault(ctx *ExpressionOrDefaultContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIfExists(ctx *IfExistsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIfNotExists(ctx *IfNotExistsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitOrReplace(ctx *OrReplaceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitWaitNowaitClause(ctx *WaitNowaitClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSpecificFunctionCall(ctx *SpecificFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAggregateFunctionCall(ctx *AggregateFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitNonAggregateFunctionCall(ctx *NonAggregateFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitScalarFunctionCall(ctx *ScalarFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUdfFunctionCall(ctx *UdfFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPasswordFunctionCall(ctx *PasswordFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSimpleFunctionCall(ctx *SimpleFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCurrentUser(ctx *CurrentUserContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDataTypeFunctionCall(ctx *DataTypeFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitValuesFunctionCall(ctx *ValuesFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCaseExpressionFunctionCall(ctx *CaseExpressionFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCaseFunctionCall(ctx *CaseFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCharFunctionCall(ctx *CharFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPositionFunctionCall(ctx *PositionFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSubstrFunctionCall(ctx *SubstrFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTrimFunctionCall(ctx *TrimFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitWeightFunctionCall(ctx *WeightFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitExtractFunctionCall(ctx *ExtractFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitGetFormatFunctionCall(ctx *GetFormatFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitJsonValueFunctionCall(ctx *JsonValueFunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCaseFuncAlternative(ctx *CaseFuncAlternativeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLevelWeightList(ctx *LevelWeightListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLevelWeightRange(ctx *LevelWeightRangeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLevelInWeightListElement(ctx *LevelInWeightListElementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAggregateWindowedFunction(ctx *AggregateWindowedFunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitNonAggregateWindowedFunction(ctx *NonAggregateWindowedFunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitOverClause(ctx *OverClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitWindowSpec(ctx *WindowSpecContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitWindowName(ctx *WindowNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFrameClause(ctx *FrameClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFrameUnits(ctx *FrameUnitsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFrameExtent(ctx *FrameExtentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFrameBetween(ctx *FrameBetweenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFrameRange(ctx *FrameRangeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPartitionClause(ctx *PartitionClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitScalarFunctionName(ctx *ScalarFunctionNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPasswordFunctionClause(ctx *PasswordFunctionClauseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFunctionArgs(ctx *FunctionArgsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFunctionArg(ctx *FunctionArgContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIsExpression(ctx *IsExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitNotExpression(ctx *NotExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLogicalExpression(ctx *LogicalExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPredicateExpression(ctx *PredicateExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSoundsLikePredicate(ctx *SoundsLikePredicateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitExpressionAtomPredicate(ctx *ExpressionAtomPredicateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSubqueryComparisonPredicate(ctx *SubqueryComparisonPredicateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitJsonMemberOfPredicate(ctx *JsonMemberOfPredicateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitBinaryComparisonPredicate(ctx *BinaryComparisonPredicateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitInPredicate(ctx *InPredicateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitBetweenPredicate(ctx *BetweenPredicateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIsNullPredicate(ctx *IsNullPredicateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLikePredicate(ctx *LikePredicateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitRegexpPredicate(ctx *RegexpPredicateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUnaryExpressionAtom(ctx *UnaryExpressionAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCollateExpressionAtom(ctx *CollateExpressionAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitVariableAssignExpressionAtom(ctx *VariableAssignExpressionAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitMysqlVariableExpressionAtom(ctx *MysqlVariableExpressionAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitNestedExpressionAtom(ctx *NestedExpressionAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitNestedRowExpressionAtom(ctx *NestedRowExpressionAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitMathExpressionAtom(ctx *MathExpressionAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitExistsExpressionAtom(ctx *ExistsExpressionAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIntervalExpressionAtom(ctx *IntervalExpressionAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitJsonExpressionAtom(ctx *JsonExpressionAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitSubqueryExpressionAtom(ctx *SubqueryExpressionAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitConstantExpressionAtom(ctx *ConstantExpressionAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFunctionCallExpressionAtom(ctx *FunctionCallExpressionAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitBinaryExpressionAtom(ctx *BinaryExpressionAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFullColumnNameExpressionAtom(ctx *FullColumnNameExpressionAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitBitExpressionAtom(ctx *BitExpressionAtomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitUnaryOperator(ctx *UnaryOperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitComparisonOperator(ctx *ComparisonOperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitLogicalOperator(ctx *LogicalOperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitBitOperator(ctx *BitOperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitMultOperator(ctx *MultOperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitAddOperator(ctx *AddOperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitJsonOperator(ctx *JsonOperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitCharsetNameBase(ctx *CharsetNameBaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitTransactionLevelBase(ctx *TransactionLevelBaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitPrivilegesBase(ctx *PrivilegesBaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitIntervalTypeBase(ctx *IntervalTypeBaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitDataTypeBase(ctx *DataTypeBaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitKeywordsCanBeId(ctx *KeywordsCanBeIdContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseMySqlParserVisitor) VisitFunctionNameBase(ctx *FunctionNameBaseContext) interface{} {
	return v.VisitChildren(ctx)
}
