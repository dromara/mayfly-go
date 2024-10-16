// Code generated from MySqlParser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package parser // MySqlParser
import "github.com/antlr4-go/antlr/v4"

// BaseMySqlParserListener is a complete listener for a parse tree produced by MySqlParser.
type BaseMySqlParserListener struct{}

var _ MySqlParserListener = &BaseMySqlParserListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseMySqlParserListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseMySqlParserListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseMySqlParserListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseMySqlParserListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterRoot is called when production root is entered.
func (s *BaseMySqlParserListener) EnterRoot(ctx *RootContext) {}

// ExitRoot is called when production root is exited.
func (s *BaseMySqlParserListener) ExitRoot(ctx *RootContext) {}

// EnterSqlStatements is called when production sqlStatements is entered.
func (s *BaseMySqlParserListener) EnterSqlStatements(ctx *SqlStatementsContext) {}

// ExitSqlStatements is called when production sqlStatements is exited.
func (s *BaseMySqlParserListener) ExitSqlStatements(ctx *SqlStatementsContext) {}

// EnterSqlStatement is called when production sqlStatement is entered.
func (s *BaseMySqlParserListener) EnterSqlStatement(ctx *SqlStatementContext) {}

// ExitSqlStatement is called when production sqlStatement is exited.
func (s *BaseMySqlParserListener) ExitSqlStatement(ctx *SqlStatementContext) {}

// EnterEmptyStatement_ is called when production emptyStatement_ is entered.
func (s *BaseMySqlParserListener) EnterEmptyStatement_(ctx *EmptyStatement_Context) {}

// ExitEmptyStatement_ is called when production emptyStatement_ is exited.
func (s *BaseMySqlParserListener) ExitEmptyStatement_(ctx *EmptyStatement_Context) {}

// EnterDdlStatement is called when production ddlStatement is entered.
func (s *BaseMySqlParserListener) EnterDdlStatement(ctx *DdlStatementContext) {}

// ExitDdlStatement is called when production ddlStatement is exited.
func (s *BaseMySqlParserListener) ExitDdlStatement(ctx *DdlStatementContext) {}

// EnterDmlStatement is called when production dmlStatement is entered.
func (s *BaseMySqlParserListener) EnterDmlStatement(ctx *DmlStatementContext) {}

// ExitDmlStatement is called when production dmlStatement is exited.
func (s *BaseMySqlParserListener) ExitDmlStatement(ctx *DmlStatementContext) {}

// EnterTransactionStatement is called when production transactionStatement is entered.
func (s *BaseMySqlParserListener) EnterTransactionStatement(ctx *TransactionStatementContext) {}

// ExitTransactionStatement is called when production transactionStatement is exited.
func (s *BaseMySqlParserListener) ExitTransactionStatement(ctx *TransactionStatementContext) {}

// EnterReplicationStatement is called when production replicationStatement is entered.
func (s *BaseMySqlParserListener) EnterReplicationStatement(ctx *ReplicationStatementContext) {}

// ExitReplicationStatement is called when production replicationStatement is exited.
func (s *BaseMySqlParserListener) ExitReplicationStatement(ctx *ReplicationStatementContext) {}

// EnterPreparedStatement is called when production preparedStatement is entered.
func (s *BaseMySqlParserListener) EnterPreparedStatement(ctx *PreparedStatementContext) {}

// ExitPreparedStatement is called when production preparedStatement is exited.
func (s *BaseMySqlParserListener) ExitPreparedStatement(ctx *PreparedStatementContext) {}

// EnterCompoundStatement is called when production compoundStatement is entered.
func (s *BaseMySqlParserListener) EnterCompoundStatement(ctx *CompoundStatementContext) {}

// ExitCompoundStatement is called when production compoundStatement is exited.
func (s *BaseMySqlParserListener) ExitCompoundStatement(ctx *CompoundStatementContext) {}

// EnterAdministrationStatement is called when production administrationStatement is entered.
func (s *BaseMySqlParserListener) EnterAdministrationStatement(ctx *AdministrationStatementContext) {}

// ExitAdministrationStatement is called when production administrationStatement is exited.
func (s *BaseMySqlParserListener) ExitAdministrationStatement(ctx *AdministrationStatementContext) {}

// EnterUtilityStatement is called when production utilityStatement is entered.
func (s *BaseMySqlParserListener) EnterUtilityStatement(ctx *UtilityStatementContext) {}

// ExitUtilityStatement is called when production utilityStatement is exited.
func (s *BaseMySqlParserListener) ExitUtilityStatement(ctx *UtilityStatementContext) {}

// EnterCreateDatabase is called when production createDatabase is entered.
func (s *BaseMySqlParserListener) EnterCreateDatabase(ctx *CreateDatabaseContext) {}

// ExitCreateDatabase is called when production createDatabase is exited.
func (s *BaseMySqlParserListener) ExitCreateDatabase(ctx *CreateDatabaseContext) {}

// EnterCreateEvent is called when production createEvent is entered.
func (s *BaseMySqlParserListener) EnterCreateEvent(ctx *CreateEventContext) {}

// ExitCreateEvent is called when production createEvent is exited.
func (s *BaseMySqlParserListener) ExitCreateEvent(ctx *CreateEventContext) {}

// EnterCreateIndex is called when production createIndex is entered.
func (s *BaseMySqlParserListener) EnterCreateIndex(ctx *CreateIndexContext) {}

// ExitCreateIndex is called when production createIndex is exited.
func (s *BaseMySqlParserListener) ExitCreateIndex(ctx *CreateIndexContext) {}

// EnterCreateLogfileGroup is called when production createLogfileGroup is entered.
func (s *BaseMySqlParserListener) EnterCreateLogfileGroup(ctx *CreateLogfileGroupContext) {}

// ExitCreateLogfileGroup is called when production createLogfileGroup is exited.
func (s *BaseMySqlParserListener) ExitCreateLogfileGroup(ctx *CreateLogfileGroupContext) {}

// EnterCreateProcedure is called when production createProcedure is entered.
func (s *BaseMySqlParserListener) EnterCreateProcedure(ctx *CreateProcedureContext) {}

// ExitCreateProcedure is called when production createProcedure is exited.
func (s *BaseMySqlParserListener) ExitCreateProcedure(ctx *CreateProcedureContext) {}

// EnterCreateFunction is called when production createFunction is entered.
func (s *BaseMySqlParserListener) EnterCreateFunction(ctx *CreateFunctionContext) {}

// ExitCreateFunction is called when production createFunction is exited.
func (s *BaseMySqlParserListener) ExitCreateFunction(ctx *CreateFunctionContext) {}

// EnterCreateRole is called when production createRole is entered.
func (s *BaseMySqlParserListener) EnterCreateRole(ctx *CreateRoleContext) {}

// ExitCreateRole is called when production createRole is exited.
func (s *BaseMySqlParserListener) ExitCreateRole(ctx *CreateRoleContext) {}

// EnterCreateServer is called when production createServer is entered.
func (s *BaseMySqlParserListener) EnterCreateServer(ctx *CreateServerContext) {}

// ExitCreateServer is called when production createServer is exited.
func (s *BaseMySqlParserListener) ExitCreateServer(ctx *CreateServerContext) {}

// EnterCopyCreateTable is called when production copyCreateTable is entered.
func (s *BaseMySqlParserListener) EnterCopyCreateTable(ctx *CopyCreateTableContext) {}

// ExitCopyCreateTable is called when production copyCreateTable is exited.
func (s *BaseMySqlParserListener) ExitCopyCreateTable(ctx *CopyCreateTableContext) {}

// EnterQueryCreateTable is called when production queryCreateTable is entered.
func (s *BaseMySqlParserListener) EnterQueryCreateTable(ctx *QueryCreateTableContext) {}

// ExitQueryCreateTable is called when production queryCreateTable is exited.
func (s *BaseMySqlParserListener) ExitQueryCreateTable(ctx *QueryCreateTableContext) {}

// EnterColumnCreateTable is called when production columnCreateTable is entered.
func (s *BaseMySqlParserListener) EnterColumnCreateTable(ctx *ColumnCreateTableContext) {}

// ExitColumnCreateTable is called when production columnCreateTable is exited.
func (s *BaseMySqlParserListener) ExitColumnCreateTable(ctx *ColumnCreateTableContext) {}

// EnterCreateTablespaceInnodb is called when production createTablespaceInnodb is entered.
func (s *BaseMySqlParserListener) EnterCreateTablespaceInnodb(ctx *CreateTablespaceInnodbContext) {}

// ExitCreateTablespaceInnodb is called when production createTablespaceInnodb is exited.
func (s *BaseMySqlParserListener) ExitCreateTablespaceInnodb(ctx *CreateTablespaceInnodbContext) {}

// EnterCreateTablespaceNdb is called when production createTablespaceNdb is entered.
func (s *BaseMySqlParserListener) EnterCreateTablespaceNdb(ctx *CreateTablespaceNdbContext) {}

// ExitCreateTablespaceNdb is called when production createTablespaceNdb is exited.
func (s *BaseMySqlParserListener) ExitCreateTablespaceNdb(ctx *CreateTablespaceNdbContext) {}

// EnterCreateTrigger is called when production createTrigger is entered.
func (s *BaseMySqlParserListener) EnterCreateTrigger(ctx *CreateTriggerContext) {}

// ExitCreateTrigger is called when production createTrigger is exited.
func (s *BaseMySqlParserListener) ExitCreateTrigger(ctx *CreateTriggerContext) {}

// EnterWithClause is called when production withClause is entered.
func (s *BaseMySqlParserListener) EnterWithClause(ctx *WithClauseContext) {}

// ExitWithClause is called when production withClause is exited.
func (s *BaseMySqlParserListener) ExitWithClause(ctx *WithClauseContext) {}

// EnterCommonTableExpressions is called when production commonTableExpressions is entered.
func (s *BaseMySqlParserListener) EnterCommonTableExpressions(ctx *CommonTableExpressionsContext) {}

// ExitCommonTableExpressions is called when production commonTableExpressions is exited.
func (s *BaseMySqlParserListener) ExitCommonTableExpressions(ctx *CommonTableExpressionsContext) {}

// EnterCteName is called when production cteName is entered.
func (s *BaseMySqlParserListener) EnterCteName(ctx *CteNameContext) {}

// ExitCteName is called when production cteName is exited.
func (s *BaseMySqlParserListener) ExitCteName(ctx *CteNameContext) {}

// EnterCteColumnName is called when production cteColumnName is entered.
func (s *BaseMySqlParserListener) EnterCteColumnName(ctx *CteColumnNameContext) {}

// ExitCteColumnName is called when production cteColumnName is exited.
func (s *BaseMySqlParserListener) ExitCteColumnName(ctx *CteColumnNameContext) {}

// EnterCreateView is called when production createView is entered.
func (s *BaseMySqlParserListener) EnterCreateView(ctx *CreateViewContext) {}

// ExitCreateView is called when production createView is exited.
func (s *BaseMySqlParserListener) ExitCreateView(ctx *CreateViewContext) {}

// EnterCreateDatabaseOption is called when production createDatabaseOption is entered.
func (s *BaseMySqlParserListener) EnterCreateDatabaseOption(ctx *CreateDatabaseOptionContext) {}

// ExitCreateDatabaseOption is called when production createDatabaseOption is exited.
func (s *BaseMySqlParserListener) ExitCreateDatabaseOption(ctx *CreateDatabaseOptionContext) {}

// EnterCharSet is called when production charSet is entered.
func (s *BaseMySqlParserListener) EnterCharSet(ctx *CharSetContext) {}

// ExitCharSet is called when production charSet is exited.
func (s *BaseMySqlParserListener) ExitCharSet(ctx *CharSetContext) {}

// EnterCurrentUserExpression is called when production currentUserExpression is entered.
func (s *BaseMySqlParserListener) EnterCurrentUserExpression(ctx *CurrentUserExpressionContext) {}

// ExitCurrentUserExpression is called when production currentUserExpression is exited.
func (s *BaseMySqlParserListener) ExitCurrentUserExpression(ctx *CurrentUserExpressionContext) {}

// EnterOwnerStatement is called when production ownerStatement is entered.
func (s *BaseMySqlParserListener) EnterOwnerStatement(ctx *OwnerStatementContext) {}

// ExitOwnerStatement is called when production ownerStatement is exited.
func (s *BaseMySqlParserListener) ExitOwnerStatement(ctx *OwnerStatementContext) {}

// EnterPreciseSchedule is called when production preciseSchedule is entered.
func (s *BaseMySqlParserListener) EnterPreciseSchedule(ctx *PreciseScheduleContext) {}

// ExitPreciseSchedule is called when production preciseSchedule is exited.
func (s *BaseMySqlParserListener) ExitPreciseSchedule(ctx *PreciseScheduleContext) {}

// EnterIntervalSchedule is called when production intervalSchedule is entered.
func (s *BaseMySqlParserListener) EnterIntervalSchedule(ctx *IntervalScheduleContext) {}

// ExitIntervalSchedule is called when production intervalSchedule is exited.
func (s *BaseMySqlParserListener) ExitIntervalSchedule(ctx *IntervalScheduleContext) {}

// EnterTimestampValue is called when production timestampValue is entered.
func (s *BaseMySqlParserListener) EnterTimestampValue(ctx *TimestampValueContext) {}

// ExitTimestampValue is called when production timestampValue is exited.
func (s *BaseMySqlParserListener) ExitTimestampValue(ctx *TimestampValueContext) {}

// EnterIntervalExpr is called when production intervalExpr is entered.
func (s *BaseMySqlParserListener) EnterIntervalExpr(ctx *IntervalExprContext) {}

// ExitIntervalExpr is called when production intervalExpr is exited.
func (s *BaseMySqlParserListener) ExitIntervalExpr(ctx *IntervalExprContext) {}

// EnterIntervalType is called when production intervalType is entered.
func (s *BaseMySqlParserListener) EnterIntervalType(ctx *IntervalTypeContext) {}

// ExitIntervalType is called when production intervalType is exited.
func (s *BaseMySqlParserListener) ExitIntervalType(ctx *IntervalTypeContext) {}

// EnterEnableType is called when production enableType is entered.
func (s *BaseMySqlParserListener) EnterEnableType(ctx *EnableTypeContext) {}

// ExitEnableType is called when production enableType is exited.
func (s *BaseMySqlParserListener) ExitEnableType(ctx *EnableTypeContext) {}

// EnterIndexType is called when production indexType is entered.
func (s *BaseMySqlParserListener) EnterIndexType(ctx *IndexTypeContext) {}

// ExitIndexType is called when production indexType is exited.
func (s *BaseMySqlParserListener) ExitIndexType(ctx *IndexTypeContext) {}

// EnterIndexOption is called when production indexOption is entered.
func (s *BaseMySqlParserListener) EnterIndexOption(ctx *IndexOptionContext) {}

// ExitIndexOption is called when production indexOption is exited.
func (s *BaseMySqlParserListener) ExitIndexOption(ctx *IndexOptionContext) {}

// EnterProcedureParameter is called when production procedureParameter is entered.
func (s *BaseMySqlParserListener) EnterProcedureParameter(ctx *ProcedureParameterContext) {}

// ExitProcedureParameter is called when production procedureParameter is exited.
func (s *BaseMySqlParserListener) ExitProcedureParameter(ctx *ProcedureParameterContext) {}

// EnterFunctionParameter is called when production functionParameter is entered.
func (s *BaseMySqlParserListener) EnterFunctionParameter(ctx *FunctionParameterContext) {}

// ExitFunctionParameter is called when production functionParameter is exited.
func (s *BaseMySqlParserListener) ExitFunctionParameter(ctx *FunctionParameterContext) {}

// EnterRoutineComment is called when production routineComment is entered.
func (s *BaseMySqlParserListener) EnterRoutineComment(ctx *RoutineCommentContext) {}

// ExitRoutineComment is called when production routineComment is exited.
func (s *BaseMySqlParserListener) ExitRoutineComment(ctx *RoutineCommentContext) {}

// EnterRoutineLanguage is called when production routineLanguage is entered.
func (s *BaseMySqlParserListener) EnterRoutineLanguage(ctx *RoutineLanguageContext) {}

// ExitRoutineLanguage is called when production routineLanguage is exited.
func (s *BaseMySqlParserListener) ExitRoutineLanguage(ctx *RoutineLanguageContext) {}

// EnterRoutineBehavior is called when production routineBehavior is entered.
func (s *BaseMySqlParserListener) EnterRoutineBehavior(ctx *RoutineBehaviorContext) {}

// ExitRoutineBehavior is called when production routineBehavior is exited.
func (s *BaseMySqlParserListener) ExitRoutineBehavior(ctx *RoutineBehaviorContext) {}

// EnterRoutineData is called when production routineData is entered.
func (s *BaseMySqlParserListener) EnterRoutineData(ctx *RoutineDataContext) {}

// ExitRoutineData is called when production routineData is exited.
func (s *BaseMySqlParserListener) ExitRoutineData(ctx *RoutineDataContext) {}

// EnterRoutineSecurity is called when production routineSecurity is entered.
func (s *BaseMySqlParserListener) EnterRoutineSecurity(ctx *RoutineSecurityContext) {}

// ExitRoutineSecurity is called when production routineSecurity is exited.
func (s *BaseMySqlParserListener) ExitRoutineSecurity(ctx *RoutineSecurityContext) {}

// EnterServerOption is called when production serverOption is entered.
func (s *BaseMySqlParserListener) EnterServerOption(ctx *ServerOptionContext) {}

// ExitServerOption is called when production serverOption is exited.
func (s *BaseMySqlParserListener) ExitServerOption(ctx *ServerOptionContext) {}

// EnterCreateDefinitions is called when production createDefinitions is entered.
func (s *BaseMySqlParserListener) EnterCreateDefinitions(ctx *CreateDefinitionsContext) {}

// ExitCreateDefinitions is called when production createDefinitions is exited.
func (s *BaseMySqlParserListener) ExitCreateDefinitions(ctx *CreateDefinitionsContext) {}

// EnterColumnDeclaration is called when production columnDeclaration is entered.
func (s *BaseMySqlParserListener) EnterColumnDeclaration(ctx *ColumnDeclarationContext) {}

// ExitColumnDeclaration is called when production columnDeclaration is exited.
func (s *BaseMySqlParserListener) ExitColumnDeclaration(ctx *ColumnDeclarationContext) {}

// EnterConstraintDeclaration is called when production constraintDeclaration is entered.
func (s *BaseMySqlParserListener) EnterConstraintDeclaration(ctx *ConstraintDeclarationContext) {}

// ExitConstraintDeclaration is called when production constraintDeclaration is exited.
func (s *BaseMySqlParserListener) ExitConstraintDeclaration(ctx *ConstraintDeclarationContext) {}

// EnterIndexDeclaration is called when production indexDeclaration is entered.
func (s *BaseMySqlParserListener) EnterIndexDeclaration(ctx *IndexDeclarationContext) {}

// ExitIndexDeclaration is called when production indexDeclaration is exited.
func (s *BaseMySqlParserListener) ExitIndexDeclaration(ctx *IndexDeclarationContext) {}

// EnterColumnDefinition is called when production columnDefinition is entered.
func (s *BaseMySqlParserListener) EnterColumnDefinition(ctx *ColumnDefinitionContext) {}

// ExitColumnDefinition is called when production columnDefinition is exited.
func (s *BaseMySqlParserListener) ExitColumnDefinition(ctx *ColumnDefinitionContext) {}

// EnterNullColumnConstraint is called when production nullColumnConstraint is entered.
func (s *BaseMySqlParserListener) EnterNullColumnConstraint(ctx *NullColumnConstraintContext) {}

// ExitNullColumnConstraint is called when production nullColumnConstraint is exited.
func (s *BaseMySqlParserListener) ExitNullColumnConstraint(ctx *NullColumnConstraintContext) {}

// EnterDefaultColumnConstraint is called when production defaultColumnConstraint is entered.
func (s *BaseMySqlParserListener) EnterDefaultColumnConstraint(ctx *DefaultColumnConstraintContext) {}

// ExitDefaultColumnConstraint is called when production defaultColumnConstraint is exited.
func (s *BaseMySqlParserListener) ExitDefaultColumnConstraint(ctx *DefaultColumnConstraintContext) {}

// EnterVisibilityColumnConstraint is called when production visibilityColumnConstraint is entered.
func (s *BaseMySqlParserListener) EnterVisibilityColumnConstraint(ctx *VisibilityColumnConstraintContext) {
}

// ExitVisibilityColumnConstraint is called when production visibilityColumnConstraint is exited.
func (s *BaseMySqlParserListener) ExitVisibilityColumnConstraint(ctx *VisibilityColumnConstraintContext) {
}

// EnterInvisibilityColumnConstraint is called when production invisibilityColumnConstraint is entered.
func (s *BaseMySqlParserListener) EnterInvisibilityColumnConstraint(ctx *InvisibilityColumnConstraintContext) {
}

// ExitInvisibilityColumnConstraint is called when production invisibilityColumnConstraint is exited.
func (s *BaseMySqlParserListener) ExitInvisibilityColumnConstraint(ctx *InvisibilityColumnConstraintContext) {
}

// EnterAutoIncrementColumnConstraint is called when production autoIncrementColumnConstraint is entered.
func (s *BaseMySqlParserListener) EnterAutoIncrementColumnConstraint(ctx *AutoIncrementColumnConstraintContext) {
}

// ExitAutoIncrementColumnConstraint is called when production autoIncrementColumnConstraint is exited.
func (s *BaseMySqlParserListener) ExitAutoIncrementColumnConstraint(ctx *AutoIncrementColumnConstraintContext) {
}

// EnterPrimaryKeyColumnConstraint is called when production primaryKeyColumnConstraint is entered.
func (s *BaseMySqlParserListener) EnterPrimaryKeyColumnConstraint(ctx *PrimaryKeyColumnConstraintContext) {
}

// ExitPrimaryKeyColumnConstraint is called when production primaryKeyColumnConstraint is exited.
func (s *BaseMySqlParserListener) ExitPrimaryKeyColumnConstraint(ctx *PrimaryKeyColumnConstraintContext) {
}

// EnterUniqueKeyColumnConstraint is called when production uniqueKeyColumnConstraint is entered.
func (s *BaseMySqlParserListener) EnterUniqueKeyColumnConstraint(ctx *UniqueKeyColumnConstraintContext) {
}

// ExitUniqueKeyColumnConstraint is called when production uniqueKeyColumnConstraint is exited.
func (s *BaseMySqlParserListener) ExitUniqueKeyColumnConstraint(ctx *UniqueKeyColumnConstraintContext) {
}

// EnterCommentColumnConstraint is called when production commentColumnConstraint is entered.
func (s *BaseMySqlParserListener) EnterCommentColumnConstraint(ctx *CommentColumnConstraintContext) {}

// ExitCommentColumnConstraint is called when production commentColumnConstraint is exited.
func (s *BaseMySqlParserListener) ExitCommentColumnConstraint(ctx *CommentColumnConstraintContext) {}

// EnterFormatColumnConstraint is called when production formatColumnConstraint is entered.
func (s *BaseMySqlParserListener) EnterFormatColumnConstraint(ctx *FormatColumnConstraintContext) {}

// ExitFormatColumnConstraint is called when production formatColumnConstraint is exited.
func (s *BaseMySqlParserListener) ExitFormatColumnConstraint(ctx *FormatColumnConstraintContext) {}

// EnterStorageColumnConstraint is called when production storageColumnConstraint is entered.
func (s *BaseMySqlParserListener) EnterStorageColumnConstraint(ctx *StorageColumnConstraintContext) {}

// ExitStorageColumnConstraint is called when production storageColumnConstraint is exited.
func (s *BaseMySqlParserListener) ExitStorageColumnConstraint(ctx *StorageColumnConstraintContext) {}

// EnterReferenceColumnConstraint is called when production referenceColumnConstraint is entered.
func (s *BaseMySqlParserListener) EnterReferenceColumnConstraint(ctx *ReferenceColumnConstraintContext) {
}

// ExitReferenceColumnConstraint is called when production referenceColumnConstraint is exited.
func (s *BaseMySqlParserListener) ExitReferenceColumnConstraint(ctx *ReferenceColumnConstraintContext) {
}

// EnterCollateColumnConstraint is called when production collateColumnConstraint is entered.
func (s *BaseMySqlParserListener) EnterCollateColumnConstraint(ctx *CollateColumnConstraintContext) {}

// ExitCollateColumnConstraint is called when production collateColumnConstraint is exited.
func (s *BaseMySqlParserListener) ExitCollateColumnConstraint(ctx *CollateColumnConstraintContext) {}

// EnterGeneratedColumnConstraint is called when production generatedColumnConstraint is entered.
func (s *BaseMySqlParserListener) EnterGeneratedColumnConstraint(ctx *GeneratedColumnConstraintContext) {
}

// ExitGeneratedColumnConstraint is called when production generatedColumnConstraint is exited.
func (s *BaseMySqlParserListener) ExitGeneratedColumnConstraint(ctx *GeneratedColumnConstraintContext) {
}

// EnterSerialDefaultColumnConstraint is called when production serialDefaultColumnConstraint is entered.
func (s *BaseMySqlParserListener) EnterSerialDefaultColumnConstraint(ctx *SerialDefaultColumnConstraintContext) {
}

// ExitSerialDefaultColumnConstraint is called when production serialDefaultColumnConstraint is exited.
func (s *BaseMySqlParserListener) ExitSerialDefaultColumnConstraint(ctx *SerialDefaultColumnConstraintContext) {
}

// EnterCheckColumnConstraint is called when production checkColumnConstraint is entered.
func (s *BaseMySqlParserListener) EnterCheckColumnConstraint(ctx *CheckColumnConstraintContext) {}

// ExitCheckColumnConstraint is called when production checkColumnConstraint is exited.
func (s *BaseMySqlParserListener) ExitCheckColumnConstraint(ctx *CheckColumnConstraintContext) {}

// EnterPrimaryKeyTableConstraint is called when production primaryKeyTableConstraint is entered.
func (s *BaseMySqlParserListener) EnterPrimaryKeyTableConstraint(ctx *PrimaryKeyTableConstraintContext) {
}

// ExitPrimaryKeyTableConstraint is called when production primaryKeyTableConstraint is exited.
func (s *BaseMySqlParserListener) ExitPrimaryKeyTableConstraint(ctx *PrimaryKeyTableConstraintContext) {
}

// EnterUniqueKeyTableConstraint is called when production uniqueKeyTableConstraint is entered.
func (s *BaseMySqlParserListener) EnterUniqueKeyTableConstraint(ctx *UniqueKeyTableConstraintContext) {
}

// ExitUniqueKeyTableConstraint is called when production uniqueKeyTableConstraint is exited.
func (s *BaseMySqlParserListener) ExitUniqueKeyTableConstraint(ctx *UniqueKeyTableConstraintContext) {
}

// EnterForeignKeyTableConstraint is called when production foreignKeyTableConstraint is entered.
func (s *BaseMySqlParserListener) EnterForeignKeyTableConstraint(ctx *ForeignKeyTableConstraintContext) {
}

// ExitForeignKeyTableConstraint is called when production foreignKeyTableConstraint is exited.
func (s *BaseMySqlParserListener) ExitForeignKeyTableConstraint(ctx *ForeignKeyTableConstraintContext) {
}

// EnterCheckTableConstraint is called when production checkTableConstraint is entered.
func (s *BaseMySqlParserListener) EnterCheckTableConstraint(ctx *CheckTableConstraintContext) {}

// ExitCheckTableConstraint is called when production checkTableConstraint is exited.
func (s *BaseMySqlParserListener) ExitCheckTableConstraint(ctx *CheckTableConstraintContext) {}

// EnterReferenceDefinition is called when production referenceDefinition is entered.
func (s *BaseMySqlParserListener) EnterReferenceDefinition(ctx *ReferenceDefinitionContext) {}

// ExitReferenceDefinition is called when production referenceDefinition is exited.
func (s *BaseMySqlParserListener) ExitReferenceDefinition(ctx *ReferenceDefinitionContext) {}

