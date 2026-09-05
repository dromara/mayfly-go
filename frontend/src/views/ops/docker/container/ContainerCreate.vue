<template>
    <auto-form-drawer v-model:visible="dialogVisible" :items="items" :data="editData" size="40%" :confirm-loading="createLoading" scroll-to-error @confirm="btnOk" @opened="onOpened" @cancel="emit('cancel')">
        <!-- 镜像（镜像列表 allow-create） -->
        <template #image="{ form }">
            <el-select v-model="form.image" filterable allow-create>
                <el-option v-for="item in state.images" :key="item.id" :label="item.tags[0]" :value="item.tags[0]"></el-option>
            </el-select>
        </template>

        <!-- 端口映射 -->
        <template #exposedPorts="{ form }">
            <el-card class="w-full">
                <el-table v-if="form.exposedPorts.length !== 0" :data="form.exposedPorts">
                    <el-table-column :label="$t('docker.server')" min-width="100">
                        <template #default="{ row }">
                            <el-input-number v-model="row.hostPort" :min="10000" :max="20000" />
                        </template>
                    </el-table-column>

                    <el-table-column :label="$t('docker.container')" min-width="100">
                        <template #default="{ row }">
                            <el-input v-model="row.containerPort" />
                        </template>
                    </el-table-column>

                    <el-table-column :label="$t('docker.protocol')" min-width="50">
                        <template #default="{ row }">
                            <el-select v-model="row.protocol" style="width: 100%" :placeholder="$t('container.serverExample')">
                                <el-option label="tcp" value="tcp" />
                                <el-option label="udp" value="udp" />
                            </el-select>
                        </template>
                    </el-table-column>

                    <el-table-column min-width="35">
                        <template #default="scope">
                            <el-button link type="primary" @click="handlePortsDelete(scope.$index)">
                                {{ $t('common.delete') }}
                            </el-button>
                        </template>
                    </el-table-column>
                </el-table>

                <el-button class="ml-1 mt-1" size="small" @click="handlePortsAdd()">
                    {{ $t('common.add') }}
                </el-button>
            </el-card>
        </template>

        <!-- 挂载卷 -->
        <template #volumes="{ form }">
            <el-card class="mb-1 w-full">
                <el-table v-if="form.volumes.length !== 0" :data="form.volumes">
                    <el-table-column :label="$t('docker.hostDir')" min-width="120">
                        <template #default="{ row }">
                            <el-input v-model="row.hostDir" />
                        </template>
                    </el-table-column>

                    <el-table-column :label="$t('docker.permission')" :width="100">
                        <template #default="{ row }">
                            <el-select v-model="row.mode">
                                <el-option value="rw" :label="$t('docker.rw')" />
                                <el-option value="ro" :label="$t('docker.ro')" />
                            </el-select>
                        </template>
                    </el-table-column>

                    <el-table-column :label="$t('docker.containerDir')" min-width="120">
                        <template #default="{ row }">
                            <el-input v-model="row.containerDir" />
                        </template>
                    </el-table-column>

                    <el-table-column min-width="40">
                        <template #default="scope">
                            <el-button link type="primary" @click="handleVolumesDelete(scope.$index)">
                                {{ $t('common.delete') }}
                            </el-button>
                        </template>
                    </el-table-column>
                </el-table>

                <el-button @click="handleVolumesAdd()" size="small">
                    {{ $t('common.add') }}
                </el-button>
            </el-card>
        </template>

        <!-- 其他选项 -->
        <template #otherOptions="{ form }">
            <div>
                <el-checkbox v-model="form.tty">{{ $t('docker.tty') }}</el-checkbox>
                <el-checkbox v-model="form.openStdin">{{ $t('docker.openStdin') }}</el-checkbox>
                <el-checkbox v-model="form.privileged">{{ $t('docker.privileged') }}</el-checkbox>
            </div>
        </template>

        <!-- CPU 配额 -->
        <template #nanoCpus="{ form }">
            <el-input v-model.number="form.nanoCpus">
                <template #append>
                    <div style="width: 35px">{{ $t('docker.core') }}</div>
                </template>
            </el-input>
        </template>

        <!-- 内存限制 -->
        <template #memory="{ form }">
            <el-input v-model.number="form.memory">
                <template #append><div style="width: 35px">GB</div></template>
            </el-input>
        </template>

        <!-- 共享内存 -->
        <template #shmSize="{ form }">
            <el-input v-model.number="form.shmSize">
                <template #append><div style="width: 35px">GB</div></template>
            </el-input>
        </template>

        <!-- 设备 -->
        <template #devices="{ form }">
            <el-card class="mb-1 w-full">
                <el-table v-if="form.devices.length !== 0" :data="form.devices">
                    <el-table-column :label="$t('docker.driver')" min-width="100">
                        <template #header>
                            {{ $t('docker.driver') }}
                            <el-tooltip :content="$t('docker.driverTips')" placement="top">
                                <SvgIcon class="ml-2 mb-2" name="question-filled" />
                            </el-tooltip>
                        </template>
                        <template #default="{ row }">
                            <el-select v-model="row.driver" filterable allow-create>
                                <el-option v-for="item in runtimeSelect" :key="item" :label="item" :value="item"></el-option>
                            </el-select>
                        </template>
                    </el-table-column>

                    <el-table-column :label="$t('docker.count')" :width="100">
                        <template #default="{ row }">
                            <el-input v-model.number="row.count" />
                        </template>
                    </el-table-column>

                    <el-table-column :label="$t('docker.capabilitie')" min-width="100">
                        <template #default="{ row }">
                            <el-input-tag v-model="row.capabilities" :placeholder="$t('docker.capabilitiePlaceholder')" />
                        </template>
                    </el-table-column>

                    <el-table-column :label="$t('docker.deviceId')" min-width="100">
                        <template #default="{ row }">
                            <el-input-tag v-model="row.deviceIds" />
                        </template>
                    </el-table-column>

                    <el-table-column min-width="40">
                        <template #default="scope">
                            <el-button class="mt-1" link type="primary" @click="handleDevicesDelete(scope.$index)">
                                {{ $t('common.delete') }}
                            </el-button>
                        </template>
                    </el-table-column>
                </el-table>

                <el-button @click="handleDevicesAdd()" size="small">
                    {{ $t('common.add') }}
                </el-button>
            </el-card>
        </template>
    </auto-form-drawer>
