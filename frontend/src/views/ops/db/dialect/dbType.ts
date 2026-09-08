/**
 * 数据库类型常量（纯常量，零依赖）。
 *
 * 独立成文件的原因：dialect/index.ts 会加载全部方言实现并形成模块循环
 * （dialect → db → completion → dialect），任何模块若在模块体执行期直接
 * 引用 dialect/index 的运行时值（如 quoter.ts 的方言引用符表），会因
 * TDZ 报 `Cannot access 'DbType' before initialization`。
 * 需要在模块顶层使用 DbType 的模块，应从本文件导入而非 dialect/index。
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
