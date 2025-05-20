import { BezierEdge, BezierEdgeModel, CircleNode, CircleNodeModel } from '@logicflow/core';
import PropSetting from './PropSetting.vue';
import { NodeTypeEnum } from '../enums';

class EdgeModel extends BezierEdgeModel {
    setAttributes() {
        this.offset = 20;

        const {
            properties: { isExecuted },
        } = this;

        if (isExecuted) {
            this.stroke = 'green';
        }
    }

    getEdgeStyle() {
        const style = super.getEdgeStyle();
        const { properties } = this;
        if (properties.isActived) {
            style.strokeDasharray = '4 4';
        }
        return style;
    }

    /**
     * 重写此方法，使保存数据是能带上锚点数据。
     */
    getData() {
        const data = super.getData();
        data.sourceAnchorId = this.sourceAnchorId;
        data.targetAnchorId = this.targetAnchorId;
        return data;
    }
}

const nodeType = NodeTypeEnum.Edge;

export default {
    type: nodeType.value,
    // 注册配置信息
    registerConf: {
        type: nodeType.value,
        model: EdgeModel,
        view: BezierEdge,
    },
    propSettingComp: PropSetting,
};
