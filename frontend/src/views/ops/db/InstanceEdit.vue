<template>
    <div>
        <auto-form-drawer ref="drawerRef" v-model:visible="dialogVisible" :title="title" :items="items" :data="editData" size="40%" :confirm-api="btnOk" @cancel="emit('cancel')">
            <!-- 关联标签 -->
            <template #tagCodePaths="{ form }">
                <TagTreeSelect multiple :code="form.code" v-model="form.tagCodePaths" />
            </template>

            <!-- 数据库类型（选项含图标 + prefix 图标；自定义插槽绕过了 auto-form 的 onChange 代理，需手动 @change 触发端口联动） -->
            <template #type="{ form }">
                <ASelect v-model="form.type" @change="(v: string) => onTypeChange(v, form)">
                    <AOption
                        v-for="(dbTypeAndDialect, key) in getDbDialectMap()"
                        :key="key"
                        :value="dbTypeAndDialect[0]"
                        :label="dbTypeAndDialect[1].getInfo().name"
                    >
                        <SvgIcon :name="dbTypeAndDialect[1].getInfo().icon" :size="20" />
                        {{ dbTypeAndDialect[1].getInfo().name }}
                    </AOption>

                    <template #prefix>
                        <SvgIcon :name="getDbDialect(form.type).getInfo().icon" :size="20" />
                    </template>
                </ASelect>
            </template>

            <!-- 认证信息表格编辑 -->
            <template #authCerts="{ form }">
                <ResourceAuthCertTableEdit
                    v-model="form.authCerts"
                    :resource-code="form.code"
                    :resource-type="TagResourceTypeEnum.DbInstance.value"
                    :test-conn-btn-loading="testConnBtnLoading"
                    @test-conn="testConn(form, $event)"
                    :disable-ciphertext-type="[AuthCertCiphertextTypeEnum.PrivateKey.value]"
                />
            </template>

            <!-- SSH 隧道 -->
            <template #sshTunnelMachineId="{ form }">
                <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { notBlankI18n } from '@/common/assert';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import SvgIcon from '@/components/svg-icon/index.vue';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { computed, type PropType, useTemplateRef } from 'vue';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import { ASelect, AOption } from '@/components/auto-form/ui/adapter';
import ResourceAuthCertTableEdit from '../component/ResourceAuthCertTableEdit.vue';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import { AuthCertCiphertextTypeEnum } from '../tag/enums';
import { dbApi } from './api';
import { DbType, getDbDialect, getDbDialectMap } from './dialect';
import type { DbInstance } from './types';
import type { MachineAuthCert } from '@/views/ops/machine/types';

/** 数据库实例编辑表单类型 (id/name/sshTunnelMachineId/extra/params 允许 null 表示未设置) */
interface DbInstanceForm extends Omit<Partial<DbInstance>, 'id' | 'name' | 'sshTunnelMachineId' | 'extra' | 'params'> {
    id?: number | null;
    type: string;
    name?: string | null;
    sshTunnelMachineId?: number | null;
    extra?: Record<string, unknown> | null;
    params?: string | null;
    tagCodePaths?: string[];
}

