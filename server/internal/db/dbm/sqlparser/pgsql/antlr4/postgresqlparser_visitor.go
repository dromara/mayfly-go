// Code generated from PostgreSQLParser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package parser // PostgreSQLParser
import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by PostgreSQLParser.
type PostgreSQLParserVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by PostgreSQLParser#root.
	VisitRoot(ctx *RootContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#plsqlroot.
	VisitPlsqlroot(ctx *PlsqlrootContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmtblock.
	VisitStmtblock(ctx *StmtblockContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmtmulti.
	VisitStmtmulti(ctx *StmtmultiContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt.
	VisitStmt(ctx *StmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#plsqlconsolecommand.
	VisitPlsqlconsolecommand(ctx *PlsqlconsolecommandContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#callstmt.
	VisitCallstmt(ctx *CallstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createrolestmt.
	VisitCreaterolestmt(ctx *CreaterolestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_with.
	VisitOpt_with(ctx *Opt_withContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#optrolelist.
	VisitOptrolelist(ctx *OptrolelistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alteroptrolelist.
	VisitAlteroptrolelist(ctx *AlteroptrolelistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alteroptroleelem.
	VisitAlteroptroleelem(ctx *AlteroptroleelemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createoptroleelem.
	VisitCreateoptroleelem(ctx *CreateoptroleelemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createuserstmt.
	VisitCreateuserstmt(ctx *CreateuserstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterrolestmt.
	VisitAlterrolestmt(ctx *AlterrolestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_in_database.
	VisitOpt_in_database(ctx *Opt_in_databaseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterrolesetstmt.
	VisitAlterrolesetstmt(ctx *AlterrolesetstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#droprolestmt.
	VisitDroprolestmt(ctx *DroprolestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#creategroupstmt.
	VisitCreategroupstmt(ctx *CreategroupstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#altergroupstmt.
	VisitAltergroupstmt(ctx *AltergroupstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#add_drop.
	VisitAdd_drop(ctx *Add_dropContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createschemastmt.
	VisitCreateschemastmt(ctx *CreateschemastmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#optschemaname.
	VisitOptschemaname(ctx *OptschemanameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#optschemaeltlist.
	VisitOptschemaeltlist(ctx *OptschemaeltlistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#schema_stmt.
	VisitSchema_stmt(ctx *Schema_stmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#variablesetstmt.
	VisitVariablesetstmt(ctx *VariablesetstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#set_rest.
	VisitSet_rest(ctx *Set_restContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#generic_set.
	VisitGeneric_set(ctx *Generic_setContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#set_rest_more.
	VisitSet_rest_more(ctx *Set_rest_moreContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#var_name.
	VisitVar_name(ctx *Var_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#var_list.
	VisitVar_list(ctx *Var_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#var_value.
	VisitVar_value(ctx *Var_valueContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#iso_level.
	VisitIso_level(ctx *Iso_levelContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_boolean_or_string.
	VisitOpt_boolean_or_string(ctx *Opt_boolean_or_stringContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#zone_value.
	VisitZone_value(ctx *Zone_valueContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_encoding.
	VisitOpt_encoding(ctx *Opt_encodingContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#nonreservedword_or_sconst.
	VisitNonreservedword_or_sconst(ctx *Nonreservedword_or_sconstContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#variableresetstmt.
	VisitVariableresetstmt(ctx *VariableresetstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#reset_rest.
	VisitReset_rest(ctx *Reset_restContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#generic_reset.
	VisitGeneric_reset(ctx *Generic_resetContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#setresetclause.
	VisitSetresetclause(ctx *SetresetclauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#functionsetresetclause.
	VisitFunctionsetresetclause(ctx *FunctionsetresetclauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#variableshowstmt.
	VisitVariableshowstmt(ctx *VariableshowstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#constraintssetstmt.
	VisitConstraintssetstmt(ctx *ConstraintssetstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#constraints_set_list.
	VisitConstraints_set_list(ctx *Constraints_set_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#constraints_set_mode.
	VisitConstraints_set_mode(ctx *Constraints_set_modeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#checkpointstmt.
	VisitCheckpointstmt(ctx *CheckpointstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#discardstmt.
	VisitDiscardstmt(ctx *DiscardstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#altertablestmt.
	VisitAltertablestmt(ctx *AltertablestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alter_table_cmds.
	VisitAlter_table_cmds(ctx *Alter_table_cmdsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#partition_cmd.
	VisitPartition_cmd(ctx *Partition_cmdContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#index_partition_cmd.
	VisitIndex_partition_cmd(ctx *Index_partition_cmdContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alter_table_cmd.
	VisitAlter_table_cmd(ctx *Alter_table_cmdContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alter_column_default.
	VisitAlter_column_default(ctx *Alter_column_defaultContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_drop_behavior.
	VisitOpt_drop_behavior(ctx *Opt_drop_behaviorContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_collate_clause.
	VisitOpt_collate_clause(ctx *Opt_collate_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alter_using.
	VisitAlter_using(ctx *Alter_usingContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#replica_identity.
	VisitReplica_identity(ctx *Replica_identityContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#reloptions.
	VisitReloptions(ctx *ReloptionsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_reloptions.
	VisitOpt_reloptions(ctx *Opt_reloptionsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#reloption_list.
	VisitReloption_list(ctx *Reloption_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#reloption_elem.
	VisitReloption_elem(ctx *Reloption_elemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alter_identity_column_option_list.
	VisitAlter_identity_column_option_list(ctx *Alter_identity_column_option_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alter_identity_column_option.
	VisitAlter_identity_column_option(ctx *Alter_identity_column_optionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#partitionboundspec.
	VisitPartitionboundspec(ctx *PartitionboundspecContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#hash_partbound_elem.
	VisitHash_partbound_elem(ctx *Hash_partbound_elemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#hash_partbound.
	VisitHash_partbound(ctx *Hash_partboundContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#altercompositetypestmt.
	VisitAltercompositetypestmt(ctx *AltercompositetypestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alter_type_cmds.
	VisitAlter_type_cmds(ctx *Alter_type_cmdsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alter_type_cmd.
	VisitAlter_type_cmd(ctx *Alter_type_cmdContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#closeportalstmt.
	VisitCloseportalstmt(ctx *CloseportalstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#copystmt.
	VisitCopystmt(ctx *CopystmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#copy_from.
	VisitCopy_from(ctx *Copy_fromContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_program.
	VisitOpt_program(ctx *Opt_programContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#copy_file_name.
	VisitCopy_file_name(ctx *Copy_file_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#copy_options.
	VisitCopy_options(ctx *Copy_optionsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#copy_opt_list.
	VisitCopy_opt_list(ctx *Copy_opt_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#copy_opt_item.
	VisitCopy_opt_item(ctx *Copy_opt_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_binary.
	VisitOpt_binary(ctx *Opt_binaryContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#copy_delimiter.
	VisitCopy_delimiter(ctx *Copy_delimiterContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_using.
	VisitOpt_using(ctx *Opt_usingContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#copy_generic_opt_list.
	VisitCopy_generic_opt_list(ctx *Copy_generic_opt_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#copy_generic_opt_elem.
	VisitCopy_generic_opt_elem(ctx *Copy_generic_opt_elemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#copy_generic_opt_arg.
	VisitCopy_generic_opt_arg(ctx *Copy_generic_opt_argContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#copy_generic_opt_arg_list.
	VisitCopy_generic_opt_arg_list(ctx *Copy_generic_opt_arg_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#copy_generic_opt_arg_list_item.
	VisitCopy_generic_opt_arg_list_item(ctx *Copy_generic_opt_arg_list_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createstmt.
	VisitCreatestmt(ctx *CreatestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opttemp.
	VisitOpttemp(ctx *OpttempContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opttableelementlist.
	VisitOpttableelementlist(ctx *OpttableelementlistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opttypedtableelementlist.
	VisitOpttypedtableelementlist(ctx *OpttypedtableelementlistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#tableelementlist.
	VisitTableelementlist(ctx *TableelementlistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#typedtableelementlist.
	VisitTypedtableelementlist(ctx *TypedtableelementlistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#tableelement.
	VisitTableelement(ctx *TableelementContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#typedtableelement.
	VisitTypedtableelement(ctx *TypedtableelementContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#columnDef.
	VisitColumnDef(ctx *ColumnDefContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#columnOptions.
	VisitColumnOptions(ctx *ColumnOptionsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#colquallist.
	VisitColquallist(ctx *ColquallistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#colconstraint.
	VisitColconstraint(ctx *ColconstraintContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#colconstraintelem.
	VisitColconstraintelem(ctx *ColconstraintelemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#generated_when.
	VisitGenerated_when(ctx *Generated_whenContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#constraintattr.
	VisitConstraintattr(ctx *ConstraintattrContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#tablelikeclause.
	VisitTablelikeclause(ctx *TablelikeclauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#tablelikeoptionlist.
	VisitTablelikeoptionlist(ctx *TablelikeoptionlistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#tablelikeoption.
	VisitTablelikeoption(ctx *TablelikeoptionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#tableconstraint.
	VisitTableconstraint(ctx *TableconstraintContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#constraintelem.
	VisitConstraintelem(ctx *ConstraintelemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_no_inherit.
	VisitOpt_no_inherit(ctx *Opt_no_inheritContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_column_list.
	VisitOpt_column_list(ctx *Opt_column_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#columnlist.
	VisitColumnlist(ctx *ColumnlistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#columnElem.
	VisitColumnElem(ctx *ColumnElemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_c_include.
	VisitOpt_c_include(ctx *Opt_c_includeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#key_match.
	VisitKey_match(ctx *Key_matchContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#exclusionconstraintlist.
	VisitExclusionconstraintlist(ctx *ExclusionconstraintlistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#exclusionconstraintelem.
	VisitExclusionconstraintelem(ctx *ExclusionconstraintelemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#exclusionwhereclause.
	VisitExclusionwhereclause(ctx *ExclusionwhereclauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#key_actions.
	VisitKey_actions(ctx *Key_actionsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#key_update.
	VisitKey_update(ctx *Key_updateContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#key_delete.
	VisitKey_delete(ctx *Key_deleteContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#key_action.
	VisitKey_action(ctx *Key_actionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#optinherit.
	VisitOptinherit(ctx *OptinheritContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#optpartitionspec.
	VisitOptpartitionspec(ctx *OptpartitionspecContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#partitionspec.
	VisitPartitionspec(ctx *PartitionspecContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#part_params.
	VisitPart_params(ctx *Part_paramsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#part_elem.
	VisitPart_elem(ctx *Part_elemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#table_access_method_clause.
	VisitTable_access_method_clause(ctx *Table_access_method_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#optwith.
	VisitOptwith(ctx *OptwithContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#oncommitoption.
	VisitOncommitoption(ctx *OncommitoptionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opttablespace.
	VisitOpttablespace(ctx *OpttablespaceContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#optconstablespace.
	VisitOptconstablespace(ctx *OptconstablespaceContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#existingindex.
	VisitExistingindex(ctx *ExistingindexContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createstatsstmt.
	VisitCreatestatsstmt(ctx *CreatestatsstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterstatsstmt.
	VisitAlterstatsstmt(ctx *AlterstatsstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createasstmt.
	VisitCreateasstmt(ctx *CreateasstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#create_as_target.
	VisitCreate_as_target(ctx *Create_as_targetContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_with_data.
	VisitOpt_with_data(ctx *Opt_with_dataContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#creatematviewstmt.
	VisitCreatematviewstmt(ctx *CreatematviewstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#create_mv_target.
	VisitCreate_mv_target(ctx *Create_mv_targetContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#optnolog.
	VisitOptnolog(ctx *OptnologContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#refreshmatviewstmt.
	VisitRefreshmatviewstmt(ctx *RefreshmatviewstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createseqstmt.
	VisitCreateseqstmt(ctx *CreateseqstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterseqstmt.
	VisitAlterseqstmt(ctx *AlterseqstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#optseqoptlist.
	VisitOptseqoptlist(ctx *OptseqoptlistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#optparenthesizedseqoptlist.
	VisitOptparenthesizedseqoptlist(ctx *OptparenthesizedseqoptlistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#seqoptlist.
	VisitSeqoptlist(ctx *SeqoptlistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#seqoptelem.
	VisitSeqoptelem(ctx *SeqoptelemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_by.
	VisitOpt_by(ctx *Opt_byContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#numericonly.
	VisitNumericonly(ctx *NumericonlyContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#numericonly_list.
	VisitNumericonly_list(ctx *Numericonly_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createplangstmt.
	VisitCreateplangstmt(ctx *CreateplangstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_trusted.
	VisitOpt_trusted(ctx *Opt_trustedContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#handler_name.
	VisitHandler_name(ctx *Handler_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_inline_handler.
	VisitOpt_inline_handler(ctx *Opt_inline_handlerContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#validator_clause.
	VisitValidator_clause(ctx *Validator_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_validator.
	VisitOpt_validator(ctx *Opt_validatorContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_procedural.
	VisitOpt_procedural(ctx *Opt_proceduralContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createtablespacestmt.
	VisitCreatetablespacestmt(ctx *CreatetablespacestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opttablespaceowner.
	VisitOpttablespaceowner(ctx *OpttablespaceownerContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#droptablespacestmt.
	VisitDroptablespacestmt(ctx *DroptablespacestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createextensionstmt.
	VisitCreateextensionstmt(ctx *CreateextensionstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#create_extension_opt_list.
	VisitCreate_extension_opt_list(ctx *Create_extension_opt_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#create_extension_opt_item.
	VisitCreate_extension_opt_item(ctx *Create_extension_opt_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterextensionstmt.
	VisitAlterextensionstmt(ctx *AlterextensionstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alter_extension_opt_list.
	VisitAlter_extension_opt_list(ctx *Alter_extension_opt_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alter_extension_opt_item.
	VisitAlter_extension_opt_item(ctx *Alter_extension_opt_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterextensioncontentsstmt.
	VisitAlterextensioncontentsstmt(ctx *AlterextensioncontentsstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createfdwstmt.
	VisitCreatefdwstmt(ctx *CreatefdwstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#fdw_option.
	VisitFdw_option(ctx *Fdw_optionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#fdw_options.
	VisitFdw_options(ctx *Fdw_optionsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_fdw_options.
	VisitOpt_fdw_options(ctx *Opt_fdw_optionsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterfdwstmt.
	VisitAlterfdwstmt(ctx *AlterfdwstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#create_generic_options.
	VisitCreate_generic_options(ctx *Create_generic_optionsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#generic_option_list.
	VisitGeneric_option_list(ctx *Generic_option_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alter_generic_options.
	VisitAlter_generic_options(ctx *Alter_generic_optionsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alter_generic_option_list.
	VisitAlter_generic_option_list(ctx *Alter_generic_option_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alter_generic_option_elem.
	VisitAlter_generic_option_elem(ctx *Alter_generic_option_elemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#generic_option_elem.
	VisitGeneric_option_elem(ctx *Generic_option_elemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#generic_option_name.
	VisitGeneric_option_name(ctx *Generic_option_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#generic_option_arg.
	VisitGeneric_option_arg(ctx *Generic_option_argContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createforeignserverstmt.
	VisitCreateforeignserverstmt(ctx *CreateforeignserverstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_type.
	VisitOpt_type(ctx *Opt_typeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#foreign_server_version.
	VisitForeign_server_version(ctx *Foreign_server_versionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_foreign_server_version.
	VisitOpt_foreign_server_version(ctx *Opt_foreign_server_versionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterforeignserverstmt.
	VisitAlterforeignserverstmt(ctx *AlterforeignserverstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createforeigntablestmt.
	VisitCreateforeigntablestmt(ctx *CreateforeigntablestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#importforeignschemastmt.
	VisitImportforeignschemastmt(ctx *ImportforeignschemastmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#import_qualification_type.
	VisitImport_qualification_type(ctx *Import_qualification_typeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#import_qualification.
	VisitImport_qualification(ctx *Import_qualificationContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createusermappingstmt.
	VisitCreateusermappingstmt(ctx *CreateusermappingstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#auth_ident.
	VisitAuth_ident(ctx *Auth_identContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#dropusermappingstmt.
	VisitDropusermappingstmt(ctx *DropusermappingstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterusermappingstmt.
	VisitAlterusermappingstmt(ctx *AlterusermappingstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createpolicystmt.
	VisitCreatepolicystmt(ctx *CreatepolicystmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterpolicystmt.
	VisitAlterpolicystmt(ctx *AlterpolicystmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#rowsecurityoptionalexpr.
	VisitRowsecurityoptionalexpr(ctx *RowsecurityoptionalexprContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#rowsecurityoptionalwithcheck.
	VisitRowsecurityoptionalwithcheck(ctx *RowsecurityoptionalwithcheckContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#rowsecuritydefaulttorole.
	VisitRowsecuritydefaulttorole(ctx *RowsecuritydefaulttoroleContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#rowsecurityoptionaltorole.
	VisitRowsecurityoptionaltorole(ctx *RowsecurityoptionaltoroleContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#rowsecuritydefaultpermissive.
	VisitRowsecuritydefaultpermissive(ctx *RowsecuritydefaultpermissiveContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#rowsecuritydefaultforcmd.
	VisitRowsecuritydefaultforcmd(ctx *RowsecuritydefaultforcmdContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#row_security_cmd.
	VisitRow_security_cmd(ctx *Row_security_cmdContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createamstmt.
	VisitCreateamstmt(ctx *CreateamstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#am_type.
	VisitAm_type(ctx *Am_typeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createtrigstmt.
	VisitCreatetrigstmt(ctx *CreatetrigstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#triggeractiontime.
	VisitTriggeractiontime(ctx *TriggeractiontimeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#triggerevents.
	VisitTriggerevents(ctx *TriggereventsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#triggeroneevent.
	VisitTriggeroneevent(ctx *TriggeroneeventContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#triggerreferencing.
	VisitTriggerreferencing(ctx *TriggerreferencingContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#triggertransitions.
	VisitTriggertransitions(ctx *TriggertransitionsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#triggertransition.
	VisitTriggertransition(ctx *TriggertransitionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#transitionoldornew.
	VisitTransitionoldornew(ctx *TransitionoldornewContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#transitionrowortable.
	VisitTransitionrowortable(ctx *TransitionrowortableContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#transitionrelname.
	VisitTransitionrelname(ctx *TransitionrelnameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#triggerforspec.
	VisitTriggerforspec(ctx *TriggerforspecContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#triggerforopteach.
	VisitTriggerforopteach(ctx *TriggerforopteachContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#triggerfortype.
	VisitTriggerfortype(ctx *TriggerfortypeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#triggerwhen.
	VisitTriggerwhen(ctx *TriggerwhenContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#function_or_procedure.
	VisitFunction_or_procedure(ctx *Function_or_procedureContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#triggerfuncargs.
	VisitTriggerfuncargs(ctx *TriggerfuncargsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#triggerfuncarg.
	VisitTriggerfuncarg(ctx *TriggerfuncargContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#optconstrfromtable.
	VisitOptconstrfromtable(ctx *OptconstrfromtableContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#constraintattributespec.
	VisitConstraintattributespec(ctx *ConstraintattributespecContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#constraintattributeElem.
	VisitConstraintattributeElem(ctx *ConstraintattributeElemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createeventtrigstmt.
	VisitCreateeventtrigstmt(ctx *CreateeventtrigstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#event_trigger_when_list.
	VisitEvent_trigger_when_list(ctx *Event_trigger_when_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#event_trigger_when_item.
	VisitEvent_trigger_when_item(ctx *Event_trigger_when_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#event_trigger_value_list.
	VisitEvent_trigger_value_list(ctx *Event_trigger_value_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#altereventtrigstmt.
	VisitAltereventtrigstmt(ctx *AltereventtrigstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#enable_trigger.
	VisitEnable_trigger(ctx *Enable_triggerContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createassertionstmt.
	VisitCreateassertionstmt(ctx *CreateassertionstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#definestmt.
	VisitDefinestmt(ctx *DefinestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#definition.
	VisitDefinition(ctx *DefinitionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#def_list.
	VisitDef_list(ctx *Def_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#def_elem.
	VisitDef_elem(ctx *Def_elemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#def_arg.
	VisitDef_arg(ctx *Def_argContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#old_aggr_definition.
	VisitOld_aggr_definition(ctx *Old_aggr_definitionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#old_aggr_list.
	VisitOld_aggr_list(ctx *Old_aggr_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#old_aggr_elem.
	VisitOld_aggr_elem(ctx *Old_aggr_elemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_enum_val_list.
	VisitOpt_enum_val_list(ctx *Opt_enum_val_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#enum_val_list.
	VisitEnum_val_list(ctx *Enum_val_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterenumstmt.
	VisitAlterenumstmt(ctx *AlterenumstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_if_not_exists.
	VisitOpt_if_not_exists(ctx *Opt_if_not_existsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createopclassstmt.
	VisitCreateopclassstmt(ctx *CreateopclassstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opclass_item_list.
	VisitOpclass_item_list(ctx *Opclass_item_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opclass_item.
	VisitOpclass_item(ctx *Opclass_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_default.
	VisitOpt_default(ctx *Opt_defaultContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_opfamily.
	VisitOpt_opfamily(ctx *Opt_opfamilyContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opclass_purpose.
	VisitOpclass_purpose(ctx *Opclass_purposeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_recheck.
	VisitOpt_recheck(ctx *Opt_recheckContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createopfamilystmt.
	VisitCreateopfamilystmt(ctx *CreateopfamilystmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alteropfamilystmt.
	VisitAlteropfamilystmt(ctx *AlteropfamilystmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opclass_drop_list.
	VisitOpclass_drop_list(ctx *Opclass_drop_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opclass_drop.
	VisitOpclass_drop(ctx *Opclass_dropContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#dropopclassstmt.
	VisitDropopclassstmt(ctx *DropopclassstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#dropopfamilystmt.
	VisitDropopfamilystmt(ctx *DropopfamilystmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#dropownedstmt.
	VisitDropownedstmt(ctx *DropownedstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#reassignownedstmt.
	VisitReassignownedstmt(ctx *ReassignownedstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#dropstmt.
	VisitDropstmt(ctx *DropstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#object_type_any_name.
	VisitObject_type_any_name(ctx *Object_type_any_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#object_type_name.
	VisitObject_type_name(ctx *Object_type_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#drop_type_name.
	VisitDrop_type_name(ctx *Drop_type_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#object_type_name_on_any_name.
	VisitObject_type_name_on_any_name(ctx *Object_type_name_on_any_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#any_name_list.
	VisitAny_name_list(ctx *Any_name_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#any_name.
	VisitAny_name(ctx *Any_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#attrs.
	VisitAttrs(ctx *AttrsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#type_name_list.
	VisitType_name_list(ctx *Type_name_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#truncatestmt.
	VisitTruncatestmt(ctx *TruncatestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_restart_seqs.
	VisitOpt_restart_seqs(ctx *Opt_restart_seqsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#commentstmt.
	VisitCommentstmt(ctx *CommentstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#comment_text.
	VisitComment_text(ctx *Comment_textContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#seclabelstmt.
	VisitSeclabelstmt(ctx *SeclabelstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_provider.
	VisitOpt_provider(ctx *Opt_providerContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#security_label.
	VisitSecurity_label(ctx *Security_labelContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#fetchstmt.
	VisitFetchstmt(ctx *FetchstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#fetch_args.
	VisitFetch_args(ctx *Fetch_argsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#from_in.
	VisitFrom_in(ctx *From_inContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_from_in.
	VisitOpt_from_in(ctx *Opt_from_inContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#grantstmt.
	VisitGrantstmt(ctx *GrantstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#revokestmt.
	VisitRevokestmt(ctx *RevokestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#privileges.
	VisitPrivileges(ctx *PrivilegesContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#privilege_list.
	VisitPrivilege_list(ctx *Privilege_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#privilege.
	VisitPrivilege(ctx *PrivilegeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#privilege_target.
	VisitPrivilege_target(ctx *Privilege_targetContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#grantee_list.
	VisitGrantee_list(ctx *Grantee_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#grantee.
	VisitGrantee(ctx *GranteeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_grant_grant_option.
	VisitOpt_grant_grant_option(ctx *Opt_grant_grant_optionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#grantrolestmt.
	VisitGrantrolestmt(ctx *GrantrolestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#revokerolestmt.
	VisitRevokerolestmt(ctx *RevokerolestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_grant_admin_option.
	VisitOpt_grant_admin_option(ctx *Opt_grant_admin_optionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_granted_by.
	VisitOpt_granted_by(ctx *Opt_granted_byContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterdefaultprivilegesstmt.
	VisitAlterdefaultprivilegesstmt(ctx *AlterdefaultprivilegesstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#defacloptionlist.
	VisitDefacloptionlist(ctx *DefacloptionlistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#defacloption.
	VisitDefacloption(ctx *DefacloptionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#defaclaction.
	VisitDefaclaction(ctx *DefaclactionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#defacl_privilege_target.
	VisitDefacl_privilege_target(ctx *Defacl_privilege_targetContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#indexstmt.
	VisitIndexstmt(ctx *IndexstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_unique.
	VisitOpt_unique(ctx *Opt_uniqueContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_concurrently.
	VisitOpt_concurrently(ctx *Opt_concurrentlyContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_index_name.
	VisitOpt_index_name(ctx *Opt_index_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#access_method_clause.
	VisitAccess_method_clause(ctx *Access_method_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#index_params.
	VisitIndex_params(ctx *Index_paramsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#index_elem_options.
	VisitIndex_elem_options(ctx *Index_elem_optionsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#index_elem.
	VisitIndex_elem(ctx *Index_elemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_include.
	VisitOpt_include(ctx *Opt_includeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#index_including_params.
	VisitIndex_including_params(ctx *Index_including_paramsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_collate.
	VisitOpt_collate(ctx *Opt_collateContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_class.
	VisitOpt_class(ctx *Opt_classContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_asc_desc.
	VisitOpt_asc_desc(ctx *Opt_asc_descContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_nulls_order.
	VisitOpt_nulls_order(ctx *Opt_nulls_orderContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createfunctionstmt.
	VisitCreatefunctionstmt(ctx *CreatefunctionstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_or_replace.
	VisitOpt_or_replace(ctx *Opt_or_replaceContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_args.
	VisitFunc_args(ctx *Func_argsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_args_list.
	VisitFunc_args_list(ctx *Func_args_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#function_with_argtypes_list.
	VisitFunction_with_argtypes_list(ctx *Function_with_argtypes_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#function_with_argtypes.
	VisitFunction_with_argtypes(ctx *Function_with_argtypesContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_args_with_defaults.
	VisitFunc_args_with_defaults(ctx *Func_args_with_defaultsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_args_with_defaults_list.
	VisitFunc_args_with_defaults_list(ctx *Func_args_with_defaults_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_arg.
	VisitFunc_arg(ctx *Func_argContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#arg_class.
	VisitArg_class(ctx *Arg_classContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#param_name.
	VisitParam_name(ctx *Param_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_return.
	VisitFunc_return(ctx *Func_returnContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_type.
	VisitFunc_type(ctx *Func_typeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_arg_with_default.
	VisitFunc_arg_with_default(ctx *Func_arg_with_defaultContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#aggr_arg.
	VisitAggr_arg(ctx *Aggr_argContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#aggr_args.
	VisitAggr_args(ctx *Aggr_argsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#aggr_args_list.
	VisitAggr_args_list(ctx *Aggr_args_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#aggregate_with_argtypes.
	VisitAggregate_with_argtypes(ctx *Aggregate_with_argtypesContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#aggregate_with_argtypes_list.
	VisitAggregate_with_argtypes_list(ctx *Aggregate_with_argtypes_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createfunc_opt_list.
	VisitCreatefunc_opt_list(ctx *Createfunc_opt_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#common_func_opt_item.
	VisitCommon_func_opt_item(ctx *Common_func_opt_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createfunc_opt_item.
	VisitCreatefunc_opt_item(ctx *Createfunc_opt_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_as.
	VisitFunc_as(ctx *Func_asContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#transform_type_list.
	VisitTransform_type_list(ctx *Transform_type_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_definition.
	VisitOpt_definition(ctx *Opt_definitionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#table_func_column.
	VisitTable_func_column(ctx *Table_func_columnContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#table_func_column_list.
	VisitTable_func_column_list(ctx *Table_func_column_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterfunctionstmt.
	VisitAlterfunctionstmt(ctx *AlterfunctionstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterfunc_opt_list.
	VisitAlterfunc_opt_list(ctx *Alterfunc_opt_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_restrict.
	VisitOpt_restrict(ctx *Opt_restrictContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#removefuncstmt.
	VisitRemovefuncstmt(ctx *RemovefuncstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#removeaggrstmt.
	VisitRemoveaggrstmt(ctx *RemoveaggrstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#removeoperstmt.
	VisitRemoveoperstmt(ctx *RemoveoperstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#oper_argtypes.
	VisitOper_argtypes(ctx *Oper_argtypesContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#any_operator.
	VisitAny_operator(ctx *Any_operatorContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#operator_with_argtypes_list.
	VisitOperator_with_argtypes_list(ctx *Operator_with_argtypes_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#operator_with_argtypes.
	VisitOperator_with_argtypes(ctx *Operator_with_argtypesContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#dostmt.
	VisitDostmt(ctx *DostmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#dostmt_opt_list.
	VisitDostmt_opt_list(ctx *Dostmt_opt_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#dostmt_opt_item.
	VisitDostmt_opt_item(ctx *Dostmt_opt_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createcaststmt.
	VisitCreatecaststmt(ctx *CreatecaststmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#cast_context.
	VisitCast_context(ctx *Cast_contextContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#dropcaststmt.
	VisitDropcaststmt(ctx *DropcaststmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_if_exists.
	VisitOpt_if_exists(ctx *Opt_if_existsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createtransformstmt.
	VisitCreatetransformstmt(ctx *CreatetransformstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#transform_element_list.
	VisitTransform_element_list(ctx *Transform_element_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#droptransformstmt.
	VisitDroptransformstmt(ctx *DroptransformstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#reindexstmt.
	VisitReindexstmt(ctx *ReindexstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#reindex_target_type.
	VisitReindex_target_type(ctx *Reindex_target_typeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#reindex_target_multitable.
	VisitReindex_target_multitable(ctx *Reindex_target_multitableContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#reindex_option_list.
	VisitReindex_option_list(ctx *Reindex_option_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#reindex_option_elem.
	VisitReindex_option_elem(ctx *Reindex_option_elemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#altertblspcstmt.
	VisitAltertblspcstmt(ctx *AltertblspcstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#renamestmt.
	VisitRenamestmt(ctx *RenamestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_column.
	VisitOpt_column(ctx *Opt_columnContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_set_data.
	VisitOpt_set_data(ctx *Opt_set_dataContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterobjectdependsstmt.
	VisitAlterobjectdependsstmt(ctx *AlterobjectdependsstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_no.
	VisitOpt_no(ctx *Opt_noContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterobjectschemastmt.
	VisitAlterobjectschemastmt(ctx *AlterobjectschemastmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alteroperatorstmt.
	VisitAlteroperatorstmt(ctx *AlteroperatorstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#operator_def_list.
	VisitOperator_def_list(ctx *Operator_def_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#operator_def_elem.
	VisitOperator_def_elem(ctx *Operator_def_elemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#operator_def_arg.
	VisitOperator_def_arg(ctx *Operator_def_argContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#altertypestmt.
	VisitAltertypestmt(ctx *AltertypestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterownerstmt.
	VisitAlterownerstmt(ctx *AlterownerstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createpublicationstmt.
	VisitCreatepublicationstmt(ctx *CreatepublicationstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_publication_for_tables.
	VisitOpt_publication_for_tables(ctx *Opt_publication_for_tablesContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#publication_for_tables.
	VisitPublication_for_tables(ctx *Publication_for_tablesContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterpublicationstmt.
	VisitAlterpublicationstmt(ctx *AlterpublicationstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createsubscriptionstmt.
	VisitCreatesubscriptionstmt(ctx *CreatesubscriptionstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#publication_name_list.
	VisitPublication_name_list(ctx *Publication_name_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#publication_name_item.
	VisitPublication_name_item(ctx *Publication_name_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#altersubscriptionstmt.
	VisitAltersubscriptionstmt(ctx *AltersubscriptionstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#dropsubscriptionstmt.
	VisitDropsubscriptionstmt(ctx *DropsubscriptionstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#rulestmt.
	VisitRulestmt(ctx *RulestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#ruleactionlist.
	VisitRuleactionlist(ctx *RuleactionlistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#ruleactionmulti.
	VisitRuleactionmulti(ctx *RuleactionmultiContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#ruleactionstmt.
	VisitRuleactionstmt(ctx *RuleactionstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#ruleactionstmtOrEmpty.
	VisitRuleactionstmtOrEmpty(ctx *RuleactionstmtOrEmptyContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#event.
	VisitEvent(ctx *EventContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_instead.
	VisitOpt_instead(ctx *Opt_insteadContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#notifystmt.
	VisitNotifystmt(ctx *NotifystmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#notify_payload.
	VisitNotify_payload(ctx *Notify_payloadContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#listenstmt.
	VisitListenstmt(ctx *ListenstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#unlistenstmt.
	VisitUnlistenstmt(ctx *UnlistenstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#transactionstmt.
	VisitTransactionstmt(ctx *TransactionstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_transaction.
	VisitOpt_transaction(ctx *Opt_transactionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#transaction_mode_item.
	VisitTransaction_mode_item(ctx *Transaction_mode_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#transaction_mode_list.
	VisitTransaction_mode_list(ctx *Transaction_mode_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#transaction_mode_list_or_empty.
	VisitTransaction_mode_list_or_empty(ctx *Transaction_mode_list_or_emptyContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_transaction_chain.
	VisitOpt_transaction_chain(ctx *Opt_transaction_chainContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#viewstmt.
	VisitViewstmt(ctx *ViewstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_check_option.
	VisitOpt_check_option(ctx *Opt_check_optionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#loadstmt.
	VisitLoadstmt(ctx *LoadstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createdbstmt.
	VisitCreatedbstmt(ctx *CreatedbstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createdb_opt_list.
	VisitCreatedb_opt_list(ctx *Createdb_opt_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createdb_opt_items.
	VisitCreatedb_opt_items(ctx *Createdb_opt_itemsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createdb_opt_item.
	VisitCreatedb_opt_item(ctx *Createdb_opt_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createdb_opt_name.
	VisitCreatedb_opt_name(ctx *Createdb_opt_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_equal.
	VisitOpt_equal(ctx *Opt_equalContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterdatabasestmt.
	VisitAlterdatabasestmt(ctx *AlterdatabasestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterdatabasesetstmt.
	VisitAlterdatabasesetstmt(ctx *AlterdatabasesetstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#dropdbstmt.
	VisitDropdbstmt(ctx *DropdbstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#drop_option_list.
	VisitDrop_option_list(ctx *Drop_option_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#drop_option.
	VisitDrop_option(ctx *Drop_optionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#altercollationstmt.
	VisitAltercollationstmt(ctx *AltercollationstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#altersystemstmt.
	VisitAltersystemstmt(ctx *AltersystemstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createdomainstmt.
	VisitCreatedomainstmt(ctx *CreatedomainstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alterdomainstmt.
	VisitAlterdomainstmt(ctx *AlterdomainstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_as.
	VisitOpt_as(ctx *Opt_asContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#altertsdictionarystmt.
	VisitAltertsdictionarystmt(ctx *AltertsdictionarystmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#altertsconfigurationstmt.
	VisitAltertsconfigurationstmt(ctx *AltertsconfigurationstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#any_with.
	VisitAny_with(ctx *Any_withContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#createconversionstmt.
	VisitCreateconversionstmt(ctx *CreateconversionstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#clusterstmt.
	VisitClusterstmt(ctx *ClusterstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#cluster_index_specification.
	VisitCluster_index_specification(ctx *Cluster_index_specificationContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#vacuumstmt.
	VisitVacuumstmt(ctx *VacuumstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#analyzestmt.
	VisitAnalyzestmt(ctx *AnalyzestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#vac_analyze_option_list.
	VisitVac_analyze_option_list(ctx *Vac_analyze_option_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#analyze_keyword.
	VisitAnalyze_keyword(ctx *Analyze_keywordContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#vac_analyze_option_elem.
	VisitVac_analyze_option_elem(ctx *Vac_analyze_option_elemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#vac_analyze_option_name.
	VisitVac_analyze_option_name(ctx *Vac_analyze_option_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#vac_analyze_option_arg.
	VisitVac_analyze_option_arg(ctx *Vac_analyze_option_argContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_analyze.
	VisitOpt_analyze(ctx *Opt_analyzeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_verbose.
	VisitOpt_verbose(ctx *Opt_verboseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_full.
	VisitOpt_full(ctx *Opt_fullContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_freeze.
	VisitOpt_freeze(ctx *Opt_freezeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_name_list.
	VisitOpt_name_list(ctx *Opt_name_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#vacuum_relation.
	VisitVacuum_relation(ctx *Vacuum_relationContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#vacuum_relation_list.
	VisitVacuum_relation_list(ctx *Vacuum_relation_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_vacuum_relation_list.
	VisitOpt_vacuum_relation_list(ctx *Opt_vacuum_relation_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#explainstmt.
	VisitExplainstmt(ctx *ExplainstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#explainablestmt.
	VisitExplainablestmt(ctx *ExplainablestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#explain_option_list.
	VisitExplain_option_list(ctx *Explain_option_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#explain_option_elem.
	VisitExplain_option_elem(ctx *Explain_option_elemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#explain_option_name.
	VisitExplain_option_name(ctx *Explain_option_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#explain_option_arg.
	VisitExplain_option_arg(ctx *Explain_option_argContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#preparestmt.
	VisitPreparestmt(ctx *PreparestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#prep_type_clause.
	VisitPrep_type_clause(ctx *Prep_type_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#preparablestmt.
	VisitPreparablestmt(ctx *PreparablestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#executestmt.
	VisitExecutestmt(ctx *ExecutestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#execute_param_clause.
	VisitExecute_param_clause(ctx *Execute_param_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#deallocatestmt.
	VisitDeallocatestmt(ctx *DeallocatestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#insertstmt.
	VisitInsertstmt(ctx *InsertstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#insert_target.
	VisitInsert_target(ctx *Insert_targetContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#insert_rest.
	VisitInsert_rest(ctx *Insert_restContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#override_kind.
	VisitOverride_kind(ctx *Override_kindContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#insert_column_list.
	VisitInsert_column_list(ctx *Insert_column_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#insert_column_item.
	VisitInsert_column_item(ctx *Insert_column_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_on_conflict.
	VisitOpt_on_conflict(ctx *Opt_on_conflictContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_conf_expr.
	VisitOpt_conf_expr(ctx *Opt_conf_exprContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#returning_clause.
	VisitReturning_clause(ctx *Returning_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#mergestmt.
	VisitMergestmt(ctx *MergestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#merge_insert_clause.
	VisitMerge_insert_clause(ctx *Merge_insert_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#merge_update_clause.
	VisitMerge_update_clause(ctx *Merge_update_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#merge_delete_clause.
	VisitMerge_delete_clause(ctx *Merge_delete_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#deletestmt.
	VisitDeletestmt(ctx *DeletestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#using_clause.
	VisitUsing_clause(ctx *Using_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#lockstmt.
	VisitLockstmt(ctx *LockstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_lock.
	VisitOpt_lock(ctx *Opt_lockContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#lock_type.
	VisitLock_type(ctx *Lock_typeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_nowait.
	VisitOpt_nowait(ctx *Opt_nowaitContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_nowait_or_skip.
	VisitOpt_nowait_or_skip(ctx *Opt_nowait_or_skipContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#updatestmt.
	VisitUpdatestmt(ctx *UpdatestmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#set_clause_list.
	VisitSet_clause_list(ctx *Set_clause_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#set_clause.
	VisitSet_clause(ctx *Set_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#set_target.
	VisitSet_target(ctx *Set_targetContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#set_target_list.
	VisitSet_target_list(ctx *Set_target_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#declarecursorstmt.
	VisitDeclarecursorstmt(ctx *DeclarecursorstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#cursor_name.
	VisitCursor_name(ctx *Cursor_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#cursor_options.
	VisitCursor_options(ctx *Cursor_optionsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_hold.
	VisitOpt_hold(ctx *Opt_holdContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#selectstmt.
	VisitSelectstmt(ctx *SelectstmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#select_with_parens.
	VisitSelect_with_parens(ctx *Select_with_parensContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#select_no_parens.
	VisitSelect_no_parens(ctx *Select_no_parensContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#select_clause.
	VisitSelect_clause(ctx *Select_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#simple_select_intersect.
	VisitSimple_select_intersect(ctx *Simple_select_intersectContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#simple_select_pramary.
	VisitSimple_select_pramary(ctx *Simple_select_pramaryContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#with_clause.
	VisitWith_clause(ctx *With_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#cte_list.
	VisitCte_list(ctx *Cte_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#common_table_expr.
	VisitCommon_table_expr(ctx *Common_table_exprContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_materialized.
	VisitOpt_materialized(ctx *Opt_materializedContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_with_clause.
	VisitOpt_with_clause(ctx *Opt_with_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#into_clause.
	VisitInto_clause(ctx *Into_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_strict.
	VisitOpt_strict(ctx *Opt_strictContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opttempTableName.
	VisitOpttempTableName(ctx *OpttempTableNameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_table.
	VisitOpt_table(ctx *Opt_tableContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#all_or_distinct.
	VisitAll_or_distinct(ctx *All_or_distinctContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#distinct_clause.
	VisitDistinct_clause(ctx *Distinct_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_all_clause.
	VisitOpt_all_clause(ctx *Opt_all_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_sort_clause.
	VisitOpt_sort_clause(ctx *Opt_sort_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#sort_clause.
	VisitSort_clause(ctx *Sort_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#sortby_list.
	VisitSortby_list(ctx *Sortby_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#sortby.
	VisitSortby(ctx *SortbyContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#select_limit.
	VisitSelect_limit(ctx *Select_limitContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_select_limit.
	VisitOpt_select_limit(ctx *Opt_select_limitContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#limit_clause.
	VisitLimit_clause(ctx *Limit_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#offset_clause.
	VisitOffset_clause(ctx *Offset_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#select_limit_value.
	VisitSelect_limit_value(ctx *Select_limit_valueContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#select_offset_value.
	VisitSelect_offset_value(ctx *Select_offset_valueContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#select_fetch_first_value.
	VisitSelect_fetch_first_value(ctx *Select_fetch_first_valueContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#i_or_f_const.
	VisitI_or_f_const(ctx *I_or_f_constContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#row_or_rows.
	VisitRow_or_rows(ctx *Row_or_rowsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#first_or_next.
	VisitFirst_or_next(ctx *First_or_nextContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#group_clause.
	VisitGroup_clause(ctx *Group_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#group_by_list.
	VisitGroup_by_list(ctx *Group_by_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#group_by_item.
	VisitGroup_by_item(ctx *Group_by_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#empty_grouping_set.
	VisitEmpty_grouping_set(ctx *Empty_grouping_setContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#rollup_clause.
	VisitRollup_clause(ctx *Rollup_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#cube_clause.
	VisitCube_clause(ctx *Cube_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#grouping_sets_clause.
	VisitGrouping_sets_clause(ctx *Grouping_sets_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#having_clause.
	VisitHaving_clause(ctx *Having_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#for_locking_clause.
	VisitFor_locking_clause(ctx *For_locking_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_for_locking_clause.
	VisitOpt_for_locking_clause(ctx *Opt_for_locking_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#for_locking_items.
	VisitFor_locking_items(ctx *For_locking_itemsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#for_locking_item.
	VisitFor_locking_item(ctx *For_locking_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#for_locking_strength.
	VisitFor_locking_strength(ctx *For_locking_strengthContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#locked_rels_list.
	VisitLocked_rels_list(ctx *Locked_rels_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#values_clause.
	VisitValues_clause(ctx *Values_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#from_clause.
	VisitFrom_clause(ctx *From_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#from_list.
	VisitFrom_list(ctx *From_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#non_ansi_join.
	VisitNon_ansi_join(ctx *Non_ansi_joinContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#table_ref.
	VisitTable_ref(ctx *Table_refContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#alias_clause.
	VisitAlias_clause(ctx *Alias_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_alias_clause.
	VisitOpt_alias_clause(ctx *Opt_alias_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#table_alias_clause.
	VisitTable_alias_clause(ctx *Table_alias_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_alias_clause.
	VisitFunc_alias_clause(ctx *Func_alias_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#join_type.
	VisitJoin_type(ctx *Join_typeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#join_qual.
	VisitJoin_qual(ctx *Join_qualContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#relation_expr.
	VisitRelation_expr(ctx *Relation_exprContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#relation_expr_list.
	VisitRelation_expr_list(ctx *Relation_expr_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#relation_expr_opt_alias.
	VisitRelation_expr_opt_alias(ctx *Relation_expr_opt_aliasContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#tablesample_clause.
	VisitTablesample_clause(ctx *Tablesample_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_repeatable_clause.
	VisitOpt_repeatable_clause(ctx *Opt_repeatable_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_table.
	VisitFunc_table(ctx *Func_tableContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#rowsfrom_item.
	VisitRowsfrom_item(ctx *Rowsfrom_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#rowsfrom_list.
	VisitRowsfrom_list(ctx *Rowsfrom_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_col_def_list.
	VisitOpt_col_def_list(ctx *Opt_col_def_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_ordinality.
	VisitOpt_ordinality(ctx *Opt_ordinalityContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#where_clause.
	VisitWhere_clause(ctx *Where_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#where_or_current_clause.
	VisitWhere_or_current_clause(ctx *Where_or_current_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opttablefuncelementlist.
	VisitOpttablefuncelementlist(ctx *OpttablefuncelementlistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#tablefuncelementlist.
	VisitTablefuncelementlist(ctx *TablefuncelementlistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#tablefuncelement.
	VisitTablefuncelement(ctx *TablefuncelementContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#xmltable.
	VisitXmltable(ctx *XmltableContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#xmltable_column_list.
	VisitXmltable_column_list(ctx *Xmltable_column_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#xmltable_column_el.
	VisitXmltable_column_el(ctx *Xmltable_column_elContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#xmltable_column_option_list.
	VisitXmltable_column_option_list(ctx *Xmltable_column_option_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#xmltable_column_option_el.
	VisitXmltable_column_option_el(ctx *Xmltable_column_option_elContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#xml_namespace_list.
	VisitXml_namespace_list(ctx *Xml_namespace_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#xml_namespace_el.
	VisitXml_namespace_el(ctx *Xml_namespace_elContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#typename.
	VisitTypename(ctx *TypenameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_array_bounds.
	VisitOpt_array_bounds(ctx *Opt_array_boundsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#simpletypename.
	VisitSimpletypename(ctx *SimpletypenameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#consttypename.
	VisitConsttypename(ctx *ConsttypenameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#generictype.
	VisitGenerictype(ctx *GenerictypeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_type_modifiers.
	VisitOpt_type_modifiers(ctx *Opt_type_modifiersContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#numeric.
	VisitNumeric(ctx *NumericContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_float.
	VisitOpt_float(ctx *Opt_floatContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#bit.
	VisitBit(ctx *BitContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#constbit.
	VisitConstbit(ctx *ConstbitContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#bitwithlength.
	VisitBitwithlength(ctx *BitwithlengthContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#bitwithoutlength.
	VisitBitwithoutlength(ctx *BitwithoutlengthContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#character.
	VisitCharacter(ctx *CharacterContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#constcharacter.
	VisitConstcharacter(ctx *ConstcharacterContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#character_c.
	VisitCharacter_c(ctx *Character_cContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_varying.
	VisitOpt_varying(ctx *Opt_varyingContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#constdatetime.
	VisitConstdatetime(ctx *ConstdatetimeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#constinterval.
	VisitConstinterval(ctx *ConstintervalContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_timezone.
	VisitOpt_timezone(ctx *Opt_timezoneContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_interval.
	VisitOpt_interval(ctx *Opt_intervalContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#interval_second.
	VisitInterval_second(ctx *Interval_secondContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_escape.
	VisitOpt_escape(ctx *Opt_escapeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr.
	VisitA_expr(ctx *A_exprContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_qual.
	VisitA_expr_qual(ctx *A_expr_qualContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_lessless.
	VisitA_expr_lessless(ctx *A_expr_lesslessContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_or.
	VisitA_expr_or(ctx *A_expr_orContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_and.
	VisitA_expr_and(ctx *A_expr_andContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_between.
	VisitA_expr_between(ctx *A_expr_betweenContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_in.
	VisitA_expr_in(ctx *A_expr_inContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_unary_not.
	VisitA_expr_unary_not(ctx *A_expr_unary_notContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_isnull.
	VisitA_expr_isnull(ctx *A_expr_isnullContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_is_not.
	VisitA_expr_is_not(ctx *A_expr_is_notContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_compare.
	VisitA_expr_compare(ctx *A_expr_compareContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_like.
	VisitA_expr_like(ctx *A_expr_likeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_qual_op.
	VisitA_expr_qual_op(ctx *A_expr_qual_opContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_unary_qualop.
	VisitA_expr_unary_qualop(ctx *A_expr_unary_qualopContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_add.
	VisitA_expr_add(ctx *A_expr_addContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_mul.
	VisitA_expr_mul(ctx *A_expr_mulContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_caret.
	VisitA_expr_caret(ctx *A_expr_caretContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_unary_sign.
	VisitA_expr_unary_sign(ctx *A_expr_unary_signContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_at_time_zone.
	VisitA_expr_at_time_zone(ctx *A_expr_at_time_zoneContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_collate.
	VisitA_expr_collate(ctx *A_expr_collateContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#a_expr_typecast.
	VisitA_expr_typecast(ctx *A_expr_typecastContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#b_expr.
	VisitB_expr(ctx *B_exprContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#c_expr_exists.
	VisitC_expr_exists(ctx *C_expr_existsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#c_expr_expr.
	VisitC_expr_expr(ctx *C_expr_exprContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#c_expr_case.
	VisitC_expr_case(ctx *C_expr_caseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#plsqlvariablename.
	VisitPlsqlvariablename(ctx *PlsqlvariablenameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_application.
	VisitFunc_application(ctx *Func_applicationContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_expr.
	VisitFunc_expr(ctx *Func_exprContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_expr_windowless.
	VisitFunc_expr_windowless(ctx *Func_expr_windowlessContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_expr_common_subexpr.
	VisitFunc_expr_common_subexpr(ctx *Func_expr_common_subexprContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#xml_root_version.
	VisitXml_root_version(ctx *Xml_root_versionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_xml_root_standalone.
	VisitOpt_xml_root_standalone(ctx *Opt_xml_root_standaloneContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#xml_attributes.
	VisitXml_attributes(ctx *Xml_attributesContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#xml_attribute_list.
	VisitXml_attribute_list(ctx *Xml_attribute_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#xml_attribute_el.
	VisitXml_attribute_el(ctx *Xml_attribute_elContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#document_or_content.
	VisitDocument_or_content(ctx *Document_or_contentContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#xml_whitespace_option.
	VisitXml_whitespace_option(ctx *Xml_whitespace_optionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#xmlexists_argument.
	VisitXmlexists_argument(ctx *Xmlexists_argumentContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#xml_passing_mech.
	VisitXml_passing_mech(ctx *Xml_passing_mechContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#within_group_clause.
	VisitWithin_group_clause(ctx *Within_group_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#filter_clause.
	VisitFilter_clause(ctx *Filter_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#window_clause.
	VisitWindow_clause(ctx *Window_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#window_definition_list.
	VisitWindow_definition_list(ctx *Window_definition_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#window_definition.
	VisitWindow_definition(ctx *Window_definitionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#over_clause.
	VisitOver_clause(ctx *Over_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#window_specification.
	VisitWindow_specification(ctx *Window_specificationContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_existing_window_name.
	VisitOpt_existing_window_name(ctx *Opt_existing_window_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_partition_clause.
	VisitOpt_partition_clause(ctx *Opt_partition_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_frame_clause.
	VisitOpt_frame_clause(ctx *Opt_frame_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#frame_extent.
	VisitFrame_extent(ctx *Frame_extentContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#frame_bound.
	VisitFrame_bound(ctx *Frame_boundContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_window_exclusion_clause.
	VisitOpt_window_exclusion_clause(ctx *Opt_window_exclusion_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#row.
	VisitRow(ctx *RowContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#explicit_row.
	VisitExplicit_row(ctx *Explicit_rowContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#implicit_row.
	VisitImplicit_row(ctx *Implicit_rowContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#sub_type.
	VisitSub_type(ctx *Sub_typeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#all_op.
	VisitAll_op(ctx *All_opContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#mathop.
	VisitMathop(ctx *MathopContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#qual_op.
	VisitQual_op(ctx *Qual_opContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#qual_all_op.
	VisitQual_all_op(ctx *Qual_all_opContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#subquery_Op.
	VisitSubquery_Op(ctx *Subquery_OpContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#expr_list.
	VisitExpr_list(ctx *Expr_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_arg_list.
	VisitFunc_arg_list(ctx *Func_arg_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_arg_expr.
	VisitFunc_arg_expr(ctx *Func_arg_exprContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#type_list.
	VisitType_list(ctx *Type_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#array_expr.
	VisitArray_expr(ctx *Array_exprContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#array_expr_list.
	VisitArray_expr_list(ctx *Array_expr_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#extract_list.
	VisitExtract_list(ctx *Extract_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#extract_arg.
	VisitExtract_arg(ctx *Extract_argContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#unicode_normal_form.
	VisitUnicode_normal_form(ctx *Unicode_normal_formContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#overlay_list.
	VisitOverlay_list(ctx *Overlay_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#position_list.
	VisitPosition_list(ctx *Position_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#substr_list.
	VisitSubstr_list(ctx *Substr_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#trim_list.
	VisitTrim_list(ctx *Trim_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#in_expr_select.
	VisitIn_expr_select(ctx *In_expr_selectContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#in_expr_list.
	VisitIn_expr_list(ctx *In_expr_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#case_expr.
	VisitCase_expr(ctx *Case_exprContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#when_clause_list.
	VisitWhen_clause_list(ctx *When_clause_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#when_clause.
	VisitWhen_clause(ctx *When_clauseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#case_default.
	VisitCase_default(ctx *Case_defaultContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#case_arg.
	VisitCase_arg(ctx *Case_argContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#columnref.
	VisitColumnref(ctx *ColumnrefContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#indirection_el.
	VisitIndirection_el(ctx *Indirection_elContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_slice_bound.
	VisitOpt_slice_bound(ctx *Opt_slice_boundContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#indirection.
	VisitIndirection(ctx *IndirectionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_indirection.
	VisitOpt_indirection(ctx *Opt_indirectionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_target_list.
	VisitOpt_target_list(ctx *Opt_target_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#target_list.
	VisitTarget_list(ctx *Target_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#target_label.
	VisitTarget_label(ctx *Target_labelContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#target_star.
	VisitTarget_star(ctx *Target_starContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#qualified_name_list.
	VisitQualified_name_list(ctx *Qualified_name_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#qualified_name.
	VisitQualified_name(ctx *Qualified_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#name_list.
	VisitName_list(ctx *Name_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#name.
	VisitName(ctx *NameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#attr_name.
	VisitAttr_name(ctx *Attr_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#file_name.
	VisitFile_name(ctx *File_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#func_name.
	VisitFunc_name(ctx *Func_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#aexprconst.
	VisitAexprconst(ctx *AexprconstContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#xconst.
	VisitXconst(ctx *XconstContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#bconst.
	VisitBconst(ctx *BconstContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#fconst.
	VisitFconst(ctx *FconstContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#iconst.
	VisitIconst(ctx *IconstContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#sconst.
	VisitSconst(ctx *SconstContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#anysconst.
	VisitAnysconst(ctx *AnysconstContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_uescape.
	VisitOpt_uescape(ctx *Opt_uescapeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#signediconst.
	VisitSignediconst(ctx *SignediconstContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#roleid.
	VisitRoleid(ctx *RoleidContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#rolespec.
	VisitRolespec(ctx *RolespecContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#role_list.
	VisitRole_list(ctx *Role_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#colid.
	VisitColid(ctx *ColidContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#table_alias.
	VisitTable_alias(ctx *Table_aliasContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#type_function_name.
	VisitType_function_name(ctx *Type_function_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#nonreservedword.
	VisitNonreservedword(ctx *NonreservedwordContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#collabel.
	VisitCollabel(ctx *CollabelContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#identifier.
	VisitIdentifier(ctx *IdentifierContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#plsqlidentifier.
	VisitPlsqlidentifier(ctx *PlsqlidentifierContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#unreserved_keyword.
	VisitUnreserved_keyword(ctx *Unreserved_keywordContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#col_name_keyword.
	VisitCol_name_keyword(ctx *Col_name_keywordContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#type_func_name_keyword.
	VisitType_func_name_keyword(ctx *Type_func_name_keywordContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#reserved_keyword.
	VisitReserved_keyword(ctx *Reserved_keywordContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#builtin_function_name.
	VisitBuiltin_function_name(ctx *Builtin_function_nameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#pl_function.
	VisitPl_function(ctx *Pl_functionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#comp_options.
	VisitComp_options(ctx *Comp_optionsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#comp_option.
	VisitComp_option(ctx *Comp_optionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#sharp.
	VisitSharp(ctx *SharpContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#option_value.
	VisitOption_value(ctx *Option_valueContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_semi.
	VisitOpt_semi(ctx *Opt_semiContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#pl_block.
	VisitPl_block(ctx *Pl_blockContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_sect.
	VisitDecl_sect(ctx *Decl_sectContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_start.
	VisitDecl_start(ctx *Decl_startContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_stmts.
	VisitDecl_stmts(ctx *Decl_stmtsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#label_decl.
	VisitLabel_decl(ctx *Label_declContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_stmt.
	VisitDecl_stmt(ctx *Decl_stmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_statement.
	VisitDecl_statement(ctx *Decl_statementContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_scrollable.
	VisitOpt_scrollable(ctx *Opt_scrollableContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_cursor_query.
	VisitDecl_cursor_query(ctx *Decl_cursor_queryContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_cursor_args.
	VisitDecl_cursor_args(ctx *Decl_cursor_argsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_cursor_arglist.
	VisitDecl_cursor_arglist(ctx *Decl_cursor_arglistContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_cursor_arg.
	VisitDecl_cursor_arg(ctx *Decl_cursor_argContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_is_for.
	VisitDecl_is_for(ctx *Decl_is_forContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_aliasitem.
	VisitDecl_aliasitem(ctx *Decl_aliasitemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_varname.
	VisitDecl_varname(ctx *Decl_varnameContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_const.
	VisitDecl_const(ctx *Decl_constContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_datatype.
	VisitDecl_datatype(ctx *Decl_datatypeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_collate.
	VisitDecl_collate(ctx *Decl_collateContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_notnull.
	VisitDecl_notnull(ctx *Decl_notnullContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_defval.
	VisitDecl_defval(ctx *Decl_defvalContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#decl_defkey.
	VisitDecl_defkey(ctx *Decl_defkeyContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#assign_operator.
	VisitAssign_operator(ctx *Assign_operatorContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#proc_sect.
	VisitProc_sect(ctx *Proc_sectContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#proc_stmt.
	VisitProc_stmt(ctx *Proc_stmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_perform.
	VisitStmt_perform(ctx *Stmt_performContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_call.
	VisitStmt_call(ctx *Stmt_callContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_expr_list.
	VisitOpt_expr_list(ctx *Opt_expr_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_assign.
	VisitStmt_assign(ctx *Stmt_assignContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_getdiag.
	VisitStmt_getdiag(ctx *Stmt_getdiagContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#getdiag_area_opt.
	VisitGetdiag_area_opt(ctx *Getdiag_area_optContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#getdiag_list.
	VisitGetdiag_list(ctx *Getdiag_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#getdiag_list_item.
	VisitGetdiag_list_item(ctx *Getdiag_list_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#getdiag_item.
	VisitGetdiag_item(ctx *Getdiag_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#getdiag_target.
	VisitGetdiag_target(ctx *Getdiag_targetContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#assign_var.
	VisitAssign_var(ctx *Assign_varContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_if.
	VisitStmt_if(ctx *Stmt_ifContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_elsifs.
	VisitStmt_elsifs(ctx *Stmt_elsifsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_else.
	VisitStmt_else(ctx *Stmt_elseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_case.
	VisitStmt_case(ctx *Stmt_caseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_expr_until_when.
	VisitOpt_expr_until_when(ctx *Opt_expr_until_whenContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#case_when_list.
	VisitCase_when_list(ctx *Case_when_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#case_when.
	VisitCase_when(ctx *Case_whenContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_case_else.
	VisitOpt_case_else(ctx *Opt_case_elseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_loop.
	VisitStmt_loop(ctx *Stmt_loopContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_while.
	VisitStmt_while(ctx *Stmt_whileContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_for.
	VisitStmt_for(ctx *Stmt_forContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#for_control.
	VisitFor_control(ctx *For_controlContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_for_using_expression.
	VisitOpt_for_using_expression(ctx *Opt_for_using_expressionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_cursor_parameters.
	VisitOpt_cursor_parameters(ctx *Opt_cursor_parametersContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_reverse.
	VisitOpt_reverse(ctx *Opt_reverseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_by_expression.
	VisitOpt_by_expression(ctx *Opt_by_expressionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#for_variable.
	VisitFor_variable(ctx *For_variableContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_foreach_a.
	VisitStmt_foreach_a(ctx *Stmt_foreach_aContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#foreach_slice.
	VisitForeach_slice(ctx *Foreach_sliceContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_exit.
	VisitStmt_exit(ctx *Stmt_exitContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#exit_type.
	VisitExit_type(ctx *Exit_typeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_return.
	VisitStmt_return(ctx *Stmt_returnContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_return_result.
	VisitOpt_return_result(ctx *Opt_return_resultContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_raise.
	VisitStmt_raise(ctx *Stmt_raiseContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_stmt_raise_level.
	VisitOpt_stmt_raise_level(ctx *Opt_stmt_raise_levelContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_raise_list.
	VisitOpt_raise_list(ctx *Opt_raise_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_raise_using.
	VisitOpt_raise_using(ctx *Opt_raise_usingContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_raise_using_elem.
	VisitOpt_raise_using_elem(ctx *Opt_raise_using_elemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_raise_using_elem_list.
	VisitOpt_raise_using_elem_list(ctx *Opt_raise_using_elem_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_assert.
	VisitStmt_assert(ctx *Stmt_assertContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_stmt_assert_message.
	VisitOpt_stmt_assert_message(ctx *Opt_stmt_assert_messageContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#loop_body.
	VisitLoop_body(ctx *Loop_bodyContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_execsql.
	VisitStmt_execsql(ctx *Stmt_execsqlContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_dynexecute.
	VisitStmt_dynexecute(ctx *Stmt_dynexecuteContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_execute_using.
	VisitOpt_execute_using(ctx *Opt_execute_usingContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_execute_using_list.
	VisitOpt_execute_using_list(ctx *Opt_execute_using_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_execute_into.
	VisitOpt_execute_into(ctx *Opt_execute_intoContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_open.
	VisitStmt_open(ctx *Stmt_openContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_open_bound_list_item.
	VisitOpt_open_bound_list_item(ctx *Opt_open_bound_list_itemContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_open_bound_list.
	VisitOpt_open_bound_list(ctx *Opt_open_bound_listContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_open_using.
	VisitOpt_open_using(ctx *Opt_open_usingContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_scroll_option.
	VisitOpt_scroll_option(ctx *Opt_scroll_optionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_scroll_option_no.
	VisitOpt_scroll_option_no(ctx *Opt_scroll_option_noContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_fetch.
	VisitStmt_fetch(ctx *Stmt_fetchContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#into_target.
	VisitInto_target(ctx *Into_targetContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_cursor_from.
	VisitOpt_cursor_from(ctx *Opt_cursor_fromContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_fetch_direction.
	VisitOpt_fetch_direction(ctx *Opt_fetch_directionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_move.
	VisitStmt_move(ctx *Stmt_moveContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_close.
	VisitStmt_close(ctx *Stmt_closeContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_null.
	VisitStmt_null(ctx *Stmt_nullContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_commit.
	VisitStmt_commit(ctx *Stmt_commitContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_rollback.
	VisitStmt_rollback(ctx *Stmt_rollbackContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#plsql_opt_transaction_chain.
	VisitPlsql_opt_transaction_chain(ctx *Plsql_opt_transaction_chainContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#stmt_set.
	VisitStmt_set(ctx *Stmt_setContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#cursor_variable.
	VisitCursor_variable(ctx *Cursor_variableContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#exception_sect.
	VisitException_sect(ctx *Exception_sectContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#proc_exceptions.
	VisitProc_exceptions(ctx *Proc_exceptionsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#proc_exception.
	VisitProc_exception(ctx *Proc_exceptionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#proc_conditions.
	VisitProc_conditions(ctx *Proc_conditionsContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#proc_condition.
	VisitProc_condition(ctx *Proc_conditionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_block_label.
	VisitOpt_block_label(ctx *Opt_block_labelContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_loop_label.
	VisitOpt_loop_label(ctx *Opt_loop_labelContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_label.
	VisitOpt_label(ctx *Opt_labelContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_exitcond.
	VisitOpt_exitcond(ctx *Opt_exitcondContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#any_identifier.
	VisitAny_identifier(ctx *Any_identifierContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#plsql_unreserved_keyword.
	VisitPlsql_unreserved_keyword(ctx *Plsql_unreserved_keywordContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#sql_expression.
	VisitSql_expression(ctx *Sql_expressionContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#expr_until_then.
	VisitExpr_until_then(ctx *Expr_until_thenContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#expr_until_semi.
	VisitExpr_until_semi(ctx *Expr_until_semiContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#expr_until_rightbracket.
	VisitExpr_until_rightbracket(ctx *Expr_until_rightbracketContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#expr_until_loop.
	VisitExpr_until_loop(ctx *Expr_until_loopContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#make_execsql_stmt.
	VisitMake_execsql_stmt(ctx *Make_execsql_stmtContext) interface{}

	// Visit a parse tree produced by PostgreSQLParser#opt_returning_clause_into.
	VisitOpt_returning_clause_into(ctx *Opt_returning_clause_intoContext) interface{}
}
