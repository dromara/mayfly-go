import EnumValue from './Enum';

// 标签关联的资源类型
export const TagResourceTypeEnum = {
    Machine: EnumValue.of(1, '机器'),
    Db: EnumValue.of(2, '数据库'),
    Redis: EnumValue.of(3, 'redis'),
    Mongo: EnumValue.of(4, 'mongo'),
};
