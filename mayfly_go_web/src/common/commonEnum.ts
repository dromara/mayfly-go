import EnumValue from './Enum';

// 资源类型
export const ResourceTypeEnum = {
    Machine: EnumValue.of(1, '机器').setExtra({ icon: 'Monitor', iconColor: 'var(--el-color-primary)' }).tagTypeSuccess(),
    Db: EnumValue.of(2, '数据库实例').setExtra({ icon: 'Coin', iconColor: 'var(--el-color-warning)' }).tagTypeWarning(),
    Redis: EnumValue.of(3, 'redis').setExtra({ icon: 'iconfont icon-redis', iconColor: 'var(--el-color-danger)' }).tagTypeInfo(),
    Mongo: EnumValue.of(4, 'mongo').setExtra({ icon: 'iconfont icon-mongo', iconColor: 'var(--el-color-success)' }).tagTypeDanger(),
};

// 标签关联的资源类型
export const TagResourceTypeEnum = {
    AuthCert: EnumValue.of(-2, '公共凭证').setExtra({ icon: 'Ticket' }),
    Tag: EnumValue.of(-1, '标签').setExtra({ icon: 'CollectionTag' }),

    Machine: ResourceTypeEnum.Machine,
    Db: ResourceTypeEnum.Db,
    Redis: ResourceTypeEnum.Redis,
    Mongo: ResourceTypeEnum.Mongo,

    MachineAuthCert: EnumValue.of(11, '机器-授权凭证').setExtra({ icon: 'Ticket', iconColor: 'var(--el-color-success)' }),
    DbAuthCert: EnumValue.of(21, '数据库-授权凭证').setExtra({ icon: 'Ticket', iconColor: 'var(--el-color-success)' }),
    DbName: EnumValue.of(22, '数据库').setExtra({ icon: 'Coin' }),
};
