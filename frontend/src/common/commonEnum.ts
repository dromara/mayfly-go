import EnumValue from './Enum';
// element plus 自带国际化
import zhcnLocale from 'element-plus/es/locale/lang/zh-cn';
import enLocale from 'element-plus/es/locale/lang/en';

// i18n
export const I18nEnum = {
    ZhCn: EnumValue.of('zh-cn', '简体中文').setExtra({ icon: 'icon layout/cn', el: zhcnLocale }),
    En: EnumValue.of('en', 'English').setExtra({ icon: 'icon layout/en', el: enLocale }),
};

export const LinkTypeEnum = {
    Iframes: EnumValue.of(1, 'ifrmaes'),
    Link: EnumValue.of(2, 'link'),
};

// 资源类型
export const ResourceTypeEnum = {
    Machine: EnumValue.of(1, '机器').setExtra({ icon: 'Monitor', iconColor: 'var(--el-color-primary)' }).tagTypeSuccess(),
    Db: EnumValue.of(2, '数据库实例').setExtra({ icon: 'Coin', iconColor: 'var(--el-color-warning)' }).tagTypeWarning(),
    Redis: EnumValue.of(3, 'redis').setExtra({ icon: 'icon redis/redis', iconColor: 'var(--el-color-danger)' }).tagTypeInfo(),
    Mongo: EnumValue.of(4, 'mongo').setExtra({ icon: 'icon mongo/mongo', iconColor: 'var(--el-color-success)' }).tagTypeDanger(),
    AuthCert: EnumValue.of(5, '授权凭证').setExtra({ icon: 'Ticket', iconColor: 'var(--el-color-success)' }),
    Es: EnumValue.of(6, 'ES实例').setExtra({ icon: 'icon es/es-color', iconColor: 'var(--el-color-warning)' }).tagTypeWarning(),
};

// 标签关联的资源类型
export const TagResourceTypeEnum = {
    PublicAuthCert: EnumValue.of(-2, '公共凭证').setExtra({ icon: 'Ticket' }),
    Tag: EnumValue.of(-1, '标签').setExtra({ icon: 'CollectionTag' }),

    Machine: ResourceTypeEnum.Machine,
    DbInstance: ResourceTypeEnum.Db,
    EsInstance: ResourceTypeEnum.Es,
    Redis: ResourceTypeEnum.Redis,
    Mongo: ResourceTypeEnum.Mongo,
    AuthCert: ResourceTypeEnum.AuthCert,

    Db: EnumValue.of(22, '数据库').setExtra({ icon: 'Coin' }),
};

// 标签关联的资源类型路径
export const TagResourceTypePath = {
    MachineAuthCert: `${TagResourceTypeEnum.Machine.value}/${TagResourceTypeEnum.AuthCert.value}`,

    DbInstanceAuthCert: `${TagResourceTypeEnum.DbInstance.value}/${TagResourceTypeEnum.AuthCert.value}`,
    Db: `${TagResourceTypeEnum.DbInstance.value}/${TagResourceTypeEnum.AuthCert.value}/${TagResourceTypeEnum.Db.value}`,
    Es: `${TagResourceTypeEnum.EsInstance.value}/${TagResourceTypeEnum.AuthCert.value}`,
};

// 消息子类型
export const MsgSubtypeEnum = {
    UserLogin: EnumValue.of('user.login', 'login.login').setExtra({
        notifyType: 'primary',
    }),

    MachineFileUploadSuccess: EnumValue.of('machine.file.upload.success', 'machine.fileUploadSuccess').setExtra({
        notifyType: 'success',
    }),
    MachineFileUploadFail: EnumValue.of('machine.file.upload.fail', 'machine.fileUploadFail').setExtra({
        notifyType: 'danger',
    }),

    DbDumpFail: EnumValue.of('db.dump.fail', 'db.dbDumpFail').setExtra({
        notifyType: 'danger',
    }),
    SqlScriptRunSuccess: EnumValue.of('db.sqlscript.run.success', 'db.sqlScriptRunSuccess').setExtra({
        notifyType: 'success',
    }),
    SqlScriptRunFail: EnumValue.of('db.sqlscript.run.fail', 'db.sqlScriptRunFail').setExtra({
        notifyType: 'danger',
    }),

    FlowUserTaskTodo: EnumValue.of('flow.usertask.todo', 'flow.todoTask').setExtra({
        notifyType: 'primary',
    }),
};
