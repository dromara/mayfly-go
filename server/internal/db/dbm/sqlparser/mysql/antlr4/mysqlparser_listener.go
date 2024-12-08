// Code generated from MySqlParser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package parser // MySqlParser
import "github.com/antlr4-go/antlr/v4"

// MySqlParserListener is a complete listener for a parse tree produced by MySqlParser.
type MySqlParserListener interface {
	antlr.ParseTreeListener

	// EnterRoot is called when entering the root production.
	EnterRoot(c *RootContext)

	// EnterSqlStatements is called when entering the sqlStatements production.
	EnterSqlStatements(c *SqlStatementsContext)

	// EnterSqlStatement is called when entering the sqlStatement production.
	EnterSqlStatement(c *SqlStatementContext)

	// EnterEmptyStatement_ is called when entering the emptyStatement_ production.
	EnterEmptyStatement_(c *EmptyStatement_Context)

	// EnterDdlStatement is called when entering the ddlStatement production.
	EnterDdlStatement(c *DdlStatementContext)

	// EnterDmlStatement is called when entering the dmlStatement production.
	EnterDmlStatement(c *DmlStatementContext)

	// EnterTransactionStatement is called when entering the transactionStatement production.
	EnterTransactionStatement(c *TransactionStatementContext)

	// EnterReplicationStatement is called when entering the replicationStatement production.
	EnterReplicationStatement(c *ReplicationStatementContext)

	// EnterPreparedStatement is called when entering the preparedStatement production.
	EnterPreparedStatement(c *PreparedStatementContext)

	// EnterCompoundStatement is called when entering the compoundStatement production.
	EnterCompoundStatement(c *CompoundStatementContext)

	// EnterAdministrationStatement is called when entering the administrationStatement production.
	EnterAdministrationStatement(c *AdministrationStatementContext)

	// EnterUtilityStatement is called when entering the utilityStatement production.
	EnterUtilityStatement(c *UtilityStatementContext)

	// EnterCreateDatabase is called when entering the createDatabase production.
	EnterCreateDatabase(c *CreateDatabaseContext)

	// EnterCreateEvent is called when entering the createEvent production.
	EnterCreateEvent(c *CreateEventContext)

	// EnterCreateIndex is called when entering the createIndex production.
	EnterCreateIndex(c *CreateIndexContext)

	// EnterCreateLogfileGroup is called when entering the createLogfileGroup production.
	EnterCreateLogfileGroup(c *CreateLogfileGroupContext)

	// EnterCreateProcedure is called when entering the createProcedure production.
	EnterCreateProcedure(c *CreateProcedureContext)

	// EnterCreateFunction is called when entering the createFunction production.
	EnterCreateFunction(c *CreateFunctionContext)

	// EnterCreateRole is called when entering the createRole production.
	EnterCreateRole(c *CreateRoleContext)

	// EnterCreateServer is called when entering the createServer production.
	EnterCreateServer(c *CreateServerContext)

	// EnterCopyCreateTable is called when entering the copyCreateTable production.
	EnterCopyCreateTable(c *CopyCreateTableContext)

	// EnterQueryCreateTable is called when entering the queryCreateTable production.
	EnterQueryCreateTable(c *QueryCreateTableContext)

	// EnterColumnCreateTable is called when entering the columnCreateTable production.
	EnterColumnCreateTable(c *ColumnCreateTableContext)

	// EnterCreateTablespaceInnodb is called when entering the createTablespaceInnodb production.
	EnterCreateTablespaceInnodb(c *CreateTablespaceInnodbContext)

	// EnterCreateTablespaceNdb is called when entering the createTablespaceNdb production.
	EnterCreateTablespaceNdb(c *CreateTablespaceNdbContext)

	// EnterCreateTrigger is called when entering the createTrigger production.
	EnterCreateTrigger(c *CreateTriggerContext)

	// EnterWithClause is called when entering the withClause production.
	EnterWithClause(c *WithClauseContext)

	// EnterCommonTableExpressions is called when entering the commonTableExpressions production.
	EnterCommonTableExpressions(c *CommonTableExpressionsContext)

	// EnterCteName is called when entering the cteName production.
	EnterCteName(c *CteNameContext)

	// EnterCteColumnName is called when entering the cteColumnName production.
	EnterCteColumnName(c *CteColumnNameContext)

	// EnterCreateView is called when entering the createView production.
	EnterCreateView(c *CreateViewContext)

	// EnterCreateDatabaseOption is called when entering the createDatabaseOption production.
	EnterCreateDatabaseOption(c *CreateDatabaseOptionContext)

	// EnterCharSet is called when entering the charSet production.
	EnterCharSet(c *CharSetContext)

	// EnterCurrentUserExpression is called when entering the currentUserExpression production.
	EnterCurrentUserExpression(c *CurrentUserExpressionContext)

	// EnterOwnerStatement is called when entering the ownerStatement production.
	EnterOwnerStatement(c *OwnerStatementContext)

	// EnterPreciseSchedule is called when entering the preciseSchedule production.
	EnterPreciseSchedule(c *PreciseScheduleContext)

	// EnterIntervalSchedule is called when entering the intervalSchedule production.
	EnterIntervalSchedule(c *IntervalScheduleContext)

	// EnterTimestampValue is called when entering the timestampValue production.
	EnterTimestampValue(c *TimestampValueContext)

	// EnterIntervalExpr is called when entering the intervalExpr production.
	EnterIntervalExpr(c *IntervalExprContext)

	// EnterIntervalType is called when entering the intervalType production.
	EnterIntervalType(c *IntervalTypeContext)

	// EnterEnableType is called when entering the enableType production.
	EnterEnableType(c *EnableTypeContext)

	// EnterIndexType is called when entering the indexType production.
	EnterIndexType(c *IndexTypeContext)

	// EnterIndexOption is called when entering the indexOption production.
	EnterIndexOption(c *IndexOptionContext)

	// EnterProcedureParameter is called when entering the procedureParameter production.
	EnterProcedureParameter(c *ProcedureParameterContext)

	// EnterFunctionParameter is called when entering the functionParameter production.
	EnterFunctionParameter(c *FunctionParameterContext)

	// EnterRoutineComment is called when entering the routineComment production.
	EnterRoutineComment(c *RoutineCommentContext)

	// EnterRoutineLanguage is called when entering the routineLanguage production.
	EnterRoutineLanguage(c *RoutineLanguageContext)

	// EnterRoutineBehavior is called when entering the routineBehavior production.
	EnterRoutineBehavior(c *RoutineBehaviorContext)

	// EnterRoutineData is called when entering the routineData production.
	EnterRoutineData(c *RoutineDataContext)

	// EnterRoutineSecurity is called when entering the routineSecurity production.
	EnterRoutineSecurity(c *RoutineSecurityContext)

	// EnterServerOption is called when entering the serverOption production.
	EnterServerOption(c *ServerOptionContext)

	// EnterCreateDefinitions is called when entering the createDefinitions production.
	EnterCreateDefinitions(c *CreateDefinitionsContext)

	// EnterColumnDeclaration is called when entering the columnDeclaration production.
	EnterColumnDeclaration(c *ColumnDeclarationContext)

	// EnterConstraintDeclaration is called when entering the constraintDeclaration production.
	EnterConstraintDeclaration(c *ConstraintDeclarationContext)

	// EnterIndexDeclaration is called when entering the indexDeclaration production.
	EnterIndexDeclaration(c *IndexDeclarationContext)

	// EnterColumnDefinition is called when entering the columnDefinition production.
	EnterColumnDefinition(c *ColumnDefinitionContext)

	// EnterNullColumnConstraint is called when entering the nullColumnConstraint production.
	EnterNullColumnConstraint(c *NullColumnConstraintContext)

	// EnterDefaultColumnConstraint is called when entering the defaultColumnConstraint production.
	EnterDefaultColumnConstraint(c *DefaultColumnConstraintContext)

	// EnterVisibilityColumnConstraint is called when entering the visibilityColumnConstraint production.
	EnterVisibilityColumnConstraint(c *VisibilityColumnConstraintContext)

	// EnterInvisibilityColumnConstraint is called when entering the invisibilityColumnConstraint production.
	EnterInvisibilityColumnConstraint(c *InvisibilityColumnConstraintContext)

	// EnterAutoIncrementColumnConstraint is called when entering the autoIncrementColumnConstraint production.
	EnterAutoIncrementColumnConstraint(c *AutoIncrementColumnConstraintContext)

	// EnterPrimaryKeyColumnConstraint is called when entering the primaryKeyColumnConstraint production.
	EnterPrimaryKeyColumnConstraint(c *PrimaryKeyColumnConstraintContext)

	// EnterUniqueKeyColumnConstraint is called when entering the uniqueKeyColumnConstraint production.
	EnterUniqueKeyColumnConstraint(c *UniqueKeyColumnConstraintContext)

	// EnterCommentColumnConstraint is called when entering the commentColumnConstraint production.
	EnterCommentColumnConstraint(c *CommentColumnConstraintContext)

	// EnterFormatColumnConstraint is called when entering the formatColumnConstraint production.
	EnterFormatColumnConstraint(c *FormatColumnConstraintContext)

	// EnterStorageColumnConstraint is called when entering the storageColumnConstraint production.
	EnterStorageColumnConstraint(c *StorageColumnConstraintContext)

	// EnterReferenceColumnConstraint is called when entering the referenceColumnConstraint production.
	EnterReferenceColumnConstraint(c *ReferenceColumnConstraintContext)

	// EnterCollateColumnConstraint is called when entering the collateColumnConstraint production.
	EnterCollateColumnConstraint(c *CollateColumnConstraintContext)

	// EnterGeneratedColumnConstraint is called when entering the generatedColumnConstraint production.
	EnterGeneratedColumnConstraint(c *GeneratedColumnConstraintContext)

	// EnterSerialDefaultColumnConstraint is called when entering the serialDefaultColumnConstraint production.
	EnterSerialDefaultColumnConstraint(c *SerialDefaultColumnConstraintContext)

	// EnterCheckColumnConstraint is called when entering the checkColumnConstraint production.
	EnterCheckColumnConstraint(c *CheckColumnConstraintContext)

	// EnterPrimaryKeyTableConstraint is called when entering the primaryKeyTableConstraint production.
	EnterPrimaryKeyTableConstraint(c *PrimaryKeyTableConstraintContext)

	// EnterUniqueKeyTableConstraint is called when entering the uniqueKeyTableConstraint production.
	EnterUniqueKeyTableConstraint(c *UniqueKeyTableConstraintContext)

	// EnterForeignKeyTableConstraint is called when entering the foreignKeyTableConstraint production.
	EnterForeignKeyTableConstraint(c *ForeignKeyTableConstraintContext)

	// EnterCheckTableConstraint is called when entering the checkTableConstraint production.
	EnterCheckTableConstraint(c *CheckTableConstraintContext)

	// EnterReferenceDefinition is called when entering the referenceDefinition production.
	EnterReferenceDefinition(c *ReferenceDefinitionContext)

	// EnterReferenceAction is called when entering the referenceAction production.
	EnterReferenceAction(c *ReferenceActionContext)

	// EnterReferenceControlType is called when entering the referenceControlType production.
	EnterReferenceControlType(c *ReferenceControlTypeContext)

	// EnterSimpleIndexDeclaration is called when entering the simpleIndexDeclaration production.
	EnterSimpleIndexDeclaration(c *SimpleIndexDeclarationContext)

	// EnterSpecialIndexDeclaration is called when entering the specialIndexDeclaration production.
	EnterSpecialIndexDeclaration(c *SpecialIndexDeclarationContext)

	// EnterTableOptionEngine is called when entering the tableOptionEngine production.
	EnterTableOptionEngine(c *TableOptionEngineContext)

	// EnterTableOptionEngineAttribute is called when entering the tableOptionEngineAttribute production.
	EnterTableOptionEngineAttribute(c *TableOptionEngineAttributeContext)

	// EnterTableOptionAutoextendSize is called when entering the tableOptionAutoextendSize production.
	EnterTableOptionAutoextendSize(c *TableOptionAutoextendSizeContext)

	// EnterTableOptionAutoIncrement is called when entering the tableOptionAutoIncrement production.
	EnterTableOptionAutoIncrement(c *TableOptionAutoIncrementContext)

	// EnterTableOptionAverage is called when entering the tableOptionAverage production.
	EnterTableOptionAverage(c *TableOptionAverageContext)

	// EnterTableOptionCharset is called when entering the tableOptionCharset production.
	EnterTableOptionCharset(c *TableOptionCharsetContext)

	// EnterTableOptionChecksum is called when entering the tableOptionChecksum production.
	EnterTableOptionChecksum(c *TableOptionChecksumContext)

	// EnterTableOptionCollate is called when entering the tableOptionCollate production.
	EnterTableOptionCollate(c *TableOptionCollateContext)

	// EnterTableOptionComment is called when entering the tableOptionComment production.
	EnterTableOptionComment(c *TableOptionCommentContext)

	// EnterTableOptionCompression is called when entering the tableOptionCompression production.
	EnterTableOptionCompression(c *TableOptionCompressionContext)

	// EnterTableOptionConnection is called when entering the tableOptionConnection production.
	EnterTableOptionConnection(c *TableOptionConnectionContext)

	// EnterTableOptionDataDirectory is called when entering the tableOptionDataDirectory production.
	EnterTableOptionDataDirectory(c *TableOptionDataDirectoryContext)

	// EnterTableOptionDelay is called when entering the tableOptionDelay production.
	EnterTableOptionDelay(c *TableOptionDelayContext)

	// EnterTableOptionEncryption is called when entering the tableOptionEncryption production.
	EnterTableOptionEncryption(c *TableOptionEncryptionContext)

	// EnterTableOptionPageCompressed is called when entering the tableOptionPageCompressed production.
	EnterTableOptionPageCompressed(c *TableOptionPageCompressedContext)

	// EnterTableOptionPageCompressionLevel is called when entering the tableOptionPageCompressionLevel production.
	EnterTableOptionPageCompressionLevel(c *TableOptionPageCompressionLevelContext)

	// EnterTableOptionEncryptionKeyId is called when entering the tableOptionEncryptionKeyId production.
	EnterTableOptionEncryptionKeyId(c *TableOptionEncryptionKeyIdContext)

	// EnterTableOptionIndexDirectory is called when entering the tableOptionIndexDirectory production.
	EnterTableOptionIndexDirectory(c *TableOptionIndexDirectoryContext)

	// EnterTableOptionInsertMethod is called when entering the tableOptionInsertMethod production.
	EnterTableOptionInsertMethod(c *TableOptionInsertMethodContext)

	// EnterTableOptionKeyBlockSize is called when entering the tableOptionKeyBlockSize production.
	EnterTableOptionKeyBlockSize(c *TableOptionKeyBlockSizeContext)

	// EnterTableOptionMaxRows is called when entering the tableOptionMaxRows production.
	EnterTableOptionMaxRows(c *TableOptionMaxRowsContext)

	// EnterTableOptionMinRows is called when entering the tableOptionMinRows production.
	EnterTableOptionMinRows(c *TableOptionMinRowsContext)

	// EnterTableOptionPackKeys is called when entering the tableOptionPackKeys production.
	EnterTableOptionPackKeys(c *TableOptionPackKeysContext)

	// EnterTableOptionPassword is called when entering the tableOptionPassword production.
	EnterTableOptionPassword(c *TableOptionPasswordContext)

	// EnterTableOptionRowFormat is called when entering the tableOptionRowFormat production.
	EnterTableOptionRowFormat(c *TableOptionRowFormatContext)

	// EnterTableOptionStartTransaction is called when entering the tableOptionStartTransaction production.
	EnterTableOptionStartTransaction(c *TableOptionStartTransactionContext)

	// EnterTableOptionSecondaryEngineAttribute is called when entering the tableOptionSecondaryEngineAttribute production.
	EnterTableOptionSecondaryEngineAttribute(c *TableOptionSecondaryEngineAttributeContext)

	// EnterTableOptionRecalculation is called when entering the tableOptionRecalculation production.
	EnterTableOptionRecalculation(c *TableOptionRecalculationContext)

	// EnterTableOptionPersistent is called when entering the tableOptionPersistent production.
	EnterTableOptionPersistent(c *TableOptionPersistentContext)

	// EnterTableOptionSamplePage is called when entering the tableOptionSamplePage production.
	EnterTableOptionSamplePage(c *TableOptionSamplePageContext)

	// EnterTableOptionTablespace is called when entering the tableOptionTablespace production.
	EnterTableOptionTablespace(c *TableOptionTablespaceContext)

	// EnterTableOptionTableType is called when entering the tableOptionTableType production.
	EnterTableOptionTableType(c *TableOptionTableTypeContext)

	// EnterTableOptionTransactional is called when entering the tableOptionTransactional production.
	EnterTableOptionTransactional(c *TableOptionTransactionalContext)

	// EnterTableOptionUnion is called when entering the tableOptionUnion production.
	EnterTableOptionUnion(c *TableOptionUnionContext)

	// EnterTableType is called when entering the tableType production.
	EnterTableType(c *TableTypeContext)

	// EnterTablespaceStorage is called when entering the tablespaceStorage production.
	EnterTablespaceStorage(c *TablespaceStorageContext)

	// EnterPartitionDefinitions is called when entering the partitionDefinitions production.
	EnterPartitionDefinitions(c *PartitionDefinitionsContext)

	// EnterPartitionFunctionHash is called when entering the partitionFunctionHash production.
	EnterPartitionFunctionHash(c *PartitionFunctionHashContext)

	// EnterPartitionFunctionKey is called when entering the partitionFunctionKey production.
	EnterPartitionFunctionKey(c *PartitionFunctionKeyContext)

	// EnterPartitionFunctionRange is called when entering the partitionFunctionRange production.
	EnterPartitionFunctionRange(c *PartitionFunctionRangeContext)

	// EnterPartitionFunctionList is called when entering the partitionFunctionList production.
	EnterPartitionFunctionList(c *PartitionFunctionListContext)

	// EnterSubPartitionFunctionHash is called when entering the subPartitionFunctionHash production.
	EnterSubPartitionFunctionHash(c *SubPartitionFunctionHashContext)

	// EnterSubPartitionFunctionKey is called when entering the subPartitionFunctionKey production.
	EnterSubPartitionFunctionKey(c *SubPartitionFunctionKeyContext)

	// EnterPartitionComparison is called when entering the partitionComparison production.
	EnterPartitionComparison(c *PartitionComparisonContext)

	// EnterPartitionListAtom is called when entering the partitionListAtom production.
	EnterPartitionListAtom(c *PartitionListAtomContext)

	// EnterPartitionListVector is called when entering the partitionListVector production.
	EnterPartitionListVector(c *PartitionListVectorContext)

	// EnterPartitionSimple is called when entering the partitionSimple production.
	EnterPartitionSimple(c *PartitionSimpleContext)

	// EnterPartitionDefinerAtom is called when entering the partitionDefinerAtom production.
	EnterPartitionDefinerAtom(c *PartitionDefinerAtomContext)

	// EnterPartitionDefinerVector is called when entering the partitionDefinerVector production.
	EnterPartitionDefinerVector(c *PartitionDefinerVectorContext)

	// EnterSubpartitionDefinition is called when entering the subpartitionDefinition production.
	EnterSubpartitionDefinition(c *SubpartitionDefinitionContext)

	// EnterPartitionOptionEngine is called when entering the partitionOptionEngine production.
	EnterPartitionOptionEngine(c *PartitionOptionEngineContext)

	// EnterPartitionOptionComment is called when entering the partitionOptionComment production.
	EnterPartitionOptionComment(c *PartitionOptionCommentContext)

	// EnterPartitionOptionDataDirectory is called when entering the partitionOptionDataDirectory production.
	EnterPartitionOptionDataDirectory(c *PartitionOptionDataDirectoryContext)

	// EnterPartitionOptionIndexDirectory is called when entering the partitionOptionIndexDirectory production.
	EnterPartitionOptionIndexDirectory(c *PartitionOptionIndexDirectoryContext)

	// EnterPartitionOptionMaxRows is called when entering the partitionOptionMaxRows production.
	EnterPartitionOptionMaxRows(c *PartitionOptionMaxRowsContext)

	// EnterPartitionOptionMinRows is called when entering the partitionOptionMinRows production.
	EnterPartitionOptionMinRows(c *PartitionOptionMinRowsContext)

	// EnterPartitionOptionTablespace is called when entering the partitionOptionTablespace production.
	EnterPartitionOptionTablespace(c *PartitionOptionTablespaceContext)

	// EnterPartitionOptionNodeGroup is called when entering the partitionOptionNodeGroup production.
	EnterPartitionOptionNodeGroup(c *PartitionOptionNodeGroupContext)

	// EnterAlterSimpleDatabase is called when entering the alterSimpleDatabase production.
	EnterAlterSimpleDatabase(c *AlterSimpleDatabaseContext)

	// EnterAlterUpgradeName is called when entering the alterUpgradeName production.
	EnterAlterUpgradeName(c *AlterUpgradeNameContext)

	// EnterAlterEvent is called when entering the alterEvent production.
	EnterAlterEvent(c *AlterEventContext)

	// EnterAlterFunction is called when entering the alterFunction production.
	EnterAlterFunction(c *AlterFunctionContext)

	// EnterAlterInstance is called when entering the alterInstance production.
	EnterAlterInstance(c *AlterInstanceContext)

	// EnterAlterLogfileGroup is called when entering the alterLogfileGroup production.
	EnterAlterLogfileGroup(c *AlterLogfileGroupContext)

	// EnterAlterProcedure is called when entering the alterProcedure production.
	EnterAlterProcedure(c *AlterProcedureContext)

	// EnterAlterServer is called when entering the alterServer production.
	EnterAlterServer(c *AlterServerContext)

	// EnterAlterTable is called when entering the alterTable production.
	EnterAlterTable(c *AlterTableContext)

	// EnterAlterTablespace is called when entering the alterTablespace production.
	EnterAlterTablespace(c *AlterTablespaceContext)

	// EnterAlterView is called when entering the alterView production.
	EnterAlterView(c *AlterViewContext)

	// EnterAlterByTableOption is called when entering the alterByTableOption production.
	EnterAlterByTableOption(c *AlterByTableOptionContext)

	// EnterAlterByAddColumn is called when entering the alterByAddColumn production.
	EnterAlterByAddColumn(c *AlterByAddColumnContext)

	// EnterAlterByAddColumns is called when entering the alterByAddColumns production.
	EnterAlterByAddColumns(c *AlterByAddColumnsContext)

	// EnterAlterByAddIndex is called when entering the alterByAddIndex production.
	EnterAlterByAddIndex(c *AlterByAddIndexContext)

	// EnterAlterByAddPrimaryKey is called when entering the alterByAddPrimaryKey production.
	EnterAlterByAddPrimaryKey(c *AlterByAddPrimaryKeyContext)

	// EnterAlterByAddUniqueKey is called when entering the alterByAddUniqueKey production.
	EnterAlterByAddUniqueKey(c *AlterByAddUniqueKeyContext)

	// EnterAlterByAddSpecialIndex is called when entering the alterByAddSpecialIndex production.
	EnterAlterByAddSpecialIndex(c *AlterByAddSpecialIndexContext)

	// EnterAlterByAddForeignKey is called when entering the alterByAddForeignKey production.
	EnterAlterByAddForeignKey(c *AlterByAddForeignKeyContext)

	// EnterAlterByAddCheckTableConstraint is called when entering the alterByAddCheckTableConstraint production.
	EnterAlterByAddCheckTableConstraint(c *AlterByAddCheckTableConstraintContext)

	// EnterAlterByAlterCheckTableConstraint is called when entering the alterByAlterCheckTableConstraint production.
	EnterAlterByAlterCheckTableConstraint(c *AlterByAlterCheckTableConstraintContext)

	// EnterAlterBySetAlgorithm is called when entering the alterBySetAlgorithm production.
	EnterAlterBySetAlgorithm(c *AlterBySetAlgorithmContext)

	// EnterAlterByChangeDefault is called when entering the alterByChangeDefault production.
	EnterAlterByChangeDefault(c *AlterByChangeDefaultContext)

	// EnterAlterByChangeColumn is called when entering the alterByChangeColumn production.
	EnterAlterByChangeColumn(c *AlterByChangeColumnContext)

	// EnterAlterByRenameColumn is called when entering the alterByRenameColumn production.
	EnterAlterByRenameColumn(c *AlterByRenameColumnContext)

	// EnterAlterByLock is called when entering the alterByLock production.
	EnterAlterByLock(c *AlterByLockContext)

	// EnterAlterByModifyColumn is called when entering the alterByModifyColumn production.
	EnterAlterByModifyColumn(c *AlterByModifyColumnContext)

	// EnterAlterByDropColumn is called when entering the alterByDropColumn production.
	EnterAlterByDropColumn(c *AlterByDropColumnContext)

	// EnterAlterByDropConstraintCheck is called when entering the alterByDropConstraintCheck production.
	EnterAlterByDropConstraintCheck(c *AlterByDropConstraintCheckContext)

	// EnterAlterByDropPrimaryKey is called when entering the alterByDropPrimaryKey production.
	EnterAlterByDropPrimaryKey(c *AlterByDropPrimaryKeyContext)

	// EnterAlterByDropIndex is called when entering the alterByDropIndex production.
	EnterAlterByDropIndex(c *AlterByDropIndexContext)

	// EnterAlterByRenameIndex is called when entering the alterByRenameIndex production.
	EnterAlterByRenameIndex(c *AlterByRenameIndexContext)

	// EnterAlterByAlterColumnDefault is called when entering the alterByAlterColumnDefault production.
	EnterAlterByAlterColumnDefault(c *AlterByAlterColumnDefaultContext)

	// EnterAlterByAlterIndexVisibility is called when entering the alterByAlterIndexVisibility production.
	EnterAlterByAlterIndexVisibility(c *AlterByAlterIndexVisibilityContext)

	// EnterAlterByDropForeignKey is called when entering the alterByDropForeignKey production.
	EnterAlterByDropForeignKey(c *AlterByDropForeignKeyContext)

	// EnterAlterByDisableKeys is called when entering the alterByDisableKeys production.
	EnterAlterByDisableKeys(c *AlterByDisableKeysContext)

	// EnterAlterByEnableKeys is called when entering the alterByEnableKeys production.
	EnterAlterByEnableKeys(c *AlterByEnableKeysContext)

	// EnterAlterByRename is called when entering the alterByRename production.
	EnterAlterByRename(c *AlterByRenameContext)

	// EnterAlterByOrder is called when entering the alterByOrder production.
	EnterAlterByOrder(c *AlterByOrderContext)

	// EnterAlterByConvertCharset is called when entering the alterByConvertCharset production.
	EnterAlterByConvertCharset(c *AlterByConvertCharsetContext)

	// EnterAlterByDefaultCharset is called when entering the alterByDefaultCharset production.
	EnterAlterByDefaultCharset(c *AlterByDefaultCharsetContext)

	// EnterAlterByDiscardTablespace is called when entering the alterByDiscardTablespace production.
	EnterAlterByDiscardTablespace(c *AlterByDiscardTablespaceContext)

	// EnterAlterByImportTablespace is called when entering the alterByImportTablespace production.
	EnterAlterByImportTablespace(c *AlterByImportTablespaceContext)

	// EnterAlterByForce is called when entering the alterByForce production.
	EnterAlterByForce(c *AlterByForceContext)

	// EnterAlterByValidate is called when entering the alterByValidate production.
	EnterAlterByValidate(c *AlterByValidateContext)

	// EnterAlterByAddDefinitions is called when entering the alterByAddDefinitions production.
	EnterAlterByAddDefinitions(c *AlterByAddDefinitionsContext)

	// EnterAlterPartition is called when entering the alterPartition production.
	EnterAlterPartition(c *AlterPartitionContext)

	// EnterAlterByAddPartition is called when entering the alterByAddPartition production.
	EnterAlterByAddPartition(c *AlterByAddPartitionContext)

	// EnterAlterByDropPartition is called when entering the alterByDropPartition production.
	EnterAlterByDropPartition(c *AlterByDropPartitionContext)

	// EnterAlterByDiscardPartition is called when entering the alterByDiscardPartition production.
	EnterAlterByDiscardPartition(c *AlterByDiscardPartitionContext)

	// EnterAlterByImportPartition is called when entering the alterByImportPartition production.
	EnterAlterByImportPartition(c *AlterByImportPartitionContext)

	// EnterAlterByTruncatePartition is called when entering the alterByTruncatePartition production.
	EnterAlterByTruncatePartition(c *AlterByTruncatePartitionContext)

	// EnterAlterByCoalescePartition is called when entering the alterByCoalescePartition production.
	EnterAlterByCoalescePartition(c *AlterByCoalescePartitionContext)

	// EnterAlterByReorganizePartition is called when entering the alterByReorganizePartition production.
	EnterAlterByReorganizePartition(c *AlterByReorganizePartitionContext)

	// EnterAlterByExchangePartition is called when entering the alterByExchangePartition production.
	EnterAlterByExchangePartition(c *AlterByExchangePartitionContext)

	// EnterAlterByAnalyzePartition is called when entering the alterByAnalyzePartition production.
	EnterAlterByAnalyzePartition(c *AlterByAnalyzePartitionContext)

	// EnterAlterByCheckPartition is called when entering the alterByCheckPartition production.
	EnterAlterByCheckPartition(c *AlterByCheckPartitionContext)

	// EnterAlterByOptimizePartition is called when entering the alterByOptimizePartition production.
	EnterAlterByOptimizePartition(c *AlterByOptimizePartitionContext)

	// EnterAlterByRebuildPartition is called when entering the alterByRebuildPartition production.
	EnterAlterByRebuildPartition(c *AlterByRebuildPartitionContext)

	// EnterAlterByRepairPartition is called when entering the alterByRepairPartition production.
	EnterAlterByRepairPartition(c *AlterByRepairPartitionContext)

	// EnterAlterByRemovePartitioning is called when entering the alterByRemovePartitioning production.
	EnterAlterByRemovePartitioning(c *AlterByRemovePartitioningContext)

	// EnterAlterByUpgradePartitioning is called when entering the alterByUpgradePartitioning production.
	EnterAlterByUpgradePartitioning(c *AlterByUpgradePartitioningContext)

	// EnterDropDatabase is called when entering the dropDatabase production.
	EnterDropDatabase(c *DropDatabaseContext)

	// EnterDropEvent is called when entering the dropEvent production.
	EnterDropEvent(c *DropEventContext)

	// EnterDropIndex is called when entering the dropIndex production.
	EnterDropIndex(c *DropIndexContext)

	// EnterDropLogfileGroup is called when entering the dropLogfileGroup production.
	EnterDropLogfileGroup(c *DropLogfileGroupContext)

	// EnterDropProcedure is called when entering the dropProcedure production.
	EnterDropProcedure(c *DropProcedureContext)

	// EnterDropFunction is called when entering the dropFunction production.
	EnterDropFunction(c *DropFunctionContext)

	// EnterDropServer is called when entering the dropServer production.
	EnterDropServer(c *DropServerContext)

	// EnterDropTable is called when entering the dropTable production.
	EnterDropTable(c *DropTableContext)

	// EnterDropTablespace is called when entering the dropTablespace production.
	EnterDropTablespace(c *DropTablespaceContext)

	// EnterDropTrigger is called when entering the dropTrigger production.
	EnterDropTrigger(c *DropTriggerContext)

	// EnterDropView is called when entering the dropView production.
	EnterDropView(c *DropViewContext)

	// EnterDropRole is called when entering the dropRole production.
	EnterDropRole(c *DropRoleContext)

	// EnterSetRole is called when entering the setRole production.
	EnterSetRole(c *SetRoleContext)

	// EnterRenameTable is called when entering the renameTable production.
	EnterRenameTable(c *RenameTableContext)

	// EnterRenameTableClause is called when entering the renameTableClause production.
	EnterRenameTableClause(c *RenameTableClauseContext)

	// EnterTruncateTable is called when entering the truncateTable production.
	EnterTruncateTable(c *TruncateTableContext)

	// EnterCallStatement is called when entering the callStatement production.
	EnterCallStatement(c *CallStatementContext)

	// EnterDeleteStatement is called when entering the deleteStatement production.
	EnterDeleteStatement(c *DeleteStatementContext)

	// EnterDoStatement is called when entering the doStatement production.
	EnterDoStatement(c *DoStatementContext)

	// EnterHandlerStatement is called when entering the handlerStatement production.
	EnterHandlerStatement(c *HandlerStatementContext)

	// EnterInsertStatement is called when entering the insertStatement production.
	EnterInsertStatement(c *InsertStatementContext)

	// EnterLoadDataStatement is called when entering the loadDataStatement production.
	EnterLoadDataStatement(c *LoadDataStatementContext)

	// EnterLoadXmlStatement is called when entering the loadXmlStatement production.
	EnterLoadXmlStatement(c *LoadXmlStatementContext)

	// EnterReplaceStatement is called when entering the replaceStatement production.
	EnterReplaceStatement(c *ReplaceStatementContext)

	// EnterSimpleSelect is called when entering the simpleSelect production.
	EnterSimpleSelect(c *SimpleSelectContext)

	// EnterParenthesisSelect is called when entering the parenthesisSelect production.
	EnterParenthesisSelect(c *ParenthesisSelectContext)

	// EnterUnionSelect is called when entering the unionSelect production.
	EnterUnionSelect(c *UnionSelectContext)

	// EnterUnionParenthesisSelect is called when entering the unionParenthesisSelect production.
	EnterUnionParenthesisSelect(c *UnionParenthesisSelectContext)

	// EnterWithLateralStatement is called when entering the withLateralStatement production.
	EnterWithLateralStatement(c *WithLateralStatementContext)

	// EnterUpdateStatement is called when entering the updateStatement production.
	EnterUpdateStatement(c *UpdateStatementContext)

	// EnterValuesStatement is called when entering the valuesStatement production.
	EnterValuesStatement(c *ValuesStatementContext)

	// EnterInsertStatementValue is called when entering the insertStatementValue production.
	EnterInsertStatementValue(c *InsertStatementValueContext)

	// EnterUpdatedElement is called when entering the updatedElement production.
	EnterUpdatedElement(c *UpdatedElementContext)

	// EnterAssignmentField is called when entering the assignmentField production.
	EnterAssignmentField(c *AssignmentFieldContext)

	// EnterLockClause is called when entering the lockClause production.
	EnterLockClause(c *LockClauseContext)

	// EnterSingleDeleteStatement is called when entering the singleDeleteStatement production.
	EnterSingleDeleteStatement(c *SingleDeleteStatementContext)

	// EnterMultipleDeleteStatement is called when entering the multipleDeleteStatement production.
	EnterMultipleDeleteStatement(c *MultipleDeleteStatementContext)

	// EnterHandlerOpenStatement is called when entering the handlerOpenStatement production.
	EnterHandlerOpenStatement(c *HandlerOpenStatementContext)

	// EnterHandlerReadIndexStatement is called when entering the handlerReadIndexStatement production.
	EnterHandlerReadIndexStatement(c *HandlerReadIndexStatementContext)

	// EnterHandlerReadStatement is called when entering the handlerReadStatement production.
	EnterHandlerReadStatement(c *HandlerReadStatementContext)

	// EnterHandlerCloseStatement is called when entering the handlerCloseStatement production.
	EnterHandlerCloseStatement(c *HandlerCloseStatementContext)

	// EnterSingleUpdateStatement is called when entering the singleUpdateStatement production.
	EnterSingleUpdateStatement(c *SingleUpdateStatementContext)

	// EnterMultipleUpdateStatement is called when entering the multipleUpdateStatement production.
	EnterMultipleUpdateStatement(c *MultipleUpdateStatementContext)

	// EnterOrderByClause is called when entering the orderByClause production.
	EnterOrderByClause(c *OrderByClauseContext)

	// EnterOrderByExpression is called when entering the orderByExpression production.
	EnterOrderByExpression(c *OrderByExpressionContext)

	// EnterTableSources is called when entering the tableSources production.
	EnterTableSources(c *TableSourcesContext)

	// EnterTableSourceBase is called when entering the tableSourceBase production.
	EnterTableSourceBase(c *TableSourceBaseContext)

	// EnterTableSourceNested is called when entering the tableSourceNested production.
	EnterTableSourceNested(c *TableSourceNestedContext)

	// EnterTableJson is called when entering the tableJson production.
	EnterTableJson(c *TableJsonContext)

	// EnterAtomTableItem is called when entering the atomTableItem production.
	EnterAtomTableItem(c *AtomTableItemContext)

	// EnterSubqueryTableItem is called when entering the subqueryTableItem production.
	EnterSubqueryTableItem(c *SubqueryTableItemContext)

	// EnterTableSourcesItem is called when entering the tableSourcesItem production.
	EnterTableSourcesItem(c *TableSourcesItemContext)

	// EnterIndexHint is called when entering the indexHint production.
	EnterIndexHint(c *IndexHintContext)

	// EnterIndexHintType is called when entering the indexHintType production.
	EnterIndexHintType(c *IndexHintTypeContext)

	// EnterInnerJoin is called when entering the innerJoin production.
	EnterInnerJoin(c *InnerJoinContext)

	// EnterStraightJoin is called when entering the straightJoin production.
	EnterStraightJoin(c *StraightJoinContext)

	// EnterOuterJoin is called when entering the outerJoin production.
	EnterOuterJoin(c *OuterJoinContext)

	// EnterNaturalJoin is called when entering the naturalJoin production.
	EnterNaturalJoin(c *NaturalJoinContext)

	// EnterJoinSpec is called when entering the joinSpec production.
	EnterJoinSpec(c *JoinSpecContext)

	// EnterQueryExpression is called when entering the queryExpression production.
	EnterQueryExpression(c *QueryExpressionContext)

	// EnterQueryExpressionNointo is called when entering the queryExpressionNointo production.
	EnterQueryExpressionNointo(c *QueryExpressionNointoContext)

	// EnterQuerySpecification is called when entering the querySpecification production.
	EnterQuerySpecification(c *QuerySpecificationContext)

	// EnterQuerySpecificationNointo is called when entering the querySpecificationNointo production.
	EnterQuerySpecificationNointo(c *QuerySpecificationNointoContext)

	// EnterUnionParenthesis is called when entering the unionParenthesis production.
	EnterUnionParenthesis(c *UnionParenthesisContext)

	// EnterUnionStatement is called when entering the unionStatement production.
	EnterUnionStatement(c *UnionStatementContext)

	// EnterLateralStatement is called when entering the lateralStatement production.
	EnterLateralStatement(c *LateralStatementContext)

	// EnterJsonTable is called when entering the jsonTable production.
	EnterJsonTable(c *JsonTableContext)

	// EnterJsonColumnList is called when entering the jsonColumnList production.
	EnterJsonColumnList(c *JsonColumnListContext)

	// EnterJsonColumn is called when entering the jsonColumn production.
	EnterJsonColumn(c *JsonColumnContext)

	// EnterJsonOnEmpty is called when entering the jsonOnEmpty production.
	EnterJsonOnEmpty(c *JsonOnEmptyContext)

	// EnterJsonOnError is called when entering the jsonOnError production.
	EnterJsonOnError(c *JsonOnErrorContext)

	// EnterSelectSpec is called when entering the selectSpec production.
	EnterSelectSpec(c *SelectSpecContext)

	// EnterSelectElements is called when entering the selectElements production.
	EnterSelectElements(c *SelectElementsContext)

	// EnterSelectStarElement is called when entering the selectStarElement production.
	EnterSelectStarElement(c *SelectStarElementContext)

	// EnterSelectColumnElement is called when entering the selectColumnElement production.
	EnterSelectColumnElement(c *SelectColumnElementContext)

	// EnterSelectFunctionElement is called when entering the selectFunctionElement production.
	EnterSelectFunctionElement(c *SelectFunctionElementContext)

	// EnterSelectExpressionElement is called when entering the selectExpressionElement production.
	EnterSelectExpressionElement(c *SelectExpressionElementContext)

	// EnterSelectIntoVariables is called when entering the selectIntoVariables production.
	EnterSelectIntoVariables(c *SelectIntoVariablesContext)

	// EnterSelectIntoDumpFile is called when entering the selectIntoDumpFile production.
	EnterSelectIntoDumpFile(c *SelectIntoDumpFileContext)

	// EnterSelectIntoTextFile is called when entering the selectIntoTextFile production.
	EnterSelectIntoTextFile(c *SelectIntoTextFileContext)

	// EnterSelectFieldsInto is called when entering the selectFieldsInto production.
	EnterSelectFieldsInto(c *SelectFieldsIntoContext)

	// EnterSelectLinesInto is called when entering the selectLinesInto production.
	EnterSelectLinesInto(c *SelectLinesIntoContext)

	// EnterFromClause is called when entering the fromClause production.
	EnterFromClause(c *FromClauseContext)

	// EnterGroupByClause is called when entering the groupByClause production.
	EnterGroupByClause(c *GroupByClauseContext)

	// EnterHavingClause is called when entering the havingClause production.
	EnterHavingClause(c *HavingClauseContext)

	// EnterWindowClause is called when entering the windowClause production.
	EnterWindowClause(c *WindowClauseContext)

	// EnterGroupByItem is called when entering the groupByItem production.
	EnterGroupByItem(c *GroupByItemContext)

	// EnterLimitClause is called when entering the limitClause production.
	EnterLimitClause(c *LimitClauseContext)

	// EnterLimitClauseAtom is called when entering the limitClauseAtom production.
	EnterLimitClauseAtom(c *LimitClauseAtomContext)

	// EnterStartTransaction is called when entering the startTransaction production.
	EnterStartTransaction(c *StartTransactionContext)

	// EnterBeginWork is called when entering the beginWork production.
	EnterBeginWork(c *BeginWorkContext)

	// EnterCommitWork is called when entering the commitWork production.
	EnterCommitWork(c *CommitWorkContext)

	// EnterRollbackWork is called when entering the rollbackWork production.
	EnterRollbackWork(c *RollbackWorkContext)

	// EnterSavepointStatement is called when entering the savepointStatement production.
	EnterSavepointStatement(c *SavepointStatementContext)

	// EnterRollbackStatement is called when entering the rollbackStatement production.
	EnterRollbackStatement(c *RollbackStatementContext)

	// EnterReleaseStatement is called when entering the releaseStatement production.
	EnterReleaseStatement(c *ReleaseStatementContext)

	// EnterLockTables is called when entering the lockTables production.
	EnterLockTables(c *LockTablesContext)

	// EnterUnlockTables is called when entering the unlockTables production.
	EnterUnlockTables(c *UnlockTablesContext)

	// EnterSetAutocommitStatement is called when entering the setAutocommitStatement production.
	EnterSetAutocommitStatement(c *SetAutocommitStatementContext)

	// EnterSetTransactionStatement is called when entering the setTransactionStatement production.
	EnterSetTransactionStatement(c *SetTransactionStatementContext)

	// EnterTransactionMode is called when entering the transactionMode production.
	EnterTransactionMode(c *TransactionModeContext)

	// EnterLockTableElement is called when entering the lockTableElement production.
	EnterLockTableElement(c *LockTableElementContext)

	// EnterLockAction is called when entering the lockAction production.
	EnterLockAction(c *LockActionContext)

	// EnterTransactionOption is called when entering the transactionOption production.
	EnterTransactionOption(c *TransactionOptionContext)

	// EnterTransactionLevel is called when entering the transactionLevel production.
	EnterTransactionLevel(c *TransactionLevelContext)

	// EnterChangeMaster is called when entering the changeMaster production.
	EnterChangeMaster(c *ChangeMasterContext)

	// EnterChangeReplicationFilter is called when entering the changeReplicationFilter production.
	EnterChangeReplicationFilter(c *ChangeReplicationFilterContext)

	// EnterPurgeBinaryLogs is called when entering the purgeBinaryLogs production.
	EnterPurgeBinaryLogs(c *PurgeBinaryLogsContext)

	// EnterResetMaster is called when entering the resetMaster production.
	EnterResetMaster(c *ResetMasterContext)

	// EnterResetSlave is called when entering the resetSlave production.
	EnterResetSlave(c *ResetSlaveContext)

	// EnterStartSlave is called when entering the startSlave production.
	EnterStartSlave(c *StartSlaveContext)

	// EnterStopSlave is called when entering the stopSlave production.
	EnterStopSlave(c *StopSlaveContext)

	// EnterStartGroupReplication is called when entering the startGroupReplication production.
	EnterStartGroupReplication(c *StartGroupReplicationContext)

	// EnterStopGroupReplication is called when entering the stopGroupReplication production.
	EnterStopGroupReplication(c *StopGroupReplicationContext)

	// EnterMasterStringOption is called when entering the masterStringOption production.
	EnterMasterStringOption(c *MasterStringOptionContext)

	// EnterMasterDecimalOption is called when entering the masterDecimalOption production.
	EnterMasterDecimalOption(c *MasterDecimalOptionContext)

	// EnterMasterBoolOption is called when entering the masterBoolOption production.
	EnterMasterBoolOption(c *MasterBoolOptionContext)

	// EnterMasterRealOption is called when entering the masterRealOption production.
	EnterMasterRealOption(c *MasterRealOptionContext)

	// EnterMasterUidListOption is called when entering the masterUidListOption production.
	EnterMasterUidListOption(c *MasterUidListOptionContext)

	// EnterStringMasterOption is called when entering the stringMasterOption production.
	EnterStringMasterOption(c *StringMasterOptionContext)

	// EnterDecimalMasterOption is called when entering the decimalMasterOption production.
	EnterDecimalMasterOption(c *DecimalMasterOptionContext)

	// EnterBoolMasterOption is called when entering the boolMasterOption production.
	EnterBoolMasterOption(c *BoolMasterOptionContext)

	// EnterChannelOption is called when entering the channelOption production.
	EnterChannelOption(c *ChannelOptionContext)

	// EnterDoDbReplication is called when entering the doDbReplication production.
	EnterDoDbReplication(c *DoDbReplicationContext)

	// EnterIgnoreDbReplication is called when entering the ignoreDbReplication production.
	EnterIgnoreDbReplication(c *IgnoreDbReplicationContext)

	// EnterDoTableReplication is called when entering the doTableReplication production.
	EnterDoTableReplication(c *DoTableReplicationContext)

	// EnterIgnoreTableReplication is called when entering the ignoreTableReplication production.
	EnterIgnoreTableReplication(c *IgnoreTableReplicationContext)

	// EnterWildDoTableReplication is called when entering the wildDoTableReplication production.
	EnterWildDoTableReplication(c *WildDoTableReplicationContext)

	// EnterWildIgnoreTableReplication is called when entering the wildIgnoreTableReplication production.
	EnterWildIgnoreTableReplication(c *WildIgnoreTableReplicationContext)

	// EnterRewriteDbReplication is called when entering the rewriteDbReplication production.
	EnterRewriteDbReplication(c *RewriteDbReplicationContext)

	// EnterTablePair is called when entering the tablePair production.
	EnterTablePair(c *TablePairContext)

	// EnterThreadType is called when entering the threadType production.
	EnterThreadType(c *ThreadTypeContext)

	// EnterGtidsUntilOption is called when entering the gtidsUntilOption production.
	EnterGtidsUntilOption(c *GtidsUntilOptionContext)

	// EnterMasterLogUntilOption is called when entering the masterLogUntilOption production.
	EnterMasterLogUntilOption(c *MasterLogUntilOptionContext)

	// EnterRelayLogUntilOption is called when entering the relayLogUntilOption production.
	EnterRelayLogUntilOption(c *RelayLogUntilOptionContext)

	// EnterSqlGapsUntilOption is called when entering the sqlGapsUntilOption production.
	EnterSqlGapsUntilOption(c *SqlGapsUntilOptionContext)

	// EnterUserConnectionOption is called when entering the userConnectionOption production.
	EnterUserConnectionOption(c *UserConnectionOptionContext)

	// EnterPasswordConnectionOption is called when entering the passwordConnectionOption production.
	EnterPasswordConnectionOption(c *PasswordConnectionOptionContext)

	// EnterDefaultAuthConnectionOption is called when entering the defaultAuthConnectionOption production.
	EnterDefaultAuthConnectionOption(c *DefaultAuthConnectionOptionContext)

	// EnterPluginDirConnectionOption is called when entering the pluginDirConnectionOption production.
	EnterPluginDirConnectionOption(c *PluginDirConnectionOptionContext)

	// EnterGtuidSet is called when entering the gtuidSet production.
	EnterGtuidSet(c *GtuidSetContext)

	// EnterXaStartTransaction is called when entering the xaStartTransaction production.
	EnterXaStartTransaction(c *XaStartTransactionContext)

	// EnterXaEndTransaction is called when entering the xaEndTransaction production.
	EnterXaEndTransaction(c *XaEndTransactionContext)

	// EnterXaPrepareStatement is called when entering the xaPrepareStatement production.
	EnterXaPrepareStatement(c *XaPrepareStatementContext)

	// EnterXaCommitWork is called when entering the xaCommitWork production.
	EnterXaCommitWork(c *XaCommitWorkContext)

	// EnterXaRollbackWork is called when entering the xaRollbackWork production.
	EnterXaRollbackWork(c *XaRollbackWorkContext)

	// EnterXaRecoverWork is called when entering the xaRecoverWork production.
	EnterXaRecoverWork(c *XaRecoverWorkContext)

	// EnterPrepareStatement is called when entering the prepareStatement production.
	EnterPrepareStatement(c *PrepareStatementContext)

	// EnterExecuteStatement is called when entering the executeStatement production.
	EnterExecuteStatement(c *ExecuteStatementContext)

	// EnterDeallocatePrepare is called when entering the deallocatePrepare production.
	EnterDeallocatePrepare(c *DeallocatePrepareContext)

	// EnterRoutineBody is called when entering the routineBody production.
	EnterRoutineBody(c *RoutineBodyContext)

	// EnterBlockStatement is called when entering the blockStatement production.
	EnterBlockStatement(c *BlockStatementContext)

	// EnterCaseStatement is called when entering the caseStatement production.
	EnterCaseStatement(c *CaseStatementContext)

	// EnterIfStatement is called when entering the ifStatement production.
	EnterIfStatement(c *IfStatementContext)

	// EnterIterateStatement is called when entering the iterateStatement production.
	EnterIterateStatement(c *IterateStatementContext)

	// EnterLeaveStatement is called when entering the leaveStatement production.
	EnterLeaveStatement(c *LeaveStatementContext)

	// EnterLoopStatement is called when entering the loopStatement production.
	EnterLoopStatement(c *LoopStatementContext)

	// EnterRepeatStatement is called when entering the repeatStatement production.
	EnterRepeatStatement(c *RepeatStatementContext)

	// EnterReturnStatement is called when entering the returnStatement production.
	EnterReturnStatement(c *ReturnStatementContext)

	// EnterWhileStatement is called when entering the whileStatement production.
	EnterWhileStatement(c *WhileStatementContext)

	// EnterCloseCursor is called when entering the CloseCursor production.
	EnterCloseCursor(c *CloseCursorContext)

	// EnterFetchCursor is called when entering the FetchCursor production.
	EnterFetchCursor(c *FetchCursorContext)

	// EnterOpenCursor is called when entering the OpenCursor production.
	EnterOpenCursor(c *OpenCursorContext)

	// EnterDeclareVariable is called when entering the declareVariable production.
	EnterDeclareVariable(c *DeclareVariableContext)

	// EnterDeclareCondition is called when entering the declareCondition production.
	EnterDeclareCondition(c *DeclareConditionContext)

	// EnterDeclareCursor is called when entering the declareCursor production.
	EnterDeclareCursor(c *DeclareCursorContext)

	// EnterDeclareHandler is called when entering the declareHandler production.
	EnterDeclareHandler(c *DeclareHandlerContext)

	// EnterHandlerConditionCode is called when entering the handlerConditionCode production.
	EnterHandlerConditionCode(c *HandlerConditionCodeContext)

	// EnterHandlerConditionState is called when entering the handlerConditionState production.
	EnterHandlerConditionState(c *HandlerConditionStateContext)

	// EnterHandlerConditionName is called when entering the handlerConditionName production.
	EnterHandlerConditionName(c *HandlerConditionNameContext)

	// EnterHandlerConditionWarning is called when entering the handlerConditionWarning production.
	EnterHandlerConditionWarning(c *HandlerConditionWarningContext)

	// EnterHandlerConditionNotfound is called when entering the handlerConditionNotfound production.
	EnterHandlerConditionNotfound(c *HandlerConditionNotfoundContext)

	// EnterHandlerConditionException is called when entering the handlerConditionException production.
	EnterHandlerConditionException(c *HandlerConditionExceptionContext)

	// EnterProcedureSqlStatement is called when entering the procedureSqlStatement production.
	EnterProcedureSqlStatement(c *ProcedureSqlStatementContext)

	// EnterCaseAlternative is called when entering the caseAlternative production.
	EnterCaseAlternative(c *CaseAlternativeContext)

	// EnterElifAlternative is called when entering the elifAlternative production.
	EnterElifAlternative(c *ElifAlternativeContext)

	// EnterAlterUserMysqlV56 is called when entering the alterUserMysqlV56 production.
	EnterAlterUserMysqlV56(c *AlterUserMysqlV56Context)

	// EnterAlterUserMysqlV80 is called when entering the alterUserMysqlV80 production.
	EnterAlterUserMysqlV80(c *AlterUserMysqlV80Context)

	// EnterCreateUserMysqlV56 is called when entering the createUserMysqlV56 production.
	EnterCreateUserMysqlV56(c *CreateUserMysqlV56Context)

	// EnterCreateUserMysqlV80 is called when entering the createUserMysqlV80 production.
	EnterCreateUserMysqlV80(c *CreateUserMysqlV80Context)

	// EnterDropUser is called when entering the dropUser production.
	EnterDropUser(c *DropUserContext)

	// EnterGrantStatement is called when entering the grantStatement production.
	EnterGrantStatement(c *GrantStatementContext)

	// EnterRoleOption is called when entering the roleOption production.
	EnterRoleOption(c *RoleOptionContext)

	// EnterGrantProxy is called when entering the grantProxy production.
	EnterGrantProxy(c *GrantProxyContext)

	// EnterRenameUser is called when entering the renameUser production.
	EnterRenameUser(c *RenameUserContext)

	// EnterDetailRevoke is called when entering the detailRevoke production.
	EnterDetailRevoke(c *DetailRevokeContext)

	// EnterShortRevoke is called when entering the shortRevoke production.
	EnterShortRevoke(c *ShortRevokeContext)

	// EnterRoleRevoke is called when entering the roleRevoke production.
	EnterRoleRevoke(c *RoleRevokeContext)

	// EnterRevokeProxy is called when entering the revokeProxy production.
	EnterRevokeProxy(c *RevokeProxyContext)

	// EnterSetPasswordStatement is called when entering the setPasswordStatement production.
	EnterSetPasswordStatement(c *SetPasswordStatementContext)

	// EnterUserSpecification is called when entering the userSpecification production.
	EnterUserSpecification(c *UserSpecificationContext)

	// EnterHashAuthOption is called when entering the hashAuthOption production.
	EnterHashAuthOption(c *HashAuthOptionContext)

	// EnterRandomAuthOption is called when entering the randomAuthOption production.
	EnterRandomAuthOption(c *RandomAuthOptionContext)

	// EnterStringAuthOption is called when entering the stringAuthOption production.
	EnterStringAuthOption(c *StringAuthOptionContext)

	// EnterModuleAuthOption is called when entering the moduleAuthOption production.
	EnterModuleAuthOption(c *ModuleAuthOptionContext)

	// EnterSimpleAuthOption is called when entering the simpleAuthOption production.
	EnterSimpleAuthOption(c *SimpleAuthOptionContext)

	// EnterAuthOptionClause is called when entering the authOptionClause production.
	EnterAuthOptionClause(c *AuthOptionClauseContext)

	// EnterModule is called when entering the module production.
	EnterModule(c *ModuleContext)

	// EnterPasswordModuleOption is called when entering the passwordModuleOption production.
	EnterPasswordModuleOption(c *PasswordModuleOptionContext)

	// EnterTlsOption is called when entering the tlsOption production.
	EnterTlsOption(c *TlsOptionContext)

	// EnterUserResourceOption is called when entering the userResourceOption production.
	EnterUserResourceOption(c *UserResourceOptionContext)

	// EnterUserPasswordOption is called when entering the userPasswordOption production.
	EnterUserPasswordOption(c *UserPasswordOptionContext)

	// EnterUserLockOption is called when entering the userLockOption production.
	EnterUserLockOption(c *UserLockOptionContext)

	// EnterPrivelegeClause is called when entering the privelegeClause production.
	EnterPrivelegeClause(c *PrivelegeClauseContext)

	// EnterPrivilege is called when entering the privilege production.
	EnterPrivilege(c *PrivilegeContext)

	// EnterCurrentSchemaPriviLevel is called when entering the currentSchemaPriviLevel production.
	EnterCurrentSchemaPriviLevel(c *CurrentSchemaPriviLevelContext)

	// EnterGlobalPrivLevel is called when entering the globalPrivLevel production.
	EnterGlobalPrivLevel(c *GlobalPrivLevelContext)

	// EnterDefiniteSchemaPrivLevel is called when entering the definiteSchemaPrivLevel production.
	EnterDefiniteSchemaPrivLevel(c *DefiniteSchemaPrivLevelContext)

	// EnterDefiniteFullTablePrivLevel is called when entering the definiteFullTablePrivLevel production.
	EnterDefiniteFullTablePrivLevel(c *DefiniteFullTablePrivLevelContext)

	// EnterDefiniteFullTablePrivLevel2 is called when entering the definiteFullTablePrivLevel2 production.
	EnterDefiniteFullTablePrivLevel2(c *DefiniteFullTablePrivLevel2Context)

	// EnterDefiniteTablePrivLevel is called when entering the definiteTablePrivLevel production.
	EnterDefiniteTablePrivLevel(c *DefiniteTablePrivLevelContext)

	// EnterRenameUserClause is called when entering the renameUserClause production.
	EnterRenameUserClause(c *RenameUserClauseContext)

	// EnterAnalyzeTable is called when entering the analyzeTable production.
	EnterAnalyzeTable(c *AnalyzeTableContext)

	// EnterCheckTable is called when entering the checkTable production.
	EnterCheckTable(c *CheckTableContext)

	// EnterChecksumTable is called when entering the checksumTable production.
	EnterChecksumTable(c *ChecksumTableContext)

	// EnterOptimizeTable is called when entering the optimizeTable production.
	EnterOptimizeTable(c *OptimizeTableContext)

	// EnterRepairTable is called when entering the repairTable production.
	EnterRepairTable(c *RepairTableContext)

	// EnterCheckTableOption is called when entering the checkTableOption production.
	EnterCheckTableOption(c *CheckTableOptionContext)

	// EnterCreateUdfunction is called when entering the createUdfunction production.
	EnterCreateUdfunction(c *CreateUdfunctionContext)

	// EnterInstallPlugin is called when entering the installPlugin production.
	EnterInstallPlugin(c *InstallPluginContext)

	// EnterUninstallPlugin is called when entering the uninstallPlugin production.
	EnterUninstallPlugin(c *UninstallPluginContext)

	// EnterSetVariable is called when entering the setVariable production.
	EnterSetVariable(c *SetVariableContext)

	// EnterSetCharset is called when entering the setCharset production.
	EnterSetCharset(c *SetCharsetContext)

	// EnterSetNames is called when entering the setNames production.
	EnterSetNames(c *SetNamesContext)

	// EnterSetPassword is called when entering the setPassword production.
	EnterSetPassword(c *SetPasswordContext)

	// EnterSetTransaction is called when entering the setTransaction production.
	EnterSetTransaction(c *SetTransactionContext)

	// EnterSetAutocommit is called when entering the setAutocommit production.
	EnterSetAutocommit(c *SetAutocommitContext)

	// EnterSetNewValueInsideTrigger is called when entering the setNewValueInsideTrigger production.
	EnterSetNewValueInsideTrigger(c *SetNewValueInsideTriggerContext)

	// EnterShowMasterLogs is called when entering the showMasterLogs production.
	EnterShowMasterLogs(c *ShowMasterLogsContext)

	// EnterShowLogEvents is called when entering the showLogEvents production.
	EnterShowLogEvents(c *ShowLogEventsContext)

	// EnterShowObjectFilter is called when entering the showObjectFilter production.
	EnterShowObjectFilter(c *ShowObjectFilterContext)

	// EnterShowColumns is called when entering the showColumns production.
	EnterShowColumns(c *ShowColumnsContext)

	// EnterShowCreateDb is called when entering the showCreateDb production.
	EnterShowCreateDb(c *ShowCreateDbContext)

	// EnterShowCreateFullIdObject is called when entering the showCreateFullIdObject production.
	EnterShowCreateFullIdObject(c *ShowCreateFullIdObjectContext)

	// EnterShowCreateUser is called when entering the showCreateUser production.
	EnterShowCreateUser(c *ShowCreateUserContext)

	// EnterShowEngine is called when entering the showEngine production.
	EnterShowEngine(c *ShowEngineContext)

	// EnterShowGlobalInfo is called when entering the showGlobalInfo production.
	EnterShowGlobalInfo(c *ShowGlobalInfoContext)

	// EnterShowErrors is called when entering the showErrors production.
	EnterShowErrors(c *ShowErrorsContext)

	// EnterShowCountErrors is called when entering the showCountErrors production.
	EnterShowCountErrors(c *ShowCountErrorsContext)

	// EnterShowSchemaFilter is called when entering the showSchemaFilter production.
	EnterShowSchemaFilter(c *ShowSchemaFilterContext)

	// EnterShowRoutine is called when entering the showRoutine production.
	EnterShowRoutine(c *ShowRoutineContext)

	// EnterShowGrants is called when entering the showGrants production.
	EnterShowGrants(c *ShowGrantsContext)

	// EnterShowIndexes is called when entering the showIndexes production.
	EnterShowIndexes(c *ShowIndexesContext)

	// EnterShowOpenTables is called when entering the showOpenTables production.
	EnterShowOpenTables(c *ShowOpenTablesContext)

	// EnterShowProfile is called when entering the showProfile production.
	EnterShowProfile(c *ShowProfileContext)

	// EnterShowSlaveStatus is called when entering the showSlaveStatus production.
	EnterShowSlaveStatus(c *ShowSlaveStatusContext)

	// EnterVariableClause is called when entering the variableClause production.
	EnterVariableClause(c *VariableClauseContext)

	// EnterShowCommonEntity is called when entering the showCommonEntity production.
	EnterShowCommonEntity(c *ShowCommonEntityContext)

	// EnterShowFilter is called when entering the showFilter production.
	EnterShowFilter(c *ShowFilterContext)

	// EnterShowGlobalInfoClause is called when entering the showGlobalInfoClause production.
	EnterShowGlobalInfoClause(c *ShowGlobalInfoClauseContext)

	// EnterShowSchemaEntity is called when entering the showSchemaEntity production.
	EnterShowSchemaEntity(c *ShowSchemaEntityContext)

	// EnterShowProfileType is called when entering the showProfileType production.
	EnterShowProfileType(c *ShowProfileTypeContext)

	// EnterBinlogStatement is called when entering the binlogStatement production.
	EnterBinlogStatement(c *BinlogStatementContext)

	// EnterCacheIndexStatement is called when entering the cacheIndexStatement production.
	EnterCacheIndexStatement(c *CacheIndexStatementContext)

	// EnterFlushStatement is called when entering the flushStatement production.
	EnterFlushStatement(c *FlushStatementContext)

	// EnterKillStatement is called when entering the killStatement production.
	EnterKillStatement(c *KillStatementContext)

	// EnterLoadIndexIntoCache is called when entering the loadIndexIntoCache production.
	EnterLoadIndexIntoCache(c *LoadIndexIntoCacheContext)

	// EnterResetStatement is called when entering the resetStatement production.
	EnterResetStatement(c *ResetStatementContext)

	// EnterShutdownStatement is called when entering the shutdownStatement production.
	EnterShutdownStatement(c *ShutdownStatementContext)

	// EnterTableIndexes is called when entering the tableIndexes production.
	EnterTableIndexes(c *TableIndexesContext)

	// EnterSimpleFlushOption is called when entering the simpleFlushOption production.
	EnterSimpleFlushOption(c *SimpleFlushOptionContext)

	// EnterChannelFlushOption is called when entering the channelFlushOption production.
	EnterChannelFlushOption(c *ChannelFlushOptionContext)

	// EnterTableFlushOption is called when entering the tableFlushOption production.
	EnterTableFlushOption(c *TableFlushOptionContext)

	// EnterFlushTableOption is called when entering the flushTableOption production.
	EnterFlushTableOption(c *FlushTableOptionContext)

	// EnterLoadedTableIndexes is called when entering the loadedTableIndexes production.
	EnterLoadedTableIndexes(c *LoadedTableIndexesContext)

	// EnterSimpleDescribeStatement is called when entering the simpleDescribeStatement production.
	EnterSimpleDescribeStatement(c *SimpleDescribeStatementContext)

	// EnterFullDescribeStatement is called when entering the fullDescribeStatement production.
	EnterFullDescribeStatement(c *FullDescribeStatementContext)

	// EnterHelpStatement is called when entering the helpStatement production.
	EnterHelpStatement(c *HelpStatementContext)

	// EnterUseStatement is called when entering the useStatement production.
	EnterUseStatement(c *UseStatementContext)

	// EnterSignalStatement is called when entering the signalStatement production.
	EnterSignalStatement(c *SignalStatementContext)

	// EnterResignalStatement is called when entering the resignalStatement production.
	EnterResignalStatement(c *ResignalStatementContext)

	// EnterSignalConditionInformation is called when entering the signalConditionInformation production.
	EnterSignalConditionInformation(c *SignalConditionInformationContext)

	// EnterWithStatement is called when entering the withStatement production.
	EnterWithStatement(c *WithStatementContext)

	// EnterTableStatement is called when entering the tableStatement production.
	EnterTableStatement(c *TableStatementContext)

	// EnterDiagnosticsStatement is called when entering the diagnosticsStatement production.
	EnterDiagnosticsStatement(c *DiagnosticsStatementContext)

	// EnterDiagnosticsConditionInformationName is called when entering the diagnosticsConditionInformationName production.
	EnterDiagnosticsConditionInformationName(c *DiagnosticsConditionInformationNameContext)

	// EnterDescribeStatements is called when entering the describeStatements production.
	EnterDescribeStatements(c *DescribeStatementsContext)

	// EnterDescribeConnection is called when entering the describeConnection production.
	EnterDescribeConnection(c *DescribeConnectionContext)

	// EnterFullId is called when entering the fullId production.
	EnterFullId(c *FullIdContext)

	// EnterTableName is called when entering the tableName production.
	EnterTableName(c *TableNameContext)

	// EnterRoleName is called when entering the roleName production.
	EnterRoleName(c *RoleNameContext)

	// EnterFullColumnName is called when entering the fullColumnName production.
	EnterFullColumnName(c *FullColumnNameContext)

	// EnterIndexColumnName is called when entering the indexColumnName production.
	EnterIndexColumnName(c *IndexColumnNameContext)

	// EnterSimpleUserName is called when entering the simpleUserName production.
	EnterSimpleUserName(c *SimpleUserNameContext)

	// EnterHostName is called when entering the hostName production.
	EnterHostName(c *HostNameContext)

	// EnterUserName is called when entering the userName production.
	EnterUserName(c *UserNameContext)

	// EnterMysqlVariable is called when entering the mysqlVariable production.
	EnterMysqlVariable(c *MysqlVariableContext)

	// EnterCharsetName is called when entering the charsetName production.
	EnterCharsetName(c *CharsetNameContext)

	// EnterCollationName is called when entering the collationName production.
	EnterCollationName(c *CollationNameContext)

	// EnterEngineName is called when entering the engineName production.
	EnterEngineName(c *EngineNameContext)

	// EnterEngineNameBase is called when entering the engineNameBase production.
	EnterEngineNameBase(c *EngineNameBaseContext)

	// EnterUuidSet is called when entering the uuidSet production.
	EnterUuidSet(c *UuidSetContext)

	// EnterXid is called when entering the xid production.
	EnterXid(c *XidContext)

	// EnterXuidStringId is called when entering the xuidStringId production.
	EnterXuidStringId(c *XuidStringIdContext)

	// EnterAuthPlugin is called when entering the authPlugin production.
	EnterAuthPlugin(c *AuthPluginContext)

	// EnterUid is called when entering the uid production.
	EnterUid(c *UidContext)

	// EnterSimpleId is called when entering the simpleId production.
	EnterSimpleId(c *SimpleIdContext)

	// EnterDottedId is called when entering the dottedId production.
	EnterDottedId(c *DottedIdContext)

	// EnterDecimalLiteral is called when entering the decimalLiteral production.
	EnterDecimalLiteral(c *DecimalLiteralContext)

	// EnterFileSizeLiteral is called when entering the fileSizeLiteral production.
	EnterFileSizeLiteral(c *FileSizeLiteralContext)

	// EnterStringLiteral is called when entering the stringLiteral production.
	EnterStringLiteral(c *StringLiteralContext)

	// EnterBooleanLiteral is called when entering the booleanLiteral production.
	EnterBooleanLiteral(c *BooleanLiteralContext)

	// EnterHexadecimalLiteral is called when entering the hexadecimalLiteral production.
	EnterHexadecimalLiteral(c *HexadecimalLiteralContext)

	// EnterNullNotnull is called when entering the nullNotnull production.
	EnterNullNotnull(c *NullNotnullContext)

	// EnterConstant is called when entering the constant production.
	EnterConstant(c *ConstantContext)

	// EnterStringDataType is called when entering the stringDataType production.
	EnterStringDataType(c *StringDataTypeContext)

	// EnterNationalVaryingStringDataType is called when entering the nationalVaryingStringDataType production.
	EnterNationalVaryingStringDataType(c *NationalVaryingStringDataTypeContext)

	// EnterNationalStringDataType is called when entering the nationalStringDataType production.
	EnterNationalStringDataType(c *NationalStringDataTypeContext)

	// EnterDimensionDataType is called when entering the dimensionDataType production.
	EnterDimensionDataType(c *DimensionDataTypeContext)

	// EnterSimpleDataType is called when entering the simpleDataType production.
	EnterSimpleDataType(c *SimpleDataTypeContext)

	// EnterCollectionDataType is called when entering the collectionDataType production.
	EnterCollectionDataType(c *CollectionDataTypeContext)

	// EnterSpatialDataType is called when entering the spatialDataType production.
	EnterSpatialDataType(c *SpatialDataTypeContext)

	// EnterLongVarcharDataType is called when entering the longVarcharDataType production.
	EnterLongVarcharDataType(c *LongVarcharDataTypeContext)

	// EnterLongVarbinaryDataType is called when entering the longVarbinaryDataType production.
	EnterLongVarbinaryDataType(c *LongVarbinaryDataTypeContext)

	// EnterCollectionOptions is called when entering the collectionOptions production.
	EnterCollectionOptions(c *CollectionOptionsContext)

	// EnterConvertedDataType is called when entering the convertedDataType production.
	EnterConvertedDataType(c *ConvertedDataTypeContext)

	// EnterLengthOneDimension is called when entering the lengthOneDimension production.
	EnterLengthOneDimension(c *LengthOneDimensionContext)

	// EnterLengthTwoDimension is called when entering the lengthTwoDimension production.
	EnterLengthTwoDimension(c *LengthTwoDimensionContext)

	// EnterLengthTwoOptionalDimension is called when entering the lengthTwoOptionalDimension production.
	EnterLengthTwoOptionalDimension(c *LengthTwoOptionalDimensionContext)

	// EnterUidList is called when entering the uidList production.
	EnterUidList(c *UidListContext)

	// EnterFullColumnNameList is called when entering the fullColumnNameList production.
	EnterFullColumnNameList(c *FullColumnNameListContext)

	// EnterTables is called when entering the tables production.
	EnterTables(c *TablesContext)

	// EnterIndexColumnNames is called when entering the indexColumnNames production.
	EnterIndexColumnNames(c *IndexColumnNamesContext)

	// EnterExpressions is called when entering the expressions production.
	EnterExpressions(c *ExpressionsContext)

	// EnterExpressionsWithDefaults is called when entering the expressionsWithDefaults production.
	EnterExpressionsWithDefaults(c *ExpressionsWithDefaultsContext)

	// EnterConstants is called when entering the constants production.
	EnterConstants(c *ConstantsContext)

	// EnterSimpleStrings is called when entering the simpleStrings production.
	EnterSimpleStrings(c *SimpleStringsContext)

	// EnterUserVariables is called when entering the userVariables production.
	EnterUserVariables(c *UserVariablesContext)

	// EnterDefaultValue is called when entering the defaultValue production.
	EnterDefaultValue(c *DefaultValueContext)

	// EnterCurrentTimestamp is called when entering the currentTimestamp production.
	EnterCurrentTimestamp(c *CurrentTimestampContext)

	// EnterExpressionOrDefault is called when entering the expressionOrDefault production.
	EnterExpressionOrDefault(c *ExpressionOrDefaultContext)

	// EnterIfExists is called when entering the ifExists production.
	EnterIfExists(c *IfExistsContext)

	// EnterIfNotExists is called when entering the ifNotExists production.
	EnterIfNotExists(c *IfNotExistsContext)

	// EnterOrReplace is called when entering the orReplace production.
	EnterOrReplace(c *OrReplaceContext)

	// EnterWaitNowaitClause is called when entering the waitNowaitClause production.
	EnterWaitNowaitClause(c *WaitNowaitClauseContext)

	// EnterSpecificFunctionCall is called when entering the specificFunctionCall production.
	EnterSpecificFunctionCall(c *SpecificFunctionCallContext)

	// EnterAggregateFunctionCall is called when entering the aggregateFunctionCall production.
	EnterAggregateFunctionCall(c *AggregateFunctionCallContext)

	// EnterNonAggregateFunctionCall is called when entering the nonAggregateFunctionCall production.
	EnterNonAggregateFunctionCall(c *NonAggregateFunctionCallContext)

	// EnterScalarFunctionCall is called when entering the scalarFunctionCall production.
	EnterScalarFunctionCall(c *ScalarFunctionCallContext)

	// EnterUdfFunctionCall is called when entering the udfFunctionCall production.
	EnterUdfFunctionCall(c *UdfFunctionCallContext)

	// EnterPasswordFunctionCall is called when entering the passwordFunctionCall production.
	EnterPasswordFunctionCall(c *PasswordFunctionCallContext)

	// EnterSimpleFunctionCall is called when entering the simpleFunctionCall production.
	EnterSimpleFunctionCall(c *SimpleFunctionCallContext)

	// EnterCurrentUser is called when entering the currentUser production.
	EnterCurrentUser(c *CurrentUserContext)

	// EnterDataTypeFunctionCall is called when entering the dataTypeFunctionCall production.
	EnterDataTypeFunctionCall(c *DataTypeFunctionCallContext)

	// EnterValuesFunctionCall is called when entering the valuesFunctionCall production.
	EnterValuesFunctionCall(c *ValuesFunctionCallContext)

	// EnterCaseExpressionFunctionCall is called when entering the caseExpressionFunctionCall production.
	EnterCaseExpressionFunctionCall(c *CaseExpressionFunctionCallContext)

	// EnterCaseFunctionCall is called when entering the caseFunctionCall production.
	EnterCaseFunctionCall(c *CaseFunctionCallContext)

	// EnterCharFunctionCall is called when entering the charFunctionCall production.
	EnterCharFunctionCall(c *CharFunctionCallContext)

	// EnterPositionFunctionCall is called when entering the positionFunctionCall production.
	EnterPositionFunctionCall(c *PositionFunctionCallContext)

	// EnterSubstrFunctionCall is called when entering the substrFunctionCall production.
	EnterSubstrFunctionCall(c *SubstrFunctionCallContext)

	// EnterTrimFunctionCall is called when entering the trimFunctionCall production.
	EnterTrimFunctionCall(c *TrimFunctionCallContext)

	// EnterWeightFunctionCall is called when entering the weightFunctionCall production.
	EnterWeightFunctionCall(c *WeightFunctionCallContext)

	// EnterExtractFunctionCall is called when entering the extractFunctionCall production.
	EnterExtractFunctionCall(c *ExtractFunctionCallContext)

	// EnterGetFormatFunctionCall is called when entering the getFormatFunctionCall production.
	EnterGetFormatFunctionCall(c *GetFormatFunctionCallContext)

	// EnterJsonValueFunctionCall is called when entering the jsonValueFunctionCall production.
	EnterJsonValueFunctionCall(c *JsonValueFunctionCallContext)

	// EnterCaseFuncAlternative is called when entering the caseFuncAlternative production.
	EnterCaseFuncAlternative(c *CaseFuncAlternativeContext)

	// EnterLevelWeightList is called when entering the levelWeightList production.
	EnterLevelWeightList(c *LevelWeightListContext)

	// EnterLevelWeightRange is called when entering the levelWeightRange production.
	EnterLevelWeightRange(c *LevelWeightRangeContext)

	// EnterLevelInWeightListElement is called when entering the levelInWeightListElement production.
	EnterLevelInWeightListElement(c *LevelInWeightListElementContext)

	// EnterAggregateWindowedFunction is called when entering the aggregateWindowedFunction production.
	EnterAggregateWindowedFunction(c *AggregateWindowedFunctionContext)

	// EnterNonAggregateWindowedFunction is called when entering the nonAggregateWindowedFunction production.
	EnterNonAggregateWindowedFunction(c *NonAggregateWindowedFunctionContext)

	// EnterOverClause is called when entering the overClause production.
	EnterOverClause(c *OverClauseContext)

	// EnterWindowSpec is called when entering the windowSpec production.
	EnterWindowSpec(c *WindowSpecContext)

	// EnterWindowName is called when entering the windowName production.
	EnterWindowName(c *WindowNameContext)

	// EnterFrameClause is called when entering the frameClause production.
	EnterFrameClause(c *FrameClauseContext)

	// EnterFrameUnits is called when entering the frameUnits production.
	EnterFrameUnits(c *FrameUnitsContext)

	// EnterFrameExtent is called when entering the frameExtent production.
	EnterFrameExtent(c *FrameExtentContext)

	// EnterFrameBetween is called when entering the frameBetween production.
	EnterFrameBetween(c *FrameBetweenContext)

	// EnterFrameRange is called when entering the frameRange production.
	EnterFrameRange(c *FrameRangeContext)

	// EnterPartitionClause is called when entering the partitionClause production.
	EnterPartitionClause(c *PartitionClauseContext)

	// EnterScalarFunctionName is called when entering the scalarFunctionName production.
	EnterScalarFunctionName(c *ScalarFunctionNameContext)

	// EnterPasswordFunctionClause is called when entering the passwordFunctionClause production.
	EnterPasswordFunctionClause(c *PasswordFunctionClauseContext)

	// EnterFunctionArgs is called when entering the functionArgs production.
	EnterFunctionArgs(c *FunctionArgsContext)

	// EnterFunctionArg is called when entering the functionArg production.
	EnterFunctionArg(c *FunctionArgContext)

	// EnterIsExpression is called when entering the isExpression production.
	EnterIsExpression(c *IsExpressionContext)

	// EnterNotExpression is called when entering the notExpression production.
	EnterNotExpression(c *NotExpressionContext)

	// EnterLogicalExpression is called when entering the logicalExpression production.
	EnterLogicalExpression(c *LogicalExpressionContext)

	// EnterPredicateExpression is called when entering the predicateExpression production.
	EnterPredicateExpression(c *PredicateExpressionContext)

	// EnterSoundsLikePredicate is called when entering the soundsLikePredicate production.
	EnterSoundsLikePredicate(c *SoundsLikePredicateContext)

	// EnterExpressionAtomPredicate is called when entering the expressionAtomPredicate production.
	EnterExpressionAtomPredicate(c *ExpressionAtomPredicateContext)

	// EnterSubqueryComparisonPredicate is called when entering the subqueryComparisonPredicate production.
	EnterSubqueryComparisonPredicate(c *SubqueryComparisonPredicateContext)

	// EnterJsonMemberOfPredicate is called when entering the jsonMemberOfPredicate production.
	EnterJsonMemberOfPredicate(c *JsonMemberOfPredicateContext)

	// EnterBinaryComparisonPredicate is called when entering the binaryComparisonPredicate production.
	EnterBinaryComparisonPredicate(c *BinaryComparisonPredicateContext)

	// EnterInPredicate is called when entering the inPredicate production.
	EnterInPredicate(c *InPredicateContext)

	// EnterBetweenPredicate is called when entering the betweenPredicate production.
	EnterBetweenPredicate(c *BetweenPredicateContext)

	// EnterIsNullPredicate is called when entering the isNullPredicate production.
	EnterIsNullPredicate(c *IsNullPredicateContext)

	// EnterLikePredicate is called when entering the likePredicate production.
	EnterLikePredicate(c *LikePredicateContext)

	// EnterRegexpPredicate is called when entering the regexpPredicate production.
	EnterRegexpPredicate(c *RegexpPredicateContext)

	// EnterUnaryExpressionAtom is called when entering the unaryExpressionAtom production.
	EnterUnaryExpressionAtom(c *UnaryExpressionAtomContext)

	// EnterCollateExpressionAtom is called when entering the collateExpressionAtom production.
	EnterCollateExpressionAtom(c *CollateExpressionAtomContext)

	// EnterVariableAssignExpressionAtom is called when entering the variableAssignExpressionAtom production.
	EnterVariableAssignExpressionAtom(c *VariableAssignExpressionAtomContext)

	// EnterMysqlVariableExpressionAtom is called when entering the mysqlVariableExpressionAtom production.
	EnterMysqlVariableExpressionAtom(c *MysqlVariableExpressionAtomContext)

	// EnterNestedExpressionAtom is called when entering the nestedExpressionAtom production.
	EnterNestedExpressionAtom(c *NestedExpressionAtomContext)

	// EnterNestedRowExpressionAtom is called when entering the nestedRowExpressionAtom production.
	EnterNestedRowExpressionAtom(c *NestedRowExpressionAtomContext)

	// EnterMathExpressionAtom is called when entering the mathExpressionAtom production.
	EnterMathExpressionAtom(c *MathExpressionAtomContext)

	// EnterExistsExpressionAtom is called when entering the existsExpressionAtom production.
	EnterExistsExpressionAtom(c *ExistsExpressionAtomContext)

	// EnterIntervalExpressionAtom is called when entering the intervalExpressionAtom production.
	EnterIntervalExpressionAtom(c *IntervalExpressionAtomContext)

	// EnterJsonExpressionAtom is called when entering the jsonExpressionAtom production.
	EnterJsonExpressionAtom(c *JsonExpressionAtomContext)

	// EnterSubqueryExpressionAtom is called when entering the subqueryExpressionAtom production.
	EnterSubqueryExpressionAtom(c *SubqueryExpressionAtomContext)

	// EnterConstantExpressionAtom is called when entering the constantExpressionAtom production.
	EnterConstantExpressionAtom(c *ConstantExpressionAtomContext)

	// EnterFunctionCallExpressionAtom is called when entering the functionCallExpressionAtom production.
	EnterFunctionCallExpressionAtom(c *FunctionCallExpressionAtomContext)

	// EnterBinaryExpressionAtom is called when entering the binaryExpressionAtom production.
	EnterBinaryExpressionAtom(c *BinaryExpressionAtomContext)

	// EnterFullColumnNameExpressionAtom is called when entering the fullColumnNameExpressionAtom production.
	EnterFullColumnNameExpressionAtom(c *FullColumnNameExpressionAtomContext)

	// EnterBitExpressionAtom is called when entering the bitExpressionAtom production.
	EnterBitExpressionAtom(c *BitExpressionAtomContext)

	// EnterUnaryOperator is called when entering the unaryOperator production.
	EnterUnaryOperator(c *UnaryOperatorContext)

	// EnterComparisonOperator is called when entering the comparisonOperator production.
	EnterComparisonOperator(c *ComparisonOperatorContext)

	// EnterLogicalOperator is called when entering the logicalOperator production.
	EnterLogicalOperator(c *LogicalOperatorContext)

	// EnterBitOperator is called when entering the bitOperator production.
	EnterBitOperator(c *BitOperatorContext)

	// EnterMultOperator is called when entering the multOperator production.
	EnterMultOperator(c *MultOperatorContext)

	// EnterAddOperator is called when entering the addOperator production.
	EnterAddOperator(c *AddOperatorContext)

	// EnterJsonOperator is called when entering the jsonOperator production.
	EnterJsonOperator(c *JsonOperatorContext)

	// EnterCharsetNameBase is called when entering the charsetNameBase production.
	EnterCharsetNameBase(c *CharsetNameBaseContext)

	// EnterTransactionLevelBase is called when entering the transactionLevelBase production.
	EnterTransactionLevelBase(c *TransactionLevelBaseContext)

	// EnterPrivilegesBase is called when entering the privilegesBase production.
	EnterPrivilegesBase(c *PrivilegesBaseContext)

	// EnterIntervalTypeBase is called when entering the intervalTypeBase production.
	EnterIntervalTypeBase(c *IntervalTypeBaseContext)

	// EnterDataTypeBase is called when entering the dataTypeBase production.
	EnterDataTypeBase(c *DataTypeBaseContext)

	// EnterKeywordsCanBeId is called when entering the keywordsCanBeId production.
	EnterKeywordsCanBeId(c *KeywordsCanBeIdContext)

	// EnterFunctionNameBase is called when entering the functionNameBase production.
	EnterFunctionNameBase(c *FunctionNameBaseContext)

	// ExitRoot is called when exiting the root production.
	ExitRoot(c *RootContext)

	// ExitSqlStatements is called when exiting the sqlStatements production.
	ExitSqlStatements(c *SqlStatementsContext)

	// ExitSqlStatement is called when exiting the sqlStatement production.
	ExitSqlStatement(c *SqlStatementContext)

	// ExitEmptyStatement_ is called when exiting the emptyStatement_ production.
	ExitEmptyStatement_(c *EmptyStatement_Context)

	// ExitDdlStatement is called when exiting the ddlStatement production.
	ExitDdlStatement(c *DdlStatementContext)

	// ExitDmlStatement is called when exiting the dmlStatement production.
	ExitDmlStatement(c *DmlStatementContext)

	// ExitTransactionStatement is called when exiting the transactionStatement production.
	ExitTransactionStatement(c *TransactionStatementContext)

	// ExitReplicationStatement is called when exiting the replicationStatement production.
	ExitReplicationStatement(c *ReplicationStatementContext)

	// ExitPreparedStatement is called when exiting the preparedStatement production.
	ExitPreparedStatement(c *PreparedStatementContext)

	// ExitCompoundStatement is called when exiting the compoundStatement production.
	ExitCompoundStatement(c *CompoundStatementContext)

	// ExitAdministrationStatement is called when exiting the administrationStatement production.
	ExitAdministrationStatement(c *AdministrationStatementContext)

	// ExitUtilityStatement is called when exiting the utilityStatement production.
	ExitUtilityStatement(c *UtilityStatementContext)

	// ExitCreateDatabase is called when exiting the createDatabase production.
	ExitCreateDatabase(c *CreateDatabaseContext)

	// ExitCreateEvent is called when exiting the createEvent production.
	ExitCreateEvent(c *CreateEventContext)

	// ExitCreateIndex is called when exiting the createIndex production.
	ExitCreateIndex(c *CreateIndexContext)

	// ExitCreateLogfileGroup is called when exiting the createLogfileGroup production.
	ExitCreateLogfileGroup(c *CreateLogfileGroupContext)

	// ExitCreateProcedure is called when exiting the createProcedure production.
	ExitCreateProcedure(c *CreateProcedureContext)

	// ExitCreateFunction is called when exiting the createFunction production.
	ExitCreateFunction(c *CreateFunctionContext)

	// ExitCreateRole is called when exiting the createRole production.
	ExitCreateRole(c *CreateRoleContext)

	// ExitCreateServer is called when exiting the createServer production.
	ExitCreateServer(c *CreateServerContext)

	// ExitCopyCreateTable is called when exiting the copyCreateTable production.
	ExitCopyCreateTable(c *CopyCreateTableContext)

	// ExitQueryCreateTable is called when exiting the queryCreateTable production.
	ExitQueryCreateTable(c *QueryCreateTableContext)

	// ExitColumnCreateTable is called when exiting the columnCreateTable production.
	ExitColumnCreateTable(c *ColumnCreateTableContext)

	// ExitCreateTablespaceInnodb is called when exiting the createTablespaceInnodb production.
	ExitCreateTablespaceInnodb(c *CreateTablespaceInnodbContext)

	// ExitCreateTablespaceNdb is called when exiting the createTablespaceNdb production.
	ExitCreateTablespaceNdb(c *CreateTablespaceNdbContext)

	// ExitCreateTrigger is called when exiting the createTrigger production.
	ExitCreateTrigger(c *CreateTriggerContext)

	// ExitWithClause is called when exiting the withClause production.
	ExitWithClause(c *WithClauseContext)

	// ExitCommonTableExpressions is called when exiting the commonTableExpressions production.
	ExitCommonTableExpressions(c *CommonTableExpressionsContext)

	// ExitCteName is called when exiting the cteName production.
	ExitCteName(c *CteNameContext)

	// ExitCteColumnName is called when exiting the cteColumnName production.
	ExitCteColumnName(c *CteColumnNameContext)

	// ExitCreateView is called when exiting the createView production.
	ExitCreateView(c *CreateViewContext)

	// ExitCreateDatabaseOption is called when exiting the createDatabaseOption production.
	ExitCreateDatabaseOption(c *CreateDatabaseOptionContext)

	// ExitCharSet is called when exiting the charSet production.
	ExitCharSet(c *CharSetContext)

	// ExitCurrentUserExpression is called when exiting the currentUserExpression production.
	ExitCurrentUserExpression(c *CurrentUserExpressionContext)

	// ExitOwnerStatement is called when exiting the ownerStatement production.
	ExitOwnerStatement(c *OwnerStatementContext)

	// ExitPreciseSchedule is called when exiting the preciseSchedule production.
	ExitPreciseSchedule(c *PreciseScheduleContext)

	// ExitIntervalSchedule is called when exiting the intervalSchedule production.
	ExitIntervalSchedule(c *IntervalScheduleContext)

	// ExitTimestampValue is called when exiting the timestampValue production.
	ExitTimestampValue(c *TimestampValueContext)

	// ExitIntervalExpr is called when exiting the intervalExpr production.
	ExitIntervalExpr(c *IntervalExprContext)

	// ExitIntervalType is called when exiting the intervalType production.
	ExitIntervalType(c *IntervalTypeContext)

	// ExitEnableType is called when exiting the enableType production.
	ExitEnableType(c *EnableTypeContext)

	// ExitIndexType is called when exiting the indexType production.
	ExitIndexType(c *IndexTypeContext)

	// ExitIndexOption is called when exiting the indexOption production.
	ExitIndexOption(c *IndexOptionContext)

	// ExitProcedureParameter is called when exiting the procedureParameter production.
	ExitProcedureParameter(c *ProcedureParameterContext)

	// ExitFunctionParameter is called when exiting the functionParameter production.
	ExitFunctionParameter(c *FunctionParameterContext)

	// ExitRoutineComment is called when exiting the routineComment production.
	ExitRoutineComment(c *RoutineCommentContext)

	// ExitRoutineLanguage is called when exiting the routineLanguage production.
	ExitRoutineLanguage(c *RoutineLanguageContext)

	// ExitRoutineBehavior is called when exiting the routineBehavior production.
	ExitRoutineBehavior(c *RoutineBehaviorContext)

	// ExitRoutineData is called when exiting the routineData production.
	ExitRoutineData(c *RoutineDataContext)

	// ExitRoutineSecurity is called when exiting the routineSecurity production.
	ExitRoutineSecurity(c *RoutineSecurityContext)

	// ExitServerOption is called when exiting the serverOption production.
	ExitServerOption(c *ServerOptionContext)

	// ExitCreateDefinitions is called when exiting the createDefinitions production.
	ExitCreateDefinitions(c *CreateDefinitionsContext)

	// ExitColumnDeclaration is called when exiting the columnDeclaration production.
	ExitColumnDeclaration(c *ColumnDeclarationContext)

	// ExitConstraintDeclaration is called when exiting the constraintDeclaration production.
	ExitConstraintDeclaration(c *ConstraintDeclarationContext)

	// ExitIndexDeclaration is called when exiting the indexDeclaration production.
	ExitIndexDeclaration(c *IndexDeclarationContext)

	// ExitColumnDefinition is called when exiting the columnDefinition production.
	ExitColumnDefinition(c *ColumnDefinitionContext)

	// ExitNullColumnConstraint is called when exiting the nullColumnConstraint production.
	ExitNullColumnConstraint(c *NullColumnConstraintContext)

	// ExitDefaultColumnConstraint is called when exiting the defaultColumnConstraint production.
	ExitDefaultColumnConstraint(c *DefaultColumnConstraintContext)

	// ExitVisibilityColumnConstraint is called when exiting the visibilityColumnConstraint production.
	ExitVisibilityColumnConstraint(c *VisibilityColumnConstraintContext)

	// ExitInvisibilityColumnConstraint is called when exiting the invisibilityColumnConstraint production.
	ExitInvisibilityColumnConstraint(c *InvisibilityColumnConstraintContext)

	// ExitAutoIncrementColumnConstraint is called when exiting the autoIncrementColumnConstraint production.
	ExitAutoIncrementColumnConstraint(c *AutoIncrementColumnConstraintContext)

	// ExitPrimaryKeyColumnConstraint is called when exiting the primaryKeyColumnConstraint production.
	ExitPrimaryKeyColumnConstraint(c *PrimaryKeyColumnConstraintContext)

	// ExitUniqueKeyColumnConstraint is called when exiting the uniqueKeyColumnConstraint production.
	ExitUniqueKeyColumnConstraint(c *UniqueKeyColumnConstraintContext)

	// ExitCommentColumnConstraint is called when exiting the commentColumnConstraint production.
	ExitCommentColumnConstraint(c *CommentColumnConstraintContext)

	// ExitFormatColumnConstraint is called when exiting the formatColumnConstraint production.
	ExitFormatColumnConstraint(c *FormatColumnConstraintContext)

	// ExitStorageColumnConstraint is called when exiting the storageColumnConstraint production.
	ExitStorageColumnConstraint(c *StorageColumnConstraintContext)

	// ExitReferenceColumnConstraint is called when exiting the referenceColumnConstraint production.
	ExitReferenceColumnConstraint(c *ReferenceColumnConstraintContext)

	// ExitCollateColumnConstraint is called when exiting the collateColumnConstraint production.
	ExitCollateColumnConstraint(c *CollateColumnConstraintContext)

	// ExitGeneratedColumnConstraint is called when exiting the generatedColumnConstraint production.
	ExitGeneratedColumnConstraint(c *GeneratedColumnConstraintContext)

	// ExitSerialDefaultColumnConstraint is called when exiting the serialDefaultColumnConstraint production.
	ExitSerialDefaultColumnConstraint(c *SerialDefaultColumnConstraintContext)

	// ExitCheckColumnConstraint is called when exiting the checkColumnConstraint production.
	ExitCheckColumnConstraint(c *CheckColumnConstraintContext)

	// ExitPrimaryKeyTableConstraint is called when exiting the primaryKeyTableConstraint production.
	ExitPrimaryKeyTableConstraint(c *PrimaryKeyTableConstraintContext)

	// ExitUniqueKeyTableConstraint is called when exiting the uniqueKeyTableConstraint production.
	ExitUniqueKeyTableConstraint(c *UniqueKeyTableConstraintContext)

	// ExitForeignKeyTableConstraint is called when exiting the foreignKeyTableConstraint production.
	ExitForeignKeyTableConstraint(c *ForeignKeyTableConstraintContext)

	// ExitCheckTableConstraint is called when exiting the checkTableConstraint production.
	ExitCheckTableConstraint(c *CheckTableConstraintContext)

	// ExitReferenceDefinition is called when exiting the referenceDefinition production.
	ExitReferenceDefinition(c *ReferenceDefinitionContext)

	// ExitReferenceAction is called when exiting the referenceAction production.
	ExitReferenceAction(c *ReferenceActionContext)

	// ExitReferenceControlType is called when exiting the referenceControlType production.
	ExitReferenceControlType(c *ReferenceControlTypeContext)

	// ExitSimpleIndexDeclaration is called when exiting the simpleIndexDeclaration production.
	ExitSimpleIndexDeclaration(c *SimpleIndexDeclarationContext)

	// ExitSpecialIndexDeclaration is called when exiting the specialIndexDeclaration production.
	ExitSpecialIndexDeclaration(c *SpecialIndexDeclarationContext)

	// ExitTableOptionEngine is called when exiting the tableOptionEngine production.
	ExitTableOptionEngine(c *TableOptionEngineContext)

	// ExitTableOptionEngineAttribute is called when exiting the tableOptionEngineAttribute production.
	ExitTableOptionEngineAttribute(c *TableOptionEngineAttributeContext)

	// ExitTableOptionAutoextendSize is called when exiting the tableOptionAutoextendSize production.
	ExitTableOptionAutoextendSize(c *TableOptionAutoextendSizeContext)

	// ExitTableOptionAutoIncrement is called when exiting the tableOptionAutoIncrement production.
	ExitTableOptionAutoIncrement(c *TableOptionAutoIncrementContext)

	// ExitTableOptionAverage is called when exiting the tableOptionAverage production.
	ExitTableOptionAverage(c *TableOptionAverageContext)

	// ExitTableOptionCharset is called when exiting the tableOptionCharset production.
	ExitTableOptionCharset(c *TableOptionCharsetContext)

	// ExitTableOptionChecksum is called when exiting the tableOptionChecksum production.
	ExitTableOptionChecksum(c *TableOptionChecksumContext)

	// ExitTableOptionCollate is called when exiting the tableOptionCollate production.
	ExitTableOptionCollate(c *TableOptionCollateContext)

	// ExitTableOptionComment is called when exiting the tableOptionComment production.
	ExitTableOptionComment(c *TableOptionCommentContext)

	// ExitTableOptionCompression is called when exiting the tableOptionCompression production.
	ExitTableOptionCompression(c *TableOptionCompressionContext)

	// ExitTableOptionConnection is called when exiting the tableOptionConnection production.
	ExitTableOptionConnection(c *TableOptionConnectionContext)

	// ExitTableOptionDataDirectory is called when exiting the tableOptionDataDirectory production.
	ExitTableOptionDataDirectory(c *TableOptionDataDirectoryContext)

	// ExitTableOptionDelay is called when exiting the tableOptionDelay production.
	ExitTableOptionDelay(c *TableOptionDelayContext)

	// ExitTableOptionEncryption is called when exiting the tableOptionEncryption production.
	ExitTableOptionEncryption(c *TableOptionEncryptionContext)

	// ExitTableOptionPageCompressed is called when exiting the tableOptionPageCompressed production.
	ExitTableOptionPageCompressed(c *TableOptionPageCompressedContext)

	// ExitTableOptionPageCompressionLevel is called when exiting the tableOptionPageCompressionLevel production.
	ExitTableOptionPageCompressionLevel(c *TableOptionPageCompressionLevelContext)

	// ExitTableOptionEncryptionKeyId is called when exiting the tableOptionEncryptionKeyId production.
	ExitTableOptionEncryptionKeyId(c *TableOptionEncryptionKeyIdContext)

	// ExitTableOptionIndexDirectory is called when exiting the tableOptionIndexDirectory production.
	ExitTableOptionIndexDirectory(c *TableOptionIndexDirectoryContext)

	// ExitTableOptionInsertMethod is called when exiting the tableOptionInsertMethod production.
	ExitTableOptionInsertMethod(c *TableOptionInsertMethodContext)

	// ExitTableOptionKeyBlockSize is called when exiting the tableOptionKeyBlockSize production.
	ExitTableOptionKeyBlockSize(c *TableOptionKeyBlockSizeContext)

	// ExitTableOptionMaxRows is called when exiting the tableOptionMaxRows production.
	ExitTableOptionMaxRows(c *TableOptionMaxRowsContext)

	// ExitTableOptionMinRows is called when exiting the tableOptionMinRows production.
	ExitTableOptionMinRows(c *TableOptionMinRowsContext)

	// ExitTableOptionPackKeys is called when exiting the tableOptionPackKeys production.
	ExitTableOptionPackKeys(c *TableOptionPackKeysContext)

	// ExitTableOptionPassword is called when exiting the tableOptionPassword production.
	ExitTableOptionPassword(c *TableOptionPasswordContext)

	// ExitTableOptionRowFormat is called when exiting the tableOptionRowFormat production.
	ExitTableOptionRowFormat(c *TableOptionRowFormatContext)

	// ExitTableOptionStartTransaction is called when exiting the tableOptionStartTransaction production.
	ExitTableOptionStartTransaction(c *TableOptionStartTransactionContext)

	// ExitTableOptionSecondaryEngineAttribute is called when exiting the tableOptionSecondaryEngineAttribute production.
	ExitTableOptionSecondaryEngineAttribute(c *TableOptionSecondaryEngineAttributeContext)

	// ExitTableOptionRecalculation is called when exiting the tableOptionRecalculation production.
	ExitTableOptionRecalculation(c *TableOptionRecalculationContext)

	// ExitTableOptionPersistent is called when exiting the tableOptionPersistent production.
	ExitTableOptionPersistent(c *TableOptionPersistentContext)

	// ExitTableOptionSamplePage is called when exiting the tableOptionSamplePage production.
	ExitTableOptionSamplePage(c *TableOptionSamplePageContext)

	// ExitTableOptionTablespace is called when exiting the tableOptionTablespace production.
	ExitTableOptionTablespace(c *TableOptionTablespaceContext)

	// ExitTableOptionTableType is called when exiting the tableOptionTableType production.
	ExitTableOptionTableType(c *TableOptionTableTypeContext)

	// ExitTableOptionTransactional is called when exiting the tableOptionTransactional production.
	ExitTableOptionTransactional(c *TableOptionTransactionalContext)

	// ExitTableOptionUnion is called when exiting the tableOptionUnion production.
	ExitTableOptionUnion(c *TableOptionUnionContext)

	// ExitTableType is called when exiting the tableType production.
	ExitTableType(c *TableTypeContext)

	// ExitTablespaceStorage is called when exiting the tablespaceStorage production.
	ExitTablespaceStorage(c *TablespaceStorageContext)

	// ExitPartitionDefinitions is called when exiting the partitionDefinitions production.
	ExitPartitionDefinitions(c *PartitionDefinitionsContext)

	// ExitPartitionFunctionHash is called when exiting the partitionFunctionHash production.
	ExitPartitionFunctionHash(c *PartitionFunctionHashContext)

	// ExitPartitionFunctionKey is called when exiting the partitionFunctionKey production.
	ExitPartitionFunctionKey(c *PartitionFunctionKeyContext)

	// ExitPartitionFunctionRange is called when exiting the partitionFunctionRange production.
	ExitPartitionFunctionRange(c *PartitionFunctionRangeContext)

	// ExitPartitionFunctionList is called when exiting the partitionFunctionList production.
	ExitPartitionFunctionList(c *PartitionFunctionListContext)

	// ExitSubPartitionFunctionHash is called when exiting the subPartitionFunctionHash production.
	ExitSubPartitionFunctionHash(c *SubPartitionFunctionHashContext)

	// ExitSubPartitionFunctionKey is called when exiting the subPartitionFunctionKey production.
	ExitSubPartitionFunctionKey(c *SubPartitionFunctionKeyContext)

	// ExitPartitionComparison is called when exiting the partitionComparison production.
	ExitPartitionComparison(c *PartitionComparisonContext)

	// ExitPartitionListAtom is called when exiting the partitionListAtom production.
	ExitPartitionListAtom(c *PartitionListAtomContext)

	// ExitPartitionListVector is called when exiting the partitionListVector production.
	ExitPartitionListVector(c *PartitionListVectorContext)

	// ExitPartitionSimple is called when exiting the partitionSimple production.
	ExitPartitionSimple(c *PartitionSimpleContext)

	// ExitPartitionDefinerAtom is called when exiting the partitionDefinerAtom production.
	ExitPartitionDefinerAtom(c *PartitionDefinerAtomContext)

	// ExitPartitionDefinerVector is called when exiting the partitionDefinerVector production.
	ExitPartitionDefinerVector(c *PartitionDefinerVectorContext)

	// ExitSubpartitionDefinition is called when exiting the subpartitionDefinition production.
	ExitSubpartitionDefinition(c *SubpartitionDefinitionContext)

	// ExitPartitionOptionEngine is called when exiting the partitionOptionEngine production.
	ExitPartitionOptionEngine(c *PartitionOptionEngineContext)

	// ExitPartitionOptionComment is called when exiting the partitionOptionComment production.
	ExitPartitionOptionComment(c *PartitionOptionCommentContext)

	// ExitPartitionOptionDataDirectory is called when exiting the partitionOptionDataDirectory production.
	ExitPartitionOptionDataDirectory(c *PartitionOptionDataDirectoryContext)

	// ExitPartitionOptionIndexDirectory is called when exiting the partitionOptionIndexDirectory production.
	ExitPartitionOptionIndexDirectory(c *PartitionOptionIndexDirectoryContext)

	// ExitPartitionOptionMaxRows is called when exiting the partitionOptionMaxRows production.
	ExitPartitionOptionMaxRows(c *PartitionOptionMaxRowsContext)

	// ExitPartitionOptionMinRows is called when exiting the partitionOptionMinRows production.
	ExitPartitionOptionMinRows(c *PartitionOptionMinRowsContext)

	// ExitPartitionOptionTablespace is called when exiting the partitionOptionTablespace production.
	ExitPartitionOptionTablespace(c *PartitionOptionTablespaceContext)

	// ExitPartitionOptionNodeGroup is called when exiting the partitionOptionNodeGroup production.
	ExitPartitionOptionNodeGroup(c *PartitionOptionNodeGroupContext)

	// ExitAlterSimpleDatabase is called when exiting the alterSimpleDatabase production.
	ExitAlterSimpleDatabase(c *AlterSimpleDatabaseContext)

	// ExitAlterUpgradeName is called when exiting the alterUpgradeName production.
	ExitAlterUpgradeName(c *AlterUpgradeNameContext)

	// ExitAlterEvent is called when exiting the alterEvent production.
	ExitAlterEvent(c *AlterEventContext)

	// ExitAlterFunction is called when exiting the alterFunction production.
	ExitAlterFunction(c *AlterFunctionContext)

	// ExitAlterInstance is called when exiting the alterInstance production.
	ExitAlterInstance(c *AlterInstanceContext)

	// ExitAlterLogfileGroup is called when exiting the alterLogfileGroup production.
	ExitAlterLogfileGroup(c *AlterLogfileGroupContext)

	// ExitAlterProcedure is called when exiting the alterProcedure production.
	ExitAlterProcedure(c *AlterProcedureContext)

	// ExitAlterServer is called when exiting the alterServer production.
	ExitAlterServer(c *AlterServerContext)

	// ExitAlterTable is called when exiting the alterTable production.
	ExitAlterTable(c *AlterTableContext)

	// ExitAlterTablespace is called when exiting the alterTablespace production.
	ExitAlterTablespace(c *AlterTablespaceContext)

	// ExitAlterView is called when exiting the alterView production.
	ExitAlterView(c *AlterViewContext)

	// ExitAlterByTableOption is called when exiting the alterByTableOption production.
	ExitAlterByTableOption(c *AlterByTableOptionContext)

	// ExitAlterByAddColumn is called when exiting the alterByAddColumn production.
	ExitAlterByAddColumn(c *AlterByAddColumnContext)

	// ExitAlterByAddColumns is called when exiting the alterByAddColumns production.
	ExitAlterByAddColumns(c *AlterByAddColumnsContext)

	// ExitAlterByAddIndex is called when exiting the alterByAddIndex production.
	ExitAlterByAddIndex(c *AlterByAddIndexContext)

	// ExitAlterByAddPrimaryKey is called when exiting the alterByAddPrimaryKey production.
	ExitAlterByAddPrimaryKey(c *AlterByAddPrimaryKeyContext)

	// ExitAlterByAddUniqueKey is called when exiting the alterByAddUniqueKey production.
	ExitAlterByAddUniqueKey(c *AlterByAddUniqueKeyContext)

	// ExitAlterByAddSpecialIndex is called when exiting the alterByAddSpecialIndex production.
	ExitAlterByAddSpecialIndex(c *AlterByAddSpecialIndexContext)

	// ExitAlterByAddForeignKey is called when exiting the alterByAddForeignKey production.
	ExitAlterByAddForeignKey(c *AlterByAddForeignKeyContext)

	// ExitAlterByAddCheckTableConstraint is called when exiting the alterByAddCheckTableConstraint production.
	ExitAlterByAddCheckTableConstraint(c *AlterByAddCheckTableConstraintContext)

	// ExitAlterByAlterCheckTableConstraint is called when exiting the alterByAlterCheckTableConstraint production.
	ExitAlterByAlterCheckTableConstraint(c *AlterByAlterCheckTableConstraintContext)

	// ExitAlterBySetAlgorithm is called when exiting the alterBySetAlgorithm production.
	ExitAlterBySetAlgorithm(c *AlterBySetAlgorithmContext)

	// ExitAlterByChangeDefault is called when exiting the alterByChangeDefault production.
	ExitAlterByChangeDefault(c *AlterByChangeDefaultContext)

	// ExitAlterByChangeColumn is called when exiting the alterByChangeColumn production.
	ExitAlterByChangeColumn(c *AlterByChangeColumnContext)

	// ExitAlterByRenameColumn is called when exiting the alterByRenameColumn production.
	ExitAlterByRenameColumn(c *AlterByRenameColumnContext)

	// ExitAlterByLock is called when exiting the alterByLock production.
	ExitAlterByLock(c *AlterByLockContext)

	// ExitAlterByModifyColumn is called when exiting the alterByModifyColumn production.
	ExitAlterByModifyColumn(c *AlterByModifyColumnContext)

	// ExitAlterByDropColumn is called when exiting the alterByDropColumn production.
	ExitAlterByDropColumn(c *AlterByDropColumnContext)

	// ExitAlterByDropConstraintCheck is called when exiting the alterByDropConstraintCheck production.
	ExitAlterByDropConstraintCheck(c *AlterByDropConstraintCheckContext)

	// ExitAlterByDropPrimaryKey is called when exiting the alterByDropPrimaryKey production.
	ExitAlterByDropPrimaryKey(c *AlterByDropPrimaryKeyContext)

	// ExitAlterByDropIndex is called when exiting the alterByDropIndex production.
	ExitAlterByDropIndex(c *AlterByDropIndexContext)

	// ExitAlterByRenameIndex is called when exiting the alterByRenameIndex production.
	ExitAlterByRenameIndex(c *AlterByRenameIndexContext)

	// ExitAlterByAlterColumnDefault is called when exiting the alterByAlterColumnDefault production.
	ExitAlterByAlterColumnDefault(c *AlterByAlterColumnDefaultContext)

	// ExitAlterByAlterIndexVisibility is called when exiting the alterByAlterIndexVisibility production.
	ExitAlterByAlterIndexVisibility(c *AlterByAlterIndexVisibilityContext)

	// ExitAlterByDropForeignKey is called when exiting the alterByDropForeignKey production.
	ExitAlterByDropForeignKey(c *AlterByDropForeignKeyContext)

	// ExitAlterByDisableKeys is called when exiting the alterByDisableKeys production.
	ExitAlterByDisableKeys(c *AlterByDisableKeysContext)

	// ExitAlterByEnableKeys is called when exiting the alterByEnableKeys production.
	ExitAlterByEnableKeys(c *AlterByEnableKeysContext)

	// ExitAlterByRename is called when exiting the alterByRename production.
	ExitAlterByRename(c *AlterByRenameContext)

	// ExitAlterByOrder is called when exiting the alterByOrder production.
	ExitAlterByOrder(c *AlterByOrderContext)

	// ExitAlterByConvertCharset is called when exiting the alterByConvertCharset production.
	ExitAlterByConvertCharset(c *AlterByConvertCharsetContext)

	// ExitAlterByDefaultCharset is called when exiting the alterByDefaultCharset production.
	ExitAlterByDefaultCharset(c *AlterByDefaultCharsetContext)

	// ExitAlterByDiscardTablespace is called when exiting the alterByDiscardTablespace production.
	ExitAlterByDiscardTablespace(c *AlterByDiscardTablespaceContext)

	// ExitAlterByImportTablespace is called when exiting the alterByImportTablespace production.
	ExitAlterByImportTablespace(c *AlterByImportTablespaceContext)

	// ExitAlterByForce is called when exiting the alterByForce production.
	ExitAlterByForce(c *AlterByForceContext)

	// ExitAlterByValidate is called when exiting the alterByValidate production.
	ExitAlterByValidate(c *AlterByValidateContext)

	// ExitAlterByAddDefinitions is called when exiting the alterByAddDefinitions production.
	ExitAlterByAddDefinitions(c *AlterByAddDefinitionsContext)

	// ExitAlterPartition is called when exiting the alterPartition production.
	ExitAlterPartition(c *AlterPartitionContext)

	// ExitAlterByAddPartition is called when exiting the alterByAddPartition production.
	ExitAlterByAddPartition(c *AlterByAddPartitionContext)

	// ExitAlterByDropPartition is called when exiting the alterByDropPartition production.
	ExitAlterByDropPartition(c *AlterByDropPartitionContext)

	// ExitAlterByDiscardPartition is called when exiting the alterByDiscardPartition production.
	ExitAlterByDiscardPartition(c *AlterByDiscardPartitionContext)

	// ExitAlterByImportPartition is called when exiting the alterByImportPartition production.
	ExitAlterByImportPartition(c *AlterByImportPartitionContext)

	// ExitAlterByTruncatePartition is called when exiting the alterByTruncatePartition production.
	ExitAlterByTruncatePartition(c *AlterByTruncatePartitionContext)

	// ExitAlterByCoalescePartition is called when exiting the alterByCoalescePartition production.
	ExitAlterByCoalescePartition(c *AlterByCoalescePartitionContext)

	// ExitAlterByReorganizePartition is called when exiting the alterByReorganizePartition production.
	ExitAlterByReorganizePartition(c *AlterByReorganizePartitionContext)

	// ExitAlterByExchangePartition is called when exiting the alterByExchangePartition production.
	ExitAlterByExchangePartition(c *AlterByExchangePartitionContext)

	// ExitAlterByAnalyzePartition is called when exiting the alterByAnalyzePartition production.
	ExitAlterByAnalyzePartition(c *AlterByAnalyzePartitionContext)

	// ExitAlterByCheckPartition is called when exiting the alterByCheckPartition production.
	ExitAlterByCheckPartition(c *AlterByCheckPartitionContext)

	// ExitAlterByOptimizePartition is called when exiting the alterByOptimizePartition production.
	ExitAlterByOptimizePartition(c *AlterByOptimizePartitionContext)

	// ExitAlterByRebuildPartition is called when exiting the alterByRebuildPartition production.
	ExitAlterByRebuildPartition(c *AlterByRebuildPartitionContext)

	// ExitAlterByRepairPartition is called when exiting the alterByRepairPartition production.
	ExitAlterByRepairPartition(c *AlterByRepairPartitionContext)

	// ExitAlterByRemovePartitioning is called when exiting the alterByRemovePartitioning production.
	ExitAlterByRemovePartitioning(c *AlterByRemovePartitioningContext)

	// ExitAlterByUpgradePartitioning is called when exiting the alterByUpgradePartitioning production.
	ExitAlterByUpgradePartitioning(c *AlterByUpgradePartitioningContext)

	// ExitDropDatabase is called when exiting the dropDatabase production.
	ExitDropDatabase(c *DropDatabaseContext)

	// ExitDropEvent is called when exiting the dropEvent production.
	ExitDropEvent(c *DropEventContext)

	// ExitDropIndex is called when exiting the dropIndex production.
	ExitDropIndex(c *DropIndexContext)

	// ExitDropLogfileGroup is called when exiting the dropLogfileGroup production.
	ExitDropLogfileGroup(c *DropLogfileGroupContext)

	// ExitDropProcedure is called when exiting the dropProcedure production.
	ExitDropProcedure(c *DropProcedureContext)

	// ExitDropFunction is called when exiting the dropFunction production.
	ExitDropFunction(c *DropFunctionContext)

	// ExitDropServer is called when exiting the dropServer production.
	ExitDropServer(c *DropServerContext)

	// ExitDropTable is called when exiting the dropTable production.
	ExitDropTable(c *DropTableContext)

	// ExitDropTablespace is called when exiting the dropTablespace production.
	ExitDropTablespace(c *DropTablespaceContext)

	// ExitDropTrigger is called when exiting the dropTrigger production.
	ExitDropTrigger(c *DropTriggerContext)

	// ExitDropView is called when exiting the dropView production.
	ExitDropView(c *DropViewContext)

	// ExitDropRole is called when exiting the dropRole production.
	ExitDropRole(c *DropRoleContext)

	// ExitSetRole is called when exiting the setRole production.
	ExitSetRole(c *SetRoleContext)

	// ExitRenameTable is called when exiting the renameTable production.
	ExitRenameTable(c *RenameTableContext)

	// ExitRenameTableClause is called when exiting the renameTableClause production.
	ExitRenameTableClause(c *RenameTableClauseContext)

	// ExitTruncateTable is called when exiting the truncateTable production.
	ExitTruncateTable(c *TruncateTableContext)

	// ExitCallStatement is called when exiting the callStatement production.
	ExitCallStatement(c *CallStatementContext)

	// ExitDeleteStatement is called when exiting the deleteStatement production.
	ExitDeleteStatement(c *DeleteStatementContext)

	// ExitDoStatement is called when exiting the doStatement production.
	ExitDoStatement(c *DoStatementContext)

	// ExitHandlerStatement is called when exiting the handlerStatement production.
	ExitHandlerStatement(c *HandlerStatementContext)

	// ExitInsertStatement is called when exiting the insertStatement production.
	ExitInsertStatement(c *InsertStatementContext)

	// ExitLoadDataStatement is called when exiting the loadDataStatement production.
	ExitLoadDataStatement(c *LoadDataStatementContext)

	// ExitLoadXmlStatement is called when exiting the loadXmlStatement production.
	ExitLoadXmlStatement(c *LoadXmlStatementContext)

	// ExitReplaceStatement is called when exiting the replaceStatement production.
	ExitReplaceStatement(c *ReplaceStatementContext)

	// ExitSimpleSelect is called when exiting the simpleSelect production.
	ExitSimpleSelect(c *SimpleSelectContext)

	// ExitParenthesisSelect is called when exiting the parenthesisSelect production.
	ExitParenthesisSelect(c *ParenthesisSelectContext)

	// ExitUnionSelect is called when exiting the unionSelect production.
	ExitUnionSelect(c *UnionSelectContext)

	// ExitUnionParenthesisSelect is called when exiting the unionParenthesisSelect production.
	ExitUnionParenthesisSelect(c *UnionParenthesisSelectContext)

	// ExitWithLateralStatement is called when exiting the withLateralStatement production.
	ExitWithLateralStatement(c *WithLateralStatementContext)

	// ExitUpdateStatement is called when exiting the updateStatement production.
	ExitUpdateStatement(c *UpdateStatementContext)

	// ExitValuesStatement is called when exiting the valuesStatement production.
	ExitValuesStatement(c *ValuesStatementContext)

	// ExitInsertStatementValue is called when exiting the insertStatementValue production.
	ExitInsertStatementValue(c *InsertStatementValueContext)

	// ExitUpdatedElement is called when exiting the updatedElement production.
	ExitUpdatedElement(c *UpdatedElementContext)

	// ExitAssignmentField is called when exiting the assignmentField production.
	ExitAssignmentField(c *AssignmentFieldContext)

	// ExitLockClause is called when exiting the lockClause production.
	ExitLockClause(c *LockClauseContext)

	// ExitSingleDeleteStatement is called when exiting the singleDeleteStatement production.
	ExitSingleDeleteStatement(c *SingleDeleteStatementContext)

	// ExitMultipleDeleteStatement is called when exiting the multipleDeleteStatement production.
	ExitMultipleDeleteStatement(c *MultipleDeleteStatementContext)

	// ExitHandlerOpenStatement is called when exiting the handlerOpenStatement production.
	ExitHandlerOpenStatement(c *HandlerOpenStatementContext)

	// ExitHandlerReadIndexStatement is called when exiting the handlerReadIndexStatement production.
	ExitHandlerReadIndexStatement(c *HandlerReadIndexStatementContext)

	// ExitHandlerReadStatement is called when exiting the handlerReadStatement production.
	ExitHandlerReadStatement(c *HandlerReadStatementContext)

	// ExitHandlerCloseStatement is called when exiting the handlerCloseStatement production.
	ExitHandlerCloseStatement(c *HandlerCloseStatementContext)

	// ExitSingleUpdateStatement is called when exiting the singleUpdateStatement production.
	ExitSingleUpdateStatement(c *SingleUpdateStatementContext)

	// ExitMultipleUpdateStatement is called when exiting the multipleUpdateStatement production.
	ExitMultipleUpdateStatement(c *MultipleUpdateStatementContext)

	// ExitOrderByClause is called when exiting the orderByClause production.
	ExitOrderByClause(c *OrderByClauseContext)

	// ExitOrderByExpression is called when exiting the orderByExpression production.
	ExitOrderByExpression(c *OrderByExpressionContext)

	// ExitTableSources is called when exiting the tableSources production.
	ExitTableSources(c *TableSourcesContext)

	// ExitTableSourceBase is called when exiting the tableSourceBase production.
	ExitTableSourceBase(c *TableSourceBaseContext)

	// ExitTableSourceNested is called when exiting the tableSourceNested production.
	ExitTableSourceNested(c *TableSourceNestedContext)

	// ExitTableJson is called when exiting the tableJson production.
	ExitTableJson(c *TableJsonContext)

	// ExitAtomTableItem is called when exiting the atomTableItem production.
	ExitAtomTableItem(c *AtomTableItemContext)

	// ExitSubqueryTableItem is called when exiting the subqueryTableItem production.
	ExitSubqueryTableItem(c *SubqueryTableItemContext)

	// ExitTableSourcesItem is called when exiting the tableSourcesItem production.
	ExitTableSourcesItem(c *TableSourcesItemContext)

	// ExitIndexHint is called when exiting the indexHint production.
	ExitIndexHint(c *IndexHintContext)

	// ExitIndexHintType is called when exiting the indexHintType production.
	ExitIndexHintType(c *IndexHintTypeContext)

	// ExitInnerJoin is called when exiting the innerJoin production.
	ExitInnerJoin(c *InnerJoinContext)

	// ExitStraightJoin is called when exiting the straightJoin production.
	ExitStraightJoin(c *StraightJoinContext)

	// ExitOuterJoin is called when exiting the outerJoin production.
	ExitOuterJoin(c *OuterJoinContext)

	// ExitNaturalJoin is called when exiting the naturalJoin production.
	ExitNaturalJoin(c *NaturalJoinContext)

	// ExitJoinSpec is called when exiting the joinSpec production.
	ExitJoinSpec(c *JoinSpecContext)

	// ExitQueryExpression is called when exiting the queryExpression production.
	ExitQueryExpression(c *QueryExpressionContext)

	// ExitQueryExpressionNointo is called when exiting the queryExpressionNointo production.
	ExitQueryExpressionNointo(c *QueryExpressionNointoContext)

	// ExitQuerySpecification is called when exiting the querySpecification production.
	ExitQuerySpecification(c *QuerySpecificationContext)

	// ExitQuerySpecificationNointo is called when exiting the querySpecificationNointo production.
	ExitQuerySpecificationNointo(c *QuerySpecificationNointoContext)

	// ExitUnionParenthesis is called when exiting the unionParenthesis production.
	ExitUnionParenthesis(c *UnionParenthesisContext)

	// ExitUnionStatement is called when exiting the unionStatement production.
	ExitUnionStatement(c *UnionStatementContext)

	// ExitLateralStatement is called when exiting the lateralStatement production.
	ExitLateralStatement(c *LateralStatementContext)

	// ExitJsonTable is called when exiting the jsonTable production.
	ExitJsonTable(c *JsonTableContext)

	// ExitJsonColumnList is called when exiting the jsonColumnList production.
	ExitJsonColumnList(c *JsonColumnListContext)

	// ExitJsonColumn is called when exiting the jsonColumn production.
	ExitJsonColumn(c *JsonColumnContext)

	// ExitJsonOnEmpty is called when exiting the jsonOnEmpty production.
	ExitJsonOnEmpty(c *JsonOnEmptyContext)

	// ExitJsonOnError is called when exiting the jsonOnError production.
	ExitJsonOnError(c *JsonOnErrorContext)

	// ExitSelectSpec is called when exiting the selectSpec production.
	ExitSelectSpec(c *SelectSpecContext)

	// ExitSelectElements is called when exiting the selectElements production.
	ExitSelectElements(c *SelectElementsContext)

	// ExitSelectStarElement is called when exiting the selectStarElement production.
	ExitSelectStarElement(c *SelectStarElementContext)

	// ExitSelectColumnElement is called when exiting the selectColumnElement production.
	ExitSelectColumnElement(c *SelectColumnElementContext)

	// ExitSelectFunctionElement is called when exiting the selectFunctionElement production.
	ExitSelectFunctionElement(c *SelectFunctionElementContext)

	// ExitSelectExpressionElement is called when exiting the selectExpressionElement production.
	ExitSelectExpressionElement(c *SelectExpressionElementContext)

	// ExitSelectIntoVariables is called when exiting the selectIntoVariables production.
	ExitSelectIntoVariables(c *SelectIntoVariablesContext)

	// ExitSelectIntoDumpFile is called when exiting the selectIntoDumpFile production.
	ExitSelectIntoDumpFile(c *SelectIntoDumpFileContext)

	// ExitSelectIntoTextFile is called when exiting the selectIntoTextFile production.
	ExitSelectIntoTextFile(c *SelectIntoTextFileContext)

	// ExitSelectFieldsInto is called when exiting the selectFieldsInto production.
	ExitSelectFieldsInto(c *SelectFieldsIntoContext)

	// ExitSelectLinesInto is called when exiting the selectLinesInto production.
	ExitSelectLinesInto(c *SelectLinesIntoContext)

	// ExitFromClause is called when exiting the fromClause production.
	ExitFromClause(c *FromClauseContext)

	// ExitGroupByClause is called when exiting the groupByClause production.
	ExitGroupByClause(c *GroupByClauseContext)

	// ExitHavingClause is called when exiting the havingClause production.
	ExitHavingClause(c *HavingClauseContext)

	// ExitWindowClause is called when exiting the windowClause production.
	ExitWindowClause(c *WindowClauseContext)

	// ExitGroupByItem is called when exiting the groupByItem production.
	ExitGroupByItem(c *GroupByItemContext)

	// ExitLimitClause is called when exiting the limitClause production.
	ExitLimitClause(c *LimitClauseContext)

	// ExitLimitClauseAtom is called when exiting the limitClauseAtom production.
	ExitLimitClauseAtom(c *LimitClauseAtomContext)

	// ExitStartTransaction is called when exiting the startTransaction production.
	ExitStartTransaction(c *StartTransactionContext)

	// ExitBeginWork is called when exiting the beginWork production.
	ExitBeginWork(c *BeginWorkContext)

	// ExitCommitWork is called when exiting the commitWork production.
	ExitCommitWork(c *CommitWorkContext)

	// ExitRollbackWork is called when exiting the rollbackWork production.
	ExitRollbackWork(c *RollbackWorkContext)

	// ExitSavepointStatement is called when exiting the savepointStatement production.
	ExitSavepointStatement(c *SavepointStatementContext)

	// ExitRollbackStatement is called when exiting the rollbackStatement production.
	ExitRollbackStatement(c *RollbackStatementContext)

	// ExitReleaseStatement is called when exiting the releaseStatement production.
	ExitReleaseStatement(c *ReleaseStatementContext)

	// ExitLockTables is called when exiting the lockTables production.
	ExitLockTables(c *LockTablesContext)

	// ExitUnlockTables is called when exiting the unlockTables production.
	ExitUnlockTables(c *UnlockTablesContext)

	// ExitSetAutocommitStatement is called when exiting the setAutocommitStatement production.
	ExitSetAutocommitStatement(c *SetAutocommitStatementContext)

	// ExitSetTransactionStatement is called when exiting the setTransactionStatement production.
	ExitSetTransactionStatement(c *SetTransactionStatementContext)

	// ExitTransactionMode is called when exiting the transactionMode production.
	ExitTransactionMode(c *TransactionModeContext)

	// ExitLockTableElement is called when exiting the lockTableElement production.
	ExitLockTableElement(c *LockTableElementContext)

	// ExitLockAction is called when exiting the lockAction production.
	ExitLockAction(c *LockActionContext)

	// ExitTransactionOption is called when exiting the transactionOption production.
	ExitTransactionOption(c *TransactionOptionContext)

	// ExitTransactionLevel is called when exiting the transactionLevel production.
	ExitTransactionLevel(c *TransactionLevelContext)

	// ExitChangeMaster is called when exiting the changeMaster production.
	ExitChangeMaster(c *ChangeMasterContext)

	// ExitChangeReplicationFilter is called when exiting the changeReplicationFilter production.
	ExitChangeReplicationFilter(c *ChangeReplicationFilterContext)

	// ExitPurgeBinaryLogs is called when exiting the purgeBinaryLogs production.
	ExitPurgeBinaryLogs(c *PurgeBinaryLogsContext)

	// ExitResetMaster is called when exiting the resetMaster production.
	ExitResetMaster(c *ResetMasterContext)

	// ExitResetSlave is called when exiting the resetSlave production.
	ExitResetSlave(c *ResetSlaveContext)

	// ExitStartSlave is called when exiting the startSlave production.
	ExitStartSlave(c *StartSlaveContext)

	// ExitStopSlave is called when exiting the stopSlave production.
	ExitStopSlave(c *StopSlaveContext)

	// ExitStartGroupReplication is called when exiting the startGroupReplication production.
	ExitStartGroupReplication(c *StartGroupReplicationContext)

	// ExitStopGroupReplication is called when exiting the stopGroupReplication production.
	ExitStopGroupReplication(c *StopGroupReplicationContext)

	// ExitMasterStringOption is called when exiting the masterStringOption production.
	ExitMasterStringOption(c *MasterStringOptionContext)

	// ExitMasterDecimalOption is called when exiting the masterDecimalOption production.
	ExitMasterDecimalOption(c *MasterDecimalOptionContext)

	// ExitMasterBoolOption is called when exiting the masterBoolOption production.
	ExitMasterBoolOption(c *MasterBoolOptionContext)

	// ExitMasterRealOption is called when exiting the masterRealOption production.
	ExitMasterRealOption(c *MasterRealOptionContext)

	// ExitMasterUidListOption is called when exiting the masterUidListOption production.
	ExitMasterUidListOption(c *MasterUidListOptionContext)

	// ExitStringMasterOption is called when exiting the stringMasterOption production.
	ExitStringMasterOption(c *StringMasterOptionContext)

	// ExitDecimalMasterOption is called when exiting the decimalMasterOption production.
	ExitDecimalMasterOption(c *DecimalMasterOptionContext)

	// ExitBoolMasterOption is called when exiting the boolMasterOption production.
	ExitBoolMasterOption(c *BoolMasterOptionContext)

	// ExitChannelOption is called when exiting the channelOption production.
	ExitChannelOption(c *ChannelOptionContext)

	// ExitDoDbReplication is called when exiting the doDbReplication production.
	ExitDoDbReplication(c *DoDbReplicationContext)

	// ExitIgnoreDbReplication is called when exiting the ignoreDbReplication production.
	ExitIgnoreDbReplication(c *IgnoreDbReplicationContext)

	// ExitDoTableReplication is called when exiting the doTableReplication production.
	ExitDoTableReplication(c *DoTableReplicationContext)

	// ExitIgnoreTableReplication is called when exiting the ignoreTableReplication production.
	ExitIgnoreTableReplication(c *IgnoreTableReplicationContext)

	// ExitWildDoTableReplication is called when exiting the wildDoTableReplication production.
	ExitWildDoTableReplication(c *WildDoTableReplicationContext)

	// ExitWildIgnoreTableReplication is called when exiting the wildIgnoreTableReplication production.
	ExitWildIgnoreTableReplication(c *WildIgnoreTableReplicationContext)

	// ExitRewriteDbReplication is called when exiting the rewriteDbReplication production.
	ExitRewriteDbReplication(c *RewriteDbReplicationContext)

	// ExitTablePair is called when exiting the tablePair production.
	ExitTablePair(c *TablePairContext)

	// ExitThreadType is called when exiting the threadType production.
	ExitThreadType(c *ThreadTypeContext)

	// ExitGtidsUntilOption is called when exiting the gtidsUntilOption production.
	ExitGtidsUntilOption(c *GtidsUntilOptionContext)

	// ExitMasterLogUntilOption is called when exiting the masterLogUntilOption production.
	ExitMasterLogUntilOption(c *MasterLogUntilOptionContext)

	// ExitRelayLogUntilOption is called when exiting the relayLogUntilOption production.
	ExitRelayLogUntilOption(c *RelayLogUntilOptionContext)

	// ExitSqlGapsUntilOption is called when exiting the sqlGapsUntilOption production.
	ExitSqlGapsUntilOption(c *SqlGapsUntilOptionContext)

	// ExitUserConnectionOption is called when exiting the userConnectionOption production.
	ExitUserConnectionOption(c *UserConnectionOptionContext)

	// ExitPasswordConnectionOption is called when exiting the passwordConnectionOption production.
	ExitPasswordConnectionOption(c *PasswordConnectionOptionContext)

	// ExitDefaultAuthConnectionOption is called when exiting the defaultAuthConnectionOption production.
	ExitDefaultAuthConnectionOption(c *DefaultAuthConnectionOptionContext)

	// ExitPluginDirConnectionOption is called when exiting the pluginDirConnectionOption production.
	ExitPluginDirConnectionOption(c *PluginDirConnectionOptionContext)

	// ExitGtuidSet is called when exiting the gtuidSet production.
	ExitGtuidSet(c *GtuidSetContext)

	// ExitXaStartTransaction is called when exiting the xaStartTransaction production.
	ExitXaStartTransaction(c *XaStartTransactionContext)

	// ExitXaEndTransaction is called when exiting the xaEndTransaction production.
	ExitXaEndTransaction(c *XaEndTransactionContext)

	// ExitXaPrepareStatement is called when exiting the xaPrepareStatement production.
	ExitXaPrepareStatement(c *XaPrepareStatementContext)

	// ExitXaCommitWork is called when exiting the xaCommitWork production.
	ExitXaCommitWork(c *XaCommitWorkContext)

	// ExitXaRollbackWork is called when exiting the xaRollbackWork production.
	ExitXaRollbackWork(c *XaRollbackWorkContext)

	// ExitXaRecoverWork is called when exiting the xaRecoverWork production.
	ExitXaRecoverWork(c *XaRecoverWorkContext)

	// ExitPrepareStatement is called when exiting the prepareStatement production.
	ExitPrepareStatement(c *PrepareStatementContext)

	// ExitExecuteStatement is called when exiting the executeStatement production.
	ExitExecuteStatement(c *ExecuteStatementContext)

	// ExitDeallocatePrepare is called when exiting the deallocatePrepare production.
	ExitDeallocatePrepare(c *DeallocatePrepareContext)

	// ExitRoutineBody is called when exiting the routineBody production.
	ExitRoutineBody(c *RoutineBodyContext)

	// ExitBlockStatement is called when exiting the blockStatement production.
	ExitBlockStatement(c *BlockStatementContext)

	// ExitCaseStatement is called when exiting the caseStatement production.
	ExitCaseStatement(c *CaseStatementContext)

	// ExitIfStatement is called when exiting the ifStatement production.
	ExitIfStatement(c *IfStatementContext)

	// ExitIterateStatement is called when exiting the iterateStatement production.
	ExitIterateStatement(c *IterateStatementContext)

	// ExitLeaveStatement is called when exiting the leaveStatement production.
	ExitLeaveStatement(c *LeaveStatementContext)

	// ExitLoopStatement is called when exiting the loopStatement production.
	ExitLoopStatement(c *LoopStatementContext)

	// ExitRepeatStatement is called when exiting the repeatStatement production.
	ExitRepeatStatement(c *RepeatStatementContext)

	// ExitReturnStatement is called when exiting the returnStatement production.
	ExitReturnStatement(c *ReturnStatementContext)

	// ExitWhileStatement is called when exiting the whileStatement production.
	ExitWhileStatement(c *WhileStatementContext)

	// ExitCloseCursor is called when exiting the CloseCursor production.
	ExitCloseCursor(c *CloseCursorContext)

	// ExitFetchCursor is called when exiting the FetchCursor production.
	ExitFetchCursor(c *FetchCursorContext)

	// ExitOpenCursor is called when exiting the OpenCursor production.
	ExitOpenCursor(c *OpenCursorContext)

	// ExitDeclareVariable is called when exiting the declareVariable production.
	ExitDeclareVariable(c *DeclareVariableContext)

	// ExitDeclareCondition is called when exiting the declareCondition production.
	ExitDeclareCondition(c *DeclareConditionContext)

	// ExitDeclareCursor is called when exiting the declareCursor production.
	ExitDeclareCursor(c *DeclareCursorContext)

	// ExitDeclareHandler is called when exiting the declareHandler production.
	ExitDeclareHandler(c *DeclareHandlerContext)

	// ExitHandlerConditionCode is called when exiting the handlerConditionCode production.
	ExitHandlerConditionCode(c *HandlerConditionCodeContext)

	// ExitHandlerConditionState is called when exiting the handlerConditionState production.
	ExitHandlerConditionState(c *HandlerConditionStateContext)

	// ExitHandlerConditionName is called when exiting the handlerConditionName production.
	ExitHandlerConditionName(c *HandlerConditionNameContext)

	// ExitHandlerConditionWarning is called when exiting the handlerConditionWarning production.
	ExitHandlerConditionWarning(c *HandlerConditionWarningContext)

	// ExitHandlerConditionNotfound is called when exiting the handlerConditionNotfound production.
	ExitHandlerConditionNotfound(c *HandlerConditionNotfoundContext)

	// ExitHandlerConditionException is called when exiting the handlerConditionException production.
	ExitHandlerConditionException(c *HandlerConditionExceptionContext)

	// ExitProcedureSqlStatement is called when exiting the procedureSqlStatement production.
	ExitProcedureSqlStatement(c *ProcedureSqlStatementContext)

	// ExitCaseAlternative is called when exiting the caseAlternative production.
	ExitCaseAlternative(c *CaseAlternativeContext)

	// ExitElifAlternative is called when exiting the elifAlternative production.
	ExitElifAlternative(c *ElifAlternativeContext)

	// ExitAlterUserMysqlV56 is called when exiting the alterUserMysqlV56 production.
	ExitAlterUserMysqlV56(c *AlterUserMysqlV56Context)

	// ExitAlterUserMysqlV80 is called when exiting the alterUserMysqlV80 production.
	ExitAlterUserMysqlV80(c *AlterUserMysqlV80Context)

	// ExitCreateUserMysqlV56 is called when exiting the createUserMysqlV56 production.
	ExitCreateUserMysqlV56(c *CreateUserMysqlV56Context)

	// ExitCreateUserMysqlV80 is called when exiting the createUserMysqlV80 production.
	ExitCreateUserMysqlV80(c *CreateUserMysqlV80Context)

	// ExitDropUser is called when exiting the dropUser production.
	ExitDropUser(c *DropUserContext)

	// ExitGrantStatement is called when exiting the grantStatement production.
	ExitGrantStatement(c *GrantStatementContext)

	// ExitRoleOption is called when exiting the roleOption production.
	ExitRoleOption(c *RoleOptionContext)

	// ExitGrantProxy is called when exiting the grantProxy production.
	ExitGrantProxy(c *GrantProxyContext)

	// ExitRenameUser is called when exiting the renameUser production.
	ExitRenameUser(c *RenameUserContext)

	// ExitDetailRevoke is called when exiting the detailRevoke production.
	ExitDetailRevoke(c *DetailRevokeContext)

	// ExitShortRevoke is called when exiting the shortRevoke production.
	ExitShortRevoke(c *ShortRevokeContext)

	// ExitRoleRevoke is called when exiting the roleRevoke production.
	ExitRoleRevoke(c *RoleRevokeContext)

	// ExitRevokeProxy is called when exiting the revokeProxy production.
	ExitRevokeProxy(c *RevokeProxyContext)

	// ExitSetPasswordStatement is called when exiting the setPasswordStatement production.
	ExitSetPasswordStatement(c *SetPasswordStatementContext)

	// ExitUserSpecification is called when exiting the userSpecification production.
	ExitUserSpecification(c *UserSpecificationContext)

	// ExitHashAuthOption is called when exiting the hashAuthOption production.
	ExitHashAuthOption(c *HashAuthOptionContext)

	// ExitRandomAuthOption is called when exiting the randomAuthOption production.
	ExitRandomAuthOption(c *RandomAuthOptionContext)

	// ExitStringAuthOption is called when exiting the stringAuthOption production.
	ExitStringAuthOption(c *StringAuthOptionContext)

	// ExitModuleAuthOption is called when exiting the moduleAuthOption production.
	ExitModuleAuthOption(c *ModuleAuthOptionContext)

	// ExitSimpleAuthOption is called when exiting the simpleAuthOption production.
	ExitSimpleAuthOption(c *SimpleAuthOptionContext)

	// ExitAuthOptionClause is called when exiting the authOptionClause production.
	ExitAuthOptionClause(c *AuthOptionClauseContext)

	// ExitModule is called when exiting the module production.
	ExitModule(c *ModuleContext)

	// ExitPasswordModuleOption is called when exiting the passwordModuleOption production.
	ExitPasswordModuleOption(c *PasswordModuleOptionContext)

	// ExitTlsOption is called when exiting the tlsOption production.
	ExitTlsOption(c *TlsOptionContext)

	// ExitUserResourceOption is called when exiting the userResourceOption production.
	ExitUserResourceOption(c *UserResourceOptionContext)

	// ExitUserPasswordOption is called when exiting the userPasswordOption production.
	ExitUserPasswordOption(c *UserPasswordOptionContext)

	// ExitUserLockOption is called when exiting the userLockOption production.
	ExitUserLockOption(c *UserLockOptionContext)

	// ExitPrivelegeClause is called when exiting the privelegeClause production.
	ExitPrivelegeClause(c *PrivelegeClauseContext)

	// ExitPrivilege is called when exiting the privilege production.
	ExitPrivilege(c *PrivilegeContext)

	// ExitCurrentSchemaPriviLevel is called when exiting the currentSchemaPriviLevel production.
	ExitCurrentSchemaPriviLevel(c *CurrentSchemaPriviLevelContext)

	// ExitGlobalPrivLevel is called when exiting the globalPrivLevel production.
	ExitGlobalPrivLevel(c *GlobalPrivLevelContext)

	// ExitDefiniteSchemaPrivLevel is called when exiting the definiteSchemaPrivLevel production.
	ExitDefiniteSchemaPrivLevel(c *DefiniteSchemaPrivLevelContext)

	// ExitDefiniteFullTablePrivLevel is called when exiting the definiteFullTablePrivLevel production.
	ExitDefiniteFullTablePrivLevel(c *DefiniteFullTablePrivLevelContext)

	// ExitDefiniteFullTablePrivLevel2 is called when exiting the definiteFullTablePrivLevel2 production.
	ExitDefiniteFullTablePrivLevel2(c *DefiniteFullTablePrivLevel2Context)

	// ExitDefiniteTablePrivLevel is called when exiting the definiteTablePrivLevel production.
	ExitDefiniteTablePrivLevel(c *DefiniteTablePrivLevelContext)

	// ExitRenameUserClause is called when exiting the renameUserClause production.
	ExitRenameUserClause(c *RenameUserClauseContext)

	// ExitAnalyzeTable is called when exiting the analyzeTable production.
	ExitAnalyzeTable(c *AnalyzeTableContext)

	// ExitCheckTable is called when exiting the checkTable production.
	ExitCheckTable(c *CheckTableContext)

	// ExitChecksumTable is called when exiting the checksumTable production.
	ExitChecksumTable(c *ChecksumTableContext)

	// ExitOptimizeTable is called when exiting the optimizeTable production.
	ExitOptimizeTable(c *OptimizeTableContext)

	// ExitRepairTable is called when exiting the repairTable production.
	ExitRepairTable(c *RepairTableContext)

	// ExitCheckTableOption is called when exiting the checkTableOption production.
	ExitCheckTableOption(c *CheckTableOptionContext)

	// ExitCreateUdfunction is called when exiting the createUdfunction production.
	ExitCreateUdfunction(c *CreateUdfunctionContext)

	// ExitInstallPlugin is called when exiting the installPlugin production.
	ExitInstallPlugin(c *InstallPluginContext)

	// ExitUninstallPlugin is called when exiting the uninstallPlugin production.
	ExitUninstallPlugin(c *UninstallPluginContext)

	// ExitSetVariable is called when exiting the setVariable production.
	ExitSetVariable(c *SetVariableContext)

	// ExitSetCharset is called when exiting the setCharset production.
	ExitSetCharset(c *SetCharsetContext)

	// ExitSetNames is called when exiting the setNames production.
	ExitSetNames(c *SetNamesContext)

	// ExitSetPassword is called when exiting the setPassword production.
	ExitSetPassword(c *SetPasswordContext)

	// ExitSetTransaction is called when exiting the setTransaction production.
	ExitSetTransaction(c *SetTransactionContext)

	// ExitSetAutocommit is called when exiting the setAutocommit production.
	ExitSetAutocommit(c *SetAutocommitContext)

	// ExitSetNewValueInsideTrigger is called when exiting the setNewValueInsideTrigger production.
	ExitSetNewValueInsideTrigger(c *SetNewValueInsideTriggerContext)

	// ExitShowMasterLogs is called when exiting the showMasterLogs production.
	ExitShowMasterLogs(c *ShowMasterLogsContext)

	// ExitShowLogEvents is called when exiting the showLogEvents production.
	ExitShowLogEvents(c *ShowLogEventsContext)

	// ExitShowObjectFilter is called when exiting the showObjectFilter production.
	ExitShowObjectFilter(c *ShowObjectFilterContext)

	// ExitShowColumns is called when exiting the showColumns production.
	ExitShowColumns(c *ShowColumnsContext)

	// ExitShowCreateDb is called when exiting the showCreateDb production.
	ExitShowCreateDb(c *ShowCreateDbContext)

	// ExitShowCreateFullIdObject is called when exiting the showCreateFullIdObject production.
	ExitShowCreateFullIdObject(c *ShowCreateFullIdObjectContext)

	// ExitShowCreateUser is called when exiting the showCreateUser production.
	ExitShowCreateUser(c *ShowCreateUserContext)

	// ExitShowEngine is called when exiting the showEngine production.
	ExitShowEngine(c *ShowEngineContext)

	// ExitShowGlobalInfo is called when exiting the showGlobalInfo production.
	ExitShowGlobalInfo(c *ShowGlobalInfoContext)

	// ExitShowErrors is called when exiting the showErrors production.
	ExitShowErrors(c *ShowErrorsContext)

	// ExitShowCountErrors is called when exiting the showCountErrors production.
	ExitShowCountErrors(c *ShowCountErrorsContext)

	// ExitShowSchemaFilter is called when exiting the showSchemaFilter production.
	ExitShowSchemaFilter(c *ShowSchemaFilterContext)

	// ExitShowRoutine is called when exiting the showRoutine production.
	ExitShowRoutine(c *ShowRoutineContext)

	// ExitShowGrants is called when exiting the showGrants production.
	ExitShowGrants(c *ShowGrantsContext)

	// ExitShowIndexes is called when exiting the showIndexes production.
	ExitShowIndexes(c *ShowIndexesContext)

	// ExitShowOpenTables is called when exiting the showOpenTables production.
	ExitShowOpenTables(c *ShowOpenTablesContext)

	// ExitShowProfile is called when exiting the showProfile production.
	ExitShowProfile(c *ShowProfileContext)

	// ExitShowSlaveStatus is called when exiting the showSlaveStatus production.
	ExitShowSlaveStatus(c *ShowSlaveStatusContext)

	// ExitVariableClause is called when exiting the variableClause production.
	ExitVariableClause(c *VariableClauseContext)

	// ExitShowCommonEntity is called when exiting the showCommonEntity production.
	ExitShowCommonEntity(c *ShowCommonEntityContext)

	// ExitShowFilter is called when exiting the showFilter production.
	ExitShowFilter(c *ShowFilterContext)

	// ExitShowGlobalInfoClause is called when exiting the showGlobalInfoClause production.
	ExitShowGlobalInfoClause(c *ShowGlobalInfoClauseContext)

	// ExitShowSchemaEntity is called when exiting the showSchemaEntity production.
	ExitShowSchemaEntity(c *ShowSchemaEntityContext)

	// ExitShowProfileType is called when exiting the showProfileType production.
	ExitShowProfileType(c *ShowProfileTypeContext)

	// ExitBinlogStatement is called when exiting the binlogStatement production.
	ExitBinlogStatement(c *BinlogStatementContext)

	// ExitCacheIndexStatement is called when exiting the cacheIndexStatement production.
	ExitCacheIndexStatement(c *CacheIndexStatementContext)

	// ExitFlushStatement is called when exiting the flushStatement production.
	ExitFlushStatement(c *FlushStatementContext)

	// ExitKillStatement is called when exiting the killStatement production.
	ExitKillStatement(c *KillStatementContext)

	// ExitLoadIndexIntoCache is called when exiting the loadIndexIntoCache production.
	ExitLoadIndexIntoCache(c *LoadIndexIntoCacheContext)

	// ExitResetStatement is called when exiting the resetStatement production.
	ExitResetStatement(c *ResetStatementContext)

	// ExitShutdownStatement is called when exiting the shutdownStatement production.
	ExitShutdownStatement(c *ShutdownStatementContext)

	// ExitTableIndexes is called when exiting the tableIndexes production.
	ExitTableIndexes(c *TableIndexesContext)

	// ExitSimpleFlushOption is called when exiting the simpleFlushOption production.
	ExitSimpleFlushOption(c *SimpleFlushOptionContext)

	// ExitChannelFlushOption is called when exiting the channelFlushOption production.
	ExitChannelFlushOption(c *ChannelFlushOptionContext)

	// ExitTableFlushOption is called when exiting the tableFlushOption production.
	ExitTableFlushOption(c *TableFlushOptionContext)

	// ExitFlushTableOption is called when exiting the flushTableOption production.
	ExitFlushTableOption(c *FlushTableOptionContext)

	// ExitLoadedTableIndexes is called when exiting the loadedTableIndexes production.
	ExitLoadedTableIndexes(c *LoadedTableIndexesContext)

	// ExitSimpleDescribeStatement is called when exiting the simpleDescribeStatement production.
	ExitSimpleDescribeStatement(c *SimpleDescribeStatementContext)

	// ExitFullDescribeStatement is called when exiting the fullDescribeStatement production.
	ExitFullDescribeStatement(c *FullDescribeStatementContext)

	// ExitHelpStatement is called when exiting the helpStatement production.
	ExitHelpStatement(c *HelpStatementContext)

	// ExitUseStatement is called when exiting the useStatement production.
	ExitUseStatement(c *UseStatementContext)

	// ExitSignalStatement is called when exiting the signalStatement production.
	ExitSignalStatement(c *SignalStatementContext)

	// ExitResignalStatement is called when exiting the resignalStatement production.
	ExitResignalStatement(c *ResignalStatementContext)

	// ExitSignalConditionInformation is called when exiting the signalConditionInformation production.
	ExitSignalConditionInformation(c *SignalConditionInformationContext)

	// ExitWithStatement is called when exiting the withStatement production.
	ExitWithStatement(c *WithStatementContext)

	// ExitTableStatement is called when exiting the tableStatement production.
	ExitTableStatement(c *TableStatementContext)

	// ExitDiagnosticsStatement is called when exiting the diagnosticsStatement production.
	ExitDiagnosticsStatement(c *DiagnosticsStatementContext)

	// ExitDiagnosticsConditionInformationName is called when exiting the diagnosticsConditionInformationName production.
	ExitDiagnosticsConditionInformationName(c *DiagnosticsConditionInformationNameContext)

	// ExitDescribeStatements is called when exiting the describeStatements production.
	ExitDescribeStatements(c *DescribeStatementsContext)

	// ExitDescribeConnection is called when exiting the describeConnection production.
	ExitDescribeConnection(c *DescribeConnectionContext)

	// ExitFullId is called when exiting the fullId production.
	ExitFullId(c *FullIdContext)

	// ExitTableName is called when exiting the tableName production.
	ExitTableName(c *TableNameContext)

	// ExitRoleName is called when exiting the roleName production.
	ExitRoleName(c *RoleNameContext)

	// ExitFullColumnName is called when exiting the fullColumnName production.
	ExitFullColumnName(c *FullColumnNameContext)

	// ExitIndexColumnName is called when exiting the indexColumnName production.
	ExitIndexColumnName(c *IndexColumnNameContext)

	// ExitSimpleUserName is called when exiting the simpleUserName production.
	ExitSimpleUserName(c *SimpleUserNameContext)

	// ExitHostName is called when exiting the hostName production.
	ExitHostName(c *HostNameContext)

	// ExitUserName is called when exiting the userName production.
	ExitUserName(c *UserNameContext)

	// ExitMysqlVariable is called when exiting the mysqlVariable production.
	ExitMysqlVariable(c *MysqlVariableContext)

	// ExitCharsetName is called when exiting the charsetName production.
	ExitCharsetName(c *CharsetNameContext)

	// ExitCollationName is called when exiting the collationName production.
	ExitCollationName(c *CollationNameContext)

	// ExitEngineName is called when exiting the engineName production.
	ExitEngineName(c *EngineNameContext)

	// ExitEngineNameBase is called when exiting the engineNameBase production.
	ExitEngineNameBase(c *EngineNameBaseContext)

	// ExitUuidSet is called when exiting the uuidSet production.
	ExitUuidSet(c *UuidSetContext)

	// ExitXid is called when exiting the xid production.
	ExitXid(c *XidContext)

	// ExitXuidStringId is called when exiting the xuidStringId production.
	ExitXuidStringId(c *XuidStringIdContext)

	// ExitAuthPlugin is called when exiting the authPlugin production.
	ExitAuthPlugin(c *AuthPluginContext)

	// ExitUid is called when exiting the uid production.
	ExitUid(c *UidContext)

	// ExitSimpleId is called when exiting the simpleId production.
	ExitSimpleId(c *SimpleIdContext)

	// ExitDottedId is called when exiting the dottedId production.
	ExitDottedId(c *DottedIdContext)

	// ExitDecimalLiteral is called when exiting the decimalLiteral production.
	ExitDecimalLiteral(c *DecimalLiteralContext)

	// ExitFileSizeLiteral is called when exiting the fileSizeLiteral production.
	ExitFileSizeLiteral(c *FileSizeLiteralContext)

	// ExitStringLiteral is called when exiting the stringLiteral production.
	ExitStringLiteral(c *StringLiteralContext)

	// ExitBooleanLiteral is called when exiting the booleanLiteral production.
	ExitBooleanLiteral(c *BooleanLiteralContext)

	// ExitHexadecimalLiteral is called when exiting the hexadecimalLiteral production.
	ExitHexadecimalLiteral(c *HexadecimalLiteralContext)

	// ExitNullNotnull is called when exiting the nullNotnull production.
	ExitNullNotnull(c *NullNotnullContext)

	// ExitConstant is called when exiting the constant production.
	ExitConstant(c *ConstantContext)

	// ExitStringDataType is called when exiting the stringDataType production.
	ExitStringDataType(c *StringDataTypeContext)

	// ExitNationalVaryingStringDataType is called when exiting the nationalVaryingStringDataType production.
	ExitNationalVaryingStringDataType(c *NationalVaryingStringDataTypeContext)

	// ExitNationalStringDataType is called when exiting the nationalStringDataType production.
	ExitNationalStringDataType(c *NationalStringDataTypeContext)

	// ExitDimensionDataType is called when exiting the dimensionDataType production.
	ExitDimensionDataType(c *DimensionDataTypeContext)

	// ExitSimpleDataType is called when exiting the simpleDataType production.
	ExitSimpleDataType(c *SimpleDataTypeContext)

	// ExitCollectionDataType is called when exiting the collectionDataType production.
	ExitCollectionDataType(c *CollectionDataTypeContext)

	// ExitSpatialDataType is called when exiting the spatialDataType production.
	ExitSpatialDataType(c *SpatialDataTypeContext)

	// ExitLongVarcharDataType is called when exiting the longVarcharDataType production.
	ExitLongVarcharDataType(c *LongVarcharDataTypeContext)

	// ExitLongVarbinaryDataType is called when exiting the longVarbinaryDataType production.
	ExitLongVarbinaryDataType(c *LongVarbinaryDataTypeContext)

	// ExitCollectionOptions is called when exiting the collectionOptions production.
	ExitCollectionOptions(c *CollectionOptionsContext)

	// ExitConvertedDataType is called when exiting the convertedDataType production.
	ExitConvertedDataType(c *ConvertedDataTypeContext)

	// ExitLengthOneDimension is called when exiting the lengthOneDimension production.
	ExitLengthOneDimension(c *LengthOneDimensionContext)

	// ExitLengthTwoDimension is called when exiting the lengthTwoDimension production.
	ExitLengthTwoDimension(c *LengthTwoDimensionContext)

	// ExitLengthTwoOptionalDimension is called when exiting the lengthTwoOptionalDimension production.
	ExitLengthTwoOptionalDimension(c *LengthTwoOptionalDimensionContext)

	// ExitUidList is called when exiting the uidList production.
	ExitUidList(c *UidListContext)

	// ExitFullColumnNameList is called when exiting the fullColumnNameList production.
	ExitFullColumnNameList(c *FullColumnNameListContext)

	// ExitTables is called when exiting the tables production.
	ExitTables(c *TablesContext)

	// ExitIndexColumnNames is called when exiting the indexColumnNames production.
	ExitIndexColumnNames(c *IndexColumnNamesContext)

	// ExitExpressions is called when exiting the expressions production.
	ExitExpressions(c *ExpressionsContext)

	// ExitExpressionsWithDefaults is called when exiting the expressionsWithDefaults production.
	ExitExpressionsWithDefaults(c *ExpressionsWithDefaultsContext)

	// ExitConstants is called when exiting the constants production.
	ExitConstants(c *ConstantsContext)

	// ExitSimpleStrings is called when exiting the simpleStrings production.
	ExitSimpleStrings(c *SimpleStringsContext)

	// ExitUserVariables is called when exiting the userVariables production.
	ExitUserVariables(c *UserVariablesContext)

	// ExitDefaultValue is called when exiting the defaultValue production.
	ExitDefaultValue(c *DefaultValueContext)

	// ExitCurrentTimestamp is called when exiting the currentTimestamp production.
	ExitCurrentTimestamp(c *CurrentTimestampContext)

	// ExitExpressionOrDefault is called when exiting the expressionOrDefault production.
	ExitExpressionOrDefault(c *ExpressionOrDefaultContext)

	// ExitIfExists is called when exiting the ifExists production.
	ExitIfExists(c *IfExistsContext)

	// ExitIfNotExists is called when exiting the ifNotExists production.
	ExitIfNotExists(c *IfNotExistsContext)

	// ExitOrReplace is called when exiting the orReplace production.
	ExitOrReplace(c *OrReplaceContext)

	// ExitWaitNowaitClause is called when exiting the waitNowaitClause production.
	ExitWaitNowaitClause(c *WaitNowaitClauseContext)

	// ExitSpecificFunctionCall is called when exiting the specificFunctionCall production.
	ExitSpecificFunctionCall(c *SpecificFunctionCallContext)

	// ExitAggregateFunctionCall is called when exiting the aggregateFunctionCall production.
	ExitAggregateFunctionCall(c *AggregateFunctionCallContext)

	// ExitNonAggregateFunctionCall is called when exiting the nonAggregateFunctionCall production.
	ExitNonAggregateFunctionCall(c *NonAggregateFunctionCallContext)

	// ExitScalarFunctionCall is called when exiting the scalarFunctionCall production.
	ExitScalarFunctionCall(c *ScalarFunctionCallContext)

	// ExitUdfFunctionCall is called when exiting the udfFunctionCall production.
	ExitUdfFunctionCall(c *UdfFunctionCallContext)

	// ExitPasswordFunctionCall is called when exiting the passwordFunctionCall production.
	ExitPasswordFunctionCall(c *PasswordFunctionCallContext)

	// ExitSimpleFunctionCall is called when exiting the simpleFunctionCall production.
	ExitSimpleFunctionCall(c *SimpleFunctionCallContext)

	// ExitCurrentUser is called when exiting the currentUser production.
	ExitCurrentUser(c *CurrentUserContext)

	// ExitDataTypeFunctionCall is called when exiting the dataTypeFunctionCall production.
	ExitDataTypeFunctionCall(c *DataTypeFunctionCallContext)

	// ExitValuesFunctionCall is called when exiting the valuesFunctionCall production.
	ExitValuesFunctionCall(c *ValuesFunctionCallContext)

	// ExitCaseExpressionFunctionCall is called when exiting the caseExpressionFunctionCall production.
	ExitCaseExpressionFunctionCall(c *CaseExpressionFunctionCallContext)

	// ExitCaseFunctionCall is called when exiting the caseFunctionCall production.
	ExitCaseFunctionCall(c *CaseFunctionCallContext)

	// ExitCharFunctionCall is called when exiting the charFunctionCall production.
	ExitCharFunctionCall(c *CharFunctionCallContext)

	// ExitPositionFunctionCall is called when exiting the positionFunctionCall production.
	ExitPositionFunctionCall(c *PositionFunctionCallContext)

	// ExitSubstrFunctionCall is called when exiting the substrFunctionCall production.
	ExitSubstrFunctionCall(c *SubstrFunctionCallContext)

	// ExitTrimFunctionCall is called when exiting the trimFunctionCall production.
	ExitTrimFunctionCall(c *TrimFunctionCallContext)

	// ExitWeightFunctionCall is called when exiting the weightFunctionCall production.
	ExitWeightFunctionCall(c *WeightFunctionCallContext)

	// ExitExtractFunctionCall is called when exiting the extractFunctionCall production.
	ExitExtractFunctionCall(c *ExtractFunctionCallContext)

	// ExitGetFormatFunctionCall is called when exiting the getFormatFunctionCall production.
	ExitGetFormatFunctionCall(c *GetFormatFunctionCallContext)

	// ExitJsonValueFunctionCall is called when exiting the jsonValueFunctionCall production.
	ExitJsonValueFunctionCall(c *JsonValueFunctionCallContext)

	// ExitCaseFuncAlternative is called when exiting the caseFuncAlternative production.
	ExitCaseFuncAlternative(c *CaseFuncAlternativeContext)

	// ExitLevelWeightList is called when exiting the levelWeightList production.
	ExitLevelWeightList(c *LevelWeightListContext)

	// ExitLevelWeightRange is called when exiting the levelWeightRange production.
	ExitLevelWeightRange(c *LevelWeightRangeContext)

	// ExitLevelInWeightListElement is called when exiting the levelInWeightListElement production.
	ExitLevelInWeightListElement(c *LevelInWeightListElementContext)

	// ExitAggregateWindowedFunction is called when exiting the aggregateWindowedFunction production.
	ExitAggregateWindowedFunction(c *AggregateWindowedFunctionContext)

	// ExitNonAggregateWindowedFunction is called when exiting the nonAggregateWindowedFunction production.
	ExitNonAggregateWindowedFunction(c *NonAggregateWindowedFunctionContext)

	// ExitOverClause is called when exiting the overClause production.
	ExitOverClause(c *OverClauseContext)

	// ExitWindowSpec is called when exiting the windowSpec production.
	ExitWindowSpec(c *WindowSpecContext)

	// ExitWindowName is called when exiting the windowName production.
	ExitWindowName(c *WindowNameContext)

	// ExitFrameClause is called when exiting the frameClause production.
	ExitFrameClause(c *FrameClauseContext)

	// ExitFrameUnits is called when exiting the frameUnits production.
	ExitFrameUnits(c *FrameUnitsContext)

	// ExitFrameExtent is called when exiting the frameExtent production.
	ExitFrameExtent(c *FrameExtentContext)

	// ExitFrameBetween is called when exiting the frameBetween production.
	ExitFrameBetween(c *FrameBetweenContext)

	// ExitFrameRange is called when exiting the frameRange production.
	ExitFrameRange(c *FrameRangeContext)

	// ExitPartitionClause is called when exiting the partitionClause production.
	ExitPartitionClause(c *PartitionClauseContext)

	// ExitScalarFunctionName is called when exiting the scalarFunctionName production.
	ExitScalarFunctionName(c *ScalarFunctionNameContext)

	// ExitPasswordFunctionClause is called when exiting the passwordFunctionClause production.
	ExitPasswordFunctionClause(c *PasswordFunctionClauseContext)

	// ExitFunctionArgs is called when exiting the functionArgs production.
	ExitFunctionArgs(c *FunctionArgsContext)

	// ExitFunctionArg is called when exiting the functionArg production.
	ExitFunctionArg(c *FunctionArgContext)

	// ExitIsExpression is called when exiting the isExpression production.
	ExitIsExpression(c *IsExpressionContext)

	// ExitNotExpression is called when exiting the notExpression production.
	ExitNotExpression(c *NotExpressionContext)

	// ExitLogicalExpression is called when exiting the logicalExpression production.
	ExitLogicalExpression(c *LogicalExpressionContext)

	// ExitPredicateExpression is called when exiting the predicateExpression production.
	ExitPredicateExpression(c *PredicateExpressionContext)

	// ExitSoundsLikePredicate is called when exiting the soundsLikePredicate production.
	ExitSoundsLikePredicate(c *SoundsLikePredicateContext)

	// ExitExpressionAtomPredicate is called when exiting the expressionAtomPredicate production.
	ExitExpressionAtomPredicate(c *ExpressionAtomPredicateContext)

	// ExitSubqueryComparisonPredicate is called when exiting the subqueryComparisonPredicate production.
	ExitSubqueryComparisonPredicate(c *SubqueryComparisonPredicateContext)

	// ExitJsonMemberOfPredicate is called when exiting the jsonMemberOfPredicate production.
	ExitJsonMemberOfPredicate(c *JsonMemberOfPredicateContext)

	// ExitBinaryComparisonPredicate is called when exiting the binaryComparisonPredicate production.
	ExitBinaryComparisonPredicate(c *BinaryComparisonPredicateContext)

	// ExitInPredicate is called when exiting the inPredicate production.
	ExitInPredicate(c *InPredicateContext)

	// ExitBetweenPredicate is called when exiting the betweenPredicate production.
	ExitBetweenPredicate(c *BetweenPredicateContext)

	// ExitIsNullPredicate is called when exiting the isNullPredicate production.
	ExitIsNullPredicate(c *IsNullPredicateContext)

	// ExitLikePredicate is called when exiting the likePredicate production.
	ExitLikePredicate(c *LikePredicateContext)

	// ExitRegexpPredicate is called when exiting the regexpPredicate production.
	ExitRegexpPredicate(c *RegexpPredicateContext)

	// ExitUnaryExpressionAtom is called when exiting the unaryExpressionAtom production.
	ExitUnaryExpressionAtom(c *UnaryExpressionAtomContext)

	// ExitCollateExpressionAtom is called when exiting the collateExpressionAtom production.
	ExitCollateExpressionAtom(c *CollateExpressionAtomContext)

	// ExitVariableAssignExpressionAtom is called when exiting the variableAssignExpressionAtom production.
	ExitVariableAssignExpressionAtom(c *VariableAssignExpressionAtomContext)

	// ExitMysqlVariableExpressionAtom is called when exiting the mysqlVariableExpressionAtom production.
	ExitMysqlVariableExpressionAtom(c *MysqlVariableExpressionAtomContext)

	// ExitNestedExpressionAtom is called when exiting the nestedExpressionAtom production.
	ExitNestedExpressionAtom(c *NestedExpressionAtomContext)

	// ExitNestedRowExpressionAtom is called when exiting the nestedRowExpressionAtom production.
	ExitNestedRowExpressionAtom(c *NestedRowExpressionAtomContext)

	// ExitMathExpressionAtom is called when exiting the mathExpressionAtom production.
	ExitMathExpressionAtom(c *MathExpressionAtomContext)

	// ExitExistsExpressionAtom is called when exiting the existsExpressionAtom production.
	ExitExistsExpressionAtom(c *ExistsExpressionAtomContext)

	// ExitIntervalExpressionAtom is called when exiting the intervalExpressionAtom production.
	ExitIntervalExpressionAtom(c *IntervalExpressionAtomContext)

	// ExitJsonExpressionAtom is called when exiting the jsonExpressionAtom production.
	ExitJsonExpressionAtom(c *JsonExpressionAtomContext)

	// ExitSubqueryExpressionAtom is called when exiting the subqueryExpressionAtom production.
	ExitSubqueryExpressionAtom(c *SubqueryExpressionAtomContext)

	// ExitConstantExpressionAtom is called when exiting the constantExpressionAtom production.
	ExitConstantExpressionAtom(c *ConstantExpressionAtomContext)

	// ExitFunctionCallExpressionAtom is called when exiting the functionCallExpressionAtom production.
	ExitFunctionCallExpressionAtom(c *FunctionCallExpressionAtomContext)

	// ExitBinaryExpressionAtom is called when exiting the binaryExpressionAtom production.
	ExitBinaryExpressionAtom(c *BinaryExpressionAtomContext)

	// ExitFullColumnNameExpressionAtom is called when exiting the fullColumnNameExpressionAtom production.
	ExitFullColumnNameExpressionAtom(c *FullColumnNameExpressionAtomContext)

	// ExitBitExpressionAtom is called when exiting the bitExpressionAtom production.
	ExitBitExpressionAtom(c *BitExpressionAtomContext)

	// ExitUnaryOperator is called when exiting the unaryOperator production.
	ExitUnaryOperator(c *UnaryOperatorContext)

	// ExitComparisonOperator is called when exiting the comparisonOperator production.
	ExitComparisonOperator(c *ComparisonOperatorContext)

	// ExitLogicalOperator is called when exiting the logicalOperator production.
	ExitLogicalOperator(c *LogicalOperatorContext)

	// ExitBitOperator is called when exiting the bitOperator production.
	ExitBitOperator(c *BitOperatorContext)

	// ExitMultOperator is called when exiting the multOperator production.
	ExitMultOperator(c *MultOperatorContext)

	// ExitAddOperator is called when exiting the addOperator production.
	ExitAddOperator(c *AddOperatorContext)

	// ExitJsonOperator is called when exiting the jsonOperator production.
	ExitJsonOperator(c *JsonOperatorContext)

	// ExitCharsetNameBase is called when exiting the charsetNameBase production.
	ExitCharsetNameBase(c *CharsetNameBaseContext)

	// ExitTransactionLevelBase is called when exiting the transactionLevelBase production.
	ExitTransactionLevelBase(c *TransactionLevelBaseContext)

	// ExitPrivilegesBase is called when exiting the privilegesBase production.
	ExitPrivilegesBase(c *PrivilegesBaseContext)

	// ExitIntervalTypeBase is called when exiting the intervalTypeBase production.
	ExitIntervalTypeBase(c *IntervalTypeBaseContext)

	// ExitDataTypeBase is called when exiting the dataTypeBase production.
	ExitDataTypeBase(c *DataTypeBaseContext)

	// ExitKeywordsCanBeId is called when exiting the keywordsCanBeId production.
	ExitKeywordsCanBeId(c *KeywordsCanBeIdContext)

	// ExitFunctionNameBase is called when exiting the functionNameBase production.
	ExitFunctionNameBase(c *FunctionNameBaseContext)
}
