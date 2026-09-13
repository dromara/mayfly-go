import { Msg } from '@/hooks/useI18n';
import { h, render } from 'vue';
import MonacoEditorDialog from './MonacoEditorDialog.vue';
import type { MonacoEditorDialogProps } from './types';

/**
 * 命令式打开 monaco 编辑器弹窗（只读查看或编辑文本）
 *
 * ## 加载边界：弹窗同步、编辑器异步
 * 编辑器主体在生产是约 967KB(gzip) 的独立 chunk，由弹窗内部用 defineAsyncComponent 引入，不在本文件与
 * 弹窗的静态图上；本函数常被 api 模块等底层文件静态引用，所以这条边界决定了 monaco 会不会回流到首屏
 * （守卫见 db/__tests__/monacoBoundary.test.ts）。
 *
 * 两侧都不能越界：
 * - 编辑器改静态引入 → 整份 monaco 挂到每个引用方（含首屏）的静态图上；
 * - 弹窗改动态引入 → 本文件那几 KB 省不掉（静态引用边还在），却把「点击 → 弹窗出现」变成毫无反馈的空白。
 *
 * 因此点击后弹窗立刻出现，编辑器区域由 main.ts 的首屏空闲预取兜住，正常情况下同一帧即可渲染，无需遮罩或占位。
 */
const MonacoEditorBox = (options: MonacoEditorDialogProps): void => {
    const boxId = options.useDrawer ? 'monaco-editor-drawer-id' : 'monaco-editor-dialog-id';

    // 同类型弹窗已打开则不重复创建
    if (document.getElementById(boxId)) {
        return;
    }

    // 默认值合入局部对象，不回写调用方传入的那个：编辑器每次变更都会经 onUpdate:modelValue 回流，
    // 写回入参意味着调用方复用同一个配置对象时会被上一次的内容污染
    const props: MonacoEditorDialogProps = {
        showConfirmButton: true,
        canChangeLang: true,
        useDrawer: false,
        ...options,
        content: options.content ?? '',
    };

    const container = document.createElement('div');
    container.id = boxId;
    // 先挂到 body 上占位，既让上面的重复打开判断立刻生效，也保证弹窗的 fixed 定位有正常的文档上下文
    document.body.appendChild(container);

    const destroy = () => {
        render(null, container);
        container.remove();
    };

    // 编辑器里的最新内容只在本函数内流动
    let content = props.content;

    try {
        render(
            h(MonacoEditorDialog, {
                ...props,
                modelValue: content,
                'onUpdate:modelValue': (value: string) => {
                    content = value;
                },
                visible: true,
                onClose: () => {
                    // 容器存在与否就是上面幂等判断的依据，卸载后必须一并移除
                    destroy();
                    props.closeFn?.();
                },
                onConfirm: () => {
                    let value = content;
                    if (props.language === 'json') {
                        let parsed;
                        try {
                            parsed = JSON.parse(value);
                            if (typeof parsed !== 'object') {
                                Msg.error('common.invalidJson');
                                return;
                            }
                        } catch (e) {
                            Msg.error('common.invalidJson');
                            return;
                        }
                        // 压缩json字符串
                        value = JSON.stringify(parsed);
                    }
                    props.confirmFn?.(value);
                },
            }),
            container
        );
    } catch (error) {
        // 组件挂载失败：dev 下 Vue 会把错误重抛出 render()，走这里撤掉容器后再交给调用方
        destroy();
        throw error;
    }

    // 同一条失败在生产构建里不会抛出（Vue 只 console.error），容器会空着留在 body 上。
    // 而「容器存在」就是上面幂等判断的依据，留着它等于此后每一次点击都被静默挡掉（表现为双击单元格彻底没反应），
    // 故挂载后必须校验真的渲染出了元素，否则撤销占位，让下一次点击还能重试。
    //
    // 判据成立的前提是弹窗根节点是容器内的真实元素（MonacoEditorDialog.vue 的 <div> 包裹；
    // el-dialog/el-drawer 的 teleport 只搬后代节点，不搬这个根）。若哪天弹窗改成 teleport/注释节点作根，
    // 这里会把正常渲染误判成失败，需连同判据一起改。
    if (!container.childElementCount) {
        destroy();
    }
};

export default MonacoEditorBox;
