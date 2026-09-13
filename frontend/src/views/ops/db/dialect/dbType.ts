/**
 * 数据库类型常量（纯常量，零依赖）。
 *
 * 独立成文件的原因：本常量被全部方言文件在模块体执行期（自注册）引用，
 * 必须与 registry.ts、types.ts 同属方言层的「零依赖内核」，才能支撑
 * index.ts 用 import.meta.glob 加载全部方言而不形成模块循环。
 * 历史上它与聚合出口同文件，曾导致
 * `Cannot access 'DbType' before initialization` 的 TDZ 运行时报错。
 */
export const DbType = {
    mysql: 'mysql',
    mariadb: 'mariadb',
    postgresql: 'postgres',
    gauss: 'gauss',
    dm: 'dm', // 达梦
    oracle: 'oracle',
    sqlite: 'sqlite',
    mssql: 'mssql', // ms sqlserver
    kingbaseEs: 'kingbaseEs', // 人大金仓 pgsql模式 https://help.kingbase.com.cn/v8/index.html
    vastbase: 'vastbase', // https://docs.vastdata.com.cn/zh/docs/VastbaseG100Ver2.2.5/doc/%E5%BC%80%E5%8F%91%E8%80%85%E6%8C%87%E5%8D%97/SQL%E5%8F%82%E8%80%83/SQL%E5%8F%82%E8%80%83.html
    clickhouse: 'clickhouse',
};
