<template>
    <div class="card !p-2">
        <el-row justify="space-between">
            <el-col :span="16">
                <el-row :gutter="5">
                    <el-col :span="6">
                        <el-input :placeholder="$t('docker.containerName')" v-model="params.name" plain clearable></el-input>
                    </el-col>
                    <el-col :span="6">
                        <EnumSelect v-model="params.state" :enums="ContainerStateEnum" :placeholder="$t('docker.status')" clearable />
                    </el-col>
                    <el-col :span="8">
                        <el-button @click="getContainers" type="primary" icon="refresh" circle plain></el-button>
                    </el-col>
                </el-row>
            </el-col>

            <el-col :span="8">
                <el-row justify="end">
                    <el-button @click="openContainerCreate" type="success" icon="plus" plain>{{ $t('docker.createContainer') }}</el-button>
                </el-row>
            </el-col>
        </el-row>
    </div>

    <el-table :data="filterTableDatas" v-loading="state.loadingContainers">
        <el-table-column prop="name" :label="$t('docker.name')" :min-width="120" show-overflow-tooltip> </el-table-column>
        <el-table-column prop="imageName" :label="$t('docker.image')" :min-width="150" show-overflow-tooltip> </el-table-column>

        <el-table-column prop="state" :label="$t('common.status')" :min-width="110">
            <template #default="{ row }">
                <el-dropdown @command="handleCommand">
                    <el-button :type="EnumValue.getEnumByValue(ContainerStateEnum, row.state)?.tag.type" round plain size="small">
                        {{ $t(EnumValue.getLabelByValue(ContainerStateEnum, row.state)) || '-' }}
                        <SvgIcon class="ml-1" :name="EnumValue.getEnumByValue(ContainerStateEnum, row.state)?.extra.icon" />
                    </el-button>

                    <template #dropdown>
                        <el-dropdown-menu>
                            <el-dropdown-item :command="{ type: 'restart', row }">
                                {{ $t('docker.restart') }}
                            </el-dropdown-item>

                            <el-dropdown-item :disabled="row.state == ContainerStateEnum.Stop.value" :command="{ type: 'stop', row }">
                                {{ $t('docker.stop') }}
                            </el-dropdown-item>
                        </el-dropdown-menu>
                    </template>
                </el-dropdown>
            </template>
        </el-table-column>

        <el-table-column v-loading="true" prop="stats" :label="$t('docker.stats')" :min-width="130">
            <template #default="{ row }">
                <SvgIcon v-if="getLoadingState(row.containerId)" class="is-loading" name="loading" color="var(--el-color-primary)" />

                <span v-else-if="row.stats">
                    <el-row>
                        <el-text size="small" class="font11">
                            CPU:
                            <span>{{ row.stats.cpuPercent.toFixed(2) }}%</span>
                        </el-text>
                    </el-row>

                    <el-row>
                        <el-text size="small" class="font11">
                            {{ $t('docker.memory') }}:
                            <span>{{ row.stats.memoryPercent.toFixed(2) }}%</span>
                        </el-text>

                        <el-popover placement="right" :width="300" trigger="hover">
                            <template #reference>
                                <SvgIcon class="mt5 ml5" color="var(--el-color-primary)" name="MoreFilled" />
                            </template>

                            <el-row>
                                <el-col :span="12">
                                    <el-statistic :title="$t('CPU使用')" :value="row.stats.cpuTotalUsage" :formatter="formatCpuValue" :precision="2">
                                    </el-statistic>
                                </el-col>
                                <el-col :span="12">
                                    <el-statistic :title="$t('CPU总计')" :value="row.stats.systemUsage" :formatter="formatCpuValue" :precision="2">
                                    </el-statistic>
                                </el-col>
                            </el-row>

                            <el-row>
                                <el-col :span="12">
                                    <el-statistic :title="$t('内存使用')" :value="row.stats.memoryUsage" :formatter="formatByteSize" :precision="2">
                                    </el-statistic>
                                </el-col>
                                <el-col :span="12">
                                    <el-statistic :title="$t('内存限额')" :value="row.stats.memoryLimit" :formatter="formatByteSize" :precision="2">
                                    </el-statistic>
                                </el-col>
                            </el-row>
                        </el-popover>
                    </el-row>
                </span>

                <span v-else>-</span>
            </template>
        </el-table-column>

        <el-table-column prop="networks" :label="$t('docker.ip')" :min-width="90">
            <template #default="scope">
                <el-tag v-for="network in scope.row.networks" :key="network" type="primary">{{ network || '-' }}</el-tag>
            </template>
        </el-table-column>

        <el-table-column prop="ports" :label="$t('machine.port')" :min-width="160">
            <template #default="scope">
                <el-tag v-for="port in scope.row.ports" :key="port" type="primary">{{ port }}</el-tag>
            </template>
        </el-table-column>

        <el-table-column prop="createTime" :label="$t('common.createTime')" width="160">
            <template #default="scope">
                {{ formatDate(scope.row.createTime) }}
            </template>
        </el-table-column>

        <el-table-column prop="status" label="运行时长" :min-width="120"> </el-table-column>

        <el-table-column :label="$t('common.operation')" :min-width="180">
            <template #default="{ row }">
                <el-row>
                    <el-button @click="openTerminal(row)" :disabled="row.state != ContainerStateEnum.Running.value" type="primary" link plain> SSH </el-button>

                    <el-button @click="openLog(row)" type="success" link plain>{{ $t('docker.log') }}</el-button>

                    <el-dropdown @command="handleCommand">
                        <el-button type="primary" link plain class="ml-3"> {{ $t('common.more') }} <SvgIcon name="arrow-down" :size="12" /> </el-button>

                        <template #dropdown>
                            <el-dropdown-menu>
                                <el-dropdown-item :command="{ type: 'remove', row }">
                                    {{ $t('common.delete') }}
                                </el-dropdown-item>
                            </el-dropdown-menu>
                        </template>
                    </el-dropdown>
                </el-row>
            </template>
        </el-table-column>
    </el-table>

    <el-dialog
        v-if="terminalDialog.visible"
        :title="terminalDialog.title"
        v-model="terminalDialog.visible"
        width="80%"
        body-class="h-[65vh]"
        :close-on-click-modal="false"
        :modal="false"
        @close="closeTerminal"
        draggable
        append-to-body
    >
        <TerminalBody ref="terminal" :socket-url="getDockerExecSocketUrl(props.id, terminalDialog.containerId)" />
    </el-dialog>

    <ContainerLog v-model:visible="logDialog.visible" :id="props.id" :container-id="logDialog.containerId" :title="logDialog.title" />

    <ContainerCreate v-model:visible="containerCreateDialog.visible" :id="props.id" @success="getContainers" />