</template>
<script setup lang="ts">
import { Rules } from '@/common/rule';
import { formatByteSize } from '@/common/utils/format';
import { deepClone } from '@/common/utils/object';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import SvgIcon from '@/components/svg-icon/index.vue';
import { Msg } from '@/hooks/useI18n';
import { computed, reactive, ref, watch, type PropType } from 'vue';
import { dockerApi } from '../api';
import type { DockerImageItem } from '../types';

const props = defineProps({
    id: {
        type: Number,
        required: true,
    },
});

const defaultForm = {
    name: '',
    image: '',
    cmdStr: '',
    forcePull: false,
    exposedPorts: [] as { hostPort: number; containerPort: string; protocol: string }[],
    networkMode: 'default',
    volumes: [] as { hostDir: string; mode: string; containerDir: string }[],
    devices: [] as { driver: string; count: number; device: string }[],
    capAdd: [] as string[],
    tty: false,
    openStdin: false,
    privileged: false,
    restartPolicy: '',
    cpuShares: 1024,
    nanoCpus: 0,
    memory: 0,
    shmSize: 0,
    labelsStr: '',
    envsStr: '',
};

const state = reactive({
    dockerInfo: {} as Record<string, unknown>,
    images: [] as DockerImageItem[],
});

const submitForm = ref({} as Record<string, unknown>);

//定义事件
const emit = defineEmits(['cancel', 'success']);

const dialogVisible = defineModel<boolean>('visible', { default: false });

/** 表单声明（computed：labelParams 需随 dockerInfo 动态计算；复杂表格编辑器走 custom 插槽） */
const items = computed<AutoFormItem[]>(() => [
    { prop: 'name', label: 'common.name', required: true },
    { prop: 'image', label: 'docker.image', type: 'custom', tooltip: 'docker.imageTips', rules: [Rules.requiredSelect('docker.image')] },
    { prop: 'forcePull', label: 'docker.forcePull', type: 'switch', tooltip: 'docker.forcePullTips' },
    { prop: 'cmdStr', label: 'Command' },
    { prop: 'exposedPorts', label: 'docker.port', type: 'custom' },
    { prop: 'volumes', label: 'docker.mount', type: 'custom' },
    {
        prop: 'networkMode',
        label: 'docker.networkMode',
        type: 'select',
        props: { filterable: true, allowCreate: true },
        options: [
            { label: 'default', value: 'default' },
            { label: 'host', value: 'host' },
            { label: 'bridge', value: 'bridge' },
            { label: 'none', value: 'none' },
        ],
    },
    { prop: 'otherOptions', label: 'docker.otherOption', type: 'custom' },
    {
        prop: 'restartPolicy',
        label: 'docker.restartPolicy',
        type: 'radio',
        options: [
            { label: 'docker.noRestart', value: 'no' },
            { label: 'docker.alwaysRestart', value: 'always' },
            { label: 'docker.onFailure', value: 'on-failure' },
            { label: 'docker.unlessStopped', value: 'unless-stopped' },
        ],
    },
    { prop: 'cpuShares', label: 'docker.cpuShare', type: 'number', tooltip: 'docker.cpuShareTips' },
    { prop: 'nanoCpus', label: 'docker.cpuQuota', type: 'custom', tooltip: 'docker.cpuLimitTips', labelParams: { cpuTotal: state.dockerInfo.NCPU } },
    {
        prop: 'memory',
        label: 'docker.memoryLimit',
        type: 'custom',
        tooltip: 'docker.memoryLimitTips',
        labelParams: { memTotal: formatByteSize(Number(state.dockerInfo.MemTotal)) },
    },
    { prop: 'shmSize', label: 'docker.shmSize', type: 'custom' },
    { prop: 'devices', label: 'docker.device', type: 'custom' },
    { prop: 'capAdd', label: 'capAdd', type: 'tags' },
    { prop: 'labelsStr', label: 'docker.tag', type: 'textarea', rows: 3, placeholder: 'docker.tagTips' },
    { prop: 'envsStr', label: 'docker.envParam', type: 'textarea', rows: 3, placeholder: 'docker.envParamTips' },
]);