// EnterReferenceAction is called when production referenceAction is entered.
func (s *BaseMySqlParserListener) EnterReferenceAction(ctx *ReferenceActionContext) {}

// ExitReferenceAction is called when production referenceAction is exited.
func (s *BaseMySqlParserListener) ExitReferenceAction(ctx *ReferenceActionContext) {}

// EnterReferenceControlType is called when production referenceControlType is entered.
func (s *BaseMySqlParserListener) EnterReferenceControlType(ctx *ReferenceControlTypeContext) {}

// ExitReferenceControlType is called when production referenceControlType is exited.
func (s *BaseMySqlParserListener) ExitReferenceControlType(ctx *ReferenceControlTypeContext) {}

// EnterSimpleIndexDeclaration is called when production simpleIndexDeclaration is entered.
func (s *BaseMySqlParserListener) EnterSimpleIndexDeclaration(ctx *SimpleIndexDeclarationContext) {}

// ExitSimpleIndexDeclaration is called when production simpleIndexDeclaration is exited.
func (s *BaseMySqlParserListener) ExitSimpleIndexDeclaration(ctx *SimpleIndexDeclarationContext) {}

// EnterSpecialIndexDeclaration is called when production specialIndexDeclaration is entered.
func (s *BaseMySqlParserListener) EnterSpecialIndexDeclaration(ctx *SpecialIndexDeclarationContext) {}

// ExitSpecialIndexDeclaration is called when production specialIndexDeclaration is exited.
func (s *BaseMySqlParserListener) ExitSpecialIndexDeclaration(ctx *SpecialIndexDeclarationContext) {}

// EnterTableOptionEngine is called when production tableOptionEngine is entered.
func (s *BaseMySqlParserListener) EnterTableOptionEngine(ctx *TableOptionEngineContext) {}

// ExitTableOptionEngine is called when production tableOptionEngine is exited.
func (s *BaseMySqlParserListener) ExitTableOptionEngine(ctx *TableOptionEngineContext) {}

// EnterTableOptionEngineAttribute is called when production tableOptionEngineAttribute is entered.
func (s *BaseMySqlParserListener) EnterTableOptionEngineAttribute(ctx *TableOptionEngineAttributeContext) {
}

// ExitTableOptionEngineAttribute is called when production tableOptionEngineAttribute is exited.
func (s *BaseMySqlParserListener) ExitTableOptionEngineAttribute(ctx *TableOptionEngineAttributeContext) {
}

// EnterTableOptionAutoextendSize is called when production tableOptionAutoextendSize is entered.
func (s *BaseMySqlParserListener) EnterTableOptionAutoextendSize(ctx *TableOptionAutoextendSizeContext) {
}

// ExitTableOptionAutoextendSize is called when production tableOptionAutoextendSize is exited.
func (s *BaseMySqlParserListener) ExitTableOptionAutoextendSize(ctx *TableOptionAutoextendSizeContext) {
}

// EnterTableOptionAutoIncrement is called when production tableOptionAutoIncrement is entered.
func (s *BaseMySqlParserListener) EnterTableOptionAutoIncrement(ctx *TableOptionAutoIncrementContext) {
}

// ExitTableOptionAutoIncrement is called when production tableOptionAutoIncrement is exited.
func (s *BaseMySqlParserListener) ExitTableOptionAutoIncrement(ctx *TableOptionAutoIncrementContext) {
}

// EnterTableOptionAverage is called when production tableOptionAverage is entered.
func (s *BaseMySqlParserListener) EnterTableOptionAverage(ctx *TableOptionAverageContext) {}

// ExitTableOptionAverage is called when production tableOptionAverage is exited.
func (s *BaseMySqlParserListener) ExitTableOptionAverage(ctx *TableOptionAverageContext) {}

// EnterTableOptionCharset is called when production tableOptionCharset is entered.
func (s *BaseMySqlParserListener) EnterTableOptionCharset(ctx *TableOptionCharsetContext) {}

// ExitTableOptionCharset is called when production tableOptionCharset is exited.
func (s *BaseMySqlParserListener) ExitTableOptionCharset(ctx *TableOptionCharsetContext) {}

// EnterTableOptionChecksum is called when production tableOptionChecksum is entered.
func (s *BaseMySqlParserListener) EnterTableOptionChecksum(ctx *TableOptionChecksumContext) {}

// ExitTableOptionChecksum is called when production tableOptionChecksum is exited.
func (s *BaseMySqlParserListener) ExitTableOptionChecksum(ctx *TableOptionChecksumContext) {}

// EnterTableOptionCollate is called when production tableOptionCollate is entered.
func (s *BaseMySqlParserListener) EnterTableOptionCollate(ctx *TableOptionCollateContext) {}

// ExitTableOptionCollate is called when production tableOptionCollate is exited.
func (s *BaseMySqlParserListener) ExitTableOptionCollate(ctx *TableOptionCollateContext) {}

// EnterTableOptionComment is called when production tableOptionComment is entered.
func (s *BaseMySqlParserListener) EnterTableOptionComment(ctx *TableOptionCommentContext) {}

// ExitTableOptionComment is called when production tableOptionComment is exited.
func (s *BaseMySqlParserListener) ExitTableOptionComment(ctx *TableOptionCommentContext) {}

// EnterTableOptionCompression is called when production tableOptionCompression is entered.
func (s *BaseMySqlParserListener) EnterTableOptionCompression(ctx *TableOptionCompressionContext) {}

// ExitTableOptionCompression is called when production tableOptionCompression is exited.
func (s *BaseMySqlParserListener) ExitTableOptionCompression(ctx *TableOptionCompressionContext) {}

// EnterTableOptionConnection is called when production tableOptionConnection is entered.
func (s *BaseMySqlParserListener) EnterTableOptionConnection(ctx *TableOptionConnectionContext) {}

// ExitTableOptionConnection is called when production tableOptionConnection is exited.
func (s *BaseMySqlParserListener) ExitTableOptionConnection(ctx *TableOptionConnectionContext) {}

// EnterTableOptionDataDirectory is called when production tableOptionDataDirectory is entered.
func (s *BaseMySqlParserListener) EnterTableOptionDataDirectory(ctx *TableOptionDataDirectoryContext) {
}

// ExitTableOptionDataDirectory is called when production tableOptionDataDirectory is exited.
func (s *BaseMySqlParserListener) ExitTableOptionDataDirectory(ctx *TableOptionDataDirectoryContext) {
}

// EnterTableOptionDelay is called when production tableOptionDelay is entered.
func (s *BaseMySqlParserListener) EnterTableOptionDelay(ctx *TableOptionDelayContext) {}

// ExitTableOptionDelay is called when production tableOptionDelay is exited.
func (s *BaseMySqlParserListener) ExitTableOptionDelay(ctx *TableOptionDelayContext) {}

// EnterTableOptionEncryption is called when production tableOptionEncryption is entered.
func (s *BaseMySqlParserListener) EnterTableOptionEncryption(ctx *TableOptionEncryptionContext) {}

// ExitTableOptionEncryption is called when production tableOptionEncryption is exited.
func (s *BaseMySqlParserListener) ExitTableOptionEncryption(ctx *TableOptionEncryptionContext) {}

// EnterTableOptionPageCompressed is called when production tableOptionPageCompressed is entered.
func (s *BaseMySqlParserListener) EnterTableOptionPageCompressed(ctx *TableOptionPageCompressedContext) {
}

// ExitTableOptionPageCompressed is called when production tableOptionPageCompressed is exited.
func (s *BaseMySqlParserListener) ExitTableOptionPageCompressed(ctx *TableOptionPageCompressedContext) {
}

// EnterTableOptionPageCompressionLevel is called when production tableOptionPageCompressionLevel is entered.
func (s *BaseMySqlParserListener) EnterTableOptionPageCompressionLevel(ctx *TableOptionPageCompressionLevelContext) {
}

// ExitTableOptionPageCompressionLevel is called when production tableOptionPageCompressionLevel is exited.
func (s *BaseMySqlParserListener) ExitTableOptionPageCompressionLevel(ctx *TableOptionPageCompressionLevelContext) {
}

// EnterTableOptionEncryptionKeyId is called when production tableOptionEncryptionKeyId is entered.
func (s *BaseMySqlParserListener) EnterTableOptionEncryptionKeyId(ctx *TableOptionEncryptionKeyIdContext) {
}

// ExitTableOptionEncryptionKeyId is called when production tableOptionEncryptionKeyId is exited.
func (s *BaseMySqlParserListener) ExitTableOptionEncryptionKeyId(ctx *TableOptionEncryptionKeyIdContext) {
}

// EnterTableOptionIndexDirectory is called when production tableOptionIndexDirectory is entered.
func (s *BaseMySqlParserListener) EnterTableOptionIndexDirectory(ctx *TableOptionIndexDirectoryContext) {
}

// ExitTableOptionIndexDirectory is called when production tableOptionIndexDirectory is exited.
func (s *BaseMySqlParserListener) ExitTableOptionIndexDirectory(ctx *TableOptionIndexDirectoryContext) {
}

// EnterTableOptionInsertMethod is called when production tableOptionInsertMethod is entered.
func (s *BaseMySqlParserListener) EnterTableOptionInsertMethod(ctx *TableOptionInsertMethodContext) {}

// ExitTableOptionInsertMethod is called when production tableOptionInsertMethod is exited.
func (s *BaseMySqlParserListener) ExitTableOptionInsertMethod(ctx *TableOptionInsertMethodContext) {}

// EnterTableOptionKeyBlockSize is called when production tableOptionKeyBlockSize is entered.
func (s *BaseMySqlParserListener) EnterTableOptionKeyBlockSize(ctx *TableOptionKeyBlockSizeContext) {}

// ExitTableOptionKeyBlockSize is called when production tableOptionKeyBlockSize is exited.
func (s *BaseMySqlParserListener) ExitTableOptionKeyBlockSize(ctx *TableOptionKeyBlockSizeContext) {}

// EnterTableOptionMaxRows is called when production tableOptionMaxRows is entered.
func (s *BaseMySqlParserListener) EnterTableOptionMaxRows(ctx *TableOptionMaxRowsContext) {}

// ExitTableOptionMaxRows is called when production tableOptionMaxRows is exited.
func (s *BaseMySqlParserListener) ExitTableOptionMaxRows(ctx *TableOptionMaxRowsContext) {}

// EnterTableOptionMinRows is called when production tableOptionMinRows is entered.
func (s *BaseMySqlParserListener) EnterTableOptionMinRows(ctx *TableOptionMinRowsContext) {}

// ExitTableOptionMinRows is called when production tableOptionMinRows is exited.
func (s *BaseMySqlParserListener) ExitTableOptionMinRows(ctx *TableOptionMinRowsContext) {}

// EnterTableOptionPackKeys is called when production tableOptionPackKeys is entered.
func (s *BaseMySqlParserListener) EnterTableOptionPackKeys(ctx *TableOptionPackKeysContext) {}

// ExitTableOptionPackKeys is called when production tableOptionPackKeys is exited.
func (s *BaseMySqlParserListener) ExitTableOptionPackKeys(ctx *TableOptionPackKeysContext) {}

// EnterTableOptionPassword is called when production tableOptionPassword is entered.
func (s *BaseMySqlParserListener) EnterTableOptionPassword(ctx *TableOptionPasswordContext) {}

// ExitTableOptionPassword is called when production tableOptionPassword is exited.
func (s *BaseMySqlParserListener) ExitTableOptionPassword(ctx *TableOptionPasswordContext) {}

// EnterTableOptionRowFormat is called when production tableOptionRowFormat is entered.
func (s *BaseMySqlParserListener) EnterTableOptionRowFormat(ctx *TableOptionRowFormatContext) {}

// ExitTableOptionRowFormat is called when production tableOptionRowFormat is exited.
func (s *BaseMySqlParserListener) ExitTableOptionRowFormat(ctx *TableOptionRowFormatContext) {}

// EnterTableOptionStartTransaction is called when production tableOptionStartTransaction is entered.
func (s *BaseMySqlParserListener) EnterTableOptionStartTransaction(ctx *TableOptionStartTransactionContext) {
}

// ExitTableOptionStartTransaction is called when production tableOptionStartTransaction is exited.
func (s *BaseMySqlParserListener) ExitTableOptionStartTransaction(ctx *TableOptionStartTransactionContext) {
}

// EnterTableOptionSecondaryEngineAttribute is called when production tableOptionSecondaryEngineAttribute is entered.
func (s *BaseMySqlParserListener) EnterTableOptionSecondaryEngineAttribute(ctx *TableOptionSecondaryEngineAttributeContext) {
}

// ExitTableOptionSecondaryEngineAttribute is called when production tableOptionSecondaryEngineAttribute is exited.
func (s *BaseMySqlParserListener) ExitTableOptionSecondaryEngineAttribute(ctx *TableOptionSecondaryEngineAttributeContext) {
}

// EnterTableOptionRecalculation is called when production tableOptionRecalculation is entered.
func (s *BaseMySqlParserListener) EnterTableOptionRecalculation(ctx *TableOptionRecalculationContext) {
}

// ExitTableOptionRecalculation is called when production tableOptionRecalculation is exited.
func (s *BaseMySqlParserListener) ExitTableOptionRecalculation(ctx *TableOptionRecalculationContext) {
}

// EnterTableOptionPersistent is called when production tableOptionPersistent is entered.
func (s *BaseMySqlParserListener) EnterTableOptionPersistent(ctx *TableOptionPersistentContext) {}

// ExitTableOptionPersistent is called when production tableOptionPersistent is exited.
func (s *BaseMySqlParserListener) ExitTableOptionPersistent(ctx *TableOptionPersistentContext) {}

// EnterTableOptionSamplePage is called when production tableOptionSamplePage is entered.
func (s *BaseMySqlParserListener) EnterTableOptionSamplePage(ctx *TableOptionSamplePageContext) {}

// ExitTableOptionSamplePage is called when production tableOptionSamplePage is exited.
func (s *BaseMySqlParserListener) ExitTableOptionSamplePage(ctx *TableOptionSamplePageContext) {}

// EnterTableOptionTablespace is called when production tableOptionTablespace is entered.
func (s *BaseMySqlParserListener) EnterTableOptionTablespace(ctx *TableOptionTablespaceContext) {}

// ExitTableOptionTablespace is called when production tableOptionTablespace is exited.
func (s *BaseMySqlParserListener) ExitTableOptionTablespace(ctx *TableOptionTablespaceContext) {}

// EnterTableOptionTableType is called when production tableOptionTableType is entered.
func (s *BaseMySqlParserListener) EnterTableOptionTableType(ctx *TableOptionTableTypeContext) {}

// ExitTableOptionTableType is called when production tableOptionTableType is exited.
func (s *BaseMySqlParserListener) ExitTableOptionTableType(ctx *TableOptionTableTypeContext) {}

// EnterTableOptionTransactional is called when production tableOptionTransactional is entered.
func (s *BaseMySqlParserListener) EnterTableOptionTransactional(ctx *TableOptionTransactionalContext) {
}

// ExitTableOptionTransactional is called when production tableOptionTransactional is exited.
func (s *BaseMySqlParserListener) ExitTableOptionTransactional(ctx *TableOptionTransactionalContext) {
}

// EnterTableOptionUnion is called when production tableOptionUnion is entered.
func (s *BaseMySqlParserListener) EnterTableOptionUnion(ctx *TableOptionUnionContext) {}

// ExitTableOptionUnion is called when production tableOptionUnion is exited.
func (s *BaseMySqlParserListener) ExitTableOptionUnion(ctx *TableOptionUnionContext) {}

// EnterTableType is called when production tableType is entered.
func (s *BaseMySqlParserListener) EnterTableType(ctx *TableTypeContext) {}

// ExitTableType is called when production tableType is exited.
func (s *BaseMySqlParserListener) ExitTableType(ctx *TableTypeContext) {}

// EnterTablespaceStorage is called when production tablespaceStorage is entered.
func (s *BaseMySqlParserListener) EnterTablespaceStorage(ctx *TablespaceStorageContext) {}

// ExitTablespaceStorage is called when production tablespaceStorage is exited.
func (s *BaseMySqlParserListener) ExitTablespaceStorage(ctx *TablespaceStorageContext) {}

// EnterPartitionDefinitions is called when production partitionDefinitions is entered.
func (s *BaseMySqlParserListener) EnterPartitionDefinitions(ctx *PartitionDefinitionsContext) {}

// ExitPartitionDefinitions is called when production partitionDefinitions is exited.
func (s *BaseMySqlParserListener) ExitPartitionDefinitions(ctx *PartitionDefinitionsContext) {}

// EnterPartitionFunctionHash is called when production partitionFunctionHash is entered.
func (s *BaseMySqlParserListener) EnterPartitionFunctionHash(ctx *PartitionFunctionHashContext) {}

// ExitPartitionFunctionHash is called when production partitionFunctionHash is exited.
func (s *BaseMySqlParserListener) ExitPartitionFunctionHash(ctx *PartitionFunctionHashContext) {}

// EnterPartitionFunctionKey is called when production partitionFunctionKey is entered.
func (s *BaseMySqlParserListener) EnterPartitionFunctionKey(ctx *PartitionFunctionKeyContext) {}

// ExitPartitionFunctionKey is called when production partitionFunctionKey is exited.
func (s *BaseMySqlParserListener) ExitPartitionFunctionKey(ctx *PartitionFunctionKeyContext) {}

// EnterPartitionFunctionRange is called when production partitionFunctionRange is entered.
func (s *BaseMySqlParserListener) EnterPartitionFunctionRange(ctx *PartitionFunctionRangeContext) {}

// ExitPartitionFunctionRange is called when production partitionFunctionRange is exited.
func (s *BaseMySqlParserListener) ExitPartitionFunctionRange(ctx *PartitionFunctionRangeContext) {}

// EnterPartitionFunctionList is called when production partitionFunctionList is entered.
func (s *BaseMySqlParserListener) EnterPartitionFunctionList(ctx *PartitionFunctionListContext) {}

// ExitPartitionFunctionList is called when production partitionFunctionList is exited.
func (s *BaseMySqlParserListener) ExitPartitionFunctionList(ctx *PartitionFunctionListContext) {}

// EnterSubPartitionFunctionHash is called when production subPartitionFunctionHash is entered.
func (s *BaseMySqlParserListener) EnterSubPartitionFunctionHash(ctx *SubPartitionFunctionHashContext) {
}

// ExitSubPartitionFunctionHash is called when production subPartitionFunctionHash is exited.
func (s *BaseMySqlParserListener) ExitSubPartitionFunctionHash(ctx *SubPartitionFunctionHashContext) {
}

// EnterSubPartitionFunctionKey is called when production subPartitionFunctionKey is entered.
func (s *BaseMySqlParserListener) EnterSubPartitionFunctionKey(ctx *SubPartitionFunctionKeyContext) {}

// ExitSubPartitionFunctionKey is called when production subPartitionFunctionKey is exited.
func (s *BaseMySqlParserListener) ExitSubPartitionFunctionKey(ctx *SubPartitionFunctionKeyContext) {}

// EnterPartitionComparison is called when production partitionComparison is entered.
func (s *BaseMySqlParserListener) EnterPartitionComparison(ctx *PartitionComparisonContext) {}

// ExitPartitionComparison is called when production partitionComparison is exited.
func (s *BaseMySqlParserListener) ExitPartitionComparison(ctx *PartitionComparisonContext) {}

// EnterPartitionListAtom is called when production partitionListAtom is entered.
func (s *BaseMySqlParserListener) EnterPartitionListAtom(ctx *PartitionListAtomContext) {}

// ExitPartitionListAtom is called when production partitionListAtom is exited.
func (s *BaseMySqlParserListener) ExitPartitionListAtom(ctx *PartitionListAtomContext) {}

// EnterPartitionListVector is called when production partitionListVector is entered.
func (s *BaseMySqlParserListener) EnterPartitionListVector(ctx *PartitionListVectorContext) {}

// ExitPartitionListVector is called when production partitionListVector is exited.
func (s *BaseMySqlParserListener) ExitPartitionListVector(ctx *PartitionListVectorContext) {}

// EnterPartitionSimple is called when production partitionSimple is entered.
func (s *BaseMySqlParserListener) EnterPartitionSimple(ctx *PartitionSimpleContext) {}

// ExitPartitionSimple is called when production partitionSimple is exited.
func (s *BaseMySqlParserListener) ExitPartitionSimple(ctx *PartitionSimpleContext) {}

// EnterPartitionDefinerAtom is called when production partitionDefinerAtom is entered.
func (s *BaseMySqlParserListener) EnterPartitionDefinerAtom(ctx *PartitionDefinerAtomContext) {}

// ExitPartitionDefinerAtom is called when production partitionDefinerAtom is exited.
func (s *BaseMySqlParserListener) ExitPartitionDefinerAtom(ctx *PartitionDefinerAtomContext) {}

// EnterPartitionDefinerVector is called when production partitionDefinerVector is entered.
func (s *BaseMySqlParserListener) EnterPartitionDefinerVector(ctx *PartitionDefinerVectorContext) {}

// ExitPartitionDefinerVector is called when production partitionDefinerVector is exited.
func (s *BaseMySqlParserListener) ExitPartitionDefinerVector(ctx *PartitionDefinerVectorContext) {}

// EnterSubpartitionDefinition is called when production subpartitionDefinition is entered.
func (s *BaseMySqlParserListener) EnterSubpartitionDefinition(ctx *SubpartitionDefinitionContext) {}

// ExitSubpartitionDefinition is called when production subpartitionDefinition is exited.
func (s *BaseMySqlParserListener) ExitSubpartitionDefinition(ctx *SubpartitionDefinitionContext) {}

// EnterPartitionOptionEngine is called when production partitionOptionEngine is entered.
func (s *BaseMySqlParserListener) EnterPartitionOptionEngine(ctx *PartitionOptionEngineContext) {}

// ExitPartitionOptionEngine is called when production partitionOptionEngine is exited.
func (s *BaseMySqlParserListener) ExitPartitionOptionEngine(ctx *PartitionOptionEngineContext) {}

// EnterPartitionOptionComment is called when production partitionOptionComment is entered.
func (s *BaseMySqlParserListener) EnterPartitionOptionComment(ctx *PartitionOptionCommentContext) {}

// ExitPartitionOptionComment is called when production partitionOptionComment is exited.
func (s *BaseMySqlParserListener) ExitPartitionOptionComment(ctx *PartitionOptionCommentContext) {}

// EnterPartitionOptionDataDirectory is called when production partitionOptionDataDirectory is entered.
func (s *BaseMySqlParserListener) EnterPartitionOptionDataDirectory(ctx *PartitionOptionDataDirectoryContext) {
}

// ExitPartitionOptionDataDirectory is called when production partitionOptionDataDirectory is exited.
func (s *BaseMySqlParserListener) ExitPartitionOptionDataDirectory(ctx *PartitionOptionDataDirectoryContext) {
}

// EnterPartitionOptionIndexDirectory is called when production partitionOptionIndexDirectory is entered.
func (s *BaseMySqlParserListener) EnterPartitionOptionIndexDirectory(ctx *PartitionOptionIndexDirectoryContext) {
}

// ExitPartitionOptionIndexDirectory is called when production partitionOptionIndexDirectory is exited.
func (s *BaseMySqlParserListener) ExitPartitionOptionIndexDirectory(ctx *PartitionOptionIndexDirectoryContext) {
}

// EnterPartitionOptionMaxRows is called when production partitionOptionMaxRows is entered.
func (s *BaseMySqlParserListener) EnterPartitionOptionMaxRows(ctx *PartitionOptionMaxRowsContext) {}

// ExitPartitionOptionMaxRows is called when production partitionOptionMaxRows is exited.
func (s *BaseMySqlParserListener) ExitPartitionOptionMaxRows(ctx *PartitionOptionMaxRowsContext) {}

// EnterPartitionOptionMinRows is called when production partitionOptionMinRows is entered.
func (s *BaseMySqlParserListener) EnterPartitionOptionMinRows(ctx *PartitionOptionMinRowsContext) {}

// ExitPartitionOptionMinRows is called when production partitionOptionMinRows is exited.
func (s *BaseMySqlParserListener) ExitPartitionOptionMinRows(ctx *PartitionOptionMinRowsContext) {}

// EnterPartitionOptionTablespace is called when production partitionOptionTablespace is entered.
func (s *BaseMySqlParserListener) EnterPartitionOptionTablespace(ctx *PartitionOptionTablespaceContext) {
}

// ExitPartitionOptionTablespace is called when production partitionOptionTablespace is exited.
func (s *BaseMySqlParserListener) ExitPartitionOptionTablespace(ctx *PartitionOptionTablespaceContext) {
}

// EnterPartitionOptionNodeGroup is called when production partitionOptionNodeGroup is entered.
func (s *BaseMySqlParserListener) EnterPartitionOptionNodeGroup(ctx *PartitionOptionNodeGroupContext) {
}

// ExitPartitionOptionNodeGroup is called when production partitionOptionNodeGroup is exited.
func (s *BaseMySqlParserListener) ExitPartitionOptionNodeGroup(ctx *PartitionOptionNodeGroupContext) {
}

// EnterAlterSimpleDatabase is called when production alterSimpleDatabase is entered.
func (s *BaseMySqlParserListener) EnterAlterSimpleDatabase(ctx *AlterSimpleDatabaseContext) {}

// ExitAlterSimpleDatabase is called when production alterSimpleDatabase is exited.
func (s *BaseMySqlParserListener) ExitAlterSimpleDatabase(ctx *AlterSimpleDatabaseContext) {}

// EnterAlterUpgradeName is called when production alterUpgradeName is entered.
func (s *BaseMySqlParserListener) EnterAlterUpgradeName(ctx *AlterUpgradeNameContext) {}

// ExitAlterUpgradeName is called when production alterUpgradeName is exited.
func (s *BaseMySqlParserListener) ExitAlterUpgradeName(ctx *AlterUpgradeNameContext) {}

// EnterAlterEvent is called when production alterEvent is entered.
func (s *BaseMySqlParserListener) EnterAlterEvent(ctx *AlterEventContext) {}

// ExitAlterEvent is called when production alterEvent is exited.
func (s *BaseMySqlParserListener) ExitAlterEvent(ctx *AlterEventContext) {}

// EnterAlterFunction is called when production alterFunction is entered.
func (s *BaseMySqlParserListener) EnterAlterFunction(ctx *AlterFunctionContext) {}

// ExitAlterFunction is called when production alterFunction is exited.
func (s *BaseMySqlParserListener) ExitAlterFunction(ctx *AlterFunctionContext) {}

// EnterAlterInstance is called when production alterInstance is entered.
func (s *BaseMySqlParserListener) EnterAlterInstance(ctx *AlterInstanceContext) {}

// ExitAlterInstance is called when production alterInstance is exited.
func (s *BaseMySqlParserListener) ExitAlterInstance(ctx *AlterInstanceContext) {}

// EnterAlterLogfileGroup is called when production alterLogfileGroup is entered.
func (s *BaseMySqlParserListener) EnterAlterLogfileGroup(ctx *AlterLogfileGroupContext) {}

// ExitAlterLogfileGroup is called when production alterLogfileGroup is exited.
func (s *BaseMySqlParserListener) ExitAlterLogfileGroup(ctx *AlterLogfileGroupContext) {}

// EnterAlterProcedure is called when production alterProcedure is entered.
func (s *BaseMySqlParserListener) EnterAlterProcedure(ctx *AlterProcedureContext) {}

// ExitAlterProcedure is called when production alterProcedure is exited.
func (s *BaseMySqlParserListener) ExitAlterProcedure(ctx *AlterProcedureContext) {}

// EnterAlterServer is called when production alterServer is entered.
func (s *BaseMySqlParserListener) EnterAlterServer(ctx *AlterServerContext) {}

// ExitAlterServer is called when production alterServer is exited.
func (s *BaseMySqlParserListener) ExitAlterServer(ctx *AlterServerContext) {}

