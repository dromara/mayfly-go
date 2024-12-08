// Code generated from MySqlParser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package parser // MySqlParser
import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by MySqlParser.
type MySqlParserVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by MySqlParser#root.
	VisitRoot(ctx *RootContext) interface{}

	// Visit a parse tree produced by MySqlParser#sqlStatements.
	VisitSqlStatements(ctx *SqlStatementsContext) interface{}

	// Visit a parse tree produced by MySqlParser#sqlStatement.
	VisitSqlStatement(ctx *SqlStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#emptyStatement_.
	VisitEmptyStatement_(ctx *EmptyStatement_Context) interface{}

	// Visit a parse tree produced by MySqlParser#ddlStatement.
	VisitDdlStatement(ctx *DdlStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#dmlStatement.
	VisitDmlStatement(ctx *DmlStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#transactionStatement.
	VisitTransactionStatement(ctx *TransactionStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#replicationStatement.
	VisitReplicationStatement(ctx *ReplicationStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#preparedStatement.
	VisitPreparedStatement(ctx *PreparedStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#compoundStatement.
	VisitCompoundStatement(ctx *CompoundStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#administrationStatement.
	VisitAdministrationStatement(ctx *AdministrationStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#utilityStatement.
	VisitUtilityStatement(ctx *UtilityStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#createDatabase.
	VisitCreateDatabase(ctx *CreateDatabaseContext) interface{}

	// Visit a parse tree produced by MySqlParser#createEvent.
	VisitCreateEvent(ctx *CreateEventContext) interface{}

	// Visit a parse tree produced by MySqlParser#createIndex.
	VisitCreateIndex(ctx *CreateIndexContext) interface{}

	// Visit a parse tree produced by MySqlParser#createLogfileGroup.
	VisitCreateLogfileGroup(ctx *CreateLogfileGroupContext) interface{}

	// Visit a parse tree produced by MySqlParser#createProcedure.
	VisitCreateProcedure(ctx *CreateProcedureContext) interface{}

	// Visit a parse tree produced by MySqlParser#createFunction.
	VisitCreateFunction(ctx *CreateFunctionContext) interface{}

	// Visit a parse tree produced by MySqlParser#createRole.
	VisitCreateRole(ctx *CreateRoleContext) interface{}

	// Visit a parse tree produced by MySqlParser#createServer.
	VisitCreateServer(ctx *CreateServerContext) interface{}

	// Visit a parse tree produced by MySqlParser#copyCreateTable.
	VisitCopyCreateTable(ctx *CopyCreateTableContext) interface{}

	// Visit a parse tree produced by MySqlParser#queryCreateTable.
	VisitQueryCreateTable(ctx *QueryCreateTableContext) interface{}

	// Visit a parse tree produced by MySqlParser#columnCreateTable.
	VisitColumnCreateTable(ctx *ColumnCreateTableContext) interface{}

	// Visit a parse tree produced by MySqlParser#createTablespaceInnodb.
	VisitCreateTablespaceInnodb(ctx *CreateTablespaceInnodbContext) interface{}

	// Visit a parse tree produced by MySqlParser#createTablespaceNdb.
	VisitCreateTablespaceNdb(ctx *CreateTablespaceNdbContext) interface{}

	// Visit a parse tree produced by MySqlParser#createTrigger.
	VisitCreateTrigger(ctx *CreateTriggerContext) interface{}

	// Visit a parse tree produced by MySqlParser#withClause.
	VisitWithClause(ctx *WithClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#commonTableExpressions.
	VisitCommonTableExpressions(ctx *CommonTableExpressionsContext) interface{}

	// Visit a parse tree produced by MySqlParser#cteName.
	VisitCteName(ctx *CteNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#cteColumnName.
	VisitCteColumnName(ctx *CteColumnNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#createView.
	VisitCreateView(ctx *CreateViewContext) interface{}

	// Visit a parse tree produced by MySqlParser#createDatabaseOption.
	VisitCreateDatabaseOption(ctx *CreateDatabaseOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#charSet.
	VisitCharSet(ctx *CharSetContext) interface{}

	// Visit a parse tree produced by MySqlParser#currentUserExpression.
	VisitCurrentUserExpression(ctx *CurrentUserExpressionContext) interface{}

	// Visit a parse tree produced by MySqlParser#ownerStatement.
	VisitOwnerStatement(ctx *OwnerStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#preciseSchedule.
	VisitPreciseSchedule(ctx *PreciseScheduleContext) interface{}

	// Visit a parse tree produced by MySqlParser#intervalSchedule.
	VisitIntervalSchedule(ctx *IntervalScheduleContext) interface{}

	// Visit a parse tree produced by MySqlParser#timestampValue.
	VisitTimestampValue(ctx *TimestampValueContext) interface{}

	// Visit a parse tree produced by MySqlParser#intervalExpr.
	VisitIntervalExpr(ctx *IntervalExprContext) interface{}

	// Visit a parse tree produced by MySqlParser#intervalType.
	VisitIntervalType(ctx *IntervalTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#enableType.
	VisitEnableType(ctx *EnableTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#indexType.
	VisitIndexType(ctx *IndexTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#indexOption.
	VisitIndexOption(ctx *IndexOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#procedureParameter.
	VisitProcedureParameter(ctx *ProcedureParameterContext) interface{}

	// Visit a parse tree produced by MySqlParser#functionParameter.
	VisitFunctionParameter(ctx *FunctionParameterContext) interface{}

	// Visit a parse tree produced by MySqlParser#routineComment.
	VisitRoutineComment(ctx *RoutineCommentContext) interface{}

	// Visit a parse tree produced by MySqlParser#routineLanguage.
	VisitRoutineLanguage(ctx *RoutineLanguageContext) interface{}

	// Visit a parse tree produced by MySqlParser#routineBehavior.
	VisitRoutineBehavior(ctx *RoutineBehaviorContext) interface{}

	// Visit a parse tree produced by MySqlParser#routineData.
	VisitRoutineData(ctx *RoutineDataContext) interface{}

	// Visit a parse tree produced by MySqlParser#routineSecurity.
	VisitRoutineSecurity(ctx *RoutineSecurityContext) interface{}

	// Visit a parse tree produced by MySqlParser#serverOption.
	VisitServerOption(ctx *ServerOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#createDefinitions.
	VisitCreateDefinitions(ctx *CreateDefinitionsContext) interface{}

	// Visit a parse tree produced by MySqlParser#columnDeclaration.
	VisitColumnDeclaration(ctx *ColumnDeclarationContext) interface{}

	// Visit a parse tree produced by MySqlParser#constraintDeclaration.
	VisitConstraintDeclaration(ctx *ConstraintDeclarationContext) interface{}

	// Visit a parse tree produced by MySqlParser#indexDeclaration.
	VisitIndexDeclaration(ctx *IndexDeclarationContext) interface{}

	// Visit a parse tree produced by MySqlParser#columnDefinition.
	VisitColumnDefinition(ctx *ColumnDefinitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#nullColumnConstraint.
	VisitNullColumnConstraint(ctx *NullColumnConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#defaultColumnConstraint.
	VisitDefaultColumnConstraint(ctx *DefaultColumnConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#visibilityColumnConstraint.
	VisitVisibilityColumnConstraint(ctx *VisibilityColumnConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#invisibilityColumnConstraint.
	VisitInvisibilityColumnConstraint(ctx *InvisibilityColumnConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#autoIncrementColumnConstraint.
	VisitAutoIncrementColumnConstraint(ctx *AutoIncrementColumnConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#primaryKeyColumnConstraint.
	VisitPrimaryKeyColumnConstraint(ctx *PrimaryKeyColumnConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#uniqueKeyColumnConstraint.
	VisitUniqueKeyColumnConstraint(ctx *UniqueKeyColumnConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#commentColumnConstraint.
	VisitCommentColumnConstraint(ctx *CommentColumnConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#formatColumnConstraint.
	VisitFormatColumnConstraint(ctx *FormatColumnConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#storageColumnConstraint.
	VisitStorageColumnConstraint(ctx *StorageColumnConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#referenceColumnConstraint.
	VisitReferenceColumnConstraint(ctx *ReferenceColumnConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#collateColumnConstraint.
	VisitCollateColumnConstraint(ctx *CollateColumnConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#generatedColumnConstraint.
	VisitGeneratedColumnConstraint(ctx *GeneratedColumnConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#serialDefaultColumnConstraint.
	VisitSerialDefaultColumnConstraint(ctx *SerialDefaultColumnConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#checkColumnConstraint.
	VisitCheckColumnConstraint(ctx *CheckColumnConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#primaryKeyTableConstraint.
	VisitPrimaryKeyTableConstraint(ctx *PrimaryKeyTableConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#uniqueKeyTableConstraint.
	VisitUniqueKeyTableConstraint(ctx *UniqueKeyTableConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#foreignKeyTableConstraint.
	VisitForeignKeyTableConstraint(ctx *ForeignKeyTableConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#checkTableConstraint.
	VisitCheckTableConstraint(ctx *CheckTableConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#referenceDefinition.
	VisitReferenceDefinition(ctx *ReferenceDefinitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#referenceAction.
	VisitReferenceAction(ctx *ReferenceActionContext) interface{}

	// Visit a parse tree produced by MySqlParser#referenceControlType.
	VisitReferenceControlType(ctx *ReferenceControlTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#simpleIndexDeclaration.
	VisitSimpleIndexDeclaration(ctx *SimpleIndexDeclarationContext) interface{}

	// Visit a parse tree produced by MySqlParser#specialIndexDeclaration.
	VisitSpecialIndexDeclaration(ctx *SpecialIndexDeclarationContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionEngine.
	VisitTableOptionEngine(ctx *TableOptionEngineContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionEngineAttribute.
	VisitTableOptionEngineAttribute(ctx *TableOptionEngineAttributeContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionAutoextendSize.
	VisitTableOptionAutoextendSize(ctx *TableOptionAutoextendSizeContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionAutoIncrement.
	VisitTableOptionAutoIncrement(ctx *TableOptionAutoIncrementContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionAverage.
	VisitTableOptionAverage(ctx *TableOptionAverageContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionCharset.
	VisitTableOptionCharset(ctx *TableOptionCharsetContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionChecksum.
	VisitTableOptionChecksum(ctx *TableOptionChecksumContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionCollate.
	VisitTableOptionCollate(ctx *TableOptionCollateContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionComment.
	VisitTableOptionComment(ctx *TableOptionCommentContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionCompression.
	VisitTableOptionCompression(ctx *TableOptionCompressionContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionConnection.
	VisitTableOptionConnection(ctx *TableOptionConnectionContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionDataDirectory.
	VisitTableOptionDataDirectory(ctx *TableOptionDataDirectoryContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionDelay.
	VisitTableOptionDelay(ctx *TableOptionDelayContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionEncryption.
	VisitTableOptionEncryption(ctx *TableOptionEncryptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionPageCompressed.
	VisitTableOptionPageCompressed(ctx *TableOptionPageCompressedContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionPageCompressionLevel.
	VisitTableOptionPageCompressionLevel(ctx *TableOptionPageCompressionLevelContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionEncryptionKeyId.
	VisitTableOptionEncryptionKeyId(ctx *TableOptionEncryptionKeyIdContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionIndexDirectory.
	VisitTableOptionIndexDirectory(ctx *TableOptionIndexDirectoryContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionInsertMethod.
	VisitTableOptionInsertMethod(ctx *TableOptionInsertMethodContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionKeyBlockSize.
	VisitTableOptionKeyBlockSize(ctx *TableOptionKeyBlockSizeContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionMaxRows.
	VisitTableOptionMaxRows(ctx *TableOptionMaxRowsContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionMinRows.
	VisitTableOptionMinRows(ctx *TableOptionMinRowsContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionPackKeys.
	VisitTableOptionPackKeys(ctx *TableOptionPackKeysContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionPassword.
	VisitTableOptionPassword(ctx *TableOptionPasswordContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionRowFormat.
	VisitTableOptionRowFormat(ctx *TableOptionRowFormatContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionStartTransaction.
	VisitTableOptionStartTransaction(ctx *TableOptionStartTransactionContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionSecondaryEngineAttribute.
	VisitTableOptionSecondaryEngineAttribute(ctx *TableOptionSecondaryEngineAttributeContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionRecalculation.
	VisitTableOptionRecalculation(ctx *TableOptionRecalculationContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionPersistent.
	VisitTableOptionPersistent(ctx *TableOptionPersistentContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionSamplePage.
	VisitTableOptionSamplePage(ctx *TableOptionSamplePageContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionTablespace.
	VisitTableOptionTablespace(ctx *TableOptionTablespaceContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionTableType.
	VisitTableOptionTableType(ctx *TableOptionTableTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionTransactional.
	VisitTableOptionTransactional(ctx *TableOptionTransactionalContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableOptionUnion.
	VisitTableOptionUnion(ctx *TableOptionUnionContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableType.
	VisitTableType(ctx *TableTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#tablespaceStorage.
	VisitTablespaceStorage(ctx *TablespaceStorageContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionDefinitions.
	VisitPartitionDefinitions(ctx *PartitionDefinitionsContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionFunctionHash.
	VisitPartitionFunctionHash(ctx *PartitionFunctionHashContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionFunctionKey.
	VisitPartitionFunctionKey(ctx *PartitionFunctionKeyContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionFunctionRange.
	VisitPartitionFunctionRange(ctx *PartitionFunctionRangeContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionFunctionList.
	VisitPartitionFunctionList(ctx *PartitionFunctionListContext) interface{}

	// Visit a parse tree produced by MySqlParser#subPartitionFunctionHash.
	VisitSubPartitionFunctionHash(ctx *SubPartitionFunctionHashContext) interface{}

	// Visit a parse tree produced by MySqlParser#subPartitionFunctionKey.
	VisitSubPartitionFunctionKey(ctx *SubPartitionFunctionKeyContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionComparison.
	VisitPartitionComparison(ctx *PartitionComparisonContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionListAtom.
	VisitPartitionListAtom(ctx *PartitionListAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionListVector.
	VisitPartitionListVector(ctx *PartitionListVectorContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionSimple.
	VisitPartitionSimple(ctx *PartitionSimpleContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionDefinerAtom.
	VisitPartitionDefinerAtom(ctx *PartitionDefinerAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionDefinerVector.
	VisitPartitionDefinerVector(ctx *PartitionDefinerVectorContext) interface{}

	// Visit a parse tree produced by MySqlParser#subpartitionDefinition.
	VisitSubpartitionDefinition(ctx *SubpartitionDefinitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionOptionEngine.
	VisitPartitionOptionEngine(ctx *PartitionOptionEngineContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionOptionComment.
	VisitPartitionOptionComment(ctx *PartitionOptionCommentContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionOptionDataDirectory.
	VisitPartitionOptionDataDirectory(ctx *PartitionOptionDataDirectoryContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionOptionIndexDirectory.
	VisitPartitionOptionIndexDirectory(ctx *PartitionOptionIndexDirectoryContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionOptionMaxRows.
	VisitPartitionOptionMaxRows(ctx *PartitionOptionMaxRowsContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionOptionMinRows.
	VisitPartitionOptionMinRows(ctx *PartitionOptionMinRowsContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionOptionTablespace.
	VisitPartitionOptionTablespace(ctx *PartitionOptionTablespaceContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionOptionNodeGroup.
	VisitPartitionOptionNodeGroup(ctx *PartitionOptionNodeGroupContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterSimpleDatabase.
	VisitAlterSimpleDatabase(ctx *AlterSimpleDatabaseContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterUpgradeName.
	VisitAlterUpgradeName(ctx *AlterUpgradeNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterEvent.
	VisitAlterEvent(ctx *AlterEventContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterFunction.
	VisitAlterFunction(ctx *AlterFunctionContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterInstance.
	VisitAlterInstance(ctx *AlterInstanceContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterLogfileGroup.
	VisitAlterLogfileGroup(ctx *AlterLogfileGroupContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterProcedure.
	VisitAlterProcedure(ctx *AlterProcedureContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterServer.
	VisitAlterServer(ctx *AlterServerContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterTable.
	VisitAlterTable(ctx *AlterTableContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterTablespace.
	VisitAlterTablespace(ctx *AlterTablespaceContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterView.
	VisitAlterView(ctx *AlterViewContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByTableOption.
	VisitAlterByTableOption(ctx *AlterByTableOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByAddColumn.
	VisitAlterByAddColumn(ctx *AlterByAddColumnContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByAddColumns.
	VisitAlterByAddColumns(ctx *AlterByAddColumnsContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByAddIndex.
	VisitAlterByAddIndex(ctx *AlterByAddIndexContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByAddPrimaryKey.
	VisitAlterByAddPrimaryKey(ctx *AlterByAddPrimaryKeyContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByAddUniqueKey.
	VisitAlterByAddUniqueKey(ctx *AlterByAddUniqueKeyContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByAddSpecialIndex.
	VisitAlterByAddSpecialIndex(ctx *AlterByAddSpecialIndexContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByAddForeignKey.
	VisitAlterByAddForeignKey(ctx *AlterByAddForeignKeyContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByAddCheckTableConstraint.
	VisitAlterByAddCheckTableConstraint(ctx *AlterByAddCheckTableConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByAlterCheckTableConstraint.
	VisitAlterByAlterCheckTableConstraint(ctx *AlterByAlterCheckTableConstraintContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterBySetAlgorithm.
	VisitAlterBySetAlgorithm(ctx *AlterBySetAlgorithmContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByChangeDefault.
	VisitAlterByChangeDefault(ctx *AlterByChangeDefaultContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByChangeColumn.
	VisitAlterByChangeColumn(ctx *AlterByChangeColumnContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByRenameColumn.
	VisitAlterByRenameColumn(ctx *AlterByRenameColumnContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByLock.
	VisitAlterByLock(ctx *AlterByLockContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByModifyColumn.
	VisitAlterByModifyColumn(ctx *AlterByModifyColumnContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByDropColumn.
	VisitAlterByDropColumn(ctx *AlterByDropColumnContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByDropConstraintCheck.
	VisitAlterByDropConstraintCheck(ctx *AlterByDropConstraintCheckContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByDropPrimaryKey.
	VisitAlterByDropPrimaryKey(ctx *AlterByDropPrimaryKeyContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByDropIndex.
	VisitAlterByDropIndex(ctx *AlterByDropIndexContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByRenameIndex.
	VisitAlterByRenameIndex(ctx *AlterByRenameIndexContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByAlterColumnDefault.
	VisitAlterByAlterColumnDefault(ctx *AlterByAlterColumnDefaultContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByAlterIndexVisibility.
	VisitAlterByAlterIndexVisibility(ctx *AlterByAlterIndexVisibilityContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByDropForeignKey.
	VisitAlterByDropForeignKey(ctx *AlterByDropForeignKeyContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByDisableKeys.
	VisitAlterByDisableKeys(ctx *AlterByDisableKeysContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByEnableKeys.
	VisitAlterByEnableKeys(ctx *AlterByEnableKeysContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByRename.
	VisitAlterByRename(ctx *AlterByRenameContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByOrder.
	VisitAlterByOrder(ctx *AlterByOrderContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByConvertCharset.
	VisitAlterByConvertCharset(ctx *AlterByConvertCharsetContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByDefaultCharset.
	VisitAlterByDefaultCharset(ctx *AlterByDefaultCharsetContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByDiscardTablespace.
	VisitAlterByDiscardTablespace(ctx *AlterByDiscardTablespaceContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByImportTablespace.
	VisitAlterByImportTablespace(ctx *AlterByImportTablespaceContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByForce.
	VisitAlterByForce(ctx *AlterByForceContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByValidate.
	VisitAlterByValidate(ctx *AlterByValidateContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByAddDefinitions.
	VisitAlterByAddDefinitions(ctx *AlterByAddDefinitionsContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterPartition.
	VisitAlterPartition(ctx *AlterPartitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByAddPartition.
	VisitAlterByAddPartition(ctx *AlterByAddPartitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByDropPartition.
	VisitAlterByDropPartition(ctx *AlterByDropPartitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByDiscardPartition.
	VisitAlterByDiscardPartition(ctx *AlterByDiscardPartitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByImportPartition.
	VisitAlterByImportPartition(ctx *AlterByImportPartitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByTruncatePartition.
	VisitAlterByTruncatePartition(ctx *AlterByTruncatePartitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByCoalescePartition.
	VisitAlterByCoalescePartition(ctx *AlterByCoalescePartitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByReorganizePartition.
	VisitAlterByReorganizePartition(ctx *AlterByReorganizePartitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByExchangePartition.
	VisitAlterByExchangePartition(ctx *AlterByExchangePartitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByAnalyzePartition.
	VisitAlterByAnalyzePartition(ctx *AlterByAnalyzePartitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByCheckPartition.
	VisitAlterByCheckPartition(ctx *AlterByCheckPartitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByOptimizePartition.
	VisitAlterByOptimizePartition(ctx *AlterByOptimizePartitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByRebuildPartition.
	VisitAlterByRebuildPartition(ctx *AlterByRebuildPartitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByRepairPartition.
	VisitAlterByRepairPartition(ctx *AlterByRepairPartitionContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByRemovePartitioning.
	VisitAlterByRemovePartitioning(ctx *AlterByRemovePartitioningContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterByUpgradePartitioning.
	VisitAlterByUpgradePartitioning(ctx *AlterByUpgradePartitioningContext) interface{}

	// Visit a parse tree produced by MySqlParser#dropDatabase.
	VisitDropDatabase(ctx *DropDatabaseContext) interface{}

	// Visit a parse tree produced by MySqlParser#dropEvent.
	VisitDropEvent(ctx *DropEventContext) interface{}

	// Visit a parse tree produced by MySqlParser#dropIndex.
	VisitDropIndex(ctx *DropIndexContext) interface{}

	// Visit a parse tree produced by MySqlParser#dropLogfileGroup.
	VisitDropLogfileGroup(ctx *DropLogfileGroupContext) interface{}

	// Visit a parse tree produced by MySqlParser#dropProcedure.
	VisitDropProcedure(ctx *DropProcedureContext) interface{}

	// Visit a parse tree produced by MySqlParser#dropFunction.
	VisitDropFunction(ctx *DropFunctionContext) interface{}

	// Visit a parse tree produced by MySqlParser#dropServer.
	VisitDropServer(ctx *DropServerContext) interface{}

	// Visit a parse tree produced by MySqlParser#dropTable.
	VisitDropTable(ctx *DropTableContext) interface{}

	// Visit a parse tree produced by MySqlParser#dropTablespace.
	VisitDropTablespace(ctx *DropTablespaceContext) interface{}

	// Visit a parse tree produced by MySqlParser#dropTrigger.
	VisitDropTrigger(ctx *DropTriggerContext) interface{}

	// Visit a parse tree produced by MySqlParser#dropView.
	VisitDropView(ctx *DropViewContext) interface{}

	// Visit a parse tree produced by MySqlParser#dropRole.
	VisitDropRole(ctx *DropRoleContext) interface{}

	// Visit a parse tree produced by MySqlParser#setRole.
	VisitSetRole(ctx *SetRoleContext) interface{}

	// Visit a parse tree produced by MySqlParser#renameTable.
	VisitRenameTable(ctx *RenameTableContext) interface{}

	// Visit a parse tree produced by MySqlParser#renameTableClause.
	VisitRenameTableClause(ctx *RenameTableClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#truncateTable.
	VisitTruncateTable(ctx *TruncateTableContext) interface{}

	// Visit a parse tree produced by MySqlParser#callStatement.
	VisitCallStatement(ctx *CallStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#deleteStatement.
	VisitDeleteStatement(ctx *DeleteStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#doStatement.
	VisitDoStatement(ctx *DoStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#handlerStatement.
	VisitHandlerStatement(ctx *HandlerStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#insertStatement.
	VisitInsertStatement(ctx *InsertStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#loadDataStatement.
	VisitLoadDataStatement(ctx *LoadDataStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#loadXmlStatement.
	VisitLoadXmlStatement(ctx *LoadXmlStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#replaceStatement.
	VisitReplaceStatement(ctx *ReplaceStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#simpleSelect.
	VisitSimpleSelect(ctx *SimpleSelectContext) interface{}

	// Visit a parse tree produced by MySqlParser#parenthesisSelect.
	VisitParenthesisSelect(ctx *ParenthesisSelectContext) interface{}

	// Visit a parse tree produced by MySqlParser#unionSelect.
	VisitUnionSelect(ctx *UnionSelectContext) interface{}

	// Visit a parse tree produced by MySqlParser#unionParenthesisSelect.
	VisitUnionParenthesisSelect(ctx *UnionParenthesisSelectContext) interface{}

	// Visit a parse tree produced by MySqlParser#withLateralStatement.
	VisitWithLateralStatement(ctx *WithLateralStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#updateStatement.
	VisitUpdateStatement(ctx *UpdateStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#valuesStatement.
	VisitValuesStatement(ctx *ValuesStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#insertStatementValue.
	VisitInsertStatementValue(ctx *InsertStatementValueContext) interface{}

	// Visit a parse tree produced by MySqlParser#updatedElement.
	VisitUpdatedElement(ctx *UpdatedElementContext) interface{}

	// Visit a parse tree produced by MySqlParser#assignmentField.
	VisitAssignmentField(ctx *AssignmentFieldContext) interface{}

	// Visit a parse tree produced by MySqlParser#lockClause.
	VisitLockClause(ctx *LockClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#singleDeleteStatement.
	VisitSingleDeleteStatement(ctx *SingleDeleteStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#multipleDeleteStatement.
	VisitMultipleDeleteStatement(ctx *MultipleDeleteStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#handlerOpenStatement.
	VisitHandlerOpenStatement(ctx *HandlerOpenStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#handlerReadIndexStatement.
	VisitHandlerReadIndexStatement(ctx *HandlerReadIndexStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#handlerReadStatement.
	VisitHandlerReadStatement(ctx *HandlerReadStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#handlerCloseStatement.
	VisitHandlerCloseStatement(ctx *HandlerCloseStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#singleUpdateStatement.
	VisitSingleUpdateStatement(ctx *SingleUpdateStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#multipleUpdateStatement.
	VisitMultipleUpdateStatement(ctx *MultipleUpdateStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#orderByClause.
	VisitOrderByClause(ctx *OrderByClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#orderByExpression.
	VisitOrderByExpression(ctx *OrderByExpressionContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableSources.
	VisitTableSources(ctx *TableSourcesContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableSourceBase.
	VisitTableSourceBase(ctx *TableSourceBaseContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableSourceNested.
	VisitTableSourceNested(ctx *TableSourceNestedContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableJson.
	VisitTableJson(ctx *TableJsonContext) interface{}

	// Visit a parse tree produced by MySqlParser#atomTableItem.
	VisitAtomTableItem(ctx *AtomTableItemContext) interface{}

	// Visit a parse tree produced by MySqlParser#subqueryTableItem.
	VisitSubqueryTableItem(ctx *SubqueryTableItemContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableSourcesItem.
	VisitTableSourcesItem(ctx *TableSourcesItemContext) interface{}

	// Visit a parse tree produced by MySqlParser#indexHint.
	VisitIndexHint(ctx *IndexHintContext) interface{}

	// Visit a parse tree produced by MySqlParser#indexHintType.
	VisitIndexHintType(ctx *IndexHintTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#innerJoin.
	VisitInnerJoin(ctx *InnerJoinContext) interface{}

	// Visit a parse tree produced by MySqlParser#straightJoin.
	VisitStraightJoin(ctx *StraightJoinContext) interface{}

	// Visit a parse tree produced by MySqlParser#outerJoin.
	VisitOuterJoin(ctx *OuterJoinContext) interface{}

	// Visit a parse tree produced by MySqlParser#naturalJoin.
	VisitNaturalJoin(ctx *NaturalJoinContext) interface{}

	// Visit a parse tree produced by MySqlParser#joinSpec.
	VisitJoinSpec(ctx *JoinSpecContext) interface{}

	// Visit a parse tree produced by MySqlParser#queryExpression.
	VisitQueryExpression(ctx *QueryExpressionContext) interface{}

	// Visit a parse tree produced by MySqlParser#queryExpressionNointo.
	VisitQueryExpressionNointo(ctx *QueryExpressionNointoContext) interface{}

	// Visit a parse tree produced by MySqlParser#querySpecification.
	VisitQuerySpecification(ctx *QuerySpecificationContext) interface{}

	// Visit a parse tree produced by MySqlParser#querySpecificationNointo.
	VisitQuerySpecificationNointo(ctx *QuerySpecificationNointoContext) interface{}

	// Visit a parse tree produced by MySqlParser#unionParenthesis.
	VisitUnionParenthesis(ctx *UnionParenthesisContext) interface{}

	// Visit a parse tree produced by MySqlParser#unionStatement.
	VisitUnionStatement(ctx *UnionStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#lateralStatement.
	VisitLateralStatement(ctx *LateralStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#jsonTable.
	VisitJsonTable(ctx *JsonTableContext) interface{}

	// Visit a parse tree produced by MySqlParser#jsonColumnList.
	VisitJsonColumnList(ctx *JsonColumnListContext) interface{}

	// Visit a parse tree produced by MySqlParser#jsonColumn.
	VisitJsonColumn(ctx *JsonColumnContext) interface{}

	// Visit a parse tree produced by MySqlParser#jsonOnEmpty.
	VisitJsonOnEmpty(ctx *JsonOnEmptyContext) interface{}

	// Visit a parse tree produced by MySqlParser#jsonOnError.
	VisitJsonOnError(ctx *JsonOnErrorContext) interface{}

	// Visit a parse tree produced by MySqlParser#selectSpec.
	VisitSelectSpec(ctx *SelectSpecContext) interface{}

	// Visit a parse tree produced by MySqlParser#selectElements.
	VisitSelectElements(ctx *SelectElementsContext) interface{}

	// Visit a parse tree produced by MySqlParser#selectStarElement.
	VisitSelectStarElement(ctx *SelectStarElementContext) interface{}

	// Visit a parse tree produced by MySqlParser#selectColumnElement.
	VisitSelectColumnElement(ctx *SelectColumnElementContext) interface{}

	// Visit a parse tree produced by MySqlParser#selectFunctionElement.
	VisitSelectFunctionElement(ctx *SelectFunctionElementContext) interface{}

	// Visit a parse tree produced by MySqlParser#selectExpressionElement.
	VisitSelectExpressionElement(ctx *SelectExpressionElementContext) interface{}

	// Visit a parse tree produced by MySqlParser#selectIntoVariables.
	VisitSelectIntoVariables(ctx *SelectIntoVariablesContext) interface{}

	// Visit a parse tree produced by MySqlParser#selectIntoDumpFile.
	VisitSelectIntoDumpFile(ctx *SelectIntoDumpFileContext) interface{}

	// Visit a parse tree produced by MySqlParser#selectIntoTextFile.
	VisitSelectIntoTextFile(ctx *SelectIntoTextFileContext) interface{}

	// Visit a parse tree produced by MySqlParser#selectFieldsInto.
	VisitSelectFieldsInto(ctx *SelectFieldsIntoContext) interface{}

	// Visit a parse tree produced by MySqlParser#selectLinesInto.
	VisitSelectLinesInto(ctx *SelectLinesIntoContext) interface{}

	// Visit a parse tree produced by MySqlParser#fromClause.
	VisitFromClause(ctx *FromClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#groupByClause.
	VisitGroupByClause(ctx *GroupByClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#havingClause.
	VisitHavingClause(ctx *HavingClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#windowClause.
	VisitWindowClause(ctx *WindowClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#groupByItem.
	VisitGroupByItem(ctx *GroupByItemContext) interface{}

	// Visit a parse tree produced by MySqlParser#limitClause.
	VisitLimitClause(ctx *LimitClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#limitClauseAtom.
	VisitLimitClauseAtom(ctx *LimitClauseAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#startTransaction.
	VisitStartTransaction(ctx *StartTransactionContext) interface{}

	// Visit a parse tree produced by MySqlParser#beginWork.
	VisitBeginWork(ctx *BeginWorkContext) interface{}

	// Visit a parse tree produced by MySqlParser#commitWork.
	VisitCommitWork(ctx *CommitWorkContext) interface{}

	// Visit a parse tree produced by MySqlParser#rollbackWork.
	VisitRollbackWork(ctx *RollbackWorkContext) interface{}

	// Visit a parse tree produced by MySqlParser#savepointStatement.
	VisitSavepointStatement(ctx *SavepointStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#rollbackStatement.
	VisitRollbackStatement(ctx *RollbackStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#releaseStatement.
	VisitReleaseStatement(ctx *ReleaseStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#lockTables.
	VisitLockTables(ctx *LockTablesContext) interface{}

	// Visit a parse tree produced by MySqlParser#unlockTables.
	VisitUnlockTables(ctx *UnlockTablesContext) interface{}

	// Visit a parse tree produced by MySqlParser#setAutocommitStatement.
	VisitSetAutocommitStatement(ctx *SetAutocommitStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#setTransactionStatement.
	VisitSetTransactionStatement(ctx *SetTransactionStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#transactionMode.
	VisitTransactionMode(ctx *TransactionModeContext) interface{}

	// Visit a parse tree produced by MySqlParser#lockTableElement.
	VisitLockTableElement(ctx *LockTableElementContext) interface{}

	// Visit a parse tree produced by MySqlParser#lockAction.
	VisitLockAction(ctx *LockActionContext) interface{}

	// Visit a parse tree produced by MySqlParser#transactionOption.
	VisitTransactionOption(ctx *TransactionOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#transactionLevel.
	VisitTransactionLevel(ctx *TransactionLevelContext) interface{}

	// Visit a parse tree produced by MySqlParser#changeMaster.
	VisitChangeMaster(ctx *ChangeMasterContext) interface{}

	// Visit a parse tree produced by MySqlParser#changeReplicationFilter.
	VisitChangeReplicationFilter(ctx *ChangeReplicationFilterContext) interface{}

	// Visit a parse tree produced by MySqlParser#purgeBinaryLogs.
	VisitPurgeBinaryLogs(ctx *PurgeBinaryLogsContext) interface{}

	// Visit a parse tree produced by MySqlParser#resetMaster.
	VisitResetMaster(ctx *ResetMasterContext) interface{}

	// Visit a parse tree produced by MySqlParser#resetSlave.
	VisitResetSlave(ctx *ResetSlaveContext) interface{}

	// Visit a parse tree produced by MySqlParser#startSlave.
	VisitStartSlave(ctx *StartSlaveContext) interface{}

	// Visit a parse tree produced by MySqlParser#stopSlave.
	VisitStopSlave(ctx *StopSlaveContext) interface{}

	// Visit a parse tree produced by MySqlParser#startGroupReplication.
	VisitStartGroupReplication(ctx *StartGroupReplicationContext) interface{}

	// Visit a parse tree produced by MySqlParser#stopGroupReplication.
	VisitStopGroupReplication(ctx *StopGroupReplicationContext) interface{}

	// Visit a parse tree produced by MySqlParser#masterStringOption.
	VisitMasterStringOption(ctx *MasterStringOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#masterDecimalOption.
	VisitMasterDecimalOption(ctx *MasterDecimalOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#masterBoolOption.
	VisitMasterBoolOption(ctx *MasterBoolOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#masterRealOption.
	VisitMasterRealOption(ctx *MasterRealOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#masterUidListOption.
	VisitMasterUidListOption(ctx *MasterUidListOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#stringMasterOption.
	VisitStringMasterOption(ctx *StringMasterOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#decimalMasterOption.
	VisitDecimalMasterOption(ctx *DecimalMasterOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#boolMasterOption.
	VisitBoolMasterOption(ctx *BoolMasterOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#channelOption.
	VisitChannelOption(ctx *ChannelOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#doDbReplication.
	VisitDoDbReplication(ctx *DoDbReplicationContext) interface{}

	// Visit a parse tree produced by MySqlParser#ignoreDbReplication.
	VisitIgnoreDbReplication(ctx *IgnoreDbReplicationContext) interface{}

	// Visit a parse tree produced by MySqlParser#doTableReplication.
	VisitDoTableReplication(ctx *DoTableReplicationContext) interface{}

	// Visit a parse tree produced by MySqlParser#ignoreTableReplication.
	VisitIgnoreTableReplication(ctx *IgnoreTableReplicationContext) interface{}

	// Visit a parse tree produced by MySqlParser#wildDoTableReplication.
	VisitWildDoTableReplication(ctx *WildDoTableReplicationContext) interface{}

	// Visit a parse tree produced by MySqlParser#wildIgnoreTableReplication.
	VisitWildIgnoreTableReplication(ctx *WildIgnoreTableReplicationContext) interface{}

	// Visit a parse tree produced by MySqlParser#rewriteDbReplication.
	VisitRewriteDbReplication(ctx *RewriteDbReplicationContext) interface{}

	// Visit a parse tree produced by MySqlParser#tablePair.
	VisitTablePair(ctx *TablePairContext) interface{}

	// Visit a parse tree produced by MySqlParser#threadType.
	VisitThreadType(ctx *ThreadTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#gtidsUntilOption.
	VisitGtidsUntilOption(ctx *GtidsUntilOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#masterLogUntilOption.
	VisitMasterLogUntilOption(ctx *MasterLogUntilOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#relayLogUntilOption.
	VisitRelayLogUntilOption(ctx *RelayLogUntilOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#sqlGapsUntilOption.
	VisitSqlGapsUntilOption(ctx *SqlGapsUntilOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#userConnectionOption.
	VisitUserConnectionOption(ctx *UserConnectionOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#passwordConnectionOption.
	VisitPasswordConnectionOption(ctx *PasswordConnectionOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#defaultAuthConnectionOption.
	VisitDefaultAuthConnectionOption(ctx *DefaultAuthConnectionOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#pluginDirConnectionOption.
	VisitPluginDirConnectionOption(ctx *PluginDirConnectionOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#gtuidSet.
	VisitGtuidSet(ctx *GtuidSetContext) interface{}

	// Visit a parse tree produced by MySqlParser#xaStartTransaction.
	VisitXaStartTransaction(ctx *XaStartTransactionContext) interface{}

	// Visit a parse tree produced by MySqlParser#xaEndTransaction.
	VisitXaEndTransaction(ctx *XaEndTransactionContext) interface{}

	// Visit a parse tree produced by MySqlParser#xaPrepareStatement.
	VisitXaPrepareStatement(ctx *XaPrepareStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#xaCommitWork.
	VisitXaCommitWork(ctx *XaCommitWorkContext) interface{}

	// Visit a parse tree produced by MySqlParser#xaRollbackWork.
	VisitXaRollbackWork(ctx *XaRollbackWorkContext) interface{}

	// Visit a parse tree produced by MySqlParser#xaRecoverWork.
	VisitXaRecoverWork(ctx *XaRecoverWorkContext) interface{}

	// Visit a parse tree produced by MySqlParser#prepareStatement.
	VisitPrepareStatement(ctx *PrepareStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#executeStatement.
	VisitExecuteStatement(ctx *ExecuteStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#deallocatePrepare.
	VisitDeallocatePrepare(ctx *DeallocatePrepareContext) interface{}

	// Visit a parse tree produced by MySqlParser#routineBody.
	VisitRoutineBody(ctx *RoutineBodyContext) interface{}

	// Visit a parse tree produced by MySqlParser#blockStatement.
	VisitBlockStatement(ctx *BlockStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#caseStatement.
	VisitCaseStatement(ctx *CaseStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#ifStatement.
	VisitIfStatement(ctx *IfStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#iterateStatement.
	VisitIterateStatement(ctx *IterateStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#leaveStatement.
	VisitLeaveStatement(ctx *LeaveStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#loopStatement.
	VisitLoopStatement(ctx *LoopStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#repeatStatement.
	VisitRepeatStatement(ctx *RepeatStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#returnStatement.
	VisitReturnStatement(ctx *ReturnStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#whileStatement.
	VisitWhileStatement(ctx *WhileStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#CloseCursor.
	VisitCloseCursor(ctx *CloseCursorContext) interface{}

	// Visit a parse tree produced by MySqlParser#FetchCursor.
	VisitFetchCursor(ctx *FetchCursorContext) interface{}

	// Visit a parse tree produced by MySqlParser#OpenCursor.
	VisitOpenCursor(ctx *OpenCursorContext) interface{}

	// Visit a parse tree produced by MySqlParser#declareVariable.
	VisitDeclareVariable(ctx *DeclareVariableContext) interface{}

	// Visit a parse tree produced by MySqlParser#declareCondition.
	VisitDeclareCondition(ctx *DeclareConditionContext) interface{}

	// Visit a parse tree produced by MySqlParser#declareCursor.
	VisitDeclareCursor(ctx *DeclareCursorContext) interface{}

	// Visit a parse tree produced by MySqlParser#declareHandler.
	VisitDeclareHandler(ctx *DeclareHandlerContext) interface{}

	// Visit a parse tree produced by MySqlParser#handlerConditionCode.
	VisitHandlerConditionCode(ctx *HandlerConditionCodeContext) interface{}

	// Visit a parse tree produced by MySqlParser#handlerConditionState.
	VisitHandlerConditionState(ctx *HandlerConditionStateContext) interface{}

	// Visit a parse tree produced by MySqlParser#handlerConditionName.
	VisitHandlerConditionName(ctx *HandlerConditionNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#handlerConditionWarning.
	VisitHandlerConditionWarning(ctx *HandlerConditionWarningContext) interface{}

	// Visit a parse tree produced by MySqlParser#handlerConditionNotfound.
	VisitHandlerConditionNotfound(ctx *HandlerConditionNotfoundContext) interface{}

	// Visit a parse tree produced by MySqlParser#handlerConditionException.
	VisitHandlerConditionException(ctx *HandlerConditionExceptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#procedureSqlStatement.
	VisitProcedureSqlStatement(ctx *ProcedureSqlStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#caseAlternative.
	VisitCaseAlternative(ctx *CaseAlternativeContext) interface{}

	// Visit a parse tree produced by MySqlParser#elifAlternative.
	VisitElifAlternative(ctx *ElifAlternativeContext) interface{}

	// Visit a parse tree produced by MySqlParser#alterUserMysqlV56.
	VisitAlterUserMysqlV56(ctx *AlterUserMysqlV56Context) interface{}

	// Visit a parse tree produced by MySqlParser#alterUserMysqlV80.
	VisitAlterUserMysqlV80(ctx *AlterUserMysqlV80Context) interface{}

	// Visit a parse tree produced by MySqlParser#createUserMysqlV56.
	VisitCreateUserMysqlV56(ctx *CreateUserMysqlV56Context) interface{}

	// Visit a parse tree produced by MySqlParser#createUserMysqlV80.
	VisitCreateUserMysqlV80(ctx *CreateUserMysqlV80Context) interface{}

	// Visit a parse tree produced by MySqlParser#dropUser.
	VisitDropUser(ctx *DropUserContext) interface{}

	// Visit a parse tree produced by MySqlParser#grantStatement.
	VisitGrantStatement(ctx *GrantStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#roleOption.
	VisitRoleOption(ctx *RoleOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#grantProxy.
	VisitGrantProxy(ctx *GrantProxyContext) interface{}

	// Visit a parse tree produced by MySqlParser#renameUser.
	VisitRenameUser(ctx *RenameUserContext) interface{}

	// Visit a parse tree produced by MySqlParser#detailRevoke.
	VisitDetailRevoke(ctx *DetailRevokeContext) interface{}

	// Visit a parse tree produced by MySqlParser#shortRevoke.
	VisitShortRevoke(ctx *ShortRevokeContext) interface{}

	// Visit a parse tree produced by MySqlParser#roleRevoke.
	VisitRoleRevoke(ctx *RoleRevokeContext) interface{}

	// Visit a parse tree produced by MySqlParser#revokeProxy.
	VisitRevokeProxy(ctx *RevokeProxyContext) interface{}

	// Visit a parse tree produced by MySqlParser#setPasswordStatement.
	VisitSetPasswordStatement(ctx *SetPasswordStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#userSpecification.
	VisitUserSpecification(ctx *UserSpecificationContext) interface{}

	// Visit a parse tree produced by MySqlParser#hashAuthOption.
	VisitHashAuthOption(ctx *HashAuthOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#randomAuthOption.
	VisitRandomAuthOption(ctx *RandomAuthOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#stringAuthOption.
	VisitStringAuthOption(ctx *StringAuthOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#moduleAuthOption.
	VisitModuleAuthOption(ctx *ModuleAuthOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#simpleAuthOption.
	VisitSimpleAuthOption(ctx *SimpleAuthOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#authOptionClause.
	VisitAuthOptionClause(ctx *AuthOptionClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#module.
	VisitModule(ctx *ModuleContext) interface{}

	// Visit a parse tree produced by MySqlParser#passwordModuleOption.
	VisitPasswordModuleOption(ctx *PasswordModuleOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#tlsOption.
	VisitTlsOption(ctx *TlsOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#userResourceOption.
	VisitUserResourceOption(ctx *UserResourceOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#userPasswordOption.
	VisitUserPasswordOption(ctx *UserPasswordOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#userLockOption.
	VisitUserLockOption(ctx *UserLockOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#privelegeClause.
	VisitPrivelegeClause(ctx *PrivelegeClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#privilege.
	VisitPrivilege(ctx *PrivilegeContext) interface{}

	// Visit a parse tree produced by MySqlParser#currentSchemaPriviLevel.
	VisitCurrentSchemaPriviLevel(ctx *CurrentSchemaPriviLevelContext) interface{}

	// Visit a parse tree produced by MySqlParser#globalPrivLevel.
	VisitGlobalPrivLevel(ctx *GlobalPrivLevelContext) interface{}

	// Visit a parse tree produced by MySqlParser#definiteSchemaPrivLevel.
	VisitDefiniteSchemaPrivLevel(ctx *DefiniteSchemaPrivLevelContext) interface{}

	// Visit a parse tree produced by MySqlParser#definiteFullTablePrivLevel.
	VisitDefiniteFullTablePrivLevel(ctx *DefiniteFullTablePrivLevelContext) interface{}

	// Visit a parse tree produced by MySqlParser#definiteFullTablePrivLevel2.
	VisitDefiniteFullTablePrivLevel2(ctx *DefiniteFullTablePrivLevel2Context) interface{}

	// Visit a parse tree produced by MySqlParser#definiteTablePrivLevel.
	VisitDefiniteTablePrivLevel(ctx *DefiniteTablePrivLevelContext) interface{}

	// Visit a parse tree produced by MySqlParser#renameUserClause.
	VisitRenameUserClause(ctx *RenameUserClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#analyzeTable.
	VisitAnalyzeTable(ctx *AnalyzeTableContext) interface{}

	// Visit a parse tree produced by MySqlParser#checkTable.
	VisitCheckTable(ctx *CheckTableContext) interface{}

	// Visit a parse tree produced by MySqlParser#checksumTable.
	VisitChecksumTable(ctx *ChecksumTableContext) interface{}

	// Visit a parse tree produced by MySqlParser#optimizeTable.
	VisitOptimizeTable(ctx *OptimizeTableContext) interface{}

	// Visit a parse tree produced by MySqlParser#repairTable.
	VisitRepairTable(ctx *RepairTableContext) interface{}

	// Visit a parse tree produced by MySqlParser#checkTableOption.
	VisitCheckTableOption(ctx *CheckTableOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#createUdfunction.
	VisitCreateUdfunction(ctx *CreateUdfunctionContext) interface{}

	// Visit a parse tree produced by MySqlParser#installPlugin.
	VisitInstallPlugin(ctx *InstallPluginContext) interface{}

	// Visit a parse tree produced by MySqlParser#uninstallPlugin.
	VisitUninstallPlugin(ctx *UninstallPluginContext) interface{}

	// Visit a parse tree produced by MySqlParser#setVariable.
	VisitSetVariable(ctx *SetVariableContext) interface{}

	// Visit a parse tree produced by MySqlParser#setCharset.
	VisitSetCharset(ctx *SetCharsetContext) interface{}

	// Visit a parse tree produced by MySqlParser#setNames.
	VisitSetNames(ctx *SetNamesContext) interface{}

	// Visit a parse tree produced by MySqlParser#setPassword.
	VisitSetPassword(ctx *SetPasswordContext) interface{}

	// Visit a parse tree produced by MySqlParser#setTransaction.
	VisitSetTransaction(ctx *SetTransactionContext) interface{}

	// Visit a parse tree produced by MySqlParser#setAutocommit.
	VisitSetAutocommit(ctx *SetAutocommitContext) interface{}

	// Visit a parse tree produced by MySqlParser#setNewValueInsideTrigger.
	VisitSetNewValueInsideTrigger(ctx *SetNewValueInsideTriggerContext) interface{}

	// Visit a parse tree produced by MySqlParser#showMasterLogs.
	VisitShowMasterLogs(ctx *ShowMasterLogsContext) interface{}

	// Visit a parse tree produced by MySqlParser#showLogEvents.
	VisitShowLogEvents(ctx *ShowLogEventsContext) interface{}

	// Visit a parse tree produced by MySqlParser#showObjectFilter.
	VisitShowObjectFilter(ctx *ShowObjectFilterContext) interface{}

	// Visit a parse tree produced by MySqlParser#showColumns.
	VisitShowColumns(ctx *ShowColumnsContext) interface{}

	// Visit a parse tree produced by MySqlParser#showCreateDb.
	VisitShowCreateDb(ctx *ShowCreateDbContext) interface{}

	// Visit a parse tree produced by MySqlParser#showCreateFullIdObject.
	VisitShowCreateFullIdObject(ctx *ShowCreateFullIdObjectContext) interface{}

	// Visit a parse tree produced by MySqlParser#showCreateUser.
	VisitShowCreateUser(ctx *ShowCreateUserContext) interface{}

	// Visit a parse tree produced by MySqlParser#showEngine.
	VisitShowEngine(ctx *ShowEngineContext) interface{}

	// Visit a parse tree produced by MySqlParser#showGlobalInfo.
	VisitShowGlobalInfo(ctx *ShowGlobalInfoContext) interface{}

	// Visit a parse tree produced by MySqlParser#showErrors.
	VisitShowErrors(ctx *ShowErrorsContext) interface{}

	// Visit a parse tree produced by MySqlParser#showCountErrors.
	VisitShowCountErrors(ctx *ShowCountErrorsContext) interface{}

	// Visit a parse tree produced by MySqlParser#showSchemaFilter.
	VisitShowSchemaFilter(ctx *ShowSchemaFilterContext) interface{}

	// Visit a parse tree produced by MySqlParser#showRoutine.
	VisitShowRoutine(ctx *ShowRoutineContext) interface{}

	// Visit a parse tree produced by MySqlParser#showGrants.
	VisitShowGrants(ctx *ShowGrantsContext) interface{}

	// Visit a parse tree produced by MySqlParser#showIndexes.
	VisitShowIndexes(ctx *ShowIndexesContext) interface{}

	// Visit a parse tree produced by MySqlParser#showOpenTables.
	VisitShowOpenTables(ctx *ShowOpenTablesContext) interface{}

	// Visit a parse tree produced by MySqlParser#showProfile.
	VisitShowProfile(ctx *ShowProfileContext) interface{}

	// Visit a parse tree produced by MySqlParser#showSlaveStatus.
	VisitShowSlaveStatus(ctx *ShowSlaveStatusContext) interface{}

	// Visit a parse tree produced by MySqlParser#variableClause.
	VisitVariableClause(ctx *VariableClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#showCommonEntity.
	VisitShowCommonEntity(ctx *ShowCommonEntityContext) interface{}

	// Visit a parse tree produced by MySqlParser#showFilter.
	VisitShowFilter(ctx *ShowFilterContext) interface{}

	// Visit a parse tree produced by MySqlParser#showGlobalInfoClause.
	VisitShowGlobalInfoClause(ctx *ShowGlobalInfoClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#showSchemaEntity.
	VisitShowSchemaEntity(ctx *ShowSchemaEntityContext) interface{}

	// Visit a parse tree produced by MySqlParser#showProfileType.
	VisitShowProfileType(ctx *ShowProfileTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#binlogStatement.
	VisitBinlogStatement(ctx *BinlogStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#cacheIndexStatement.
	VisitCacheIndexStatement(ctx *CacheIndexStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#flushStatement.
	VisitFlushStatement(ctx *FlushStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#killStatement.
	VisitKillStatement(ctx *KillStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#loadIndexIntoCache.
	VisitLoadIndexIntoCache(ctx *LoadIndexIntoCacheContext) interface{}

	// Visit a parse tree produced by MySqlParser#resetStatement.
	VisitResetStatement(ctx *ResetStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#shutdownStatement.
	VisitShutdownStatement(ctx *ShutdownStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableIndexes.
	VisitTableIndexes(ctx *TableIndexesContext) interface{}

	// Visit a parse tree produced by MySqlParser#simpleFlushOption.
	VisitSimpleFlushOption(ctx *SimpleFlushOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#channelFlushOption.
	VisitChannelFlushOption(ctx *ChannelFlushOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableFlushOption.
	VisitTableFlushOption(ctx *TableFlushOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#flushTableOption.
	VisitFlushTableOption(ctx *FlushTableOptionContext) interface{}

	// Visit a parse tree produced by MySqlParser#loadedTableIndexes.
	VisitLoadedTableIndexes(ctx *LoadedTableIndexesContext) interface{}

	// Visit a parse tree produced by MySqlParser#simpleDescribeStatement.
	VisitSimpleDescribeStatement(ctx *SimpleDescribeStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#fullDescribeStatement.
	VisitFullDescribeStatement(ctx *FullDescribeStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#helpStatement.
	VisitHelpStatement(ctx *HelpStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#useStatement.
	VisitUseStatement(ctx *UseStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#signalStatement.
	VisitSignalStatement(ctx *SignalStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#resignalStatement.
	VisitResignalStatement(ctx *ResignalStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#signalConditionInformation.
	VisitSignalConditionInformation(ctx *SignalConditionInformationContext) interface{}

	// Visit a parse tree produced by MySqlParser#withStatement.
	VisitWithStatement(ctx *WithStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableStatement.
	VisitTableStatement(ctx *TableStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#diagnosticsStatement.
	VisitDiagnosticsStatement(ctx *DiagnosticsStatementContext) interface{}

	// Visit a parse tree produced by MySqlParser#diagnosticsConditionInformationName.
	VisitDiagnosticsConditionInformationName(ctx *DiagnosticsConditionInformationNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#describeStatements.
	VisitDescribeStatements(ctx *DescribeStatementsContext) interface{}

	// Visit a parse tree produced by MySqlParser#describeConnection.
	VisitDescribeConnection(ctx *DescribeConnectionContext) interface{}

	// Visit a parse tree produced by MySqlParser#fullId.
	VisitFullId(ctx *FullIdContext) interface{}

	// Visit a parse tree produced by MySqlParser#tableName.
	VisitTableName(ctx *TableNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#roleName.
	VisitRoleName(ctx *RoleNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#fullColumnName.
	VisitFullColumnName(ctx *FullColumnNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#indexColumnName.
	VisitIndexColumnName(ctx *IndexColumnNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#simpleUserName.
	VisitSimpleUserName(ctx *SimpleUserNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#hostName.
	VisitHostName(ctx *HostNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#userName.
	VisitUserName(ctx *UserNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#mysqlVariable.
	VisitMysqlVariable(ctx *MysqlVariableContext) interface{}

	// Visit a parse tree produced by MySqlParser#charsetName.
	VisitCharsetName(ctx *CharsetNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#collationName.
	VisitCollationName(ctx *CollationNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#engineName.
	VisitEngineName(ctx *EngineNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#engineNameBase.
	VisitEngineNameBase(ctx *EngineNameBaseContext) interface{}

	// Visit a parse tree produced by MySqlParser#uuidSet.
	VisitUuidSet(ctx *UuidSetContext) interface{}

	// Visit a parse tree produced by MySqlParser#xid.
	VisitXid(ctx *XidContext) interface{}

	// Visit a parse tree produced by MySqlParser#xuidStringId.
	VisitXuidStringId(ctx *XuidStringIdContext) interface{}

	// Visit a parse tree produced by MySqlParser#authPlugin.
	VisitAuthPlugin(ctx *AuthPluginContext) interface{}

	// Visit a parse tree produced by MySqlParser#uid.
	VisitUid(ctx *UidContext) interface{}

	// Visit a parse tree produced by MySqlParser#simpleId.
	VisitSimpleId(ctx *SimpleIdContext) interface{}

	// Visit a parse tree produced by MySqlParser#dottedId.
	VisitDottedId(ctx *DottedIdContext) interface{}

	// Visit a parse tree produced by MySqlParser#decimalLiteral.
	VisitDecimalLiteral(ctx *DecimalLiteralContext) interface{}

	// Visit a parse tree produced by MySqlParser#fileSizeLiteral.
	VisitFileSizeLiteral(ctx *FileSizeLiteralContext) interface{}

	// Visit a parse tree produced by MySqlParser#stringLiteral.
	VisitStringLiteral(ctx *StringLiteralContext) interface{}

	// Visit a parse tree produced by MySqlParser#booleanLiteral.
	VisitBooleanLiteral(ctx *BooleanLiteralContext) interface{}

	// Visit a parse tree produced by MySqlParser#hexadecimalLiteral.
	VisitHexadecimalLiteral(ctx *HexadecimalLiteralContext) interface{}

	// Visit a parse tree produced by MySqlParser#nullNotnull.
	VisitNullNotnull(ctx *NullNotnullContext) interface{}

	// Visit a parse tree produced by MySqlParser#constant.
	VisitConstant(ctx *ConstantContext) interface{}

	// Visit a parse tree produced by MySqlParser#stringDataType.
	VisitStringDataType(ctx *StringDataTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#nationalVaryingStringDataType.
	VisitNationalVaryingStringDataType(ctx *NationalVaryingStringDataTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#nationalStringDataType.
	VisitNationalStringDataType(ctx *NationalStringDataTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#dimensionDataType.
	VisitDimensionDataType(ctx *DimensionDataTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#simpleDataType.
	VisitSimpleDataType(ctx *SimpleDataTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#collectionDataType.
	VisitCollectionDataType(ctx *CollectionDataTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#spatialDataType.
	VisitSpatialDataType(ctx *SpatialDataTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#longVarcharDataType.
	VisitLongVarcharDataType(ctx *LongVarcharDataTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#longVarbinaryDataType.
	VisitLongVarbinaryDataType(ctx *LongVarbinaryDataTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#collectionOptions.
	VisitCollectionOptions(ctx *CollectionOptionsContext) interface{}

	// Visit a parse tree produced by MySqlParser#convertedDataType.
	VisitConvertedDataType(ctx *ConvertedDataTypeContext) interface{}

	// Visit a parse tree produced by MySqlParser#lengthOneDimension.
	VisitLengthOneDimension(ctx *LengthOneDimensionContext) interface{}

	// Visit a parse tree produced by MySqlParser#lengthTwoDimension.
	VisitLengthTwoDimension(ctx *LengthTwoDimensionContext) interface{}

	// Visit a parse tree produced by MySqlParser#lengthTwoOptionalDimension.
	VisitLengthTwoOptionalDimension(ctx *LengthTwoOptionalDimensionContext) interface{}

	// Visit a parse tree produced by MySqlParser#uidList.
	VisitUidList(ctx *UidListContext) interface{}

	// Visit a parse tree produced by MySqlParser#fullColumnNameList.
	VisitFullColumnNameList(ctx *FullColumnNameListContext) interface{}

	// Visit a parse tree produced by MySqlParser#tables.
	VisitTables(ctx *TablesContext) interface{}

	// Visit a parse tree produced by MySqlParser#indexColumnNames.
	VisitIndexColumnNames(ctx *IndexColumnNamesContext) interface{}

	// Visit a parse tree produced by MySqlParser#expressions.
	VisitExpressions(ctx *ExpressionsContext) interface{}

	// Visit a parse tree produced by MySqlParser#expressionsWithDefaults.
	VisitExpressionsWithDefaults(ctx *ExpressionsWithDefaultsContext) interface{}

	// Visit a parse tree produced by MySqlParser#constants.
	VisitConstants(ctx *ConstantsContext) interface{}

	// Visit a parse tree produced by MySqlParser#simpleStrings.
	VisitSimpleStrings(ctx *SimpleStringsContext) interface{}

	// Visit a parse tree produced by MySqlParser#userVariables.
	VisitUserVariables(ctx *UserVariablesContext) interface{}

	// Visit a parse tree produced by MySqlParser#defaultValue.
	VisitDefaultValue(ctx *DefaultValueContext) interface{}

	// Visit a parse tree produced by MySqlParser#currentTimestamp.
	VisitCurrentTimestamp(ctx *CurrentTimestampContext) interface{}

	// Visit a parse tree produced by MySqlParser#expressionOrDefault.
	VisitExpressionOrDefault(ctx *ExpressionOrDefaultContext) interface{}

	// Visit a parse tree produced by MySqlParser#ifExists.
	VisitIfExists(ctx *IfExistsContext) interface{}

	// Visit a parse tree produced by MySqlParser#ifNotExists.
	VisitIfNotExists(ctx *IfNotExistsContext) interface{}

	// Visit a parse tree produced by MySqlParser#orReplace.
	VisitOrReplace(ctx *OrReplaceContext) interface{}

	// Visit a parse tree produced by MySqlParser#waitNowaitClause.
	VisitWaitNowaitClause(ctx *WaitNowaitClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#specificFunctionCall.
	VisitSpecificFunctionCall(ctx *SpecificFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#aggregateFunctionCall.
	VisitAggregateFunctionCall(ctx *AggregateFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#nonAggregateFunctionCall.
	VisitNonAggregateFunctionCall(ctx *NonAggregateFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#scalarFunctionCall.
	VisitScalarFunctionCall(ctx *ScalarFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#udfFunctionCall.
	VisitUdfFunctionCall(ctx *UdfFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#passwordFunctionCall.
	VisitPasswordFunctionCall(ctx *PasswordFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#simpleFunctionCall.
	VisitSimpleFunctionCall(ctx *SimpleFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#currentUser.
	VisitCurrentUser(ctx *CurrentUserContext) interface{}

	// Visit a parse tree produced by MySqlParser#dataTypeFunctionCall.
	VisitDataTypeFunctionCall(ctx *DataTypeFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#valuesFunctionCall.
	VisitValuesFunctionCall(ctx *ValuesFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#caseExpressionFunctionCall.
	VisitCaseExpressionFunctionCall(ctx *CaseExpressionFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#caseFunctionCall.
	VisitCaseFunctionCall(ctx *CaseFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#charFunctionCall.
	VisitCharFunctionCall(ctx *CharFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#positionFunctionCall.
	VisitPositionFunctionCall(ctx *PositionFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#substrFunctionCall.
	VisitSubstrFunctionCall(ctx *SubstrFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#trimFunctionCall.
	VisitTrimFunctionCall(ctx *TrimFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#weightFunctionCall.
	VisitWeightFunctionCall(ctx *WeightFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#extractFunctionCall.
	VisitExtractFunctionCall(ctx *ExtractFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#getFormatFunctionCall.
	VisitGetFormatFunctionCall(ctx *GetFormatFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#jsonValueFunctionCall.
	VisitJsonValueFunctionCall(ctx *JsonValueFunctionCallContext) interface{}

	// Visit a parse tree produced by MySqlParser#caseFuncAlternative.
	VisitCaseFuncAlternative(ctx *CaseFuncAlternativeContext) interface{}

	// Visit a parse tree produced by MySqlParser#levelWeightList.
	VisitLevelWeightList(ctx *LevelWeightListContext) interface{}

	// Visit a parse tree produced by MySqlParser#levelWeightRange.
	VisitLevelWeightRange(ctx *LevelWeightRangeContext) interface{}

	// Visit a parse tree produced by MySqlParser#levelInWeightListElement.
	VisitLevelInWeightListElement(ctx *LevelInWeightListElementContext) interface{}

	// Visit a parse tree produced by MySqlParser#aggregateWindowedFunction.
	VisitAggregateWindowedFunction(ctx *AggregateWindowedFunctionContext) interface{}

	// Visit a parse tree produced by MySqlParser#nonAggregateWindowedFunction.
	VisitNonAggregateWindowedFunction(ctx *NonAggregateWindowedFunctionContext) interface{}

	// Visit a parse tree produced by MySqlParser#overClause.
	VisitOverClause(ctx *OverClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#windowSpec.
	VisitWindowSpec(ctx *WindowSpecContext) interface{}

	// Visit a parse tree produced by MySqlParser#windowName.
	VisitWindowName(ctx *WindowNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#frameClause.
	VisitFrameClause(ctx *FrameClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#frameUnits.
	VisitFrameUnits(ctx *FrameUnitsContext) interface{}

	// Visit a parse tree produced by MySqlParser#frameExtent.
	VisitFrameExtent(ctx *FrameExtentContext) interface{}

	// Visit a parse tree produced by MySqlParser#frameBetween.
	VisitFrameBetween(ctx *FrameBetweenContext) interface{}

	// Visit a parse tree produced by MySqlParser#frameRange.
	VisitFrameRange(ctx *FrameRangeContext) interface{}

	// Visit a parse tree produced by MySqlParser#partitionClause.
	VisitPartitionClause(ctx *PartitionClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#scalarFunctionName.
	VisitScalarFunctionName(ctx *ScalarFunctionNameContext) interface{}

	// Visit a parse tree produced by MySqlParser#passwordFunctionClause.
	VisitPasswordFunctionClause(ctx *PasswordFunctionClauseContext) interface{}

	// Visit a parse tree produced by MySqlParser#functionArgs.
	VisitFunctionArgs(ctx *FunctionArgsContext) interface{}

	// Visit a parse tree produced by MySqlParser#functionArg.
	VisitFunctionArg(ctx *FunctionArgContext) interface{}

	// Visit a parse tree produced by MySqlParser#isExpression.
	VisitIsExpression(ctx *IsExpressionContext) interface{}

	// Visit a parse tree produced by MySqlParser#notExpression.
	VisitNotExpression(ctx *NotExpressionContext) interface{}

	// Visit a parse tree produced by MySqlParser#logicalExpression.
	VisitLogicalExpression(ctx *LogicalExpressionContext) interface{}

	// Visit a parse tree produced by MySqlParser#predicateExpression.
	VisitPredicateExpression(ctx *PredicateExpressionContext) interface{}

	// Visit a parse tree produced by MySqlParser#soundsLikePredicate.
	VisitSoundsLikePredicate(ctx *SoundsLikePredicateContext) interface{}

	// Visit a parse tree produced by MySqlParser#expressionAtomPredicate.
	VisitExpressionAtomPredicate(ctx *ExpressionAtomPredicateContext) interface{}

	// Visit a parse tree produced by MySqlParser#subqueryComparisonPredicate.
	VisitSubqueryComparisonPredicate(ctx *SubqueryComparisonPredicateContext) interface{}

	// Visit a parse tree produced by MySqlParser#jsonMemberOfPredicate.
	VisitJsonMemberOfPredicate(ctx *JsonMemberOfPredicateContext) interface{}

	// Visit a parse tree produced by MySqlParser#binaryComparisonPredicate.
	VisitBinaryComparisonPredicate(ctx *BinaryComparisonPredicateContext) interface{}

	// Visit a parse tree produced by MySqlParser#inPredicate.
	VisitInPredicate(ctx *InPredicateContext) interface{}

	// Visit a parse tree produced by MySqlParser#betweenPredicate.
	VisitBetweenPredicate(ctx *BetweenPredicateContext) interface{}

	// Visit a parse tree produced by MySqlParser#isNullPredicate.
	VisitIsNullPredicate(ctx *IsNullPredicateContext) interface{}

	// Visit a parse tree produced by MySqlParser#likePredicate.
	VisitLikePredicate(ctx *LikePredicateContext) interface{}

	// Visit a parse tree produced by MySqlParser#regexpPredicate.
	VisitRegexpPredicate(ctx *RegexpPredicateContext) interface{}

	// Visit a parse tree produced by MySqlParser#unaryExpressionAtom.
	VisitUnaryExpressionAtom(ctx *UnaryExpressionAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#collateExpressionAtom.
	VisitCollateExpressionAtom(ctx *CollateExpressionAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#variableAssignExpressionAtom.
	VisitVariableAssignExpressionAtom(ctx *VariableAssignExpressionAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#mysqlVariableExpressionAtom.
	VisitMysqlVariableExpressionAtom(ctx *MysqlVariableExpressionAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#nestedExpressionAtom.
	VisitNestedExpressionAtom(ctx *NestedExpressionAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#nestedRowExpressionAtom.
	VisitNestedRowExpressionAtom(ctx *NestedRowExpressionAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#mathExpressionAtom.
	VisitMathExpressionAtom(ctx *MathExpressionAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#existsExpressionAtom.
	VisitExistsExpressionAtom(ctx *ExistsExpressionAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#intervalExpressionAtom.
	VisitIntervalExpressionAtom(ctx *IntervalExpressionAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#jsonExpressionAtom.
	VisitJsonExpressionAtom(ctx *JsonExpressionAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#subqueryExpressionAtom.
	VisitSubqueryExpressionAtom(ctx *SubqueryExpressionAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#constantExpressionAtom.
	VisitConstantExpressionAtom(ctx *ConstantExpressionAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#functionCallExpressionAtom.
	VisitFunctionCallExpressionAtom(ctx *FunctionCallExpressionAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#binaryExpressionAtom.
	VisitBinaryExpressionAtom(ctx *BinaryExpressionAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#fullColumnNameExpressionAtom.
	VisitFullColumnNameExpressionAtom(ctx *FullColumnNameExpressionAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#bitExpressionAtom.
	VisitBitExpressionAtom(ctx *BitExpressionAtomContext) interface{}

	// Visit a parse tree produced by MySqlParser#unaryOperator.
	VisitUnaryOperator(ctx *UnaryOperatorContext) interface{}

	// Visit a parse tree produced by MySqlParser#comparisonOperator.
	VisitComparisonOperator(ctx *ComparisonOperatorContext) interface{}

	// Visit a parse tree produced by MySqlParser#logicalOperator.
	VisitLogicalOperator(ctx *LogicalOperatorContext) interface{}

	// Visit a parse tree produced by MySqlParser#bitOperator.
	VisitBitOperator(ctx *BitOperatorContext) interface{}

	// Visit a parse tree produced by MySqlParser#multOperator.
	VisitMultOperator(ctx *MultOperatorContext) interface{}

	// Visit a parse tree produced by MySqlParser#addOperator.
	VisitAddOperator(ctx *AddOperatorContext) interface{}

	// Visit a parse tree produced by MySqlParser#jsonOperator.
	VisitJsonOperator(ctx *JsonOperatorContext) interface{}

	// Visit a parse tree produced by MySqlParser#charsetNameBase.
	VisitCharsetNameBase(ctx *CharsetNameBaseContext) interface{}

	// Visit a parse tree produced by MySqlParser#transactionLevelBase.
	VisitTransactionLevelBase(ctx *TransactionLevelBaseContext) interface{}

	// Visit a parse tree produced by MySqlParser#privilegesBase.
	VisitPrivilegesBase(ctx *PrivilegesBaseContext) interface{}

	// Visit a parse tree produced by MySqlParser#intervalTypeBase.
	VisitIntervalTypeBase(ctx *IntervalTypeBaseContext) interface{}

	// Visit a parse tree produced by MySqlParser#dataTypeBase.
	VisitDataTypeBase(ctx *DataTypeBaseContext) interface{}

	// Visit a parse tree produced by MySqlParser#keywordsCanBeId.
	VisitKeywordsCanBeId(ctx *KeywordsCanBeIdContext) interface{}

	// Visit a parse tree produced by MySqlParser#functionNameBase.
	VisitFunctionNameBase(ctx *FunctionNameBaseContext) interface{}
}
