<template>
    <div class="flex-shrink-0">
        <div class="card p-1! flex items-center justify-between">
            <div>
                <!-- 执行类动作归堆：执行 / 执行全部 / 原子执行开关（原子只作用于批量执行，故同组） -->
                <el-tooltip :show-after="1000" class="box-item" effect="dark" :content="$t('db.runSql')" placement="top">
                    <el-link @click="emit('run')" type="primary" underline="never" class="ml-3.5" icon="VideoPlay"> </el-link>
                </el-tooltip>
                <el-divider direction="vertical" border-style="dashed" />

                <el-tooltip :show-after="1000" class="box-item" effect="dark" :content="$t('db.runAllSql')" placement="top">
                    <el-link @click="emit('runAll')" type="primary" underline="never" icon="DArrowRight"> </el-link>
                </el-tooltip>
                <el-divider direction="vertical" border-style="dashed" />

                <!-- 原子执行：整批全部成功才提交，任一条失败即整批回滚（后端在同一请求内共用一个事务） -->
                <el-tooltip :show-after="1000" class="box-item" effect="dark" :content="$t('db.atomicExecTip')" placement="top">
                    <el-switch
                        :model-value="atomic"
                        class="ml-1"
                        size="small"
                        :active-text="$t('db.atomicExec')"
                        @update:model-value="emit('update:atomic', $event === true)"
                    />
                </el-tooltip>
                <el-divider direction="vertical" border-style="dashed" />

                <!-- 查看执行计划：不执行、纯预览（含 SELECT 等不会弹执行确认框的语句） -->
                <el-tooltip :show-after="1000" class="box-item" effect="dark" :content="$t('db.viewPlan')" placement="top">
                    <el-link @click="emit('review')" type="warning" underline="never" icon="Search"> </el-link>
                </el-tooltip>
                <el-divider direction="vertical" border-style="dashed" />

                <!-- DDL 预检：不执行，只读表规模与主键，说明「批的是什么」 -->
                <el-tooltip :show-after="1000" class="box-item" effect="dark" :content="$t('db.ddlPrecheck')" placement="top">
                    <el-link @click="emit('precheck')" type="danger" underline="never" icon="WarnTriangleFilled"> </el-link>
                </el-tooltip>
                <el-divider direction="vertical" border-style="dashed" />

                <el-tooltip :show-after="1000" class="box-item" effect="dark" content="format sql" placement="top">
                    <el-link @click="emit('format')" type="primary" underline="never" icon="MagicStick"> </el-link>
                </el-tooltip>
                <el-divider direction="vertical" border-style="dashed" />

                <el-upload
                    class="sql-file-exec"
                    :before-upload="beforeUpload"
                    :on-success="execSqlFileSuccess"
                    :http-request="handleSqlFileUpload"
                    :headers="{ Authorization: token }"
                    :action="uploadUrl"
                    :show-file-list="false"
                    name="file"
                    multiple
                    :limit="100"
                >
                    <el-tooltip :show-after="1000" class="box-item" effect="dark" :content="$t('db.sqlScriptRun')" placement="top">
                        <el-link v-auth="'db:sqlscript:run'" type="success" underline="never" icon="Document"></el-link>
                    </el-tooltip>
                </el-upload>
            </div>

            <div class="flex items-center gap-2">
                <!-- AI 入口：功能型入口带文字，不与纯图标动作混排。
                     工具条只留这一个 AI 入口：选中 SQL 不再浮出动作条，生成 / 改写 / 优化 / 解释
                     都在打开的面板里选（右键菜单与 ⌘/Ctrl+K 仍保留为直达加速器）；
                     「注释补全」是计费门禁但属 AI 偏好，收在面板底部，不常驻工具条 -->
                <el-tooltip v-if="aiEnabled" :show-after="800" effect="dark" :content="$t('db.draft.entryTip', { key: modKey })" placement="top">
                    <el-button v-auth="'db:sqlscript:run'" @click="emit('openSqlAssistant')" icon="Cpu" plain size="small">
                        {{ $t('db.draft.entry') }}
                    </el-button>
                </el-tooltip>
                <el-button @click="emit('save')" type="primary" icon="document-add" plain size="small">{{ $t('db.saveSql') }}</el-button>
            </div>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { Msg } from '@/hooks/useI18n';
import { computed, toRefs } from 'vue';
import { currentModKeyLabel } from '../sql-ai/shortcut';

const props = defineProps<{
    token: string;
    uploadUrl: string;
    uploadFn: (options: { file: File }) => { abort: () => void };
    /** 是否以原子批次执行多条语句（由父级编辑器保持状态，工具条只负责开关与展示） */
    atomic: boolean;
    /** AI 是否可用（模型已配置）；false 时本组件的 AI 入口都不渲染 */
    aiEnabled: boolean;
}>();

const { atomic } = toRefs(props);

// 入口 tooltip 里的快捷键前缀随平台变（mac 显示 ⌘K，其余显示 Ctrl+K）
const modKey = computed(() => currentModKeyLabel());

const emit = defineEmits<{
    run: [];
    runAll: [];
    review: [];
    format: [];
    save: [];
    'update:atomic': [value: boolean];
    precheck: [];
    openSqlAssistant: [];
}>();

const beforeUpload = (file: File) => {
    Msg.success('db.scriptFileUploadRunning', { filename: file.name });
};

const handleSqlFileUpload = (options: { file: File }) => {
    return props.uploadFn(options);
};

const execSqlFileSuccess = (res: { code: number; msg: string }) => {
    if (res.code !== 200) {
        Msg.error(res.msg);
    }
};
</script>