// EnterAlterTable is called when production alterTable is entered.
func (s *BaseMySqlParserListener) EnterAlterTable(ctx *AlterTableContext) {}

// ExitAlterTable is called when production alterTable is exited.
func (s *BaseMySqlParserListener) ExitAlterTable(ctx *AlterTableContext) {}

// EnterAlterTablespace is called when production alterTablespace is entered.
func (s *BaseMySqlParserListener) EnterAlterTablespace(ctx *AlterTablespaceContext) {}

// ExitAlterTablespace is called when production alterTablespace is exited.
func (s *BaseMySqlParserListener) ExitAlterTablespace(ctx *AlterTablespaceContext) {}

// EnterAlterView is called when production alterView is entered.
func (s *BaseMySqlParserListener) EnterAlterView(ctx *AlterViewContext) {}

// ExitAlterView is called when production alterView is exited.
func (s *BaseMySqlParserListener) ExitAlterView(ctx *AlterViewContext) {}

// EnterAlterByTableOption is called when production alterByTableOption is entered.
func (s *BaseMySqlParserListener) EnterAlterByTableOption(ctx *AlterByTableOptionContext) {}

// ExitAlterByTableOption is called when production alterByTableOption is exited.
func (s *BaseMySqlParserListener) ExitAlterByTableOption(ctx *AlterByTableOptionContext) {}

// EnterAlterByAddColumn is called when production alterByAddColumn is entered.
func (s *BaseMySqlParserListener) EnterAlterByAddColumn(ctx *AlterByAddColumnContext) {}

// ExitAlterByAddColumn is called when production alterByAddColumn is exited.
func (s *BaseMySqlParserListener) ExitAlterByAddColumn(ctx *AlterByAddColumnContext) {}

// EnterAlterByAddColumns is called when production alterByAddColumns is entered.
func (s *BaseMySqlParserListener) EnterAlterByAddColumns(ctx *AlterByAddColumnsContext) {}

// ExitAlterByAddColumns is called when production alterByAddColumns is exited.
func (s *BaseMySqlParserListener) ExitAlterByAddColumns(ctx *AlterByAddColumnsContext) {}

// EnterAlterByAddIndex is called when production alterByAddIndex is entered.
func (s *BaseMySqlParserListener) EnterAlterByAddIndex(ctx *AlterByAddIndexContext) {}

// ExitAlterByAddIndex is called when production alterByAddIndex is exited.
func (s *BaseMySqlParserListener) ExitAlterByAddIndex(ctx *AlterByAddIndexContext) {}

// EnterAlterByAddPrimaryKey is called when production alterByAddPrimaryKey is entered.
func (s *BaseMySqlParserListener) EnterAlterByAddPrimaryKey(ctx *AlterByAddPrimaryKeyContext) {}

// ExitAlterByAddPrimaryKey is called when production alterByAddPrimaryKey is exited.
func (s *BaseMySqlParserListener) ExitAlterByAddPrimaryKey(ctx *AlterByAddPrimaryKeyContext) {}

// EnterAlterByAddUniqueKey is called when production alterByAddUniqueKey is entered.
func (s *BaseMySqlParserListener) EnterAlterByAddUniqueKey(ctx *AlterByAddUniqueKeyContext) {}

// ExitAlterByAddUniqueKey is called when production alterByAddUniqueKey is exited.
func (s *BaseMySqlParserListener) ExitAlterByAddUniqueKey(ctx *AlterByAddUniqueKeyContext) {}

// EnterAlterByAddSpecialIndex is called when production alterByAddSpecialIndex is entered.
func (s *BaseMySqlParserListener) EnterAlterByAddSpecialIndex(ctx *AlterByAddSpecialIndexContext) {}

// ExitAlterByAddSpecialIndex is called when production alterByAddSpecialIndex is exited.
func (s *BaseMySqlParserListener) ExitAlterByAddSpecialIndex(ctx *AlterByAddSpecialIndexContext) {}

// EnterAlterByAddForeignKey is called when production alterByAddForeignKey is entered.
func (s *BaseMySqlParserListener) EnterAlterByAddForeignKey(ctx *AlterByAddForeignKeyContext) {}

// ExitAlterByAddForeignKey is called when production alterByAddForeignKey is exited.
func (s *BaseMySqlParserListener) ExitAlterByAddForeignKey(ctx *AlterByAddForeignKeyContext) {}

// EnterAlterByAddCheckTableConstraint is called when production alterByAddCheckTableConstraint is entered.
func (s *BaseMySqlParserListener) EnterAlterByAddCheckTableConstraint(ctx *AlterByAddCheckTableConstraintContext) {
}

// ExitAlterByAddCheckTableConstraint is called when production alterByAddCheckTableConstraint is exited.
func (s *BaseMySqlParserListener) ExitAlterByAddCheckTableConstraint(ctx *AlterByAddCheckTableConstraintContext) {
}

// EnterAlterByAlterCheckTableConstraint is called when production alterByAlterCheckTableConstraint is entered.
func (s *BaseMySqlParserListener) EnterAlterByAlterCheckTableConstraint(ctx *AlterByAlterCheckTableConstraintContext) {
}

// ExitAlterByAlterCheckTableConstraint is called when production alterByAlterCheckTableConstraint is exited.
func (s *BaseMySqlParserListener) ExitAlterByAlterCheckTableConstraint(ctx *AlterByAlterCheckTableConstraintContext) {
}

// EnterAlterBySetAlgorithm is called when production alterBySetAlgorithm is entered.
func (s *BaseMySqlParserListener) EnterAlterBySetAlgorithm(ctx *AlterBySetAlgorithmContext) {}

// ExitAlterBySetAlgorithm is called when production alterBySetAlgorithm is exited.
func (s *BaseMySqlParserListener) ExitAlterBySetAlgorithm(ctx *AlterBySetAlgorithmContext) {}

// EnterAlterByChangeDefault is called when production alterByChangeDefault is entered.
func (s *BaseMySqlParserListener) EnterAlterByChangeDefault(ctx *AlterByChangeDefaultContext) {}

// ExitAlterByChangeDefault is called when production alterByChangeDefault is exited.
func (s *BaseMySqlParserListener) ExitAlterByChangeDefault(ctx *AlterByChangeDefaultContext) {}

// EnterAlterByChangeColumn is called when production alterByChangeColumn is entered.
func (s *BaseMySqlParserListener) EnterAlterByChangeColumn(ctx *AlterByChangeColumnContext) {}

// ExitAlterByChangeColumn is called when production alterByChangeColumn is exited.
func (s *BaseMySqlParserListener) ExitAlterByChangeColumn(ctx *AlterByChangeColumnContext) {}

// EnterAlterByRenameColumn is called when production alterByRenameColumn is entered.
func (s *BaseMySqlParserListener) EnterAlterByRenameColumn(ctx *AlterByRenameColumnContext) {}

// ExitAlterByRenameColumn is called when production alterByRenameColumn is exited.
func (s *BaseMySqlParserListener) ExitAlterByRenameColumn(ctx *AlterByRenameColumnContext) {}

// EnterAlterByLock is called when production alterByLock is entered.
func (s *BaseMySqlParserListener) EnterAlterByLock(ctx *AlterByLockContext) {}

// ExitAlterByLock is called when production alterByLock is exited.
func (s *BaseMySqlParserListener) ExitAlterByLock(ctx *AlterByLockContext) {}

// EnterAlterByModifyColumn is called when production alterByModifyColumn is entered.
func (s *BaseMySqlParserListener) EnterAlterByModifyColumn(ctx *AlterByModifyColumnContext) {}

// ExitAlterByModifyColumn is called when production alterByModifyColumn is exited.
func (s *BaseMySqlParserListener) ExitAlterByModifyColumn(ctx *AlterByModifyColumnContext) {}

// EnterAlterByDropColumn is called when production alterByDropColumn is entered.
func (s *BaseMySqlParserListener) EnterAlterByDropColumn(ctx *AlterByDropColumnContext) {}

// ExitAlterByDropColumn is called when production alterByDropColumn is exited.
func (s *BaseMySqlParserListener) ExitAlterByDropColumn(ctx *AlterByDropColumnContext) {}

// EnterAlterByDropConstraintCheck is called when production alterByDropConstraintCheck is entered.
func (s *BaseMySqlParserListener) EnterAlterByDropConstraintCheck(ctx *AlterByDropConstraintCheckContext) {
}

// ExitAlterByDropConstraintCheck is called when production alterByDropConstraintCheck is exited.
func (s *BaseMySqlParserListener) ExitAlterByDropConstraintCheck(ctx *AlterByDropConstraintCheckContext) {
}

// EnterAlterByDropPrimaryKey is called when production alterByDropPrimaryKey is entered.
func (s *BaseMySqlParserListener) EnterAlterByDropPrimaryKey(ctx *AlterByDropPrimaryKeyContext) {}

// ExitAlterByDropPrimaryKey is called when production alterByDropPrimaryKey is exited.
func (s *BaseMySqlParserListener) ExitAlterByDropPrimaryKey(ctx *AlterByDropPrimaryKeyContext) {}

// EnterAlterByDropIndex is called when production alterByDropIndex is entered.
func (s *BaseMySqlParserListener) EnterAlterByDropIndex(ctx *AlterByDropIndexContext) {}

// ExitAlterByDropIndex is called when production alterByDropIndex is exited.
func (s *BaseMySqlParserListener) ExitAlterByDropIndex(ctx *AlterByDropIndexContext) {}

// EnterAlterByRenameIndex is called when production alterByRenameIndex is entered.
func (s *BaseMySqlParserListener) EnterAlterByRenameIndex(ctx *AlterByRenameIndexContext) {}

// ExitAlterByRenameIndex is called when production alterByRenameIndex is exited.
func (s *BaseMySqlParserListener) ExitAlterByRenameIndex(ctx *AlterByRenameIndexContext) {}

// EnterAlterByAlterColumnDefault is called when production alterByAlterColumnDefault is entered.
func (s *BaseMySqlParserListener) EnterAlterByAlterColumnDefault(ctx *AlterByAlterColumnDefaultContext) {
}

// ExitAlterByAlterColumnDefault is called when production alterByAlterColumnDefault is exited.
func (s *BaseMySqlParserListener) ExitAlterByAlterColumnDefault(ctx *AlterByAlterColumnDefaultContext) {
}

// EnterAlterByAlterIndexVisibility is called when production alterByAlterIndexVisibility is entered.
func (s *BaseMySqlParserListener) EnterAlterByAlterIndexVisibility(ctx *AlterByAlterIndexVisibilityContext) {
}

// ExitAlterByAlterIndexVisibility is called when production alterByAlterIndexVisibility is exited.
func (s *BaseMySqlParserListener) ExitAlterByAlterIndexVisibility(ctx *AlterByAlterIndexVisibilityContext) {
}

// EnterAlterByDropForeignKey is called when production alterByDropForeignKey is entered.
func (s *BaseMySqlParserListener) EnterAlterByDropForeignKey(ctx *AlterByDropForeignKeyContext) {}

// ExitAlterByDropForeignKey is called when production alterByDropForeignKey is exited.
func (s *BaseMySqlParserListener) ExitAlterByDropForeignKey(ctx *AlterByDropForeignKeyContext) {}

// EnterAlterByDisableKeys is called when production alterByDisableKeys is entered.
func (s *BaseMySqlParserListener) EnterAlterByDisableKeys(ctx *AlterByDisableKeysContext) {}

// ExitAlterByDisableKeys is called when production alterByDisableKeys is exited.
func (s *BaseMySqlParserListener) ExitAlterByDisableKeys(ctx *AlterByDisableKeysContext) {}

// EnterAlterByEnableKeys is called when production alterByEnableKeys is entered.
func (s *BaseMySqlParserListener) EnterAlterByEnableKeys(ctx *AlterByEnableKeysContext) {}

// ExitAlterByEnableKeys is called when production alterByEnableKeys is exited.
func (s *BaseMySqlParserListener) ExitAlterByEnableKeys(ctx *AlterByEnableKeysContext) {}

// EnterAlterByRename is called when production alterByRename is entered.
func (s *BaseMySqlParserListener) EnterAlterByRename(ctx *AlterByRenameContext) {}

// ExitAlterByRename is called when production alterByRename is exited.
func (s *BaseMySqlParserListener) ExitAlterByRename(ctx *AlterByRenameContext) {}

// EnterAlterByOrder is called when production alterByOrder is entered.
func (s *BaseMySqlParserListener) EnterAlterByOrder(ctx *AlterByOrderContext) {}

// ExitAlterByOrder is called when production alterByOrder is exited.
func (s *BaseMySqlParserListener) ExitAlterByOrder(ctx *AlterByOrderContext) {}

// EnterAlterByConvertCharset is called when production alterByConvertCharset is entered.
func (s *BaseMySqlParserListener) EnterAlterByConvertCharset(ctx *AlterByConvertCharsetContext) {}

// ExitAlterByConvertCharset is called when production alterByConvertCharset is exited.
func (s *BaseMySqlParserListener) ExitAlterByConvertCharset(ctx *AlterByConvertCharsetContext) {}

// EnterAlterByDefaultCharset is called when production alterByDefaultCharset is entered.
func (s *BaseMySqlParserListener) EnterAlterByDefaultCharset(ctx *AlterByDefaultCharsetContext) {}

// ExitAlterByDefaultCharset is called when production alterByDefaultCharset is exited.
func (s *BaseMySqlParserListener) ExitAlterByDefaultCharset(ctx *AlterByDefaultCharsetContext) {}

// EnterAlterByDiscardTablespace is called when production alterByDiscardTablespace is entered.
func (s *BaseMySqlParserListener) EnterAlterByDiscardTablespace(ctx *AlterByDiscardTablespaceContext) {
}

// ExitAlterByDiscardTablespace is called when production alterByDiscardTablespace is exited.
func (s *BaseMySqlParserListener) ExitAlterByDiscardTablespace(ctx *AlterByDiscardTablespaceContext) {
}

// EnterAlterByImportTablespace is called when production alterByImportTablespace is entered.
func (s *BaseMySqlParserListener) EnterAlterByImportTablespace(ctx *AlterByImportTablespaceContext) {}

// ExitAlterByImportTablespace is called when production alterByImportTablespace is exited.
func (s *BaseMySqlParserListener) ExitAlterByImportTablespace(ctx *AlterByImportTablespaceContext) {}

// EnterAlterByForce is called when production alterByForce is entered.
func (s *BaseMySqlParserListener) EnterAlterByForce(ctx *AlterByForceContext) {}

// ExitAlterByForce is called when production alterByForce is exited.
func (s *BaseMySqlParserListener) ExitAlterByForce(ctx *AlterByForceContext) {}

// EnterAlterByValidate is called when production alterByValidate is entered.
func (s *BaseMySqlParserListener) EnterAlterByValidate(ctx *AlterByValidateContext) {}

// ExitAlterByValidate is called when production alterByValidate is exited.
func (s *BaseMySqlParserListener) ExitAlterByValidate(ctx *AlterByValidateContext) {}

// EnterAlterByAddDefinitions is called when production alterByAddDefinitions is entered.
func (s *BaseMySqlParserListener) EnterAlterByAddDefinitions(ctx *AlterByAddDefinitionsContext) {}

// ExitAlterByAddDefinitions is called when production alterByAddDefinitions is exited.
func (s *BaseMySqlParserListener) ExitAlterByAddDefinitions(ctx *AlterByAddDefinitionsContext) {}

// EnterAlterPartition is called when production alterPartition is entered.
func (s *BaseMySqlParserListener) EnterAlterPartition(ctx *AlterPartitionContext) {}

// ExitAlterPartition is called when production alterPartition is exited.
func (s *BaseMySqlParserListener) ExitAlterPartition(ctx *AlterPartitionContext) {}

// EnterAlterByAddPartition is called when production alterByAddPartition is entered.
func (s *BaseMySqlParserListener) EnterAlterByAddPartition(ctx *AlterByAddPartitionContext) {}

// ExitAlterByAddPartition is called when production alterByAddPartition is exited.
func (s *BaseMySqlParserListener) ExitAlterByAddPartition(ctx *AlterByAddPartitionContext) {}

// EnterAlterByDropPartition is called when production alterByDropPartition is entered.
func (s *BaseMySqlParserListener) EnterAlterByDropPartition(ctx *AlterByDropPartitionContext) {}

// ExitAlterByDropPartition is called when production alterByDropPartition is exited.
func (s *BaseMySqlParserListener) ExitAlterByDropPartition(ctx *AlterByDropPartitionContext) {}

// EnterAlterByDiscardPartition is called when production alterByDiscardPartition is entered.
func (s *BaseMySqlParserListener) EnterAlterByDiscardPartition(ctx *AlterByDiscardPartitionContext) {}

// ExitAlterByDiscardPartition is called when production alterByDiscardPartition is exited.
func (s *BaseMySqlParserListener) ExitAlterByDiscardPartition(ctx *AlterByDiscardPartitionContext) {}

// EnterAlterByImportPartition is called when production alterByImportPartition is entered.
func (s *BaseMySqlParserListener) EnterAlterByImportPartition(ctx *AlterByImportPartitionContext) {}

// ExitAlterByImportPartition is called when production alterByImportPartition is exited.
func (s *BaseMySqlParserListener) ExitAlterByImportPartition(ctx *AlterByImportPartitionContext) {}

// EnterAlterByTruncatePartition is called when production alterByTruncatePartition is entered.
func (s *BaseMySqlParserListener) EnterAlterByTruncatePartition(ctx *AlterByTruncatePartitionContext) {
}

// ExitAlterByTruncatePartition is called when production alterByTruncatePartition is exited.
func (s *BaseMySqlParserListener) ExitAlterByTruncatePartition(ctx *AlterByTruncatePartitionContext) {
}

// EnterAlterByCoalescePartition is called when production alterByCoalescePartition is entered.
func (s *BaseMySqlParserListener) EnterAlterByCoalescePartition(ctx *AlterByCoalescePartitionContext) {
}

// ExitAlterByCoalescePartition is called when production alterByCoalescePartition is exited.
func (s *BaseMySqlParserListener) ExitAlterByCoalescePartition(ctx *AlterByCoalescePartitionContext) {
}

// EnterAlterByReorganizePartition is called when production alterByReorganizePartition is entered.
func (s *BaseMySqlParserListener) EnterAlterByReorganizePartition(ctx *AlterByReorganizePartitionContext) {
}

// ExitAlterByReorganizePartition is called when production alterByReorganizePartition is exited.
func (s *BaseMySqlParserListener) ExitAlterByReorganizePartition(ctx *AlterByReorganizePartitionContext) {
}

// EnterAlterByExchangePartition is called when production alterByExchangePartition is entered.
func (s *BaseMySqlParserListener) EnterAlterByExchangePartition(ctx *AlterByExchangePartitionContext) {
}

// ExitAlterByExchangePartition is called when production alterByExchangePartition is exited.
func (s *BaseMySqlParserListener) ExitAlterByExchangePartition(ctx *AlterByExchangePartitionContext) {
}

// EnterAlterByAnalyzePartition is called when production alterByAnalyzePartition is entered.
func (s *BaseMySqlParserListener) EnterAlterByAnalyzePartition(ctx *AlterByAnalyzePartitionContext) {}

// ExitAlterByAnalyzePartition is called when production alterByAnalyzePartition is exited.
func (s *BaseMySqlParserListener) ExitAlterByAnalyzePartition(ctx *AlterByAnalyzePartitionContext) {}

// EnterAlterByCheckPartition is called when production alterByCheckPartition is entered.
func (s *BaseMySqlParserListener) EnterAlterByCheckPartition(ctx *AlterByCheckPartitionContext) {}

// ExitAlterByCheckPartition is called when production alterByCheckPartition is exited.
func (s *BaseMySqlParserListener) ExitAlterByCheckPartition(ctx *AlterByCheckPartitionContext) {}

// EnterAlterByOptimizePartition is called when production alterByOptimizePartition is entered.
func (s *BaseMySqlParserListener) EnterAlterByOptimizePartition(ctx *AlterByOptimizePartitionContext) {
}

// ExitAlterByOptimizePartition is called when production alterByOptimizePartition is exited.
func (s *BaseMySqlParserListener) ExitAlterByOptimizePartition(ctx *AlterByOptimizePartitionContext) {
}

// EnterAlterByRebuildPartition is called when production alterByRebuildPartition is entered.
func (s *BaseMySqlParserListener) EnterAlterByRebuildPartition(ctx *AlterByRebuildPartitionContext) {}

// ExitAlterByRebuildPartition is called when production alterByRebuildPartition is exited.
func (s *BaseMySqlParserListener) ExitAlterByRebuildPartition(ctx *AlterByRebuildPartitionContext) {}

// EnterAlterByRepairPartition is called when production alterByRepairPartition is entered.
func (s *BaseMySqlParserListener) EnterAlterByRepairPartition(ctx *AlterByRepairPartitionContext) {}

// ExitAlterByRepairPartition is called when production alterByRepairPartition is exited.
func (s *BaseMySqlParserListener) ExitAlterByRepairPartition(ctx *AlterByRepairPartitionContext) {}

// EnterAlterByRemovePartitioning is called when production alterByRemovePartitioning is entered.
func (s *BaseMySqlParserListener) EnterAlterByRemovePartitioning(ctx *AlterByRemovePartitioningContext) {
}

// ExitAlterByRemovePartitioning is called when production alterByRemovePartitioning is exited.
func (s *BaseMySqlParserListener) ExitAlterByRemovePartitioning(ctx *AlterByRemovePartitioningContext) {
}

// EnterAlterByUpgradePartitioning is called when production alterByUpgradePartitioning is entered.
func (s *BaseMySqlParserListener) EnterAlterByUpgradePartitioning(ctx *AlterByUpgradePartitioningContext) {
}

// ExitAlterByUpgradePartitioning is called when production alterByUpgradePartitioning is exited.
func (s *BaseMySqlParserListener) ExitAlterByUpgradePartitioning(ctx *AlterByUpgradePartitioningContext) {
}

// EnterDropDatabase is called when production dropDatabase is entered.
func (s *BaseMySqlParserListener) EnterDropDatabase(ctx *DropDatabaseContext) {}

// ExitDropDatabase is called when production dropDatabase is exited.
func (s *BaseMySqlParserListener) ExitDropDatabase(ctx *DropDatabaseContext) {}

// EnterDropEvent is called when production dropEvent is entered.
func (s *BaseMySqlParserListener) EnterDropEvent(ctx *DropEventContext) {}

// ExitDropEvent is called when production dropEvent is exited.
func (s *BaseMySqlParserListener) ExitDropEvent(ctx *DropEventContext) {}

// EnterDropIndex is called when production dropIndex is entered.
func (s *BaseMySqlParserListener) EnterDropIndex(ctx *DropIndexContext) {}

// ExitDropIndex is called when production dropIndex is exited.
func (s *BaseMySqlParserListener) ExitDropIndex(ctx *DropIndexContext) {}

// EnterDropLogfileGroup is called when production dropLogfileGroup is entered.
func (s *BaseMySqlParserListener) EnterDropLogfileGroup(ctx *DropLogfileGroupContext) {}

// ExitDropLogfileGroup is called when production dropLogfileGroup is exited.
func (s *BaseMySqlParserListener) ExitDropLogfileGroup(ctx *DropLogfileGroupContext) {}

// EnterDropProcedure is called when production dropProcedure is entered.
func (s *BaseMySqlParserListener) EnterDropProcedure(ctx *DropProcedureContext) {}

// ExitDropProcedure is called when production dropProcedure is exited.
func (s *BaseMySqlParserListener) ExitDropProcedure(ctx *DropProcedureContext) {}

// EnterDropFunction is called when production dropFunction is entered.
func (s *BaseMySqlParserListener) EnterDropFunction(ctx *DropFunctionContext) {}

// ExitDropFunction is called when production dropFunction is exited.
func (s *BaseMySqlParserListener) ExitDropFunction(ctx *DropFunctionContext) {}

// EnterDropServer is called when production dropServer is entered.
func (s *BaseMySqlParserListener) EnterDropServer(ctx *DropServerContext) {}

// ExitDropServer is called when production dropServer is exited.
func (s *BaseMySqlParserListener) ExitDropServer(ctx *DropServerContext) {}

// EnterDropTable is called when production dropTable is entered.
func (s *BaseMySqlParserListener) EnterDropTable(ctx *DropTableContext) {}

// ExitDropTable is called when production dropTable is exited.
func (s *BaseMySqlParserListener) ExitDropTable(ctx *DropTableContext) {}

// EnterDropTablespace is called when production dropTablespace is entered.
func (s *BaseMySqlParserListener) EnterDropTablespace(ctx *DropTablespaceContext) {}

// ExitDropTablespace is called when production dropTablespace is exited.
func (s *BaseMySqlParserListener) ExitDropTablespace(ctx *DropTablespaceContext) {}

// EnterDropTrigger is called when production dropTrigger is entered.
func (s *BaseMySqlParserListener) EnterDropTrigger(ctx *DropTriggerContext) {}

// ExitDropTrigger is called when production dropTrigger is exited.
func (s *BaseMySqlParserListener) ExitDropTrigger(ctx *DropTriggerContext) {}

// EnterDropView is called when production dropView is entered.
func (s *BaseMySqlParserListener) EnterDropView(ctx *DropViewContext) {}

// ExitDropView is called when production dropView is exited.
func (s *BaseMySqlParserListener) ExitDropView(ctx *DropViewContext) {}

// EnterDropRole is called when production dropRole is entered.
func (s *BaseMySqlParserListener) EnterDropRole(ctx *DropRoleContext) {}

// ExitDropRole is called when production dropRole is exited.
func (s *BaseMySqlParserListener) ExitDropRole(ctx *DropRoleContext) {}

// EnterSetRole is called when production setRole is entered.
func (s *BaseMySqlParserListener) EnterSetRole(ctx *SetRoleContext) {}

// ExitSetRole is called when production setRole is exited.
func (s *BaseMySqlParserListener) ExitSetRole(ctx *SetRoleContext) {}

// EnterRenameTable is called when production renameTable is entered.
func (s *BaseMySqlParserListener) EnterRenameTable(ctx *RenameTableContext) {}

// ExitRenameTable is called when production renameTable is exited.
func (s *BaseMySqlParserListener) ExitRenameTable(ctx *RenameTableContext) {}

// EnterRenameTableClause is called when production renameTableClause is entered.
func (s *BaseMySqlParserListener) EnterRenameTableClause(ctx *RenameTableClauseContext) {}

// ExitRenameTableClause is called when production renameTableClause is exited.
func (s *BaseMySqlParserListener) ExitRenameTableClause(ctx *RenameTableClauseContext) {}

// EnterTruncateTable is called when production truncateTable is entered.
func (s *BaseMySqlParserListener) EnterTruncateTable(ctx *TruncateTableContext) {}

// ExitTruncateTable is called when production truncateTable is exited.
func (s *BaseMySqlParserListener) ExitTruncateTable(ctx *TruncateTableContext) {}

// EnterCallStatement is called when production callStatement is entered.
func (s *BaseMySqlParserListener) EnterCallStatement(ctx *CallStatementContext) {}

// ExitCallStatement is called when production callStatement is exited.
func (s *BaseMySqlParserListener) ExitCallStatement(ctx *CallStatementContext) {}

// EnterDeleteStatement is called when production deleteStatement is entered.
func (s *BaseMySqlParserListener) EnterDeleteStatement(ctx *DeleteStatementContext) {}

// ExitDeleteStatement is called when production deleteStatement is exited.
func (s *BaseMySqlParserListener) ExitDeleteStatement(ctx *DeleteStatementContext) {}

// EnterDoStatement is called when production doStatement is entered.
func (s *BaseMySqlParserListener) EnterDoStatement(ctx *DoStatementContext) {}

// ExitDoStatement is called when production doStatement is exited.
func (s *BaseMySqlParserListener) ExitDoStatement(ctx *DoStatementContext) {}

// EnterHandlerStatement is called when production handlerStatement is entered.
func (s *BaseMySqlParserListener) EnterHandlerStatement(ctx *HandlerStatementContext) {}