</template>

<script lang="ts" setup>
import { computed, defineAsyncComponent, onMounted, reactive, toRefs, watch } from 'vue';
import { dockerApi, getDockerExecSocketUrl } from '../api';
import { formatByteSize, formatDate } from '@/common/utils/format';
import EnumSelect from '@/components/enumselect/EnumSelect.vue';
import { ContainerStateEnum } from '../enums';
import { fuzzyMatchField } from '@/common/utils/string';
import TerminalBody from '@/components/terminal/TerminalBody.vue';
import { useI18nConfirm, useI18nDeleteSuccessMsg, useI18nOperateSuccessMsg } from '@/hooks/useI18n';
import SvgIcon from '@/components/svgIcon/index.vue';
import { useDataState } from '@/hooks/useDataState';
import EnumValue from '@/common/Enum';

const ContainerLog = defineAsyncComponent(() => import('./ContainerLog.vue'));
const ContainerCreate = defineAsyncComponent(() => import('./ContainerCreate.vue'));

const props = defineProps({
    id: {
        type: Number,
        default: 0,
    },
});

const state = reactive({
    params: {
        id: props.id,
        name: '',
        state: null,
    },
    loadingContainers: false,
    containers: [],
    terminalDialog: {
        visible: false,
        title: '',
        containerId: '',
    },
    logDialog: {
        visible: false,
        title: '',
        containerId: '',
    },
    containerCreateDialog: {
        visible: false,
        title: '',
        containerId: '',
    },
});