const props = defineProps({
    data: {
        type: Object as PropType<DbInstance | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

/** 切换数据库类型联动：新增时重置默认端口，并清空类型相关的额外参数（自定义插槽需手动调用，auto-form 的 onChange 代理对 custom slot 不生效） */
const onTypeChange = (val: string, form: AutoFormData) => {
    const dbForm = form as DbInstanceForm;
    if (!dbForm.id) {
        dbForm.port = getDbDialect(val).getInfo().defaultPort as number;
    }
    dbForm.extra = {};
};

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；group 分组容器 + tagCodePaths/type/authCerts/sshTunnel 走插槽，oracle 额外参数用嵌套路径 prop） */
const items: AutoFormItem[] = [
    { type: 'group', label: 'common.basic' },
    { prop: 'tagCodePaths', label: 'tag.relateTag', required: true },
    { prop: 'name', label: 'common.name', required: true },
    {
        prop: 'type',
        label: 'common.type',
        required: true,
        // 注意：type 使用自定义插槽渲染，onChange 代理不生效；端口联动逻辑在 onTypeChange 中由模板 @change 手动触发
    },
    { prop: 'host', label: 'Host', required: true, when: (form) => (form as DbInstanceForm).type !== DbType.sqlite, span: 17 },
    { prop: 'port', label: 'Port', type: 'number', when: (form) => (form as DbInstanceForm).type !== DbType.sqlite, span: 7 },
    { prop: 'host', label: 'Path', required: true, placeholder: 'db.sqlitePathPlaceholder', when: (form) => (form as DbInstanceForm).type === DbType.sqlite },
    {
        prop: 'extra.stype',
        label: 'SID|Service',
        type: 'select',
        options: [
            { value: 1, label: 'Service' },
            { value: 2, label: 'SID' },
        ],
        when: (form) => (form as DbInstanceForm).type === DbType.oracle,
        onChange: (_val: unknown, form: AutoFormData) => {
            const extra = (form as DbInstanceForm).extra;
            if (extra) {
                extra.serviceName = '';
                extra.sid = '';
            }
        },
    },
    { prop: 'extra.serviceName', label: 'Service Name', placeholder: 'Service Name', when: (form) => (form as DbInstanceForm).type === DbType.oracle && (form as DbInstanceForm).extra?.stype == 1 },
    { prop: 'extra.sid', label: 'SID', placeholder: 'SID', when: (form) => (form as DbInstanceForm).type === DbType.oracle && (form as DbInstanceForm).extra?.stype == 2 },
    { prop: 'remark', label: 'common.remark', type: 'textarea' },
    { type: 'group', label: 'common.account' },
    { prop: 'authCerts', label: 'db.acName', type: 'custom' },
    { type: 'group', label: 'common.other' },
    { prop: 'params', label: 'db.connParam', placeholder: 'db.connParamPlaceholder' },
    { prop: 'sshTunnelMachineId', label: 'machine.sshTunnel' },
];

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('drawerRef');

const DefaultForm: DbInstanceForm = {
    id: null,
    type: DbType.mysql,
    code: '',
    name: null,
    host: '',
    port: getDbDialect(DbType.mysql).getInfo().defaultPort,
    extra: {} as Record<string, unknown>, // 连接需要的额外参数（json）
    params: null,
    remark: '',
    sshTunnelMachineId: null as number | null,
    authCerts: [],
    tagCodePaths: [],
};

/** 传给 AutoFormDrawer 的回填数据：新增时应用 DefaultForm；编辑时兜底 extra 为空对象（深拷贝由组件内部完成） */
const editData = computed<AutoFormData | null>(() => {
    const dbInst = props.data;
    if (!dbInst) {
        return { ...DefaultForm, authCerts: [], tagCodePaths: [] } as unknown as AutoFormData;
    }
    return { ...dbInst, extra: (dbInst.extra || {}) as Record<string, unknown> } as unknown as AutoFormData;
});

const { execute: saveInstanceExec, data: saveInstanceRes } = dbApi.saveInstance.useApi();
const { isFetching: testConnBtnLoading, execute: testConnExec } = dbApi.testConn.useApi();

const buildSubmitForm = (form: DbInstanceForm): Record<string, unknown> => {
    const reqForm: Record<string, unknown> = { ...form };
    reqForm.selectAuthCert = null;
    reqForm.tags = null;
    if (!form.sshTunnelMachineId) {
        reqForm.sshTunnelMachineId = -1;
    }
    if (form.extra && Object.keys(form.extra).length > 0) {
        reqForm.extra = form.extra;
    }
    return reqForm;
};

const testConn = async (rawForm: AutoFormData, authCert: MachineAuthCert) => {
    const form = rawForm as unknown as DbInstanceForm;
    await useI18nFormValidate(drawerRef);
    await testConnExec({
        ...buildSubmitForm(form),
        authCerts: [authCert],
    });
    Msg.success('db.connSuccess');
};

// confirmApi 提交动作：notBlankI18n 校验失败抛错中止（组件保持抽屉打开）；成功提示与关闭抽屉由组件内置逻辑处理
const btnOk = async (form: AutoFormData) => {
    const dbForm = form as unknown as DbInstanceForm;
    notBlankI18n(dbForm.authCerts, 'db.acName');
    await saveInstanceExec(buildSubmitForm(dbForm));
    dbForm.id = saveInstanceRes.value;
    emit('val-change', dbForm);
};
</script>
<style lang="scss"></style>
