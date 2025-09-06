<template>
    <div class="card !p-2">
        <el-row :gutter="5">
            <el-col :span="4">
                <el-input :placeholder="$t('docker.imageName')" v-model="params.name" plain clearable></el-input>
            </el-col>

            <el-col :span="4">
                <EnumSelect v-model="params.state" :enums="ImageStateEnum" :placeholder="$t('docker.status')" clearable />
            </el-col>

            <el-col :span="4">
                <div class="flex items-center gap-2">
                    <el-button @click="getImages" type="primary" icon="refresh" circle plain></el-button>
                    <el-upload :on-success="uploadSuccess" action="" :http-request="uploadImage" :headers="{ token }" :show-file-list="false" name="file">
                        <el-button type="primary" icon="upload" circle plain></el-button>
                    </el-upload>
                </div>
            </el-col>
        </el-row>
    </div>

    <el-table :data="filterTableDatas" v-loading="state.loadingImages">
        <el-table-column prop="id" label="ID" :min-width="100" show-overflow-tooltip>
            <template #default="{ row }">
                <el-link type="primary" :underline="false">
                    {{ row.id.split(':')[1].substring(0, 12) }}
                </el-link>
            </template>
        </el-table-column>

        <el-table-column prop="tags" :label="$t('docker.tag')" :min-width="250">
            <template #default="{ row }">
                <el-tag v-for="tag in row.tags" :key="tag" type="primary">{{ tag || '-' }}</el-tag>
            </template>
        </el-table-column>

        <el-table-column prop="size" :label="$t('docker.size')" :min-width="60">
            <template #default="{ row }">
                {{ formatByteSize(row.size) }}
            </template>
        </el-table-column>

        <el-table-column prop="createTime" :label="$t('common.createTime')" width="160">
            <template #default="scope">
                {{ formatDate(scope.row.createTime) }}
            </template>
        </el-table-column>

        <el-table-column prop="isUse" :label="$t('common.status')" :min-width="50">
            <template #default="{ row }">
                <EnumTag :enums="ImageStateEnum" :value="row.isUse" />
            </template>
        </el-table-column>

        <el-table-column :label="$t('common.operation')" width="130">
            <template #default="{ row }">
                <el-button @click="exportImage(row)" type="warning" link plain>{{ $t('docker.export') }}</el-button>

                <el-popconfirm :title="$t('docker.stopImageConfirm')" @confirm="imageRemove(row)" width="170">
                    <template #reference>
                        <el-button :disabled="row.isUse == ImageStateEnum.Used.value" type="danger" link plain>
                            {{ $t('common.delete') }}
                        </el-button>
                    </template>
                </el-popconfirm>
            </template>
        </el-table-column>
    </el-table>

    <el-dialog
        v-if="terminalDialog.visible"
        :title="terminalDialog.title"
        v-model="terminalDialog.visible"
        width="80%"
        :close-on-click-modal="false"
        :modal="false"
        @close="closeTerminal"
        draggable
        append-to-body
    >
        <TerminalBody ref="terminal" :socket-url="getDockerExecSocketUrl(props.id, terminalDialog.containerId)" height="560px" />
    </el-dialog>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, toRefs } from 'vue';
import { dockerApi, getDockerExecSocketUrl } from '../api';
import { formatByteSize, formatDate } from '@/common/utils/format';
import EnumSelect from '@/components/enumselect/EnumSelect.vue';
import { ImageStateEnum } from '../enums';
import EnumTag from '@/components/enumtag/EnumTag.vue';
import { fuzzyMatchField } from '@/common/utils/string';
import TerminalBody from '@/components/terminal/TerminalBody.vue';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';
import { getToken } from '@/common/utils/storage';
import { ElMessage } from 'element-plus';
import { i18n } from '@/i18n';

const props = defineProps({
    id: {
        type: Number,
        default: '',
    },
});

const state = reactive({
    params: {
        id: 0,
        name: '',
        state: null,
    },
    loadingImages: false,
    images: [],
    terminalDialog: {
        visible: false,
        title: '',
        containerId: '',
    },
});

const { params, terminalDialog } = toRefs(state);

const token = getToken();

onMounted(() => {
    getImages();
});

const filterTableDatas = computed(() => {
    let tables: any = state.images;
    const nameSearch = state.params.name;
    const stateSearch = state.params.state;

    if (stateSearch != null) {
        tables = tables.filter((table: any) => {
            return table.isUse === stateSearch;
        });
    }

    if (nameSearch) {
        tables = fuzzyMatchField(nameSearch, tables, (table: any) => table.tags[0]);
    }

    return tables;
});

const getImages = async () => {
    if (!props.id) {
        return;
    }
    state.params.id = props.id;
    state.loadingImages = true;
    try {
        state.images = await dockerApi.images.request(state.params);
    } finally {
        state.loadingImages = false;
    }
};

const exportImage = async (row: any) => {
    const a = document.createElement('a');
    a.setAttribute('href', `${config.baseApiUrl}/docker/${props.id}/images/save?id=${props.id}&tag=${row.tags[0]}&${joinClientParams()}`);
    a.setAttribute('target', '_blank');
    a.click();
};

const uploadImage = (content: any) => {
    const params = new FormData();
    // const path = state.nowPath;
    params.append('file', content.file);
    params.append('id', props.id + '');
    params.append('token', token);
    dockerApi.imageUpload
        .xhrReq(params, {
            headers: { 'Content-Type': 'multipart/form-data; boundary=----WebKitFormBoundaryF1uyUD0tWdqmJqpl' },
            // onUploadProgress: onUploadProgress,
            timeout: 3 * 60 * 60 * 1000,
        })
        .then(() => {
            ElMessage.success(i18n.global.t('machine.uploadSuccess'));
            setTimeout(() => {
                getImages();
            }, 3000);
        })
        .catch(() => {
            // state.uploadProgressShow = false;
        });
    ElMessage.info(i18n.global.t('docker.imageUploading'));
};

const uploadSuccess = (res: any) => {
    if (res.code !== 200) {
        ElMessage.error(res.msg);
    }
};

const imageRemove = async (row: any) => {
    await dockerApi.imageRemove.request({ id: props.id, imageId: row.id });
    getImages();
};

const openTerminal = (row: any) => {
    state.terminalDialog.containerId = row.containerId;
    state.terminalDialog.title = `Terminal - ${row.name}`;
    state.terminalDialog.visible = true;
};

const closeTerminal = () => {
    state.terminalDialog.visible = false;
};
</script>
