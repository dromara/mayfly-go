export default {
    home: {
        // 欢迎区
        greetingDawn: '凌晨好',
        greetingMorning: '早上好',
        greetingNoon: '中午好',
        greetingAfternoon: '下午好',
        greetingEvening: '晚上好',
        greetingName: '{greeting}，{name}',
        heroSubtitle: '这里是你的运维工作台，随时掌握资源与待办动态',
        lastLogin: '上次登录',
        today: '今天',
        yesterday: '昨天',
        justNow: '刚刚',
        minutesAgo: '{n} 分钟前',
        refresh: '刷新',
        personalCenter: '个人中心',

        // 指标卡
        pendingTasks: '待办审批',
        activeAlerts: '活跃告警',
        unreadMessages: '未读消息',
        resourceTotal: '资源总数',
        pendingTasksHint: '等待你处理的审批工单',
        activeAlertsHint: '尚未恢复的告警事件',
        unreadMessagesHint: '系统与告警来源的通知',
        resourceTotalHint: '已纳管的运维资源',
        loadFailed: '加载失败',
        retry: '重试',
        navigateFailed: '目标页面暂不可用，请从左侧菜单进入',

        // 工作台面板
        todoTasks: '待办审批',
        noTodo: '暂无待办审批',
        noTodoDesc: '新的审批工单会出现在这里',
        noAlert: '暂无活跃告警',
        noAlertDesc: '所有资源指标都在阈值范围内',
        taskFallback: '工单 #{id}',
        viewAll: '查看全部',
        viewAction: '查看',
        openResource: '打开',
        metricValue: '{metric}：{value}',

        // 资源概览
        resourceOverview: '资源概览',
        resourceSubtitle: '点击卡片可查看资源列表',
        myMachines: '机器',
        myDatabases: '数据库',
        myRedis: 'Redis',
        myMongo: 'MongoDB',
        myEs: 'Elasticsearch',
        myMilvus: 'Milvus',
        myContainers: '容器',
        myKafka: 'Kafka',

        // 快捷操作
        quickActions: '快捷操作',
        quickActionsDesc: '常用入口一步直达',
        goMyResource: '我的资源',
        goMyResourceDesc: '按标签树浏览全部资源',
        goAlerts: '告警中心',
        goAlertsDesc: '查看与处置全部告警事件',
        goAiAssistant: 'AI 助手',
        goAiAssistantDesc: '用自然语言完成运维操作',
        goFlow: '工单流程',
        goFlowDesc: '审批与流程实例跟踪',
        noQuickAction: '暂无可用的快捷入口',
        noQuickActionDesc: '当前账号没有开通对应菜单权限',

        // 最近操作
        recentOperations: '最近操作',
        noOpRecord: '暂无操作记录',

        // 加载提示
        loadMore: '滚动加载更多...',
        loadedAll: '已加载全部',
    },
};
