<template>
    <el-drawer
        size="50%"
        :destroy-on-close="false"
        :close-on-click-modal="false"
        :close-on-press-escape="false"
        class="es-op-temp h-full"
        v-model="visible"
        :title="t('es.templates')"
    >
        <el-auto-resizer>
            <template #default="{ height, width }">
                <el-space class="mb-3">
                    <el-input :placeholder="t('es.temp.filter')" v-model.trim="state.filterTableName" @input="onFilterTemplates" />
                    <el-button type="primary" @click="onAddTemplate" icon="plus">{{ t('common.add') }}</el-button>
                    <SvgIcon name="refresh" @click="fetchTemplates" :size="20" />
                    <el-dropdown :hide-on-click="false">
                        <SvgIcon name="setting" :size="20" />
                        <el-button link icon="setting" />

                        <template #dropdown>
                            <el-dropdown-menu>
                                <el-dropdown-item>
                                    <el-checkbox v-model="state.showHideTemps" @change="onSwitchShowHide">{{ t('es.temp.showHide') }}</el-checkbox>
                                </el-dropdown-item>
                            </el-dropdown-menu>
                        </template>
                    </el-dropdown>
                    <el-tooltip>
                        <template #content>
                            <pre>{{ t('es.temp.note') }}</pre>
                        </template>
                        <SvgIcon name="Warning" :size="20" />
                    </el-tooltip>
                    <el-text type="warning" size="small">{{ t('es.temp.versionAlert') }}</el-text>
                </el-space>

                <el-table :data="state.templates" :max-height="height - 40" stripe size="small">
                    <el-table-column prop="name" :label="t('es.temp.name')" />
                    <el-table-column prop="index_patterns" :label="t('es.temp.index_patterns')" />
                    <el-table-column prop="description" :label="t('es.temp.description')" />
                    <el-table-column :label="t('common.operation')" width="100px" align="center">
                        <template #default="scope">
                            <el-space>
                                <el-button link type="primary" size="small" @click="onViewTemplate(scope.row)">{{ t('common.detail') }}</el-button>
                                <el-button link type="danger" size="small" @click="onDelTemplate(scope.row.name)">{{ t('common.delete') }}</el-button>
                            </el-space>
                        </template>
                    </el-table-column>
                </el-table>
            </template>
        </el-auto-resizer>
    </el-drawer>

    <el-drawer
        size="50%"
        :destroy-on-close="true"
        :close-on-click-modal="false"
        :close-on-press-escape="false"
        class="es-add-temp h-full"
        v-model="state.addVisible"
        :title="state.formReadonly ? t('es.temp.view') : t('es.temp.addTemp')"
    >
        <el-auto-resizer>
            <template #default="{ height, width }">
                <el-form :model="form" ref="formRef" label-position="right" label-width="80">
                    <el-form-item :label="t('es.temp.name')" required prop="name">
                        <el-input v-model.trim="form.name" :disabled="state.formReadonly" />
                    </el-form-item>
                    <el-form-item :label="t('es.temp.priority')" required prop="priority">
                        <el-input-number v-model="form.priority" :disabled="state.formReadonly" />
                    </el-form-item>
                    <el-form-item :label="t('es.temp.index_patterns')" prop="index_patterns">
                        <el-select allow-create filterable multiple clearable v-model="form.index_patterns" :disabled="state.formReadonly"></el-select>
                    </el-form-item>
                    <el-form-item :label="t('es.temp.description')" required prop="description">
                        <el-input v-model.trim="form.description" :disabled="state.formReadonly" />
                    </el-form-item>
                    <el-form-item :label="t('es.temp.content')" required prop="template">
                        <monaco-editor
                            v-model="form.template"
                            language="json"
                            :height="height - 200 + 'px'"
                            :options="{ tabSize: 2, readOnly: state.formReadonly }"
                        />
                    </el-form-item>
                </el-form>
            </template>
        </el-auto-resizer>
        <template #footer>
            <el-button @click="state.addVisible = false">{{ t('common.close') }}</el-button>
            <el-button v-if="!state.formReadonly" type="primary" @click="doAddTemplate">{{ t('common.confirm') }}</el-button>
        </template>
    </el-drawer>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { esApi } from '@/views/ops/es/api';
import { nextTick, reactive, ref, unref, watch } from 'vue';
import { useI18nConfirm, useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nOperateSuccessMsg } from '@/hooks/useI18n';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import SvgIcon from '@/components/svgIcon/index.vue';

const { t } = useI18n();

const visible = defineModel<boolean>();

interface Props {
    instId: any;
    version: string;
}
const props = defineProps<Props>();

