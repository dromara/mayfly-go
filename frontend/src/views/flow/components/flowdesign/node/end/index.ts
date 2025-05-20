import { CircleNode, CircleNodeModel } from '@logicflow/core';
import PropSetting from './PropSetting.vue';
import { NodeTypeEnum } from '../enums';
import { HisProcinstOpState } from '@/views/flow/enums';

class endModel extends CircleNodeModel {
    initNodeData(data: any) {
        super.initNodeData(data);
        this.r = 20;
    }

    getNodeStyle() {
        const style = super.getNodeStyle();
        const properties = this.properties;

        const opLog: any = properties.opLog;
        if (!opLog) {
            return style;
        }

        if (opLog.state == HisProcinstOpState.Completed.value) {
            style.stroke = 'green';
        } else if (opLog.state == HisProcinstOpState.Failed.value) {
            style.stroke = 'red';
        } else {
            style.stroke = 'rgb(24, 125, 255)';
        }

        return style;
    }
}

class endView extends CircleNode {}

const nodeType = NodeTypeEnum.End;
const nodeTypeExtra = nodeType.extra;

export default {
    order: 10,
    type: nodeType.value,
    // 注册配置信息
    registerConf: {
        type: nodeType.value,
        model: endModel,
        view: endView,
    },
    dndPanelConf: {
        type: nodeType.value,
        text: nodeTypeExtra.text,
        label: nodeType.label,
        icon: 'data:image/svg+xml;charset=utf-8;base64,PHN2ZyB0PSIxNzQ1ODg4Njg1MDk0IiBjbGFzcz0iaWNvbiIgdmlld0JveD0iMCAwIDEwNzkgMTAyNCIgdmVyc2lvbj0iMS4xIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHAtaWQ9IjI0NzEiIHdpZHRoPSIzMiIgaGVpZ2h0PSIzMiI+PHBhdGggZD0iTTQ4Mi4zODEzMjIgNDYuOTg2MTQ4QzI4NS4wMjc4MjUgNDUuNTE2NzkzIDk5Ljg4MTQ2MiAxOTUuMTI5MDU2IDU4LjM1NzkzNiAzODcuODYzNzY2Yy00MC4wNDk0MjQgMTY5LjI5ODExMyAzMS42NTc1ODMgMzU4LjE0MzAxOCAxNzUuMjc0NzI1IDQ1Ni44NDA4ODYgMTM5LjM5OTM5IDk5Ljg0MTE3NSAzMzguMDcwMzcxIDEwNy42NjU4NzEgNDgzLjA3NjY2MiAxNC45Mzc0OTYgMTQ1LjgwNzg4Ni04OS4yODg5NjMgMjMxLjY3MDI1Mi0yNjcuNTkxNjI0IDIwNC40NjM0NDktNDM3LjI1NjYtMjQuOTEzNTQ2LTE3NS44Nzc5MzktMTY1LjQwODMwOC0zMjguODQyMjg4LTMzOS44NzM4NDMtMzY0Ljk5NDMxNC0zMi40MzYzOTgtNy4yMzY2Ny02NS42OTE0NzgtMTAuNjI1Mjk5LTk4LjkxNzYwNy0xMC40MDUwODZ6IG0xMC40NzkxMjMgMTM3LjU3NzQwOWMxNDUuMDE4MTU1LTAuNDU5NDExIDI3OC4yMjgzMTMgMTE5LjU1ODM0NyAyOTMuMDA5MTkyIDI2My45MzE1MjQgMTguODY3MTY4IDEzNy45OTUwNTUtNzAuMDk5NTQ0IDI4Mi4zNDQ5NzYtMjAzLjg1Nzg2MiAzMjMuODMyNDMyLTEzMC41NTA1MTEgNDQuNDI2NjQxLTI4Ny45NzI3NTktMTIuMjY5MzA3LTM1NS45ODQwNzItMTMzLjM2NjMwMS03NS4yNTQxNTItMTI1LjYzNzk0Ny00My4yMzU4NzUtMzA0LjY3NTI4NSA3Ni4wMTM5ODQtMzkxLjkwMjU5NSA1NC4wNzkwMTUtNDEuNzI2NjUzIDEyMi41NDIxNDUtNjQuMDY1OTggMTkwLjgxODc1OC02Mi40OTUwNnoiIHAtaWQ9IjI0NzIiPjwvcGF0aD48L3N2Zz4=',
        properties: nodeTypeExtra.defaultProp,
    },
    propSettingComp: PropSetting,
};