// ExitHandlerStatement is called when production handlerStatement is exited.
func (s *BaseMySqlParserListener) ExitHandlerStatement(ctx *HandlerStatementContext) {}

// EnterInsertStatement is called when production insertStatement is entered.
func (s *BaseMySqlParserListener) EnterInsertStatement(ctx *InsertStatementContext) {}

// ExitInsertStatement is called when production insertStatement is exited.
func (s *BaseMySqlParserListener) ExitInsertStatement(ctx *InsertStatementContext) {}

// EnterLoadDataStatement is called when production loadDataStatement is entered.
func (s *BaseMySqlParserListener) EnterLoadDataStatement(ctx *LoadDataStatementContext) {}

// ExitLoadDataStatement is called when production loadDataStatement is exited.
func (s *BaseMySqlParserListener) ExitLoadDataStatement(ctx *LoadDataStatementContext) {}

// EnterLoadXmlStatement is called when production loadXmlStatement is entered.
func (s *BaseMySqlParserListener) EnterLoadXmlStatement(ctx *LoadXmlStatementContext) {}

// ExitLoadXmlStatement is called when production loadXmlStatement is exited.
func (s *BaseMySqlParserListener) ExitLoadXmlStatement(ctx *LoadXmlStatementContext) {}

// EnterReplaceStatement is called when production replaceStatement is entered.
func (s *BaseMySqlParserListener) EnterReplaceStatement(ctx *ReplaceStatementContext) {}

// ExitReplaceStatement is called when production replaceStatement is exited.
func (s *BaseMySqlParserListener) ExitReplaceStatement(ctx *ReplaceStatementContext) {}

// EnterSimpleSelect is called when production simpleSelect is entered.
func (s *BaseMySqlParserListener) EnterSimpleSelect(ctx *SimpleSelectContext) {}

// ExitSimpleSelect is called when production simpleSelect is exited.
func (s *BaseMySqlParserListener) ExitSimpleSelect(ctx *SimpleSelectContext) {}

// EnterParenthesisSelect is called when production parenthesisSelect is entered.
func (s *BaseMySqlParserListener) EnterParenthesisSelect(ctx *ParenthesisSelectContext) {}

// ExitParenthesisSelect is called when production parenthesisSelect is exited.
func (s *BaseMySqlParserListener) ExitParenthesisSelect(ctx *ParenthesisSelectContext) {}

// EnterUnionSelect is called when production unionSelect is entered.
func (s *BaseMySqlParserListener) EnterUnionSelect(ctx *UnionSelectContext) {}

// ExitUnionSelect is called when production unionSelect is exited.
func (s *BaseMySqlParserListener) ExitUnionSelect(ctx *UnionSelectContext) {}

// EnterUnionParenthesisSelect is called when production unionParenthesisSelect is entered.
func (s *BaseMySqlParserListener) EnterUnionParenthesisSelect(ctx *UnionParenthesisSelectContext) {}

// ExitUnionParenthesisSelect is called when production unionParenthesisSelect is exited.
func (s *BaseMySqlParserListener) ExitUnionParenthesisSelect(ctx *UnionParenthesisSelectContext) {}

// EnterWithLateralStatement is called when production withLateralStatement is entered.
func (s *BaseMySqlParserListener) EnterWithLateralStatement(ctx *WithLateralStatementContext) {}

// ExitWithLateralStatement is called when production withLateralStatement is exited.
func (s *BaseMySqlParserListener) ExitWithLateralStatement(ctx *WithLateralStatementContext) {}

// EnterUpdateStatement is called when production updateStatement is entered.
func (s *BaseMySqlParserListener) EnterUpdateStatement(ctx *UpdateStatementContext) {}

// ExitUpdateStatement is called when production updateStatement is exited.
func (s *BaseMySqlParserListener) ExitUpdateStatement(ctx *UpdateStatementContext) {}

// EnterValuesStatement is called when production valuesStatement is entered.
func (s *BaseMySqlParserListener) EnterValuesStatement(ctx *ValuesStatementContext) {}

// ExitValuesStatement is called when production valuesStatement is exited.
func (s *BaseMySqlParserListener) ExitValuesStatement(ctx *ValuesStatementContext) {}

// EnterInsertStatementValue is called when production insertStatementValue is entered.
func (s *BaseMySqlParserListener) EnterInsertStatementValue(ctx *InsertStatementValueContext) {}

// ExitInsertStatementValue is called when production insertStatementValue is exited.
func (s *BaseMySqlParserListener) ExitInsertStatementValue(ctx *InsertStatementValueContext) {}

// EnterUpdatedElement is called when production updatedElement is entered.
func (s *BaseMySqlParserListener) EnterUpdatedElement(ctx *UpdatedElementContext) {}

// ExitUpdatedElement is called when production updatedElement is exited.
func (s *BaseMySqlParserListener) ExitUpdatedElement(ctx *UpdatedElementContext) {}

// EnterAssignmentField is called when production assignmentField is entered.
func (s *BaseMySqlParserListener) EnterAssignmentField(ctx *AssignmentFieldContext) {}

// ExitAssignmentField is called when production assignmentField is exited.
func (s *BaseMySqlParserListener) ExitAssignmentField(ctx *AssignmentFieldContext) {}

// EnterLockClause is called when production lockClause is entered.
func (s *BaseMySqlParserListener) EnterLockClause(ctx *LockClauseContext) {}

// ExitLockClause is called when production lockClause is exited.
func (s *BaseMySqlParserListener) ExitLockClause(ctx *LockClauseContext) {}

// EnterSingleDeleteStatement is called when production singleDeleteStatement is entered.
func (s *BaseMySqlParserListener) EnterSingleDeleteStatement(ctx *SingleDeleteStatementContext) {}

// ExitSingleDeleteStatement is called when production singleDeleteStatement is exited.
func (s *BaseMySqlParserListener) ExitSingleDeleteStatement(ctx *SingleDeleteStatementContext) {}

// EnterMultipleDeleteStatement is called when production multipleDeleteStatement is entered.
func (s *BaseMySqlParserListener) EnterMultipleDeleteStatement(ctx *MultipleDeleteStatementContext) {}

// ExitMultipleDeleteStatement is called when production multipleDeleteStatement is exited.
func (s *BaseMySqlParserListener) ExitMultipleDeleteStatement(ctx *MultipleDeleteStatementContext) {}

// EnterHandlerOpenStatement is called when production handlerOpenStatement is entered.
func (s *BaseMySqlParserListener) EnterHandlerOpenStatement(ctx *HandlerOpenStatementContext) {}

// ExitHandlerOpenStatement is called when production handlerOpenStatement is exited.
func (s *BaseMySqlParserListener) ExitHandlerOpenStatement(ctx *HandlerOpenStatementContext) {}

// EnterHandlerReadIndexStatement is called when production handlerReadIndexStatement is entered.
func (s *BaseMySqlParserListener) EnterHandlerReadIndexStatement(ctx *HandlerReadIndexStatementContext) {
}

// ExitHandlerReadIndexStatement is called when production handlerReadIndexStatement is exited.
func (s *BaseMySqlParserListener) ExitHandlerReadIndexStatement(ctx *HandlerReadIndexStatementContext) {
}

// EnterHandlerReadStatement is called when production handlerReadStatement is entered.
func (s *BaseMySqlParserListener) EnterHandlerReadStatement(ctx *HandlerReadStatementContext) {}

// ExitHandlerReadStatement is called when production handlerReadStatement is exited.
func (s *BaseMySqlParserListener) ExitHandlerReadStatement(ctx *HandlerReadStatementContext) {}

// EnterHandlerCloseStatement is called when production handlerCloseStatement is entered.
func (s *BaseMySqlParserListener) EnterHandlerCloseStatement(ctx *HandlerCloseStatementContext) {}

// ExitHandlerCloseStatement is called when production handlerCloseStatement is exited.
func (s *BaseMySqlParserListener) ExitHandlerCloseStatement(ctx *HandlerCloseStatementContext) {}

// EnterSingleUpdateStatement is called when production singleUpdateStatement is entered.
func (s *BaseMySqlParserListener) EnterSingleUpdateStatement(ctx *SingleUpdateStatementContext) {}

// ExitSingleUpdateStatement is called when production singleUpdateStatement is exited.
func (s *BaseMySqlParserListener) ExitSingleUpdateStatement(ctx *SingleUpdateStatementContext) {}

// EnterMultipleUpdateStatement is called when production multipleUpdateStatement is entered.
func (s *BaseMySqlParserListener) EnterMultipleUpdateStatement(ctx *MultipleUpdateStatementContext) {}

// ExitMultipleUpdateStatement is called when production multipleUpdateStatement is exited.
func (s *BaseMySqlParserListener) ExitMultipleUpdateStatement(ctx *MultipleUpdateStatementContext) {}

// EnterOrderByClause is called when production orderByClause is entered.
func (s *BaseMySqlParserListener) EnterOrderByClause(ctx *OrderByClauseContext) {}

// ExitOrderByClause is called when production orderByClause is exited.
func (s *BaseMySqlParserListener) ExitOrderByClause(ctx *OrderByClauseContext) {}

// EnterOrderByExpression is called when production orderByExpression is entered.
func (s *BaseMySqlParserListener) EnterOrderByExpression(ctx *OrderByExpressionContext) {}

// ExitOrderByExpression is called when production orderByExpression is exited.
func (s *BaseMySqlParserListener) ExitOrderByExpression(ctx *OrderByExpressionContext) {}

// EnterTableSources is called when production tableSources is entered.
func (s *BaseMySqlParserListener) EnterTableSources(ctx *TableSourcesContext) {}

// ExitTableSources is called when production tableSources is exited.
func (s *BaseMySqlParserListener) ExitTableSources(ctx *TableSourcesContext) {}

// EnterTableSourceBase is called when production tableSourceBase is entered.
func (s *BaseMySqlParserListener) EnterTableSourceBase(ctx *TableSourceBaseContext) {}

// ExitTableSourceBase is called when production tableSourceBase is exited.
func (s *BaseMySqlParserListener) ExitTableSourceBase(ctx *TableSourceBaseContext) {}

// EnterTableSourceNested is called when production tableSourceNested is entered.
func (s *BaseMySqlParserListener) EnterTableSourceNested(ctx *TableSourceNestedContext) {}

// ExitTableSourceNested is called when production tableSourceNested is exited.
func (s *BaseMySqlParserListener) ExitTableSourceNested(ctx *TableSourceNestedContext) {}

// EnterTableJson is called when production tableJson is entered.
func (s *BaseMySqlParserListener) EnterTableJson(ctx *TableJsonContext) {}

// ExitTableJson is called when production tableJson is exited.
func (s *BaseMySqlParserListener) ExitTableJson(ctx *TableJsonContext) {}

// EnterAtomTableItem is called when production atomTableItem is entered.
func (s *BaseMySqlParserListener) EnterAtomTableItem(ctx *AtomTableItemContext) {}

// ExitAtomTableItem is called when production atomTableItem is exited.
func (s *BaseMySqlParserListener) ExitAtomTableItem(ctx *AtomTableItemContext) {}

// EnterSubqueryTableItem is called when production subqueryTableItem is entered.
func (s *BaseMySqlParserListener) EnterSubqueryTableItem(ctx *SubqueryTableItemContext) {}

// ExitSubqueryTableItem is called when production subqueryTableItem is exited.
func (s *BaseMySqlParserListener) ExitSubqueryTableItem(ctx *SubqueryTableItemContext) {}

// EnterTableSourcesItem is called when production tableSourcesItem is entered.
func (s *BaseMySqlParserListener) EnterTableSourcesItem(ctx *TableSourcesItemContext) {}

// ExitTableSourcesItem is called when production tableSourcesItem is exited.
func (s *BaseMySqlParserListener) ExitTableSourcesItem(ctx *TableSourcesItemContext) {}

// EnterIndexHint is called when production indexHint is entered.
func (s *BaseMySqlParserListener) EnterIndexHint(ctx *IndexHintContext) {}

// ExitIndexHint is called when production indexHint is exited.
func (s *BaseMySqlParserListener) ExitIndexHint(ctx *IndexHintContext) {}

// EnterIndexHintType is called when production indexHintType is entered.
func (s *BaseMySqlParserListener) EnterIndexHintType(ctx *IndexHintTypeContext) {}

// ExitIndexHintType is called when production indexHintType is exited.
func (s *BaseMySqlParserListener) ExitIndexHintType(ctx *IndexHintTypeContext) {}

// EnterInnerJoin is called when production innerJoin is entered.
func (s *BaseMySqlParserListener) EnterInnerJoin(ctx *InnerJoinContext) {}

// ExitInnerJoin is called when production innerJoin is exited.
func (s *BaseMySqlParserListener) ExitInnerJoin(ctx *InnerJoinContext) {}

// EnterStraightJoin is called when production straightJoin is entered.
func (s *BaseMySqlParserListener) EnterStraightJoin(ctx *StraightJoinContext) {}

// ExitStraightJoin is called when production straightJoin is exited.
func (s *BaseMySqlParserListener) ExitStraightJoin(ctx *StraightJoinContext) {}

// EnterOuterJoin is called when production outerJoin is entered.
func (s *BaseMySqlParserListener) EnterOuterJoin(ctx *OuterJoinContext) {}

// ExitOuterJoin is called when production outerJoin is exited.
func (s *BaseMySqlParserListener) ExitOuterJoin(ctx *OuterJoinContext) {}

// EnterNaturalJoin is called when production naturalJoin is entered.
func (s *BaseMySqlParserListener) EnterNaturalJoin(ctx *NaturalJoinContext) {}

// ExitNaturalJoin is called when production naturalJoin is exited.
func (s *BaseMySqlParserListener) ExitNaturalJoin(ctx *NaturalJoinContext) {}

// EnterJoinSpec is called when production joinSpec is entered.
func (s *BaseMySqlParserListener) EnterJoinSpec(ctx *JoinSpecContext) {}

// ExitJoinSpec is called when production joinSpec is exited.
func (s *BaseMySqlParserListener) ExitJoinSpec(ctx *JoinSpecContext) {}

// EnterQueryExpression is called when production queryExpression is entered.
func (s *BaseMySqlParserListener) EnterQueryExpression(ctx *QueryExpressionContext) {}

// ExitQueryExpression is called when production queryExpression is exited.
func (s *BaseMySqlParserListener) ExitQueryExpression(ctx *QueryExpressionContext) {}

// EnterQueryExpressionNointo is called when production queryExpressionNointo is entered.
func (s *BaseMySqlParserListener) EnterQueryExpressionNointo(ctx *QueryExpressionNointoContext) {}

// ExitQueryExpressionNointo is called when production queryExpressionNointo is exited.
func (s *BaseMySqlParserListener) ExitQueryExpressionNointo(ctx *QueryExpressionNointoContext) {}

// EnterQuerySpecification is called when production querySpecification is entered.
func (s *BaseMySqlParserListener) EnterQuerySpecification(ctx *QuerySpecificationContext) {}

// ExitQuerySpecification is called when production querySpecification is exited.
func (s *BaseMySqlParserListener) ExitQuerySpecification(ctx *QuerySpecificationContext) {}

// EnterQuerySpecificationNointo is called when production querySpecificationNointo is entered.
func (s *BaseMySqlParserListener) EnterQuerySpecificationNointo(ctx *QuerySpecificationNointoContext) {
}

// ExitQuerySpecificationNointo is called when production querySpecificationNointo is exited.
func (s *BaseMySqlParserListener) ExitQuerySpecificationNointo(ctx *QuerySpecificationNointoContext) {
}

// EnterUnionParenthesis is called when production unionParenthesis is entered.
func (s *BaseMySqlParserListener) EnterUnionParenthesis(ctx *UnionParenthesisContext) {}

// ExitUnionParenthesis is called when production unionParenthesis is exited.
func (s *BaseMySqlParserListener) ExitUnionParenthesis(ctx *UnionParenthesisContext) {}

// EnterUnionStatement is called when production unionStatement is entered.
func (s *BaseMySqlParserListener) EnterUnionStatement(ctx *UnionStatementContext) {}

// ExitUnionStatement is called when production unionStatement is exited.
func (s *BaseMySqlParserListener) ExitUnionStatement(ctx *UnionStatementContext) {}

// EnterLateralStatement is called when production lateralStatement is entered.
func (s *BaseMySqlParserListener) EnterLateralStatement(ctx *LateralStatementContext) {}

// ExitLateralStatement is called when production lateralStatement is exited.
func (s *BaseMySqlParserListener) ExitLateralStatement(ctx *LateralStatementContext) {}

// EnterJsonTable is called when production jsonTable is entered.
func (s *BaseMySqlParserListener) EnterJsonTable(ctx *JsonTableContext) {}

// ExitJsonTable is called when production jsonTable is exited.
func (s *BaseMySqlParserListener) ExitJsonTable(ctx *JsonTableContext) {}

// EnterJsonColumnList is called when production jsonColumnList is entered.
func (s *BaseMySqlParserListener) EnterJsonColumnList(ctx *JsonColumnListContext) {}

// ExitJsonColumnList is called when production jsonColumnList is exited.
func (s *BaseMySqlParserListener) ExitJsonColumnList(ctx *JsonColumnListContext) {}

// EnterJsonColumn is called when production jsonColumn is entered.
func (s *BaseMySqlParserListener) EnterJsonColumn(ctx *JsonColumnContext) {}

// ExitJsonColumn is called when production jsonColumn is exited.
func (s *BaseMySqlParserListener) ExitJsonColumn(ctx *JsonColumnContext) {}

// EnterJsonOnEmpty is called when production jsonOnEmpty is entered.
func (s *BaseMySqlParserListener) EnterJsonOnEmpty(ctx *JsonOnEmptyContext) {}

// ExitJsonOnEmpty is called when production jsonOnEmpty is exited.
func (s *BaseMySqlParserListener) ExitJsonOnEmpty(ctx *JsonOnEmptyContext) {}

// EnterJsonOnError is called when production jsonOnError is entered.
func (s *BaseMySqlParserListener) EnterJsonOnError(ctx *JsonOnErrorContext) {}

// ExitJsonOnError is called when production jsonOnError is exited.
func (s *BaseMySqlParserListener) ExitJsonOnError(ctx *JsonOnErrorContext) {}

// EnterSelectSpec is called when production selectSpec is entered.
func (s *BaseMySqlParserListener) EnterSelectSpec(ctx *SelectSpecContext) {}

// ExitSelectSpec is called when production selectSpec is exited.
func (s *BaseMySqlParserListener) ExitSelectSpec(ctx *SelectSpecContext) {}

// EnterSelectElements is called when production selectElements is entered.
func (s *BaseMySqlParserListener) EnterSelectElements(ctx *SelectElementsContext) {}

// ExitSelectElements is called when production selectElements is exited.
func (s *BaseMySqlParserListener) ExitSelectElements(ctx *SelectElementsContext) {}

// EnterSelectStarElement is called when production selectStarElement is entered.
func (s *BaseMySqlParserListener) EnterSelectStarElement(ctx *SelectStarElementContext) {}

// ExitSelectStarElement is called when production selectStarElement is exited.
func (s *BaseMySqlParserListener) ExitSelectStarElement(ctx *SelectStarElementContext) {}

// EnterSelectColumnElement is called when production selectColumnElement is entered.
func (s *BaseMySqlParserListener) EnterSelectColumnElement(ctx *SelectColumnElementContext) {}

// ExitSelectColumnElement is called when production selectColumnElement is exited.
func (s *BaseMySqlParserListener) ExitSelectColumnElement(ctx *SelectColumnElementContext) {}

// EnterSelectFunctionElement is called when production selectFunctionElement is entered.
func (s *BaseMySqlParserListener) EnterSelectFunctionElement(ctx *SelectFunctionElementContext) {}

// ExitSelectFunctionElement is called when production selectFunctionElement is exited.
func (s *BaseMySqlParserListener) ExitSelectFunctionElement(ctx *SelectFunctionElementContext) {}

// EnterSelectExpressionElement is called when production selectExpressionElement is entered.
func (s *BaseMySqlParserListener) EnterSelectExpressionElement(ctx *SelectExpressionElementContext) {}

// ExitSelectExpressionElement is called when production selectExpressionElement is exited.
func (s *BaseMySqlParserListener) ExitSelectExpressionElement(ctx *SelectExpressionElementContext) {}

// EnterSelectIntoVariables is called when production selectIntoVariables is entered.
func (s *BaseMySqlParserListener) EnterSelectIntoVariables(ctx *SelectIntoVariablesContext) {}

// ExitSelectIntoVariables is called when production selectIntoVariables is exited.
func (s *BaseMySqlParserListener) ExitSelectIntoVariables(ctx *SelectIntoVariablesContext) {}

// EnterSelectIntoDumpFile is called when production selectIntoDumpFile is entered.
func (s *BaseMySqlParserListener) EnterSelectIntoDumpFile(ctx *SelectIntoDumpFileContext) {}

// ExitSelectIntoDumpFile is called when production selectIntoDumpFile is exited.
func (s *BaseMySqlParserListener) ExitSelectIntoDumpFile(ctx *SelectIntoDumpFileContext) {}

// EnterSelectIntoTextFile is called when production selectIntoTextFile is entered.
func (s *BaseMySqlParserListener) EnterSelectIntoTextFile(ctx *SelectIntoTextFileContext) {}

// ExitSelectIntoTextFile is called when production selectIntoTextFile is exited.
func (s *BaseMySqlParserListener) ExitSelectIntoTextFile(ctx *SelectIntoTextFileContext) {}

// EnterSelectFieldsInto is called when production selectFieldsInto is entered.
func (s *BaseMySqlParserListener) EnterSelectFieldsInto(ctx *SelectFieldsIntoContext) {}

// ExitSelectFieldsInto is called when production selectFieldsInto is exited.
func (s *BaseMySqlParserListener) ExitSelectFieldsInto(ctx *SelectFieldsIntoContext) {}

// EnterSelectLinesInto is called when production selectLinesInto is entered.
func (s *BaseMySqlParserListener) EnterSelectLinesInto(ctx *SelectLinesIntoContext) {}

// ExitSelectLinesInto is called when production selectLinesInto is exited.
func (s *BaseMySqlParserListener) ExitSelectLinesInto(ctx *SelectLinesIntoContext) {}

// EnterFromClause is called when production fromClause is entered.
func (s *BaseMySqlParserListener) EnterFromClause(ctx *FromClauseContext) {}

// ExitFromClause is called when production fromClause is exited.
func (s *BaseMySqlParserListener) ExitFromClause(ctx *FromClauseContext) {}

// EnterGroupByClause is called when production groupByClause is entered.
func (s *BaseMySqlParserListener) EnterGroupByClause(ctx *GroupByClauseContext) {}

// ExitGroupByClause is called when production groupByClause is exited.
func (s *BaseMySqlParserListener) ExitGroupByClause(ctx *GroupByClauseContext) {}

// EnterHavingClause is called when production havingClause is entered.
func (s *BaseMySqlParserListener) EnterHavingClause(ctx *HavingClauseContext) {}

// ExitHavingClause is called when production havingClause is exited.
func (s *BaseMySqlParserListener) ExitHavingClause(ctx *HavingClauseContext) {}

// EnterWindowClause is called when production windowClause is entered.
func (s *BaseMySqlParserListener) EnterWindowClause(ctx *WindowClauseContext) {}

// ExitWindowClause is called when production windowClause is exited.
func (s *BaseMySqlParserListener) ExitWindowClause(ctx *WindowClauseContext) {}

// EnterGroupByItem is called when production groupByItem is entered.
func (s *BaseMySqlParserListener) EnterGroupByItem(ctx *GroupByItemContext) {}

// ExitGroupByItem is called when production groupByItem is exited.
func (s *BaseMySqlParserListener) ExitGroupByItem(ctx *GroupByItemContext) {}

// EnterLimitClause is called when production limitClause is entered.
func (s *BaseMySqlParserListener) EnterLimitClause(ctx *LimitClauseContext) {}

// ExitLimitClause is called when production limitClause is exited.
func (s *BaseMySqlParserListener) ExitLimitClause(ctx *LimitClauseContext) {}

// EnterLimitClauseAtom is called when production limitClauseAtom is entered.
func (s *BaseMySqlParserListener) EnterLimitClauseAtom(ctx *LimitClauseAtomContext) {}

// ExitLimitClauseAtom is called when production limitClauseAtom is exited.
func (s *BaseMySqlParserListener) ExitLimitClauseAtom(ctx *LimitClauseAtomContext) {}

// EnterStartTransaction is called when production startTransaction is entered.
func (s *BaseMySqlParserListener) EnterStartTransaction(ctx *StartTransactionContext) {}

// ExitStartTransaction is called when production startTransaction is exited.
func (s *BaseMySqlParserListener) ExitStartTransaction(ctx *StartTransactionContext) {}

// EnterBeginWork is called when production beginWork is entered.
func (s *BaseMySqlParserListener) EnterBeginWork(ctx *BeginWorkContext) {}

// ExitBeginWork is called when production beginWork is exited.
func (s *BaseMySqlParserListener) ExitBeginWork(ctx *BeginWorkContext) {}

// EnterCommitWork is called when production commitWork is entered.
func (s *BaseMySqlParserListener) EnterCommitWork(ctx *CommitWorkContext) {}

// ExitCommitWork is called when production commitWork is exited.
func (s *BaseMySqlParserListener) ExitCommitWork(ctx *CommitWorkContext) {}

// EnterRollbackWork is called when production rollbackWork is entered.
func (s *BaseMySqlParserListener) EnterRollbackWork(ctx *RollbackWorkContext) {}

// ExitRollbackWork is called when production rollbackWork is exited.
func (s *BaseMySqlParserListener) ExitRollbackWork(ctx *RollbackWorkContext) {}

// EnterSavepointStatement is called when production savepointStatement is entered.
func (s *BaseMySqlParserListener) EnterSavepointStatement(ctx *SavepointStatementContext) {}

// ExitSavepointStatement is called when production savepointStatement is exited.
func (s *BaseMySqlParserListener) ExitSavepointStatement(ctx *SavepointStatementContext) {}

// EnterRollbackStatement is called when production rollbackStatement is entered.
func (s *BaseMySqlParserListener) EnterRollbackStatement(ctx *RollbackStatementContext) {}

// ExitRollbackStatement is called when production rollbackStatement is exited.
func (s *BaseMySqlParserListener) ExitRollbackStatement(ctx *RollbackStatementContext) {}

// EnterReleaseStatement is called when production releaseStatement is entered.
func (s *BaseMySqlParserListener) EnterReleaseStatement(ctx *ReleaseStatementContext) {}

// ExitReleaseStatement is called when production releaseStatement is exited.
func (s *BaseMySqlParserListener) ExitReleaseStatement(ctx *ReleaseStatementContext) {}

// EnterLockTables is called when production lockTables is entered.
func (s *BaseMySqlParserListener) EnterLockTables(ctx *LockTablesContext) {}

// ExitLockTables is called when production lockTables is exited.
func (s *BaseMySqlParserListener) ExitLockTables(ctx *LockTablesContext) {}

// EnterUnlockTables is called when production unlockTables is entered.
func (s *BaseMySqlParserListener) EnterUnlockTables(ctx *UnlockTablesContext) {}

// ExitUnlockTables is called when production unlockTables is exited.
func (s *BaseMySqlParserListener) ExitUnlockTables(ctx *UnlockTablesContext) {}

// EnterSetAutocommitStatement is called when production setAutocommitStatement is entered.
func (s *BaseMySqlParserListener) EnterSetAutocommitStatement(ctx *SetAutocommitStatementContext) {}

// ExitSetAutocommitStatement is called when production setAutocommitStatement is exited.
func (s *BaseMySqlParserListener) ExitSetAutocommitStatement(ctx *SetAutocommitStatementContext) {}

// EnterSetTransactionStatement is called when production setTransactionStatement is entered.
func (s *BaseMySqlParserListener) EnterSetTransactionStatement(ctx *SetTransactionStatementContext) {}