const formRef = ref();

const state = reactive({
    originTemplates: [] as any,
    templates: [] as any,
    showHideTemps: false,
    filterTableName: '',
    addVisible: false,
    formReadonly: false,
    form: {
        name: '',
        priority: 100,
        index_patterns: [],
        template: '',
        description: '',
    },

    // es 版本是否小于 7.8.0
    // 7.8之前的版本模板接口为_template，优先级字段为order,
    // 7.8之后的版本模板接口为_index_template，优先级字段为priority
    v: {
        oldVersion: false,
        api: '_index_template',
        priority: 'priority',
    },
});

const { form } = unref(state);

const getDefaultTemplate = () => {
    return {
        settings: {
            number_of_shards: 5,
            number_of_replicas: 1,
            blocks: {
                read_only: 'false',
            },
            max_result_window: '1000000',
            refresh_interval: '30s',
        },
        mappings: { properties: {} },
    };
};

watch(visible, async (x: any) => {
    if (x) {
        // 初始化状态
        state.filterTableName = '';
        state.originTemplates = [];
        state.templates = [];
        state.showHideTemps = false;

        state.v.oldVersion = isVersionBefore7_8_0(props.version);
        if (state.v.oldVersion) {
            state.v.api = '_template';
            state.v.priority = 'order';
        } else {
            state.v.api = '_index_template';
            state.v.priority = 'priority';
        }

        await nextTick(fetchTemplates);
    }
});

const isVersionBefore7_8_0 = (version: string) => {
    if (!version) {
        return false;
    }
    const v1 = version.split('.').map(Number);
    const v2 = [7, 8, 0]; // 比较目标版本

    for (let i = 0; i < 3; i++) {
        if (v1[i] < v2[i]) return true;
        if (v1[i] > v2[i]) return false;
    }
    return false; // 等于 7.8.0 时返回 false
};

const fetchTemplates = async () => {
    const data = await esApi.proxyReq('get', props.instId, `/${state.v.api}`);
    state.originTemplates = data.index_templates
        .map((a: any) => {
            return {
                name: a.name,
                priority: a.index_template.priority || 'NULL',
                index_patterns: JSON.stringify(a.index_template.index_patterns || '[]'),
                template: JSON.stringify(a.index_template.template || {}, null, 2),
                description: a.index_template._meta?.description || '',
            };
        })
        .sort((a: any, b: any) => a.name.localeCompare(b.name));

    onSwitchShowHide();
};

const onSwitchShowHide = () => {
    if (state.showHideTemps) {
        state.templates = state.originTemplates;
    } else {
        state.templates = state.originTemplates.filter((item: any) => item.name.indexOf('.') < 0);
    }
};
const onFilterTemplates = () => {
    onSwitchShowHide();

    let regx = createPattern(state.filterTableName);
    state.templates = state.templates.filter((item: any) => regx.test(item.name) || regx.test(item.description));
};

function createPattern(str: string): RegExp {
    const escaped = str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); // 转义特殊字符
    const pattern = [...escaped].join('.*');
    return new RegExp(`.*${pattern}.*`);
}

const onViewTemplate = async (data: any) => {
    state.addVisible = true;
    state.formReadonly = true;

    state.form.name = data.name;
    state.form.priority = data.priority;
    state.form.index_patterns = JSON.parse(data.index_patterns);
    state.form.template = data.template;
    state.form.description = data.description;
};

const onAddTemplate = () => {
    state.addVisible = true;
    state.formReadonly = false;

    state.form.name = '';
    state.form.priority = 100;
    state.form.index_patterns = [];
    state.form.template = JSON.stringify(getDefaultTemplate(), null, 2);
    state.form.description = '';
};

const doAddTemplate = async () => {
    await formRef.value.validate();
    let data = {
        index_patterns: state.form.index_patterns,
        [state.v.priority]: state.form.priority,
        template: JSON.parse(state.form.template),
        _meta: {
            description: state.form.description,
        },
    };
    await esApi.proxyReq('put', props.instId, `/${state.v.api}/${state.form.name}`, data);
    useI18nOperateSuccessMsg();

    setTimeout(async () => {
        state.addVisible = false;
        await fetchTemplates();
    }, 500);
};

const onDelTemplate = async (name: any) => {
    await useI18nDeleteConfirm(name);
    await useI18nConfirm('es.deleteTemplateConfirm', { name: name });
    await esApi.proxyReq('delete', props.instId, `/${state.v.api}/${name}`);
    useI18nDeleteSuccessMsg();

    setTimeout(async () => {
        await fetchTemplates();
    }, 500);
};
</script>

<style scoped lang="scss"></style>
