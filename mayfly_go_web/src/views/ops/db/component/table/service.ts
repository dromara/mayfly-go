import { DbOption, DbType, sqlType } from '@/views/ops/db/component/table/dbs/db-option';
import { MysqlOption } from '@/views/ops/db/component/table/dbs/mysql-option';
import { PostgresqlOption } from '@/views/ops/db/component/table/dbs/postgresql-option';

export const MYSQL_TYPE_LIST = ['bigint', 'binary', 'blob', 'char', 'datetime', 'date', 'decimal', 'double', 'enum', 'float', 'int', 'json', 'longblob', 'longtext', 'mediumblob', 'mediumtext', 'set', 'smallint', 'text', 'time', 'timestamp', 'tinyint', 'varbinary', 'varchar'];

export const GAUSS_TYPE_LIST: sqlType[] = [
    // 数值 - 整数型
    { udtName: 'int1', dataType: 'tinyint', desc: '微整数，别名为INT1', space: '1字节', range: '0 ~ +255' },
    { udtName: 'int2', dataType: 'smallint', desc: '小范围整数，别名为INT2。', space: '2字节', range: '-32,768 ~ +32,767' },
    { udtName: 'int4', dataType: 'integer', desc: '常用的整数，别名为INT4。', space: '4字节', range: '-2,147,483,648 ~ +2,147,483,647' },
    { udtName: 'int8', dataType: 'bigint', desc: '大范围的整数，别名为INT8。', space: '8字节', range: '很大' },

    // 数值 - 任意精度型
    {
        udtName: 'numeric',
        dataType: 'numeric',
        desc: '精度(总位数)取值范围为[1,1000]，标度(小数位数)取值范围为[0,精度]。',
        space: '每四位（十进制位）占用两个字节，然后在整个数据上加上八个字节的额外开销',
        range: '未指定精度的情况下，小数点前最大131,072位，小数点后最大16,383位',
    },
    // 数值 - 任意精度型
    { udtName: 'decimal', dataType: 'decimal', desc: '等同于number类型', space: '等同于number类型' },

    // 数值 - 序列整型
    { udtName: 'smallserial', dataType: 'smallserial', desc: '二字节序列整型。', space: '2字节', range: '-32,768 ~ +32,767' },
    { udtName: 'serial', dataType: 'serial', desc: '四字节序列整型。', space: '4字节', range: '-2,147,483,648 ~ +2,147,483,647' },
    { udtName: 'bigserial', dataType: 'bigserial', desc: '八字节序列整型', space: '8字节', range: '-9,223,372,036,854,775,808 ~ +9,223,372,036,854,775,807' },
    {
        udtName: 'largeserial',
        dataType: 'largeserial',
        desc: '默认插入十六字节序列整型，实际数值类型和numeric相同',
        space: '变长类型，每四位（十进制位）占用两个字节，然后在整个数据上加上八个字节的额外开销。',
        range: '小数点前最大131,072位，小数点后最大16,383位。',
    },

    // 数值 - 浮点类型（不常用 就不列出来了）

    // 货币类型
    { udtName: 'money', dataType: 'money', desc: '货币金额', space: '8字节', range: '-92233720368547758.08 ~ +92233720368547758.07' },

    // 布尔类型
    { udtName: 'bool', dataType: 'bool', desc: '布尔类型', space: '1字节', range: 'true：真 , false：假 , null：未知（unknown）' },

    // 字符类型
    { udtName: 'char', dataType: 'char', desc: '定长字符串，不足补空格。n是指字节长度，如不带精度n，默认精度为1。', space: '最大为10MB' },
    { udtName: 'character', dataType: 'character', desc: '定长字符串，不足补空格。n是指字节长度，如不带精度n，默认精度为1。', space: '最大为10MB' },
    { udtName: 'nchar', dataType: 'nchar', desc: '定长字符串，不足补空格。n是指字节长度，如不带精度n，默认精度为1。', space: '最大为10MB' },
    { udtName: 'varchar', dataType: 'varchar', desc: '变长字符串。PG兼容模式下，n是字符长度。其他兼容模式下，n是指字节长度。', space: '最大为10MB。' },
    { udtName: 'text', dataType: 'text', desc: '变长字符串。', space: '最大稍微小于1GB-1。' },
    { udtName: 'clob', dataType: 'clob', desc: '文本大对象。是TEXT类型的别名。', space: '最大稍微小于32TB-1。' },

    //特殊字符类型  用的很少，先屏蔽了
    // { udtName: 'name', dataType: 'name', desc: '用于对象名的内部类型。', space: '64字节。' },
    // { udtName: '"char"', dataType: '"char"', desc: '单字节内部类型。', space: '1字节。' },

    // 二进制类型
    { udtName: 'bytea', dataType: 'bytea', desc: '变长的二进制字符串', space: '4字节加上实际的二进制字符串。最大为1GB减去8203字节（即1073733621字节）。' },

    // 日期/时间类型
    { udtName: 'date', dataType: 'date', desc: '日期', space: '4字节' },
    { udtName: 'time', dataType: 'time', desc: 'TIME [(p)] 只用于一日内时间,p表示小数点后的精度，取值范围为0~6。', space: '8-12字节' },
    { udtName: 'timestamp', dataType: 'timestamp', desc: 'TIMESTAMP[(p)]日期和时间,p表示小数点后的精度，取值范围为0~6', space: '8字节' },
    // 带时区的时间戳用的少，先屏蔽了
    //{ udtName: 'TIMESTAMPTZ', dataType: 'TIMESTAMP WITH TIME ZONE', desc: '带时区的时间戳', space: '8字节' },
    {
        udtName: 'interval',
        dataType: 'interval',
        desc: '时间间隔', // 可以跟参数：YEAR，MONTH，DAY，HOUR，MINUTE，SECOND，DAY TO HOUR，DAY TO MINUTE，DAY TO SECOND，HOUR TO MINUTE，HOUR TO SECOND，MINUTE TO SECOND
        space: '精度取值范围为0~6，且参数为SECOND，DAY TO SECOND，HOUR TO SECOND或MINUTE TO SECOND时，参数p才有效',
    },
    // 几何类型
    { udtName: 'point', dataType: 'point', desc: '平面中的点， 如:(x,y)', space: '16字节' },
    { udtName: 'lseg', dataType: 'lseg', desc: '（有限）线段， 如:((x1,y1),(x2,y2))', space: '32字节' },
    { udtName: 'box', dataType: 'box', desc: '矩形， 如:((x1,y1),(x2,y2))', space: '32字节' },
    { udtName: 'path', dataType: 'path', desc: '闭合路径（与多边形类似）， 如:((x1,y1),...)', space: '16+16n字节' },
    { udtName: 'path', dataType: 'path', desc: '开放路径（与多边形类似）， 如:[(x1,y1),...]', space: '16+16n字节' },
    { udtName: 'polygon', dataType: 'polygon', desc: '多边形（与闭合路径相似）， 如:((x1,y1),...)', space: '40+16n字节' },
    { udtName: 'circle', dataType: 'polygon', desc: '圆,如:<(x,y),r> （圆心和半径）', space: '24 字节' },

    // 网络地址类型
    { udtName: 'cidr', dataType: 'cidr', desc: 'IPv4网络', space: '7字节' },
    { udtName: 'inet', dataType: 'inet', desc: 'IPv4主机和网络', space: '7字节' },
    { udtName: 'macaddr', dataType: 'macaddr', desc: 'MAC地址', space: '6字节' },
];

export const getDbOption = (dbType: string | undefined): DbOption => {
    if (dbType === DbType.mysql) {
        return new MysqlOption();
    } else if (dbType === DbType.postgresql) {
        return new PostgresqlOption();
    }
    throw new Error('不支持的数据库');
};