// ExitSetTransactionStatement is called when production setTransactionStatement is exited.
func (s *BaseMySqlParserListener) ExitSetTransactionStatement(ctx *SetTransactionStatementContext) {}

// EnterTransactionMode is called when production transactionMode is entered.
func (s *BaseMySqlParserListener) EnterTransactionMode(ctx *TransactionModeContext) {}

// ExitTransactionMode is called when production transactionMode is exited.
func (s *BaseMySqlParserListener) ExitTransactionMode(ctx *TransactionModeContext) {}

// EnterLockTableElement is called when production lockTableElement is entered.
func (s *BaseMySqlParserListener) EnterLockTableElement(ctx *LockTableElementContext) {}

// ExitLockTableElement is called when production lockTableElement is exited.
func (s *BaseMySqlParserListener) ExitLockTableElement(ctx *LockTableElementContext) {}

// EnterLockAction is called when production lockAction is entered.
func (s *BaseMySqlParserListener) EnterLockAction(ctx *LockActionContext) {}

// ExitLockAction is called when production lockAction is exited.
func (s *BaseMySqlParserListener) ExitLockAction(ctx *LockActionContext) {}

// EnterTransactionOption is called when production transactionOption is entered.
func (s *BaseMySqlParserListener) EnterTransactionOption(ctx *TransactionOptionContext) {}

// ExitTransactionOption is called when production transactionOption is exited.
func (s *BaseMySqlParserListener) ExitTransactionOption(ctx *TransactionOptionContext) {}

// EnterTransactionLevel is called when production transactionLevel is entered.
func (s *BaseMySqlParserListener) EnterTransactionLevel(ctx *TransactionLevelContext) {}

// ExitTransactionLevel is called when production transactionLevel is exited.
func (s *BaseMySqlParserListener) ExitTransactionLevel(ctx *TransactionLevelContext) {}

// EnterChangeMaster is called when production changeMaster is entered.
func (s *BaseMySqlParserListener) EnterChangeMaster(ctx *ChangeMasterContext) {}

// ExitChangeMaster is called when production changeMaster is exited.
func (s *BaseMySqlParserListener) ExitChangeMaster(ctx *ChangeMasterContext) {}

// EnterChangeReplicationFilter is called when production changeReplicationFilter is entered.
func (s *BaseMySqlParserListener) EnterChangeReplicationFilter(ctx *ChangeReplicationFilterContext) {}

// ExitChangeReplicationFilter is called when production changeReplicationFilter is exited.
func (s *BaseMySqlParserListener) ExitChangeReplicationFilter(ctx *ChangeReplicationFilterContext) {}

// EnterPurgeBinaryLogs is called when production purgeBinaryLogs is entered.
func (s *BaseMySqlParserListener) EnterPurgeBinaryLogs(ctx *PurgeBinaryLogsContext) {}

// ExitPurgeBinaryLogs is called when production purgeBinaryLogs is exited.
func (s *BaseMySqlParserListener) ExitPurgeBinaryLogs(ctx *PurgeBinaryLogsContext) {}

// EnterResetMaster is called when production resetMaster is entered.
func (s *BaseMySqlParserListener) EnterResetMaster(ctx *ResetMasterContext) {}

// ExitResetMaster is called when production resetMaster is exited.
func (s *BaseMySqlParserListener) ExitResetMaster(ctx *ResetMasterContext) {}

// EnterResetSlave is called when production resetSlave is entered.
func (s *BaseMySqlParserListener) EnterResetSlave(ctx *ResetSlaveContext) {}

// ExitResetSlave is called when production resetSlave is exited.
func (s *BaseMySqlParserListener) ExitResetSlave(ctx *ResetSlaveContext) {}

// EnterStartSlave is called when production startSlave is entered.
func (s *BaseMySqlParserListener) EnterStartSlave(ctx *StartSlaveContext) {}

// ExitStartSlave is called when production startSlave is exited.
func (s *BaseMySqlParserListener) ExitStartSlave(ctx *StartSlaveContext) {}

// EnterStopSlave is called when production stopSlave is entered.
func (s *BaseMySqlParserListener) EnterStopSlave(ctx *StopSlaveContext) {}

// ExitStopSlave is called when production stopSlave is exited.
func (s *BaseMySqlParserListener) ExitStopSlave(ctx *StopSlaveContext) {}

// EnterStartGroupReplication is called when production startGroupReplication is entered.
func (s *BaseMySqlParserListener) EnterStartGroupReplication(ctx *StartGroupReplicationContext) {}

// ExitStartGroupReplication is called when production startGroupReplication is exited.
func (s *BaseMySqlParserListener) ExitStartGroupReplication(ctx *StartGroupReplicationContext) {}

// EnterStopGroupReplication is called when production stopGroupReplication is entered.
func (s *BaseMySqlParserListener) EnterStopGroupReplication(ctx *StopGroupReplicationContext) {}

// ExitStopGroupReplication is called when production stopGroupReplication is exited.
func (s *BaseMySqlParserListener) ExitStopGroupReplication(ctx *StopGroupReplicationContext) {}

// EnterMasterStringOption is called when production masterStringOption is entered.
func (s *BaseMySqlParserListener) EnterMasterStringOption(ctx *MasterStringOptionContext) {}

// ExitMasterStringOption is called when production masterStringOption is exited.
func (s *BaseMySqlParserListener) ExitMasterStringOption(ctx *MasterStringOptionContext) {}

// EnterMasterDecimalOption is called when production masterDecimalOption is entered.
func (s *BaseMySqlParserListener) EnterMasterDecimalOption(ctx *MasterDecimalOptionContext) {}

// ExitMasterDecimalOption is called when production masterDecimalOption is exited.
func (s *BaseMySqlParserListener) ExitMasterDecimalOption(ctx *MasterDecimalOptionContext) {}

// EnterMasterBoolOption is called when production masterBoolOption is entered.
func (s *BaseMySqlParserListener) EnterMasterBoolOption(ctx *MasterBoolOptionContext) {}

// ExitMasterBoolOption is called when production masterBoolOption is exited.
func (s *BaseMySqlParserListener) ExitMasterBoolOption(ctx *MasterBoolOptionContext) {}

// EnterMasterRealOption is called when production masterRealOption is entered.
func (s *BaseMySqlParserListener) EnterMasterRealOption(ctx *MasterRealOptionContext) {}

// ExitMasterRealOption is called when production masterRealOption is exited.
func (s *BaseMySqlParserListener) ExitMasterRealOption(ctx *MasterRealOptionContext) {}

// EnterMasterUidListOption is called when production masterUidListOption is entered.
func (s *BaseMySqlParserListener) EnterMasterUidListOption(ctx *MasterUidListOptionContext) {}

// ExitMasterUidListOption is called when production masterUidListOption is exited.
func (s *BaseMySqlParserListener) ExitMasterUidListOption(ctx *MasterUidListOptionContext) {}

// EnterStringMasterOption is called when production stringMasterOption is entered.
func (s *BaseMySqlParserListener) EnterStringMasterOption(ctx *StringMasterOptionContext) {}

// ExitStringMasterOption is called when production stringMasterOption is exited.
func (s *BaseMySqlParserListener) ExitStringMasterOption(ctx *StringMasterOptionContext) {}

// EnterDecimalMasterOption is called when production decimalMasterOption is entered.
func (s *BaseMySqlParserListener) EnterDecimalMasterOption(ctx *DecimalMasterOptionContext) {}

// ExitDecimalMasterOption is called when production decimalMasterOption is exited.
func (s *BaseMySqlParserListener) ExitDecimalMasterOption(ctx *DecimalMasterOptionContext) {}

// EnterBoolMasterOption is called when production boolMasterOption is entered.
func (s *BaseMySqlParserListener) EnterBoolMasterOption(ctx *BoolMasterOptionContext) {}

// ExitBoolMasterOption is called when production boolMasterOption is exited.
func (s *BaseMySqlParserListener) ExitBoolMasterOption(ctx *BoolMasterOptionContext) {}

// EnterChannelOption is called when production channelOption is entered.
func (s *BaseMySqlParserListener) EnterChannelOption(ctx *ChannelOptionContext) {}

// ExitChannelOption is called when production channelOption is exited.
func (s *BaseMySqlParserListener) ExitChannelOption(ctx *ChannelOptionContext) {}

// EnterDoDbReplication is called when production doDbReplication is entered.
func (s *BaseMySqlParserListener) EnterDoDbReplication(ctx *DoDbReplicationContext) {}

// ExitDoDbReplication is called when production doDbReplication is exited.
func (s *BaseMySqlParserListener) ExitDoDbReplication(ctx *DoDbReplicationContext) {}

// EnterIgnoreDbReplication is called when production ignoreDbReplication is entered.
func (s *BaseMySqlParserListener) EnterIgnoreDbReplication(ctx *IgnoreDbReplicationContext) {}

// ExitIgnoreDbReplication is called when production ignoreDbReplication is exited.
func (s *BaseMySqlParserListener) ExitIgnoreDbReplication(ctx *IgnoreDbReplicationContext) {}

// EnterDoTableReplication is called when production doTableReplication is entered.
func (s *BaseMySqlParserListener) EnterDoTableReplication(ctx *DoTableReplicationContext) {}

// ExitDoTableReplication is called when production doTableReplication is exited.
func (s *BaseMySqlParserListener) ExitDoTableReplication(ctx *DoTableReplicationContext) {}

// EnterIgnoreTableReplication is called when production ignoreTableReplication is entered.
func (s *BaseMySqlParserListener) EnterIgnoreTableReplication(ctx *IgnoreTableReplicationContext) {}

// ExitIgnoreTableReplication is called when production ignoreTableReplication is exited.
func (s *BaseMySqlParserListener) ExitIgnoreTableReplication(ctx *IgnoreTableReplicationContext) {}

// EnterWildDoTableReplication is called when production wildDoTableReplication is entered.
func (s *BaseMySqlParserListener) EnterWildDoTableReplication(ctx *WildDoTableReplicationContext) {}

// ExitWildDoTableReplication is called when production wildDoTableReplication is exited.
func (s *BaseMySqlParserListener) ExitWildDoTableReplication(ctx *WildDoTableReplicationContext) {}

// EnterWildIgnoreTableReplication is called when production wildIgnoreTableReplication is entered.
func (s *BaseMySqlParserListener) EnterWildIgnoreTableReplication(ctx *WildIgnoreTableReplicationContext) {
}

// ExitWildIgnoreTableReplication is called when production wildIgnoreTableReplication is exited.
func (s *BaseMySqlParserListener) ExitWildIgnoreTableReplication(ctx *WildIgnoreTableReplicationContext) {
}

// EnterRewriteDbReplication is called when production rewriteDbReplication is entered.
func (s *BaseMySqlParserListener) EnterRewriteDbReplication(ctx *RewriteDbReplicationContext) {}

// ExitRewriteDbReplication is called when production rewriteDbReplication is exited.
func (s *BaseMySqlParserListener) ExitRewriteDbReplication(ctx *RewriteDbReplicationContext) {}

// EnterTablePair is called when production tablePair is entered.
func (s *BaseMySqlParserListener) EnterTablePair(ctx *TablePairContext) {}

// ExitTablePair is called when production tablePair is exited.
func (s *BaseMySqlParserListener) ExitTablePair(ctx *TablePairContext) {}

// EnterThreadType is called when production threadType is entered.
func (s *BaseMySqlParserListener) EnterThreadType(ctx *ThreadTypeContext) {}

// ExitThreadType is called when production threadType is exited.
func (s *BaseMySqlParserListener) ExitThreadType(ctx *ThreadTypeContext) {}

// EnterGtidsUntilOption is called when production gtidsUntilOption is entered.
func (s *BaseMySqlParserListener) EnterGtidsUntilOption(ctx *GtidsUntilOptionContext) {}

// ExitGtidsUntilOption is called when production gtidsUntilOption is exited.
func (s *BaseMySqlParserListener) ExitGtidsUntilOption(ctx *GtidsUntilOptionContext) {}

// EnterMasterLogUntilOption is called when production masterLogUntilOption is entered.
func (s *BaseMySqlParserListener) EnterMasterLogUntilOption(ctx *MasterLogUntilOptionContext) {}

// ExitMasterLogUntilOption is called when production masterLogUntilOption is exited.
func (s *BaseMySqlParserListener) ExitMasterLogUntilOption(ctx *MasterLogUntilOptionContext) {}

// EnterRelayLogUntilOption is called when production relayLogUntilOption is entered.
func (s *BaseMySqlParserListener) EnterRelayLogUntilOption(ctx *RelayLogUntilOptionContext) {}

// ExitRelayLogUntilOption is called when production relayLogUntilOption is exited.
func (s *BaseMySqlParserListener) ExitRelayLogUntilOption(ctx *RelayLogUntilOptionContext) {}

// EnterSqlGapsUntilOption is called when production sqlGapsUntilOption is entered.
func (s *BaseMySqlParserListener) EnterSqlGapsUntilOption(ctx *SqlGapsUntilOptionContext) {}

// ExitSqlGapsUntilOption is called when production sqlGapsUntilOption is exited.
func (s *BaseMySqlParserListener) ExitSqlGapsUntilOption(ctx *SqlGapsUntilOptionContext) {}

// EnterUserConnectionOption is called when production userConnectionOption is entered.
func (s *BaseMySqlParserListener) EnterUserConnectionOption(ctx *UserConnectionOptionContext) {}

// ExitUserConnectionOption is called when production userConnectionOption is exited.
func (s *BaseMySqlParserListener) ExitUserConnectionOption(ctx *UserConnectionOptionContext) {}

// EnterPasswordConnectionOption is called when production passwordConnectionOption is entered.
func (s *BaseMySqlParserListener) EnterPasswordConnectionOption(ctx *PasswordConnectionOptionContext) {
}

// ExitPasswordConnectionOption is called when production passwordConnectionOption is exited.
func (s *BaseMySqlParserListener) ExitPasswordConnectionOption(ctx *PasswordConnectionOptionContext) {
}

// EnterDefaultAuthConnectionOption is called when production defaultAuthConnectionOption is entered.
func (s *BaseMySqlParserListener) EnterDefaultAuthConnectionOption(ctx *DefaultAuthConnectionOptionContext) {
}

// ExitDefaultAuthConnectionOption is called when production defaultAuthConnectionOption is exited.
func (s *BaseMySqlParserListener) ExitDefaultAuthConnectionOption(ctx *DefaultAuthConnectionOptionContext) {
}

// EnterPluginDirConnectionOption is called when production pluginDirConnectionOption is entered.
func (s *BaseMySqlParserListener) EnterPluginDirConnectionOption(ctx *PluginDirConnectionOptionContext) {
}

// ExitPluginDirConnectionOption is called when production pluginDirConnectionOption is exited.
func (s *BaseMySqlParserListener) ExitPluginDirConnectionOption(ctx *PluginDirConnectionOptionContext) {
}

// EnterGtuidSet is called when production gtuidSet is entered.
func (s *BaseMySqlParserListener) EnterGtuidSet(ctx *GtuidSetContext) {}

// ExitGtuidSet is called when production gtuidSet is exited.
func (s *BaseMySqlParserListener) ExitGtuidSet(ctx *GtuidSetContext) {}

// EnterXaStartTransaction is called when production xaStartTransaction is entered.
func (s *BaseMySqlParserListener) EnterXaStartTransaction(ctx *XaStartTransactionContext) {}

// ExitXaStartTransaction is called when production xaStartTransaction is exited.
func (s *BaseMySqlParserListener) ExitXaStartTransaction(ctx *XaStartTransactionContext) {}

// EnterXaEndTransaction is called when production xaEndTransaction is entered.
func (s *BaseMySqlParserListener) EnterXaEndTransaction(ctx *XaEndTransactionContext) {}

// ExitXaEndTransaction is called when production xaEndTransaction is exited.
func (s *BaseMySqlParserListener) ExitXaEndTransaction(ctx *XaEndTransactionContext) {}

// EnterXaPrepareStatement is called when production xaPrepareStatement is entered.
func (s *BaseMySqlParserListener) EnterXaPrepareStatement(ctx *XaPrepareStatementContext) {}

// ExitXaPrepareStatement is called when production xaPrepareStatement is exited.
func (s *BaseMySqlParserListener) ExitXaPrepareStatement(ctx *XaPrepareStatementContext) {}

// EnterXaCommitWork is called when production xaCommitWork is entered.
func (s *BaseMySqlParserListener) EnterXaCommitWork(ctx *XaCommitWorkContext) {}

// ExitXaCommitWork is called when production xaCommitWork is exited.
func (s *BaseMySqlParserListener) ExitXaCommitWork(ctx *XaCommitWorkContext) {}

// EnterXaRollbackWork is called when production xaRollbackWork is entered.
func (s *BaseMySqlParserListener) EnterXaRollbackWork(ctx *XaRollbackWorkContext) {}

// ExitXaRollbackWork is called when production xaRollbackWork is exited.
func (s *BaseMySqlParserListener) ExitXaRollbackWork(ctx *XaRollbackWorkContext) {}

// EnterXaRecoverWork is called when production xaRecoverWork is entered.
func (s *BaseMySqlParserListener) EnterXaRecoverWork(ctx *XaRecoverWorkContext) {}

// ExitXaRecoverWork is called when production xaRecoverWork is exited.
func (s *BaseMySqlParserListener) ExitXaRecoverWork(ctx *XaRecoverWorkContext) {}

// EnterPrepareStatement is called when production prepareStatement is entered.
func (s *BaseMySqlParserListener) EnterPrepareStatement(ctx *PrepareStatementContext) {}

// ExitPrepareStatement is called when production prepareStatement is exited.
func (s *BaseMySqlParserListener) ExitPrepareStatement(ctx *PrepareStatementContext) {}

// EnterExecuteStatement is called when production executeStatement is entered.
func (s *BaseMySqlParserListener) EnterExecuteStatement(ctx *ExecuteStatementContext) {}

// ExitExecuteStatement is called when production executeStatement is exited.
func (s *BaseMySqlParserListener) ExitExecuteStatement(ctx *ExecuteStatementContext) {}

// EnterDeallocatePrepare is called when production deallocatePrepare is entered.
func (s *BaseMySqlParserListener) EnterDeallocatePrepare(ctx *DeallocatePrepareContext) {}

// ExitDeallocatePrepare is called when production deallocatePrepare is exited.
func (s *BaseMySqlParserListener) ExitDeallocatePrepare(ctx *DeallocatePrepareContext) {}

// EnterRoutineBody is called when production routineBody is entered.
func (s *BaseMySqlParserListener) EnterRoutineBody(ctx *RoutineBodyContext) {}

// ExitRoutineBody is called when production routineBody is exited.
func (s *BaseMySqlParserListener) ExitRoutineBody(ctx *RoutineBodyContext) {}

// EnterBlockStatement is called when production blockStatement is entered.
func (s *BaseMySqlParserListener) EnterBlockStatement(ctx *BlockStatementContext) {}

// ExitBlockStatement is called when production blockStatement is exited.
func (s *BaseMySqlParserListener) ExitBlockStatement(ctx *BlockStatementContext) {}

// EnterCaseStatement is called when production caseStatement is entered.
func (s *BaseMySqlParserListener) EnterCaseStatement(ctx *CaseStatementContext) {}

// ExitCaseStatement is called when production caseStatement is exited.
func (s *BaseMySqlParserListener) ExitCaseStatement(ctx *CaseStatementContext) {}

// EnterIfStatement is called when production ifStatement is entered.
func (s *BaseMySqlParserListener) EnterIfStatement(ctx *IfStatementContext) {}

// ExitIfStatement is called when production ifStatement is exited.
func (s *BaseMySqlParserListener) ExitIfStatement(ctx *IfStatementContext) {}

// EnterIterateStatement is called when production iterateStatement is entered.
func (s *BaseMySqlParserListener) EnterIterateStatement(ctx *IterateStatementContext) {}

// ExitIterateStatement is called when production iterateStatement is exited.
func (s *BaseMySqlParserListener) ExitIterateStatement(ctx *IterateStatementContext) {}

// EnterLeaveStatement is called when production leaveStatement is entered.
func (s *BaseMySqlParserListener) EnterLeaveStatement(ctx *LeaveStatementContext) {}

// ExitLeaveStatement is called when production leaveStatement is exited.
func (s *BaseMySqlParserListener) ExitLeaveStatement(ctx *LeaveStatementContext) {}

// EnterLoopStatement is called when production loopStatement is entered.
func (s *BaseMySqlParserListener) EnterLoopStatement(ctx *LoopStatementContext) {}

// ExitLoopStatement is called when production loopStatement is exited.
func (s *BaseMySqlParserListener) ExitLoopStatement(ctx *LoopStatementContext) {}

// EnterRepeatStatement is called when production repeatStatement is entered.
func (s *BaseMySqlParserListener) EnterRepeatStatement(ctx *RepeatStatementContext) {}

// ExitRepeatStatement is called when production repeatStatement is exited.
func (s *BaseMySqlParserListener) ExitRepeatStatement(ctx *RepeatStatementContext) {}

// EnterReturnStatement is called when production returnStatement is entered.
func (s *BaseMySqlParserListener) EnterReturnStatement(ctx *ReturnStatementContext) {}

// ExitReturnStatement is called when production returnStatement is exited.
func (s *BaseMySqlParserListener) ExitReturnStatement(ctx *ReturnStatementContext) {}

// EnterWhileStatement is called when production whileStatement is entered.
func (s *BaseMySqlParserListener) EnterWhileStatement(ctx *WhileStatementContext) {}

// ExitWhileStatement is called when production whileStatement is exited.
func (s *BaseMySqlParserListener) ExitWhileStatement(ctx *WhileStatementContext) {}

// EnterCloseCursor is called when production CloseCursor is entered.
func (s *BaseMySqlParserListener) EnterCloseCursor(ctx *CloseCursorContext) {}

// ExitCloseCursor is called when production CloseCursor is exited.
func (s *BaseMySqlParserListener) ExitCloseCursor(ctx *CloseCursorContext) {}

// EnterFetchCursor is called when production FetchCursor is entered.
func (s *BaseMySqlParserListener) EnterFetchCursor(ctx *FetchCursorContext) {}

// ExitFetchCursor is called when production FetchCursor is exited.
func (s *BaseMySqlParserListener) ExitFetchCursor(ctx *FetchCursorContext) {}

// EnterOpenCursor is called when production OpenCursor is entered.
func (s *BaseMySqlParserListener) EnterOpenCursor(ctx *OpenCursorContext) {}

// ExitOpenCursor is called when production OpenCursor is exited.
func (s *BaseMySqlParserListener) ExitOpenCursor(ctx *OpenCursorContext) {}

// EnterDeclareVariable is called when production declareVariable is entered.
func (s *BaseMySqlParserListener) EnterDeclareVariable(ctx *DeclareVariableContext) {}

// ExitDeclareVariable is called when production declareVariable is exited.
func (s *BaseMySqlParserListener) ExitDeclareVariable(ctx *DeclareVariableContext) {}

// EnterDeclareCondition is called when production declareCondition is entered.
func (s *BaseMySqlParserListener) EnterDeclareCondition(ctx *DeclareConditionContext) {}

// ExitDeclareCondition is called when production declareCondition is exited.
func (s *BaseMySqlParserListener) ExitDeclareCondition(ctx *DeclareConditionContext) {}

// EnterDeclareCursor is called when production declareCursor is entered.
func (s *BaseMySqlParserListener) EnterDeclareCursor(ctx *DeclareCursorContext) {}

// ExitDeclareCursor is called when production declareCursor is exited.
func (s *BaseMySqlParserListener) ExitDeclareCursor(ctx *DeclareCursorContext) {}

// EnterDeclareHandler is called when production declareHandler is entered.
func (s *BaseMySqlParserListener) EnterDeclareHandler(ctx *DeclareHandlerContext) {}

// ExitDeclareHandler is called when production declareHandler is exited.
func (s *BaseMySqlParserListener) ExitDeclareHandler(ctx *DeclareHandlerContext) {}

// EnterHandlerConditionCode is called when production handlerConditionCode is entered.
func (s *BaseMySqlParserListener) EnterHandlerConditionCode(ctx *HandlerConditionCodeContext) {}

// ExitHandlerConditionCode is called when production handlerConditionCode is exited.
func (s *BaseMySqlParserListener) ExitHandlerConditionCode(ctx *HandlerConditionCodeContext) {}

// EnterHandlerConditionState is called when production handlerConditionState is entered.
func (s *BaseMySqlParserListener) EnterHandlerConditionState(ctx *HandlerConditionStateContext) {}

// ExitHandlerConditionState is called when production handlerConditionState is exited.
func (s *BaseMySqlParserListener) ExitHandlerConditionState(ctx *HandlerConditionStateContext) {}

// EnterHandlerConditionName is called when production handlerConditionName is entered.
func (s *BaseMySqlParserListener) EnterHandlerConditionName(ctx *HandlerConditionNameContext) {}

// ExitHandlerConditionName is called when production handlerConditionName is exited.
func (s *BaseMySqlParserListener) ExitHandlerConditionName(ctx *HandlerConditionNameContext) {}

// EnterHandlerConditionWarning is called when production handlerConditionWarning is entered.
func (s *BaseMySqlParserListener) EnterHandlerConditionWarning(ctx *HandlerConditionWarningContext) {}

// ExitHandlerConditionWarning is called when production handlerConditionWarning is exited.
func (s *BaseMySqlParserListener) ExitHandlerConditionWarning(ctx *HandlerConditionWarningContext) {}

// EnterHandlerConditionNotfound is called when production handlerConditionNotfound is entered.
func (s *BaseMySqlParserListener) EnterHandlerConditionNotfound(ctx *HandlerConditionNotfoundContext) {
}

// ExitHandlerConditionNotfound is called when production handlerConditionNotfound is exited.
func (s *BaseMySqlParserListener) ExitHandlerConditionNotfound(ctx *HandlerConditionNotfoundContext) {
}

// EnterHandlerConditionException is called when production handlerConditionException is entered.
func (s *BaseMySqlParserListener) EnterHandlerConditionException(ctx *HandlerConditionExceptionContext) {
}

// ExitHandlerConditionException is called when production handlerConditionException is exited.
func (s *BaseMySqlParserListener) ExitHandlerConditionException(ctx *HandlerConditionExceptionContext) {
}

// EnterProcedureSqlStatement is called when production procedureSqlStatement is entered.
func (s *BaseMySqlParserListener) EnterProcedureSqlStatement(ctx *ProcedureSqlStatementContext) {}

// ExitProcedureSqlStatement is called when production procedureSqlStatement is exited.
func (s *BaseMySqlParserListener) ExitProcedureSqlStatement(ctx *ProcedureSqlStatementContext) {}

// EnterCaseAlternative is called when production caseAlternative is entered.
func (s *BaseMySqlParserListener) EnterCaseAlternative(ctx *CaseAlternativeContext) {}

// ExitCaseAlternative is called when production caseAlternative is exited.
func (s *BaseMySqlParserListener) ExitCaseAlternative(ctx *CaseAlternativeContext) {}

// EnterElifAlternative is called when production elifAlternative is entered.
func (s *BaseMySqlParserListener) EnterElifAlternative(ctx *ElifAlternativeContext) {}

// ExitElifAlternative is called when production elifAlternative is exited.
func (s *BaseMySqlParserListener) ExitElifAlternative(ctx *ElifAlternativeContext) {}

// EnterAlterUserMysqlV56 is called when production alterUserMysqlV56 is entered.
func (s *BaseMySqlParserListener) EnterAlterUserMysqlV56(ctx *AlterUserMysqlV56Context) {}

// ExitAlterUserMysqlV56 is called when production alterUserMysqlV56 is exited.
func (s *BaseMySqlParserListener) ExitAlterUserMysqlV56(ctx *AlterUserMysqlV56Context) {}

// EnterAlterUserMysqlV80 is called when production alterUserMysqlV80 is entered.
func (s *BaseMySqlParserListener) EnterAlterUserMysqlV80(ctx *AlterUserMysqlV80Context) {}

