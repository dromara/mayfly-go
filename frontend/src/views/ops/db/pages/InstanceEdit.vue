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

            <!-- 安全连接：档位由方言能力驱动；证书材料只做「本次提交」，已存内容不回显 -->
            <template #tlsConfig="{ form }">
                <div class="flex flex-col gap-2">
                    <el-select v-model="form.extra.tlsMode" clearable size="small" :placeholder="$t('db.tlsModePlaceholder')">
                        <el-option v-for="mode in tlsModeOptions(form)" :key="mode" :value="mode" :label="$t(tlsModeLabelKey(mode))" />
                    </el-select>

                    <template v-if="form.extra && form.extra.tlsMode">
                        <el-input
                            v-if="form.extra.tlsMode === 'verify-full'"
                            v-model="form.extra.tlsServerName"
                            size="small"
                            :placeholder="$t('db.tlsServerNamePlaceholder')"
                        />
                        <el-checkbox v-if="tlsModeNeedsCa(form)" v-model="form.extra.tlsUseSystemCA">
                            {{ $t('db.tlsUseSystemCA') }}
                        </el-checkbox>
                        <el-input
                            v-model="form.tlsMaterial!.rootCa"
                            type="textarea"
                            :rows="3"
                            size="small"
                            :placeholder="$t('db.tlsRootCaPlaceholder')"
                        />
                        <el-input
                            v-model="form.tlsMaterial!.clientCert"
                            type="textarea"
                            :rows="3"
                            size="small"
                            :placeholder="$t('db.tlsClientCertPlaceholder')"
                        />
                        <el-input
                            v-model="form.tlsMaterial!.clientKey"
                            type="textarea"
                            :rows="3"
                            size="small"
                            :placeholder="$t('db.tlsClientKeyPlaceholder')"
                        />
                        <div class="text-[12px] text-gray-500">{{ $t('db.tlsKeepExistingHint') }}</div>
                        <div v-if="form.extra.tlsFingerprint" class="text-[12px] text-gray-500">
                            {{ $t('db.tlsConfigured', { fingerprint: form.extra.tlsFingerprint }) }}
                        </div>
                    </template>
                </div>
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
import { relateTagItem } from '../../component/relateTagItem';
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

    /**
     * 本次提交的 TLS 证书材料（明文 PEM）。
     * 已存证书从不回显，因此新增/编辑都从空对象起步：留空即「不修改已保存的证书」。
     */
    tlsMaterial: TlsMaterialForm;
}

