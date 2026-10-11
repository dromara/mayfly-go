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

            <div>
                <el-button @click="emit('save')" type="primary" icon="document-add" plain size="small">{{ $t('db.saveSql') }}</el-button>
            </div>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { Msg } from '@/hooks/useI18n';
import { toRefs } from 'vue';

const props = defineProps<{
    token: string;
    uploadUrl: string;
    uploadFn: (options: { file: File }) => { abort: () => void };
    /** 是否以原子批次执行多条语句（由父级编辑器保持状态，工具条只负责开关与展示） */
    atomic: boolean;
}>();

const { atomic } = toRefs(props);

const emit = defineEmits<{
    run: [];
    runAll: [];
    review: [];
    format: [];
    save: [];
    'update:atomic': [value: boolean];
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
