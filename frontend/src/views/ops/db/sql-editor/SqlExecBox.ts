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
    /**
     * 整批语句原子执行（全成功才提交，任一条失败即整批回滚）。
     *
     * 数据网格的批量行编辑、删除、新增表单都传 true：用户勾的是「提交修改」，
     * 一批改动里有一半落库一半失败，比整批不生效更难收拾
     */
    atomic?: boolean;
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
