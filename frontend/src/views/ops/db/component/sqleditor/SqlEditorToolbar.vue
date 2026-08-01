<template>
    <div class="flex-shrink-0">
        <div class="card p-1! flex items-center justify-between">
            <div>
                <el-link @click="emit('run')" underline="never" class="ml-3.5" icon="VideoPlay"> </el-link>
                <el-divider direction="vertical" border-style="dashed" />

                <el-tooltip :show-after="1000" class="box-item" effect="dark" content="format sql" placement="top">
                    <el-link @click="emit('format')" type="primary" underline="never" icon="MagicStick"> </el-link>
                </el-tooltip>
                <el-divider direction="vertical" border-style="dashed" />

                <el-tooltip :show-after="1000" class="box-item" effect="dark" content="commit" placement="top">
                    <el-link @click="emit('commit')" type="success" underline="never" icon="CircleCheck"> </el-link>
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

const props = defineProps<{
    token: string;
    uploadUrl: string;
    uploadFn: (options: { file: File }) => { abort: () => void };
}>();

const emit = defineEmits<{
    run: [];
    format: [];
    commit: [];
    save: [];
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
