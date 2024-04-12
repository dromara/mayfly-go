import EnumValue from './Enum';

// 标签关联的资源类型
export const TagResourceTypeEnum = {
    AuthCert: EnumValue.of(-2, '公共凭证').setExtra({ icon: 'Ticket' }),
    Tag: EnumValue.of(-1, '标签').setExtra({ icon: 'CollectionTag' }),

    Machine: EnumValue.of(1, '机器').setExtra({ icon: 'Monitor' }).tagTypeSuccess(),
    Db: EnumValue.of(2, '数据库').setExtra({ icon: 'Coin' }).tagTypeWarning(),
    Redis: EnumValue.of(3, 'redis').setExtra({ icon: 'iconfont icon-redis' }).tagTypeInfo(),
    Mongo: EnumValue.of(4, 'mongo').setExtra({ icon: 'iconfont icon-mongo' }).tagTypeDanger(),

    MachineAuthCert: EnumValue.of(11, '机器-授权凭证').setExtra({ icon: 'Ticket' }),
};
