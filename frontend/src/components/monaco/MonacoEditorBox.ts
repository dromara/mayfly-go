import { VNode, h, render } from 'vue';
import MonacoEditorDialog from './MonacoEditorDialog.vue';
import * as monaco from 'monaco-editor';
import { ElMessage } from 'element-plus';

export type MonacoEditorDialogProps = {
    content: string;
    title: string;
    language: string;
    height?: string;
    width?: string;
    options?: any; // 可选项，如字体大小等
    canChangeLang?: boolean; // 是否可以切换语言
    showConfirmButton?: boolean;
    confirmFn?: Function; // 点击确认的回调函数，入参editor value
    closeFn?: Function; // 点击取消 或 关闭弹窗的回调函数
    completionItemProvider?: monaco.languages.CompletionItemProvider; // 自定义补全项
};

const MonacoEditorBox = (props: MonacoEditorDialogProps): void => {
    const boxId = 'monaco-editor-dialog-id';
    let boxInstance: VNode;

    const container = document.getElementById(boxId);
    if (!container) {
        const container = document.createElement('div');
        container.id = boxId;

        if (props.showConfirmButton === undefined) {
            props.showConfirmButton = true;
        }
        if (props.canChangeLang === undefined) {
            props.canChangeLang = true;
        }
        if (props.content === undefined) {
            props.content = '';
        }

        // 创建 虚拟dom
        boxInstance = h(MonacoEditorDialog, {
            ...props,
            modelValue: props.content,
            'onUpdate:modelValue': (value: string) => {
                props.content = value;
            },
            // 'onUpdate:visible': (value: boolean) => {},
            visible: true,
            onClose: () => {
                // 卸载组件
                if (boxInstance) {
                    render(null, container);
                    boxInstance = null as any;
                }
                // 移除 container DOM 元素
                document.body.removeChild(container);
                props.closeFn && props.closeFn();
            },
            onConfirm: () => {
                let value = props.content;
                if (props.language === 'json') {
                    let val;
                    try {
                        val = JSON.parse(value);
                        if (typeof val !== 'object') {
                            ElMessage.error('Invalid json');
                            return;
                        }
                    } catch (e) {
                        ElMessage.error('Invalid json');
                        return;
                    }
                    // 压缩json字符串
                    value = JSON.stringify(val);
                }
                props.confirmFn && props.confirmFn(value);
            },
        });
        // 将虚拟dom渲染到 container dom 上
        render(boxInstance, container);
        // 最后将 container 追加到 body 上
        document.body.appendChild(container);
    }
};

export default MonacoEditorBox;