// ExitAlterUserMysqlV80 is called when production alterUserMysqlV80 is exited.
func (s *BaseMySqlParserListener) ExitAlterUserMysqlV80(ctx *AlterUserMysqlV80Context) {}

// EnterCreateUserMysqlV56 is called when production createUserMysqlV56 is entered.
func (s *BaseMySqlParserListener) EnterCreateUserMysqlV56(ctx *CreateUserMysqlV56Context) {}

// ExitCreateUserMysqlV56 is called when production createUserMysqlV56 is exited.
func (s *BaseMySqlParserListener) ExitCreateUserMysqlV56(ctx *CreateUserMysqlV56Context) {}

// EnterCreateUserMysqlV80 is called when production createUserMysqlV80 is entered.
func (s *BaseMySqlParserListener) EnterCreateUserMysqlV80(ctx *CreateUserMysqlV80Context) {}

// ExitCreateUserMysqlV80 is called when production createUserMysqlV80 is exited.
func (s *BaseMySqlParserListener) ExitCreateUserMysqlV80(ctx *CreateUserMysqlV80Context) {}

// EnterDropUser is called when production dropUser is entered.
func (s *BaseMySqlParserListener) EnterDropUser(ctx *DropUserContext) {}

// ExitDropUser is called when production dropUser is exited.
func (s *BaseMySqlParserListener) ExitDropUser(ctx *DropUserContext) {}

// EnterGrantStatement is called when production grantStatement is entered.
func (s *BaseMySqlParserListener) EnterGrantStatement(ctx *GrantStatementContext) {}

// ExitGrantStatement is called when production grantStatement is exited.
func (s *BaseMySqlParserListener) ExitGrantStatement(ctx *GrantStatementContext) {}

// EnterRoleOption is called when production roleOption is entered.
func (s *BaseMySqlParserListener) EnterRoleOption(ctx *RoleOptionContext) {}

// ExitRoleOption is called when production roleOption is exited.
func (s *BaseMySqlParserListener) ExitRoleOption(ctx *RoleOptionContext) {}

// EnterGrantProxy is called when production grantProxy is entered.
func (s *BaseMySqlParserListener) EnterGrantProxy(ctx *GrantProxyContext) {}

// ExitGrantProxy is called when production grantProxy is exited.
func (s *BaseMySqlParserListener) ExitGrantProxy(ctx *GrantProxyContext) {}

// EnterRenameUser is called when production renameUser is entered.
func (s *BaseMySqlParserListener) EnterRenameUser(ctx *RenameUserContext) {}

// ExitRenameUser is called when production renameUser is exited.
func (s *BaseMySqlParserListener) ExitRenameUser(ctx *RenameUserContext) {}

// EnterDetailRevoke is called when production detailRevoke is entered.
func (s *BaseMySqlParserListener) EnterDetailRevoke(ctx *DetailRevokeContext) {}

// ExitDetailRevoke is called when production detailRevoke is exited.
func (s *BaseMySqlParserListener) ExitDetailRevoke(ctx *DetailRevokeContext) {}

// EnterShortRevoke is called when production shortRevoke is entered.
func (s *BaseMySqlParserListener) EnterShortRevoke(ctx *ShortRevokeContext) {}

// ExitShortRevoke is called when production shortRevoke is exited.
func (s *BaseMySqlParserListener) ExitShortRevoke(ctx *ShortRevokeContext) {}

// EnterRoleRevoke is called when production roleRevoke is entered.
func (s *BaseMySqlParserListener) EnterRoleRevoke(ctx *RoleRevokeContext) {}

// ExitRoleRevoke is called when production roleRevoke is exited.
func (s *BaseMySqlParserListener) ExitRoleRevoke(ctx *RoleRevokeContext) {}

// EnterRevokeProxy is called when production revokeProxy is entered.
func (s *BaseMySqlParserListener) EnterRevokeProxy(ctx *RevokeProxyContext) {}

// ExitRevokeProxy is called when production revokeProxy is exited.
func (s *BaseMySqlParserListener) ExitRevokeProxy(ctx *RevokeProxyContext) {}

// EnterSetPasswordStatement is called when production setPasswordStatement is entered.
func (s *BaseMySqlParserListener) EnterSetPasswordStatement(ctx *SetPasswordStatementContext) {}

// ExitSetPasswordStatement is called when production setPasswordStatement is exited.
func (s *BaseMySqlParserListener) ExitSetPasswordStatement(ctx *SetPasswordStatementContext) {}

// EnterUserSpecification is called when production userSpecification is entered.
func (s *BaseMySqlParserListener) EnterUserSpecification(ctx *UserSpecificationContext) {}

// ExitUserSpecification is called when production userSpecification is exited.
func (s *BaseMySqlParserListener) ExitUserSpecification(ctx *UserSpecificationContext) {}

// EnterHashAuthOption is called when production hashAuthOption is entered.
func (s *BaseMySqlParserListener) EnterHashAuthOption(ctx *HashAuthOptionContext) {}

// ExitHashAuthOption is called when production hashAuthOption is exited.
func (s *BaseMySqlParserListener) ExitHashAuthOption(ctx *HashAuthOptionContext) {}

// EnterRandomAuthOption is called when production randomAuthOption is entered.
func (s *BaseMySqlParserListener) EnterRandomAuthOption(ctx *RandomAuthOptionContext) {}

// ExitRandomAuthOption is called when production randomAuthOption is exited.
func (s *BaseMySqlParserListener) ExitRandomAuthOption(ctx *RandomAuthOptionContext) {}

// EnterStringAuthOption is called when production stringAuthOption is entered.
func (s *BaseMySqlParserListener) EnterStringAuthOption(ctx *StringAuthOptionContext) {}

// ExitStringAuthOption is called when production stringAuthOption is exited.
func (s *BaseMySqlParserListener) ExitStringAuthOption(ctx *StringAuthOptionContext) {}

// EnterModuleAuthOption is called when production moduleAuthOption is entered.
func (s *BaseMySqlParserListener) EnterModuleAuthOption(ctx *ModuleAuthOptionContext) {}

// ExitModuleAuthOption is called when production moduleAuthOption is exited.
func (s *BaseMySqlParserListener) ExitModuleAuthOption(ctx *ModuleAuthOptionContext) {}

// EnterSimpleAuthOption is called when production simpleAuthOption is entered.
func (s *BaseMySqlParserListener) EnterSimpleAuthOption(ctx *SimpleAuthOptionContext) {}

// ExitSimpleAuthOption is called when production simpleAuthOption is exited.
func (s *BaseMySqlParserListener) ExitSimpleAuthOption(ctx *SimpleAuthOptionContext) {}

// EnterAuthOptionClause is called when production authOptionClause is entered.
func (s *BaseMySqlParserListener) EnterAuthOptionClause(ctx *AuthOptionClauseContext) {}

// ExitAuthOptionClause is called when production authOptionClause is exited.
func (s *BaseMySqlParserListener) ExitAuthOptionClause(ctx *AuthOptionClauseContext) {}

// EnterModule is called when production module is entered.
func (s *BaseMySqlParserListener) EnterModule(ctx *ModuleContext) {}

// ExitModule is called when production module is exited.
func (s *BaseMySqlParserListener) ExitModule(ctx *ModuleContext) {}

// EnterPasswordModuleOption is called when production passwordModuleOption is entered.
func (s *BaseMySqlParserListener) EnterPasswordModuleOption(ctx *PasswordModuleOptionContext) {}

// ExitPasswordModuleOption is called when production passwordModuleOption is exited.
func (s *BaseMySqlParserListener) ExitPasswordModuleOption(ctx *PasswordModuleOptionContext) {}

// EnterTlsOption is called when production tlsOption is entered.
func (s *BaseMySqlParserListener) EnterTlsOption(ctx *TlsOptionContext) {}

// ExitTlsOption is called when production tlsOption is exited.
func (s *BaseMySqlParserListener) ExitTlsOption(ctx *TlsOptionContext) {}

// EnterUserResourceOption is called when production userResourceOption is entered.
func (s *BaseMySqlParserListener) EnterUserResourceOption(ctx *UserResourceOptionContext) {}

// ExitUserResourceOption is called when production userResourceOption is exited.
func (s *BaseMySqlParserListener) ExitUserResourceOption(ctx *UserResourceOptionContext) {}

// EnterUserPasswordOption is called when production userPasswordOption is entered.
func (s *BaseMySqlParserListener) EnterUserPasswordOption(ctx *UserPasswordOptionContext) {}

// ExitUserPasswordOption is called when production userPasswordOption is exited.
func (s *BaseMySqlParserListener) ExitUserPasswordOption(ctx *UserPasswordOptionContext) {}

// EnterUserLockOption is called when production userLockOption is entered.
func (s *BaseMySqlParserListener) EnterUserLockOption(ctx *UserLockOptionContext) {}

// ExitUserLockOption is called when production userLockOption is exited.
func (s *BaseMySqlParserListener) ExitUserLockOption(ctx *UserLockOptionContext) {}

// EnterPrivelegeClause is called when production privelegeClause is entered.
func (s *BaseMySqlParserListener) EnterPrivelegeClause(ctx *PrivelegeClauseContext) {}

// ExitPrivelegeClause is called when production privelegeClause is exited.
func (s *BaseMySqlParserListener) ExitPrivelegeClause(ctx *PrivelegeClauseContext) {}

// EnterPrivilege is called when production privilege is entered.
func (s *BaseMySqlParserListener) EnterPrivilege(ctx *PrivilegeContext) {}

// ExitPrivilege is called when production privilege is exited.
func (s *BaseMySqlParserListener) ExitPrivilege(ctx *PrivilegeContext) {}

// EnterCurrentSchemaPriviLevel is called when production currentSchemaPriviLevel is entered.
func (s *BaseMySqlParserListener) EnterCurrentSchemaPriviLevel(ctx *CurrentSchemaPriviLevelContext) {}

// ExitCurrentSchemaPriviLevel is called when production currentSchemaPriviLevel is exited.
func (s *BaseMySqlParserListener) ExitCurrentSchemaPriviLevel(ctx *CurrentSchemaPriviLevelContext) {}

// EnterGlobalPrivLevel is called when production globalPrivLevel is entered.
func (s *BaseMySqlParserListener) EnterGlobalPrivLevel(ctx *GlobalPrivLevelContext) {}

// ExitGlobalPrivLevel is called when production globalPrivLevel is exited.
func (s *BaseMySqlParserListener) ExitGlobalPrivLevel(ctx *GlobalPrivLevelContext) {}

// EnterDefiniteSchemaPrivLevel is called when production definiteSchemaPrivLevel is entered.
func (s *BaseMySqlParserListener) EnterDefiniteSchemaPrivLevel(ctx *DefiniteSchemaPrivLevelContext) {}

// ExitDefiniteSchemaPrivLevel is called when production definiteSchemaPrivLevel is exited.
func (s *BaseMySqlParserListener) ExitDefiniteSchemaPrivLevel(ctx *DefiniteSchemaPrivLevelContext) {}

// EnterDefiniteFullTablePrivLevel is called when production definiteFullTablePrivLevel is entered.
func (s *BaseMySqlParserListener) EnterDefiniteFullTablePrivLevel(ctx *DefiniteFullTablePrivLevelContext) {
}

// ExitDefiniteFullTablePrivLevel is called when production definiteFullTablePrivLevel is exited.
func (s *BaseMySqlParserListener) ExitDefiniteFullTablePrivLevel(ctx *DefiniteFullTablePrivLevelContext) {
}

// EnterDefiniteFullTablePrivLevel2 is called when production definiteFullTablePrivLevel2 is entered.
func (s *BaseMySqlParserListener) EnterDefiniteFullTablePrivLevel2(ctx *DefiniteFullTablePrivLevel2Context) {
}

// ExitDefiniteFullTablePrivLevel2 is called when production definiteFullTablePrivLevel2 is exited.
func (s *BaseMySqlParserListener) ExitDefiniteFullTablePrivLevel2(ctx *DefiniteFullTablePrivLevel2Context) {
}

// EnterDefiniteTablePrivLevel is called when production definiteTablePrivLevel is entered.
func (s *BaseMySqlParserListener) EnterDefiniteTablePrivLevel(ctx *DefiniteTablePrivLevelContext) {}

// ExitDefiniteTablePrivLevel is called when production definiteTablePrivLevel is exited.
func (s *BaseMySqlParserListener) ExitDefiniteTablePrivLevel(ctx *DefiniteTablePrivLevelContext) {}

// EnterRenameUserClause is called when production renameUserClause is entered.
func (s *BaseMySqlParserListener) EnterRenameUserClause(ctx *RenameUserClauseContext) {}

// ExitRenameUserClause is called when production renameUserClause is exited.
func (s *BaseMySqlParserListener) ExitRenameUserClause(ctx *RenameUserClauseContext) {}

// EnterAnalyzeTable is called when production analyzeTable is entered.
func (s *BaseMySqlParserListener) EnterAnalyzeTable(ctx *AnalyzeTableContext) {}

// ExitAnalyzeTable is called when production analyzeTable is exited.
func (s *BaseMySqlParserListener) ExitAnalyzeTable(ctx *AnalyzeTableContext) {}

// EnterCheckTable is called when production checkTable is entered.
func (s *BaseMySqlParserListener) EnterCheckTable(ctx *CheckTableContext) {}

// ExitCheckTable is called when production checkTable is exited.
func (s *BaseMySqlParserListener) ExitCheckTable(ctx *CheckTableContext) {}

// EnterChecksumTable is called when production checksumTable is entered.
func (s *BaseMySqlParserListener) EnterChecksumTable(ctx *ChecksumTableContext) {}

// ExitChecksumTable is called when production checksumTable is exited.
func (s *BaseMySqlParserListener) ExitChecksumTable(ctx *ChecksumTableContext) {}

// EnterOptimizeTable is called when production optimizeTable is entered.
func (s *BaseMySqlParserListener) EnterOptimizeTable(ctx *OptimizeTableContext) {}

// ExitOptimizeTable is called when production optimizeTable is exited.
func (s *BaseMySqlParserListener) ExitOptimizeTable(ctx *OptimizeTableContext) {}

// EnterRepairTable is called when production repairTable is entered.
func (s *BaseMySqlParserListener) EnterRepairTable(ctx *RepairTableContext) {}

// ExitRepairTable is called when production repairTable is exited.
func (s *BaseMySqlParserListener) ExitRepairTable(ctx *RepairTableContext) {}

// EnterCheckTableOption is called when production checkTableOption is entered.
func (s *BaseMySqlParserListener) EnterCheckTableOption(ctx *CheckTableOptionContext) {}

// ExitCheckTableOption is called when production checkTableOption is exited.
func (s *BaseMySqlParserListener) ExitCheckTableOption(ctx *CheckTableOptionContext) {}

// EnterCreateUdfunction is called when production createUdfunction is entered.
func (s *BaseMySqlParserListener) EnterCreateUdfunction(ctx *CreateUdfunctionContext) {}

// ExitCreateUdfunction is called when production createUdfunction is exited.
func (s *BaseMySqlParserListener) ExitCreateUdfunction(ctx *CreateUdfunctionContext) {}

// EnterInstallPlugin is called when production installPlugin is entered.
func (s *BaseMySqlParserListener) EnterInstallPlugin(ctx *InstallPluginContext) {}

// ExitInstallPlugin is called when production installPlugin is exited.
func (s *BaseMySqlParserListener) ExitInstallPlugin(ctx *InstallPluginContext) {}

// EnterUninstallPlugin is called when production uninstallPlugin is entered.
func (s *BaseMySqlParserListener) EnterUninstallPlugin(ctx *UninstallPluginContext) {}

// ExitUninstallPlugin is called when production uninstallPlugin is exited.
func (s *BaseMySqlParserListener) ExitUninstallPlugin(ctx *UninstallPluginContext) {}

// EnterSetVariable is called when production setVariable is entered.
func (s *BaseMySqlParserListener) EnterSetVariable(ctx *SetVariableContext) {}

// ExitSetVariable is called when production setVariable is exited.
func (s *BaseMySqlParserListener) ExitSetVariable(ctx *SetVariableContext) {}

// EnterSetCharset is called when production setCharset is entered.
func (s *BaseMySqlParserListener) EnterSetCharset(ctx *SetCharsetContext) {}

// ExitSetCharset is called when production setCharset is exited.
func (s *BaseMySqlParserListener) ExitSetCharset(ctx *SetCharsetContext) {}

// EnterSetNames is called when production setNames is entered.
func (s *BaseMySqlParserListener) EnterSetNames(ctx *SetNamesContext) {}

// ExitSetNames is called when production setNames is exited.
func (s *BaseMySqlParserListener) ExitSetNames(ctx *SetNamesContext) {}

// EnterSetPassword is called when production setPassword is entered.
func (s *BaseMySqlParserListener) EnterSetPassword(ctx *SetPasswordContext) {}

// ExitSetPassword is called when production setPassword is exited.
func (s *BaseMySqlParserListener) ExitSetPassword(ctx *SetPasswordContext) {}

// EnterSetTransaction is called when production setTransaction is entered.
func (s *BaseMySqlParserListener) EnterSetTransaction(ctx *SetTransactionContext) {}

// ExitSetTransaction is called when production setTransaction is exited.
func (s *BaseMySqlParserListener) ExitSetTransaction(ctx *SetTransactionContext) {}

// EnterSetAutocommit is called when production setAutocommit is entered.
func (s *BaseMySqlParserListener) EnterSetAutocommit(ctx *SetAutocommitContext) {}

// ExitSetAutocommit is called when production setAutocommit is exited.
func (s *BaseMySqlParserListener) ExitSetAutocommit(ctx *SetAutocommitContext) {}

// EnterSetNewValueInsideTrigger is called when production setNewValueInsideTrigger is entered.
func (s *BaseMySqlParserListener) EnterSetNewValueInsideTrigger(ctx *SetNewValueInsideTriggerContext) {
}

// ExitSetNewValueInsideTrigger is called when production setNewValueInsideTrigger is exited.
func (s *BaseMySqlParserListener) ExitSetNewValueInsideTrigger(ctx *SetNewValueInsideTriggerContext) {
}

// EnterShowMasterLogs is called when production showMasterLogs is entered.
func (s *BaseMySqlParserListener) EnterShowMasterLogs(ctx *ShowMasterLogsContext) {}

// ExitShowMasterLogs is called when production showMasterLogs is exited.
func (s *BaseMySqlParserListener) ExitShowMasterLogs(ctx *ShowMasterLogsContext) {}

// EnterShowLogEvents is called when production showLogEvents is entered.
func (s *BaseMySqlParserListener) EnterShowLogEvents(ctx *ShowLogEventsContext) {}

// ExitShowLogEvents is called when production showLogEvents is exited.
func (s *BaseMySqlParserListener) ExitShowLogEvents(ctx *ShowLogEventsContext) {}

// EnterShowObjectFilter is called when production showObjectFilter is entered.
func (s *BaseMySqlParserListener) EnterShowObjectFilter(ctx *ShowObjectFilterContext) {}

// ExitShowObjectFilter is called when production showObjectFilter is exited.
func (s *BaseMySqlParserListener) ExitShowObjectFilter(ctx *ShowObjectFilterContext) {}

// EnterShowColumns is called when production showColumns is entered.
func (s *BaseMySqlParserListener) EnterShowColumns(ctx *ShowColumnsContext) {}

// ExitShowColumns is called when production showColumns is exited.
func (s *BaseMySqlParserListener) ExitShowColumns(ctx *ShowColumnsContext) {}

// EnterShowCreateDb is called when production showCreateDb is entered.
func (s *BaseMySqlParserListener) EnterShowCreateDb(ctx *ShowCreateDbContext) {}

// ExitShowCreateDb is called when production showCreateDb is exited.
func (s *BaseMySqlParserListener) ExitShowCreateDb(ctx *ShowCreateDbContext) {}

// EnterShowCreateFullIdObject is called when production showCreateFullIdObject is entered.
func (s *BaseMySqlParserListener) EnterShowCreateFullIdObject(ctx *ShowCreateFullIdObjectContext) {}

// ExitShowCreateFullIdObject is called when production showCreateFullIdObject is exited.
func (s *BaseMySqlParserListener) ExitShowCreateFullIdObject(ctx *ShowCreateFullIdObjectContext) {}

// EnterShowCreateUser is called when production showCreateUser is entered.
func (s *BaseMySqlParserListener) EnterShowCreateUser(ctx *ShowCreateUserContext) {}

// ExitShowCreateUser is called when production showCreateUser is exited.
func (s *BaseMySqlParserListener) ExitShowCreateUser(ctx *ShowCreateUserContext) {}

// EnterShowEngine is called when production showEngine is entered.
func (s *BaseMySqlParserListener) EnterShowEngine(ctx *ShowEngineContext) {}

// ExitShowEngine is called when production showEngine is exited.
func (s *BaseMySqlParserListener) ExitShowEngine(ctx *ShowEngineContext) {}

// EnterShowGlobalInfo is called when production showGlobalInfo is entered.
func (s *BaseMySqlParserListener) EnterShowGlobalInfo(ctx *ShowGlobalInfoContext) {}

// ExitShowGlobalInfo is called when production showGlobalInfo is exited.
func (s *BaseMySqlParserListener) ExitShowGlobalInfo(ctx *ShowGlobalInfoContext) {}

// EnterShowErrors is called when production showErrors is entered.
func (s *BaseMySqlParserListener) EnterShowErrors(ctx *ShowErrorsContext) {}

// ExitShowErrors is called when production showErrors is exited.
func (s *BaseMySqlParserListener) ExitShowErrors(ctx *ShowErrorsContext) {}

// EnterShowCountErrors is called when production showCountErrors is entered.
func (s *BaseMySqlParserListener) EnterShowCountErrors(ctx *ShowCountErrorsContext) {}

// ExitShowCountErrors is called when production showCountErrors is exited.
func (s *BaseMySqlParserListener) ExitShowCountErrors(ctx *ShowCountErrorsContext) {}

// EnterShowSchemaFilter is called when production showSchemaFilter is entered.
func (s *BaseMySqlParserListener) EnterShowSchemaFilter(ctx *ShowSchemaFilterContext) {}

// ExitShowSchemaFilter is called when production showSchemaFilter is exited.
func (s *BaseMySqlParserListener) ExitShowSchemaFilter(ctx *ShowSchemaFilterContext) {}

// EnterShowRoutine is called when production showRoutine is entered.
func (s *BaseMySqlParserListener) EnterShowRoutine(ctx *ShowRoutineContext) {}

// ExitShowRoutine is called when production showRoutine is exited.
func (s *BaseMySqlParserListener) ExitShowRoutine(ctx *ShowRoutineContext) {}

// EnterShowGrants is called when production showGrants is entered.
func (s *BaseMySqlParserListener) EnterShowGrants(ctx *ShowGrantsContext) {}

// ExitShowGrants is called when production showGrants is exited.
func (s *BaseMySqlParserListener) ExitShowGrants(ctx *ShowGrantsContext) {}

// EnterShowIndexes is called when production showIndexes is entered.
func (s *BaseMySqlParserListener) EnterShowIndexes(ctx *ShowIndexesContext) {}

// ExitShowIndexes is called when production showIndexes is exited.
func (s *BaseMySqlParserListener) ExitShowIndexes(ctx *ShowIndexesContext) {}

// EnterShowOpenTables is called when production showOpenTables is entered.
func (s *BaseMySqlParserListener) EnterShowOpenTables(ctx *ShowOpenTablesContext) {}

// ExitShowOpenTables is called when production showOpenTables is exited.
func (s *BaseMySqlParserListener) ExitShowOpenTables(ctx *ShowOpenTablesContext) {}

// EnterShowProfile is called when production showProfile is entered.
func (s *BaseMySqlParserListener) EnterShowProfile(ctx *ShowProfileContext) {}

// ExitShowProfile is called when production showProfile is exited.
func (s *BaseMySqlParserListener) ExitShowProfile(ctx *ShowProfileContext) {}

// EnterShowSlaveStatus is called when production showSlaveStatus is entered.
func (s *BaseMySqlParserListener) EnterShowSlaveStatus(ctx *ShowSlaveStatusContext) {}

// ExitShowSlaveStatus is called when production showSlaveStatus is exited.
func (s *BaseMySqlParserListener) ExitShowSlaveStatus(ctx *ShowSlaveStatusContext) {}

// EnterVariableClause is called when production variableClause is entered.
func (s *BaseMySqlParserListener) EnterVariableClause(ctx *VariableClauseContext) {}

// ExitVariableClause is called when production variableClause is exited.
func (s *BaseMySqlParserListener) ExitVariableClause(ctx *VariableClauseContext) {}

// EnterShowCommonEntity is called when production showCommonEntity is entered.
func (s *BaseMySqlParserListener) EnterShowCommonEntity(ctx *ShowCommonEntityContext) {}

// ExitShowCommonEntity is called when production showCommonEntity is exited.
func (s *BaseMySqlParserListener) ExitShowCommonEntity(ctx *ShowCommonEntityContext) {}

// EnterShowFilter is called when production showFilter is entered.
func (s *BaseMySqlParserListener) EnterShowFilter(ctx *ShowFilterContext) {}

// ExitShowFilter is called when production showFilter is exited.
func (s *BaseMySqlParserListener) ExitShowFilter(ctx *ShowFilterContext) {}

// EnterShowGlobalInfoClause is called when production showGlobalInfoClause is entered.
func (s *BaseMySqlParserListener) EnterShowGlobalInfoClause(ctx *ShowGlobalInfoClauseContext) {}

// ExitShowGlobalInfoClause is called when production showGlobalInfoClause is exited.
func (s *BaseMySqlParserListener) ExitShowGlobalInfoClause(ctx *ShowGlobalInfoClauseContext) {}

// EnterShowSchemaEntity is called when production showSchemaEntity is entered.
func (s *BaseMySqlParserListener) EnterShowSchemaEntity(ctx *ShowSchemaEntityContext) {}

// ExitShowSchemaEntity is called when production showSchemaEntity is exited.
func (s *BaseMySqlParserListener) ExitShowSchemaEntity(ctx *ShowSchemaEntityContext) {}

// EnterShowProfileType is called when production showProfileType is entered.
func (s *BaseMySqlParserListener) EnterShowProfileType(ctx *ShowProfileTypeContext) {}

// ExitShowProfileType is called when production showProfileType is exited.
func (s *BaseMySqlParserListener) ExitShowProfileType(ctx *ShowProfileTypeContext) {}

// EnterBinlogStatement is called when production binlogStatement is entered.
func (s *BaseMySqlParserListener) EnterBinlogStatement(ctx *BinlogStatementContext) {}

// ExitBinlogStatement is called when production binlogStatement is exited.
func (s *BaseMySqlParserListener) ExitBinlogStatement(ctx *BinlogStatementContext) {}

// EnterCacheIndexStatement is called when production cacheIndexStatement is entered.
func (s *BaseMySqlParserListener) EnterCacheIndexStatement(ctx *CacheIndexStatementContext) {}

// ExitCacheIndexStatement is called when production cacheIndexStatement is exited.
func (s *BaseMySqlParserListener) ExitCacheIndexStatement(ctx *CacheIndexStatementContext) {}

// EnterFlushStatement is called when production flushStatement is entered.
func (s *BaseMySqlParserListener) EnterFlushStatement(ctx *FlushStatementContext) {}

// ExitFlushStatement is called when production flushStatement is exited.
func (s *BaseMySqlParserListener) ExitFlushStatement(ctx *FlushStatementContext) {}

// EnterKillStatement is called when production killStatement is entered.
func (s *BaseMySqlParserListener) EnterKillStatement(ctx *KillStatementContext) {}

// ExitKillStatement is called when production killStatement is exited.
func (s *BaseMySqlParserListener) ExitKillStatement(ctx *KillStatementContext) {}

// EnterLoadIndexIntoCache is called when production loadIndexIntoCache is entered.
func (s *BaseMySqlParserListener) EnterLoadIndexIntoCache(ctx *LoadIndexIntoCacheContext) {}

