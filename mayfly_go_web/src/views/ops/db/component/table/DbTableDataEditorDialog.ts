import { h, render, VNode } from 'vue';
import SqlExecDialog from './DbTableDataEditorDialog.vue';

export type TableDataEditorProps = {
    content: string;
    title: string;
    language: string;
    runSuccessCallback?: Function;
    cancelCallback?: Function;
};

const boxId = 'table-data-editor-id';

const renderBox = (): VNode => {
    const props: TableDataEditorProps = {
        content: '',
        title: '',
        language: '',
    };
    const container = document.createElement('div');
    container.id = boxId;
    // 创建 虚拟dom
    const boxVNode = h(SqlExecDialog, props);
    // 将虚拟dom渲染到 container dom 上
    render(boxVNode, container);
    // 最后将 container 追加到 body 上
    document.body.appendChild(container);

    return boxVNode;
};

let boxInstance: any;

const TableDataEditorBox = (props: TableDataEditorProps): void => {
    if (boxInstance) {
        const boxVue = boxInstance.component;
        // 调用open方法显示弹框，注意不能使用boxVue.ctx来调用组件函数（build打包后ctx会获取不到）
        boxVue.exposed.open(props);
    } else {
        boxInstance = renderBox();
        TableDataEditorBox(props);
    }
};

export default TableDataEditorBox;
