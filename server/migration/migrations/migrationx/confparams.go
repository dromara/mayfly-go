package migrationx

// DbmsConfParams DbmsConfig 配置项定义（v1 JSON schema），init 建配置与 v1.12 升级 params 共用
const DbmsConfParams = `{"version":1,"cols":2,"fields":[
	{"prop":"querySqlSave","label":"system.sysconf.recordQuerySql","type":"select","options":[{"label":"true","value":"true"},{"label":"false","value":"false"}]},
	{"prop":"sqlExecTl","label":"system.sysconf.sqlExecLimt"},
	{"prop":"maxResultSet","label":"system.sysconf.maxResultSet"},
	{"prop":"maskEnabled","label":"system.sysconf.maskEnabled","type":"select","options":[{"label":"true","value":"true"},{"label":"false","value":"false"}],"tooltip":"system.sysconf.maskEnabledPlaceholder"},
	{"prop":"maskFailClosed","label":"system.sysconf.maskFailClosed","type":"select","options":[{"label":"true","value":"true"},{"label":"false","value":"false"}],"tooltip":"system.sysconf.maskFailClosedPlaceholder"},
	{"prop":"maskExemptRoleIds","label":"system.sysconf.maskExemptRoleIds","tooltip":"system.sysconf.maskExemptRoleIdsPlaceholder"}
]}`