// ExitLoadIndexIntoCache is called when production loadIndexIntoCache is exited.
func (s *BaseMySqlParserListener) ExitLoadIndexIntoCache(ctx *LoadIndexIntoCacheContext) {}

// EnterResetStatement is called when production resetStatement is entered.
func (s *BaseMySqlParserListener) EnterResetStatement(ctx *ResetStatementContext) {}

// ExitResetStatement is called when production resetStatement is exited.
func (s *BaseMySqlParserListener) ExitResetStatement(ctx *ResetStatementContext) {}

// EnterShutdownStatement is called when production shutdownStatement is entered.
func (s *BaseMySqlParserListener) EnterShutdownStatement(ctx *ShutdownStatementContext) {}

// ExitShutdownStatement is called when production shutdownStatement is exited.
func (s *BaseMySqlParserListener) ExitShutdownStatement(ctx *ShutdownStatementContext) {}

// EnterTableIndexes is called when production tableIndexes is entered.
func (s *BaseMySqlParserListener) EnterTableIndexes(ctx *TableIndexesContext) {}

// ExitTableIndexes is called when production tableIndexes is exited.
func (s *BaseMySqlParserListener) ExitTableIndexes(ctx *TableIndexesContext) {}

// EnterSimpleFlushOption is called when production simpleFlushOption is entered.
func (s *BaseMySqlParserListener) EnterSimpleFlushOption(ctx *SimpleFlushOptionContext) {}

// ExitSimpleFlushOption is called when production simpleFlushOption is exited.
func (s *BaseMySqlParserListener) ExitSimpleFlushOption(ctx *SimpleFlushOptionContext) {}

// EnterChannelFlushOption is called when production channelFlushOption is entered.
func (s *BaseMySqlParserListener) EnterChannelFlushOption(ctx *ChannelFlushOptionContext) {}

// ExitChannelFlushOption is called when production channelFlushOption is exited.
func (s *BaseMySqlParserListener) ExitChannelFlushOption(ctx *ChannelFlushOptionContext) {}

// EnterTableFlushOption is called when production tableFlushOption is entered.
func (s *BaseMySqlParserListener) EnterTableFlushOption(ctx *TableFlushOptionContext) {}

// ExitTableFlushOption is called when production tableFlushOption is exited.
func (s *BaseMySqlParserListener) ExitTableFlushOption(ctx *TableFlushOptionContext) {}

// EnterFlushTableOption is called when production flushTableOption is entered.
func (s *BaseMySqlParserListener) EnterFlushTableOption(ctx *FlushTableOptionContext) {}

// ExitFlushTableOption is called when production flushTableOption is exited.
func (s *BaseMySqlParserListener) ExitFlushTableOption(ctx *FlushTableOptionContext) {}

// EnterLoadedTableIndexes is called when production loadedTableIndexes is entered.
func (s *BaseMySqlParserListener) EnterLoadedTableIndexes(ctx *LoadedTableIndexesContext) {}

// ExitLoadedTableIndexes is called when production loadedTableIndexes is exited.
func (s *BaseMySqlParserListener) ExitLoadedTableIndexes(ctx *LoadedTableIndexesContext) {}

// EnterSimpleDescribeStatement is called when production simpleDescribeStatement is entered.
func (s *BaseMySqlParserListener) EnterSimpleDescribeStatement(ctx *SimpleDescribeStatementContext) {}

// ExitSimpleDescribeStatement is called when production simpleDescribeStatement is exited.
func (s *BaseMySqlParserListener) ExitSimpleDescribeStatement(ctx *SimpleDescribeStatementContext) {}

// EnterFullDescribeStatement is called when production fullDescribeStatement is entered.
func (s *BaseMySqlParserListener) EnterFullDescribeStatement(ctx *FullDescribeStatementContext) {}

// ExitFullDescribeStatement is called when production fullDescribeStatement is exited.
func (s *BaseMySqlParserListener) ExitFullDescribeStatement(ctx *FullDescribeStatementContext) {}

// EnterHelpStatement is called when production helpStatement is entered.
func (s *BaseMySqlParserListener) EnterHelpStatement(ctx *HelpStatementContext) {}

// ExitHelpStatement is called when production helpStatement is exited.
func (s *BaseMySqlParserListener) ExitHelpStatement(ctx *HelpStatementContext) {}

// EnterUseStatement is called when production useStatement is entered.
func (s *BaseMySqlParserListener) EnterUseStatement(ctx *UseStatementContext) {}

// ExitUseStatement is called when production useStatement is exited.
func (s *BaseMySqlParserListener) ExitUseStatement(ctx *UseStatementContext) {}

// EnterSignalStatement is called when production signalStatement is entered.
func (s *BaseMySqlParserListener) EnterSignalStatement(ctx *SignalStatementContext) {}

// ExitSignalStatement is called when production signalStatement is exited.
func (s *BaseMySqlParserListener) ExitSignalStatement(ctx *SignalStatementContext) {}

// EnterResignalStatement is called when production resignalStatement is entered.
func (s *BaseMySqlParserListener) EnterResignalStatement(ctx *ResignalStatementContext) {}

// ExitResignalStatement is called when production resignalStatement is exited.
func (s *BaseMySqlParserListener) ExitResignalStatement(ctx *ResignalStatementContext) {}

// EnterSignalConditionInformation is called when production signalConditionInformation is entered.
func (s *BaseMySqlParserListener) EnterSignalConditionInformation(ctx *SignalConditionInformationContext) {
}

// ExitSignalConditionInformation is called when production signalConditionInformation is exited.
func (s *BaseMySqlParserListener) ExitSignalConditionInformation(ctx *SignalConditionInformationContext) {
}

// EnterWithStatement is called when production withStatement is entered.
func (s *BaseMySqlParserListener) EnterWithStatement(ctx *WithStatementContext) {}

// ExitWithStatement is called when production withStatement is exited.
func (s *BaseMySqlParserListener) ExitWithStatement(ctx *WithStatementContext) {}

// EnterTableStatement is called when production tableStatement is entered.
func (s *BaseMySqlParserListener) EnterTableStatement(ctx *TableStatementContext) {}

// ExitTableStatement is called when production tableStatement is exited.
func (s *BaseMySqlParserListener) ExitTableStatement(ctx *TableStatementContext) {}

// EnterDiagnosticsStatement is called when production diagnosticsStatement is entered.
func (s *BaseMySqlParserListener) EnterDiagnosticsStatement(ctx *DiagnosticsStatementContext) {}

// ExitDiagnosticsStatement is called when production diagnosticsStatement is exited.
func (s *BaseMySqlParserListener) ExitDiagnosticsStatement(ctx *DiagnosticsStatementContext) {}

// EnterDiagnosticsConditionInformationName is called when production diagnosticsConditionInformationName is entered.
func (s *BaseMySqlParserListener) EnterDiagnosticsConditionInformationName(ctx *DiagnosticsConditionInformationNameContext) {
}

// ExitDiagnosticsConditionInformationName is called when production diagnosticsConditionInformationName is exited.
func (s *BaseMySqlParserListener) ExitDiagnosticsConditionInformationName(ctx *DiagnosticsConditionInformationNameContext) {
}

// EnterDescribeStatements is called when production describeStatements is entered.
func (s *BaseMySqlParserListener) EnterDescribeStatements(ctx *DescribeStatementsContext) {}

// ExitDescribeStatements is called when production describeStatements is exited.
func (s *BaseMySqlParserListener) ExitDescribeStatements(ctx *DescribeStatementsContext) {}

// EnterDescribeConnection is called when production describeConnection is entered.
func (s *BaseMySqlParserListener) EnterDescribeConnection(ctx *DescribeConnectionContext) {}

// ExitDescribeConnection is called when production describeConnection is exited.
func (s *BaseMySqlParserListener) ExitDescribeConnection(ctx *DescribeConnectionContext) {}

// EnterFullId is called when production fullId is entered.
func (s *BaseMySqlParserListener) EnterFullId(ctx *FullIdContext) {}

// ExitFullId is called when production fullId is exited.
func (s *BaseMySqlParserListener) ExitFullId(ctx *FullIdContext) {}

// EnterTableName is called when production tableName is entered.
func (s *BaseMySqlParserListener) EnterTableName(ctx *TableNameContext) {}

// ExitTableName is called when production tableName is exited.
func (s *BaseMySqlParserListener) ExitTableName(ctx *TableNameContext) {}

// EnterRoleName is called when production roleName is entered.
func (s *BaseMySqlParserListener) EnterRoleName(ctx *RoleNameContext) {}

// ExitRoleName is called when production roleName is exited.
func (s *BaseMySqlParserListener) ExitRoleName(ctx *RoleNameContext) {}

// EnterFullColumnName is called when production fullColumnName is entered.
func (s *BaseMySqlParserListener) EnterFullColumnName(ctx *FullColumnNameContext) {}

// ExitFullColumnName is called when production fullColumnName is exited.
func (s *BaseMySqlParserListener) ExitFullColumnName(ctx *FullColumnNameContext) {}

// EnterIndexColumnName is called when production indexColumnName is entered.
func (s *BaseMySqlParserListener) EnterIndexColumnName(ctx *IndexColumnNameContext) {}

// ExitIndexColumnName is called when production indexColumnName is exited.
func (s *BaseMySqlParserListener) ExitIndexColumnName(ctx *IndexColumnNameContext) {}

// EnterSimpleUserName is called when production simpleUserName is entered.
func (s *BaseMySqlParserListener) EnterSimpleUserName(ctx *SimpleUserNameContext) {}

// ExitSimpleUserName is called when production simpleUserName is exited.
func (s *BaseMySqlParserListener) ExitSimpleUserName(ctx *SimpleUserNameContext) {}

// EnterHostName is called when production hostName is entered.
func (s *BaseMySqlParserListener) EnterHostName(ctx *HostNameContext) {}

// ExitHostName is called when production hostName is exited.
func (s *BaseMySqlParserListener) ExitHostName(ctx *HostNameContext) {}

// EnterUserName is called when production userName is entered.
func (s *BaseMySqlParserListener) EnterUserName(ctx *UserNameContext) {}

// ExitUserName is called when production userName is exited.
func (s *BaseMySqlParserListener) ExitUserName(ctx *UserNameContext) {}

// EnterMysqlVariable is called when production mysqlVariable is entered.
func (s *BaseMySqlParserListener) EnterMysqlVariable(ctx *MysqlVariableContext) {}

// ExitMysqlVariable is called when production mysqlVariable is exited.
func (s *BaseMySqlParserListener) ExitMysqlVariable(ctx *MysqlVariableContext) {}

// EnterCharsetName is called when production charsetName is entered.
func (s *BaseMySqlParserListener) EnterCharsetName(ctx *CharsetNameContext) {}

// ExitCharsetName is called when production charsetName is exited.
func (s *BaseMySqlParserListener) ExitCharsetName(ctx *CharsetNameContext) {}

// EnterCollationName is called when production collationName is entered.
func (s *BaseMySqlParserListener) EnterCollationName(ctx *CollationNameContext) {}

// ExitCollationName is called when production collationName is exited.
func (s *BaseMySqlParserListener) ExitCollationName(ctx *CollationNameContext) {}

// EnterEngineName is called when production engineName is entered.
func (s *BaseMySqlParserListener) EnterEngineName(ctx *EngineNameContext) {}

// ExitEngineName is called when production engineName is exited.
func (s *BaseMySqlParserListener) ExitEngineName(ctx *EngineNameContext) {}

// EnterEngineNameBase is called when production engineNameBase is entered.
func (s *BaseMySqlParserListener) EnterEngineNameBase(ctx *EngineNameBaseContext) {}

// ExitEngineNameBase is called when production engineNameBase is exited.
func (s *BaseMySqlParserListener) ExitEngineNameBase(ctx *EngineNameBaseContext) {}

// EnterUuidSet is called when production uuidSet is entered.
func (s *BaseMySqlParserListener) EnterUuidSet(ctx *UuidSetContext) {}

// ExitUuidSet is called when production uuidSet is exited.
func (s *BaseMySqlParserListener) ExitUuidSet(ctx *UuidSetContext) {}

// EnterXid is called when production xid is entered.
func (s *BaseMySqlParserListener) EnterXid(ctx *XidContext) {}

// ExitXid is called when production xid is exited.
func (s *BaseMySqlParserListener) ExitXid(ctx *XidContext) {}

// EnterXuidStringId is called when production xuidStringId is entered.
func (s *BaseMySqlParserListener) EnterXuidStringId(ctx *XuidStringIdContext) {}

// ExitXuidStringId is called when production xuidStringId is exited.
func (s *BaseMySqlParserListener) ExitXuidStringId(ctx *XuidStringIdContext) {}

// EnterAuthPlugin is called when production authPlugin is entered.
func (s *BaseMySqlParserListener) EnterAuthPlugin(ctx *AuthPluginContext) {}

// ExitAuthPlugin is called when production authPlugin is exited.
func (s *BaseMySqlParserListener) ExitAuthPlugin(ctx *AuthPluginContext) {}

// EnterUid is called when production uid is entered.
func (s *BaseMySqlParserListener) EnterUid(ctx *UidContext) {}

// ExitUid is called when production uid is exited.
func (s *BaseMySqlParserListener) ExitUid(ctx *UidContext) {}

// EnterSimpleId is called when production simpleId is entered.
func (s *BaseMySqlParserListener) EnterSimpleId(ctx *SimpleIdContext) {}

// ExitSimpleId is called when production simpleId is exited.
func (s *BaseMySqlParserListener) ExitSimpleId(ctx *SimpleIdContext) {}

// EnterDottedId is called when production dottedId is entered.
func (s *BaseMySqlParserListener) EnterDottedId(ctx *DottedIdContext) {}

// ExitDottedId is called when production dottedId is exited.
func (s *BaseMySqlParserListener) ExitDottedId(ctx *DottedIdContext) {}

// EnterDecimalLiteral is called when production decimalLiteral is entered.
func (s *BaseMySqlParserListener) EnterDecimalLiteral(ctx *DecimalLiteralContext) {}

// ExitDecimalLiteral is called when production decimalLiteral is exited.
func (s *BaseMySqlParserListener) ExitDecimalLiteral(ctx *DecimalLiteralContext) {}

// EnterFileSizeLiteral is called when production fileSizeLiteral is entered.
func (s *BaseMySqlParserListener) EnterFileSizeLiteral(ctx *FileSizeLiteralContext) {}

// ExitFileSizeLiteral is called when production fileSizeLiteral is exited.
func (s *BaseMySqlParserListener) ExitFileSizeLiteral(ctx *FileSizeLiteralContext) {}

// EnterStringLiteral is called when production stringLiteral is entered.
func (s *BaseMySqlParserListener) EnterStringLiteral(ctx *StringLiteralContext) {}

// ExitStringLiteral is called when production stringLiteral is exited.
func (s *BaseMySqlParserListener) ExitStringLiteral(ctx *StringLiteralContext) {}

// EnterBooleanLiteral is called when production booleanLiteral is entered.
func (s *BaseMySqlParserListener) EnterBooleanLiteral(ctx *BooleanLiteralContext) {}

// ExitBooleanLiteral is called when production booleanLiteral is exited.
func (s *BaseMySqlParserListener) ExitBooleanLiteral(ctx *BooleanLiteralContext) {}

// EnterHexadecimalLiteral is called when production hexadecimalLiteral is entered.
func (s *BaseMySqlParserListener) EnterHexadecimalLiteral(ctx *HexadecimalLiteralContext) {}

// ExitHexadecimalLiteral is called when production hexadecimalLiteral is exited.
func (s *BaseMySqlParserListener) ExitHexadecimalLiteral(ctx *HexadecimalLiteralContext) {}

// EnterNullNotnull is called when production nullNotnull is entered.
func (s *BaseMySqlParserListener) EnterNullNotnull(ctx *NullNotnullContext) {}

// ExitNullNotnull is called when production nullNotnull is exited.
func (s *BaseMySqlParserListener) ExitNullNotnull(ctx *NullNotnullContext) {}

// EnterConstant is called when production constant is entered.
func (s *BaseMySqlParserListener) EnterConstant(ctx *ConstantContext) {}

// ExitConstant is called when production constant is exited.
func (s *BaseMySqlParserListener) ExitConstant(ctx *ConstantContext) {}

// EnterStringDataType is called when production stringDataType is entered.
func (s *BaseMySqlParserListener) EnterStringDataType(ctx *StringDataTypeContext) {}

// ExitStringDataType is called when production stringDataType is exited.
func (s *BaseMySqlParserListener) ExitStringDataType(ctx *StringDataTypeContext) {}

// EnterNationalVaryingStringDataType is called when production nationalVaryingStringDataType is entered.
func (s *BaseMySqlParserListener) EnterNationalVaryingStringDataType(ctx *NationalVaryingStringDataTypeContext) {
}

// ExitNationalVaryingStringDataType is called when production nationalVaryingStringDataType is exited.
func (s *BaseMySqlParserListener) ExitNationalVaryingStringDataType(ctx *NationalVaryingStringDataTypeContext) {
}

// EnterNationalStringDataType is called when production nationalStringDataType is entered.
func (s *BaseMySqlParserListener) EnterNationalStringDataType(ctx *NationalStringDataTypeContext) {}

// ExitNationalStringDataType is called when production nationalStringDataType is exited.
func (s *BaseMySqlParserListener) ExitNationalStringDataType(ctx *NationalStringDataTypeContext) {}

// EnterDimensionDataType is called when production dimensionDataType is entered.
func (s *BaseMySqlParserListener) EnterDimensionDataType(ctx *DimensionDataTypeContext) {}

// ExitDimensionDataType is called when production dimensionDataType is exited.
func (s *BaseMySqlParserListener) ExitDimensionDataType(ctx *DimensionDataTypeContext) {}

// EnterSimpleDataType is called when production simpleDataType is entered.
func (s *BaseMySqlParserListener) EnterSimpleDataType(ctx *SimpleDataTypeContext) {}

// ExitSimpleDataType is called when production simpleDataType is exited.
func (s *BaseMySqlParserListener) ExitSimpleDataType(ctx *SimpleDataTypeContext) {}

// EnterCollectionDataType is called when production collectionDataType is entered.
func (s *BaseMySqlParserListener) EnterCollectionDataType(ctx *CollectionDataTypeContext) {}

// ExitCollectionDataType is called when production collectionDataType is exited.
func (s *BaseMySqlParserListener) ExitCollectionDataType(ctx *CollectionDataTypeContext) {}

// EnterSpatialDataType is called when production spatialDataType is entered.
func (s *BaseMySqlParserListener) EnterSpatialDataType(ctx *SpatialDataTypeContext) {}

// ExitSpatialDataType is called when production spatialDataType is exited.
func (s *BaseMySqlParserListener) ExitSpatialDataType(ctx *SpatialDataTypeContext) {}

// EnterLongVarcharDataType is called when production longVarcharDataType is entered.
func (s *BaseMySqlParserListener) EnterLongVarcharDataType(ctx *LongVarcharDataTypeContext) {}

// ExitLongVarcharDataType is called when production longVarcharDataType is exited.
func (s *BaseMySqlParserListener) ExitLongVarcharDataType(ctx *LongVarcharDataTypeContext) {}

// EnterLongVarbinaryDataType is called when production longVarbinaryDataType is entered.
func (s *BaseMySqlParserListener) EnterLongVarbinaryDataType(ctx *LongVarbinaryDataTypeContext) {}

// ExitLongVarbinaryDataType is called when production longVarbinaryDataType is exited.
func (s *BaseMySqlParserListener) ExitLongVarbinaryDataType(ctx *LongVarbinaryDataTypeContext) {}

// EnterCollectionOptions is called when production collectionOptions is entered.
func (s *BaseMySqlParserListener) EnterCollectionOptions(ctx *CollectionOptionsContext) {}

// ExitCollectionOptions is called when production collectionOptions is exited.
func (s *BaseMySqlParserListener) ExitCollectionOptions(ctx *CollectionOptionsContext) {}

// EnterConvertedDataType is called when production convertedDataType is entered.
func (s *BaseMySqlParserListener) EnterConvertedDataType(ctx *ConvertedDataTypeContext) {}

// ExitConvertedDataType is called when production convertedDataType is exited.
func (s *BaseMySqlParserListener) ExitConvertedDataType(ctx *ConvertedDataTypeContext) {}

// EnterLengthOneDimension is called when production lengthOneDimension is entered.
func (s *BaseMySqlParserListener) EnterLengthOneDimension(ctx *LengthOneDimensionContext) {}

// ExitLengthOneDimension is called when production lengthOneDimension is exited.
func (s *BaseMySqlParserListener) ExitLengthOneDimension(ctx *LengthOneDimensionContext) {}

// EnterLengthTwoDimension is called when production lengthTwoDimension is entered.
func (s *BaseMySqlParserListener) EnterLengthTwoDimension(ctx *LengthTwoDimensionContext) {}

// ExitLengthTwoDimension is called when production lengthTwoDimension is exited.
func (s *BaseMySqlParserListener) ExitLengthTwoDimension(ctx *LengthTwoDimensionContext) {}

// EnterLengthTwoOptionalDimension is called when production lengthTwoOptionalDimension is entered.
func (s *BaseMySqlParserListener) EnterLengthTwoOptionalDimension(ctx *LengthTwoOptionalDimensionContext) {
}

// ExitLengthTwoOptionalDimension is called when production lengthTwoOptionalDimension is exited.
func (s *BaseMySqlParserListener) ExitLengthTwoOptionalDimension(ctx *LengthTwoOptionalDimensionContext) {
}

// EnterUidList is called when production uidList is entered.
func (s *BaseMySqlParserListener) EnterUidList(ctx *UidListContext) {}

// ExitUidList is called when production uidList is exited.
func (s *BaseMySqlParserListener) ExitUidList(ctx *UidListContext) {}

// EnterFullColumnNameList is called when production fullColumnNameList is entered.
func (s *BaseMySqlParserListener) EnterFullColumnNameList(ctx *FullColumnNameListContext) {}

// ExitFullColumnNameList is called when production fullColumnNameList is exited.
func (s *BaseMySqlParserListener) ExitFullColumnNameList(ctx *FullColumnNameListContext) {}

// EnterTables is called when production tables is entered.
func (s *BaseMySqlParserListener) EnterTables(ctx *TablesContext) {}

// ExitTables is called when production tables is exited.
func (s *BaseMySqlParserListener) ExitTables(ctx *TablesContext) {}

// EnterIndexColumnNames is called when production indexColumnNames is entered.
func (s *BaseMySqlParserListener) EnterIndexColumnNames(ctx *IndexColumnNamesContext) {}

// ExitIndexColumnNames is called when production indexColumnNames is exited.
func (s *BaseMySqlParserListener) ExitIndexColumnNames(ctx *IndexColumnNamesContext) {}

// EnterExpressions is called when production expressions is entered.
func (s *BaseMySqlParserListener) EnterExpressions(ctx *ExpressionsContext) {}

// ExitExpressions is called when production expressions is exited.
func (s *BaseMySqlParserListener) ExitExpressions(ctx *ExpressionsContext) {}

// EnterExpressionsWithDefaults is called when production expressionsWithDefaults is entered.
func (s *BaseMySqlParserListener) EnterExpressionsWithDefaults(ctx *ExpressionsWithDefaultsContext) {}

// ExitExpressionsWithDefaults is called when production expressionsWithDefaults is exited.
func (s *BaseMySqlParserListener) ExitExpressionsWithDefaults(ctx *ExpressionsWithDefaultsContext) {}

// EnterConstants is called when production constants is entered.
func (s *BaseMySqlParserListener) EnterConstants(ctx *ConstantsContext) {}

// ExitConstants is called when production constants is exited.
func (s *BaseMySqlParserListener) ExitConstants(ctx *ConstantsContext) {}

// EnterSimpleStrings is called when production simpleStrings is entered.
func (s *BaseMySqlParserListener) EnterSimpleStrings(ctx *SimpleStringsContext) {}

// ExitSimpleStrings is called when production simpleStrings is exited.
func (s *BaseMySqlParserListener) ExitSimpleStrings(ctx *SimpleStringsContext) {}

// EnterUserVariables is called when production userVariables is entered.
func (s *BaseMySqlParserListener) EnterUserVariables(ctx *UserVariablesContext) {}

// ExitUserVariables is called when production userVariables is exited.
func (s *BaseMySqlParserListener) ExitUserVariables(ctx *UserVariablesContext) {}

// EnterDefaultValue is called when production defaultValue is entered.
func (s *BaseMySqlParserListener) EnterDefaultValue(ctx *DefaultValueContext) {}

// ExitDefaultValue is called when production defaultValue is exited.
func (s *BaseMySqlParserListener) ExitDefaultValue(ctx *DefaultValueContext) {}

// EnterCurrentTimestamp is called when production currentTimestamp is entered.
func (s *BaseMySqlParserListener) EnterCurrentTimestamp(ctx *CurrentTimestampContext) {}

// ExitCurrentTimestamp is called when production currentTimestamp is exited.
func (s *BaseMySqlParserListener) ExitCurrentTimestamp(ctx *CurrentTimestampContext) {}

// EnterExpressionOrDefault is called when production expressionOrDefault is entered.
func (s *BaseMySqlParserListener) EnterExpressionOrDefault(ctx *ExpressionOrDefaultContext) {}

// ExitExpressionOrDefault is called when production expressionOrDefault is exited.
func (s *BaseMySqlParserListener) ExitExpressionOrDefault(ctx *ExpressionOrDefaultContext) {}

// EnterIfExists is called when production ifExists is entered.
func (s *BaseMySqlParserListener) EnterIfExists(ctx *IfExistsContext) {}

// ExitIfExists is called when production ifExists is exited.
func (s *BaseMySqlParserListener) ExitIfExists(ctx *IfExistsContext) {}

// EnterIfNotExists is called when production ifNotExists is entered.
func (s *BaseMySqlParserListener) EnterIfNotExists(ctx *IfNotExistsContext) {}

// ExitIfNotExists is called when production ifNotExists is exited.
func (s *BaseMySqlParserListener) ExitIfNotExists(ctx *IfNotExistsContext) {}

// EnterOrReplace is called when production orReplace is entered.
func (s *BaseMySqlParserListener) EnterOrReplace(ctx *OrReplaceContext) {}

// ExitOrReplace is called when production orReplace is exited.
func (s *BaseMySqlParserListener) ExitOrReplace(ctx *OrReplaceContext) {}

// EnterWaitNowaitClause is called when production waitNowaitClause is entered.
func (s *BaseMySqlParserListener) EnterWaitNowaitClause(ctx *WaitNowaitClauseContext) {}

// ExitWaitNowaitClause is called when production waitNowaitClause is exited.
func (s *BaseMySqlParserListener) ExitWaitNowaitClause(ctx *WaitNowaitClauseContext) {}

// EnterSpecificFunctionCall is called when production specificFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterSpecificFunctionCall(ctx *SpecificFunctionCallContext) {}

// ExitSpecificFunctionCall is called when production specificFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitSpecificFunctionCall(ctx *SpecificFunctionCallContext) {}

// EnterAggregateFunctionCall is called when production aggregateFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterAggregateFunctionCall(ctx *AggregateFunctionCallContext) {}

// ExitAggregateFunctionCall is called when production aggregateFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitAggregateFunctionCall(ctx *AggregateFunctionCallContext) {}

// EnterNonAggregateFunctionCall is called when production nonAggregateFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterNonAggregateFunctionCall(ctx *NonAggregateFunctionCallContext) {
}

// ExitNonAggregateFunctionCall is called when production nonAggregateFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitNonAggregateFunctionCall(ctx *NonAggregateFunctionCallContext) {
}

// EnterScalarFunctionCall is called when production scalarFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterScalarFunctionCall(ctx *ScalarFunctionCallContext) {}

// ExitScalarFunctionCall is called when production scalarFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitScalarFunctionCall(ctx *ScalarFunctionCallContext) {}

