<template>
    <div>
        <auto-form-drawer
            ref="drawerRef"
            v-model:visible="dialogVisible"
            :title="title"
            :items="items"
            :data="editData"
            size="40%"
            :confirm-api="btnOk"
            @opened="onOpened"
            @cancel="emit('cancel')"
        >
            <!-- 关联标签 -->
            <template #tagCodePaths="{ form }">
                <TagTreeSelect multiple :code="form.code" v-model="form.tagCodePaths" />
            </template>

            <!-- 数据库类型（选项含图标 + prefix 图标；自定义插槽绕过了 auto-form 的 onChange 代理，需手动 @change 触发端口联动） -->
            <template #type="{ form }">
                <ASelect v-model="form.type" @change="(v: string) => onTypeChange(v)">
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
                    @test-conn="testConn($event)"
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
import { TagResourceTypeEnum } from '@/common/commonEnum';
import SvgIcon from '@/components/svg-icon/index.vue';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { computed, type PropType, useTemplateRef } from 'vue';
import { AutoFormDrawer, defineFormItems } from '@/components/auto-form';
import { useAutoFormModel } from '@/hooks/useAutoFormModel';
import { ASelect, AOption } from '@/components/auto-form/ui/adapter';
import ResourceAuthCertTableEdit from '../../component/ResourceAuthCertTableEdit.vue';
import SshTunnelSelect from '../../component/SshTunnelSelect.vue';
import TagTreeSelect from '../../component/TagTreeSelect.vue';
import { AuthCertCiphertextTypeEnum } from '../../tag/enums';
import { dbApi } from '../api';
import { DbType, getDbDialect, getDbDialectMap, getDialectCapabilities } from '../dialect';
import type { DbInstance } from '../types';
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
const emit = defineEmits<{
    /** 取消编辑，父级关闭弹窗 */
    cancel: [];
    /** 保存成功，回传表单，父级据此刷新实例列表 */
    'val-change': [form: DbInstanceForm];
}>();

/** 切换数据库类型联动：新增时重置默认端口，并清空类型相关的额外参数（自定义插槽需手动调用，auto-form 的 onChange 代理对 custom slot 不生效） */
const onTypeChange = (val: string) => {
    const dbForm = requireForm();
    if (!dbForm.id) {
        dbForm.port = getDbDialect(val).getInfo().defaultPort as number;
    }
    dbForm.extra = {};
};

/**
 * 连接参数的差异项一律问方言能力，新增方言只需在自身声明能力，无需改动本文件。
 */
const capabilityOf = (form: DbInstanceForm) => getDialectCapabilities(getDbDialect(form.type));
/** 是否以 host/port 连接（sqlite 等文件型库为 false） */
const isHostPortConn = (form: DbInstanceForm) => capabilityOf(form).connectionMode === 'host_port';
/** 是否以本地文件路径连接 */
const isFilePathConn = (form: DbInstanceForm) => capabilityOf(form).connectionMode === 'file_path';
/** 是否需要 SID / Service Name 连接描述符 */
const needSidService = (form: DbInstanceForm) => capabilityOf(form).connectDescriptor === 'sid_service';

/** 表单声明（defineFormItems<DbInstanceForm>，渲染 + 校验唯一数据源；group 分组容器 + tagCodePaths/type/authCerts/sshTunnel 走插槽，方言额外连接参数用嵌套路径 prop） */
const items = defineFormItems<DbInstanceForm>([
    { type: 'group', label: 'common.basic' },
    { prop: 'tagCodePaths', label: 'tag.relateTag', required: true },
    { prop: 'name', label: 'common.name', required: true },
    {
        prop: 'type',
        label: 'common.type',
        required: true,
        // 注意：type 使用自定义插槽渲染，onChange 代理不生效；端口联动逻辑在 onTypeChange 中由模板 @change 手动触发
    },
    { prop: 'host', label: 'Host', required: true, when: isHostPortConn, span: 17 },
    { prop: 'port', label: 'Port', type: 'number', when: isHostPortConn, span: 7 },
    { prop: 'host', label: 'Path', required: true, placeholder: 'db.sqlitePathPlaceholder', when: isFilePathConn },
    {
        prop: 'extra.stype',
        label: 'SID|Service',
        type: 'select',
        options: [
            { value: 1, label: 'Service' },
            { value: 2, label: 'SID' },
        ],
        when: needSidService,
        // 切换 SID/Service 后两项描述符互斥，需清空避免残留脏值
        onChange: (_value, form) => {
            const extra = form.extra;
            if (extra) {
                extra.serviceName = '';
                extra.sid = '';
            }
        },
    },
    { prop: 'extra.serviceName', label: 'Service Name', placeholder: 'Service Name', when: (form) => needSidService(form) && form.extra?.stype == 1 },
    { prop: 'extra.sid', label: 'SID', placeholder: 'SID', when: (form) => needSidService(form) && form.extra?.stype == 2 },
    { prop: 'remark', label: 'common.remark', type: 'textarea' },
    { type: 'group', label: 'common.account' },
    // 无凭证则实例无法连接，后端也会拒绝保存
    { prop: 'authCerts', label: 'db.acName', type: 'custom', validate: (value) => (Array.isArray(value) && value.length > 0 ? true : 'db.acNameRequired') },
    { type: 'group', label: 'common.other' },
    { prop: 'params', label: 'db.connParam', placeholder: 'db.connParamPlaceholder' },
    { prop: 'sshTunnelMachineId', label: 'machine.sshTunnel' },
]);

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('drawerRef');

const DefaultForm: DbInstanceForm = {
    id: null,
    type: DbType.mysql,
    code: '',
    name: null,
    host: '',
    port: getDbDialect(DbType.mysql).getInfo().defaultPort,
    extra: {}, // 连接需要的额外参数（json）
    params: null,
    remark: '',
    sshTunnelMachineId: null,
    authCerts: [],
    tagCodePaths: [],
};

// 宿主内部表单在 @opened 接管（抛出的即宿主持有的同一对象），提交与连通性测试均基于它
const { onOpened, requireForm } = useAutoFormModel<DbInstanceForm>();

/** 传给 AutoFormDrawer 的回填数据：新增时应用 DefaultForm；编辑时以默认值为底再覆盖行数据（深拷贝由组件内部完成） */
const editData = computed<DbInstanceForm>(() => {
    const dbInst = props.data;
    if (!dbInst) {
        return { ...DefaultForm, authCerts: [], tagCodePaths: [] };
    }
    return { ...DefaultForm, ...dbInst, extra: dbInst.extra || {} };
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

const testConn = async (authCert: MachineAuthCert) => {
    await useI18nFormValidate(drawerRef);
    await testConnExec({
        ...buildSubmitForm(requireForm()),
        authCerts: [authCert],
    });
    Msg.success('db.connSuccess');
};

// confirmApi 提交动作：凭证非空已由 authCerts 字段的 validate 声明；成功提示与关闭抽屉由组件内置逻辑处理
const btnOk = async () => {
    const dbForm = requireForm();
    await saveInstanceExec(buildSubmitForm(dbForm));
    dbForm.id = saveInstanceRes.value;
    emit('val-change', dbForm);
};
</script>
<style lang="scss"></style>
