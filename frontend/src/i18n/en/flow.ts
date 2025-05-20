export default {
    flow: {
        // procdef
        approvalNode: 'Approval Node',
        procdef: 'Process Definition',
        triggeringCondition: 'Condition',
        triggeringConditionTips: 'go template syntax. If the output is 1, the approval process is triggered',
        conditionPlaceholder: 'Trigger condition, return value =1, means to trigger the approval process',
        conditionDefault: `{'{{'}/* DBMS- Run Sql rules The param parameter is described as follows */{'}}'}
{'{{'}/* stmtType: select / read / insert / update / delete / ddl ;  */{'}}'}
{'{{'} if eq .bizType "db_sql_exec_flow"{'}}'}
{'{{'}/* Enable process approval when select and read statements are not available */{'}}'}
    {'{{'} if and (ne .param.stmtType "select") (ne .param.stmtType "read") {'}}'}
        1
    {'{{'} end {'}}'}
{'{{'} end {'}}'}

{'{{'}/* Redis-Run Cmd rules;   param: parameter is described as follows */{'}}'}
{'{{'}/* cmdType: read(Read cmd) / write(Write cmd);  */{'}}'}
{'{{'}/* cmd: get/set/hset... */{'}}'}
{'{{'} if eq .bizType "redis_run_cmd_flow"{'}}'}
    {'{{'} if eq .param.cmdType "write" {'}}'}
        1
    {'{{'} end {'}}'}
{'{{'} end {'}}'}`,
        nodeName: 'Node Name',
        nodeNameTips: 'Click the specified node to drag and drop sort',
        auditor: 'Auditor',
        tasksNotEmpty: 'Please improve the task of the approval node',
        tasksNoComplete: 'Please complete the task information of the {index} th approval node',
        // procdef status enum
        enable: 'Enable',
        disable: 'Disable',

        todoTask: 'Pending Tasks',
        doneTask: 'Completed Tasks',
        flowDesign: 'Flow Design',
        clear: 'Clear',
        approvalMode: 'Approval Mode',
        andSign: 'All Approve (AND)',
        orSign: 'Any Approve (OR)',
        voteSign: 'Vote Approval',
        taskCandidate: 'Task Assignees',
        mustOneStartNode: 'There must be one start node in the flow',
        mustOneEndNode: 'There must be one end node in the flow',
        mustOneOutEdgeForStartNode: 'The start node must have at least one outgoing edge',
        mustOneInEdgeForEndNode: 'The end node must have at least one incoming edge',
        approvalRecord: 'Approval Records',
        start: 'Start',
        end: 'End',
        usertask: 'User Task', // 建议拼写修正为 userTask
        serial: 'Exclusive Gateway',
        parallel: 'Parallel Gateway',
        flowEdge: 'Sequence Flow',

        // procinst
        startProcess: 'Start Process',
        cancelProcessConfirm: 'Confirm canceling the process?',
        bizType: 'Business Type',
        bizKey: 'Business Key',
        initiator: 'Initiator',
        procdefName: 'Process Name',
        bizStatus: 'Business Status',
        startingTime: 'Starting Time',
        endTime: 'End Time',
        duration: 'Duration',
        proc: 'Process',
        bizInfo: 'Business Information',
        approvalNodeNotExist: 'There is no approval node',
        resourceNotExistFlow: 'This resource does not require an approval operation',
        procinstFormError: 'Please fill in the information correctly',
        procinstStartSuccess: 'Process initiated successfully',
        // db run sql flow biz
        runSql: 'Run SQL',
        selectDbPlaceholder: 'Please select database',
        // redis run cmd flow biz
        runCmd: 'Rum Cmd',
        selectRedisPlaceholder: 'Select the Redis instance and db',
        cmdPlaceholder: `For example: SET 'key' 'value'; Multiple commands; segmentation`,
        // ProcinstStatusEnum
        active: 'Active',
        completed: 'Completed',
        suspended: 'Suspended',
        terminated: 'Terminated',
        cancelled: 'Cancelled',
        handleResult: 'Result of handling',
        runResult: 'Result of execution',
        // ProcinstBizStatus
        waitHandle: 'Pending',
        handleSuccess: 'Success',
        handleFail: 'Fail',
        noHandle: 'No processing',
        // ProcinstTaskStatus
        waitProcess: 'Waiting',
        pass: 'Pass',
        reject: 'Reject',
        back: 'Back',
        canceled: 'Canceled',
        // FlowBizType
        dbSqlExec: 'DBMS-Run SQL',
        redisRunCmd: 'Redis-Run Cmd',

        // task
        approveNode: 'Approval Node',
        approveForm: 'Approval Form',
        approveResult: 'Approval Result',
        approvalRemark: 'Approval Comments',
        approver: 'Approver',
        audit: 'Audit',
        procinstStatus: 'Process Status',
        taskStatus: 'Task Status',
        taskName: 'Task Name',
        taskBeginTime: 'Start Time',
        flowAudit: 'Process Audit',
        notify: 'Notify',
    },
};
