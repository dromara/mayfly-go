<template>
    <div>
        <el-dialog v-model="visible" width="760px" :title="$t('mongo.commandConsole')" :before-close="onClose" destroy-on-close>
            <auto-form v-model="state" :items="runCmdItems" label-width="auto">
                <template #cmdName>
                    <el-select v-model="state.cmdName" class="w-full!" filterable :placeholder="$t('mongo.cmdTemplatePlaceholder')" @change="onPickTemplate">
                        <el-option
                            v-for="item in catalog"
                            :key="item.name"
                            :label="`${item.name} | ${item.descKey ? $t(item.descKey) : ''}`"
                            :value="item.name"
                        >
                            <div class="flex items-center justify-between gap-3">
                                <span>{{ item.name }}</span>
                                <span class="flex items-center gap-2">
                                    <el-tag v-if="item.needConfirm" size="small" type="danger">{{ levelText(item.level) }}</el-tag>
                                    <span class="text-[12px] text-gray-400">{{ item.descKey ? $t(item.descKey) : '' }}</span>
                                </span>
                            </div>
                        </el-option>
                    </el-select>
                </template>

                <template #db>
                    <!--
                        allow-create 是降级路径：库列表要 listDatabases 权限（未认证的实例必被拒），
                        拿不到列表时也要能直接输入库名执行命令，而不是被一个下拉框卡住。
                    -->
                    <el-select
                        v-model="state.db"
                        class="w-full!"
                        filterable
                        allow-create
                        default-first-option
                        :loading="dbLoading"
                        :placeholder="dbError ? $t('mongo.dbInputPlaceholder') : $t('mongo.db')"
                    >
                        <el-option v-for="item in databases" :key="item.name" :label="item.name" :value="item.name" />
                    </el-select>
                    <!--
                        提示必须只占一行：它挂在「数据库」这一栅格列下（约 200px 宽），
                        整句说明换行会把弹窗顶部撑出一大块空白。原因与服务端原文收进 tooltip，
                        首次失败也已经 toast 过一条，这里只留「该怎么办」。
                    -->
                    <div v-if="dbError" class="flex min-w-0 items-center gap-1 text-[12px] leading-4 text-yellow-700">
                        <el-tooltip :content="$t('mongo.dbListFailedTip')" placement="bottom">
                            <span class="truncate">{{ $t('mongo.dbListFailed') }}</span>
                        </el-tooltip>
                        <el-button class="shrink-0" link type="primary" :loading="dbLoading" @click="loadDatabases">
                            {{ $t('common.refresh') }}
                        </el-button>
                    </div>
                </template>

                <template #runBtn>
                    <div class="flex items-center gap-2">
                        <el-button type="primary" :loading="running" @click="onRunCommand">{{ $t('mongo.run') }}</el-button>
                        <el-tooltip effect="dark" placement="top">
                            <template #content>
                                {{ $t('mongo.moreCmdTips') }}
                                <a class="text-blue-400" href="https://www.mongodb.com/docs/manual/reference/command/" target="_blank">
                                    mongodb.com/docs/manual/reference/command
                                </a>
                            </template>
                            <span
                                ><el-icon><InfoFilled /></el-icon
                            ></span>
                        </el-tooltip>
                    </div>
                </template>

                <template #cmd>
                    <monaco-editor v-model="state.cmd" language="json" style="width: 100%" height="230px" />
                </template>

                <template #res>
                    <monaco-editor
                        v-model="state.cmdRes"
                        language="json"
                        :options="{ readOnly: true, minimap: { enabled: false } }"
                        style="width: 100%"
                        height="230px"
                    />
                </template>
            </auto-form>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { defineAsyncComponent, reactive, ref, toRefs, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { mongoApi } from './api';
import { loadCommandCatalog } from './command/catalog';
import type { MongoCommandSpec, MongoDatabase } from './types';

const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

const { t } = useI18n();

const props = defineProps<{ id: number }>();

const visible = defineModel<boolean>('visible', { default: false });

const state = reactive({
    cmdName: '',
    db: '',
    cmd: '',
    cmdRes: '',
    databases: [] as MongoDatabase[],
    catalog: [] as MongoCommandSpec[],
});

/** 库列表的加载态与错误：与命令目录分开，一方失败不牵连另一方 */
const dbLoading = ref(false);
const dbError = ref('');

const { databases, catalog } = toRefs(state);

const running = ref(false);

/** runCommand 表单声明（模板与库选择、运行按钮、命令与结果编辑器走自定义插槽） */
const runCmdItems: AutoFormItem[] = [
    { prop: 'cmdName', label: 'mongo.template', type: 'custom', span: 12 },
    { prop: 'db', label: 'mongo.db', type: 'custom', span: 8 },
    { prop: 'runBtn', type: 'custom', span: 4 },
    { prop: 'cmd', label: 'mongo.command', type: 'custom' },
    { prop: 'res', label: 'mongo.result', type: 'custom' },
];

async function loadDatabases() {
    dbLoading.value = true;
    dbError.value = '';
    try {
        state.databases = (await mongoApi.databases.request({ id: props.id })) ?? [];
        if (!state.db) {
            state.db = state.databases[0]?.name ?? '';
        }
    } catch (e) {
        // 请求层已 toast 服务端原文（未认证实例上就是「需要认证」那句），这里只把下拉降级成可手输
        state.databases = [];
        dbError.value = e instanceof Error ? e.message : '';
    } finally {
        dbLoading.value = false;
    }
}

watch(visible, async (open) => {
    if (!open) {
        return;
    }
    // 目录与库列表各自加载：目录按实例缓存、库列表每次都要刷新（别人可能刚建了库）。
    // 用 Promise.all 会让权限不足时本来可用的命令目录一起变空，整个弹窗看起来是坏的。
    void loadDatabases();
    try {
        state.catalog = await loadCommandCatalog(props.id);
    } catch (e) {
        state.catalog = [];
        Msg.error('mongo.catalogLoadFailed');
    }
});

function levelText(level: string): string {
    const keys: Record<string, string> = {
        read: 'mongo.levelRead',
        dataSave: 'mongo.levelDataSave',
        dataDel: 'mongo.levelDataDel',
        structSave: 'mongo.levelStructSave',
        structDel: 'mongo.levelStructDel',
        admin: 'mongo.levelAdmin',
    };
    return t(keys[level] ?? 'mongo.levelAdmin');
}

function onPickTemplate(name: string) {
    const spec = state.catalog.find((item) => item.name === name);
    state.cmd = spec?.template ? JSON.stringify(JSON.parse(spec.template), null, 4) : '';
    state.cmdRes = '';
}

function findSpec(command: Record<string, unknown>): MongoCommandSpec | undefined {
    const [name] = Object.keys(command);
    return state.catalog.find((item) => item.name === name);
}

async function onRunCommand() {
    state.cmdRes = '';

    let command: unknown;
    try {
        command = JSON.parse(state.cmd);
    } catch {
        Msg.error('mongo.commandInvalid');
        return;
    }
    // 命令必须是文档对象：数组与标量会被后端拒绝，在这里拦下能直接说明原因
    if (command === null || typeof command !== 'object' || Array.isArray(command)) {
        Msg.error('mongo.commandInvalid');
        return;
    }

    // 确认口径来自后端下发的 needConfirm，前端不再自己维护危险命令名单。
    // 手工输入且不在目录里的命令不做前端确认：确认属于体验，安全边界始终是后端的分级鉴权。
    const spec = findSpec(command as Record<string, unknown>);
    if (spec?.needConfirm) {
        if (!(await useI18nConfirm('mongo.dangerousCmdConfirm', { command: spec.name, level: levelText(spec.level) }))) {
            // 取消或关掉弹窗：不继续后续操作
            return;
        }
    }

    running.value = true;
    try {
        const res = await mongoApi.runCommand.request({
            id: props.id,
            database: state.db,
            // 原样提交解析结果：字段顺序即用户书写顺序，后端按序编码为 BSON，
            // 复合索引键与多字段命令因此不再被 map 迭代序打乱
            command,
        });
        state.cmdRes = JSON.stringify(res ?? {}, null, 4);
        Msg.success('mongo.runSuccess');
    } finally {
        running.value = false;
    }
}

function onClose() {
    visible.value = false;
    state.cmdName = '';
    state.db = '';
    state.cmd = '';
    state.cmdRes = '';
    state.databases = [];
    dbError.value = '';
}
</script>

<style lang="scss" scoped></style>
