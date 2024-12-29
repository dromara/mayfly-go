export default {
    flow: {
        // procdef
        approvalNode: '审批节点',
        procdef: '流程定义',
        triggeringCondition: '触发条件',
        triggeringConditionTips: 'go template语法。若输出结果为1，则表示触发该审批流程',
        conditionPlaceholder: '触发条件, 返回值=1, 则表示触发该审批流程',
        conditionDefault: `{{/* DBMS-执行sql规则;  param参数描述如下 */}}
{{/* stmtType: select / read / insert / update / delete / ddl ;  */}}
{{ if eq .bizType "db_sql_exec_flow"}}
   {{/* 不是select和read语句时，开启流程审批 */}}
   {{ if and (ne .param.stmtType "select") (ne .param.stmtType "read") }}
       1
   {{ end }}
{{ end }}

{{/* Redis-执行命令规则;   param参数描述如下 */}}
{{/* cmdType: read(读命令) / write(写命令);  */}}
{{/* cmd: get/set/hset...等 */}}
{{ if eq .bizType "redis_run_cmd_flow"}}
   {{ if eq .param.cmdType "write" }}
       1
   {{ end }}
{{ end }}`,
        nodeName: '节点名称',
        nodeNameTips: '点击指定节点可进行拖拽排序',
        auditor: '审核人员',
        tasksNotEmpty: '请完善审批节点任务',
        tasksNoComplete: '请完善第{index}个审批节点任务信息',
        // procdef status enum
        enable: '启用',
        disable: '禁用',

        // procinst
        startProcess: '发起流程',
        cancelProcessConfirm: '确认取消该流程?',
        bizType: '业务类型',
        bizKey: '业务Key',
        initiator: '发起人',
        procdefName: '流程名',
        bizStatus: '业务状态',
        startingTime: '发起时间',
        endTime: '结束时间',
        duration: '持续时间',
        proc: '流程',
        bizInfo: '业务信息',
        approvalNodeNotExist: '不存在审批节点',
        resourceNotExistFlow: '该资源无需审批操作',
        procinstFormError: '请正确填写信息',
        procinstStartSuccess: '流程发起成功',
        // db run sql flow biz
        runSql: '执行SQL',
        selectDbPlaceholder: '请选择数据库',
        // redis run cmd flow biz
        runCmd: '执行Cmd',
        selectRedisPlaceholder: '请选择Redis实例与库',
        cmdPlaceholder: `如: SET 'key' 'value'; 多条命令;分割`,
        // ProcinstStatusEnum
        active: '执行中',
        completed: '完成',
        suspended: '挂起',
        terminated: '终止',
        cancelled: '取消',
        handleResult: '处理结果',
        runResult: '执行结果',
        // ProcinstBizStatus
        waitHandle: '待处理',
        handleSuccess: '处理成功',
        handleFail: '处理失败',
        noHandle: '不处理',
        // ProcinstTaskStatus
        waitProcess: '待处理',
        pass: '通过',
        reject: '拒绝',
        back: '回退',
        canceled: '取消',
        // FlowBizType
        dbSqlExec: 'DBMS-执行SQL',
        redisRunCmd: 'Redis-执行命令',

        // task
        approveNode: '审批节点',
        approveForm: '审批表单',
        approveResult: '审批结果',
        audit: '审核',
        procinstStatus: '流程状态',
        taskStatus: '任务状态',
        taskName: '当前节点',
        taskBeginTime: '开始时间',
        flowAudit: '流程审批',
    },
};
