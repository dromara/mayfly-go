// Code generated from PostgreSQLParser.g4 by ANTLR 4.13.1. DO NOT EDIT.

package parser // PostgreSQLParser
import "github.com/antlr4-go/antlr/v4"

// PostgreSQLParserListener is a complete listener for a parse tree produced by PostgreSQLParser.
type PostgreSQLParserListener interface {
	antlr.ParseTreeListener

	// EnterRoot is called when entering the root production.
	EnterRoot(c *RootContext)

	// EnterPlsqlroot is called when entering the plsqlroot production.
	EnterPlsqlroot(c *PlsqlrootContext)

	// EnterStmtblock is called when entering the stmtblock production.
	EnterStmtblock(c *StmtblockContext)

	// EnterStmtmulti is called when entering the stmtmulti production.
	EnterStmtmulti(c *StmtmultiContext)

	// EnterStmt is called when entering the stmt production.
	EnterStmt(c *StmtContext)

	// EnterPlsqlconsolecommand is called when entering the plsqlconsolecommand production.
	EnterPlsqlconsolecommand(c *PlsqlconsolecommandContext)

	// EnterCallstmt is called when entering the callstmt production.
	EnterCallstmt(c *CallstmtContext)

	// EnterCreaterolestmt is called when entering the createrolestmt production.
	EnterCreaterolestmt(c *CreaterolestmtContext)

	// EnterOpt_with is called when entering the opt_with production.
	EnterOpt_with(c *Opt_withContext)

	// EnterOptrolelist is called when entering the optrolelist production.
	EnterOptrolelist(c *OptrolelistContext)

	// EnterAlteroptrolelist is called when entering the alteroptrolelist production.
	EnterAlteroptrolelist(c *AlteroptrolelistContext)

	// EnterAlteroptroleelem is called when entering the alteroptroleelem production.
	EnterAlteroptroleelem(c *AlteroptroleelemContext)

	// EnterCreateoptroleelem is called when entering the createoptroleelem production.
	EnterCreateoptroleelem(c *CreateoptroleelemContext)

	// EnterCreateuserstmt is called when entering the createuserstmt production.
	EnterCreateuserstmt(c *CreateuserstmtContext)

	// EnterAlterrolestmt is called when entering the alterrolestmt production.
	EnterAlterrolestmt(c *AlterrolestmtContext)

	// EnterOpt_in_database is called when entering the opt_in_database production.
	EnterOpt_in_database(c *Opt_in_databaseContext)

	// EnterAlterrolesetstmt is called when entering the alterrolesetstmt production.
	EnterAlterrolesetstmt(c *AlterrolesetstmtContext)

	// EnterDroprolestmt is called when entering the droprolestmt production.
	EnterDroprolestmt(c *DroprolestmtContext)

	// EnterCreategroupstmt is called when entering the creategroupstmt production.
	EnterCreategroupstmt(c *CreategroupstmtContext)

	// EnterAltergroupstmt is called when entering the altergroupstmt production.
	EnterAltergroupstmt(c *AltergroupstmtContext)

	// EnterAdd_drop is called when entering the add_drop production.
	EnterAdd_drop(c *Add_dropContext)

	// EnterCreateschemastmt is called when entering the createschemastmt production.
	EnterCreateschemastmt(c *CreateschemastmtContext)

	// EnterOptschemaname is called when entering the optschemaname production.
	EnterOptschemaname(c *OptschemanameContext)

	// EnterOptschemaeltlist is called when entering the optschemaeltlist production.
	EnterOptschemaeltlist(c *OptschemaeltlistContext)

	// EnterSchema_stmt is called when entering the schema_stmt production.
	EnterSchema_stmt(c *Schema_stmtContext)

	// EnterVariablesetstmt is called when entering the variablesetstmt production.
	EnterVariablesetstmt(c *VariablesetstmtContext)

	// EnterSet_rest is called when entering the set_rest production.
	EnterSet_rest(c *Set_restContext)

	// EnterGeneric_set is called when entering the generic_set production.
	EnterGeneric_set(c *Generic_setContext)

	// EnterSet_rest_more is called when entering the set_rest_more production.
	EnterSet_rest_more(c *Set_rest_moreContext)

	// EnterVar_name is called when entering the var_name production.
	EnterVar_name(c *Var_nameContext)

	// EnterVar_list is called when entering the var_list production.
	EnterVar_list(c *Var_listContext)

	// EnterVar_value is called when entering the var_value production.
	EnterVar_value(c *Var_valueContext)

	// EnterIso_level is called when entering the iso_level production.
	EnterIso_level(c *Iso_levelContext)

	// EnterOpt_boolean_or_string is called when entering the opt_boolean_or_string production.
	EnterOpt_boolean_or_string(c *Opt_boolean_or_stringContext)

	// EnterZone_value is called when entering the zone_value production.
	EnterZone_value(c *Zone_valueContext)

	// EnterOpt_encoding is called when entering the opt_encoding production.
	EnterOpt_encoding(c *Opt_encodingContext)

	// EnterNonreservedword_or_sconst is called when entering the nonreservedword_or_sconst production.
	EnterNonreservedword_or_sconst(c *Nonreservedword_or_sconstContext)

	// EnterVariableresetstmt is called when entering the variableresetstmt production.
	EnterVariableresetstmt(c *VariableresetstmtContext)

	// EnterReset_rest is called when entering the reset_rest production.
	EnterReset_rest(c *Reset_restContext)

	// EnterGeneric_reset is called when entering the generic_reset production.
	EnterGeneric_reset(c *Generic_resetContext)

	// EnterSetresetclause is called when entering the setresetclause production.
	EnterSetresetclause(c *SetresetclauseContext)

	// EnterFunctionsetresetclause is called when entering the functionsetresetclause production.
	EnterFunctionsetresetclause(c *FunctionsetresetclauseContext)

	// EnterVariableshowstmt is called when entering the variableshowstmt production.
	EnterVariableshowstmt(c *VariableshowstmtContext)

	// EnterConstraintssetstmt is called when entering the constraintssetstmt production.
	EnterConstraintssetstmt(c *ConstraintssetstmtContext)

	// EnterConstraints_set_list is called when entering the constraints_set_list production.
	EnterConstraints_set_list(c *Constraints_set_listContext)

	// EnterConstraints_set_mode is called when entering the constraints_set_mode production.
	EnterConstraints_set_mode(c *Constraints_set_modeContext)

	// EnterCheckpointstmt is called when entering the checkpointstmt production.
	EnterCheckpointstmt(c *CheckpointstmtContext)

	// EnterDiscardstmt is called when entering the discardstmt production.
	EnterDiscardstmt(c *DiscardstmtContext)

	// EnterAltertablestmt is called when entering the altertablestmt production.
	EnterAltertablestmt(c *AltertablestmtContext)

	// EnterAlter_table_cmds is called when entering the alter_table_cmds production.
	EnterAlter_table_cmds(c *Alter_table_cmdsContext)

	// EnterPartition_cmd is called when entering the partition_cmd production.
	EnterPartition_cmd(c *Partition_cmdContext)

	// EnterIndex_partition_cmd is called when entering the index_partition_cmd production.
	EnterIndex_partition_cmd(c *Index_partition_cmdContext)

	// EnterAlter_table_cmd is called when entering the alter_table_cmd production.
	EnterAlter_table_cmd(c *Alter_table_cmdContext)

	// EnterAlter_column_default is called when entering the alter_column_default production.
	EnterAlter_column_default(c *Alter_column_defaultContext)

	// EnterOpt_drop_behavior is called when entering the opt_drop_behavior production.
	EnterOpt_drop_behavior(c *Opt_drop_behaviorContext)

	// EnterOpt_collate_clause is called when entering the opt_collate_clause production.
	EnterOpt_collate_clause(c *Opt_collate_clauseContext)

	// EnterAlter_using is called when entering the alter_using production.
	EnterAlter_using(c *Alter_usingContext)

	// EnterReplica_identity is called when entering the replica_identity production.
	EnterReplica_identity(c *Replica_identityContext)

	// EnterReloptions is called when entering the reloptions production.
	EnterReloptions(c *ReloptionsContext)

	// EnterOpt_reloptions is called when entering the opt_reloptions production.
	EnterOpt_reloptions(c *Opt_reloptionsContext)

	// EnterReloption_list is called when entering the reloption_list production.
	EnterReloption_list(c *Reloption_listContext)

	// EnterReloption_elem is called when entering the reloption_elem production.
	EnterReloption_elem(c *Reloption_elemContext)

	// EnterAlter_identity_column_option_list is called when entering the alter_identity_column_option_list production.
	EnterAlter_identity_column_option_list(c *Alter_identity_column_option_listContext)

	// EnterAlter_identity_column_option is called when entering the alter_identity_column_option production.
	EnterAlter_identity_column_option(c *Alter_identity_column_optionContext)

	// EnterPartitionboundspec is called when entering the partitionboundspec production.
	EnterPartitionboundspec(c *PartitionboundspecContext)

	// EnterHash_partbound_elem is called when entering the hash_partbound_elem production.
	EnterHash_partbound_elem(c *Hash_partbound_elemContext)

	// EnterHash_partbound is called when entering the hash_partbound production.
	EnterHash_partbound(c *Hash_partboundContext)

	// EnterAltercompositetypestmt is called when entering the altercompositetypestmt production.
	EnterAltercompositetypestmt(c *AltercompositetypestmtContext)

	// EnterAlter_type_cmds is called when entering the alter_type_cmds production.
	EnterAlter_type_cmds(c *Alter_type_cmdsContext)

	// EnterAlter_type_cmd is called when entering the alter_type_cmd production.
	EnterAlter_type_cmd(c *Alter_type_cmdContext)

	// EnterCloseportalstmt is called when entering the closeportalstmt production.
	EnterCloseportalstmt(c *CloseportalstmtContext)

	// EnterCopystmt is called when entering the copystmt production.
	EnterCopystmt(c *CopystmtContext)

	// EnterCopy_from is called when entering the copy_from production.
	EnterCopy_from(c *Copy_fromContext)

	// EnterOpt_program is called when entering the opt_program production.
	EnterOpt_program(c *Opt_programContext)

	// EnterCopy_file_name is called when entering the copy_file_name production.
	EnterCopy_file_name(c *Copy_file_nameContext)

	// EnterCopy_options is called when entering the copy_options production.
	EnterCopy_options(c *Copy_optionsContext)

	// EnterCopy_opt_list is called when entering the copy_opt_list production.
	EnterCopy_opt_list(c *Copy_opt_listContext)

	// EnterCopy_opt_item is called when entering the copy_opt_item production.
	EnterCopy_opt_item(c *Copy_opt_itemContext)

	// EnterOpt_binary is called when entering the opt_binary production.
	EnterOpt_binary(c *Opt_binaryContext)

	// EnterCopy_delimiter is called when entering the copy_delimiter production.
	EnterCopy_delimiter(c *Copy_delimiterContext)

	// EnterOpt_using is called when entering the opt_using production.
	EnterOpt_using(c *Opt_usingContext)

	// EnterCopy_generic_opt_list is called when entering the copy_generic_opt_list production.
	EnterCopy_generic_opt_list(c *Copy_generic_opt_listContext)

	// EnterCopy_generic_opt_elem is called when entering the copy_generic_opt_elem production.
	EnterCopy_generic_opt_elem(c *Copy_generic_opt_elemContext)

	// EnterCopy_generic_opt_arg is called when entering the copy_generic_opt_arg production.
	EnterCopy_generic_opt_arg(c *Copy_generic_opt_argContext)

	// EnterCopy_generic_opt_arg_list is called when entering the copy_generic_opt_arg_list production.
	EnterCopy_generic_opt_arg_list(c *Copy_generic_opt_arg_listContext)

	// EnterCopy_generic_opt_arg_list_item is called when entering the copy_generic_opt_arg_list_item production.
	EnterCopy_generic_opt_arg_list_item(c *Copy_generic_opt_arg_list_itemContext)

	// EnterCreatestmt is called when entering the createstmt production.
	EnterCreatestmt(c *CreatestmtContext)

	// EnterOpttemp is called when entering the opttemp production.
	EnterOpttemp(c *OpttempContext)

	// EnterOpttableelementlist is called when entering the opttableelementlist production.
	EnterOpttableelementlist(c *OpttableelementlistContext)

	// EnterOpttypedtableelementlist is called when entering the opttypedtableelementlist production.
	EnterOpttypedtableelementlist(c *OpttypedtableelementlistContext)

	// EnterTableelementlist is called when entering the tableelementlist production.
	EnterTableelementlist(c *TableelementlistContext)

	// EnterTypedtableelementlist is called when entering the typedtableelementlist production.
	EnterTypedtableelementlist(c *TypedtableelementlistContext)

	// EnterTableelement is called when entering the tableelement production.
	EnterTableelement(c *TableelementContext)

	// EnterTypedtableelement is called when entering the typedtableelement production.
	EnterTypedtableelement(c *TypedtableelementContext)

	// EnterColumnDef is called when entering the columnDef production.
	EnterColumnDef(c *ColumnDefContext)

	// EnterColumnOptions is called when entering the columnOptions production.
	EnterColumnOptions(c *ColumnOptionsContext)

	// EnterColquallist is called when entering the colquallist production.
	EnterColquallist(c *ColquallistContext)

	// EnterColconstraint is called when entering the colconstraint production.
	EnterColconstraint(c *ColconstraintContext)

	// EnterColconstraintelem is called when entering the colconstraintelem production.
	EnterColconstraintelem(c *ColconstraintelemContext)

	// EnterGenerated_when is called when entering the generated_when production.
	EnterGenerated_when(c *Generated_whenContext)

	// EnterConstraintattr is called when entering the constraintattr production.
	EnterConstraintattr(c *ConstraintattrContext)

	// EnterTablelikeclause is called when entering the tablelikeclause production.
	EnterTablelikeclause(c *TablelikeclauseContext)

	// EnterTablelikeoptionlist is called when entering the tablelikeoptionlist production.
	EnterTablelikeoptionlist(c *TablelikeoptionlistContext)

	// EnterTablelikeoption is called when entering the tablelikeoption production.
	EnterTablelikeoption(c *TablelikeoptionContext)

	// EnterTableconstraint is called when entering the tableconstraint production.
	EnterTableconstraint(c *TableconstraintContext)

	// EnterConstraintelem is called when entering the constraintelem production.
	EnterConstraintelem(c *ConstraintelemContext)

	// EnterOpt_no_inherit is called when entering the opt_no_inherit production.
	EnterOpt_no_inherit(c *Opt_no_inheritContext)

	// EnterOpt_column_list is called when entering the opt_column_list production.
	EnterOpt_column_list(c *Opt_column_listContext)

	// EnterColumnlist is called when entering the columnlist production.
	EnterColumnlist(c *ColumnlistContext)

	// EnterColumnElem is called when entering the columnElem production.
	EnterColumnElem(c *ColumnElemContext)

	// EnterOpt_c_include is called when entering the opt_c_include production.
	EnterOpt_c_include(c *Opt_c_includeContext)

	// EnterKey_match is called when entering the key_match production.
	EnterKey_match(c *Key_matchContext)

	// EnterExclusionconstraintlist is called when entering the exclusionconstraintlist production.
	EnterExclusionconstraintlist(c *ExclusionconstraintlistContext)

	// EnterExclusionconstraintelem is called when entering the exclusionconstraintelem production.
	EnterExclusionconstraintelem(c *ExclusionconstraintelemContext)

	// EnterExclusionwhereclause is called when entering the exclusionwhereclause production.
	EnterExclusionwhereclause(c *ExclusionwhereclauseContext)

	// EnterKey_actions is called when entering the key_actions production.
	EnterKey_actions(c *Key_actionsContext)

	// EnterKey_update is called when entering the key_update production.
	EnterKey_update(c *Key_updateContext)

	// EnterKey_delete is called when entering the key_delete production.
	EnterKey_delete(c *Key_deleteContext)

	// EnterKey_action is called when entering the key_action production.
	EnterKey_action(c *Key_actionContext)

	// EnterOptinherit is called when entering the optinherit production.
	EnterOptinherit(c *OptinheritContext)

	// EnterOptpartitionspec is called when entering the optpartitionspec production.
	EnterOptpartitionspec(c *OptpartitionspecContext)

	// EnterPartitionspec is called when entering the partitionspec production.
	EnterPartitionspec(c *PartitionspecContext)

	// EnterPart_params is called when entering the part_params production.
	EnterPart_params(c *Part_paramsContext)

	// EnterPart_elem is called when entering the part_elem production.
	EnterPart_elem(c *Part_elemContext)

	// EnterTable_access_method_clause is called when entering the table_access_method_clause production.
	EnterTable_access_method_clause(c *Table_access_method_clauseContext)

	// EnterOptwith is called when entering the optwith production.
	EnterOptwith(c *OptwithContext)

	// EnterOncommitoption is called when entering the oncommitoption production.
	EnterOncommitoption(c *OncommitoptionContext)

	// EnterOpttablespace is called when entering the opttablespace production.
	EnterOpttablespace(c *OpttablespaceContext)

	// EnterOptconstablespace is called when entering the optconstablespace production.
	EnterOptconstablespace(c *OptconstablespaceContext)

	// EnterExistingindex is called when entering the existingindex production.
	EnterExistingindex(c *ExistingindexContext)

	// EnterCreatestatsstmt is called when entering the createstatsstmt production.
	EnterCreatestatsstmt(c *CreatestatsstmtContext)

	// EnterAlterstatsstmt is called when entering the alterstatsstmt production.
	EnterAlterstatsstmt(c *AlterstatsstmtContext)

	// EnterCreateasstmt is called when entering the createasstmt production.
	EnterCreateasstmt(c *CreateasstmtContext)

	// EnterCreate_as_target is called when entering the create_as_target production.
	EnterCreate_as_target(c *Create_as_targetContext)

	// EnterOpt_with_data is called when entering the opt_with_data production.
	EnterOpt_with_data(c *Opt_with_dataContext)

	// EnterCreatematviewstmt is called when entering the creatematviewstmt production.
	EnterCreatematviewstmt(c *CreatematviewstmtContext)

	// EnterCreate_mv_target is called when entering the create_mv_target production.
	EnterCreate_mv_target(c *Create_mv_targetContext)

	// EnterOptnolog is called when entering the optnolog production.
	EnterOptnolog(c *OptnologContext)

	// EnterRefreshmatviewstmt is called when entering the refreshmatviewstmt production.
	EnterRefreshmatviewstmt(c *RefreshmatviewstmtContext)

	// EnterCreateseqstmt is called when entering the createseqstmt production.
	EnterCreateseqstmt(c *CreateseqstmtContext)

	// EnterAlterseqstmt is called when entering the alterseqstmt production.
	EnterAlterseqstmt(c *AlterseqstmtContext)

	// EnterOptseqoptlist is called when entering the optseqoptlist production.
	EnterOptseqoptlist(c *OptseqoptlistContext)

	// EnterOptparenthesizedseqoptlist is called when entering the optparenthesizedseqoptlist production.
	EnterOptparenthesizedseqoptlist(c *OptparenthesizedseqoptlistContext)

	// EnterSeqoptlist is called when entering the seqoptlist production.
	EnterSeqoptlist(c *SeqoptlistContext)

	// EnterSeqoptelem is called when entering the seqoptelem production.
	EnterSeqoptelem(c *SeqoptelemContext)

	// EnterOpt_by is called when entering the opt_by production.
	EnterOpt_by(c *Opt_byContext)

	// EnterNumericonly is called when entering the numericonly production.
	EnterNumericonly(c *NumericonlyContext)

	// EnterNumericonly_list is called when entering the numericonly_list production.
	EnterNumericonly_list(c *Numericonly_listContext)

	// EnterCreateplangstmt is called when entering the createplangstmt production.
	EnterCreateplangstmt(c *CreateplangstmtContext)

	// EnterOpt_trusted is called when entering the opt_trusted production.
	EnterOpt_trusted(c *Opt_trustedContext)

	// EnterHandler_name is called when entering the handler_name production.
	EnterHandler_name(c *Handler_nameContext)

	// EnterOpt_inline_handler is called when entering the opt_inline_handler production.
	EnterOpt_inline_handler(c *Opt_inline_handlerContext)

	// EnterValidator_clause is called when entering the validator_clause production.
	EnterValidator_clause(c *Validator_clauseContext)

	// EnterOpt_validator is called when entering the opt_validator production.
	EnterOpt_validator(c *Opt_validatorContext)

	// EnterOpt_procedural is called when entering the opt_procedural production.
	EnterOpt_procedural(c *Opt_proceduralContext)

	// EnterCreatetablespacestmt is called when entering the createtablespacestmt production.
	EnterCreatetablespacestmt(c *CreatetablespacestmtContext)

	// EnterOpttablespaceowner is called when entering the opttablespaceowner production.
	EnterOpttablespaceowner(c *OpttablespaceownerContext)

	// EnterDroptablespacestmt is called when entering the droptablespacestmt production.
	EnterDroptablespacestmt(c *DroptablespacestmtContext)

	// EnterCreateextensionstmt is called when entering the createextensionstmt production.
	EnterCreateextensionstmt(c *CreateextensionstmtContext)

	// EnterCreate_extension_opt_list is called when entering the create_extension_opt_list production.
	EnterCreate_extension_opt_list(c *Create_extension_opt_listContext)

	// EnterCreate_extension_opt_item is called when entering the create_extension_opt_item production.
	EnterCreate_extension_opt_item(c *Create_extension_opt_itemContext)

	// EnterAlterextensionstmt is called when entering the alterextensionstmt production.
	EnterAlterextensionstmt(c *AlterextensionstmtContext)

	// EnterAlter_extension_opt_list is called when entering the alter_extension_opt_list production.
	EnterAlter_extension_opt_list(c *Alter_extension_opt_listContext)

	// EnterAlter_extension_opt_item is called when entering the alter_extension_opt_item production.
	EnterAlter_extension_opt_item(c *Alter_extension_opt_itemContext)

	// EnterAlterextensioncontentsstmt is called when entering the alterextensioncontentsstmt production.
	EnterAlterextensioncontentsstmt(c *AlterextensioncontentsstmtContext)

	// EnterCreatefdwstmt is called when entering the createfdwstmt production.
	EnterCreatefdwstmt(c *CreatefdwstmtContext)

	// EnterFdw_option is called when entering the fdw_option production.
	EnterFdw_option(c *Fdw_optionContext)

	// EnterFdw_options is called when entering the fdw_options production.
	EnterFdw_options(c *Fdw_optionsContext)

	// EnterOpt_fdw_options is called when entering the opt_fdw_options production.
	EnterOpt_fdw_options(c *Opt_fdw_optionsContext)

	// EnterAlterfdwstmt is called when entering the alterfdwstmt production.
	EnterAlterfdwstmt(c *AlterfdwstmtContext)

	// EnterCreate_generic_options is called when entering the create_generic_options production.
	EnterCreate_generic_options(c *Create_generic_optionsContext)

	// EnterGeneric_option_list is called when entering the generic_option_list production.
	EnterGeneric_option_list(c *Generic_option_listContext)

	// EnterAlter_generic_options is called when entering the alter_generic_options production.
	EnterAlter_generic_options(c *Alter_generic_optionsContext)

	// EnterAlter_generic_option_list is called when entering the alter_generic_option_list production.
	EnterAlter_generic_option_list(c *Alter_generic_option_listContext)

	// EnterAlter_generic_option_elem is called when entering the alter_generic_option_elem production.
	EnterAlter_generic_option_elem(c *Alter_generic_option_elemContext)

	// EnterGeneric_option_elem is called when entering the generic_option_elem production.
	EnterGeneric_option_elem(c *Generic_option_elemContext)

	// EnterGeneric_option_name is called when entering the generic_option_name production.
	EnterGeneric_option_name(c *Generic_option_nameContext)

	// EnterGeneric_option_arg is called when entering the generic_option_arg production.
	EnterGeneric_option_arg(c *Generic_option_argContext)

	// EnterCreateforeignserverstmt is called when entering the createforeignserverstmt production.
	EnterCreateforeignserverstmt(c *CreateforeignserverstmtContext)

	// EnterOpt_type is called when entering the opt_type production.
	EnterOpt_type(c *Opt_typeContext)

	// EnterForeign_server_version is called when entering the foreign_server_version production.
	EnterForeign_server_version(c *Foreign_server_versionContext)

	// EnterOpt_foreign_server_version is called when entering the opt_foreign_server_version production.
	EnterOpt_foreign_server_version(c *Opt_foreign_server_versionContext)

	// EnterAlterforeignserverstmt is called when entering the alterforeignserverstmt production.
	EnterAlterforeignserverstmt(c *AlterforeignserverstmtContext)

	// EnterCreateforeigntablestmt is called when entering the createforeigntablestmt production.
	EnterCreateforeigntablestmt(c *CreateforeigntablestmtContext)

	// EnterImportforeignschemastmt is called when entering the importforeignschemastmt production.
	EnterImportforeignschemastmt(c *ImportforeignschemastmtContext)

	// EnterImport_qualification_type is called when entering the import_qualification_type production.
	EnterImport_qualification_type(c *Import_qualification_typeContext)

	// EnterImport_qualification is called when entering the import_qualification production.
	EnterImport_qualification(c *Import_qualificationContext)

	// EnterCreateusermappingstmt is called when entering the createusermappingstmt production.
	EnterCreateusermappingstmt(c *CreateusermappingstmtContext)

	// EnterAuth_ident is called when entering the auth_ident production.
	EnterAuth_ident(c *Auth_identContext)

	// EnterDropusermappingstmt is called when entering the dropusermappingstmt production.
	EnterDropusermappingstmt(c *DropusermappingstmtContext)

	// EnterAlterusermappingstmt is called when entering the alterusermappingstmt production.
	EnterAlterusermappingstmt(c *AlterusermappingstmtContext)

	// EnterCreatepolicystmt is called when entering the createpolicystmt production.
	EnterCreatepolicystmt(c *CreatepolicystmtContext)

	// EnterAlterpolicystmt is called when entering the alterpolicystmt production.
	EnterAlterpolicystmt(c *AlterpolicystmtContext)

	// EnterRowsecurityoptionalexpr is called when entering the rowsecurityoptionalexpr production.
	EnterRowsecurityoptionalexpr(c *RowsecurityoptionalexprContext)

	// EnterRowsecurityoptionalwithcheck is called when entering the rowsecurityoptionalwithcheck production.
	EnterRowsecurityoptionalwithcheck(c *RowsecurityoptionalwithcheckContext)

	// EnterRowsecuritydefaulttorole is called when entering the rowsecuritydefaulttorole production.
	EnterRowsecuritydefaulttorole(c *RowsecuritydefaulttoroleContext)

	// EnterRowsecurityoptionaltorole is called when entering the rowsecurityoptionaltorole production.
	EnterRowsecurityoptionaltorole(c *RowsecurityoptionaltoroleContext)

	// EnterRowsecuritydefaultpermissive is called when entering the rowsecuritydefaultpermissive production.
	EnterRowsecuritydefaultpermissive(c *RowsecuritydefaultpermissiveContext)

	// EnterRowsecuritydefaultforcmd is called when entering the rowsecuritydefaultforcmd production.
	EnterRowsecuritydefaultforcmd(c *RowsecuritydefaultforcmdContext)

	// EnterRow_security_cmd is called when entering the row_security_cmd production.
	EnterRow_security_cmd(c *Row_security_cmdContext)

	// EnterCreateamstmt is called when entering the createamstmt production.
	EnterCreateamstmt(c *CreateamstmtContext)

	// EnterAm_type is called when entering the am_type production.
	EnterAm_type(c *Am_typeContext)

	// EnterCreatetrigstmt is called when entering the createtrigstmt production.
	EnterCreatetrigstmt(c *CreatetrigstmtContext)

	// EnterTriggeractiontime is called when entering the triggeractiontime production.
	EnterTriggeractiontime(c *TriggeractiontimeContext)

	// EnterTriggerevents is called when entering the triggerevents production.
	EnterTriggerevents(c *TriggereventsContext)

	// EnterTriggeroneevent is called when entering the triggeroneevent production.
	EnterTriggeroneevent(c *TriggeroneeventContext)

	// EnterTriggerreferencing is called when entering the triggerreferencing production.
	EnterTriggerreferencing(c *TriggerreferencingContext)

	// EnterTriggertransitions is called when entering the triggertransitions production.
	EnterTriggertransitions(c *TriggertransitionsContext)

	// EnterTriggertransition is called when entering the triggertransition production.
	EnterTriggertransition(c *TriggertransitionContext)

	// EnterTransitionoldornew is called when entering the transitionoldornew production.
	EnterTransitionoldornew(c *TransitionoldornewContext)

	// EnterTransitionrowortable is called when entering the transitionrowortable production.
	EnterTransitionrowortable(c *TransitionrowortableContext)

	// EnterTransitionrelname is called when entering the transitionrelname production.
	EnterTransitionrelname(c *TransitionrelnameContext)

	// EnterTriggerforspec is called when entering the triggerforspec production.
	EnterTriggerforspec(c *TriggerforspecContext)

	// EnterTriggerforopteach is called when entering the triggerforopteach production.
	EnterTriggerforopteach(c *TriggerforopteachContext)

	// EnterTriggerfortype is called when entering the triggerfortype production.
	EnterTriggerfortype(c *TriggerfortypeContext)

	// EnterTriggerwhen is called when entering the triggerwhen production.
	EnterTriggerwhen(c *TriggerwhenContext)

	// EnterFunction_or_procedure is called when entering the function_or_procedure production.
	EnterFunction_or_procedure(c *Function_or_procedureContext)

	// EnterTriggerfuncargs is called when entering the triggerfuncargs production.
	EnterTriggerfuncargs(c *TriggerfuncargsContext)

	// EnterTriggerfuncarg is called when entering the triggerfuncarg production.
	EnterTriggerfuncarg(c *TriggerfuncargContext)

	// EnterOptconstrfromtable is called when entering the optconstrfromtable production.
	EnterOptconstrfromtable(c *OptconstrfromtableContext)

	// EnterConstraintattributespec is called when entering the constraintattributespec production.
	EnterConstraintattributespec(c *ConstraintattributespecContext)

	// EnterConstraintattributeElem is called when entering the constraintattributeElem production.
	EnterConstraintattributeElem(c *ConstraintattributeElemContext)

	// EnterCreateeventtrigstmt is called when entering the createeventtrigstmt production.
	EnterCreateeventtrigstmt(c *CreateeventtrigstmtContext)

	// EnterEvent_trigger_when_list is called when entering the event_trigger_when_list production.
	EnterEvent_trigger_when_list(c *Event_trigger_when_listContext)

	// EnterEvent_trigger_when_item is called when entering the event_trigger_when_item production.
	EnterEvent_trigger_when_item(c *Event_trigger_when_itemContext)

	// EnterEvent_trigger_value_list is called when entering the event_trigger_value_list production.
	EnterEvent_trigger_value_list(c *Event_trigger_value_listContext)

	// EnterAltereventtrigstmt is called when entering the altereventtrigstmt production.
	EnterAltereventtrigstmt(c *AltereventtrigstmtContext)

	// EnterEnable_trigger is called when entering the enable_trigger production.
	EnterEnable_trigger(c *Enable_triggerContext)

	// EnterCreateassertionstmt is called when entering the createassertionstmt production.
	EnterCreateassertionstmt(c *CreateassertionstmtContext)

	// EnterDefinestmt is called when entering the definestmt production.
	EnterDefinestmt(c *DefinestmtContext)

	// EnterDefinition is called when entering the definition production.
	EnterDefinition(c *DefinitionContext)

	// EnterDef_list is called when entering the def_list production.
	EnterDef_list(c *Def_listContext)

	// EnterDef_elem is called when entering the def_elem production.
	EnterDef_elem(c *Def_elemContext)

	// EnterDef_arg is called when entering the def_arg production.
	EnterDef_arg(c *Def_argContext)

	// EnterOld_aggr_definition is called when entering the old_aggr_definition production.
	EnterOld_aggr_definition(c *Old_aggr_definitionContext)

	// EnterOld_aggr_list is called when entering the old_aggr_list production.
	EnterOld_aggr_list(c *Old_aggr_listContext)

	// EnterOld_aggr_elem is called when entering the old_aggr_elem production.
	EnterOld_aggr_elem(c *Old_aggr_elemContext)

	// EnterOpt_enum_val_list is called when entering the opt_enum_val_list production.
	EnterOpt_enum_val_list(c *Opt_enum_val_listContext)

	// EnterEnum_val_list is called when entering the enum_val_list production.
	EnterEnum_val_list(c *Enum_val_listContext)

	// EnterAlterenumstmt is called when entering the alterenumstmt production.
	EnterAlterenumstmt(c *AlterenumstmtContext)

	// EnterOpt_if_not_exists is called when entering the opt_if_not_exists production.
	EnterOpt_if_not_exists(c *Opt_if_not_existsContext)

	// EnterCreateopclassstmt is called when entering the createopclassstmt production.
	EnterCreateopclassstmt(c *CreateopclassstmtContext)

	// EnterOpclass_item_list is called when entering the opclass_item_list production.
	EnterOpclass_item_list(c *Opclass_item_listContext)

	// EnterOpclass_item is called when entering the opclass_item production.
	EnterOpclass_item(c *Opclass_itemContext)

	// EnterOpt_default is called when entering the opt_default production.
	EnterOpt_default(c *Opt_defaultContext)

	// EnterOpt_opfamily is called when entering the opt_opfamily production.
	EnterOpt_opfamily(c *Opt_opfamilyContext)

	// EnterOpclass_purpose is called when entering the opclass_purpose production.
	EnterOpclass_purpose(c *Opclass_purposeContext)

	// EnterOpt_recheck is called when entering the opt_recheck production.
	EnterOpt_recheck(c *Opt_recheckContext)

	// EnterCreateopfamilystmt is called when entering the createopfamilystmt production.
	EnterCreateopfamilystmt(c *CreateopfamilystmtContext)

	// EnterAlteropfamilystmt is called when entering the alteropfamilystmt production.
	EnterAlteropfamilystmt(c *AlteropfamilystmtContext)

	// EnterOpclass_drop_list is called when entering the opclass_drop_list production.
	EnterOpclass_drop_list(c *Opclass_drop_listContext)

	// EnterOpclass_drop is called when entering the opclass_drop production.
	EnterOpclass_drop(c *Opclass_dropContext)

	// EnterDropopclassstmt is called when entering the dropopclassstmt production.
	EnterDropopclassstmt(c *DropopclassstmtContext)

	// EnterDropopfamilystmt is called when entering the dropopfamilystmt production.
	EnterDropopfamilystmt(c *DropopfamilystmtContext)

	// EnterDropownedstmt is called when entering the dropownedstmt production.
	EnterDropownedstmt(c *DropownedstmtContext)

	// EnterReassignownedstmt is called when entering the reassignownedstmt production.
	EnterReassignownedstmt(c *ReassignownedstmtContext)

	// EnterDropstmt is called when entering the dropstmt production.
	EnterDropstmt(c *DropstmtContext)

	// EnterObject_type_any_name is called when entering the object_type_any_name production.
	EnterObject_type_any_name(c *Object_type_any_nameContext)

	// EnterObject_type_name is called when entering the object_type_name production.
	EnterObject_type_name(c *Object_type_nameContext)

	// EnterDrop_type_name is called when entering the drop_type_name production.
	EnterDrop_type_name(c *Drop_type_nameContext)

	// EnterObject_type_name_on_any_name is called when entering the object_type_name_on_any_name production.
	EnterObject_type_name_on_any_name(c *Object_type_name_on_any_nameContext)

	// EnterAny_name_list is called when entering the any_name_list production.
	EnterAny_name_list(c *Any_name_listContext)

	// EnterAny_name is called when entering the any_name production.
	EnterAny_name(c *Any_nameContext)

	// EnterAttrs is called when entering the attrs production.
	EnterAttrs(c *AttrsContext)

	// EnterType_name_list is called when entering the type_name_list production.
	EnterType_name_list(c *Type_name_listContext)

	// EnterTruncatestmt is called when entering the truncatestmt production.
	EnterTruncatestmt(c *TruncatestmtContext)

	// EnterOpt_restart_seqs is called when entering the opt_restart_seqs production.
	EnterOpt_restart_seqs(c *Opt_restart_seqsContext)

	// EnterCommentstmt is called when entering the commentstmt production.
	EnterCommentstmt(c *CommentstmtContext)

	// EnterComment_text is called when entering the comment_text production.
	EnterComment_text(c *Comment_textContext)

	// EnterSeclabelstmt is called when entering the seclabelstmt production.
	EnterSeclabelstmt(c *SeclabelstmtContext)

	// EnterOpt_provider is called when entering the opt_provider production.
	EnterOpt_provider(c *Opt_providerContext)

	// EnterSecurity_label is called when entering the security_label production.
	EnterSecurity_label(c *Security_labelContext)

	// EnterFetchstmt is called when entering the fetchstmt production.
	EnterFetchstmt(c *FetchstmtContext)

	// EnterFetch_args is called when entering the fetch_args production.
	EnterFetch_args(c *Fetch_argsContext)

	// EnterFrom_in is called when entering the from_in production.
	EnterFrom_in(c *From_inContext)

	// EnterOpt_from_in is called when entering the opt_from_in production.
	EnterOpt_from_in(c *Opt_from_inContext)

	// EnterGrantstmt is called when entering the grantstmt production.
	EnterGrantstmt(c *GrantstmtContext)

	// EnterRevokestmt is called when entering the revokestmt production.
	EnterRevokestmt(c *RevokestmtContext)

	// EnterPrivileges is called when entering the privileges production.
	EnterPrivileges(c *PrivilegesContext)

	// EnterPrivilege_list is called when entering the privilege_list production.
	EnterPrivilege_list(c *Privilege_listContext)

	// EnterPrivilege is called when entering the privilege production.
	EnterPrivilege(c *PrivilegeContext)

	// EnterPrivilege_target is called when entering the privilege_target production.
	EnterPrivilege_target(c *Privilege_targetContext)

	// EnterGrantee_list is called when entering the grantee_list production.
	EnterGrantee_list(c *Grantee_listContext)

	// EnterGrantee is called when entering the grantee production.
	EnterGrantee(c *GranteeContext)

	// EnterOpt_grant_grant_option is called when entering the opt_grant_grant_option production.
	EnterOpt_grant_grant_option(c *Opt_grant_grant_optionContext)

	// EnterGrantrolestmt is called when entering the grantrolestmt production.
	EnterGrantrolestmt(c *GrantrolestmtContext)

	// EnterRevokerolestmt is called when entering the revokerolestmt production.
	EnterRevokerolestmt(c *RevokerolestmtContext)

	// EnterOpt_grant_admin_option is called when entering the opt_grant_admin_option production.
	EnterOpt_grant_admin_option(c *Opt_grant_admin_optionContext)

	// EnterOpt_granted_by is called when entering the opt_granted_by production.
	EnterOpt_granted_by(c *Opt_granted_byContext)

	// EnterAlterdefaultprivilegesstmt is called when entering the alterdefaultprivilegesstmt production.
	EnterAlterdefaultprivilegesstmt(c *AlterdefaultprivilegesstmtContext)

	// EnterDefacloptionlist is called when entering the defacloptionlist production.
	EnterDefacloptionlist(c *DefacloptionlistContext)

	// EnterDefacloption is called when entering the defacloption production.
	EnterDefacloption(c *DefacloptionContext)

	// EnterDefaclaction is called when entering the defaclaction production.
	EnterDefaclaction(c *DefaclactionContext)

	// EnterDefacl_privilege_target is called when entering the defacl_privilege_target production.
	EnterDefacl_privilege_target(c *Defacl_privilege_targetContext)

	// EnterIndexstmt is called when entering the indexstmt production.
	EnterIndexstmt(c *IndexstmtContext)

	// EnterOpt_unique is called when entering the opt_unique production.
	EnterOpt_unique(c *Opt_uniqueContext)

	// EnterOpt_concurrently is called when entering the opt_concurrently production.
	EnterOpt_concurrently(c *Opt_concurrentlyContext)

	// EnterOpt_index_name is called when entering the opt_index_name production.
	EnterOpt_index_name(c *Opt_index_nameContext)

	// EnterAccess_method_clause is called when entering the access_method_clause production.
	EnterAccess_method_clause(c *Access_method_clauseContext)

	// EnterIndex_params is called when entering the index_params production.
	EnterIndex_params(c *Index_paramsContext)

	// EnterIndex_elem_options is called when entering the index_elem_options production.
	EnterIndex_elem_options(c *Index_elem_optionsContext)

	// EnterIndex_elem is called when entering the index_elem production.
	EnterIndex_elem(c *Index_elemContext)

	// EnterOpt_include is called when entering the opt_include production.
	EnterOpt_include(c *Opt_includeContext)

	// EnterIndex_including_params is called when entering the index_including_params production.
	EnterIndex_including_params(c *Index_including_paramsContext)

	// EnterOpt_collate is called when entering the opt_collate production.
	EnterOpt_collate(c *Opt_collateContext)

	// EnterOpt_class is called when entering the opt_class production.
	EnterOpt_class(c *Opt_classContext)

	// EnterOpt_asc_desc is called when entering the opt_asc_desc production.
	EnterOpt_asc_desc(c *Opt_asc_descContext)

	// EnterOpt_nulls_order is called when entering the opt_nulls_order production.
	EnterOpt_nulls_order(c *Opt_nulls_orderContext)

	// EnterCreatefunctionstmt is called when entering the createfunctionstmt production.
	EnterCreatefunctionstmt(c *CreatefunctionstmtContext)

	// EnterOpt_or_replace is called when entering the opt_or_replace production.
	EnterOpt_or_replace(c *Opt_or_replaceContext)

	// EnterFunc_args is called when entering the func_args production.
	EnterFunc_args(c *Func_argsContext)

	// EnterFunc_args_list is called when entering the func_args_list production.
	EnterFunc_args_list(c *Func_args_listContext)

	// EnterFunction_with_argtypes_list is called when entering the function_with_argtypes_list production.
	EnterFunction_with_argtypes_list(c *Function_with_argtypes_listContext)

	// EnterFunction_with_argtypes is called when entering the function_with_argtypes production.
	EnterFunction_with_argtypes(c *Function_with_argtypesContext)

	// EnterFunc_args_with_defaults is called when entering the func_args_with_defaults production.
	EnterFunc_args_with_defaults(c *Func_args_with_defaultsContext)

	// EnterFunc_args_with_defaults_list is called when entering the func_args_with_defaults_list production.
	EnterFunc_args_with_defaults_list(c *Func_args_with_defaults_listContext)

	// EnterFunc_arg is called when entering the func_arg production.
	EnterFunc_arg(c *Func_argContext)

	// EnterArg_class is called when entering the arg_class production.
	EnterArg_class(c *Arg_classContext)

	// EnterParam_name is called when entering the param_name production.
	EnterParam_name(c *Param_nameContext)

	// EnterFunc_return is called when entering the func_return production.
	EnterFunc_return(c *Func_returnContext)

	// EnterFunc_type is called when entering the func_type production.
	EnterFunc_type(c *Func_typeContext)

	// EnterFunc_arg_with_default is called when entering the func_arg_with_default production.
	EnterFunc_arg_with_default(c *Func_arg_with_defaultContext)

	// EnterAggr_arg is called when entering the aggr_arg production.
	EnterAggr_arg(c *Aggr_argContext)

	// EnterAggr_args is called when entering the aggr_args production.
	EnterAggr_args(c *Aggr_argsContext)

	// EnterAggr_args_list is called when entering the aggr_args_list production.
	EnterAggr_args_list(c *Aggr_args_listContext)

	// EnterAggregate_with_argtypes is called when entering the aggregate_with_argtypes production.
	EnterAggregate_with_argtypes(c *Aggregate_with_argtypesContext)

	// EnterAggregate_with_argtypes_list is called when entering the aggregate_with_argtypes_list production.
	EnterAggregate_with_argtypes_list(c *Aggregate_with_argtypes_listContext)

	// EnterCreatefunc_opt_list is called when entering the createfunc_opt_list production.
	EnterCreatefunc_opt_list(c *Createfunc_opt_listContext)

	// EnterCommon_func_opt_item is called when entering the common_func_opt_item production.
	EnterCommon_func_opt_item(c *Common_func_opt_itemContext)

	// EnterCreatefunc_opt_item is called when entering the createfunc_opt_item production.
	EnterCreatefunc_opt_item(c *Createfunc_opt_itemContext)

	// EnterFunc_as is called when entering the func_as production.
	EnterFunc_as(c *Func_asContext)

	// EnterTransform_type_list is called when entering the transform_type_list production.
	EnterTransform_type_list(c *Transform_type_listContext)

	// EnterOpt_definition is called when entering the opt_definition production.
	EnterOpt_definition(c *Opt_definitionContext)

	// EnterTable_func_column is called when entering the table_func_column production.
	EnterTable_func_column(c *Table_func_columnContext)

	// EnterTable_func_column_list is called when entering the table_func_column_list production.
	EnterTable_func_column_list(c *Table_func_column_listContext)

	// EnterAlterfunctionstmt is called when entering the alterfunctionstmt production.
	EnterAlterfunctionstmt(c *AlterfunctionstmtContext)

	// EnterAlterfunc_opt_list is called when entering the alterfunc_opt_list production.
	EnterAlterfunc_opt_list(c *Alterfunc_opt_listContext)

	// EnterOpt_restrict is called when entering the opt_restrict production.
	EnterOpt_restrict(c *Opt_restrictContext)

	// EnterRemovefuncstmt is called when entering the removefuncstmt production.
	EnterRemovefuncstmt(c *RemovefuncstmtContext)

	// EnterRemoveaggrstmt is called when entering the removeaggrstmt production.
	EnterRemoveaggrstmt(c *RemoveaggrstmtContext)

	// EnterRemoveoperstmt is called when entering the removeoperstmt production.
	EnterRemoveoperstmt(c *RemoveoperstmtContext)

	// EnterOper_argtypes is called when entering the oper_argtypes production.
	EnterOper_argtypes(c *Oper_argtypesContext)

	// EnterAny_operator is called when entering the any_operator production.
	EnterAny_operator(c *Any_operatorContext)

	// EnterOperator_with_argtypes_list is called when entering the operator_with_argtypes_list production.
	EnterOperator_with_argtypes_list(c *Operator_with_argtypes_listContext)

	// EnterOperator_with_argtypes is called when entering the operator_with_argtypes production.
	EnterOperator_with_argtypes(c *Operator_with_argtypesContext)

	// EnterDostmt is called when entering the dostmt production.
	EnterDostmt(c *DostmtContext)

	// EnterDostmt_opt_list is called when entering the dostmt_opt_list production.
	EnterDostmt_opt_list(c *Dostmt_opt_listContext)

	// EnterDostmt_opt_item is called when entering the dostmt_opt_item production.
	EnterDostmt_opt_item(c *Dostmt_opt_itemContext)

	// EnterCreatecaststmt is called when entering the createcaststmt production.
	EnterCreatecaststmt(c *CreatecaststmtContext)

	// EnterCast_context is called when entering the cast_context production.
	EnterCast_context(c *Cast_contextContext)

	// EnterDropcaststmt is called when entering the dropcaststmt production.
	EnterDropcaststmt(c *DropcaststmtContext)

	// EnterOpt_if_exists is called when entering the opt_if_exists production.
	EnterOpt_if_exists(c *Opt_if_existsContext)

	// EnterCreatetransformstmt is called when entering the createtransformstmt production.
	EnterCreatetransformstmt(c *CreatetransformstmtContext)

	// EnterTransform_element_list is called when entering the transform_element_list production.
	EnterTransform_element_list(c *Transform_element_listContext)

	// EnterDroptransformstmt is called when entering the droptransformstmt production.
	EnterDroptransformstmt(c *DroptransformstmtContext)

	// EnterReindexstmt is called when entering the reindexstmt production.
	EnterReindexstmt(c *ReindexstmtContext)

	// EnterReindex_target_type is called when entering the reindex_target_type production.
	EnterReindex_target_type(c *Reindex_target_typeContext)

	// EnterReindex_target_multitable is called when entering the reindex_target_multitable production.
	EnterReindex_target_multitable(c *Reindex_target_multitableContext)

	// EnterReindex_option_list is called when entering the reindex_option_list production.
	EnterReindex_option_list(c *Reindex_option_listContext)

	// EnterReindex_option_elem is called when entering the reindex_option_elem production.
	EnterReindex_option_elem(c *Reindex_option_elemContext)

	// EnterAltertblspcstmt is called when entering the altertblspcstmt production.
	EnterAltertblspcstmt(c *AltertblspcstmtContext)

	// EnterRenamestmt is called when entering the renamestmt production.
	EnterRenamestmt(c *RenamestmtContext)

	// EnterOpt_column is called when entering the opt_column production.
	EnterOpt_column(c *Opt_columnContext)

	// EnterOpt_set_data is called when entering the opt_set_data production.
	EnterOpt_set_data(c *Opt_set_dataContext)

	// EnterAlterobjectdependsstmt is called when entering the alterobjectdependsstmt production.
	EnterAlterobjectdependsstmt(c *AlterobjectdependsstmtContext)

	// EnterOpt_no is called when entering the opt_no production.
	EnterOpt_no(c *Opt_noContext)

	// EnterAlterobjectschemastmt is called when entering the alterobjectschemastmt production.
	EnterAlterobjectschemastmt(c *AlterobjectschemastmtContext)

	// EnterAlteroperatorstmt is called when entering the alteroperatorstmt production.
	EnterAlteroperatorstmt(c *AlteroperatorstmtContext)

	// EnterOperator_def_list is called when entering the operator_def_list production.
	EnterOperator_def_list(c *Operator_def_listContext)

	// EnterOperator_def_elem is called when entering the operator_def_elem production.
	EnterOperator_def_elem(c *Operator_def_elemContext)

	// EnterOperator_def_arg is called when entering the operator_def_arg production.
	EnterOperator_def_arg(c *Operator_def_argContext)

	// EnterAltertypestmt is called when entering the altertypestmt production.
	EnterAltertypestmt(c *AltertypestmtContext)

	// EnterAlterownerstmt is called when entering the alterownerstmt production.
	EnterAlterownerstmt(c *AlterownerstmtContext)

	// EnterCreatepublicationstmt is called when entering the createpublicationstmt production.
	EnterCreatepublicationstmt(c *CreatepublicationstmtContext)

	// EnterOpt_publication_for_tables is called when entering the opt_publication_for_tables production.
	EnterOpt_publication_for_tables(c *Opt_publication_for_tablesContext)

	// EnterPublication_for_tables is called when entering the publication_for_tables production.
	EnterPublication_for_tables(c *Publication_for_tablesContext)

	// EnterAlterpublicationstmt is called when entering the alterpublicationstmt production.
	EnterAlterpublicationstmt(c *AlterpublicationstmtContext)

	// EnterCreatesubscriptionstmt is called when entering the createsubscriptionstmt production.
	EnterCreatesubscriptionstmt(c *CreatesubscriptionstmtContext)

	// EnterPublication_name_list is called when entering the publication_name_list production.
	EnterPublication_name_list(c *Publication_name_listContext)

	// EnterPublication_name_item is called when entering the publication_name_item production.
	EnterPublication_name_item(c *Publication_name_itemContext)

	// EnterAltersubscriptionstmt is called when entering the altersubscriptionstmt production.
	EnterAltersubscriptionstmt(c *AltersubscriptionstmtContext)

	// EnterDropsubscriptionstmt is called when entering the dropsubscriptionstmt production.
	EnterDropsubscriptionstmt(c *DropsubscriptionstmtContext)

	// EnterRulestmt is called when entering the rulestmt production.
	EnterRulestmt(c *RulestmtContext)

	// EnterRuleactionlist is called when entering the ruleactionlist production.
	EnterRuleactionlist(c *RuleactionlistContext)

	// EnterRuleactionmulti is called when entering the ruleactionmulti production.
	EnterRuleactionmulti(c *RuleactionmultiContext)

	// EnterRuleactionstmt is called when entering the ruleactionstmt production.
	EnterRuleactionstmt(c *RuleactionstmtContext)

	// EnterRuleactionstmtOrEmpty is called when entering the ruleactionstmtOrEmpty production.
	EnterRuleactionstmtOrEmpty(c *RuleactionstmtOrEmptyContext)

	// EnterEvent is called when entering the event production.
	EnterEvent(c *EventContext)

	// EnterOpt_instead is called when entering the opt_instead production.
	EnterOpt_instead(c *Opt_insteadContext)

	// EnterNotifystmt is called when entering the notifystmt production.
	EnterNotifystmt(c *NotifystmtContext)

	// EnterNotify_payload is called when entering the notify_payload production.
	EnterNotify_payload(c *Notify_payloadContext)

	// EnterListenstmt is called when entering the listenstmt production.
	EnterListenstmt(c *ListenstmtContext)

	// EnterUnlistenstmt is called when entering the unlistenstmt production.
	EnterUnlistenstmt(c *UnlistenstmtContext)

	// EnterTransactionstmt is called when entering the transactionstmt production.
	EnterTransactionstmt(c *TransactionstmtContext)

	// EnterOpt_transaction is called when entering the opt_transaction production.
	EnterOpt_transaction(c *Opt_transactionContext)

	// EnterTransaction_mode_item is called when entering the transaction_mode_item production.
	EnterTransaction_mode_item(c *Transaction_mode_itemContext)

	// EnterTransaction_mode_list is called when entering the transaction_mode_list production.
	EnterTransaction_mode_list(c *Transaction_mode_listContext)

	// EnterTransaction_mode_list_or_empty is called when entering the transaction_mode_list_or_empty production.
	EnterTransaction_mode_list_or_empty(c *Transaction_mode_list_or_emptyContext)

	// EnterOpt_transaction_chain is called when entering the opt_transaction_chain production.
	EnterOpt_transaction_chain(c *Opt_transaction_chainContext)

	// EnterViewstmt is called when entering the viewstmt production.
	EnterViewstmt(c *ViewstmtContext)

	// EnterOpt_check_option is called when entering the opt_check_option production.
	EnterOpt_check_option(c *Opt_check_optionContext)

	// EnterLoadstmt is called when entering the loadstmt production.
	EnterLoadstmt(c *LoadstmtContext)

	// EnterCreatedbstmt is called when entering the createdbstmt production.
	EnterCreatedbstmt(c *CreatedbstmtContext)

	// EnterCreatedb_opt_list is called when entering the createdb_opt_list production.
	EnterCreatedb_opt_list(c *Createdb_opt_listContext)

	// EnterCreatedb_opt_items is called when entering the createdb_opt_items production.
	EnterCreatedb_opt_items(c *Createdb_opt_itemsContext)

	// EnterCreatedb_opt_item is called when entering the createdb_opt_item production.
	EnterCreatedb_opt_item(c *Createdb_opt_itemContext)

	// EnterCreatedb_opt_name is called when entering the createdb_opt_name production.
	EnterCreatedb_opt_name(c *Createdb_opt_nameContext)

	// EnterOpt_equal is called when entering the opt_equal production.
	EnterOpt_equal(c *Opt_equalContext)

	// EnterAlterdatabasestmt is called when entering the alterdatabasestmt production.
	EnterAlterdatabasestmt(c *AlterdatabasestmtContext)

	// EnterAlterdatabasesetstmt is called when entering the alterdatabasesetstmt production.
	EnterAlterdatabasesetstmt(c *AlterdatabasesetstmtContext)

	// EnterDropdbstmt is called when entering the dropdbstmt production.
	EnterDropdbstmt(c *DropdbstmtContext)

	// EnterDrop_option_list is called when entering the drop_option_list production.
	EnterDrop_option_list(c *Drop_option_listContext)

	// EnterDrop_option is called when entering the drop_option production.
	EnterDrop_option(c *Drop_optionContext)

	// EnterAltercollationstmt is called when entering the altercollationstmt production.
	EnterAltercollationstmt(c *AltercollationstmtContext)

	// EnterAltersystemstmt is called when entering the altersystemstmt production.
	EnterAltersystemstmt(c *AltersystemstmtContext)

	// EnterCreatedomainstmt is called when entering the createdomainstmt production.
	EnterCreatedomainstmt(c *CreatedomainstmtContext)

	// EnterAlterdomainstmt is called when entering the alterdomainstmt production.
	EnterAlterdomainstmt(c *AlterdomainstmtContext)

	// EnterOpt_as is called when entering the opt_as production.
	EnterOpt_as(c *Opt_asContext)

	// EnterAltertsdictionarystmt is called when entering the altertsdictionarystmt production.
	EnterAltertsdictionarystmt(c *AltertsdictionarystmtContext)

	// EnterAltertsconfigurationstmt is called when entering the altertsconfigurationstmt production.
	EnterAltertsconfigurationstmt(c *AltertsconfigurationstmtContext)

	// EnterAny_with is called when entering the any_with production.
	EnterAny_with(c *Any_withContext)

	// EnterCreateconversionstmt is called when entering the createconversionstmt production.
	EnterCreateconversionstmt(c *CreateconversionstmtContext)

	// EnterClusterstmt is called when entering the clusterstmt production.
	EnterClusterstmt(c *ClusterstmtContext)

	// EnterCluster_index_specification is called when entering the cluster_index_specification production.
	EnterCluster_index_specification(c *Cluster_index_specificationContext)

	// EnterVacuumstmt is called when entering the vacuumstmt production.
	EnterVacuumstmt(c *VacuumstmtContext)

	// EnterAnalyzestmt is called when entering the analyzestmt production.
	EnterAnalyzestmt(c *AnalyzestmtContext)

	// EnterVac_analyze_option_list is called when entering the vac_analyze_option_list production.
	EnterVac_analyze_option_list(c *Vac_analyze_option_listContext)

	// EnterAnalyze_keyword is called when entering the analyze_keyword production.
	EnterAnalyze_keyword(c *Analyze_keywordContext)

	// EnterVac_analyze_option_elem is called when entering the vac_analyze_option_elem production.
	EnterVac_analyze_option_elem(c *Vac_analyze_option_elemContext)

	// EnterVac_analyze_option_name is called when entering the vac_analyze_option_name production.
	EnterVac_analyze_option_name(c *Vac_analyze_option_nameContext)

	// EnterVac_analyze_option_arg is called when entering the vac_analyze_option_arg production.
	EnterVac_analyze_option_arg(c *Vac_analyze_option_argContext)

	// EnterOpt_analyze is called when entering the opt_analyze production.
	EnterOpt_analyze(c *Opt_analyzeContext)

	// EnterOpt_verbose is called when entering the opt_verbose production.
	EnterOpt_verbose(c *Opt_verboseContext)

	// EnterOpt_full is called when entering the opt_full production.
	EnterOpt_full(c *Opt_fullContext)

	// EnterOpt_freeze is called when entering the opt_freeze production.
	EnterOpt_freeze(c *Opt_freezeContext)

	// EnterOpt_name_list is called when entering the opt_name_list production.
	EnterOpt_name_list(c *Opt_name_listContext)

	// EnterVacuum_relation is called when entering the vacuum_relation production.
	EnterVacuum_relation(c *Vacuum_relationContext)

	// EnterVacuum_relation_list is called when entering the vacuum_relation_list production.
	EnterVacuum_relation_list(c *Vacuum_relation_listContext)

	// EnterOpt_vacuum_relation_list is called when entering the opt_vacuum_relation_list production.
	EnterOpt_vacuum_relation_list(c *Opt_vacuum_relation_listContext)

	// EnterExplainstmt is called when entering the explainstmt production.
	EnterExplainstmt(c *ExplainstmtContext)

	// EnterExplainablestmt is called when entering the explainablestmt production.
	EnterExplainablestmt(c *ExplainablestmtContext)

	// EnterExplain_option_list is called when entering the explain_option_list production.
	EnterExplain_option_list(c *Explain_option_listContext)

	// EnterExplain_option_elem is called when entering the explain_option_elem production.
	EnterExplain_option_elem(c *Explain_option_elemContext)

	// EnterExplain_option_name is called when entering the explain_option_name production.
	EnterExplain_option_name(c *Explain_option_nameContext)

	// EnterExplain_option_arg is called when entering the explain_option_arg production.
	EnterExplain_option_arg(c *Explain_option_argContext)

	// EnterPreparestmt is called when entering the preparestmt production.
	EnterPreparestmt(c *PreparestmtContext)

	// EnterPrep_type_clause is called when entering the prep_type_clause production.
	EnterPrep_type_clause(c *Prep_type_clauseContext)

	// EnterPreparablestmt is called when entering the preparablestmt production.
	EnterPreparablestmt(c *PreparablestmtContext)

	// EnterExecutestmt is called when entering the executestmt production.
	EnterExecutestmt(c *ExecutestmtContext)

	// EnterExecute_param_clause is called when entering the execute_param_clause production.
	EnterExecute_param_clause(c *Execute_param_clauseContext)

	// EnterDeallocatestmt is called when entering the deallocatestmt production.
	EnterDeallocatestmt(c *DeallocatestmtContext)

	// EnterInsertstmt is called when entering the insertstmt production.
	EnterInsertstmt(c *InsertstmtContext)

	// EnterInsert_target is called when entering the insert_target production.
	EnterInsert_target(c *Insert_targetContext)

	// EnterInsert_rest is called when entering the insert_rest production.
	EnterInsert_rest(c *Insert_restContext)

	// EnterOverride_kind is called when entering the override_kind production.
	EnterOverride_kind(c *Override_kindContext)

	// EnterInsert_column_list is called when entering the insert_column_list production.
	EnterInsert_column_list(c *Insert_column_listContext)

	// EnterInsert_column_item is called when entering the insert_column_item production.
	EnterInsert_column_item(c *Insert_column_itemContext)

	// EnterOpt_on_conflict is called when entering the opt_on_conflict production.
	EnterOpt_on_conflict(c *Opt_on_conflictContext)

	// EnterOpt_conf_expr is called when entering the opt_conf_expr production.
	EnterOpt_conf_expr(c *Opt_conf_exprContext)

	// EnterReturning_clause is called when entering the returning_clause production.
	EnterReturning_clause(c *Returning_clauseContext)

	// EnterMergestmt is called when entering the mergestmt production.
	EnterMergestmt(c *MergestmtContext)

	// EnterMerge_insert_clause is called when entering the merge_insert_clause production.
	EnterMerge_insert_clause(c *Merge_insert_clauseContext)

	// EnterMerge_update_clause is called when entering the merge_update_clause production.
	EnterMerge_update_clause(c *Merge_update_clauseContext)

	// EnterMerge_delete_clause is called when entering the merge_delete_clause production.
	EnterMerge_delete_clause(c *Merge_delete_clauseContext)

	// EnterDeletestmt is called when entering the deletestmt production.
	EnterDeletestmt(c *DeletestmtContext)

	// EnterUsing_clause is called when entering the using_clause production.
	EnterUsing_clause(c *Using_clauseContext)

	// EnterLockstmt is called when entering the lockstmt production.
	EnterLockstmt(c *LockstmtContext)

	// EnterOpt_lock is called when entering the opt_lock production.
	EnterOpt_lock(c *Opt_lockContext)

	// EnterLock_type is called when entering the lock_type production.
	EnterLock_type(c *Lock_typeContext)

	// EnterOpt_nowait is called when entering the opt_nowait production.
	EnterOpt_nowait(c *Opt_nowaitContext)

	// EnterOpt_nowait_or_skip is called when entering the opt_nowait_or_skip production.
	EnterOpt_nowait_or_skip(c *Opt_nowait_or_skipContext)

	// EnterUpdatestmt is called when entering the updatestmt production.
	EnterUpdatestmt(c *UpdatestmtContext)

	// EnterSet_clause_list is called when entering the set_clause_list production.
	EnterSet_clause_list(c *Set_clause_listContext)

	// EnterSet_clause is called when entering the set_clause production.
	EnterSet_clause(c *Set_clauseContext)

	// EnterSet_target is called when entering the set_target production.
	EnterSet_target(c *Set_targetContext)

	// EnterSet_target_list is called when entering the set_target_list production.
	EnterSet_target_list(c *Set_target_listContext)

	// EnterDeclarecursorstmt is called when entering the declarecursorstmt production.
	EnterDeclarecursorstmt(c *DeclarecursorstmtContext)

	// EnterCursor_name is called when entering the cursor_name production.
	EnterCursor_name(c *Cursor_nameContext)

	// EnterCursor_options is called when entering the cursor_options production.
	EnterCursor_options(c *Cursor_optionsContext)

	// EnterOpt_hold is called when entering the opt_hold production.
	EnterOpt_hold(c *Opt_holdContext)

	// EnterSelectstmt is called when entering the selectstmt production.
	EnterSelectstmt(c *SelectstmtContext)

	// EnterSelect_with_parens is called when entering the select_with_parens production.
	EnterSelect_with_parens(c *Select_with_parensContext)

	// EnterSelect_no_parens is called when entering the select_no_parens production.
	EnterSelect_no_parens(c *Select_no_parensContext)

	// EnterSelect_clause is called when entering the select_clause production.
	EnterSelect_clause(c *Select_clauseContext)

	// EnterSimple_select_intersect is called when entering the simple_select_intersect production.
	EnterSimple_select_intersect(c *Simple_select_intersectContext)

	// EnterSimple_select_pramary is called when entering the simple_select_pramary production.
	EnterSimple_select_pramary(c *Simple_select_pramaryContext)

	// EnterWith_clause is called when entering the with_clause production.
	EnterWith_clause(c *With_clauseContext)

	// EnterCte_list is called when entering the cte_list production.
	EnterCte_list(c *Cte_listContext)

	// EnterCommon_table_expr is called when entering the common_table_expr production.
	EnterCommon_table_expr(c *Common_table_exprContext)

	// EnterOpt_materialized is called when entering the opt_materialized production.
	EnterOpt_materialized(c *Opt_materializedContext)

	// EnterOpt_with_clause is called when entering the opt_with_clause production.
	EnterOpt_with_clause(c *Opt_with_clauseContext)

	// EnterInto_clause is called when entering the into_clause production.
	EnterInto_clause(c *Into_clauseContext)

	// EnterOpt_strict is called when entering the opt_strict production.
	EnterOpt_strict(c *Opt_strictContext)

	// EnterOpttempTableName is called when entering the opttempTableName production.
	EnterOpttempTableName(c *OpttempTableNameContext)

	// EnterOpt_table is called when entering the opt_table production.
	EnterOpt_table(c *Opt_tableContext)

	// EnterAll_or_distinct is called when entering the all_or_distinct production.
	EnterAll_or_distinct(c *All_or_distinctContext)

	// EnterDistinct_clause is called when entering the distinct_clause production.
	EnterDistinct_clause(c *Distinct_clauseContext)

	// EnterOpt_all_clause is called when entering the opt_all_clause production.
	EnterOpt_all_clause(c *Opt_all_clauseContext)

	// EnterOpt_sort_clause is called when entering the opt_sort_clause production.
	EnterOpt_sort_clause(c *Opt_sort_clauseContext)

	// EnterSort_clause is called when entering the sort_clause production.
	EnterSort_clause(c *Sort_clauseContext)

	// EnterSortby_list is called when entering the sortby_list production.
	EnterSortby_list(c *Sortby_listContext)

	// EnterSortby is called when entering the sortby production.
	EnterSortby(c *SortbyContext)

	// EnterSelect_limit is called when entering the select_limit production.
	EnterSelect_limit(c *Select_limitContext)

	// EnterOpt_select_limit is called when entering the opt_select_limit production.
	EnterOpt_select_limit(c *Opt_select_limitContext)

	// EnterLimit_clause is called when entering the limit_clause production.
	EnterLimit_clause(c *Limit_clauseContext)

	// EnterOffset_clause is called when entering the offset_clause production.
	EnterOffset_clause(c *Offset_clauseContext)

	// EnterSelect_limit_value is called when entering the select_limit_value production.
	EnterSelect_limit_value(c *Select_limit_valueContext)

	// EnterSelect_offset_value is called when entering the select_offset_value production.
	EnterSelect_offset_value(c *Select_offset_valueContext)

	// EnterSelect_fetch_first_value is called when entering the select_fetch_first_value production.
	EnterSelect_fetch_first_value(c *Select_fetch_first_valueContext)

	// EnterI_or_f_const is called when entering the i_or_f_const production.
	EnterI_or_f_const(c *I_or_f_constContext)

	// EnterRow_or_rows is called when entering the row_or_rows production.
	EnterRow_or_rows(c *Row_or_rowsContext)

	// EnterFirst_or_next is called when entering the first_or_next production.
	EnterFirst_or_next(c *First_or_nextContext)

	// EnterGroup_clause is called when entering the group_clause production.
	EnterGroup_clause(c *Group_clauseContext)

	// EnterGroup_by_list is called when entering the group_by_list production.
	EnterGroup_by_list(c *Group_by_listContext)

	// EnterGroup_by_item is called when entering the group_by_item production.
	EnterGroup_by_item(c *Group_by_itemContext)

	// EnterEmpty_grouping_set is called when entering the empty_grouping_set production.
	EnterEmpty_grouping_set(c *Empty_grouping_setContext)

	// EnterRollup_clause is called when entering the rollup_clause production.
	EnterRollup_clause(c *Rollup_clauseContext)

	// EnterCube_clause is called when entering the cube_clause production.
	EnterCube_clause(c *Cube_clauseContext)

	// EnterGrouping_sets_clause is called when entering the grouping_sets_clause production.
	EnterGrouping_sets_clause(c *Grouping_sets_clauseContext)

	// EnterHaving_clause is called when entering the having_clause production.
	EnterHaving_clause(c *Having_clauseContext)

	// EnterFor_locking_clause is called when entering the for_locking_clause production.
	EnterFor_locking_clause(c *For_locking_clauseContext)

	// EnterOpt_for_locking_clause is called when entering the opt_for_locking_clause production.
	EnterOpt_for_locking_clause(c *Opt_for_locking_clauseContext)

	// EnterFor_locking_items is called when entering the for_locking_items production.
	EnterFor_locking_items(c *For_locking_itemsContext)

	// EnterFor_locking_item is called when entering the for_locking_item production.
	EnterFor_locking_item(c *For_locking_itemContext)

	// EnterFor_locking_strength is called when entering the for_locking_strength production.
	EnterFor_locking_strength(c *For_locking_strengthContext)

	// EnterLocked_rels_list is called when entering the locked_rels_list production.
	EnterLocked_rels_list(c *Locked_rels_listContext)

	// EnterValues_clause is called when entering the values_clause production.
	EnterValues_clause(c *Values_clauseContext)

	// EnterFrom_clause is called when entering the from_clause production.
	EnterFrom_clause(c *From_clauseContext)

	// EnterFrom_list is called when entering the from_list production.
	EnterFrom_list(c *From_listContext)

	// EnterNon_ansi_join is called when entering the non_ansi_join production.
	EnterNon_ansi_join(c *Non_ansi_joinContext)

	// EnterTable_ref is called when entering the table_ref production.
	EnterTable_ref(c *Table_refContext)

	// EnterAlias_clause is called when entering the alias_clause production.
	EnterAlias_clause(c *Alias_clauseContext)

	// EnterOpt_alias_clause is called when entering the opt_alias_clause production.
	EnterOpt_alias_clause(c *Opt_alias_clauseContext)

	// EnterTable_alias_clause is called when entering the table_alias_clause production.
	EnterTable_alias_clause(c *Table_alias_clauseContext)

	// EnterFunc_alias_clause is called when entering the func_alias_clause production.
	EnterFunc_alias_clause(c *Func_alias_clauseContext)

	// EnterJoin_type is called when entering the join_type production.
	EnterJoin_type(c *Join_typeContext)

	// EnterJoin_qual is called when entering the join_qual production.
	EnterJoin_qual(c *Join_qualContext)

	// EnterRelation_expr is called when entering the relation_expr production.
	EnterRelation_expr(c *Relation_exprContext)

	// EnterRelation_expr_list is called when entering the relation_expr_list production.
	EnterRelation_expr_list(c *Relation_expr_listContext)

	// EnterRelation_expr_opt_alias is called when entering the relation_expr_opt_alias production.
	EnterRelation_expr_opt_alias(c *Relation_expr_opt_aliasContext)

	// EnterTablesample_clause is called when entering the tablesample_clause production.
	EnterTablesample_clause(c *Tablesample_clauseContext)

	// EnterOpt_repeatable_clause is called when entering the opt_repeatable_clause production.
	EnterOpt_repeatable_clause(c *Opt_repeatable_clauseContext)

	// EnterFunc_table is called when entering the func_table production.
	EnterFunc_table(c *Func_tableContext)

	// EnterRowsfrom_item is called when entering the rowsfrom_item production.
	EnterRowsfrom_item(c *Rowsfrom_itemContext)

	// EnterRowsfrom_list is called when entering the rowsfrom_list production.
	EnterRowsfrom_list(c *Rowsfrom_listContext)

	// EnterOpt_col_def_list is called when entering the opt_col_def_list production.
	EnterOpt_col_def_list(c *Opt_col_def_listContext)

	// EnterOpt_ordinality is called when entering the opt_ordinality production.
	EnterOpt_ordinality(c *Opt_ordinalityContext)

	// EnterWhere_clause is called when entering the where_clause production.
	EnterWhere_clause(c *Where_clauseContext)

	// EnterWhere_or_current_clause is called when entering the where_or_current_clause production.
	EnterWhere_or_current_clause(c *Where_or_current_clauseContext)

	// EnterOpttablefuncelementlist is called when entering the opttablefuncelementlist production.
	EnterOpttablefuncelementlist(c *OpttablefuncelementlistContext)

	// EnterTablefuncelementlist is called when entering the tablefuncelementlist production.
	EnterTablefuncelementlist(c *TablefuncelementlistContext)

	// EnterTablefuncelement is called when entering the tablefuncelement production.
	EnterTablefuncelement(c *TablefuncelementContext)

	// EnterXmltable is called when entering the xmltable production.
	EnterXmltable(c *XmltableContext)

	// EnterXmltable_column_list is called when entering the xmltable_column_list production.
	EnterXmltable_column_list(c *Xmltable_column_listContext)

	// EnterXmltable_column_el is called when entering the xmltable_column_el production.
	EnterXmltable_column_el(c *Xmltable_column_elContext)

	// EnterXmltable_column_option_list is called when entering the xmltable_column_option_list production.
	EnterXmltable_column_option_list(c *Xmltable_column_option_listContext)

	// EnterXmltable_column_option_el is called when entering the xmltable_column_option_el production.
	EnterXmltable_column_option_el(c *Xmltable_column_option_elContext)

	// EnterXml_namespace_list is called when entering the xml_namespace_list production.
	EnterXml_namespace_list(c *Xml_namespace_listContext)

	// EnterXml_namespace_el is called when entering the xml_namespace_el production.
	EnterXml_namespace_el(c *Xml_namespace_elContext)

	// EnterTypename is called when entering the typename production.
	EnterTypename(c *TypenameContext)

	// EnterOpt_array_bounds is called when entering the opt_array_bounds production.
	EnterOpt_array_bounds(c *Opt_array_boundsContext)

	// EnterSimpletypename is called when entering the simpletypename production.
	EnterSimpletypename(c *SimpletypenameContext)

	// EnterConsttypename is called when entering the consttypename production.
	EnterConsttypename(c *ConsttypenameContext)

	// EnterGenerictype is called when entering the generictype production.
	EnterGenerictype(c *GenerictypeContext)

	// EnterOpt_type_modifiers is called when entering the opt_type_modifiers production.
	EnterOpt_type_modifiers(c *Opt_type_modifiersContext)

	// EnterNumeric is called when entering the numeric production.
	EnterNumeric(c *NumericContext)

	// EnterOpt_float is called when entering the opt_float production.
	EnterOpt_float(c *Opt_floatContext)

	// EnterBit is called when entering the bit production.
	EnterBit(c *BitContext)

	// EnterConstbit is called when entering the constbit production.
	EnterConstbit(c *ConstbitContext)

	// EnterBitwithlength is called when entering the bitwithlength production.
	EnterBitwithlength(c *BitwithlengthContext)

	// EnterBitwithoutlength is called when entering the bitwithoutlength production.
	EnterBitwithoutlength(c *BitwithoutlengthContext)

	// EnterCharacter is called when entering the character production.
	EnterCharacter(c *CharacterContext)

	// EnterConstcharacter is called when entering the constcharacter production.
	EnterConstcharacter(c *ConstcharacterContext)

	// EnterCharacter_c is called when entering the character_c production.
	EnterCharacter_c(c *Character_cContext)

	// EnterOpt_varying is called when entering the opt_varying production.
	EnterOpt_varying(c *Opt_varyingContext)

	// EnterConstdatetime is called when entering the constdatetime production.
	EnterConstdatetime(c *ConstdatetimeContext)

	// EnterConstinterval is called when entering the constinterval production.
	EnterConstinterval(c *ConstintervalContext)

	// EnterOpt_timezone is called when entering the opt_timezone production.
	EnterOpt_timezone(c *Opt_timezoneContext)

	// EnterOpt_interval is called when entering the opt_interval production.
	EnterOpt_interval(c *Opt_intervalContext)

	// EnterInterval_second is called when entering the interval_second production.
	EnterInterval_second(c *Interval_secondContext)

	// EnterOpt_escape is called when entering the opt_escape production.
	EnterOpt_escape(c *Opt_escapeContext)

	// EnterA_expr is called when entering the a_expr production.
	EnterA_expr(c *A_exprContext)

	// EnterA_expr_qual is called when entering the a_expr_qual production.
	EnterA_expr_qual(c *A_expr_qualContext)

	// EnterA_expr_lessless is called when entering the a_expr_lessless production.
	EnterA_expr_lessless(c *A_expr_lesslessContext)

	// EnterA_expr_or is called when entering the a_expr_or production.
	EnterA_expr_or(c *A_expr_orContext)

	// EnterA_expr_and is called when entering the a_expr_and production.
	EnterA_expr_and(c *A_expr_andContext)

	// EnterA_expr_between is called when entering the a_expr_between production.
	EnterA_expr_between(c *A_expr_betweenContext)

	// EnterA_expr_in is called when entering the a_expr_in production.
	EnterA_expr_in(c *A_expr_inContext)

	// EnterA_expr_unary_not is called when entering the a_expr_unary_not production.
	EnterA_expr_unary_not(c *A_expr_unary_notContext)

	// EnterA_expr_isnull is called when entering the a_expr_isnull production.
	EnterA_expr_isnull(c *A_expr_isnullContext)

	// EnterA_expr_is_not is called when entering the a_expr_is_not production.
	EnterA_expr_is_not(c *A_expr_is_notContext)

	// EnterA_expr_compare is called when entering the a_expr_compare production.
	EnterA_expr_compare(c *A_expr_compareContext)

	// EnterA_expr_like is called when entering the a_expr_like production.
	EnterA_expr_like(c *A_expr_likeContext)

	// EnterA_expr_qual_op is called when entering the a_expr_qual_op production.
	EnterA_expr_qual_op(c *A_expr_qual_opContext)

	// EnterA_expr_unary_qualop is called when entering the a_expr_unary_qualop production.
	EnterA_expr_unary_qualop(c *A_expr_unary_qualopContext)

	// EnterA_expr_add is called when entering the a_expr_add production.
	EnterA_expr_add(c *A_expr_addContext)

	// EnterA_expr_mul is called when entering the a_expr_mul production.
	EnterA_expr_mul(c *A_expr_mulContext)

	// EnterA_expr_caret is called when entering the a_expr_caret production.
	EnterA_expr_caret(c *A_expr_caretContext)

	// EnterA_expr_unary_sign is called when entering the a_expr_unary_sign production.
	EnterA_expr_unary_sign(c *A_expr_unary_signContext)

	// EnterA_expr_at_time_zone is called when entering the a_expr_at_time_zone production.
	EnterA_expr_at_time_zone(c *A_expr_at_time_zoneContext)

	// EnterA_expr_collate is called when entering the a_expr_collate production.
	EnterA_expr_collate(c *A_expr_collateContext)

	// EnterA_expr_typecast is called when entering the a_expr_typecast production.
	EnterA_expr_typecast(c *A_expr_typecastContext)

	// EnterB_expr is called when entering the b_expr production.
	EnterB_expr(c *B_exprContext)

	// EnterC_expr_exists is called when entering the c_expr_exists production.
	EnterC_expr_exists(c *C_expr_existsContext)

	// EnterC_expr_expr is called when entering the c_expr_expr production.
	EnterC_expr_expr(c *C_expr_exprContext)

	// EnterC_expr_case is called when entering the c_expr_case production.
	EnterC_expr_case(c *C_expr_caseContext)

	// EnterPlsqlvariablename is called when entering the plsqlvariablename production.
	EnterPlsqlvariablename(c *PlsqlvariablenameContext)

	// EnterFunc_application is called when entering the func_application production.
	EnterFunc_application(c *Func_applicationContext)

	// EnterFunc_expr is called when entering the func_expr production.
	EnterFunc_expr(c *Func_exprContext)

	// EnterFunc_expr_windowless is called when entering the func_expr_windowless production.
	EnterFunc_expr_windowless(c *Func_expr_windowlessContext)

	// EnterFunc_expr_common_subexpr is called when entering the func_expr_common_subexpr production.
	EnterFunc_expr_common_subexpr(c *Func_expr_common_subexprContext)

	// EnterXml_root_version is called when entering the xml_root_version production.
	EnterXml_root_version(c *Xml_root_versionContext)

	// EnterOpt_xml_root_standalone is called when entering the opt_xml_root_standalone production.
	EnterOpt_xml_root_standalone(c *Opt_xml_root_standaloneContext)

	// EnterXml_attributes is called when entering the xml_attributes production.
	EnterXml_attributes(c *Xml_attributesContext)

	// EnterXml_attribute_list is called when entering the xml_attribute_list production.
	EnterXml_attribute_list(c *Xml_attribute_listContext)

	// EnterXml_attribute_el is called when entering the xml_attribute_el production.
	EnterXml_attribute_el(c *Xml_attribute_elContext)

	// EnterDocument_or_content is called when entering the document_or_content production.
	EnterDocument_or_content(c *Document_or_contentContext)

	// EnterXml_whitespace_option is called when entering the xml_whitespace_option production.
	EnterXml_whitespace_option(c *Xml_whitespace_optionContext)

	// EnterXmlexists_argument is called when entering the xmlexists_argument production.
	EnterXmlexists_argument(c *Xmlexists_argumentContext)

	// EnterXml_passing_mech is called when entering the xml_passing_mech production.
	EnterXml_passing_mech(c *Xml_passing_mechContext)

	// EnterWithin_group_clause is called when entering the within_group_clause production.
	EnterWithin_group_clause(c *Within_group_clauseContext)

	// EnterFilter_clause is called when entering the filter_clause production.
	EnterFilter_clause(c *Filter_clauseContext)

	// EnterWindow_clause is called when entering the window_clause production.
	EnterWindow_clause(c *Window_clauseContext)

	// EnterWindow_definition_list is called when entering the window_definition_list production.
	EnterWindow_definition_list(c *Window_definition_listContext)

	// EnterWindow_definition is called when entering the window_definition production.
	EnterWindow_definition(c *Window_definitionContext)

	// EnterOver_clause is called when entering the over_clause production.
	EnterOver_clause(c *Over_clauseContext)

	// EnterWindow_specification is called when entering the window_specification production.
	EnterWindow_specification(c *Window_specificationContext)

	// EnterOpt_existing_window_name is called when entering the opt_existing_window_name production.
	EnterOpt_existing_window_name(c *Opt_existing_window_nameContext)

	// EnterOpt_partition_clause is called when entering the opt_partition_clause production.
	EnterOpt_partition_clause(c *Opt_partition_clauseContext)

	// EnterOpt_frame_clause is called when entering the opt_frame_clause production.
	EnterOpt_frame_clause(c *Opt_frame_clauseContext)

	// EnterFrame_extent is called when entering the frame_extent production.
	EnterFrame_extent(c *Frame_extentContext)

	// EnterFrame_bound is called when entering the frame_bound production.
	EnterFrame_bound(c *Frame_boundContext)

	// EnterOpt_window_exclusion_clause is called when entering the opt_window_exclusion_clause production.
	EnterOpt_window_exclusion_clause(c *Opt_window_exclusion_clauseContext)

	// EnterRow is called when entering the row production.
	EnterRow(c *RowContext)

	// EnterExplicit_row is called when entering the explicit_row production.
	EnterExplicit_row(c *Explicit_rowContext)

	// EnterImplicit_row is called when entering the implicit_row production.
	EnterImplicit_row(c *Implicit_rowContext)

	// EnterSub_type is called when entering the sub_type production.
	EnterSub_type(c *Sub_typeContext)

	// EnterAll_op is called when entering the all_op production.
	EnterAll_op(c *All_opContext)

	// EnterMathop is called when entering the mathop production.
	EnterMathop(c *MathopContext)

	// EnterQual_op is called when entering the qual_op production.
	EnterQual_op(c *Qual_opContext)

	// EnterQual_all_op is called when entering the qual_all_op production.
	EnterQual_all_op(c *Qual_all_opContext)

	// EnterSubquery_Op is called when entering the subquery_Op production.
	EnterSubquery_Op(c *Subquery_OpContext)

	// EnterExpr_list is called when entering the expr_list production.
	EnterExpr_list(c *Expr_listContext)

	// EnterFunc_arg_list is called when entering the func_arg_list production.
	EnterFunc_arg_list(c *Func_arg_listContext)

	// EnterFunc_arg_expr is called when entering the func_arg_expr production.
	EnterFunc_arg_expr(c *Func_arg_exprContext)

	// EnterType_list is called when entering the type_list production.
	EnterType_list(c *Type_listContext)

	// EnterArray_expr is called when entering the array_expr production.
	EnterArray_expr(c *Array_exprContext)

	// EnterArray_expr_list is called when entering the array_expr_list production.
	EnterArray_expr_list(c *Array_expr_listContext)

	// EnterExtract_list is called when entering the extract_list production.
	EnterExtract_list(c *Extract_listContext)

	// EnterExtract_arg is called when entering the extract_arg production.
	EnterExtract_arg(c *Extract_argContext)

	// EnterUnicode_normal_form is called when entering the unicode_normal_form production.
	EnterUnicode_normal_form(c *Unicode_normal_formContext)

	// EnterOverlay_list is called when entering the overlay_list production.
	EnterOverlay_list(c *Overlay_listContext)

	// EnterPosition_list is called when entering the position_list production.
	EnterPosition_list(c *Position_listContext)

	// EnterSubstr_list is called when entering the substr_list production.
	EnterSubstr_list(c *Substr_listContext)

	// EnterTrim_list is called when entering the trim_list production.
	EnterTrim_list(c *Trim_listContext)

	// EnterIn_expr_select is called when entering the in_expr_select production.
	EnterIn_expr_select(c *In_expr_selectContext)

	// EnterIn_expr_list is called when entering the in_expr_list production.
	EnterIn_expr_list(c *In_expr_listContext)

	// EnterCase_expr is called when entering the case_expr production.
	EnterCase_expr(c *Case_exprContext)

	// EnterWhen_clause_list is called when entering the when_clause_list production.
	EnterWhen_clause_list(c *When_clause_listContext)

	// EnterWhen_clause is called when entering the when_clause production.
	EnterWhen_clause(c *When_clauseContext)

	// EnterCase_default is called when entering the case_default production.
	EnterCase_default(c *Case_defaultContext)

	// EnterCase_arg is called when entering the case_arg production.
	EnterCase_arg(c *Case_argContext)

	// EnterColumnref is called when entering the columnref production.
	EnterColumnref(c *ColumnrefContext)

	// EnterIndirection_el is called when entering the indirection_el production.
	EnterIndirection_el(c *Indirection_elContext)

	// EnterOpt_slice_bound is called when entering the opt_slice_bound production.
	EnterOpt_slice_bound(c *Opt_slice_boundContext)

	// EnterIndirection is called when entering the indirection production.
	EnterIndirection(c *IndirectionContext)

	// EnterOpt_indirection is called when entering the opt_indirection production.
	EnterOpt_indirection(c *Opt_indirectionContext)

	// EnterOpt_target_list is called when entering the opt_target_list production.
	EnterOpt_target_list(c *Opt_target_listContext)

	// EnterTarget_list is called when entering the target_list production.
	EnterTarget_list(c *Target_listContext)

	// EnterTarget_label is called when entering the target_label production.
	EnterTarget_label(c *Target_labelContext)

	// EnterTarget_star is called when entering the target_star production.
	EnterTarget_star(c *Target_starContext)

	// EnterQualified_name_list is called when entering the qualified_name_list production.
	EnterQualified_name_list(c *Qualified_name_listContext)

	// EnterQualified_name is called when entering the qualified_name production.
	EnterQualified_name(c *Qualified_nameContext)

	// EnterName_list is called when entering the name_list production.
	EnterName_list(c *Name_listContext)

	// EnterName is called when entering the name production.
	EnterName(c *NameContext)

	// EnterAttr_name is called when entering the attr_name production.
	EnterAttr_name(c *Attr_nameContext)

	// EnterFile_name is called when entering the file_name production.
	EnterFile_name(c *File_nameContext)

	// EnterFunc_name is called when entering the func_name production.
	EnterFunc_name(c *Func_nameContext)

	// EnterAexprconst is called when entering the aexprconst production.
	EnterAexprconst(c *AexprconstContext)

	// EnterXconst is called when entering the xconst production.
	EnterXconst(c *XconstContext)

	// EnterBconst is called when entering the bconst production.
	EnterBconst(c *BconstContext)

	// EnterFconst is called when entering the fconst production.
	EnterFconst(c *FconstContext)

	// EnterIconst is called when entering the iconst production.
	EnterIconst(c *IconstContext)

	// EnterSconst is called when entering the sconst production.
	EnterSconst(c *SconstContext)

	// EnterAnysconst is called when entering the anysconst production.
	EnterAnysconst(c *AnysconstContext)

	// EnterOpt_uescape is called when entering the opt_uescape production.
	EnterOpt_uescape(c *Opt_uescapeContext)

	// EnterSignediconst is called when entering the signediconst production.
	EnterSignediconst(c *SignediconstContext)

	// EnterRoleid is called when entering the roleid production.
	EnterRoleid(c *RoleidContext)

	// EnterRolespec is called when entering the rolespec production.
	EnterRolespec(c *RolespecContext)

	// EnterRole_list is called when entering the role_list production.
	EnterRole_list(c *Role_listContext)

	// EnterColid is called when entering the colid production.
	EnterColid(c *ColidContext)

	// EnterTable_alias is called when entering the table_alias production.
	EnterTable_alias(c *Table_aliasContext)

	// EnterType_function_name is called when entering the type_function_name production.
	EnterType_function_name(c *Type_function_nameContext)

	// EnterNonreservedword is called when entering the nonreservedword production.
	EnterNonreservedword(c *NonreservedwordContext)

	// EnterCollabel is called when entering the collabel production.
	EnterCollabel(c *CollabelContext)

	// EnterIdentifier is called when entering the identifier production.
	EnterIdentifier(c *IdentifierContext)

	// EnterPlsqlidentifier is called when entering the plsqlidentifier production.
	EnterPlsqlidentifier(c *PlsqlidentifierContext)

	// EnterUnreserved_keyword is called when entering the unreserved_keyword production.
	EnterUnreserved_keyword(c *Unreserved_keywordContext)

	// EnterCol_name_keyword is called when entering the col_name_keyword production.
	EnterCol_name_keyword(c *Col_name_keywordContext)

	// EnterType_func_name_keyword is called when entering the type_func_name_keyword production.
	EnterType_func_name_keyword(c *Type_func_name_keywordContext)

	// EnterReserved_keyword is called when entering the reserved_keyword production.
	EnterReserved_keyword(c *Reserved_keywordContext)

	// EnterBuiltin_function_name is called when entering the builtin_function_name production.
	EnterBuiltin_function_name(c *Builtin_function_nameContext)

	// EnterPl_function is called when entering the pl_function production.
	EnterPl_function(c *Pl_functionContext)

	// EnterComp_options is called when entering the comp_options production.
	EnterComp_options(c *Comp_optionsContext)

	// EnterComp_option is called when entering the comp_option production.
	EnterComp_option(c *Comp_optionContext)

	// EnterSharp is called when entering the sharp production.
	EnterSharp(c *SharpContext)

	// EnterOption_value is called when entering the option_value production.
	EnterOption_value(c *Option_valueContext)

	// EnterOpt_semi is called when entering the opt_semi production.
	EnterOpt_semi(c *Opt_semiContext)

	// EnterPl_block is called when entering the pl_block production.
	EnterPl_block(c *Pl_blockContext)

	// EnterDecl_sect is called when entering the decl_sect production.
	EnterDecl_sect(c *Decl_sectContext)

	// EnterDecl_start is called when entering the decl_start production.
	EnterDecl_start(c *Decl_startContext)

	// EnterDecl_stmts is called when entering the decl_stmts production.
	EnterDecl_stmts(c *Decl_stmtsContext)

	// EnterLabel_decl is called when entering the label_decl production.
	EnterLabel_decl(c *Label_declContext)

	// EnterDecl_stmt is called when entering the decl_stmt production.
	EnterDecl_stmt(c *Decl_stmtContext)

	// EnterDecl_statement is called when entering the decl_statement production.
	EnterDecl_statement(c *Decl_statementContext)

	// EnterOpt_scrollable is called when entering the opt_scrollable production.
	EnterOpt_scrollable(c *Opt_scrollableContext)

	// EnterDecl_cursor_query is called when entering the decl_cursor_query production.
	EnterDecl_cursor_query(c *Decl_cursor_queryContext)

	// EnterDecl_cursor_args is called when entering the decl_cursor_args production.
	EnterDecl_cursor_args(c *Decl_cursor_argsContext)

	// EnterDecl_cursor_arglist is called when entering the decl_cursor_arglist production.
	EnterDecl_cursor_arglist(c *Decl_cursor_arglistContext)

	// EnterDecl_cursor_arg is called when entering the decl_cursor_arg production.
	EnterDecl_cursor_arg(c *Decl_cursor_argContext)

	// EnterDecl_is_for is called when entering the decl_is_for production.
	EnterDecl_is_for(c *Decl_is_forContext)

	// EnterDecl_aliasitem is called when entering the decl_aliasitem production.
	EnterDecl_aliasitem(c *Decl_aliasitemContext)

	// EnterDecl_varname is called when entering the decl_varname production.
	EnterDecl_varname(c *Decl_varnameContext)

	// EnterDecl_const is called when entering the decl_const production.
	EnterDecl_const(c *Decl_constContext)

	// EnterDecl_datatype is called when entering the decl_datatype production.
	EnterDecl_datatype(c *Decl_datatypeContext)

	// EnterDecl_collate is called when entering the decl_collate production.
	EnterDecl_collate(c *Decl_collateContext)

	// EnterDecl_notnull is called when entering the decl_notnull production.
	EnterDecl_notnull(c *Decl_notnullContext)

	// EnterDecl_defval is called when entering the decl_defval production.
	EnterDecl_defval(c *Decl_defvalContext)

	// EnterDecl_defkey is called when entering the decl_defkey production.
	EnterDecl_defkey(c *Decl_defkeyContext)

	// EnterAssign_operator is called when entering the assign_operator production.
	EnterAssign_operator(c *Assign_operatorContext)

	// EnterProc_sect is called when entering the proc_sect production.
	EnterProc_sect(c *Proc_sectContext)

	// EnterProc_stmt is called when entering the proc_stmt production.
	EnterProc_stmt(c *Proc_stmtContext)

	// EnterStmt_perform is called when entering the stmt_perform production.
	EnterStmt_perform(c *Stmt_performContext)

	// EnterStmt_call is called when entering the stmt_call production.
	EnterStmt_call(c *Stmt_callContext)

	// EnterOpt_expr_list is called when entering the opt_expr_list production.
	EnterOpt_expr_list(c *Opt_expr_listContext)

	// EnterStmt_assign is called when entering the stmt_assign production.
	EnterStmt_assign(c *Stmt_assignContext)

	// EnterStmt_getdiag is called when entering the stmt_getdiag production.
	EnterStmt_getdiag(c *Stmt_getdiagContext)

	// EnterGetdiag_area_opt is called when entering the getdiag_area_opt production.
	EnterGetdiag_area_opt(c *Getdiag_area_optContext)

	// EnterGetdiag_list is called when entering the getdiag_list production.
	EnterGetdiag_list(c *Getdiag_listContext)

	// EnterGetdiag_list_item is called when entering the getdiag_list_item production.
	EnterGetdiag_list_item(c *Getdiag_list_itemContext)

	// EnterGetdiag_item is called when entering the getdiag_item production.
	EnterGetdiag_item(c *Getdiag_itemContext)

	// EnterGetdiag_target is called when entering the getdiag_target production.
	EnterGetdiag_target(c *Getdiag_targetContext)

	// EnterAssign_var is called when entering the assign_var production.
	EnterAssign_var(c *Assign_varContext)

	// EnterStmt_if is called when entering the stmt_if production.
	EnterStmt_if(c *Stmt_ifContext)

	// EnterStmt_elsifs is called when entering the stmt_elsifs production.
	EnterStmt_elsifs(c *Stmt_elsifsContext)

	// EnterStmt_else is called when entering the stmt_else production.
	EnterStmt_else(c *Stmt_elseContext)

	// EnterStmt_case is called when entering the stmt_case production.
	EnterStmt_case(c *Stmt_caseContext)

	// EnterOpt_expr_until_when is called when entering the opt_expr_until_when production.
	EnterOpt_expr_until_when(c *Opt_expr_until_whenContext)

	// EnterCase_when_list is called when entering the case_when_list production.
	EnterCase_when_list(c *Case_when_listContext)

	// EnterCase_when is called when entering the case_when production.
	EnterCase_when(c *Case_whenContext)

	// EnterOpt_case_else is called when entering the opt_case_else production.
	EnterOpt_case_else(c *Opt_case_elseContext)

	// EnterStmt_loop is called when entering the stmt_loop production.
	EnterStmt_loop(c *Stmt_loopContext)

	// EnterStmt_while is called when entering the stmt_while production.
	EnterStmt_while(c *Stmt_whileContext)

	// EnterStmt_for is called when entering the stmt_for production.
	EnterStmt_for(c *Stmt_forContext)

	// EnterFor_control is called when entering the for_control production.
	EnterFor_control(c *For_controlContext)

	// EnterOpt_for_using_expression is called when entering the opt_for_using_expression production.
	EnterOpt_for_using_expression(c *Opt_for_using_expressionContext)

	// EnterOpt_cursor_parameters is called when entering the opt_cursor_parameters production.
	EnterOpt_cursor_parameters(c *Opt_cursor_parametersContext)

	// EnterOpt_reverse is called when entering the opt_reverse production.
	EnterOpt_reverse(c *Opt_reverseContext)

	// EnterOpt_by_expression is called when entering the opt_by_expression production.
	EnterOpt_by_expression(c *Opt_by_expressionContext)

	// EnterFor_variable is called when entering the for_variable production.
	EnterFor_variable(c *For_variableContext)

	// EnterStmt_foreach_a is called when entering the stmt_foreach_a production.
	EnterStmt_foreach_a(c *Stmt_foreach_aContext)

	// EnterForeach_slice is called when entering the foreach_slice production.
	EnterForeach_slice(c *Foreach_sliceContext)

	// EnterStmt_exit is called when entering the stmt_exit production.
	EnterStmt_exit(c *Stmt_exitContext)

	// EnterExit_type is called when entering the exit_type production.
	EnterExit_type(c *Exit_typeContext)

	// EnterStmt_return is called when entering the stmt_return production.
	EnterStmt_return(c *Stmt_returnContext)

	// EnterOpt_return_result is called when entering the opt_return_result production.
	EnterOpt_return_result(c *Opt_return_resultContext)

	// EnterStmt_raise is called when entering the stmt_raise production.
	EnterStmt_raise(c *Stmt_raiseContext)

	// EnterOpt_stmt_raise_level is called when entering the opt_stmt_raise_level production.
	EnterOpt_stmt_raise_level(c *Opt_stmt_raise_levelContext)

	// EnterOpt_raise_list is called when entering the opt_raise_list production.
	EnterOpt_raise_list(c *Opt_raise_listContext)

	// EnterOpt_raise_using is called when entering the opt_raise_using production.
	EnterOpt_raise_using(c *Opt_raise_usingContext)

	// EnterOpt_raise_using_elem is called when entering the opt_raise_using_elem production.
	EnterOpt_raise_using_elem(c *Opt_raise_using_elemContext)

	// EnterOpt_raise_using_elem_list is called when entering the opt_raise_using_elem_list production.
	EnterOpt_raise_using_elem_list(c *Opt_raise_using_elem_listContext)

	// EnterStmt_assert is called when entering the stmt_assert production.
	EnterStmt_assert(c *Stmt_assertContext)

	// EnterOpt_stmt_assert_message is called when entering the opt_stmt_assert_message production.
	EnterOpt_stmt_assert_message(c *Opt_stmt_assert_messageContext)

	// EnterLoop_body is called when entering the loop_body production.
	EnterLoop_body(c *Loop_bodyContext)

	// EnterStmt_execsql is called when entering the stmt_execsql production.
	EnterStmt_execsql(c *Stmt_execsqlContext)

	// EnterStmt_dynexecute is called when entering the stmt_dynexecute production.
	EnterStmt_dynexecute(c *Stmt_dynexecuteContext)

	// EnterOpt_execute_using is called when entering the opt_execute_using production.
	EnterOpt_execute_using(c *Opt_execute_usingContext)

	// EnterOpt_execute_using_list is called when entering the opt_execute_using_list production.
	EnterOpt_execute_using_list(c *Opt_execute_using_listContext)

	// EnterOpt_execute_into is called when entering the opt_execute_into production.
	EnterOpt_execute_into(c *Opt_execute_intoContext)

	// EnterStmt_open is called when entering the stmt_open production.
	EnterStmt_open(c *Stmt_openContext)

	// EnterOpt_open_bound_list_item is called when entering the opt_open_bound_list_item production.
	EnterOpt_open_bound_list_item(c *Opt_open_bound_list_itemContext)

	// EnterOpt_open_bound_list is called when entering the opt_open_bound_list production.
	EnterOpt_open_bound_list(c *Opt_open_bound_listContext)

	// EnterOpt_open_using is called when entering the opt_open_using production.
	EnterOpt_open_using(c *Opt_open_usingContext)

	// EnterOpt_scroll_option is called when entering the opt_scroll_option production.
	EnterOpt_scroll_option(c *Opt_scroll_optionContext)

	// EnterOpt_scroll_option_no is called when entering the opt_scroll_option_no production.
	EnterOpt_scroll_option_no(c *Opt_scroll_option_noContext)

	// EnterStmt_fetch is called when entering the stmt_fetch production.
	EnterStmt_fetch(c *Stmt_fetchContext)

	// EnterInto_target is called when entering the into_target production.
	EnterInto_target(c *Into_targetContext)

	// EnterOpt_cursor_from is called when entering the opt_cursor_from production.
	EnterOpt_cursor_from(c *Opt_cursor_fromContext)

	// EnterOpt_fetch_direction is called when entering the opt_fetch_direction production.
	EnterOpt_fetch_direction(c *Opt_fetch_directionContext)

	// EnterStmt_move is called when entering the stmt_move production.
	EnterStmt_move(c *Stmt_moveContext)

	// EnterStmt_close is called when entering the stmt_close production.
	EnterStmt_close(c *Stmt_closeContext)

	// EnterStmt_null is called when entering the stmt_null production.
	EnterStmt_null(c *Stmt_nullContext)

	// EnterStmt_commit is called when entering the stmt_commit production.
	EnterStmt_commit(c *Stmt_commitContext)

	// EnterStmt_rollback is called when entering the stmt_rollback production.
	EnterStmt_rollback(c *Stmt_rollbackContext)

	// EnterPlsql_opt_transaction_chain is called when entering the plsql_opt_transaction_chain production.
	EnterPlsql_opt_transaction_chain(c *Plsql_opt_transaction_chainContext)

	// EnterStmt_set is called when entering the stmt_set production.
	EnterStmt_set(c *Stmt_setContext)

	// EnterCursor_variable is called when entering the cursor_variable production.
	EnterCursor_variable(c *Cursor_variableContext)

	// EnterException_sect is called when entering the exception_sect production.
	EnterException_sect(c *Exception_sectContext)

	// EnterProc_exceptions is called when entering the proc_exceptions production.
	EnterProc_exceptions(c *Proc_exceptionsContext)

	// EnterProc_exception is called when entering the proc_exception production.
	EnterProc_exception(c *Proc_exceptionContext)

	// EnterProc_conditions is called when entering the proc_conditions production.
	EnterProc_conditions(c *Proc_conditionsContext)

	// EnterProc_condition is called when entering the proc_condition production.
	EnterProc_condition(c *Proc_conditionContext)

	// EnterOpt_block_label is called when entering the opt_block_label production.
	EnterOpt_block_label(c *Opt_block_labelContext)

	// EnterOpt_loop_label is called when entering the opt_loop_label production.
	EnterOpt_loop_label(c *Opt_loop_labelContext)

	// EnterOpt_label is called when entering the opt_label production.
	EnterOpt_label(c *Opt_labelContext)

	// EnterOpt_exitcond is called when entering the opt_exitcond production.
	EnterOpt_exitcond(c *Opt_exitcondContext)

	// EnterAny_identifier is called when entering the any_identifier production.
	EnterAny_identifier(c *Any_identifierContext)

	// EnterPlsql_unreserved_keyword is called when entering the plsql_unreserved_keyword production.
	EnterPlsql_unreserved_keyword(c *Plsql_unreserved_keywordContext)

	// EnterSql_expression is called when entering the sql_expression production.
	EnterSql_expression(c *Sql_expressionContext)

	// EnterExpr_until_then is called when entering the expr_until_then production.
	EnterExpr_until_then(c *Expr_until_thenContext)

	// EnterExpr_until_semi is called when entering the expr_until_semi production.
	EnterExpr_until_semi(c *Expr_until_semiContext)

	// EnterExpr_until_rightbracket is called when entering the expr_until_rightbracket production.
	EnterExpr_until_rightbracket(c *Expr_until_rightbracketContext)

	// EnterExpr_until_loop is called when entering the expr_until_loop production.
	EnterExpr_until_loop(c *Expr_until_loopContext)

	// EnterMake_execsql_stmt is called when entering the make_execsql_stmt production.
	EnterMake_execsql_stmt(c *Make_execsql_stmtContext)

	// EnterOpt_returning_clause_into is called when entering the opt_returning_clause_into production.
	EnterOpt_returning_clause_into(c *Opt_returning_clause_intoContext)

	// ExitRoot is called when exiting the root production.
	ExitRoot(c *RootContext)

	// ExitPlsqlroot is called when exiting the plsqlroot production.
	ExitPlsqlroot(c *PlsqlrootContext)

	// ExitStmtblock is called when exiting the stmtblock production.
	ExitStmtblock(c *StmtblockContext)

	// ExitStmtmulti is called when exiting the stmtmulti production.
	ExitStmtmulti(c *StmtmultiContext)

	// ExitStmt is called when exiting the stmt production.
	ExitStmt(c *StmtContext)

	// ExitPlsqlconsolecommand is called when exiting the plsqlconsolecommand production.
	ExitPlsqlconsolecommand(c *PlsqlconsolecommandContext)

	// ExitCallstmt is called when exiting the callstmt production.
	ExitCallstmt(c *CallstmtContext)

	// ExitCreaterolestmt is called when exiting the createrolestmt production.
	ExitCreaterolestmt(c *CreaterolestmtContext)

	// ExitOpt_with is called when exiting the opt_with production.
	ExitOpt_with(c *Opt_withContext)

	// ExitOptrolelist is called when exiting the optrolelist production.
	ExitOptrolelist(c *OptrolelistContext)

	// ExitAlteroptrolelist is called when exiting the alteroptrolelist production.
	ExitAlteroptrolelist(c *AlteroptrolelistContext)

	// ExitAlteroptroleelem is called when exiting the alteroptroleelem production.
	ExitAlteroptroleelem(c *AlteroptroleelemContext)

	// ExitCreateoptroleelem is called when exiting the createoptroleelem production.
	ExitCreateoptroleelem(c *CreateoptroleelemContext)

	// ExitCreateuserstmt is called when exiting the createuserstmt production.
	ExitCreateuserstmt(c *CreateuserstmtContext)

	// ExitAlterrolestmt is called when exiting the alterrolestmt production.
	ExitAlterrolestmt(c *AlterrolestmtContext)

	// ExitOpt_in_database is called when exiting the opt_in_database production.
	ExitOpt_in_database(c *Opt_in_databaseContext)

	// ExitAlterrolesetstmt is called when exiting the alterrolesetstmt production.
	ExitAlterrolesetstmt(c *AlterrolesetstmtContext)

	// ExitDroprolestmt is called when exiting the droprolestmt production.
	ExitDroprolestmt(c *DroprolestmtContext)

	// ExitCreategroupstmt is called when exiting the creategroupstmt production.
	ExitCreategroupstmt(c *CreategroupstmtContext)

	// ExitAltergroupstmt is called when exiting the altergroupstmt production.
	ExitAltergroupstmt(c *AltergroupstmtContext)

	// ExitAdd_drop is called when exiting the add_drop production.
	ExitAdd_drop(c *Add_dropContext)

	// ExitCreateschemastmt is called when exiting the createschemastmt production.
	ExitCreateschemastmt(c *CreateschemastmtContext)

	// ExitOptschemaname is called when exiting the optschemaname production.
	ExitOptschemaname(c *OptschemanameContext)

	// ExitOptschemaeltlist is called when exiting the optschemaeltlist production.
	ExitOptschemaeltlist(c *OptschemaeltlistContext)

	// ExitSchema_stmt is called when exiting the schema_stmt production.
	ExitSchema_stmt(c *Schema_stmtContext)

	// ExitVariablesetstmt is called when exiting the variablesetstmt production.
	ExitVariablesetstmt(c *VariablesetstmtContext)

	// ExitSet_rest is called when exiting the set_rest production.
	ExitSet_rest(c *Set_restContext)

	// ExitGeneric_set is called when exiting the generic_set production.
	ExitGeneric_set(c *Generic_setContext)

	// ExitSet_rest_more is called when exiting the set_rest_more production.
	ExitSet_rest_more(c *Set_rest_moreContext)

	// ExitVar_name is called when exiting the var_name production.
	ExitVar_name(c *Var_nameContext)

	// ExitVar_list is called when exiting the var_list production.
	ExitVar_list(c *Var_listContext)

	// ExitVar_value is called when exiting the var_value production.
	ExitVar_value(c *Var_valueContext)

	// ExitIso_level is called when exiting the iso_level production.
	ExitIso_level(c *Iso_levelContext)

	// ExitOpt_boolean_or_string is called when exiting the opt_boolean_or_string production.
	ExitOpt_boolean_or_string(c *Opt_boolean_or_stringContext)

	// ExitZone_value is called when exiting the zone_value production.
	ExitZone_value(c *Zone_valueContext)

	// ExitOpt_encoding is called when exiting the opt_encoding production.
	ExitOpt_encoding(c *Opt_encodingContext)

	// ExitNonreservedword_or_sconst is called when exiting the nonreservedword_or_sconst production.
	ExitNonreservedword_or_sconst(c *Nonreservedword_or_sconstContext)

	// ExitVariableresetstmt is called when exiting the variableresetstmt production.
	ExitVariableresetstmt(c *VariableresetstmtContext)

	// ExitReset_rest is called when exiting the reset_rest production.
	ExitReset_rest(c *Reset_restContext)

	// ExitGeneric_reset is called when exiting the generic_reset production.
	ExitGeneric_reset(c *Generic_resetContext)

	// ExitSetresetclause is called when exiting the setresetclause production.
	ExitSetresetclause(c *SetresetclauseContext)

	// ExitFunctionsetresetclause is called when exiting the functionsetresetclause production.
	ExitFunctionsetresetclause(c *FunctionsetresetclauseContext)

	// ExitVariableshowstmt is called when exiting the variableshowstmt production.
	ExitVariableshowstmt(c *VariableshowstmtContext)

	// ExitConstraintssetstmt is called when exiting the constraintssetstmt production.
	ExitConstraintssetstmt(c *ConstraintssetstmtContext)

	// ExitConstraints_set_list is called when exiting the constraints_set_list production.
	ExitConstraints_set_list(c *Constraints_set_listContext)

	// ExitConstraints_set_mode is called when exiting the constraints_set_mode production.
	ExitConstraints_set_mode(c *Constraints_set_modeContext)

	// ExitCheckpointstmt is called when exiting the checkpointstmt production.
	ExitCheckpointstmt(c *CheckpointstmtContext)

	// ExitDiscardstmt is called when exiting the discardstmt production.
	ExitDiscardstmt(c *DiscardstmtContext)

	// ExitAltertablestmt is called when exiting the altertablestmt production.
	ExitAltertablestmt(c *AltertablestmtContext)

	// ExitAlter_table_cmds is called when exiting the alter_table_cmds production.
	ExitAlter_table_cmds(c *Alter_table_cmdsContext)

	// ExitPartition_cmd is called when exiting the partition_cmd production.
	ExitPartition_cmd(c *Partition_cmdContext)

	// ExitIndex_partition_cmd is called when exiting the index_partition_cmd production.
	ExitIndex_partition_cmd(c *Index_partition_cmdContext)

	// ExitAlter_table_cmd is called when exiting the alter_table_cmd production.
	ExitAlter_table_cmd(c *Alter_table_cmdContext)

	// ExitAlter_column_default is called when exiting the alter_column_default production.
	ExitAlter_column_default(c *Alter_column_defaultContext)

	// ExitOpt_drop_behavior is called when exiting the opt_drop_behavior production.
	ExitOpt_drop_behavior(c *Opt_drop_behaviorContext)

	// ExitOpt_collate_clause is called when exiting the opt_collate_clause production.
	ExitOpt_collate_clause(c *Opt_collate_clauseContext)

	// ExitAlter_using is called when exiting the alter_using production.
	ExitAlter_using(c *Alter_usingContext)

	// ExitReplica_identity is called when exiting the replica_identity production.
	ExitReplica_identity(c *Replica_identityContext)

	// ExitReloptions is called when exiting the reloptions production.
	ExitReloptions(c *ReloptionsContext)

	// ExitOpt_reloptions is called when exiting the opt_reloptions production.
	ExitOpt_reloptions(c *Opt_reloptionsContext)

	// ExitReloption_list is called when exiting the reloption_list production.
	ExitReloption_list(c *Reloption_listContext)

	// ExitReloption_elem is called when exiting the reloption_elem production.
	ExitReloption_elem(c *Reloption_elemContext)

	// ExitAlter_identity_column_option_list is called when exiting the alter_identity_column_option_list production.
	ExitAlter_identity_column_option_list(c *Alter_identity_column_option_listContext)

	// ExitAlter_identity_column_option is called when exiting the alter_identity_column_option production.
	ExitAlter_identity_column_option(c *Alter_identity_column_optionContext)

	// ExitPartitionboundspec is called when exiting the partitionboundspec production.
	ExitPartitionboundspec(c *PartitionboundspecContext)

	// ExitHash_partbound_elem is called when exiting the hash_partbound_elem production.
	ExitHash_partbound_elem(c *Hash_partbound_elemContext)

	// ExitHash_partbound is called when exiting the hash_partbound production.
	ExitHash_partbound(c *Hash_partboundContext)

	// ExitAltercompositetypestmt is called when exiting the altercompositetypestmt production.
	ExitAltercompositetypestmt(c *AltercompositetypestmtContext)

	// ExitAlter_type_cmds is called when exiting the alter_type_cmds production.
	ExitAlter_type_cmds(c *Alter_type_cmdsContext)

	// ExitAlter_type_cmd is called when exiting the alter_type_cmd production.
	ExitAlter_type_cmd(c *Alter_type_cmdContext)

	// ExitCloseportalstmt is called when exiting the closeportalstmt production.
	ExitCloseportalstmt(c *CloseportalstmtContext)

	// ExitCopystmt is called when exiting the copystmt production.
	ExitCopystmt(c *CopystmtContext)

	// ExitCopy_from is called when exiting the copy_from production.
	ExitCopy_from(c *Copy_fromContext)

	// ExitOpt_program is called when exiting the opt_program production.
	ExitOpt_program(c *Opt_programContext)

	// ExitCopy_file_name is called when exiting the copy_file_name production.
	ExitCopy_file_name(c *Copy_file_nameContext)

	// ExitCopy_options is called when exiting the copy_options production.
	ExitCopy_options(c *Copy_optionsContext)

	// ExitCopy_opt_list is called when exiting the copy_opt_list production.
	ExitCopy_opt_list(c *Copy_opt_listContext)

	// ExitCopy_opt_item is called when exiting the copy_opt_item production.
	ExitCopy_opt_item(c *Copy_opt_itemContext)

	// ExitOpt_binary is called when exiting the opt_binary production.
	ExitOpt_binary(c *Opt_binaryContext)

	// ExitCopy_delimiter is called when exiting the copy_delimiter production.
	ExitCopy_delimiter(c *Copy_delimiterContext)

	// ExitOpt_using is called when exiting the opt_using production.
	ExitOpt_using(c *Opt_usingContext)

	// ExitCopy_generic_opt_list is called when exiting the copy_generic_opt_list production.
	ExitCopy_generic_opt_list(c *Copy_generic_opt_listContext)

	// ExitCopy_generic_opt_elem is called when exiting the copy_generic_opt_elem production.
	ExitCopy_generic_opt_elem(c *Copy_generic_opt_elemContext)

	// ExitCopy_generic_opt_arg is called when exiting the copy_generic_opt_arg production.
	ExitCopy_generic_opt_arg(c *Copy_generic_opt_argContext)

	// ExitCopy_generic_opt_arg_list is called when exiting the copy_generic_opt_arg_list production.
	ExitCopy_generic_opt_arg_list(c *Copy_generic_opt_arg_listContext)

	// ExitCopy_generic_opt_arg_list_item is called when exiting the copy_generic_opt_arg_list_item production.
	ExitCopy_generic_opt_arg_list_item(c *Copy_generic_opt_arg_list_itemContext)

	// ExitCreatestmt is called when exiting the createstmt production.
	ExitCreatestmt(c *CreatestmtContext)

	// ExitOpttemp is called when exiting the opttemp production.
	ExitOpttemp(c *OpttempContext)

	// ExitOpttableelementlist is called when exiting the opttableelementlist production.
	ExitOpttableelementlist(c *OpttableelementlistContext)

	// ExitOpttypedtableelementlist is called when exiting the opttypedtableelementlist production.
	ExitOpttypedtableelementlist(c *OpttypedtableelementlistContext)

	// ExitTableelementlist is called when exiting the tableelementlist production.
	ExitTableelementlist(c *TableelementlistContext)

	// ExitTypedtableelementlist is called when exiting the typedtableelementlist production.
	ExitTypedtableelementlist(c *TypedtableelementlistContext)

	// ExitTableelement is called when exiting the tableelement production.
	ExitTableelement(c *TableelementContext)

	// ExitTypedtableelement is called when exiting the typedtableelement production.
	ExitTypedtableelement(c *TypedtableelementContext)

	// ExitColumnDef is called when exiting the columnDef production.
	ExitColumnDef(c *ColumnDefContext)

	// ExitColumnOptions is called when exiting the columnOptions production.
	ExitColumnOptions(c *ColumnOptionsContext)

	// ExitColquallist is called when exiting the colquallist production.
	ExitColquallist(c *ColquallistContext)

	// ExitColconstraint is called when exiting the colconstraint production.
	ExitColconstraint(c *ColconstraintContext)

	// ExitColconstraintelem is called when exiting the colconstraintelem production.
	ExitColconstraintelem(c *ColconstraintelemContext)

	// ExitGenerated_when is called when exiting the generated_when production.
	ExitGenerated_when(c *Generated_whenContext)

	// ExitConstraintattr is called when exiting the constraintattr production.
	ExitConstraintattr(c *ConstraintattrContext)

	// ExitTablelikeclause is called when exiting the tablelikeclause production.
	ExitTablelikeclause(c *TablelikeclauseContext)

	// ExitTablelikeoptionlist is called when exiting the tablelikeoptionlist production.
	ExitTablelikeoptionlist(c *TablelikeoptionlistContext)

	// ExitTablelikeoption is called when exiting the tablelikeoption production.
	ExitTablelikeoption(c *TablelikeoptionContext)

	// ExitTableconstraint is called when exiting the tableconstraint production.
	ExitTableconstraint(c *TableconstraintContext)

	// ExitConstraintelem is called when exiting the constraintelem production.
	ExitConstraintelem(c *ConstraintelemContext)

	// ExitOpt_no_inherit is called when exiting the opt_no_inherit production.
	ExitOpt_no_inherit(c *Opt_no_inheritContext)

	// ExitOpt_column_list is called when exiting the opt_column_list production.
	ExitOpt_column_list(c *Opt_column_listContext)

	// ExitColumnlist is called when exiting the columnlist production.
	ExitColumnlist(c *ColumnlistContext)

	// ExitColumnElem is called when exiting the columnElem production.
	ExitColumnElem(c *ColumnElemContext)

	// ExitOpt_c_include is called when exiting the opt_c_include production.
	ExitOpt_c_include(c *Opt_c_includeContext)

	// ExitKey_match is called when exiting the key_match production.
	ExitKey_match(c *Key_matchContext)

	// ExitExclusionconstraintlist is called when exiting the exclusionconstraintlist production.
	ExitExclusionconstraintlist(c *ExclusionconstraintlistContext)

	// ExitExclusionconstraintelem is called when exiting the exclusionconstraintelem production.
	ExitExclusionconstraintelem(c *ExclusionconstraintelemContext)

	// ExitExclusionwhereclause is called when exiting the exclusionwhereclause production.
	ExitExclusionwhereclause(c *ExclusionwhereclauseContext)

	// ExitKey_actions is called when exiting the key_actions production.
	ExitKey_actions(c *Key_actionsContext)

	// ExitKey_update is called when exiting the key_update production.
	ExitKey_update(c *Key_updateContext)

	// ExitKey_delete is called when exiting the key_delete production.
	ExitKey_delete(c *Key_deleteContext)

	// ExitKey_action is called when exiting the key_action production.
	ExitKey_action(c *Key_actionContext)

	// ExitOptinherit is called when exiting the optinherit production.
	ExitOptinherit(c *OptinheritContext)

	// ExitOptpartitionspec is called when exiting the optpartitionspec production.
	ExitOptpartitionspec(c *OptpartitionspecContext)

	// ExitPartitionspec is called when exiting the partitionspec production.
	ExitPartitionspec(c *PartitionspecContext)

	// ExitPart_params is called when exiting the part_params production.
	ExitPart_params(c *Part_paramsContext)

	// ExitPart_elem is called when exiting the part_elem production.
	ExitPart_elem(c *Part_elemContext)

	// ExitTable_access_method_clause is called when exiting the table_access_method_clause production.
	ExitTable_access_method_clause(c *Table_access_method_clauseContext)

	// ExitOptwith is called when exiting the optwith production.
	ExitOptwith(c *OptwithContext)

	// ExitOncommitoption is called when exiting the oncommitoption production.
	ExitOncommitoption(c *OncommitoptionContext)

	// ExitOpttablespace is called when exiting the opttablespace production.
	ExitOpttablespace(c *OpttablespaceContext)

	// ExitOptconstablespace is called when exiting the optconstablespace production.
	ExitOptconstablespace(c *OptconstablespaceContext)

	// ExitExistingindex is called when exiting the existingindex production.
	ExitExistingindex(c *ExistingindexContext)

	// ExitCreatestatsstmt is called when exiting the createstatsstmt production.
	ExitCreatestatsstmt(c *CreatestatsstmtContext)

	// ExitAlterstatsstmt is called when exiting the alterstatsstmt production.
	ExitAlterstatsstmt(c *AlterstatsstmtContext)

	// ExitCreateasstmt is called when exiting the createasstmt production.
	ExitCreateasstmt(c *CreateasstmtContext)

	// ExitCreate_as_target is called when exiting the create_as_target production.
	ExitCreate_as_target(c *Create_as_targetContext)

	// ExitOpt_with_data is called when exiting the opt_with_data production.
	ExitOpt_with_data(c *Opt_with_dataContext)

	// ExitCreatematviewstmt is called when exiting the creatematviewstmt production.
	ExitCreatematviewstmt(c *CreatematviewstmtContext)

	// ExitCreate_mv_target is called when exiting the create_mv_target production.
	ExitCreate_mv_target(c *Create_mv_targetContext)

	// ExitOptnolog is called when exiting the optnolog production.
	ExitOptnolog(c *OptnologContext)

	// ExitRefreshmatviewstmt is called when exiting the refreshmatviewstmt production.
	ExitRefreshmatviewstmt(c *RefreshmatviewstmtContext)

	// ExitCreateseqstmt is called when exiting the createseqstmt production.
	ExitCreateseqstmt(c *CreateseqstmtContext)

	// ExitAlterseqstmt is called when exiting the alterseqstmt production.
	ExitAlterseqstmt(c *AlterseqstmtContext)

	// ExitOptseqoptlist is called when exiting the optseqoptlist production.
	ExitOptseqoptlist(c *OptseqoptlistContext)

	// ExitOptparenthesizedseqoptlist is called when exiting the optparenthesizedseqoptlist production.
	ExitOptparenthesizedseqoptlist(c *OptparenthesizedseqoptlistContext)

	// ExitSeqoptlist is called when exiting the seqoptlist production.
	ExitSeqoptlist(c *SeqoptlistContext)

	// ExitSeqoptelem is called when exiting the seqoptelem production.
	ExitSeqoptelem(c *SeqoptelemContext)

	// ExitOpt_by is called when exiting the opt_by production.
	ExitOpt_by(c *Opt_byContext)

	// ExitNumericonly is called when exiting the numericonly production.
	ExitNumericonly(c *NumericonlyContext)

	// ExitNumericonly_list is called when exiting the numericonly_list production.
	ExitNumericonly_list(c *Numericonly_listContext)

	// ExitCreateplangstmt is called when exiting the createplangstmt production.
	ExitCreateplangstmt(c *CreateplangstmtContext)

	// ExitOpt_trusted is called when exiting the opt_trusted production.
	ExitOpt_trusted(c *Opt_trustedContext)

	// ExitHandler_name is called when exiting the handler_name production.
	ExitHandler_name(c *Handler_nameContext)

	// ExitOpt_inline_handler is called when exiting the opt_inline_handler production.
	ExitOpt_inline_handler(c *Opt_inline_handlerContext)

	// ExitValidator_clause is called when exiting the validator_clause production.
	ExitValidator_clause(c *Validator_clauseContext)

	// ExitOpt_validator is called when exiting the opt_validator production.
	ExitOpt_validator(c *Opt_validatorContext)

	// ExitOpt_procedural is called when exiting the opt_procedural production.
	ExitOpt_procedural(c *Opt_proceduralContext)

	// ExitCreatetablespacestmt is called when exiting the createtablespacestmt production.
	ExitCreatetablespacestmt(c *CreatetablespacestmtContext)

	// ExitOpttablespaceowner is called when exiting the opttablespaceowner production.
	ExitOpttablespaceowner(c *OpttablespaceownerContext)

	// ExitDroptablespacestmt is called when exiting the droptablespacestmt production.
	ExitDroptablespacestmt(c *DroptablespacestmtContext)

	// ExitCreateextensionstmt is called when exiting the createextensionstmt production.
	ExitCreateextensionstmt(c *CreateextensionstmtContext)

	// ExitCreate_extension_opt_list is called when exiting the create_extension_opt_list production.
	ExitCreate_extension_opt_list(c *Create_extension_opt_listContext)

	// ExitCreate_extension_opt_item is called when exiting the create_extension_opt_item production.
	ExitCreate_extension_opt_item(c *Create_extension_opt_itemContext)

	// ExitAlterextensionstmt is called when exiting the alterextensionstmt production.
	ExitAlterextensionstmt(c *AlterextensionstmtContext)

	// ExitAlter_extension_opt_list is called when exiting the alter_extension_opt_list production.
	ExitAlter_extension_opt_list(c *Alter_extension_opt_listContext)

	// ExitAlter_extension_opt_item is called when exiting the alter_extension_opt_item production.
	ExitAlter_extension_opt_item(c *Alter_extension_opt_itemContext)

	// ExitAlterextensioncontentsstmt is called when exiting the alterextensioncontentsstmt production.
	ExitAlterextensioncontentsstmt(c *AlterextensioncontentsstmtContext)

	// ExitCreatefdwstmt is called when exiting the createfdwstmt production.
	ExitCreatefdwstmt(c *CreatefdwstmtContext)

	// ExitFdw_option is called when exiting the fdw_option production.
	ExitFdw_option(c *Fdw_optionContext)

	// ExitFdw_options is called when exiting the fdw_options production.
	ExitFdw_options(c *Fdw_optionsContext)

	// ExitOpt_fdw_options is called when exiting the opt_fdw_options production.
	ExitOpt_fdw_options(c *Opt_fdw_optionsContext)

	// ExitAlterfdwstmt is called when exiting the alterfdwstmt production.
	ExitAlterfdwstmt(c *AlterfdwstmtContext)

	// ExitCreate_generic_options is called when exiting the create_generic_options production.
	ExitCreate_generic_options(c *Create_generic_optionsContext)

	// ExitGeneric_option_list is called when exiting the generic_option_list production.
	ExitGeneric_option_list(c *Generic_option_listContext)

	// ExitAlter_generic_options is called when exiting the alter_generic_options production.
	ExitAlter_generic_options(c *Alter_generic_optionsContext)

	// ExitAlter_generic_option_list is called when exiting the alter_generic_option_list production.
	ExitAlter_generic_option_list(c *Alter_generic_option_listContext)

	// ExitAlter_generic_option_elem is called when exiting the alter_generic_option_elem production.
	ExitAlter_generic_option_elem(c *Alter_generic_option_elemContext)

	// ExitGeneric_option_elem is called when exiting the generic_option_elem production.
	ExitGeneric_option_elem(c *Generic_option_elemContext)

	// ExitGeneric_option_name is called when exiting the generic_option_name production.
	ExitGeneric_option_name(c *Generic_option_nameContext)

	// ExitGeneric_option_arg is called when exiting the generic_option_arg production.
	ExitGeneric_option_arg(c *Generic_option_argContext)

	// ExitCreateforeignserverstmt is called when exiting the createforeignserverstmt production.
	ExitCreateforeignserverstmt(c *CreateforeignserverstmtContext)

	// ExitOpt_type is called when exiting the opt_type production.
	ExitOpt_type(c *Opt_typeContext)

	// ExitForeign_server_version is called when exiting the foreign_server_version production.
	ExitForeign_server_version(c *Foreign_server_versionContext)

	// ExitOpt_foreign_server_version is called when exiting the opt_foreign_server_version production.
	ExitOpt_foreign_server_version(c *Opt_foreign_server_versionContext)

	// ExitAlterforeignserverstmt is called when exiting the alterforeignserverstmt production.
	ExitAlterforeignserverstmt(c *AlterforeignserverstmtContext)

	// ExitCreateforeigntablestmt is called when exiting the createforeigntablestmt production.
	ExitCreateforeigntablestmt(c *CreateforeigntablestmtContext)

	// ExitImportforeignschemastmt is called when exiting the importforeignschemastmt production.
	ExitImportforeignschemastmt(c *ImportforeignschemastmtContext)

	// ExitImport_qualification_type is called when exiting the import_qualification_type production.
	ExitImport_qualification_type(c *Import_qualification_typeContext)

	// ExitImport_qualification is called when exiting the import_qualification production.
	ExitImport_qualification(c *Import_qualificationContext)

	// ExitCreateusermappingstmt is called when exiting the createusermappingstmt production.
	ExitCreateusermappingstmt(c *CreateusermappingstmtContext)

	// ExitAuth_ident is called when exiting the auth_ident production.
	ExitAuth_ident(c *Auth_identContext)

	// ExitDropusermappingstmt is called when exiting the dropusermappingstmt production.
	ExitDropusermappingstmt(c *DropusermappingstmtContext)

	// ExitAlterusermappingstmt is called when exiting the alterusermappingstmt production.
	ExitAlterusermappingstmt(c *AlterusermappingstmtContext)

	// ExitCreatepolicystmt is called when exiting the createpolicystmt production.
	ExitCreatepolicystmt(c *CreatepolicystmtContext)

	// ExitAlterpolicystmt is called when exiting the alterpolicystmt production.
	ExitAlterpolicystmt(c *AlterpolicystmtContext)

	// ExitRowsecurityoptionalexpr is called when exiting the rowsecurityoptionalexpr production.
	ExitRowsecurityoptionalexpr(c *RowsecurityoptionalexprContext)

	// ExitRowsecurityoptionalwithcheck is called when exiting the rowsecurityoptionalwithcheck production.
	ExitRowsecurityoptionalwithcheck(c *RowsecurityoptionalwithcheckContext)

	// ExitRowsecuritydefaulttorole is called when exiting the rowsecuritydefaulttorole production.
	ExitRowsecuritydefaulttorole(c *RowsecuritydefaulttoroleContext)

	// ExitRowsecurityoptionaltorole is called when exiting the rowsecurityoptionaltorole production.
	ExitRowsecurityoptionaltorole(c *RowsecurityoptionaltoroleContext)

	// ExitRowsecuritydefaultpermissive is called when exiting the rowsecuritydefaultpermissive production.
	ExitRowsecuritydefaultpermissive(c *RowsecuritydefaultpermissiveContext)

	// ExitRowsecuritydefaultforcmd is called when exiting the rowsecuritydefaultforcmd production.
	ExitRowsecuritydefaultforcmd(c *RowsecuritydefaultforcmdContext)

	// ExitRow_security_cmd is called when exiting the row_security_cmd production.
	ExitRow_security_cmd(c *Row_security_cmdContext)

	// ExitCreateamstmt is called when exiting the createamstmt production.
	ExitCreateamstmt(c *CreateamstmtContext)

	// ExitAm_type is called when exiting the am_type production.
	ExitAm_type(c *Am_typeContext)

	// ExitCreatetrigstmt is called when exiting the createtrigstmt production.
	ExitCreatetrigstmt(c *CreatetrigstmtContext)

	// ExitTriggeractiontime is called when exiting the triggeractiontime production.
	ExitTriggeractiontime(c *TriggeractiontimeContext)

	// ExitTriggerevents is called when exiting the triggerevents production.
	ExitTriggerevents(c *TriggereventsContext)

	// ExitTriggeroneevent is called when exiting the triggeroneevent production.
	ExitTriggeroneevent(c *TriggeroneeventContext)

	// ExitTriggerreferencing is called when exiting the triggerreferencing production.
	ExitTriggerreferencing(c *TriggerreferencingContext)

	// ExitTriggertransitions is called when exiting the triggertransitions production.
	ExitTriggertransitions(c *TriggertransitionsContext)

	// ExitTriggertransition is called when exiting the triggertransition production.
	ExitTriggertransition(c *TriggertransitionContext)

	// ExitTransitionoldornew is called when exiting the transitionoldornew production.
	ExitTransitionoldornew(c *TransitionoldornewContext)

	// ExitTransitionrowortable is called when exiting the transitionrowortable production.
	ExitTransitionrowortable(c *TransitionrowortableContext)

	// ExitTransitionrelname is called when exiting the transitionrelname production.
	ExitTransitionrelname(c *TransitionrelnameContext)

	// ExitTriggerforspec is called when exiting the triggerforspec production.
	ExitTriggerforspec(c *TriggerforspecContext)

	// ExitTriggerforopteach is called when exiting the triggerforopteach production.
	ExitTriggerforopteach(c *TriggerforopteachContext)

	// ExitTriggerfortype is called when exiting the triggerfortype production.
	ExitTriggerfortype(c *TriggerfortypeContext)

	// ExitTriggerwhen is called when exiting the triggerwhen production.
	ExitTriggerwhen(c *TriggerwhenContext)

	// ExitFunction_or_procedure is called when exiting the function_or_procedure production.
	ExitFunction_or_procedure(c *Function_or_procedureContext)

	// ExitTriggerfuncargs is called when exiting the triggerfuncargs production.
	ExitTriggerfuncargs(c *TriggerfuncargsContext)

	// ExitTriggerfuncarg is called when exiting the triggerfuncarg production.
	ExitTriggerfuncarg(c *TriggerfuncargContext)

	// ExitOptconstrfromtable is called when exiting the optconstrfromtable production.
	ExitOptconstrfromtable(c *OptconstrfromtableContext)

	// ExitConstraintattributespec is called when exiting the constraintattributespec production.
	ExitConstraintattributespec(c *ConstraintattributespecContext)

	// ExitConstraintattributeElem is called when exiting the constraintattributeElem production.
	ExitConstraintattributeElem(c *ConstraintattributeElemContext)

	// ExitCreateeventtrigstmt is called when exiting the createeventtrigstmt production.
	ExitCreateeventtrigstmt(c *CreateeventtrigstmtContext)

	// ExitEvent_trigger_when_list is called when exiting the event_trigger_when_list production.
	ExitEvent_trigger_when_list(c *Event_trigger_when_listContext)

	// ExitEvent_trigger_when_item is called when exiting the event_trigger_when_item production.
	ExitEvent_trigger_when_item(c *Event_trigger_when_itemContext)

	// ExitEvent_trigger_value_list is called when exiting the event_trigger_value_list production.
	ExitEvent_trigger_value_list(c *Event_trigger_value_listContext)

	// ExitAltereventtrigstmt is called when exiting the altereventtrigstmt production.
	ExitAltereventtrigstmt(c *AltereventtrigstmtContext)

	// ExitEnable_trigger is called when exiting the enable_trigger production.
	ExitEnable_trigger(c *Enable_triggerContext)

	// ExitCreateassertionstmt is called when exiting the createassertionstmt production.
	ExitCreateassertionstmt(c *CreateassertionstmtContext)

	// ExitDefinestmt is called when exiting the definestmt production.
	ExitDefinestmt(c *DefinestmtContext)

	// ExitDefinition is called when exiting the definition production.
	ExitDefinition(c *DefinitionContext)

	// ExitDef_list is called when exiting the def_list production.
	ExitDef_list(c *Def_listContext)

	// ExitDef_elem is called when exiting the def_elem production.
	ExitDef_elem(c *Def_elemContext)

	// ExitDef_arg is called when exiting the def_arg production.
	ExitDef_arg(c *Def_argContext)

	// ExitOld_aggr_definition is called when exiting the old_aggr_definition production.
	ExitOld_aggr_definition(c *Old_aggr_definitionContext)

	// ExitOld_aggr_list is called when exiting the old_aggr_list production.
	ExitOld_aggr_list(c *Old_aggr_listContext)

	// ExitOld_aggr_elem is called when exiting the old_aggr_elem production.
	ExitOld_aggr_elem(c *Old_aggr_elemContext)

	// ExitOpt_enum_val_list is called when exiting the opt_enum_val_list production.
	ExitOpt_enum_val_list(c *Opt_enum_val_listContext)

	// ExitEnum_val_list is called when exiting the enum_val_list production.
	ExitEnum_val_list(c *Enum_val_listContext)

	// ExitAlterenumstmt is called when exiting the alterenumstmt production.
	ExitAlterenumstmt(c *AlterenumstmtContext)

	// ExitOpt_if_not_exists is called when exiting the opt_if_not_exists production.
	ExitOpt_if_not_exists(c *Opt_if_not_existsContext)

	// ExitCreateopclassstmt is called when exiting the createopclassstmt production.
	ExitCreateopclassstmt(c *CreateopclassstmtContext)

	// ExitOpclass_item_list is called when exiting the opclass_item_list production.
	ExitOpclass_item_list(c *Opclass_item_listContext)

	// ExitOpclass_item is called when exiting the opclass_item production.
	ExitOpclass_item(c *Opclass_itemContext)

	// ExitOpt_default is called when exiting the opt_default production.
	ExitOpt_default(c *Opt_defaultContext)

	// ExitOpt_opfamily is called when exiting the opt_opfamily production.
	ExitOpt_opfamily(c *Opt_opfamilyContext)

	// ExitOpclass_purpose is called when exiting the opclass_purpose production.
	ExitOpclass_purpose(c *Opclass_purposeContext)

	// ExitOpt_recheck is called when exiting the opt_recheck production.
	ExitOpt_recheck(c *Opt_recheckContext)

	// ExitCreateopfamilystmt is called when exiting the createopfamilystmt production.
	ExitCreateopfamilystmt(c *CreateopfamilystmtContext)

	// ExitAlteropfamilystmt is called when exiting the alteropfamilystmt production.
	ExitAlteropfamilystmt(c *AlteropfamilystmtContext)

	// ExitOpclass_drop_list is called when exiting the opclass_drop_list production.
	ExitOpclass_drop_list(c *Opclass_drop_listContext)

	// ExitOpclass_drop is called when exiting the opclass_drop production.
	ExitOpclass_drop(c *Opclass_dropContext)

	// ExitDropopclassstmt is called when exiting the dropopclassstmt production.
	ExitDropopclassstmt(c *DropopclassstmtContext)

	// ExitDropopfamilystmt is called when exiting the dropopfamilystmt production.
	ExitDropopfamilystmt(c *DropopfamilystmtContext)

	// ExitDropownedstmt is called when exiting the dropownedstmt production.
	ExitDropownedstmt(c *DropownedstmtContext)

	// ExitReassignownedstmt is called when exiting the reassignownedstmt production.
	ExitReassignownedstmt(c *ReassignownedstmtContext)

	// ExitDropstmt is called when exiting the dropstmt production.
	ExitDropstmt(c *DropstmtContext)

	// ExitObject_type_any_name is called when exiting the object_type_any_name production.
	ExitObject_type_any_name(c *Object_type_any_nameContext)

	// ExitObject_type_name is called when exiting the object_type_name production.
	ExitObject_type_name(c *Object_type_nameContext)

	// ExitDrop_type_name is called when exiting the drop_type_name production.
	ExitDrop_type_name(c *Drop_type_nameContext)

	// ExitObject_type_name_on_any_name is called when exiting the object_type_name_on_any_name production.
	ExitObject_type_name_on_any_name(c *Object_type_name_on_any_nameContext)

	// ExitAny_name_list is called when exiting the any_name_list production.
	ExitAny_name_list(c *Any_name_listContext)

	// ExitAny_name is called when exiting the any_name production.
	ExitAny_name(c *Any_nameContext)

	// ExitAttrs is called when exiting the attrs production.
	ExitAttrs(c *AttrsContext)

	// ExitType_name_list is called when exiting the type_name_list production.
	ExitType_name_list(c *Type_name_listContext)

	// ExitTruncatestmt is called when exiting the truncatestmt production.
	ExitTruncatestmt(c *TruncatestmtContext)

	// ExitOpt_restart_seqs is called when exiting the opt_restart_seqs production.
	ExitOpt_restart_seqs(c *Opt_restart_seqsContext)

	// ExitCommentstmt is called when exiting the commentstmt production.
	ExitCommentstmt(c *CommentstmtContext)

	// ExitComment_text is called when exiting the comment_text production.
	ExitComment_text(c *Comment_textContext)

	// ExitSeclabelstmt is called when exiting the seclabelstmt production.
	ExitSeclabelstmt(c *SeclabelstmtContext)

	// ExitOpt_provider is called when exiting the opt_provider production.
	ExitOpt_provider(c *Opt_providerContext)

	// ExitSecurity_label is called when exiting the security_label production.
	ExitSecurity_label(c *Security_labelContext)

	// ExitFetchstmt is called when exiting the fetchstmt production.
	ExitFetchstmt(c *FetchstmtContext)

	// ExitFetch_args is called when exiting the fetch_args production.
	ExitFetch_args(c *Fetch_argsContext)

	// ExitFrom_in is called when exiting the from_in production.
	ExitFrom_in(c *From_inContext)

	// ExitOpt_from_in is called when exiting the opt_from_in production.
	ExitOpt_from_in(c *Opt_from_inContext)

	// ExitGrantstmt is called when exiting the grantstmt production.
	ExitGrantstmt(c *GrantstmtContext)

	// ExitRevokestmt is called when exiting the revokestmt production.
	ExitRevokestmt(c *RevokestmtContext)

	// ExitPrivileges is called when exiting the privileges production.
	ExitPrivileges(c *PrivilegesContext)

	// ExitPrivilege_list is called when exiting the privilege_list production.
	ExitPrivilege_list(c *Privilege_listContext)

	// ExitPrivilege is called when exiting the privilege production.
	ExitPrivilege(c *PrivilegeContext)

	// ExitPrivilege_target is called when exiting the privilege_target production.
	ExitPrivilege_target(c *Privilege_targetContext)

	// ExitGrantee_list is called when exiting the grantee_list production.
	ExitGrantee_list(c *Grantee_listContext)

	// ExitGrantee is called when exiting the grantee production.
	ExitGrantee(c *GranteeContext)

	// ExitOpt_grant_grant_option is called when exiting the opt_grant_grant_option production.
	ExitOpt_grant_grant_option(c *Opt_grant_grant_optionContext)

	// ExitGrantrolestmt is called when exiting the grantrolestmt production.
	ExitGrantrolestmt(c *GrantrolestmtContext)

	// ExitRevokerolestmt is called when exiting the revokerolestmt production.
	ExitRevokerolestmt(c *RevokerolestmtContext)

	// ExitOpt_grant_admin_option is called when exiting the opt_grant_admin_option production.
	ExitOpt_grant_admin_option(c *Opt_grant_admin_optionContext)

	// ExitOpt_granted_by is called when exiting the opt_granted_by production.
	ExitOpt_granted_by(c *Opt_granted_byContext)

	// ExitAlterdefaultprivilegesstmt is called when exiting the alterdefaultprivilegesstmt production.
	ExitAlterdefaultprivilegesstmt(c *AlterdefaultprivilegesstmtContext)

	// ExitDefacloptionlist is called when exiting the defacloptionlist production.
	ExitDefacloptionlist(c *DefacloptionlistContext)

	// ExitDefacloption is called when exiting the defacloption production.
	ExitDefacloption(c *DefacloptionContext)

	// ExitDefaclaction is called when exiting the defaclaction production.
	ExitDefaclaction(c *DefaclactionContext)

	// ExitDefacl_privilege_target is called when exiting the defacl_privilege_target production.
	ExitDefacl_privilege_target(c *Defacl_privilege_targetContext)

	// ExitIndexstmt is called when exiting the indexstmt production.
	ExitIndexstmt(c *IndexstmtContext)

	// ExitOpt_unique is called when exiting the opt_unique production.
	ExitOpt_unique(c *Opt_uniqueContext)

	// ExitOpt_concurrently is called when exiting the opt_concurrently production.
	ExitOpt_concurrently(c *Opt_concurrentlyContext)

	// ExitOpt_index_name is called when exiting the opt_index_name production.
	ExitOpt_index_name(c *Opt_index_nameContext)

	// ExitAccess_method_clause is called when exiting the access_method_clause production.
	ExitAccess_method_clause(c *Access_method_clauseContext)

	// ExitIndex_params is called when exiting the index_params production.
	ExitIndex_params(c *Index_paramsContext)

	// ExitIndex_elem_options is called when exiting the index_elem_options production.
	ExitIndex_elem_options(c *Index_elem_optionsContext)

	// ExitIndex_elem is called when exiting the index_elem production.
	ExitIndex_elem(c *Index_elemContext)

	// ExitOpt_include is called when exiting the opt_include production.
	ExitOpt_include(c *Opt_includeContext)

	// ExitIndex_including_params is called when exiting the index_including_params production.
	ExitIndex_including_params(c *Index_including_paramsContext)

	// ExitOpt_collate is called when exiting the opt_collate production.
	ExitOpt_collate(c *Opt_collateContext)

	// ExitOpt_class is called when exiting the opt_class production.
	ExitOpt_class(c *Opt_classContext)

	// ExitOpt_asc_desc is called when exiting the opt_asc_desc production.
	ExitOpt_asc_desc(c *Opt_asc_descContext)

	// ExitOpt_nulls_order is called when exiting the opt_nulls_order production.
	ExitOpt_nulls_order(c *Opt_nulls_orderContext)

	// ExitCreatefunctionstmt is called when exiting the createfunctionstmt production.
	ExitCreatefunctionstmt(c *CreatefunctionstmtContext)

	// ExitOpt_or_replace is called when exiting the opt_or_replace production.
	ExitOpt_or_replace(c *Opt_or_replaceContext)

	// ExitFunc_args is called when exiting the func_args production.
	ExitFunc_args(c *Func_argsContext)

	// ExitFunc_args_list is called when exiting the func_args_list production.
	ExitFunc_args_list(c *Func_args_listContext)

	// ExitFunction_with_argtypes_list is called when exiting the function_with_argtypes_list production.
	ExitFunction_with_argtypes_list(c *Function_with_argtypes_listContext)

	// ExitFunction_with_argtypes is called when exiting the function_with_argtypes production.
	ExitFunction_with_argtypes(c *Function_with_argtypesContext)

	// ExitFunc_args_with_defaults is called when exiting the func_args_with_defaults production.
	ExitFunc_args_with_defaults(c *Func_args_with_defaultsContext)

	// ExitFunc_args_with_defaults_list is called when exiting the func_args_with_defaults_list production.
	ExitFunc_args_with_defaults_list(c *Func_args_with_defaults_listContext)

	// ExitFunc_arg is called when exiting the func_arg production.
	ExitFunc_arg(c *Func_argContext)

	// ExitArg_class is called when exiting the arg_class production.
	ExitArg_class(c *Arg_classContext)

	// ExitParam_name is called when exiting the param_name production.
	ExitParam_name(c *Param_nameContext)

	// ExitFunc_return is called when exiting the func_return production.
	ExitFunc_return(c *Func_returnContext)

	// ExitFunc_type is called when exiting the func_type production.
	ExitFunc_type(c *Func_typeContext)

	// ExitFunc_arg_with_default is called when exiting the func_arg_with_default production.
	ExitFunc_arg_with_default(c *Func_arg_with_defaultContext)

	// ExitAggr_arg is called when exiting the aggr_arg production.
	ExitAggr_arg(c *Aggr_argContext)

	// ExitAggr_args is called when exiting the aggr_args production.
	ExitAggr_args(c *Aggr_argsContext)

	// ExitAggr_args_list is called when exiting the aggr_args_list production.
	ExitAggr_args_list(c *Aggr_args_listContext)

	// ExitAggregate_with_argtypes is called when exiting the aggregate_with_argtypes production.
	ExitAggregate_with_argtypes(c *Aggregate_with_argtypesContext)

	// ExitAggregate_with_argtypes_list is called when exiting the aggregate_with_argtypes_list production.
	ExitAggregate_with_argtypes_list(c *Aggregate_with_argtypes_listContext)

	// ExitCreatefunc_opt_list is called when exiting the createfunc_opt_list production.
	ExitCreatefunc_opt_list(c *Createfunc_opt_listContext)

	// ExitCommon_func_opt_item is called when exiting the common_func_opt_item production.
	ExitCommon_func_opt_item(c *Common_func_opt_itemContext)

	// ExitCreatefunc_opt_item is called when exiting the createfunc_opt_item production.
	ExitCreatefunc_opt_item(c *Createfunc_opt_itemContext)

	// ExitFunc_as is called when exiting the func_as production.
	ExitFunc_as(c *Func_asContext)

	// ExitTransform_type_list is called when exiting the transform_type_list production.
	ExitTransform_type_list(c *Transform_type_listContext)

	// ExitOpt_definition is called when exiting the opt_definition production.
	ExitOpt_definition(c *Opt_definitionContext)

	// ExitTable_func_column is called when exiting the table_func_column production.
	ExitTable_func_column(c *Table_func_columnContext)

	// ExitTable_func_column_list is called when exiting the table_func_column_list production.
	ExitTable_func_column_list(c *Table_func_column_listContext)

	// ExitAlterfunctionstmt is called when exiting the alterfunctionstmt production.
	ExitAlterfunctionstmt(c *AlterfunctionstmtContext)

	// ExitAlterfunc_opt_list is called when exiting the alterfunc_opt_list production.
	ExitAlterfunc_opt_list(c *Alterfunc_opt_listContext)

	// ExitOpt_restrict is called when exiting the opt_restrict production.
	ExitOpt_restrict(c *Opt_restrictContext)

	// ExitRemovefuncstmt is called when exiting the removefuncstmt production.
	ExitRemovefuncstmt(c *RemovefuncstmtContext)

	// ExitRemoveaggrstmt is called when exiting the removeaggrstmt production.
	ExitRemoveaggrstmt(c *RemoveaggrstmtContext)

	// ExitRemoveoperstmt is called when exiting the removeoperstmt production.
	ExitRemoveoperstmt(c *RemoveoperstmtContext)

	// ExitOper_argtypes is called when exiting the oper_argtypes production.
	ExitOper_argtypes(c *Oper_argtypesContext)

	// ExitAny_operator is called when exiting the any_operator production.
	ExitAny_operator(c *Any_operatorContext)

	// ExitOperator_with_argtypes_list is called when exiting the operator_with_argtypes_list production.
	ExitOperator_with_argtypes_list(c *Operator_with_argtypes_listContext)

	// ExitOperator_with_argtypes is called when exiting the operator_with_argtypes production.
	ExitOperator_with_argtypes(c *Operator_with_argtypesContext)

	// ExitDostmt is called when exiting the dostmt production.
	ExitDostmt(c *DostmtContext)

	// ExitDostmt_opt_list is called when exiting the dostmt_opt_list production.
	ExitDostmt_opt_list(c *Dostmt_opt_listContext)

	// ExitDostmt_opt_item is called when exiting the dostmt_opt_item production.
	ExitDostmt_opt_item(c *Dostmt_opt_itemContext)

	// ExitCreatecaststmt is called when exiting the createcaststmt production.
	ExitCreatecaststmt(c *CreatecaststmtContext)

	// ExitCast_context is called when exiting the cast_context production.
	ExitCast_context(c *Cast_contextContext)

	// ExitDropcaststmt is called when exiting the dropcaststmt production.
	ExitDropcaststmt(c *DropcaststmtContext)

	// ExitOpt_if_exists is called when exiting the opt_if_exists production.
	ExitOpt_if_exists(c *Opt_if_existsContext)

	// ExitCreatetransformstmt is called when exiting the createtransformstmt production.
	ExitCreatetransformstmt(c *CreatetransformstmtContext)

	// ExitTransform_element_list is called when exiting the transform_element_list production.
	ExitTransform_element_list(c *Transform_element_listContext)

	// ExitDroptransformstmt is called when exiting the droptransformstmt production.
	ExitDroptransformstmt(c *DroptransformstmtContext)

	// ExitReindexstmt is called when exiting the reindexstmt production.
	ExitReindexstmt(c *ReindexstmtContext)

	// ExitReindex_target_type is called when exiting the reindex_target_type production.
	ExitReindex_target_type(c *Reindex_target_typeContext)

	// ExitReindex_target_multitable is called when exiting the reindex_target_multitable production.
	ExitReindex_target_multitable(c *Reindex_target_multitableContext)

	// ExitReindex_option_list is called when exiting the reindex_option_list production.
	ExitReindex_option_list(c *Reindex_option_listContext)

	// ExitReindex_option_elem is called when exiting the reindex_option_elem production.
	ExitReindex_option_elem(c *Reindex_option_elemContext)

	// ExitAltertblspcstmt is called when exiting the altertblspcstmt production.
	ExitAltertblspcstmt(c *AltertblspcstmtContext)

	// ExitRenamestmt is called when exiting the renamestmt production.
	ExitRenamestmt(c *RenamestmtContext)

	// ExitOpt_column is called when exiting the opt_column production.
	ExitOpt_column(c *Opt_columnContext)

	// ExitOpt_set_data is called when exiting the opt_set_data production.
	ExitOpt_set_data(c *Opt_set_dataContext)

	// ExitAlterobjectdependsstmt is called when exiting the alterobjectdependsstmt production.
	ExitAlterobjectdependsstmt(c *AlterobjectdependsstmtContext)

	// ExitOpt_no is called when exiting the opt_no production.
	ExitOpt_no(c *Opt_noContext)

	// ExitAlterobjectschemastmt is called when exiting the alterobjectschemastmt production.
	ExitAlterobjectschemastmt(c *AlterobjectschemastmtContext)

	// ExitAlteroperatorstmt is called when exiting the alteroperatorstmt production.
	ExitAlteroperatorstmt(c *AlteroperatorstmtContext)

	// ExitOperator_def_list is called when exiting the operator_def_list production.
	ExitOperator_def_list(c *Operator_def_listContext)

	// ExitOperator_def_elem is called when exiting the operator_def_elem production.
	ExitOperator_def_elem(c *Operator_def_elemContext)

	// ExitOperator_def_arg is called when exiting the operator_def_arg production.
	ExitOperator_def_arg(c *Operator_def_argContext)

	// ExitAltertypestmt is called when exiting the altertypestmt production.
	ExitAltertypestmt(c *AltertypestmtContext)

	// ExitAlterownerstmt is called when exiting the alterownerstmt production.
	ExitAlterownerstmt(c *AlterownerstmtContext)

	// ExitCreatepublicationstmt is called when exiting the createpublicationstmt production.
	ExitCreatepublicationstmt(c *CreatepublicationstmtContext)

	// ExitOpt_publication_for_tables is called when exiting the opt_publication_for_tables production.
	ExitOpt_publication_for_tables(c *Opt_publication_for_tablesContext)

	// ExitPublication_for_tables is called when exiting the publication_for_tables production.
	ExitPublication_for_tables(c *Publication_for_tablesContext)

	// ExitAlterpublicationstmt is called when exiting the alterpublicationstmt production.
	ExitAlterpublicationstmt(c *AlterpublicationstmtContext)

	// ExitCreatesubscriptionstmt is called when exiting the createsubscriptionstmt production.
	ExitCreatesubscriptionstmt(c *CreatesubscriptionstmtContext)

	// ExitPublication_name_list is called when exiting the publication_name_list production.
	ExitPublication_name_list(c *Publication_name_listContext)

	// ExitPublication_name_item is called when exiting the publication_name_item production.
	ExitPublication_name_item(c *Publication_name_itemContext)

	// ExitAltersubscriptionstmt is called when exiting the altersubscriptionstmt production.
	ExitAltersubscriptionstmt(c *AltersubscriptionstmtContext)

	// ExitDropsubscriptionstmt is called when exiting the dropsubscriptionstmt production.
	ExitDropsubscriptionstmt(c *DropsubscriptionstmtContext)

	// ExitRulestmt is called when exiting the rulestmt production.
	ExitRulestmt(c *RulestmtContext)

	// ExitRuleactionlist is called when exiting the ruleactionlist production.
	ExitRuleactionlist(c *RuleactionlistContext)

	// ExitRuleactionmulti is called when exiting the ruleactionmulti production.
	ExitRuleactionmulti(c *RuleactionmultiContext)

	// ExitRuleactionstmt is called when exiting the ruleactionstmt production.
	ExitRuleactionstmt(c *RuleactionstmtContext)

	// ExitRuleactionstmtOrEmpty is called when exiting the ruleactionstmtOrEmpty production.
	ExitRuleactionstmtOrEmpty(c *RuleactionstmtOrEmptyContext)

	// ExitEvent is called when exiting the event production.
	ExitEvent(c *EventContext)

	// ExitOpt_instead is called when exiting the opt_instead production.
	ExitOpt_instead(c *Opt_insteadContext)

	// ExitNotifystmt is called when exiting the notifystmt production.
	ExitNotifystmt(c *NotifystmtContext)

	// ExitNotify_payload is called when exiting the notify_payload production.
	ExitNotify_payload(c *Notify_payloadContext)

	// ExitListenstmt is called when exiting the listenstmt production.
	ExitListenstmt(c *ListenstmtContext)

	// ExitUnlistenstmt is called when exiting the unlistenstmt production.
	ExitUnlistenstmt(c *UnlistenstmtContext)

	// ExitTransactionstmt is called when exiting the transactionstmt production.
	ExitTransactionstmt(c *TransactionstmtContext)

	// ExitOpt_transaction is called when exiting the opt_transaction production.
	ExitOpt_transaction(c *Opt_transactionContext)

	// ExitTransaction_mode_item is called when exiting the transaction_mode_item production.
	ExitTransaction_mode_item(c *Transaction_mode_itemContext)

	// ExitTransaction_mode_list is called when exiting the transaction_mode_list production.
	ExitTransaction_mode_list(c *Transaction_mode_listContext)

	// ExitTransaction_mode_list_or_empty is called when exiting the transaction_mode_list_or_empty production.
	ExitTransaction_mode_list_or_empty(c *Transaction_mode_list_or_emptyContext)

	// ExitOpt_transaction_chain is called when exiting the opt_transaction_chain production.
	ExitOpt_transaction_chain(c *Opt_transaction_chainContext)

	// ExitViewstmt is called when exiting the viewstmt production.
	ExitViewstmt(c *ViewstmtContext)

	// ExitOpt_check_option is called when exiting the opt_check_option production.
	ExitOpt_check_option(c *Opt_check_optionContext)

	// ExitLoadstmt is called when exiting the loadstmt production.
	ExitLoadstmt(c *LoadstmtContext)

	// ExitCreatedbstmt is called when exiting the createdbstmt production.
	ExitCreatedbstmt(c *CreatedbstmtContext)

	// ExitCreatedb_opt_list is called when exiting the createdb_opt_list production.
	ExitCreatedb_opt_list(c *Createdb_opt_listContext)

	// ExitCreatedb_opt_items is called when exiting the createdb_opt_items production.
	ExitCreatedb_opt_items(c *Createdb_opt_itemsContext)

	// ExitCreatedb_opt_item is called when exiting the createdb_opt_item production.
	ExitCreatedb_opt_item(c *Createdb_opt_itemContext)

	// ExitCreatedb_opt_name is called when exiting the createdb_opt_name production.
	ExitCreatedb_opt_name(c *Createdb_opt_nameContext)

	// ExitOpt_equal is called when exiting the opt_equal production.
	ExitOpt_equal(c *Opt_equalContext)

	// ExitAlterdatabasestmt is called when exiting the alterdatabasestmt production.
	ExitAlterdatabasestmt(c *AlterdatabasestmtContext)

	// ExitAlterdatabasesetstmt is called when exiting the alterdatabasesetstmt production.
	ExitAlterdatabasesetstmt(c *AlterdatabasesetstmtContext)

	// ExitDropdbstmt is called when exiting the dropdbstmt production.
	ExitDropdbstmt(c *DropdbstmtContext)

	// ExitDrop_option_list is called when exiting the drop_option_list production.
	ExitDrop_option_list(c *Drop_option_listContext)

	// ExitDrop_option is called when exiting the drop_option production.
	ExitDrop_option(c *Drop_optionContext)

	// ExitAltercollationstmt is called when exiting the altercollationstmt production.
	ExitAltercollationstmt(c *AltercollationstmtContext)

	// ExitAltersystemstmt is called when exiting the altersystemstmt production.
	ExitAltersystemstmt(c *AltersystemstmtContext)

	// ExitCreatedomainstmt is called when exiting the createdomainstmt production.
	ExitCreatedomainstmt(c *CreatedomainstmtContext)

	// ExitAlterdomainstmt is called when exiting the alterdomainstmt production.
	ExitAlterdomainstmt(c *AlterdomainstmtContext)

	// ExitOpt_as is called when exiting the opt_as production.
	ExitOpt_as(c *Opt_asContext)

	// ExitAltertsdictionarystmt is called when exiting the altertsdictionarystmt production.
	ExitAltertsdictionarystmt(c *AltertsdictionarystmtContext)

	// ExitAltertsconfigurationstmt is called when exiting the altertsconfigurationstmt production.
	ExitAltertsconfigurationstmt(c *AltertsconfigurationstmtContext)

	// ExitAny_with is called when exiting the any_with production.
	ExitAny_with(c *Any_withContext)

	// ExitCreateconversionstmt is called when exiting the createconversionstmt production.
	ExitCreateconversionstmt(c *CreateconversionstmtContext)

	// ExitClusterstmt is called when exiting the clusterstmt production.
	ExitClusterstmt(c *ClusterstmtContext)

	// ExitCluster_index_specification is called when exiting the cluster_index_specification production.
	ExitCluster_index_specification(c *Cluster_index_specificationContext)

	// ExitVacuumstmt is called when exiting the vacuumstmt production.
	ExitVacuumstmt(c *VacuumstmtContext)

	// ExitAnalyzestmt is called when exiting the analyzestmt production.
	ExitAnalyzestmt(c *AnalyzestmtContext)

	// ExitVac_analyze_option_list is called when exiting the vac_analyze_option_list production.
	ExitVac_analyze_option_list(c *Vac_analyze_option_listContext)

	// ExitAnalyze_keyword is called when exiting the analyze_keyword production.
	ExitAnalyze_keyword(c *Analyze_keywordContext)

	// ExitVac_analyze_option_elem is called when exiting the vac_analyze_option_elem production.
	ExitVac_analyze_option_elem(c *Vac_analyze_option_elemContext)

	// ExitVac_analyze_option_name is called when exiting the vac_analyze_option_name production.
	ExitVac_analyze_option_name(c *Vac_analyze_option_nameContext)

	// ExitVac_analyze_option_arg is called when exiting the vac_analyze_option_arg production.
	ExitVac_analyze_option_arg(c *Vac_analyze_option_argContext)

	// ExitOpt_analyze is called when exiting the opt_analyze production.
	ExitOpt_analyze(c *Opt_analyzeContext)

	// ExitOpt_verbose is called when exiting the opt_verbose production.
	ExitOpt_verbose(c *Opt_verboseContext)

	// ExitOpt_full is called when exiting the opt_full production.
	ExitOpt_full(c *Opt_fullContext)

	// ExitOpt_freeze is called when exiting the opt_freeze production.
	ExitOpt_freeze(c *Opt_freezeContext)

	// ExitOpt_name_list is called when exiting the opt_name_list production.
	ExitOpt_name_list(c *Opt_name_listContext)

	// ExitVacuum_relation is called when exiting the vacuum_relation production.
	ExitVacuum_relation(c *Vacuum_relationContext)

	// ExitVacuum_relation_list is called when exiting the vacuum_relation_list production.
	ExitVacuum_relation_list(c *Vacuum_relation_listContext)

	// ExitOpt_vacuum_relation_list is called when exiting the opt_vacuum_relation_list production.
	ExitOpt_vacuum_relation_list(c *Opt_vacuum_relation_listContext)

	// ExitExplainstmt is called when exiting the explainstmt production.
	ExitExplainstmt(c *ExplainstmtContext)

	// ExitExplainablestmt is called when exiting the explainablestmt production.
	ExitExplainablestmt(c *ExplainablestmtContext)

	// ExitExplain_option_list is called when exiting the explain_option_list production.
	ExitExplain_option_list(c *Explain_option_listContext)

	// ExitExplain_option_elem is called when exiting the explain_option_elem production.
	ExitExplain_option_elem(c *Explain_option_elemContext)

	// ExitExplain_option_name is called when exiting the explain_option_name production.
	ExitExplain_option_name(c *Explain_option_nameContext)

	// ExitExplain_option_arg is called when exiting the explain_option_arg production.
	ExitExplain_option_arg(c *Explain_option_argContext)

	// ExitPreparestmt is called when exiting the preparestmt production.
	ExitPreparestmt(c *PreparestmtContext)

	// ExitPrep_type_clause is called when exiting the prep_type_clause production.
	ExitPrep_type_clause(c *Prep_type_clauseContext)

	// ExitPreparablestmt is called when exiting the preparablestmt production.
	ExitPreparablestmt(c *PreparablestmtContext)

	// ExitExecutestmt is called when exiting the executestmt production.
	ExitExecutestmt(c *ExecutestmtContext)

	// ExitExecute_param_clause is called when exiting the execute_param_clause production.
	ExitExecute_param_clause(c *Execute_param_clauseContext)

	// ExitDeallocatestmt is called when exiting the deallocatestmt production.
	ExitDeallocatestmt(c *DeallocatestmtContext)

	// ExitInsertstmt is called when exiting the insertstmt production.
	ExitInsertstmt(c *InsertstmtContext)

	// ExitInsert_target is called when exiting the insert_target production.
	ExitInsert_target(c *Insert_targetContext)

	// ExitInsert_rest is called when exiting the insert_rest production.
	ExitInsert_rest(c *Insert_restContext)

	// ExitOverride_kind is called when exiting the override_kind production.
	ExitOverride_kind(c *Override_kindContext)

	// ExitInsert_column_list is called when exiting the insert_column_list production.
	ExitInsert_column_list(c *Insert_column_listContext)

	// ExitInsert_column_item is called when exiting the insert_column_item production.
	ExitInsert_column_item(c *Insert_column_itemContext)

	// ExitOpt_on_conflict is called when exiting the opt_on_conflict production.
	ExitOpt_on_conflict(c *Opt_on_conflictContext)

	// ExitOpt_conf_expr is called when exiting the opt_conf_expr production.
	ExitOpt_conf_expr(c *Opt_conf_exprContext)

	// ExitReturning_clause is called when exiting the returning_clause production.
	ExitReturning_clause(c *Returning_clauseContext)

	// ExitMergestmt is called when exiting the mergestmt production.
	ExitMergestmt(c *MergestmtContext)

	// ExitMerge_insert_clause is called when exiting the merge_insert_clause production.
	ExitMerge_insert_clause(c *Merge_insert_clauseContext)

	// ExitMerge_update_clause is called when exiting the merge_update_clause production.
	ExitMerge_update_clause(c *Merge_update_clauseContext)

	// ExitMerge_delete_clause is called when exiting the merge_delete_clause production.
	ExitMerge_delete_clause(c *Merge_delete_clauseContext)

	// ExitDeletestmt is called when exiting the deletestmt production.
	ExitDeletestmt(c *DeletestmtContext)

	// ExitUsing_clause is called when exiting the using_clause production.
	ExitUsing_clause(c *Using_clauseContext)

	// ExitLockstmt is called when exiting the lockstmt production.
	ExitLockstmt(c *LockstmtContext)

	// ExitOpt_lock is called when exiting the opt_lock production.
	ExitOpt_lock(c *Opt_lockContext)

	// ExitLock_type is called when exiting the lock_type production.
	ExitLock_type(c *Lock_typeContext)

	// ExitOpt_nowait is called when exiting the opt_nowait production.
	ExitOpt_nowait(c *Opt_nowaitContext)

	// ExitOpt_nowait_or_skip is called when exiting the opt_nowait_or_skip production.
	ExitOpt_nowait_or_skip(c *Opt_nowait_or_skipContext)

	// ExitUpdatestmt is called when exiting the updatestmt production.
	ExitUpdatestmt(c *UpdatestmtContext)

	// ExitSet_clause_list is called when exiting the set_clause_list production.
	ExitSet_clause_list(c *Set_clause_listContext)

	// ExitSet_clause is called when exiting the set_clause production.
	ExitSet_clause(c *Set_clauseContext)

	// ExitSet_target is called when exiting the set_target production.
	ExitSet_target(c *Set_targetContext)

	// ExitSet_target_list is called when exiting the set_target_list production.
	ExitSet_target_list(c *Set_target_listContext)

	// ExitDeclarecursorstmt is called when exiting the declarecursorstmt production.
	ExitDeclarecursorstmt(c *DeclarecursorstmtContext)

	// ExitCursor_name is called when exiting the cursor_name production.
	ExitCursor_name(c *Cursor_nameContext)

	// ExitCursor_options is called when exiting the cursor_options production.
	ExitCursor_options(c *Cursor_optionsContext)

	// ExitOpt_hold is called when exiting the opt_hold production.
	ExitOpt_hold(c *Opt_holdContext)

	// ExitSelectstmt is called when exiting the selectstmt production.
	ExitSelectstmt(c *SelectstmtContext)

	// ExitSelect_with_parens is called when exiting the select_with_parens production.
	ExitSelect_with_parens(c *Select_with_parensContext)

	// ExitSelect_no_parens is called when exiting the select_no_parens production.
	ExitSelect_no_parens(c *Select_no_parensContext)

	// ExitSelect_clause is called when exiting the select_clause production.
	ExitSelect_clause(c *Select_clauseContext)

	// ExitSimple_select_intersect is called when exiting the simple_select_intersect production.
	ExitSimple_select_intersect(c *Simple_select_intersectContext)

	// ExitSimple_select_pramary is called when exiting the simple_select_pramary production.
	ExitSimple_select_pramary(c *Simple_select_pramaryContext)

	// ExitWith_clause is called when exiting the with_clause production.
	ExitWith_clause(c *With_clauseContext)

	// ExitCte_list is called when exiting the cte_list production.
	ExitCte_list(c *Cte_listContext)

	// ExitCommon_table_expr is called when exiting the common_table_expr production.
	ExitCommon_table_expr(c *Common_table_exprContext)

	// ExitOpt_materialized is called when exiting the opt_materialized production.
	ExitOpt_materialized(c *Opt_materializedContext)

	// ExitOpt_with_clause is called when exiting the opt_with_clause production.
	ExitOpt_with_clause(c *Opt_with_clauseContext)

	// ExitInto_clause is called when exiting the into_clause production.
	ExitInto_clause(c *Into_clauseContext)

	// ExitOpt_strict is called when exiting the opt_strict production.
	ExitOpt_strict(c *Opt_strictContext)

	// ExitOpttempTableName is called when exiting the opttempTableName production.
	ExitOpttempTableName(c *OpttempTableNameContext)

	// ExitOpt_table is called when exiting the opt_table production.
	ExitOpt_table(c *Opt_tableContext)

	// ExitAll_or_distinct is called when exiting the all_or_distinct production.
	ExitAll_or_distinct(c *All_or_distinctContext)

	// ExitDistinct_clause is called when exiting the distinct_clause production.
	ExitDistinct_clause(c *Distinct_clauseContext)

	// ExitOpt_all_clause is called when exiting the opt_all_clause production.
	ExitOpt_all_clause(c *Opt_all_clauseContext)

	// ExitOpt_sort_clause is called when exiting the opt_sort_clause production.
	ExitOpt_sort_clause(c *Opt_sort_clauseContext)

	// ExitSort_clause is called when exiting the sort_clause production.
	ExitSort_clause(c *Sort_clauseContext)

	// ExitSortby_list is called when exiting the sortby_list production.
	ExitSortby_list(c *Sortby_listContext)

	// ExitSortby is called when exiting the sortby production.
	ExitSortby(c *SortbyContext)

	// ExitSelect_limit is called when exiting the select_limit production.
	ExitSelect_limit(c *Select_limitContext)

	// ExitOpt_select_limit is called when exiting the opt_select_limit production.
	ExitOpt_select_limit(c *Opt_select_limitContext)

	// ExitLimit_clause is called when exiting the limit_clause production.
	ExitLimit_clause(c *Limit_clauseContext)

	// ExitOffset_clause is called when exiting the offset_clause production.
	ExitOffset_clause(c *Offset_clauseContext)

	// ExitSelect_limit_value is called when exiting the select_limit_value production.
	ExitSelect_limit_value(c *Select_limit_valueContext)

	// ExitSelect_offset_value is called when exiting the select_offset_value production.
	ExitSelect_offset_value(c *Select_offset_valueContext)

	// ExitSelect_fetch_first_value is called when exiting the select_fetch_first_value production.
	ExitSelect_fetch_first_value(c *Select_fetch_first_valueContext)

	// ExitI_or_f_const is called when exiting the i_or_f_const production.
	ExitI_or_f_const(c *I_or_f_constContext)

	// ExitRow_or_rows is called when exiting the row_or_rows production.
	ExitRow_or_rows(c *Row_or_rowsContext)

	// ExitFirst_or_next is called when exiting the first_or_next production.
	ExitFirst_or_next(c *First_or_nextContext)

	// ExitGroup_clause is called when exiting the group_clause production.
	ExitGroup_clause(c *Group_clauseContext)

	// ExitGroup_by_list is called when exiting the group_by_list production.
	ExitGroup_by_list(c *Group_by_listContext)

	// ExitGroup_by_item is called when exiting the group_by_item production.
	ExitGroup_by_item(c *Group_by_itemContext)

	// ExitEmpty_grouping_set is called when exiting the empty_grouping_set production.
	ExitEmpty_grouping_set(c *Empty_grouping_setContext)

	// ExitRollup_clause is called when exiting the rollup_clause production.
	ExitRollup_clause(c *Rollup_clauseContext)

	// ExitCube_clause is called when exiting the cube_clause production.
	ExitCube_clause(c *Cube_clauseContext)

	// ExitGrouping_sets_clause is called when exiting the grouping_sets_clause production.
	ExitGrouping_sets_clause(c *Grouping_sets_clauseContext)

	// ExitHaving_clause is called when exiting the having_clause production.
	ExitHaving_clause(c *Having_clauseContext)

	// ExitFor_locking_clause is called when exiting the for_locking_clause production.
	ExitFor_locking_clause(c *For_locking_clauseContext)

	// ExitOpt_for_locking_clause is called when exiting the opt_for_locking_clause production.
	ExitOpt_for_locking_clause(c *Opt_for_locking_clauseContext)

	// ExitFor_locking_items is called when exiting the for_locking_items production.
	ExitFor_locking_items(c *For_locking_itemsContext)

	// ExitFor_locking_item is called when exiting the for_locking_item production.
	ExitFor_locking_item(c *For_locking_itemContext)

	// ExitFor_locking_strength is called when exiting the for_locking_strength production.
	ExitFor_locking_strength(c *For_locking_strengthContext)

	// ExitLocked_rels_list is called when exiting the locked_rels_list production.
	ExitLocked_rels_list(c *Locked_rels_listContext)

	// ExitValues_clause is called when exiting the values_clause production.
	ExitValues_clause(c *Values_clauseContext)

	// ExitFrom_clause is called when exiting the from_clause production.
	ExitFrom_clause(c *From_clauseContext)

	// ExitFrom_list is called when exiting the from_list production.
	ExitFrom_list(c *From_listContext)

	// ExitNon_ansi_join is called when exiting the non_ansi_join production.
	ExitNon_ansi_join(c *Non_ansi_joinContext)

	// ExitTable_ref is called when exiting the table_ref production.
	ExitTable_ref(c *Table_refContext)

	// ExitAlias_clause is called when exiting the alias_clause production.
	ExitAlias_clause(c *Alias_clauseContext)

	// ExitOpt_alias_clause is called when exiting the opt_alias_clause production.
	ExitOpt_alias_clause(c *Opt_alias_clauseContext)

	// ExitTable_alias_clause is called when exiting the table_alias_clause production.
	ExitTable_alias_clause(c *Table_alias_clauseContext)

	// ExitFunc_alias_clause is called when exiting the func_alias_clause production.
	ExitFunc_alias_clause(c *Func_alias_clauseContext)

	// ExitJoin_type is called when exiting the join_type production.
	ExitJoin_type(c *Join_typeContext)

	// ExitJoin_qual is called when exiting the join_qual production.
	ExitJoin_qual(c *Join_qualContext)

	// ExitRelation_expr is called when exiting the relation_expr production.
	ExitRelation_expr(c *Relation_exprContext)

	// ExitRelation_expr_list is called when exiting the relation_expr_list production.
	ExitRelation_expr_list(c *Relation_expr_listContext)

	// ExitRelation_expr_opt_alias is called when exiting the relation_expr_opt_alias production.
	ExitRelation_expr_opt_alias(c *Relation_expr_opt_aliasContext)

	// ExitTablesample_clause is called when exiting the tablesample_clause production.
	ExitTablesample_clause(c *Tablesample_clauseContext)

	// ExitOpt_repeatable_clause is called when exiting the opt_repeatable_clause production.
	ExitOpt_repeatable_clause(c *Opt_repeatable_clauseContext)

	// ExitFunc_table is called when exiting the func_table production.
	ExitFunc_table(c *Func_tableContext)

	// ExitRowsfrom_item is called when exiting the rowsfrom_item production.
	ExitRowsfrom_item(c *Rowsfrom_itemContext)

	// ExitRowsfrom_list is called when exiting the rowsfrom_list production.
	ExitRowsfrom_list(c *Rowsfrom_listContext)

	// ExitOpt_col_def_list is called when exiting the opt_col_def_list production.
	ExitOpt_col_def_list(c *Opt_col_def_listContext)

	// ExitOpt_ordinality is called when exiting the opt_ordinality production.
	ExitOpt_ordinality(c *Opt_ordinalityContext)

	// ExitWhere_clause is called when exiting the where_clause production.
	ExitWhere_clause(c *Where_clauseContext)

	// ExitWhere_or_current_clause is called when exiting the where_or_current_clause production.
	ExitWhere_or_current_clause(c *Where_or_current_clauseContext)

	// ExitOpttablefuncelementlist is called when exiting the opttablefuncelementlist production.
	ExitOpttablefuncelementlist(c *OpttablefuncelementlistContext)

	// ExitTablefuncelementlist is called when exiting the tablefuncelementlist production.
	ExitTablefuncelementlist(c *TablefuncelementlistContext)

	// ExitTablefuncelement is called when exiting the tablefuncelement production.
	ExitTablefuncelement(c *TablefuncelementContext)

	// ExitXmltable is called when exiting the xmltable production.
	ExitXmltable(c *XmltableContext)

	// ExitXmltable_column_list is called when exiting the xmltable_column_list production.
	ExitXmltable_column_list(c *Xmltable_column_listContext)

	// ExitXmltable_column_el is called when exiting the xmltable_column_el production.
	ExitXmltable_column_el(c *Xmltable_column_elContext)

	// ExitXmltable_column_option_list is called when exiting the xmltable_column_option_list production.
	ExitXmltable_column_option_list(c *Xmltable_column_option_listContext)

	// ExitXmltable_column_option_el is called when exiting the xmltable_column_option_el production.
	ExitXmltable_column_option_el(c *Xmltable_column_option_elContext)

	// ExitXml_namespace_list is called when exiting the xml_namespace_list production.
	ExitXml_namespace_list(c *Xml_namespace_listContext)

	// ExitXml_namespace_el is called when exiting the xml_namespace_el production.
	ExitXml_namespace_el(c *Xml_namespace_elContext)

	// ExitTypename is called when exiting the typename production.
	ExitTypename(c *TypenameContext)

	// ExitOpt_array_bounds is called when exiting the opt_array_bounds production.
	ExitOpt_array_bounds(c *Opt_array_boundsContext)

	// ExitSimpletypename is called when exiting the simpletypename production.
	ExitSimpletypename(c *SimpletypenameContext)

	// ExitConsttypename is called when exiting the consttypename production.
	ExitConsttypename(c *ConsttypenameContext)

	// ExitGenerictype is called when exiting the generictype production.
	ExitGenerictype(c *GenerictypeContext)

	// ExitOpt_type_modifiers is called when exiting the opt_type_modifiers production.
	ExitOpt_type_modifiers(c *Opt_type_modifiersContext)

	// ExitNumeric is called when exiting the numeric production.
	ExitNumeric(c *NumericContext)

	// ExitOpt_float is called when exiting the opt_float production.
	ExitOpt_float(c *Opt_floatContext)

	// ExitBit is called when exiting the bit production.
	ExitBit(c *BitContext)

	// ExitConstbit is called when exiting the constbit production.
	ExitConstbit(c *ConstbitContext)

	// ExitBitwithlength is called when exiting the bitwithlength production.
	ExitBitwithlength(c *BitwithlengthContext)

	// ExitBitwithoutlength is called when exiting the bitwithoutlength production.
	ExitBitwithoutlength(c *BitwithoutlengthContext)

	// ExitCharacter is called when exiting the character production.
	ExitCharacter(c *CharacterContext)

	// ExitConstcharacter is called when exiting the constcharacter production.
	ExitConstcharacter(c *ConstcharacterContext)

	// ExitCharacter_c is called when exiting the character_c production.
	ExitCharacter_c(c *Character_cContext)

	// ExitOpt_varying is called when exiting the opt_varying production.
	ExitOpt_varying(c *Opt_varyingContext)

	// ExitConstdatetime is called when exiting the constdatetime production.
	ExitConstdatetime(c *ConstdatetimeContext)

	// ExitConstinterval is called when exiting the constinterval production.
	ExitConstinterval(c *ConstintervalContext)

	// ExitOpt_timezone is called when exiting the opt_timezone production.
	ExitOpt_timezone(c *Opt_timezoneContext)

	// ExitOpt_interval is called when exiting the opt_interval production.
	ExitOpt_interval(c *Opt_intervalContext)

	// ExitInterval_second is called when exiting the interval_second production.
	ExitInterval_second(c *Interval_secondContext)

	// ExitOpt_escape is called when exiting the opt_escape production.
	ExitOpt_escape(c *Opt_escapeContext)

	// ExitA_expr is called when exiting the a_expr production.
	ExitA_expr(c *A_exprContext)

	// ExitA_expr_qual is called when exiting the a_expr_qual production.
	ExitA_expr_qual(c *A_expr_qualContext)

	// ExitA_expr_lessless is called when exiting the a_expr_lessless production.
	ExitA_expr_lessless(c *A_expr_lesslessContext)

	// ExitA_expr_or is called when exiting the a_expr_or production.
	ExitA_expr_or(c *A_expr_orContext)

	// ExitA_expr_and is called when exiting the a_expr_and production.
	ExitA_expr_and(c *A_expr_andContext)

	// ExitA_expr_between is called when exiting the a_expr_between production.
	ExitA_expr_between(c *A_expr_betweenContext)

	// ExitA_expr_in is called when exiting the a_expr_in production.
	ExitA_expr_in(c *A_expr_inContext)

	// ExitA_expr_unary_not is called when exiting the a_expr_unary_not production.
	ExitA_expr_unary_not(c *A_expr_unary_notContext)

	// ExitA_expr_isnull is called when exiting the a_expr_isnull production.
	ExitA_expr_isnull(c *A_expr_isnullContext)

	// ExitA_expr_is_not is called when exiting the a_expr_is_not production.
	ExitA_expr_is_not(c *A_expr_is_notContext)

	// ExitA_expr_compare is called when exiting the a_expr_compare production.
	ExitA_expr_compare(c *A_expr_compareContext)

	// ExitA_expr_like is called when exiting the a_expr_like production.
	ExitA_expr_like(c *A_expr_likeContext)

	// ExitA_expr_qual_op is called when exiting the a_expr_qual_op production.
	ExitA_expr_qual_op(c *A_expr_qual_opContext)

	// ExitA_expr_unary_qualop is called when exiting the a_expr_unary_qualop production.
	ExitA_expr_unary_qualop(c *A_expr_unary_qualopContext)

	// ExitA_expr_add is called when exiting the a_expr_add production.
	ExitA_expr_add(c *A_expr_addContext)

	// ExitA_expr_mul is called when exiting the a_expr_mul production.
	ExitA_expr_mul(c *A_expr_mulContext)

	// ExitA_expr_caret is called when exiting the a_expr_caret production.
	ExitA_expr_caret(c *A_expr_caretContext)

	// ExitA_expr_unary_sign is called when exiting the a_expr_unary_sign production.
	ExitA_expr_unary_sign(c *A_expr_unary_signContext)

	// ExitA_expr_at_time_zone is called when exiting the a_expr_at_time_zone production.
	ExitA_expr_at_time_zone(c *A_expr_at_time_zoneContext)

	// ExitA_expr_collate is called when exiting the a_expr_collate production.
	ExitA_expr_collate(c *A_expr_collateContext)

	// ExitA_expr_typecast is called when exiting the a_expr_typecast production.
	ExitA_expr_typecast(c *A_expr_typecastContext)

	// ExitB_expr is called when exiting the b_expr production.
	ExitB_expr(c *B_exprContext)

	// ExitC_expr_exists is called when exiting the c_expr_exists production.
	ExitC_expr_exists(c *C_expr_existsContext)

	// ExitC_expr_expr is called when exiting the c_expr_expr production.
	ExitC_expr_expr(c *C_expr_exprContext)

	// ExitC_expr_case is called when exiting the c_expr_case production.
	ExitC_expr_case(c *C_expr_caseContext)

	// ExitPlsqlvariablename is called when exiting the plsqlvariablename production.
	ExitPlsqlvariablename(c *PlsqlvariablenameContext)

	// ExitFunc_application is called when exiting the func_application production.
	ExitFunc_application(c *Func_applicationContext)

	// ExitFunc_expr is called when exiting the func_expr production.
	ExitFunc_expr(c *Func_exprContext)

	// ExitFunc_expr_windowless is called when exiting the func_expr_windowless production.
	ExitFunc_expr_windowless(c *Func_expr_windowlessContext)

	// ExitFunc_expr_common_subexpr is called when exiting the func_expr_common_subexpr production.
	ExitFunc_expr_common_subexpr(c *Func_expr_common_subexprContext)

	// ExitXml_root_version is called when exiting the xml_root_version production.
	ExitXml_root_version(c *Xml_root_versionContext)

	// ExitOpt_xml_root_standalone is called when exiting the opt_xml_root_standalone production.
	ExitOpt_xml_root_standalone(c *Opt_xml_root_standaloneContext)

	// ExitXml_attributes is called when exiting the xml_attributes production.
	ExitXml_attributes(c *Xml_attributesContext)

	// ExitXml_attribute_list is called when exiting the xml_attribute_list production.
	ExitXml_attribute_list(c *Xml_attribute_listContext)

	// ExitXml_attribute_el is called when exiting the xml_attribute_el production.
	ExitXml_attribute_el(c *Xml_attribute_elContext)

	// ExitDocument_or_content is called when exiting the document_or_content production.
	ExitDocument_or_content(c *Document_or_contentContext)

	// ExitXml_whitespace_option is called when exiting the xml_whitespace_option production.
	ExitXml_whitespace_option(c *Xml_whitespace_optionContext)

	// ExitXmlexists_argument is called when exiting the xmlexists_argument production.
	ExitXmlexists_argument(c *Xmlexists_argumentContext)

	// ExitXml_passing_mech is called when exiting the xml_passing_mech production.
	ExitXml_passing_mech(c *Xml_passing_mechContext)

	// ExitWithin_group_clause is called when exiting the within_group_clause production.
	ExitWithin_group_clause(c *Within_group_clauseContext)

	// ExitFilter_clause is called when exiting the filter_clause production.
	ExitFilter_clause(c *Filter_clauseContext)

	// ExitWindow_clause is called when exiting the window_clause production.
	ExitWindow_clause(c *Window_clauseContext)

	// ExitWindow_definition_list is called when exiting the window_definition_list production.
	ExitWindow_definition_list(c *Window_definition_listContext)

	// ExitWindow_definition is called when exiting the window_definition production.
	ExitWindow_definition(c *Window_definitionContext)

	// ExitOver_clause is called when exiting the over_clause production.
	ExitOver_clause(c *Over_clauseContext)

	// ExitWindow_specification is called when exiting the window_specification production.
	ExitWindow_specification(c *Window_specificationContext)

	// ExitOpt_existing_window_name is called when exiting the opt_existing_window_name production.
	ExitOpt_existing_window_name(c *Opt_existing_window_nameContext)

	// ExitOpt_partition_clause is called when exiting the opt_partition_clause production.
	ExitOpt_partition_clause(c *Opt_partition_clauseContext)

	// ExitOpt_frame_clause is called when exiting the opt_frame_clause production.
	ExitOpt_frame_clause(c *Opt_frame_clauseContext)

	// ExitFrame_extent is called when exiting the frame_extent production.
	ExitFrame_extent(c *Frame_extentContext)

	// ExitFrame_bound is called when exiting the frame_bound production.
	ExitFrame_bound(c *Frame_boundContext)

	// ExitOpt_window_exclusion_clause is called when exiting the opt_window_exclusion_clause production.
	ExitOpt_window_exclusion_clause(c *Opt_window_exclusion_clauseContext)

	// ExitRow is called when exiting the row production.
	ExitRow(c *RowContext)

	// ExitExplicit_row is called when exiting the explicit_row production.
	ExitExplicit_row(c *Explicit_rowContext)

	// ExitImplicit_row is called when exiting the implicit_row production.
	ExitImplicit_row(c *Implicit_rowContext)

	// ExitSub_type is called when exiting the sub_type production.
	ExitSub_type(c *Sub_typeContext)

	// ExitAll_op is called when exiting the all_op production.
	ExitAll_op(c *All_opContext)

	// ExitMathop is called when exiting the mathop production.
	ExitMathop(c *MathopContext)

	// ExitQual_op is called when exiting the qual_op production.
	ExitQual_op(c *Qual_opContext)

	// ExitQual_all_op is called when exiting the qual_all_op production.
	ExitQual_all_op(c *Qual_all_opContext)

	// ExitSubquery_Op is called when exiting the subquery_Op production.
	ExitSubquery_Op(c *Subquery_OpContext)

	// ExitExpr_list is called when exiting the expr_list production.
	ExitExpr_list(c *Expr_listContext)

	// ExitFunc_arg_list is called when exiting the func_arg_list production.
	ExitFunc_arg_list(c *Func_arg_listContext)

	// ExitFunc_arg_expr is called when exiting the func_arg_expr production.
	ExitFunc_arg_expr(c *Func_arg_exprContext)

	// ExitType_list is called when exiting the type_list production.
	ExitType_list(c *Type_listContext)

	// ExitArray_expr is called when exiting the array_expr production.
	ExitArray_expr(c *Array_exprContext)

	// ExitArray_expr_list is called when exiting the array_expr_list production.
	ExitArray_expr_list(c *Array_expr_listContext)

	// ExitExtract_list is called when exiting the extract_list production.
	ExitExtract_list(c *Extract_listContext)

	// ExitExtract_arg is called when exiting the extract_arg production.
	ExitExtract_arg(c *Extract_argContext)

	// ExitUnicode_normal_form is called when exiting the unicode_normal_form production.
	ExitUnicode_normal_form(c *Unicode_normal_formContext)

	// ExitOverlay_list is called when exiting the overlay_list production.
	ExitOverlay_list(c *Overlay_listContext)

	// ExitPosition_list is called when exiting the position_list production.
	ExitPosition_list(c *Position_listContext)

	// ExitSubstr_list is called when exiting the substr_list production.
	ExitSubstr_list(c *Substr_listContext)

	// ExitTrim_list is called when exiting the trim_list production.
	ExitTrim_list(c *Trim_listContext)

	// ExitIn_expr_select is called when exiting the in_expr_select production.
	ExitIn_expr_select(c *In_expr_selectContext)

	// ExitIn_expr_list is called when exiting the in_expr_list production.
	ExitIn_expr_list(c *In_expr_listContext)

	// ExitCase_expr is called when exiting the case_expr production.
	ExitCase_expr(c *Case_exprContext)

	// ExitWhen_clause_list is called when exiting the when_clause_list production.
	ExitWhen_clause_list(c *When_clause_listContext)

	// ExitWhen_clause is called when exiting the when_clause production.
	ExitWhen_clause(c *When_clauseContext)

	// ExitCase_default is called when exiting the case_default production.
	ExitCase_default(c *Case_defaultContext)

	// ExitCase_arg is called when exiting the case_arg production.
	ExitCase_arg(c *Case_argContext)

	// ExitColumnref is called when exiting the columnref production.
	ExitColumnref(c *ColumnrefContext)

	// ExitIndirection_el is called when exiting the indirection_el production.
	ExitIndirection_el(c *Indirection_elContext)

	// ExitOpt_slice_bound is called when exiting the opt_slice_bound production.
	ExitOpt_slice_bound(c *Opt_slice_boundContext)

	// ExitIndirection is called when exiting the indirection production.
	ExitIndirection(c *IndirectionContext)

	// ExitOpt_indirection is called when exiting the opt_indirection production.
	ExitOpt_indirection(c *Opt_indirectionContext)

	// ExitOpt_target_list is called when exiting the opt_target_list production.
	ExitOpt_target_list(c *Opt_target_listContext)

	// ExitTarget_list is called when exiting the target_list production.
	ExitTarget_list(c *Target_listContext)

	// ExitTarget_label is called when exiting the target_label production.
	ExitTarget_label(c *Target_labelContext)

	// ExitTarget_star is called when exiting the target_star production.
	ExitTarget_star(c *Target_starContext)

	// ExitQualified_name_list is called when exiting the qualified_name_list production.
	ExitQualified_name_list(c *Qualified_name_listContext)

	// ExitQualified_name is called when exiting the qualified_name production.
	ExitQualified_name(c *Qualified_nameContext)

	// ExitName_list is called when exiting the name_list production.
	ExitName_list(c *Name_listContext)

	// ExitName is called when exiting the name production.
	ExitName(c *NameContext)

	// ExitAttr_name is called when exiting the attr_name production.
	ExitAttr_name(c *Attr_nameContext)

	// ExitFile_name is called when exiting the file_name production.
	ExitFile_name(c *File_nameContext)

	// ExitFunc_name is called when exiting the func_name production.
	ExitFunc_name(c *Func_nameContext)

	// ExitAexprconst is called when exiting the aexprconst production.
	ExitAexprconst(c *AexprconstContext)

	// ExitXconst is called when exiting the xconst production.
	ExitXconst(c *XconstContext)

	// ExitBconst is called when exiting the bconst production.
	ExitBconst(c *BconstContext)

	// ExitFconst is called when exiting the fconst production.
	ExitFconst(c *FconstContext)

	// ExitIconst is called when exiting the iconst production.
	ExitIconst(c *IconstContext)

	// ExitSconst is called when exiting the sconst production.
	ExitSconst(c *SconstContext)

	// ExitAnysconst is called when exiting the anysconst production.
	ExitAnysconst(c *AnysconstContext)

	// ExitOpt_uescape is called when exiting the opt_uescape production.
	ExitOpt_uescape(c *Opt_uescapeContext)

	// ExitSignediconst is called when exiting the signediconst production.
	ExitSignediconst(c *SignediconstContext)

	// ExitRoleid is called when exiting the roleid production.
	ExitRoleid(c *RoleidContext)

	// ExitRolespec is called when exiting the rolespec production.
	ExitRolespec(c *RolespecContext)

	// ExitRole_list is called when exiting the role_list production.
	ExitRole_list(c *Role_listContext)

	// ExitColid is called when exiting the colid production.
	ExitColid(c *ColidContext)

	// ExitTable_alias is called when exiting the table_alias production.
	ExitTable_alias(c *Table_aliasContext)

	// ExitType_function_name is called when exiting the type_function_name production.
	ExitType_function_name(c *Type_function_nameContext)

	// ExitNonreservedword is called when exiting the nonreservedword production.
	ExitNonreservedword(c *NonreservedwordContext)

	// ExitCollabel is called when exiting the collabel production.
	ExitCollabel(c *CollabelContext)

	// ExitIdentifier is called when exiting the identifier production.
	ExitIdentifier(c *IdentifierContext)

	// ExitPlsqlidentifier is called when exiting the plsqlidentifier production.
	ExitPlsqlidentifier(c *PlsqlidentifierContext)

	// ExitUnreserved_keyword is called when exiting the unreserved_keyword production.
	ExitUnreserved_keyword(c *Unreserved_keywordContext)

	// ExitCol_name_keyword is called when exiting the col_name_keyword production.
	ExitCol_name_keyword(c *Col_name_keywordContext)

	// ExitType_func_name_keyword is called when exiting the type_func_name_keyword production.
	ExitType_func_name_keyword(c *Type_func_name_keywordContext)

	// ExitReserved_keyword is called when exiting the reserved_keyword production.
	ExitReserved_keyword(c *Reserved_keywordContext)

	// ExitBuiltin_function_name is called when exiting the builtin_function_name production.
	ExitBuiltin_function_name(c *Builtin_function_nameContext)

	// ExitPl_function is called when exiting the pl_function production.
	ExitPl_function(c *Pl_functionContext)

	// ExitComp_options is called when exiting the comp_options production.
	ExitComp_options(c *Comp_optionsContext)

	// ExitComp_option is called when exiting the comp_option production.
	ExitComp_option(c *Comp_optionContext)

	// ExitSharp is called when exiting the sharp production.
	ExitSharp(c *SharpContext)

	// ExitOption_value is called when exiting the option_value production.
	ExitOption_value(c *Option_valueContext)

	// ExitOpt_semi is called when exiting the opt_semi production.
	ExitOpt_semi(c *Opt_semiContext)

	// ExitPl_block is called when exiting the pl_block production.
	ExitPl_block(c *Pl_blockContext)

	// ExitDecl_sect is called when exiting the decl_sect production.
	ExitDecl_sect(c *Decl_sectContext)

	// ExitDecl_start is called when exiting the decl_start production.
	ExitDecl_start(c *Decl_startContext)

	// ExitDecl_stmts is called when exiting the decl_stmts production.
	ExitDecl_stmts(c *Decl_stmtsContext)

	// ExitLabel_decl is called when exiting the label_decl production.
	ExitLabel_decl(c *Label_declContext)

	// ExitDecl_stmt is called when exiting the decl_stmt production.
	ExitDecl_stmt(c *Decl_stmtContext)

	// ExitDecl_statement is called when exiting the decl_statement production.
	ExitDecl_statement(c *Decl_statementContext)

	// ExitOpt_scrollable is called when exiting the opt_scrollable production.
	ExitOpt_scrollable(c *Opt_scrollableContext)

	// ExitDecl_cursor_query is called when exiting the decl_cursor_query production.
	ExitDecl_cursor_query(c *Decl_cursor_queryContext)

	// ExitDecl_cursor_args is called when exiting the decl_cursor_args production.
	ExitDecl_cursor_args(c *Decl_cursor_argsContext)

	// ExitDecl_cursor_arglist is called when exiting the decl_cursor_arglist production.
	ExitDecl_cursor_arglist(c *Decl_cursor_arglistContext)

	// ExitDecl_cursor_arg is called when exiting the decl_cursor_arg production.
	ExitDecl_cursor_arg(c *Decl_cursor_argContext)

	// ExitDecl_is_for is called when exiting the decl_is_for production.
	ExitDecl_is_for(c *Decl_is_forContext)

	// ExitDecl_aliasitem is called when exiting the decl_aliasitem production.
	ExitDecl_aliasitem(c *Decl_aliasitemContext)

	// ExitDecl_varname is called when exiting the decl_varname production.
	ExitDecl_varname(c *Decl_varnameContext)

	// ExitDecl_const is called when exiting the decl_const production.
	ExitDecl_const(c *Decl_constContext)

	// ExitDecl_datatype is called when exiting the decl_datatype production.
	ExitDecl_datatype(c *Decl_datatypeContext)

	// ExitDecl_collate is called when exiting the decl_collate production.
	ExitDecl_collate(c *Decl_collateContext)

	// ExitDecl_notnull is called when exiting the decl_notnull production.
	ExitDecl_notnull(c *Decl_notnullContext)

	// ExitDecl_defval is called when exiting the decl_defval production.
	ExitDecl_defval(c *Decl_defvalContext)

	// ExitDecl_defkey is called when exiting the decl_defkey production.
	ExitDecl_defkey(c *Decl_defkeyContext)

	// ExitAssign_operator is called when exiting the assign_operator production.
	ExitAssign_operator(c *Assign_operatorContext)

	// ExitProc_sect is called when exiting the proc_sect production.
	ExitProc_sect(c *Proc_sectContext)

	// ExitProc_stmt is called when exiting the proc_stmt production.
	ExitProc_stmt(c *Proc_stmtContext)

	// ExitStmt_perform is called when exiting the stmt_perform production.
	ExitStmt_perform(c *Stmt_performContext)

	// ExitStmt_call is called when exiting the stmt_call production.
	ExitStmt_call(c *Stmt_callContext)

	// ExitOpt_expr_list is called when exiting the opt_expr_list production.
	ExitOpt_expr_list(c *Opt_expr_listContext)

	// ExitStmt_assign is called when exiting the stmt_assign production.
	ExitStmt_assign(c *Stmt_assignContext)

	// ExitStmt_getdiag is called when exiting the stmt_getdiag production.
	ExitStmt_getdiag(c *Stmt_getdiagContext)

	// ExitGetdiag_area_opt is called when exiting the getdiag_area_opt production.
	ExitGetdiag_area_opt(c *Getdiag_area_optContext)

	// ExitGetdiag_list is called when exiting the getdiag_list production.
	ExitGetdiag_list(c *Getdiag_listContext)

	// ExitGetdiag_list_item is called when exiting the getdiag_list_item production.
	ExitGetdiag_list_item(c *Getdiag_list_itemContext)

	// ExitGetdiag_item is called when exiting the getdiag_item production.
	ExitGetdiag_item(c *Getdiag_itemContext)

	// ExitGetdiag_target is called when exiting the getdiag_target production.
	ExitGetdiag_target(c *Getdiag_targetContext)

	// ExitAssign_var is called when exiting the assign_var production.
	ExitAssign_var(c *Assign_varContext)

	// ExitStmt_if is called when exiting the stmt_if production.
	ExitStmt_if(c *Stmt_ifContext)

	// ExitStmt_elsifs is called when exiting the stmt_elsifs production.
	ExitStmt_elsifs(c *Stmt_elsifsContext)

	// ExitStmt_else is called when exiting the stmt_else production.
	ExitStmt_else(c *Stmt_elseContext)

	// ExitStmt_case is called when exiting the stmt_case production.
	ExitStmt_case(c *Stmt_caseContext)

	// ExitOpt_expr_until_when is called when exiting the opt_expr_until_when production.
	ExitOpt_expr_until_when(c *Opt_expr_until_whenContext)

	// ExitCase_when_list is called when exiting the case_when_list production.
	ExitCase_when_list(c *Case_when_listContext)

	// ExitCase_when is called when exiting the case_when production.
	ExitCase_when(c *Case_whenContext)

	// ExitOpt_case_else is called when exiting the opt_case_else production.
	ExitOpt_case_else(c *Opt_case_elseContext)

	// ExitStmt_loop is called when exiting the stmt_loop production.
	ExitStmt_loop(c *Stmt_loopContext)

	// ExitStmt_while is called when exiting the stmt_while production.
	ExitStmt_while(c *Stmt_whileContext)

	// ExitStmt_for is called when exiting the stmt_for production.
	ExitStmt_for(c *Stmt_forContext)

	// ExitFor_control is called when exiting the for_control production.
	ExitFor_control(c *For_controlContext)

	// ExitOpt_for_using_expression is called when exiting the opt_for_using_expression production.
	ExitOpt_for_using_expression(c *Opt_for_using_expressionContext)

	// ExitOpt_cursor_parameters is called when exiting the opt_cursor_parameters production.
	ExitOpt_cursor_parameters(c *Opt_cursor_parametersContext)

	// ExitOpt_reverse is called when exiting the opt_reverse production.
	ExitOpt_reverse(c *Opt_reverseContext)

	// ExitOpt_by_expression is called when exiting the opt_by_expression production.
	ExitOpt_by_expression(c *Opt_by_expressionContext)

	// ExitFor_variable is called when exiting the for_variable production.
	ExitFor_variable(c *For_variableContext)

	// ExitStmt_foreach_a is called when exiting the stmt_foreach_a production.
	ExitStmt_foreach_a(c *Stmt_foreach_aContext)

	// ExitForeach_slice is called when exiting the foreach_slice production.
	ExitForeach_slice(c *Foreach_sliceContext)

	// ExitStmt_exit is called when exiting the stmt_exit production.
	ExitStmt_exit(c *Stmt_exitContext)

	// ExitExit_type is called when exiting the exit_type production.
	ExitExit_type(c *Exit_typeContext)

	// ExitStmt_return is called when exiting the stmt_return production.
	ExitStmt_return(c *Stmt_returnContext)

	// ExitOpt_return_result is called when exiting the opt_return_result production.
	ExitOpt_return_result(c *Opt_return_resultContext)

	// ExitStmt_raise is called when exiting the stmt_raise production.
	ExitStmt_raise(c *Stmt_raiseContext)

	// ExitOpt_stmt_raise_level is called when exiting the opt_stmt_raise_level production.
	ExitOpt_stmt_raise_level(c *Opt_stmt_raise_levelContext)

	// ExitOpt_raise_list is called when exiting the opt_raise_list production.
	ExitOpt_raise_list(c *Opt_raise_listContext)

	// ExitOpt_raise_using is called when exiting the opt_raise_using production.
	ExitOpt_raise_using(c *Opt_raise_usingContext)

	// ExitOpt_raise_using_elem is called when exiting the opt_raise_using_elem production.
	ExitOpt_raise_using_elem(c *Opt_raise_using_elemContext)

	// ExitOpt_raise_using_elem_list is called when exiting the opt_raise_using_elem_list production.
	ExitOpt_raise_using_elem_list(c *Opt_raise_using_elem_listContext)

	// ExitStmt_assert is called when exiting the stmt_assert production.
	ExitStmt_assert(c *Stmt_assertContext)

	// ExitOpt_stmt_assert_message is called when exiting the opt_stmt_assert_message production.
	ExitOpt_stmt_assert_message(c *Opt_stmt_assert_messageContext)

	// ExitLoop_body is called when exiting the loop_body production.
	ExitLoop_body(c *Loop_bodyContext)

	// ExitStmt_execsql is called when exiting the stmt_execsql production.
	ExitStmt_execsql(c *Stmt_execsqlContext)

	// ExitStmt_dynexecute is called when exiting the stmt_dynexecute production.
	ExitStmt_dynexecute(c *Stmt_dynexecuteContext)

	// ExitOpt_execute_using is called when exiting the opt_execute_using production.
	ExitOpt_execute_using(c *Opt_execute_usingContext)

	// ExitOpt_execute_using_list is called when exiting the opt_execute_using_list production.
	ExitOpt_execute_using_list(c *Opt_execute_using_listContext)

	// ExitOpt_execute_into is called when exiting the opt_execute_into production.
	ExitOpt_execute_into(c *Opt_execute_intoContext)

	// ExitStmt_open is called when exiting the stmt_open production.
	ExitStmt_open(c *Stmt_openContext)

	// ExitOpt_open_bound_list_item is called when exiting the opt_open_bound_list_item production.
	ExitOpt_open_bound_list_item(c *Opt_open_bound_list_itemContext)

	// ExitOpt_open_bound_list is called when exiting the opt_open_bound_list production.
	ExitOpt_open_bound_list(c *Opt_open_bound_listContext)

	// ExitOpt_open_using is called when exiting the opt_open_using production.
	ExitOpt_open_using(c *Opt_open_usingContext)

	// ExitOpt_scroll_option is called when exiting the opt_scroll_option production.
	ExitOpt_scroll_option(c *Opt_scroll_optionContext)

	// ExitOpt_scroll_option_no is called when exiting the opt_scroll_option_no production.
	ExitOpt_scroll_option_no(c *Opt_scroll_option_noContext)

	// ExitStmt_fetch is called when exiting the stmt_fetch production.
	ExitStmt_fetch(c *Stmt_fetchContext)

	// ExitInto_target is called when exiting the into_target production.
	ExitInto_target(c *Into_targetContext)

	// ExitOpt_cursor_from is called when exiting the opt_cursor_from production.
	ExitOpt_cursor_from(c *Opt_cursor_fromContext)

	// ExitOpt_fetch_direction is called when exiting the opt_fetch_direction production.
	ExitOpt_fetch_direction(c *Opt_fetch_directionContext)

	// ExitStmt_move is called when exiting the stmt_move production.
	ExitStmt_move(c *Stmt_moveContext)

	// ExitStmt_close is called when exiting the stmt_close production.
	ExitStmt_close(c *Stmt_closeContext)

	// ExitStmt_null is called when exiting the stmt_null production.
	ExitStmt_null(c *Stmt_nullContext)

	// ExitStmt_commit is called when exiting the stmt_commit production.
	ExitStmt_commit(c *Stmt_commitContext)

	// ExitStmt_rollback is called when exiting the stmt_rollback production.
	ExitStmt_rollback(c *Stmt_rollbackContext)

	// ExitPlsql_opt_transaction_chain is called when exiting the plsql_opt_transaction_chain production.
	ExitPlsql_opt_transaction_chain(c *Plsql_opt_transaction_chainContext)

	// ExitStmt_set is called when exiting the stmt_set production.
	ExitStmt_set(c *Stmt_setContext)

	// ExitCursor_variable is called when exiting the cursor_variable production.
	ExitCursor_variable(c *Cursor_variableContext)

	// ExitException_sect is called when exiting the exception_sect production.
	ExitException_sect(c *Exception_sectContext)

	// ExitProc_exceptions is called when exiting the proc_exceptions production.
	ExitProc_exceptions(c *Proc_exceptionsContext)

	// ExitProc_exception is called when exiting the proc_exception production.
	ExitProc_exception(c *Proc_exceptionContext)

	// ExitProc_conditions is called when exiting the proc_conditions production.
	ExitProc_conditions(c *Proc_conditionsContext)

	// ExitProc_condition is called when exiting the proc_condition production.
	ExitProc_condition(c *Proc_conditionContext)

	// ExitOpt_block_label is called when exiting the opt_block_label production.
	ExitOpt_block_label(c *Opt_block_labelContext)

	// ExitOpt_loop_label is called when exiting the opt_loop_label production.
	ExitOpt_loop_label(c *Opt_loop_labelContext)

	// ExitOpt_label is called when exiting the opt_label production.
	ExitOpt_label(c *Opt_labelContext)

	// ExitOpt_exitcond is called when exiting the opt_exitcond production.
	ExitOpt_exitcond(c *Opt_exitcondContext)

	// ExitAny_identifier is called when exiting the any_identifier production.
	ExitAny_identifier(c *Any_identifierContext)

	// ExitPlsql_unreserved_keyword is called when exiting the plsql_unreserved_keyword production.
	ExitPlsql_unreserved_keyword(c *Plsql_unreserved_keywordContext)

	// ExitSql_expression is called when exiting the sql_expression production.
	ExitSql_expression(c *Sql_expressionContext)

	// ExitExpr_until_then is called when exiting the expr_until_then production.
	ExitExpr_until_then(c *Expr_until_thenContext)

	// ExitExpr_until_semi is called when exiting the expr_until_semi production.
	ExitExpr_until_semi(c *Expr_until_semiContext)

	// ExitExpr_until_rightbracket is called when exiting the expr_until_rightbracket production.
	ExitExpr_until_rightbracket(c *Expr_until_rightbracketContext)

	// ExitExpr_until_loop is called when exiting the expr_until_loop production.
	ExitExpr_until_loop(c *Expr_until_loopContext)

	// ExitMake_execsql_stmt is called when exiting the make_execsql_stmt production.
	ExitMake_execsql_stmt(c *Make_execsql_stmtContext)

	// ExitOpt_returning_clause_into is called when exiting the opt_returning_clause_into production.
	ExitOpt_returning_clause_into(c *Opt_returning_clause_intoContext)
}
