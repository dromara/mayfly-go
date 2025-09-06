<template>
    <el-drawer v-model="dialogVisible" :append-to-body="true" :destroy-on-close="true" :close-on-click-modal="false" :before-close="cancel" size="40%">
        <template #header>
            <DrawerHeader :header="$t('docker.createContainer')" :back="cancel">
                <template #extra>
                    <div class="mr20"></div>
                </template>
            </DrawerHeader>
        </template>

        <el-form :model="form" ref="formRef" label-position="top" :rules="rules" scroll-to-error>
            <el-form-item prop="name" :label="$t('common.name')" clearable>
                <el-input v-model.trim="form.name" auto-complete="off"></el-input>
            </el-form-item>

            <el-form-item prop="image" :label="$t('docker.image')">
                <template #label>
                    {{ $t('docker.image') }}
                    <el-tooltip :content="$t('docker.imageTips')" placement="top">
                        <SvgIcon class="mb-1" name="question-filled" />
                    </el-tooltip>
                </template>

                <el-select v-model="form.image" filterable allow-create>
                    <el-option v-for="item in state.images" :key="item.id" :label="item.tags[0]" :value="item.tags[0]"></el-option>
                </el-select>
            </el-form-item>

            <el-form-item>
                <el-checkbox v-model="form.forcePull">{{ $t('docker.forcePull') }}</el-checkbox>
                <el-tooltip :content="$t('docker.forcePullTips')" placement="top">
                    <SvgIcon class="ml-2" name="question-filled" />
                </el-tooltip>
            </el-form-item>

            <el-form-item prop="cmdStr" :label="$t('Command')">
                <el-input v-model="form.cmdStr" />
            </el-form-item>

            <el-form-item :label="$t('docker.port')">
                <el-card class="w-full">
                    <el-table v-if="form.exposedPorts.length !== 0" :data="form.exposedPorts">
                        <el-table-column :label="$t('docker.server')" min-width="100">
                            <template #default="{ row }">
                                <el-input-number v-model="row.hostPort" :min="10000" :max="20000" />
                                <!-- <el-input v-model="row.hostPort" :placeholder="$t('docker.hostPortPlaceholder')" /> -->
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
            </el-form-item>

            <el-form-item prop="mount" :label="$t('docker.mount')">
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
            </el-form-item>

            <el-form-item :label="$t('docker.networkMode')">
                <el-select v-model="form.networkMode" filterable allow-create>
                    <el-option label="default" value="default"></el-option>
                    <el-option label="host" value="host"></el-option>
                    <el-option label="bridge" value="bridge"></el-option>
                    <el-option label="none" value="none"></el-option>
                </el-select>
            </el-form-item>

            <el-form-item :label="$t('docker.otherOption')">
                <el-checkbox v-model="form.tty">{{ $t('docker.tty') }}</el-checkbox>
                <el-checkbox v-model="form.openStdin">
                    {{ $t('docker.openStdin') }}
                </el-checkbox>

                <el-checkbox v-model="form.privileged">
                    {{ $t('docker.privileged') }}
                </el-checkbox>
            </el-form-item>

            <el-form-item :label="$t('docker.restartPolicy')" prop="restartPolicy">
                <el-radio-group v-model="form.restartPolicy">
                    <el-radio value="no">{{ $t('docker.noRestart') }}</el-radio>
                    <el-radio value="always">{{ $t('docker.alwaysRestart') }}</el-radio>
                    <el-radio value="on-failure">{{ $t('docker.onFailure') }}</el-radio>
                    <el-radio value="unless-stopped">{{ $t('docker.unlessStopped') }}</el-radio>
                </el-radio-group>
            </el-form-item>

            <el-form-item :label="$t('docker.cpuShare')" prop="cpuShares">
                <template #label>
                    <el-row>
                        {{ $t('docker.cpuShare') }}
                        <el-tooltip :content="$t('docker.cpuShareTips')" placement="top">
                            <SvgIcon class="ml-2" name="question-filled" />
                        </el-tooltip>
                    </el-row>
                </template>
                <el-input v-model.number="form.cpuShares" />
            </el-form-item>

            <el-form-item prop="nanoCPUs">
                <template #label>
                    <el-row>
                        {{ $t('docker.cpuQuota') }}
                        <el-tooltip :content="$t('docker.cpuLimitTips')" placement="top">
                            <SvgIcon class="ml-2" name="question-filled" />
                        </el-tooltip>
                        <el-text class="ml-2" size="small">{{ $t('docker.cpuCanUseTips', { cpuTotal: dockerInfo.NCPU }) }}</el-text>
                    </el-row>
                </template>

                <el-input v-model.number="form.nanoCpus">
                    <template #append>
                        <div style="width: 35px">{{ $t('docker.core') }}</div>
                    </template>
                </el-input>
            </el-form-item>

            <el-form-item :label="$t('docker.memoryLimit')" prop="memory">
                <template #label>
                    <el-row>
                        {{ $t('docker.memoryLimit') }}
                        <el-tooltip :content="$t('docker.memoryLimitTips')" placement="top">
                            <SvgIcon class="ml-2" name="question-filled" />
                        </el-tooltip>
                        <el-text class="ml-2" size="small">{{ $t('docker.memoryCanUseTips', { memTotal: formatByteSize(dockerInfo.MemTotal) }) }}</el-text>
                    </el-row>
                </template>

                <el-input v-model.number="form.memory">
                    <template #append><div style="width: 35px">GB</div></template>
                </el-input>
            </el-form-item>

            <el-form-item :label="$t('docker.shmSize')" prop="memory">
                <el-input v-model.number="form.shmSize">
                    <template #append><div style="width: 35px">GB</div></template>
                </el-input>
            </el-form-item>

            <el-form-item prop="device" :label="$t('docker.device')">
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

                        <el-table-column min-width="35">
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
            </el-form-item>

            <el-form-item :label="$t('capAdd')" prop="capAdd">
                <el-input-tag v-model="form.capAdd" />
            </el-form-item>

            <el-form-item :label="$t('docker.tag')" prop="labelsStr">
                <el-input type="textarea" :placeholder="$t('docker.tagTips')" :rows="3" v-model="form.labelsStr" />
            </el-form-item>

            <el-form-item :label="$t('docker.envParam')" prop="envStr">
                <el-input type="textarea" :placeholder="$t('docker.envParamTips')" :rows="3" v-model="form.envsStr" />
            </el-form-item>
        </el-form>

        <template #footer>
            <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" :loading="createLoading" @click="btnOk">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-drawer>
