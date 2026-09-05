<template>
    <div>
        <auto-form-drawer ref="drawerRef" v-model:visible="dialogVisible" :title="title" :items="items" :data="editData" size="40%" :confirm-api="onConfirm" @submitted="emit('cancel')" @opened="onOpened" @cancel="emit('cancel')">
            <!-- 关联标签（自定义插槽） -->
            <template #tagCodePaths="{ form }">
                <TagTreeSelect multiple :code="form.code" v-model="form.tagCodePaths" />
            </template>
            <!-- DB 多选（支持手输库号，逗号拼接回 form.db；cluster 模式禁用） -->
            <template #db="{ form }">
                <el-select :model-value="dbList" :disabled="form.mode == 'cluster'" multiple allow-create filterable class="w-full!" @update:model-value="onDbListChange">
                    <el-option v-for="db in DB_OPTIONS" :key="db" :label="db" :value="db" />
                </el-select>
            </template>
            <!-- SSH 隧道机器（自定义插槽） -->
            <template #sshTunnelMachineId="{ form }">
                <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
            </template>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="onTestConn" :loading="testConnBtnLoading" type="success">{{ $t('ac.testConn') }}</el-button>
                    <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="drawerRef?.submitting" @click="drawerRef?.submit()">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { Rules } from '@/common/rule';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { computed, ref, useTemplateRef, type PropType } from 'vue';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import { redisApi } from './api';
import type { Redis, RedisSaveForm } from './types';

const props = defineProps({
    redis: {
        type: Object as PropType<Redis | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits(['val-change', 'cancel']);

/** 可选 DB 列表 */
const DB_OPTIONS = Array.from({ length: 16 }, (_, i) => i);

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => Promise<unknown>; submitting: boolean; submit: () => Promise<void> }>('drawerRef');

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；sentinel 专属字段按 mode 条件显隐） */
const items = computed<AutoFormItem[]>(() => [
    { prop: 'tagCodePaths', label: 'tag.relateTag', required: true, slot: 'tagCodePaths' },
    { prop: 'name', label: 'common.name', required: true },
    { prop: 'mode', label: 'mode', type: 'select', required: true, options: [{ value: 'standalone', label: 'standalone' }, { value: 'cluster', label: 'cluster' }, { value: 'sentinel', label: 'sentinel' }] },
    { prop: 'host', label: 'host', type: 'textarea', rows: 2, required: true, placeholder: 'redis.hostTips' },
    { prop: 'username', label: 'common.username' },
    { prop: 'password', label: 'common.password', type: 'password', props: { autocomplete: 'new-password' } },
    { prop: 'redisNodePassword', label: 'redis.nodePassword', type: 'password', when: (f) => f.mode == 'sentinel', props: { autocomplete: 'new-password' } },
    { prop: 'db', label: 'DB', required: true, slot: 'db' },
    { prop: 'remark', label: 'common.remark', type: 'textarea' },
    { prop: 'sshTunnelMachineId', label: 'machine.sshTunnel', slot: 'sshTunnelMachineId' },
]);

/** 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成） */
const editData = computed<AutoFormData>(() => {
    const redis = props.redis as RedisSaveForm | false | undefined;
    if (redis) {
        return { ...redis } as AutoFormData;
    }
    return { db: '0', tagCodePaths: [] } as AutoFormData;
});

const dbList = ref<number[]>([0]);

/** 抽屉打开后暂存的内部表单引用（DB 互转与提交均基于它） */
const internalForm = ref<AutoFormData>({});

const onOpened = (form: AutoFormData) => {
    internalForm.value = form;
    if (props.redis) {
        convertDb((form.db as string) || '0');
    } else {
        dbList.value = [0];
    }
};

const { isFetching: testConnBtnLoading, execute: testConnExec } = redisApi.testConn.useApi();
const { execute: saveRedisExec } = redisApi.saveRedis.useApi();

const convertDb = (db: string) => {
    dbList.value = db.split(',').map((x) => Number.parseInt(x));
};

/**
 * 改变表单中的数据库字段，方便表单错误提示。如全部删光，可提示请添加库号
 */
const onDbListChange = (list: number[]) => {
    dbList.value = list;
    internalForm.value.db = list.length == 0 ? '' : list.join(',');
};

const getReqForm = () => {
    const reqForm = { ...internalForm.value } as RedisSaveForm;
    if (reqForm.mode == 'sentinel' && (reqForm.host ?? '').split('=').length != 2) {
        Msg.error('redis.sentinelHostErr');
        return;
    }
    if (!reqForm.sshTunnelMachineId || reqForm.sshTunnelMachineId <= 0) {
        reqForm.sshTunnelMachineId = -1;
    }
    return reqForm;
};

const onTestConn = async () => {
    await useI18nFormValidate(drawerRef);
    await testConnExec(getReqForm());
    Msg.success('ac.connSuccess');
};

// confirmApi 提交动作（组装内部表单为请求参数）；成功提示与关闭抽屉由组件内置逻辑处理
const onConfirm = async () => {
    const reqForm = getReqForm();
    if (!reqForm) {
        // sentinel 主从形态校验失败（getReqForm 内已 toast），保持抽屉打开
        return;
    }
    await saveRedisExec(reqForm);
    emit('val-change', internalForm.value);
};
</script>
<style lang="scss"></style>
