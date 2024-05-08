import { h, render } from 'vue';
import SqlExecDialog from './SqlExecDialog.vue';

export type SqlExecProps = {
    sql: string;
    dbId: number;
    db: string;
    dbType?: string;
    flowProcdef?: any;
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
        visible: true,
    });
    render(vnode, document.body);
};

export default SqlExecBox;