</template>
<script setup lang="ts">
import { useI18nFormValidate, useI18nOperateSuccessMsg } from '@/hooks/useI18n';
import { computed, reactive, toRefs, useTemplateRef, watch } from 'vue';
import { dockerApi } from '../api';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { Rules } from '@/common/rule';
import { formatByteSize } from '@/common/utils/format';
import { deepClone } from '@/common/utils/object';

const rules = {
    name: [Rules.requiredInput('common.name')],
    image: [Rules.requiredSelect('docker.image')],
};

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
    exposedPorts: [] as any,
    networkMode: 'default',
    volumes: [] as any,
    devices: [] as any,
    capAdd: [] as any,
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
    dockerInfo: {} as any,
    images: [] as any,
    form: defaultForm,
    submitForm: {} as any,
    pwd: '',
});

const { dockerInfo, form, submitForm } = toRefs(state);

//定义事件
const emit = defineEmits(['cancel', 'success']);

const dialogVisible = defineModel<boolean>('visible', { default: false });

const formRef = useTemplateRef('formRef');

const { isFetching: createLoading, execute: createExec } = dockerApi.containerCreate.useApi(submitForm);

// onMounted(async () => {
//     init();
// });

watch(dialogVisible, async (val) => {
    if (val) {
        init();
    }
});

const runtimeSelect = computed(() => {
    return state.dockerInfo ? Object.keys(state.dockerInfo?.Runtimes) : [];
});

const init = async () => {
    state.form = deepClone(defaultForm);
    state.submitForm = {};
    dockerApi.info.request({ id: props.id }).then((res) => {
        state.dockerInfo = res;
    });
    state.images = await dockerApi.images.request({ id: props.id });
};

const handlePortsAdd = () => {
    let item = {
        host: '',
        hostIP: '',
        containerPort: '',
        hostPort: '',
        protocol: 'tcp',
    };
    state.form.exposedPorts.push(item);
};

const handlePortsDelete = (index: number) => {
    state.form.exposedPorts.splice(index, 1);
};

const handleVolumesAdd = () => {
    let item = {
        hostDir: '',
        containerDir: '',
        mode: 'rw',
    };
    state.form.volumes.push(item);
};

const handleVolumesDelete = (index: number) => {
    state.form.volumes.splice(index, 1);
};

const handleDevicesAdd = () => {
    let item = {
        count: 0,
    };
    state.form.devices.push(item);
};

const handleDevicesDelete = (index: number) => {
    state.form.devices.splice(index, 1);
};

const btnOk = async () => {
    await useI18nFormValidate(formRef);

    state.submitForm = { ...state.form };
    state.submitForm.id = props.id;

    if (state.submitForm.exposedPorts) {
        state.submitForm.exposedPorts = state.form.exposedPorts.map((item: any) => {
            return {
                ...item,
                hostPort: item.hostPort + '', // 转为字符串
            };
        });
    }

    if (state.form.envsStr) {
        state.submitForm.envs = state.form.envsStr.split('\n');
    }
    if (state.form.labelsStr) {
        state.submitForm.labels = state.form.labelsStr.split('\n');
    }
    if (state.form.cmdStr) {
        let itemCmd = splitStringIgnoringQuotes(state.form.cmdStr);
        const cmds = [];
        for (const item of itemCmd) {
            cmds.push(item.replace(/(?<!\\)"/g, '').replaceAll('\\"', '"'));
        }
        state.submitForm.cmd = cmds;
    }
    await createExec();
    useI18nOperateSuccessMsg();
    emit('success', submitForm);
    cancel();
};

const cancel = () => {
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