const { params, terminalDialog, logDialog, containerCreateDialog } = toRefs(state);

// 容器状态加载状态，key -> containerId, value -> loading
const { setState: setLoadingState, getState: getLoadingState } = useDataState<string, boolean>();

onMounted(() => {
    getContainers();
});

watch(
    () => props.id,
    () => {
        getContainers();
    }
);

const filterTableDatas = computed(() => {
    let tables: any = state.containers;
    const nameSearch = state.params.name;
    const stateSearch = state.params.state;

    if (stateSearch) {
        tables = tables.filter((table: any) => {
            return table.state === stateSearch;
        });
    }

    if (nameSearch) {
        tables = fuzzyMatchField(nameSearch, tables, (table: any) => table.name);
    }

    return tables;
});

const getContainers = async () => {
    if (!props.id) {
        return;
    }
    state.params.id = props.id;
    state.loadingContainers = true;
    try {
        state.containers = await dockerApi.containers.request(state.params);
        setContainersStats();
    } finally {
        state.loadingContainers = false;
    }
};

const setContainersStats = () => {
    if (state.containers.length === 0) {
        return;
    }

    state.containers.forEach((container: any) => {
        if (container.state === ContainerStateEnum.Running.value) {
            setLoadingState(container.containerId, true);
        }
    });

    dockerApi.containersStats
        .request(state.params)
        .then((res) => {
            state.containers.forEach((container: any) => {
                const stats = res.find((stat: any) => stat.containerId === container.containerId);
                if (stats) {
                    container.stats = stats;
                }
            });
        })
        .finally(() => {
            state.containers.forEach((container: any) => {
                if (container.state === ContainerStateEnum.Running.value) {
                    setLoadingState(container.containerId, false);
                }
            });
        });
};

const containerRestart = async (param: any) => {
    await dockerApi.containerRestart.request({ id: props.id, containerId: param.containerId });
    useI18nOperateSuccessMsg();
    getContainers();
};

const containerStop = async (param: any) => {
    await useI18nConfirm('docker.stopContainerConfirm', { name: param.name });
    await dockerApi.containerStop.request({ id: props.id, containerId: param.containerId });
    useI18nOperateSuccessMsg();
    getContainers();
};

const containerRemove = async (param: any) => {
    await useI18nConfirm('docker.removeContainerConfirm', { name: param.name });
    await dockerApi.containerRemove.request({ id: props.id, containerId: param.containerId });
    useI18nDeleteSuccessMsg();
    getContainers();
};

const openTerminal = (row: any) => {
    state.terminalDialog.containerId = row.containerId;
    state.terminalDialog.title = `Terminal - ${row.name}`;
    state.terminalDialog.visible = true;
};

const closeTerminal = () => {
    state.terminalDialog.visible = false;
};

const openLog = (row: any) => {
    state.logDialog.containerId = row.containerId;
    state.logDialog.title = `Log - ${row.name}`;
    state.logDialog.visible = true;
};

const handleCommand = async (commond: any) => {
    const row = commond.row;
    const type = commond.type;
    switch (type) {
        case 'restart': {
            containerRestart({ containerId: row.containerId, name: row.name });
            return;
        }
        case 'stop': {
            containerStop({ containerId: row.containerId, name: row.name });
            return;
        }
        case 'remove': {
            containerRemove({ containerId: row.containerId, name: row.name });
            return;
        }
    }
};

const openContainerCreate = () => {
    state.containerCreateDialog.visible = true;
};

function formatCpuValue(t: number) {
    const num = 1000;
    if (t < num) return t + ' ns';
    if (t < Math.pow(num, 2)) return Number((t / num).toFixed(2)) + ' μs';
    if (t < Math.pow(num, 3)) return Number((t / Math.pow(num, 2)).toFixed(2)) + ' ms';
    return Number((t / Math.pow(num, 3)).toFixed(2)) + ' s';
}
</script>
