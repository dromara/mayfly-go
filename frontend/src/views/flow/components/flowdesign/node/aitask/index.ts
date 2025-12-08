import { RectNode, RectNodeModel, h } from '@logicflow/core';
import PropSetting from './PropSetting.vue';
import { NodeTypeEnum } from '../enums';
import { HisProcinstOpState, ProcinstTaskStatus } from '@/views/flow/enums';

class AiTaskNodeModel extends RectNodeModel {
    initNodeData(data: any) {
        super.initNodeData(data);
        this.width = 100;
        this.height = 60;
        this.radius = 5;
    }

    getNodeStyle() {
        const style = super.getNodeStyle();
        const properties = this.properties;

        const opLog: any = properties.opLog;
        if (!opLog) {
            return style;
        }

        if (opLog.state == HisProcinstOpState.Completed.value && opLog.extra) {
            if (opLog.extra.approvalResult == ProcinstTaskStatus.Pass.value) {
                style.stroke = 'green';
            } else if (opLog.extra.approvalResult == ProcinstTaskStatus.Back.value) {
                style.stroke = '#e6a23c';
            } else {
                style.stroke = 'red';
            }
        } else if (opLog.state == HisProcinstOpState.Failed.value) {
            style.stroke = 'red';
        } else {
            style.stroke = 'rgb(100, 100, 255)'; // AI模型节点使用不同的颜色
        }

        return style;
    }
}

class AiTaskNodeView extends RectNode {
    // 获取标签形状的方法，用于在节点中添加一个自定义的 SVG 元素
    getShape() {
        // 获取XxxNodeModel中定义的形状属性
        const { model } = this.props;
        console.log(model.properties);
        const { x, y, width, height, radius } = model;
        // 获取XxxNodeModel中定义的样式属性
        const style = model.getNodeStyle();

        return h('g', {}, [
            h('rect', {
                ...style,
                x: x - width / 2,
                y: y - height / 2,
                width,
                height,
                rx: radius,
                ry: radius,
            }),
            h(
                'svg',
                {
                    x: x - width / 2 + 5,
                    y: y - height / 2 + 5,
                    width: 20,
                    height: 20,
                    viewBox: '0 0 1024 1024',
                },
                [
                    h('path', {
                        d: 'M517.818182 23.272727a488.727273 488.727273 0 1 0 488.727273 488.727273 488.727273 488.727273 0 0 0-488.727273-488.727273z m0 930.909091a442.181818 442.181818 0 1 1 442.181818-442.181818 442.181818 442.181818 0 0 1-442.181818 442.181818z',
                    }),
                    h('path', {
                        d: 'M490.356364 346.298182l-40.029091-18.618182-162.909091 349.090909 42.123636 19.781818 47.941818-102.865454h162.909091v-25.6l48.174546 126.836363 43.52-16.523636-128-337.454545z m-91.229091 200.610909l73.774545-158.254546 60.043637 158.254546zM704 337.454545h46.545455v349.09091h-46.545455z',
                    }),
                ]
            ),
        ]);
    }
}

const nodeType = NodeTypeEnum.AiTask;
const nodeTypeExtra = nodeType.extra;

export default {
    order: nodeTypeExtra.order,
    type: nodeType.value,
    // 注册配置信息
    registerConf: {
        type: nodeType.value,
        model: AiTaskNodeModel,
        view: AiTaskNodeView,
    },
    dndPanelConf: {
        type: nodeType.value,
        text: nodeTypeExtra.text,
        label: nodeType.label,
        icon: 'data:image/svg+xml;charset=utf-8;base64,PHN2ZyB0PSIxNzY0NDkwMzI5ODU0IiBjbGFzcz0iaWNvbiIgdmlld0JveD0iMCAwIDEwMjQgMTAyNCIgdmVyc2lvbj0iMS4xIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHAtaWQ9IjEzMTMxIiB3aWR0aD0iMzIiIGhlaWdodD0iMzIiPjxwYXRoIGQ9Ik01MTcuODE4MTgyIDIzLjI3MjcyN2E0ODguNzI3MjczIDQ4OC43MjcyNzMgMCAxIDAgNDg4LjcyNzI3MyA0ODguNzI3MjczIDQ4OC43MjcyNzMgNDg4LjcyNzI3MyAwIDAgMC00ODguNzI3MjczLTQ4OC43MjcyNzN6IG0wIDkzMC45MDkwOTFhNDQyLjE4MTgxOCA0NDIuMTgxODE4IDAgMSAxIDQ0Mi4xODE4MTgtNDQyLjE4MTgxOCA0NDIuMTgxODE4IDQ0Mi4xODE4MTggMCAwIDEtNDQyLjE4MTgxOCA0NDIuMTgxODE4eiIgcC1pZD0iMTMxMzIiPjwvcGF0aD48cGF0aCBkPSJNNDkwLjM1NjM2NCAzNDYuMjk4MTgybC00MC4wMjkwOTEtMTguNjE4MTgyLTE2Mi45MDkwOTEgMzQ5LjA5MDkwOSA0Mi4xMjM2MzYgMTkuNzgxODE4IDQ3Ljk0MTgxOC0xMDIuODY1NDU0aDE2Mi45MDkwOTF2LTI1LjZsNDguMTc0NTQ2IDEyNi44MzYzNjMgNDMuNTItMTYuNTIzNjM2LTEyOC0zMzcuNDU0NTQ1eiBtLTkxLjIyOTA5MSAyMDAuNjEwOTA5bDczLjc3NDU0NS0xNTguMjU0NTQ2IDYwLjA0MzYzNyAxNTguMjU0NTQ2ek03MDQgMzM3LjQ1NDU0NWg0Ni41NDU0NTV2MzQ5LjA5MDkxaC00Ni41NDU0NTV6IiBwLWlkPSIxMzEzMyI+PC9wYXRoPjwvc3ZnPg==',
        properties: nodeTypeExtra.defaultProp,
    },
    propSettingComp: PropSetting,
};
