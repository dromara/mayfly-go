import { VNode, h, render } from 'vue';
import MonacoEditorDialogComp from './MonacoEditorDialogComp.vue';

export type MonacoEditorDialogProps = {
    content: string;
    title: string;
    language: string;
    height?: string;
    width?: string;
    confirmFn?: Function; // 点击确认的回调函数，入参editor value
    cancelFn?: Function; // 点击取消 或 关闭弹窗的回调函数
};

const boxId = 'monaco-editor-dialog-id';

let boxInstance: VNode;

const MonacoEditorDialog = (props: MonacoEditorDialogProps): void => {
    if (!boxInstance) {
        const container = document.createElement('div');
        container.id = boxId;
        // 创建 虚拟dom
        boxInstance = h(MonacoEditorDialogComp);
        // 将虚拟dom渲染到 container dom 上
        render(boxInstance, container);
        // 最后将 container 追加到 body 上
        document.body.appendChild(container);
    }

    const boxVue = boxInstance.component;
    if (boxVue) {
        // 调用open方法显示弹框，注意不能使用boxVue.ctx来调用组件函数（build打包后ctx会获取不到）
        boxVue.exposed?.open(props);
    }
};

export default MonacoEditorDialog;
