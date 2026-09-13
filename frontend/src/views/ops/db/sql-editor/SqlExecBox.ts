import { h, render } from 'vue';
import type { SqlFormatterLanguage } from '../dialect/types';
import SqlExecDialog from './SqlExecDialog.vue';

export type SqlExecProps = {
    sql: string;
    dbId: number;
    db: string;
    /**
     * 弹窗内格式化 sql 用的方言标识，一律传 `dialect.getInfo().formatSqlDialect`。
     *
     * 注意别传成 DbType（'postgres'/'mssql' 等）：那是数据库类型标识，
     * sql-formatter 只认自己的方言名，传错会直接抛 ConfigError 使弹窗打不开。
     */
    formatDialect: SqlFormatterLanguage;
    runSuccessCallback?: Function;
    cancelCallback?: Function;
};

const SqlExecBox = (props: SqlExecProps): void => {
    const propsCancelFn = props.cancelCallback;
    //  包装取消回调函数，新增销毁组件代码
    props.cancelCallback = () => {
        propsCancelFn && propsCancelFn();
        setTimeout(() => {
            // 销毁组件
            render(null, document.body);
        }, 500);
    };
    const vnode = h(SqlExecDialog, {
        ...props,
    });
    render(vnode, document.body);
};

export default SqlExecBox;
