import { CircleNode, CircleNodeModel } from '@logicflow/core';
import PropSetting from './PropSetting.vue';
import { NodeTypeEnum } from '../enums';
import { HisProcinstOpState } from '@/views/flow/enums';

class StartModel extends CircleNodeModel {
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

class StartView extends CircleNode {}

const nodeType = NodeTypeEnum.Start;
const nodeTypeExtra = nodeType.extra;

export default {
    order: nodeTypeExtra.order,
    type: nodeType.value,
    // 注册配置信息
    registerConf: {
        type: nodeType.value,
        model: StartModel,
        view: StartView,
    },
    // 拖拽面板配置
    dndPanelConf: {
        type: nodeType.value,
        text: nodeTypeExtra.text,
        label: nodeType.label,
        icon: 'data:image/svg+xml;charset=utf-8;base64,PHN2ZyB0PSIxNzQ1ODg4MTUzNDkzIiBjbGFzcz0iaWNvbiIgdmlld0JveD0iMCAwIDEwNzggMTAyNCIgdmVyc2lvbj0iMS4xIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHAtaWQ9IjE5MjUiIHdpZHRoPSIzMiIgaGVpZ2h0PSIzMiI+PHBhdGggZD0iTTQ4Mi4wMjQ5NDcgNDYuOTUzNjM5QzMxMC44Mjg4ODkgNDYuMzg1MDM5IDE0Ni41NTkwMjMgMTU3LjAwMjg5MiA4MS41ODM0MjMgMzE1LjI0MDc1NmMtNjguNDAxOTE0IDE1Ni42OTg2NTctMzIuODYxNTY3IDM1MS4zNTY4MjIgODYuNzkwNzYgNDczLjU2NjkgMTE1LjA3ODg0NyAxMjMuODczMTYyIDMwNC42ODQ2MzUgMTY5LjMzOTMyOCA0NjMuNDgwMTgzIDExMS4zODEwNDggMTY0LjU3MDc3Ny01Ni4wNDA3OTcgMjg2LjA2NzAyLTIxNy42NTk4ODUgMjkyLjY3MTQxOC0zOTEuNjA4NzY1IDExLjA3NzczMy0xNzAuOTI2NDcyLTg5LjQ4MzMwNC0zNDEuNDU0NzM0LTI0My40NTk5OTEtNDE1LjkxMDAwN0M2MjAuNzcxODg3IDYyLjU4MjA3IDU1My40NDI2MjQgNDYuODkxNDYzIDQ4Ni4wNzAxNzEgNDYuOTg3ODEyYTM5NS4wMjk4NTcgMzk1LjAyOTg1NyAwIDAgMC00LjA0NTIyNC0wLjAzNDE3M3ogbTEyLjAzMjIwMiA0Ny40NDkxNDdjMTY3LjU3NzA0OCAwLjkyMDI5NyAzMjUuNDA0ODM2IDEyMi4xNDkzMjYgMzY4Ljc0NDIxMSAyODQuMzE0MjMyIDQ2LjA1MjgwMiAxNTUuMTU4MDI3LTE2LjIzODc5OCAzMzQuODAyMzk5LTE0OS45MzY2ODQgNDI2LjY5OTE2OEM1NzMuOTgzNDE3IDkwNy40OTYwMjEgMzY4LjAzMTA5MSA5MDAuMTEwODY2IDIzNi44ODE0NjQgNzg4LjI1NjE0MmMtMTMyLjE0MjA2Ny0xMDUuMzU3MTE2LTE3OS40NDIxODItMzAwLjIzNTk4MS0xMTEuNjQ0NDY1LTQ1NC43NzY1MjFDMTgzLjk5ODgxNyAxOTAuNjExNTE3IDMzMS41MDMwNTEgOTIuNjM3MTgzIDQ4Ni4wNzAxNzEgOTQuNDUwMjQ4YzIuNjY0NTQxLTAuMDQ2NTEzIDUuMzI3MTg0LTAuMDYxNzAxIDcuOTg2OTc4LTAuMDQ3NDYyeiIgcC1pZD0iMTkyNiI+PC9wYXRoPjwvc3ZnPg==',
        properties: nodeTypeExtra.defaultProp,
    },
    propSettingComp: PropSetting,
};