interface TlsMaterialForm {
    rootCa?: string;
    clientCert?: string;
    clientKey?: string;
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
    // extra 与证书材料都按类型作废：不同方言支持的档位不同，跨类型沿用旧值会被后端能力校验拒绝
    dbForm.extra = {};
    dbForm.tlsMaterial = {};
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

// ==================== 安全连接（TLS） ====================

/**
 * 模板插槽传入的表单按需读取口径。
 *
 * 不直接声明为 AutoFormData：表单类型契约测试禁止它在页面里出现（页面不应依赖表单袋形状），
 * 这里只声明本组件真正读的两个字段，并对取值做运行时收窄。
 */
interface TlsFormShape {
    type?: unknown;
    extra?: Record<string, unknown> | null;
}

/**
 * 按数据库类型取能力。
 *
 * 插槽里的 form 是表单袋（Record<string, any>），而能力查询只需要 type 一个字段，
 * 故按类型入参而不是把整表单断言成 DbInstanceForm——断言会掩盖字段缺失，
 * 类型入参则让「type 为空」自然落到 fallback 方言。
 */
const capabilitiesOfType = (type: unknown) => getDialectCapabilities(getDbDialect(String(type ?? '')));

/** 该方言是否声明了加密档位（未声明则整组不渲染，后端也会拒绝其档位） */
const hasTlsModes = (form: DbInstanceForm) => capabilitiesOfType(form.type).tlsModes.length > 0;

/** 当前方言可选的档位（后端按同一份能力位做保存期校验） */
const tlsModeOptions = (form: TlsFormShape) => capabilitiesOfType(form.type).tlsModes;

/** 档位 → 文案 key（未知档位原样展示，避免新增档位时前端静默空白） */
const tlsModeLabelKey = (mode: string) => {
    const keys: Record<string, string> = {
        require: 'db.tlsModeRequire',
        'verify-ca': 'db.tlsModeVerifyCa',
        'verify-full': 'db.tlsModeVerifyFull',
    };
    return keys[mode] ?? mode;
};

/** verify-* 才需要 CA；require 只要求加密，不校验证书 */
const tlsModeNeedsCa = (form: TlsFormShape) => {
    const mode = String(form.extra?.tlsMode ?? '');
    return mode === 'verify-ca' || mode === 'verify-full';
};

/** 表单声明（defineFormItems<DbInstanceForm>，渲染 + 校验唯一数据源；group 分组容器 + tagCodePaths/type/authCerts/sshTunnel 走插槽，方言额外连接参数用嵌套路径 prop） */
const items = defineFormItems<DbInstanceForm>([
    { type: 'group', label: 'common.basic' },
    relateTagItem<DbInstanceForm>(),
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
    // 分组容器在「组内字段全被 when 隐藏」时不渲染，故未声明 TLS 能力的方言（dm/sqlite）
    // 不会留下空的安全连接分组
    { type: 'group', label: 'db.securityGroup' },
    { prop: 'tlsConfig', label: 'db.tlsMode', type: 'custom', when: hasTlsModes },
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
    // 证书材料只在本次提交里出现，不从服务端回填（服务端也不下发）
    tlsMaterial: {},
};

// 宿主内部表单在 @opened 接管（抛出的即宿主持有的同一对象），提交与连通性测试均基于它
const { onOpened, requireForm } = useAutoFormModel<DbInstanceForm>();

/** 传给 AutoFormDrawer 的回填数据：新增时应用 DefaultForm；编辑时以默认值为底再覆盖行数据（深拷贝由组件内部完成） */
const editData = computed<DbInstanceForm>(() => {
    const dbInst = props.data;
    if (!dbInst) {
        return { ...DefaultForm, authCerts: [], tagCodePaths: [] };
    }
    // tlsMaterial 显式置空：即便后端将来误下发材料，编辑表单也不会把它回显/回传
    return { ...DefaultForm, ...dbInst, extra: dbInst.extra || {}, tlsMaterial: {} };
});

const { execute: saveInstanceExec, data: saveInstanceRes } = dbApi.saveInstance.useApi();
const { isFetching: testConnBtnLoading, execute: testConnExec, data: testConnResult } = dbApi.testConn.useApi();

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
    // 只有真的填了材料才提交：全空表示「沿用已保存证书」，N 个空串传上去会让后端白做一次合并
    const material = form.tlsMaterial ?? {};
    const hasMaterial = Object.values(material).some((value) => typeof value === 'string' && value.trim() !== '');
    if (hasMaterial) {
        reqForm.tlsMaterial = material;
    } else {
        delete reqForm.tlsMaterial;
    }
    return reqForm;
};

const testConn = async (authCert: MachineAuthCert) => {
    await useI18nFormValidate(drawerRef);
    await testConnExec({
        ...buildSubmitForm(requireForm()),
        authCerts: [authCert],
    });
    // 连上后按服务端观测到的事实回话：配置了 TLS 不等于协商成功，这里给的是「这条连接实际如何」
    const tls = testConnResult.value?.tls;
    if (tls?.available && tls.encrypted) {
        Msg.success('db.connSuccessTls', { version: tls.version || '-', cipher: tls.cipher || '-' });
        return;
    }
    if (tls?.available && !tls.encrypted) {
        // 用 warning 而不是 success：测通只说明能连，不代表加密要求已被满足
        Msg.warning('db.connSuccessNoTls');
        return;
    }
    // available=false：该方言读不到协商结果，如实只报「连接成功」，不猜加密与否
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
