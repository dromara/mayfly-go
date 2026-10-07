<template>
    <div>
        <auto-form-drawer
            ref="drawerRef"
            v-model:visible="dialogVisible"
            :title="title"
            :items="items"
            :data="editData"
            size="40%"
            :confirm-api="onConfirm"
            @opened="onOpened"
            @submitted="emit('cancel')"
            @cancel="emit('cancel')"
        >
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="onTestConn" :loading="testConnBtnLoading" type="success">{{ $t('ac.testConn') }}</el-button>
                    <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="drawerRef?.submitting" @click="drawerRef?.submit()">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>

            <!-- 关联标签 -->
            <template #tagCodePaths="{ form }">
                <TagTreeSelect multiple :code="form.code" v-model="form.tagCodePaths" />
            </template>

            <!-- SSH 隧道 -->
            <template #sshTunnelMachineId="{ form }">
                <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { useAutoFormModel } from '@/hooks/useAutoFormModel';
import { useSshTunnelTransform } from '@/hooks/useResourceForm';
import { computed, useTemplateRef, type PropType } from 'vue';
import { AutoFormDrawer, defineFormItems } from '@/components/auto-form';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import { mongoApi } from './api';
import type { Mongo } from './types';

/** Mongo 编辑表单类型 (允许 null 的字段重定义) */
interface MongoForm extends Omit<Partial<Mongo>, 'id' | 'name' | 'uri' | 'sshTunnelMachineId'> {
    id?: number | null;
    name?: string | null;
    uri?: string | null;
    sshTunnelMachineId?: number | null;
    tagCodePaths?: string[];
}

const props = defineProps({
    data: {
        type: Object as PropType<Mongo | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits(['cancel', 'val-change']);

/**
 * 表单声明。
 *
 * uri 的必填与否按新建/修改分开：接口不再回传明文连接串，编辑时只能留空表示保持不变，
 * 若仍标为必填会逼用户去猜一个看不到的值；新建时则必须真实给出。
 */
const items = computed(() =>
    defineFormItems<MongoForm>([
        { prop: 'tagCodePaths', label: 'tag.relateTag', required: true },
        { prop: 'name', label: 'common.name', required: true },
        {
            prop: 'uri',
            label: 'uri',
            type: 'textarea',
            rows: 2,
            required: !props.data,
            placeholder: props.data ? '' : 'mongodb://username:password@host1:port1',
            tooltip: props.data ? 'mongo.uriKeepTip' : undefined,
        },
        { prop: 'sshTunnelMachineId', label: 'machine.sshTunnel' },
    ])
);

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => Promise<unknown>; submitting: boolean; submit: () => Promise<void> }>('drawerRef');

/** 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成） */
const editData = computed<MongoForm>(() => {
    if (!props.data) {
        return { tagCodePaths: [] };
    }
    // 连接串不回显：接口给出的是脱敏值，原样提交会把真实凭证换成 ****
    return { ...props.data, uri: '' };
});

// 宿主内部表单在 @opened 接管（提交组装基于它）
const { onOpened, requireForm } = useAutoFormModel<MongoForm>();

const submitForm = useSshTunnelTransform(computed(requireForm));

const { isFetching: testConnBtnLoading, execute: testConnExec } = mongoApi.testConn.useApi();
const { execute: saveMongoExec } = mongoApi.saveMongo.useApi();

const onTestConn = async () => {
    const valid = await useI18nFormValidate(drawerRef).catch(() => false);
    if (valid === false) return;
    await testConnExec(submitForm.value);
    Msg.success('ac.connSuccess');
};

// confirmApi 提交动作（从接管的内部表单读取提交数据）；成功提示与关闭抽屉由组件内置逻辑处理
const onConfirm = async () => {
    await saveMongoExec(submitForm.value);
    emit('val-change', requireForm());
};
</script>
<style lang="scss"></style>
