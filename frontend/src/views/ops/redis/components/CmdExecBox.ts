import { h, render } from 'vue';
import CmdExecDialog from './CmdExecDialog.vue';

export type CmdExecProps = {
    id: number;
    db: number | string;
    cmd: any[];
    flowProcdef?: any;
    visible?: boolean;
    runSuccessFn?: Function;
    cancelFn?: Function;
};

const showCmdExecBox = (props: CmdExecProps): void => {
    const propsCancelFn = props.cancelFn;
    props.cancelFn = () => {
        propsCancelFn && propsCancelFn();
        setTimeout(() => {
            // 销毁组件
            render(null, document.body);
        }, 500);
    };

    const vnode = h(CmdExecDialog, { ...props, visible: true });
    render(vnode, document.body);
};

export default showCmdExecBox;