/** 传给 AutoFormDrawer 的回填数据（新建态默认值，深拷贝由组件内部完成） */
const editData = deepClone(defaultForm) as unknown as AutoFormData;

/** 抽屉打开后暂存的内部表单引用（端口/挂载卷/设备表格编辑与提交组装基于它） */
const internalForm = ref<AutoFormData>({});

const onOpened = (form: AutoFormData) => {
    internalForm.value = form;
};

const { isFetching: createLoading, execute: createExec } = dockerApi.containerCreate.useApi(submitForm);

watch(dialogVisible, async (val) => {
    if (val) {
        init();
    }
});

const runtimeSelect = computed(() => {
    return state.dockerInfo ? Object.keys((state.dockerInfo?.Runtimes as Record<string, unknown>) ?? {}) : [];
});

const init = async () => {
    submitForm.value = {};
    dockerApi.info.request({ id: props.id }).then((res) => {
        state.dockerInfo = res;
    });
    state.images = await dockerApi.images.request({ id: props.id });
};

const handlePortsAdd = () => {
    let item = {
        containerPort: '',
        hostPort: 0,
        protocol: 'tcp',
    };
    internalForm.value.exposedPorts.push(item);
};

const handlePortsDelete = (index: number) => {
    internalForm.value.exposedPorts.splice(index, 1);
};

const handleVolumesAdd = () => {
    let item = {
        hostDir: '',
        containerDir: '',
        mode: 'rw',
    };
    internalForm.value.volumes.push(item);
};

const handleVolumesDelete = (index: number) => {
    internalForm.value.volumes.splice(index, 1);
};

const handleDevicesAdd = () => {
    let item = {
        driver: '',
        count: 0,
        device: '',
    };
    internalForm.value.devices.push(item);
};

const handleDevicesDelete = (index: number) => {
    internalForm.value.devices.splice(index, 1);
};

// @confirm 触发前 AutoFormDrawer 已完成表单校验
const btnOk = async (rawForm: AutoFormData) => {
    const form = rawForm as typeof defaultForm;
    submitForm.value = { ...form };
    submitForm.value.id = props.id;

    if (submitForm.value.exposedPorts) {
        submitForm.value.exposedPorts = form.exposedPorts.map((item) => {
            return {
                ...item,
                hostPort: item.hostPort + '', // 转为字符串
            };
        });
    }

    if (form.envsStr) {
        submitForm.value.envs = form.envsStr.split('\n');
    }
    if (form.labelsStr) {
        submitForm.value.labels = form.labelsStr.split('\n');
    }
    if (form.cmdStr) {
        let itemCmd = splitStringIgnoringQuotes(form.cmdStr);
        const cmds = [];
        for (const item of itemCmd) {
            cmds.push(item.replace(/(?<!\\)"/g, '').replaceAll('\\"', '"'));
        }
        submitForm.value.cmd = cmds;
    }
    await createExec();
    Msg.operateSuccess();
    emit('success', submitForm);
    dialogVisible.value = false;
    emit('cancel');
};

const splitStringIgnoringQuotes = (input: string) => {
    input = input.replace(/\\"/g, '<quota>');
    const regex = /"([^"]*)"|(\S+)/g;
    const result = [];
    let match;

    while ((match = regex.exec(input)) !== null) {
        if (match[1]) {
            result.push(match[1].replaceAll('<quota>', '\\"'));
        } else if (match[2]) {
            result.push(match[2].replaceAll('<quota>', '\\"'));
        }
    }

    return result;
};
</script>