// EnterUdfFunctionCall is called when production udfFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterUdfFunctionCall(ctx *UdfFunctionCallContext) {}

// ExitUdfFunctionCall is called when production udfFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitUdfFunctionCall(ctx *UdfFunctionCallContext) {}

// EnterPasswordFunctionCall is called when production passwordFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterPasswordFunctionCall(ctx *PasswordFunctionCallContext) {}

// ExitPasswordFunctionCall is called when production passwordFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitPasswordFunctionCall(ctx *PasswordFunctionCallContext) {}

// EnterSimpleFunctionCall is called when production simpleFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterSimpleFunctionCall(ctx *SimpleFunctionCallContext) {}

// ExitSimpleFunctionCall is called when production simpleFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitSimpleFunctionCall(ctx *SimpleFunctionCallContext) {}

// EnterCurrentUser is called when production currentUser is entered.
func (s *BaseMySqlParserListener) EnterCurrentUser(ctx *CurrentUserContext) {}

// ExitCurrentUser is called when production currentUser is exited.
func (s *BaseMySqlParserListener) ExitCurrentUser(ctx *CurrentUserContext) {}

// EnterDataTypeFunctionCall is called when production dataTypeFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterDataTypeFunctionCall(ctx *DataTypeFunctionCallContext) {}

// ExitDataTypeFunctionCall is called when production dataTypeFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitDataTypeFunctionCall(ctx *DataTypeFunctionCallContext) {}

// EnterValuesFunctionCall is called when production valuesFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterValuesFunctionCall(ctx *ValuesFunctionCallContext) {}

// ExitValuesFunctionCall is called when production valuesFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitValuesFunctionCall(ctx *ValuesFunctionCallContext) {}

// EnterCaseExpressionFunctionCall is called when production caseExpressionFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterCaseExpressionFunctionCall(ctx *CaseExpressionFunctionCallContext) {
}

// ExitCaseExpressionFunctionCall is called when production caseExpressionFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitCaseExpressionFunctionCall(ctx *CaseExpressionFunctionCallContext) {
}

// EnterCaseFunctionCall is called when production caseFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterCaseFunctionCall(ctx *CaseFunctionCallContext) {}

// ExitCaseFunctionCall is called when production caseFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitCaseFunctionCall(ctx *CaseFunctionCallContext) {}

// EnterCharFunctionCall is called when production charFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterCharFunctionCall(ctx *CharFunctionCallContext) {}

// ExitCharFunctionCall is called when production charFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitCharFunctionCall(ctx *CharFunctionCallContext) {}

// EnterPositionFunctionCall is called when production positionFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterPositionFunctionCall(ctx *PositionFunctionCallContext) {}

// ExitPositionFunctionCall is called when production positionFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitPositionFunctionCall(ctx *PositionFunctionCallContext) {}

// EnterSubstrFunctionCall is called when production substrFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterSubstrFunctionCall(ctx *SubstrFunctionCallContext) {}

// ExitSubstrFunctionCall is called when production substrFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitSubstrFunctionCall(ctx *SubstrFunctionCallContext) {}

// EnterTrimFunctionCall is called when production trimFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterTrimFunctionCall(ctx *TrimFunctionCallContext) {}

// ExitTrimFunctionCall is called when production trimFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitTrimFunctionCall(ctx *TrimFunctionCallContext) {}

// EnterWeightFunctionCall is called when production weightFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterWeightFunctionCall(ctx *WeightFunctionCallContext) {}

// ExitWeightFunctionCall is called when production weightFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitWeightFunctionCall(ctx *WeightFunctionCallContext) {}

// EnterExtractFunctionCall is called when production extractFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterExtractFunctionCall(ctx *ExtractFunctionCallContext) {}

// ExitExtractFunctionCall is called when production extractFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitExtractFunctionCall(ctx *ExtractFunctionCallContext) {}

// EnterGetFormatFunctionCall is called when production getFormatFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterGetFormatFunctionCall(ctx *GetFormatFunctionCallContext) {}

// ExitGetFormatFunctionCall is called when production getFormatFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitGetFormatFunctionCall(ctx *GetFormatFunctionCallContext) {}

// EnterJsonValueFunctionCall is called when production jsonValueFunctionCall is entered.
func (s *BaseMySqlParserListener) EnterJsonValueFunctionCall(ctx *JsonValueFunctionCallContext) {}

// ExitJsonValueFunctionCall is called when production jsonValueFunctionCall is exited.
func (s *BaseMySqlParserListener) ExitJsonValueFunctionCall(ctx *JsonValueFunctionCallContext) {}

// EnterCaseFuncAlternative is called when production caseFuncAlternative is entered.
func (s *BaseMySqlParserListener) EnterCaseFuncAlternative(ctx *CaseFuncAlternativeContext) {}

// ExitCaseFuncAlternative is called when production caseFuncAlternative is exited.
func (s *BaseMySqlParserListener) ExitCaseFuncAlternative(ctx *CaseFuncAlternativeContext) {}

// EnterLevelWeightList is called when production levelWeightList is entered.
func (s *BaseMySqlParserListener) EnterLevelWeightList(ctx *LevelWeightListContext) {}

// ExitLevelWeightList is called when production levelWeightList is exited.
func (s *BaseMySqlParserListener) ExitLevelWeightList(ctx *LevelWeightListContext) {}

// EnterLevelWeightRange is called when production levelWeightRange is entered.
func (s *BaseMySqlParserListener) EnterLevelWeightRange(ctx *LevelWeightRangeContext) {}

// ExitLevelWeightRange is called when production levelWeightRange is exited.
func (s *BaseMySqlParserListener) ExitLevelWeightRange(ctx *LevelWeightRangeContext) {}

// EnterLevelInWeightListElement is called when production levelInWeightListElement is entered.
func (s *BaseMySqlParserListener) EnterLevelInWeightListElement(ctx *LevelInWeightListElementContext) {
}

// ExitLevelInWeightListElement is called when production levelInWeightListElement is exited.
func (s *BaseMySqlParserListener) ExitLevelInWeightListElement(ctx *LevelInWeightListElementContext) {
}

// EnterAggregateWindowedFunction is called when production aggregateWindowedFunction is entered.
func (s *BaseMySqlParserListener) EnterAggregateWindowedFunction(ctx *AggregateWindowedFunctionContext) {
}

// ExitAggregateWindowedFunction is called when production aggregateWindowedFunction is exited.
func (s *BaseMySqlParserListener) ExitAggregateWindowedFunction(ctx *AggregateWindowedFunctionContext) {
}

// EnterNonAggregateWindowedFunction is called when production nonAggregateWindowedFunction is entered.
func (s *BaseMySqlParserListener) EnterNonAggregateWindowedFunction(ctx *NonAggregateWindowedFunctionContext) {
}

// ExitNonAggregateWindowedFunction is called when production nonAggregateWindowedFunction is exited.
func (s *BaseMySqlParserListener) ExitNonAggregateWindowedFunction(ctx *NonAggregateWindowedFunctionContext) {
}

// EnterOverClause is called when production overClause is entered.
func (s *BaseMySqlParserListener) EnterOverClause(ctx *OverClauseContext) {}

// ExitOverClause is called when production overClause is exited.
func (s *BaseMySqlParserListener) ExitOverClause(ctx *OverClauseContext) {}

// EnterWindowSpec is called when production windowSpec is entered.
func (s *BaseMySqlParserListener) EnterWindowSpec(ctx *WindowSpecContext) {}

// ExitWindowSpec is called when production windowSpec is exited.
func (s *BaseMySqlParserListener) ExitWindowSpec(ctx *WindowSpecContext) {}

// EnterWindowName is called when production windowName is entered.
func (s *BaseMySqlParserListener) EnterWindowName(ctx *WindowNameContext) {}

// ExitWindowName is called when production windowName is exited.
func (s *BaseMySqlParserListener) ExitWindowName(ctx *WindowNameContext) {}

// EnterFrameClause is called when production frameClause is entered.
func (s *BaseMySqlParserListener) EnterFrameClause(ctx *FrameClauseContext) {}

// ExitFrameClause is called when production frameClause is exited.
func (s *BaseMySqlParserListener) ExitFrameClause(ctx *FrameClauseContext) {}

// EnterFrameUnits is called when production frameUnits is entered.
func (s *BaseMySqlParserListener) EnterFrameUnits(ctx *FrameUnitsContext) {}

// ExitFrameUnits is called when production frameUnits is exited.
func (s *BaseMySqlParserListener) ExitFrameUnits(ctx *FrameUnitsContext) {}

// EnterFrameExtent is called when production frameExtent is entered.
func (s *BaseMySqlParserListener) EnterFrameExtent(ctx *FrameExtentContext) {}

// ExitFrameExtent is called when production frameExtent is exited.
func (s *BaseMySqlParserListener) ExitFrameExtent(ctx *FrameExtentContext) {}

// EnterFrameBetween is called when production frameBetween is entered.
func (s *BaseMySqlParserListener) EnterFrameBetween(ctx *FrameBetweenContext) {}

// ExitFrameBetween is called when production frameBetween is exited.
func (s *BaseMySqlParserListener) ExitFrameBetween(ctx *FrameBetweenContext) {}

// EnterFrameRange is called when production frameRange is entered.
func (s *BaseMySqlParserListener) EnterFrameRange(ctx *FrameRangeContext) {}

// ExitFrameRange is called when production frameRange is exited.
func (s *BaseMySqlParserListener) ExitFrameRange(ctx *FrameRangeContext) {}

// EnterPartitionClause is called when production partitionClause is entered.
func (s *BaseMySqlParserListener) EnterPartitionClause(ctx *PartitionClauseContext) {}

// ExitPartitionClause is called when production partitionClause is exited.
func (s *BaseMySqlParserListener) ExitPartitionClause(ctx *PartitionClauseContext) {}

// EnterScalarFunctionName is called when production scalarFunctionName is entered.
func (s *BaseMySqlParserListener) EnterScalarFunctionName(ctx *ScalarFunctionNameContext) {}

// ExitScalarFunctionName is called when production scalarFunctionName is exited.
func (s *BaseMySqlParserListener) ExitScalarFunctionName(ctx *ScalarFunctionNameContext) {}

// EnterPasswordFunctionClause is called when production passwordFunctionClause is entered.
func (s *BaseMySqlParserListener) EnterPasswordFunctionClause(ctx *PasswordFunctionClauseContext) {}

// ExitPasswordFunctionClause is called when production passwordFunctionClause is exited.
func (s *BaseMySqlParserListener) ExitPasswordFunctionClause(ctx *PasswordFunctionClauseContext) {}

// EnterFunctionArgs is called when production functionArgs is entered.
func (s *BaseMySqlParserListener) EnterFunctionArgs(ctx *FunctionArgsContext) {}

// ExitFunctionArgs is called when production functionArgs is exited.
func (s *BaseMySqlParserListener) ExitFunctionArgs(ctx *FunctionArgsContext) {}

// EnterFunctionArg is called when production functionArg is entered.
func (s *BaseMySqlParserListener) EnterFunctionArg(ctx *FunctionArgContext) {}

// ExitFunctionArg is called when production functionArg is exited.
func (s *BaseMySqlParserListener) ExitFunctionArg(ctx *FunctionArgContext) {}

// EnterIsExpression is called when production isExpression is entered.
func (s *BaseMySqlParserListener) EnterIsExpression(ctx *IsExpressionContext) {}

// ExitIsExpression is called when production isExpression is exited.
func (s *BaseMySqlParserListener) ExitIsExpression(ctx *IsExpressionContext) {}

// EnterNotExpression is called when production notExpression is entered.
func (s *BaseMySqlParserListener) EnterNotExpression(ctx *NotExpressionContext) {}

// ExitNotExpression is called when production notExpression is exited.
func (s *BaseMySqlParserListener) ExitNotExpression(ctx *NotExpressionContext) {}

// EnterLogicalExpression is called when production logicalExpression is entered.
func (s *BaseMySqlParserListener) EnterLogicalExpression(ctx *LogicalExpressionContext) {}

// ExitLogicalExpression is called when production logicalExpression is exited.
func (s *BaseMySqlParserListener) ExitLogicalExpression(ctx *LogicalExpressionContext) {}

// EnterPredicateExpression is called when production predicateExpression is entered.
func (s *BaseMySqlParserListener) EnterPredicateExpression(ctx *PredicateExpressionContext) {}

// ExitPredicateExpression is called when production predicateExpression is exited.
func (s *BaseMySqlParserListener) ExitPredicateExpression(ctx *PredicateExpressionContext) {}

// EnterSoundsLikePredicate is called when production soundsLikePredicate is entered.
func (s *BaseMySqlParserListener) EnterSoundsLikePredicate(ctx *SoundsLikePredicateContext) {}

// ExitSoundsLikePredicate is called when production soundsLikePredicate is exited.
func (s *BaseMySqlParserListener) ExitSoundsLikePredicate(ctx *SoundsLikePredicateContext) {}

// EnterExpressionAtomPredicate is called when production expressionAtomPredicate is entered.
func (s *BaseMySqlParserListener) EnterExpressionAtomPredicate(ctx *ExpressionAtomPredicateContext) {}

// ExitExpressionAtomPredicate is called when production expressionAtomPredicate is exited.
func (s *BaseMySqlParserListener) ExitExpressionAtomPredicate(ctx *ExpressionAtomPredicateContext) {}

// EnterSubqueryComparisonPredicate is called when production subqueryComparisonPredicate is entered.
func (s *BaseMySqlParserListener) EnterSubqueryComparisonPredicate(ctx *SubqueryComparisonPredicateContext) {
}

// ExitSubqueryComparisonPredicate is called when production subqueryComparisonPredicate is exited.
func (s *BaseMySqlParserListener) ExitSubqueryComparisonPredicate(ctx *SubqueryComparisonPredicateContext) {
}

// EnterJsonMemberOfPredicate is called when production jsonMemberOfPredicate is entered.
func (s *BaseMySqlParserListener) EnterJsonMemberOfPredicate(ctx *JsonMemberOfPredicateContext) {}

// ExitJsonMemberOfPredicate is called when production jsonMemberOfPredicate is exited.
func (s *BaseMySqlParserListener) ExitJsonMemberOfPredicate(ctx *JsonMemberOfPredicateContext) {}

// EnterBinaryComparisonPredicate is called when production binaryComparisonPredicate is entered.
func (s *BaseMySqlParserListener) EnterBinaryComparisonPredicate(ctx *BinaryComparisonPredicateContext) {
}

// ExitBinaryComparisonPredicate is called when production binaryComparisonPredicate is exited.
func (s *BaseMySqlParserListener) ExitBinaryComparisonPredicate(ctx *BinaryComparisonPredicateContext) {
}

// EnterInPredicate is called when production inPredicate is entered.
func (s *BaseMySqlParserListener) EnterInPredicate(ctx *InPredicateContext) {}

// ExitInPredicate is called when production inPredicate is exited.
func (s *BaseMySqlParserListener) ExitInPredicate(ctx *InPredicateContext) {}

// EnterBetweenPredicate is called when production betweenPredicate is entered.
func (s *BaseMySqlParserListener) EnterBetweenPredicate(ctx *BetweenPredicateContext) {}

// ExitBetweenPredicate is called when production betweenPredicate is exited.
func (s *BaseMySqlParserListener) ExitBetweenPredicate(ctx *BetweenPredicateContext) {}

// EnterIsNullPredicate is called when production isNullPredicate is entered.
func (s *BaseMySqlParserListener) EnterIsNullPredicate(ctx *IsNullPredicateContext) {}

// ExitIsNullPredicate is called when production isNullPredicate is exited.
func (s *BaseMySqlParserListener) ExitIsNullPredicate(ctx *IsNullPredicateContext) {}

// EnterLikePredicate is called when production likePredicate is entered.
func (s *BaseMySqlParserListener) EnterLikePredicate(ctx *LikePredicateContext) {}

// ExitLikePredicate is called when production likePredicate is exited.
func (s *BaseMySqlParserListener) ExitLikePredicate(ctx *LikePredicateContext) {}

// EnterRegexpPredicate is called when production regexpPredicate is entered.
func (s *BaseMySqlParserListener) EnterRegexpPredicate(ctx *RegexpPredicateContext) {}

// ExitRegexpPredicate is called when production regexpPredicate is exited.
func (s *BaseMySqlParserListener) ExitRegexpPredicate(ctx *RegexpPredicateContext) {}

// EnterUnaryExpressionAtom is called when production unaryExpressionAtom is entered.
func (s *BaseMySqlParserListener) EnterUnaryExpressionAtom(ctx *UnaryExpressionAtomContext) {}

// ExitUnaryExpressionAtom is called when production unaryExpressionAtom is exited.
func (s *BaseMySqlParserListener) ExitUnaryExpressionAtom(ctx *UnaryExpressionAtomContext) {}

// EnterCollateExpressionAtom is called when production collateExpressionAtom is entered.
func (s *BaseMySqlParserListener) EnterCollateExpressionAtom(ctx *CollateExpressionAtomContext) {}

// ExitCollateExpressionAtom is called when production collateExpressionAtom is exited.
func (s *BaseMySqlParserListener) ExitCollateExpressionAtom(ctx *CollateExpressionAtomContext) {}

// EnterVariableAssignExpressionAtom is called when production variableAssignExpressionAtom is entered.
func (s *BaseMySqlParserListener) EnterVariableAssignExpressionAtom(ctx *VariableAssignExpressionAtomContext) {
}

// ExitVariableAssignExpressionAtom is called when production variableAssignExpressionAtom is exited.
func (s *BaseMySqlParserListener) ExitVariableAssignExpressionAtom(ctx *VariableAssignExpressionAtomContext) {
}

// EnterMysqlVariableExpressionAtom is called when production mysqlVariableExpressionAtom is entered.
func (s *BaseMySqlParserListener) EnterMysqlVariableExpressionAtom(ctx *MysqlVariableExpressionAtomContext) {
}

// ExitMysqlVariableExpressionAtom is called when production mysqlVariableExpressionAtom is exited.
func (s *BaseMySqlParserListener) ExitMysqlVariableExpressionAtom(ctx *MysqlVariableExpressionAtomContext) {
}

// EnterNestedExpressionAtom is called when production nestedExpressionAtom is entered.
func (s *BaseMySqlParserListener) EnterNestedExpressionAtom(ctx *NestedExpressionAtomContext) {}

// ExitNestedExpressionAtom is called when production nestedExpressionAtom is exited.
func (s *BaseMySqlParserListener) ExitNestedExpressionAtom(ctx *NestedExpressionAtomContext) {}

// EnterNestedRowExpressionAtom is called when production nestedRowExpressionAtom is entered.
func (s *BaseMySqlParserListener) EnterNestedRowExpressionAtom(ctx *NestedRowExpressionAtomContext) {}

// ExitNestedRowExpressionAtom is called when production nestedRowExpressionAtom is exited.
func (s *BaseMySqlParserListener) ExitNestedRowExpressionAtom(ctx *NestedRowExpressionAtomContext) {}

// EnterMathExpressionAtom is called when production mathExpressionAtom is entered.
func (s *BaseMySqlParserListener) EnterMathExpressionAtom(ctx *MathExpressionAtomContext) {}

// ExitMathExpressionAtom is called when production mathExpressionAtom is exited.
func (s *BaseMySqlParserListener) ExitMathExpressionAtom(ctx *MathExpressionAtomContext) {}

// EnterExistsExpressionAtom is called when production existsExpressionAtom is entered.
func (s *BaseMySqlParserListener) EnterExistsExpressionAtom(ctx *ExistsExpressionAtomContext) {}

// ExitExistsExpressionAtom is called when production existsExpressionAtom is exited.
func (s *BaseMySqlParserListener) ExitExistsExpressionAtom(ctx *ExistsExpressionAtomContext) {}

// EnterIntervalExpressionAtom is called when production intervalExpressionAtom is entered.
func (s *BaseMySqlParserListener) EnterIntervalExpressionAtom(ctx *IntervalExpressionAtomContext) {}

// ExitIntervalExpressionAtom is called when production intervalExpressionAtom is exited.
func (s *BaseMySqlParserListener) ExitIntervalExpressionAtom(ctx *IntervalExpressionAtomContext) {}

// EnterJsonExpressionAtom is called when production jsonExpressionAtom is entered.
func (s *BaseMySqlParserListener) EnterJsonExpressionAtom(ctx *JsonExpressionAtomContext) {}

// ExitJsonExpressionAtom is called when production jsonExpressionAtom is exited.
func (s *BaseMySqlParserListener) ExitJsonExpressionAtom(ctx *JsonExpressionAtomContext) {}

// EnterSubqueryExpressionAtom is called when production subqueryExpressionAtom is entered.
func (s *BaseMySqlParserListener) EnterSubqueryExpressionAtom(ctx *SubqueryExpressionAtomContext) {}

// ExitSubqueryExpressionAtom is called when production subqueryExpressionAtom is exited.
func (s *BaseMySqlParserListener) ExitSubqueryExpressionAtom(ctx *SubqueryExpressionAtomContext) {}

// EnterConstantExpressionAtom is called when production constantExpressionAtom is entered.
func (s *BaseMySqlParserListener) EnterConstantExpressionAtom(ctx *ConstantExpressionAtomContext) {}

// ExitConstantExpressionAtom is called when production constantExpressionAtom is exited.
func (s *BaseMySqlParserListener) ExitConstantExpressionAtom(ctx *ConstantExpressionAtomContext) {}

// EnterFunctionCallExpressionAtom is called when production functionCallExpressionAtom is entered.
func (s *BaseMySqlParserListener) EnterFunctionCallExpressionAtom(ctx *FunctionCallExpressionAtomContext) {
}

// ExitFunctionCallExpressionAtom is called when production functionCallExpressionAtom is exited.
func (s *BaseMySqlParserListener) ExitFunctionCallExpressionAtom(ctx *FunctionCallExpressionAtomContext) {
}

// EnterBinaryExpressionAtom is called when production binaryExpressionAtom is entered.
func (s *BaseMySqlParserListener) EnterBinaryExpressionAtom(ctx *BinaryExpressionAtomContext) {}

// ExitBinaryExpressionAtom is called when production binaryExpressionAtom is exited.
func (s *BaseMySqlParserListener) ExitBinaryExpressionAtom(ctx *BinaryExpressionAtomContext) {}

// EnterFullColumnNameExpressionAtom is called when production fullColumnNameExpressionAtom is entered.
func (s *BaseMySqlParserListener) EnterFullColumnNameExpressionAtom(ctx *FullColumnNameExpressionAtomContext) {
}

// ExitFullColumnNameExpressionAtom is called when production fullColumnNameExpressionAtom is exited.
func (s *BaseMySqlParserListener) ExitFullColumnNameExpressionAtom(ctx *FullColumnNameExpressionAtomContext) {
}

// EnterBitExpressionAtom is called when production bitExpressionAtom is entered.
func (s *BaseMySqlParserListener) EnterBitExpressionAtom(ctx *BitExpressionAtomContext) {}

// ExitBitExpressionAtom is called when production bitExpressionAtom is exited.
func (s *BaseMySqlParserListener) ExitBitExpressionAtom(ctx *BitExpressionAtomContext) {}

// EnterUnaryOperator is called when production unaryOperator is entered.
func (s *BaseMySqlParserListener) EnterUnaryOperator(ctx *UnaryOperatorContext) {}

// ExitUnaryOperator is called when production unaryOperator is exited.
func (s *BaseMySqlParserListener) ExitUnaryOperator(ctx *UnaryOperatorContext) {}

// EnterComparisonOperator is called when production comparisonOperator is entered.
func (s *BaseMySqlParserListener) EnterComparisonOperator(ctx *ComparisonOperatorContext) {}

// ExitComparisonOperator is called when production comparisonOperator is exited.
func (s *BaseMySqlParserListener) ExitComparisonOperator(ctx *ComparisonOperatorContext) {}

// EnterLogicalOperator is called when production logicalOperator is entered.
func (s *BaseMySqlParserListener) EnterLogicalOperator(ctx *LogicalOperatorContext) {}

// ExitLogicalOperator is called when production logicalOperator is exited.
func (s *BaseMySqlParserListener) ExitLogicalOperator(ctx *LogicalOperatorContext) {}

// EnterBitOperator is called when production bitOperator is entered.
func (s *BaseMySqlParserListener) EnterBitOperator(ctx *BitOperatorContext) {}

// ExitBitOperator is called when production bitOperator is exited.
func (s *BaseMySqlParserListener) ExitBitOperator(ctx *BitOperatorContext) {}

// EnterMultOperator is called when production multOperator is entered.
func (s *BaseMySqlParserListener) EnterMultOperator(ctx *MultOperatorContext) {}

// ExitMultOperator is called when production multOperator is exited.
func (s *BaseMySqlParserListener) ExitMultOperator(ctx *MultOperatorContext) {}

// EnterAddOperator is called when production addOperator is entered.
func (s *BaseMySqlParserListener) EnterAddOperator(ctx *AddOperatorContext) {}

// ExitAddOperator is called when production addOperator is exited.
func (s *BaseMySqlParserListener) ExitAddOperator(ctx *AddOperatorContext) {}

// EnterJsonOperator is called when production jsonOperator is entered.
func (s *BaseMySqlParserListener) EnterJsonOperator(ctx *JsonOperatorContext) {}

// ExitJsonOperator is called when production jsonOperator is exited.
func (s *BaseMySqlParserListener) ExitJsonOperator(ctx *JsonOperatorContext) {}

// EnterCharsetNameBase is called when production charsetNameBase is entered.
func (s *BaseMySqlParserListener) EnterCharsetNameBase(ctx *CharsetNameBaseContext) {}

// ExitCharsetNameBase is called when production charsetNameBase is exited.
func (s *BaseMySqlParserListener) ExitCharsetNameBase(ctx *CharsetNameBaseContext) {}

// EnterTransactionLevelBase is called when production transactionLevelBase is entered.
func (s *BaseMySqlParserListener) EnterTransactionLevelBase(ctx *TransactionLevelBaseContext) {}

// ExitTransactionLevelBase is called when production transactionLevelBase is exited.
func (s *BaseMySqlParserListener) ExitTransactionLevelBase(ctx *TransactionLevelBaseContext) {}

// EnterPrivilegesBase is called when production privilegesBase is entered.
func (s *BaseMySqlParserListener) EnterPrivilegesBase(ctx *PrivilegesBaseContext) {}

// ExitPrivilegesBase is called when production privilegesBase is exited.
func (s *BaseMySqlParserListener) ExitPrivilegesBase(ctx *PrivilegesBaseContext) {}

// EnterIntervalTypeBase is called when production intervalTypeBase is entered.
func (s *BaseMySqlParserListener) EnterIntervalTypeBase(ctx *IntervalTypeBaseContext) {}

// ExitIntervalTypeBase is called when production intervalTypeBase is exited.
func (s *BaseMySqlParserListener) ExitIntervalTypeBase(ctx *IntervalTypeBaseContext) {}

// EnterDataTypeBase is called when production dataTypeBase is entered.
func (s *BaseMySqlParserListener) EnterDataTypeBase(ctx *DataTypeBaseContext) {}

// ExitDataTypeBase is called when production dataTypeBase is exited.
func (s *BaseMySqlParserListener) ExitDataTypeBase(ctx *DataTypeBaseContext) {}

// EnterKeywordsCanBeId is called when production keywordsCanBeId is entered.
func (s *BaseMySqlParserListener) EnterKeywordsCanBeId(ctx *KeywordsCanBeIdContext) {}

// ExitKeywordsCanBeId is called when production keywordsCanBeId is exited.
func (s *BaseMySqlParserListener) ExitKeywordsCanBeId(ctx *KeywordsCanBeIdContext) {}

// EnterFunctionNameBase is called when production functionNameBase is entered.
func (s *BaseMySqlParserListener) EnterFunctionNameBase(ctx *FunctionNameBaseContext) {}

// ExitFunctionNameBase is called when production functionNameBase is exited.
func (s *BaseMySqlParserListener) ExitFunctionNameBase(ctx *FunctionNameBaseContext) {}
