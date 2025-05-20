<template>
    <div :style="{ height: props.height }" class="flex flex-col" v-loading="saveing">
        <div class="h-[100vh]" ref="flowContainerRef"></div>
    </div>

    <PropSettingDrawer
        v-model:visible="propSettingEditor.visible"
        :disabled="props.disabled"
        :lf="lf"
        :node="propSettingEditor.node"
        :nodes="propSettingEditor.nodes"
    ></PropSettingDrawer>
</template>

<script lang="ts" setup>
import { onMounted, ref, useTemplateRef, watch } from 'vue';
import LogicFlow from '@logicflow/core';
import '@logicflow/core/lib/style/index.css';
import '@logicflow/extension/lib/style/index.css';
import { Control, DndPanel, Menu, SelectionSelect } from '@logicflow/extension';
import { initCustomNodes } from './node';
import PropSettingDrawer from './node/PropSettingDrawer.vue';
import { NodeTypeEnum } from './node/enums';
import { isTrue } from '@/common/assert';
import { useI18n } from 'vue-i18n';

import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';

const props = defineProps({
    disabled: {
        type: Boolean,
        default: false,
    },
    // 流程数据
    data: {
        type: [Object, String],
    },
    // 居中显示
    center: {
        type: Boolean,
        default: false,
    },
    height: {
        type: [Number, String],
        default: '100%',
    },
});

const { themeConfig } = storeToRefs(useThemeConfig());

const { t } = useI18n();

const flowContainerRef = useTemplateRef('flowContainerRef');

const emit = defineEmits(['save']);

const propSettingEditor = ref({
    visible: false,
    node: {},
    nodes: [],
});

const saveing = ref(false);

let lf: LogicFlow;

onMounted(() => {
    LogicFlow.use(DndPanel);
    LogicFlow.use(SelectionSelect);
    LogicFlow.use(Control);
    LogicFlow.use(Menu);

    const isDark = themeConfig.value.isDark;

    lf = new LogicFlow({
        container: flowContainerRef.value as HTMLElement,
        grid: true,
        nodeTextEdit: false, // 节点文本不可编辑
        edgeTextEdit: false, // 连线文本不可编辑
        edgeType: NodeTypeEnum.Edge.value,
        isSilentMode: props.disabled,
        background: {
            backgroundColor: isDark ? '#000000' : false,
        },
    });

    if (isDark) {
        lf.setTheme({
            baseEdge: {
                stroke: '#FFFFFF', // 连线颜色
                strokeWidth: 2,
            },
        });
    }

    initCustomNodes(lf, props.disabled);
    initControl();
    initEvent();

    // custom node -> logicflow node,userData由 lf.render(userData)传入
    lf.adapterIn = function (userData: any) {
        const nodes = userData.nodes?.map((node: any) => {
            const extra = node.extra;
            const lfNode = extra.lfNode;
            extra.lfNode = null; // 置空节点信息
            lfNode.properties = extra;
            return lfNode;
        });

        const edges = userData.edges?.map((edge: any) => {
            const extra = edge.extra;
            const lfEdge = extra.lfEdge;
            extra.lfEdge = null; // 置空连线信息
            lfEdge.properties = extra;
            return lfEdge;
        });

        // 这里把userData转换为LogicFlow支持的格式
        return { nodes, edges };
    };

    // logicflow node -> custom node
    lf.adapterOut = function (logicFlowData) {
        const flowNodes = logicFlowData.nodes.map((node) => {
            const nodeProps = node.properties;
            node.properties = {};
            const text = node.text;
            return {
                name: text instanceof Object ? text.value : text,
                key: node.id,
                type: node.type,
                extra: { ...nodeProps, lfNode: node },
            };
        });

        const flowEdges = logicFlowData.edges.map((edge) => {
            const edgeProps = edge.properties;
            edge.properties = {};
            const text = edge.text || '';
            return {
                name: text instanceof Object ? text.value : text,
                key: edge.id,
                sourceNodeKey: edge.sourceNodeId,
                targetNodeKey: edge.targetNodeId,
                extra: { ...edgeProps, lfEdge: edge },
            };
        });

        // 这里把LogicFlow生成的数据转换为后端需要的格式。
        return { edges: flowEdges, nodes: flowNodes };
    };

    renderData(props.data);
});

watch(
    () => props.data,
    (data) => {
        renderData(data);
    }
);

const renderData = (data: any) => {
    if (typeof data == 'string') {
        data = JSON.parse(data);
    }

    lf.render(data || {});
    if (props.center) {
        lf.translateCenter();
    }
};

const getLfExtension = (): any => {
    return lf.extension;
};

/**
 * 初始化控制面板
 */
function initControl() {
    if (props.disabled) {
        return;
    }
    const control = getLfExtension().control;
    // 控制面板-清空画布
    control.addItem({
        iconClass: 'lf-control-clear',
        title: 'clear',
        text: t('flow.clear'),
        onClick: (lf: LogicFlow, ev: any) => {
            lf.clearData();
        },
    });
    // 控制面板-保存
    control.addItem({
        iconClass: 'lf-control-save',
        title: '',
        text: t('common.save'),
        onClick: async (lf: LogicFlow, ev: any) => {
            validateFlow(lf.getGraphRawData());
            try {
                saveing.value = true;
                let graphData = lf.getGraphData();
                emit('save', graphData);
            } finally {
                saveing.value = false;
            }
        },
    });
}

function validateFlow(rawData: LogicFlow.GraphData) {
    // 提取节点和边
    const nodes = rawData.nodes || [];
    const edges = rawData.edges || [];

    // 查找开始节点和结束节点
    const startNodes = nodes.filter((node) => node.type === 'start');
    const endNodes = nodes.filter((node) => node.type === 'end');

    // 检查是否只有一个开始节点和结束节点
    isTrue(startNodes.length == 1, 'flow.mustOneStartNode');
    isTrue(endNodes.length == 1, 'flow.mustOneEndNode');

    const startNode = startNodes[0];
    const endNode = endNodes[0];

    // 检查开始节点是否有出线

    isTrue(
        edges.some((edge) => edge.sourceNodeId === startNode.id),
        'flow.mustOneOutEdgeForStartNode'
    );

    // 检查结束节点是否有入线
    isTrue(
        edges.some((edge) => edge.targetNodeId === endNode.id),
        'flow.mustOneInEdgeForEndNode'
    );
}

const initEvent = () => {
    const { eventCenter } = lf.graphModel;
    eventCenter.on('node:dbclick', (args: any) => {
        propSettingEditor.value.node = args.data;
        let graphData: any = lf.getGraphData();
        propSettingEditor.value.nodes = graphData['nodes'];
        propSettingEditor.value.visible = true;
    });

    eventCenter.on('edge:dbclick  ', (args: any) => {
        propSettingEditor.value.node = args.data;
        let graphData: any = lf.getGraphData();
        propSettingEditor.value.nodes = graphData['edges'];
        propSettingEditor.value.visible = true;
    });

    eventCenter.on('edge:add', (args: any) => {
        // 调整边类型
        // lf.changeEdgeType(args.data.id, NodeTypeEnum.Edge.value);
        // lf.setProperties(args.data.id, {
        //     condition: 'PASS',
        // });
    });

    // eventCenter.on('blank:click', () => {
    //     propSettingEditor.value.visible = false;
    // });
};
</script>

<style lang="scss">
.lf-control-save {
    background-image: url('data:image/svg+xml;charset=utf-8;base64,PHN2ZyB0PSIxNzQ1ODg5NTU4MjQ3IiBjbGFzcz0iaWNvbiIgdmlld0JveD0iMCAwIDEwMjQgMTAyNCIgdmVyc2lvbj0iMS4xIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHAtaWQ9IjE1ODQ4IiB3aWR0aD0iNDgiIGhlaWdodD0iNDgiPjxwYXRoIGQ9Ik01NjMuOTM1NTQgMTIyLjYxMTM2OGE0OC42MDk2MTkgNDguNjA5NjE5IDAgMCAwLTQ3Ljk3MDAxOCA0OS4zMTMxNzl2MzAuNjM2ODUyYTQ4LjYwOTYxOSA0OC42MDk2MTkgMCAwIDAgNDcuOTcwMDE4IDQ5LjMxMzE3OWMyNi40Nzk0NSAwIDQ3Ljk3MDAxOS0yMi4xMzAxNjkgNDcuOTcwMDE5LTQ5LjMxMzE3OXYtMzAuNjM2ODUyYTQ4LjYwOTYxOSA0OC42MDk2MTkgMCAwIDAtNDcuOTcwMDE5LTQ5LjMxMzE3OXoiIGZpbGw9IiMwMzAwMDAiIHAtaWQ9IjE1ODQ5Ij48L3BhdGg+PHBhdGggZD0iTTk5MS43MDAxODcgMjc3LjI2NjcwOGMwLTIuMDQ2NzIxLTAuODk1NDQtMy44Mzc2MDEtMS4xNTEyOC01LjgyMDM2MmE0OC45Mjk0MTkgNDguOTI5NDE5IDAgMCAwLTEzLjYyMzQ4NS00MC45OTgzNzZsLTIxNS43MzcxNjUtMjE1LjY3MzIwNGE0OS4zNzcxMzkgNDkuMzc3MTM5IDAgMCAwLTM2LjA3MzQ1NC0xNC40NTQ5NjZjLTAuNTExNjggMC0wLjg5NTQ0LTAuMjU1ODQtMS4zNDMxNi0wLjI1NTg0aC0zOC4yNDgwOTVMNjg1LjMzMTY2OCAwbC0wLjQ0NzcyIDAuMDYzOTZIMzM5LjExNjA1MkwzMzguOTI0MTcyIDBIODEuNjEyOTkybC0wLjcwMzU2IDAuMTI3OTJMODAuMjY5ODMxIDBhNDYuNDk4OTM4IDQ2LjQ5ODkzOCAwIDAgMC0zMC40NDQ5NzIgMTIuMDI0NDg1Yy0wLjk1OTQgMC44OTU0NC0yLjMwMjU2MSAxLjM0MzE2MS0zLjE5ODAwMSAyLjMwMjU2MS0xLjA4NzMyIDEuMDIzMzYtMS41OTkwMDEgMi40OTQ0NDEtMi40OTQ0NDEgMy42NDU3MjFhNDUuODU5MzM4IDQ1Ljg1OTMzOCAwIDAgMC0xMS44MzI2MDQgMjkuOTk3MjUybDAuMTI3OTIgMC42Mzk2LTAuMTI3OTIgMC43MDM1NnY5MjQuNzM0MDQxbDAuMTI3OTIgMC44MzE0ODEtMC4xMjc5MiAwLjU3NTY0YzAgMTIuMDI0NDg1IDQuOTI0OTIyIDIyLjY0MTg0OSAxMi4zNDQyODQgMzAuOTU2NjUyIDAuNzAzNTYgMC44MzE0OCAxLjE1MTI4IDEuODU0ODQxIDEuODU0ODQxIDIuNjIyMzYxIDEuMjc5MiAxLjM0MzE2MSAyLjk0MjE2MSAxLjk4Mjc2MSA0LjM0OTI4MiAzLjEzNDA0MWE0Ny4wNzQ1NzggNDcuMDc0NTc4IDAgMCAwIDI5LjQyMTYxMSAxMS4xOTMwMDVsMC41NzU2NDEtMC4xMjc5MiAwLjU3NTY0IDAuMTI3OTJoODYxLjE1Nzc3NmwwLjYzOTYtMC4xMjc5MiAwLjUxMTY4MSAwLjEyNzkyYTQ2LjQzNDk3OCA0Ni40MzQ5NzggMCAwIDAgMjkuMzU3NjUxLTExLjI1Njk2NWMxLjM0MzE2MS0xLjE1MTI4IDMuMTM0MDQxLTEuNzkwODgxIDQuMzQ5MjgyLTMuMTM0MDQxIDAuNzY3NTItMC43Njc1MiAxLjIxNTI0LTEuNzkwODgxIDEuODU0ODQxLTIuNjIyMzYxYTQ2LjgxODczOCA0Ni44MTg3MzggMCAwIDAgMTIuMjgwMzI0LTMwLjk1NjY1MmwtMC4xMjc5Mi0wLjYzOTYgMC4xMjc5Mi0wLjc2NzUyMSAwLjEyNzkyLTY5Ni43MTY1NTJ6TTM4Ni43NjYyNzEgOTUuOTQwMDM3aDI1MC40Njc0NTh2MTkzLjIyMzIzNkgzODYuNzY2MjcxVjk1Ljk0MDAzN3ogbTM1Mi40ODM2OTggODMxLjQ4MDMyNUgyODQuODEzOTkxdi0yNTAuODUxMjE4bDYyLjIzMzEwNS02Mi4yMzMxMDRoMzI5LjkwNTgwOGw2Mi4yOTcwNjUgNjIuMzYxMDI0djI1MC43MjMyOTh6IG0xNTYuNTEwMTgxIDBoLTYwLjU3MDE0NHYtMjY3LjA5NzA2NGMwLTAuNjM5Ni0wLjMxOTgtMS4yNzkyLTAuMzgzNzYtMS43OTA4ODFhNDguODY1NDU5IDQ4Ljg2NTQ1OSAwIDAgMC0xNC4zOTEwMDYtMzYuMzkzMjU0bC04OS4wOTYzMTQtODguOTA0NDM1YTQ5LjMxMzE3OSA0OS4zMTMxNzkgMCAwIDAtMzYuMDA5NDk0LTE0LjQ1NDk2NWMtMC41NzU2NCAwLTEuMDg3MzItMC4zMTk4LTEuNTM1MDQxLTAuMzE5OEgzMzAuODY1MjA5Yy0wLjYzOTYgMC0xLjIxNTI0IDAuMzE5OC0xLjg1NDg0IDAuMzgzNzZhNDkuMjQ5MjE5IDQ5LjI0OTIxOSAwIDAgMC0zNi40NTcyMTUgMTQuMzI3MDQ1bC04OS4wMzIzNTQgODguOTY4Mzk1YTQ5Ljg4ODgxOSA0OS44ODg4MTkgMCAwIDAtMTQuMzI3MDQ2IDM2LjU4NTEzNGMwIDAuNTExNjgtMC4zMTk4IDEuMDIzMzYtMC4zMTk4IDEuNTk5MDAxVjkyNy40MjAzNjJIMTI4LjIzOTg1di04MzEuNDgwMzI1aDE2Mi41ODYzODR2MjM5Ljg1MDA5NGwwLjEyNzkyIDAuNzAzNTYtMC4xMjc5MiAwLjYzOTYwMWMwIDExLjY0MDcyNSA0Ljc5NzAwMiAyMS45MzgyODkgMTEuOTYwNTI0IDMwLjI1MzA5MiAwLjgzMTQ4IDEuMDg3MzIgMS4zNDMxNjEgMi40MzA0ODEgMi4zNjY1MjEgMy4zODk4ODEgMS4wMjMzNiAxLjAyMzM2IDIuMzAyNTYxIDEuNDcxMDgxIDMuMzg5ODgyIDIuMzY2NTIxYTQ2LjY5MDgxOCA0Ni42OTA4MTggMCAwIDAgMzAuMzE3MDUxIDExLjk2MDUyNGwwLjcwMzU2MS0wLjEyNzkyIDAuNTc1NjQgMC4xMjc5MmgzNDMuNjU3MjE0bDAuNzY3NTItMC4xMjc5MiAwLjcwMzU2MSAwLjEyNzkyYTQ2LjM3MTAxOCA0Ni4zNzEwMTggMCAwIDAgMzAuMzE3MDUyLTExLjk2MDUyNGMxLjE1MTI4LTAuODk1NDQgMi4zNjY1MjEtMS4zNDMxNjEgMy4zODk4ODEtMi4zNjY1MjEgMS4wODczMi0wLjk1OTQgMS40NzEwODEtMi4zNjY1MjEgMi4zNjY1MjEtMy4zODk4ODFhNDYuNjI2ODU4IDQ2LjYyNjg1OCAwIDAgMCAxMi4wMjQ0ODQtMzAuMjUzMDkybC0wLjEyNzkyLTAuNjM5NjAxIDAuMTI3OTItMC43MDM1NlYxMjIuNDE5NDg4TDg5NS43NjAxNSAyODQuOTQxOTExVjkyNy40MjAzNjJ6IiBmaWxsPSIjMDMwMDAwIiBwLWlkPSIxNTg1MCI+PC9wYXRoPjwvc3ZnPg==');
}
.lf-control-clear {
    background-image: url('data:image/svg+xml;charset=utf-8;base64,PHN2ZyB0PSIxNzQ1ODg5NjY5MjU1IiBjbGFzcz0iaWNvbiIgdmlld0JveD0iMCAwIDEwMjQgMTAyNCIgdmVyc2lvbj0iMS4xIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHAtaWQ9IjE5MzQ1IiB3aWR0aD0iNDgiIGhlaWdodD0iNDgiPjxwYXRoIGQ9Ik05NzcuMTg4NTcxIDIxOC42OTcxNDNINjU2LjgyMjg1N1Y0Ni44MTE0MjljMC0yNi4zMzE0MjktMjEuMjExNDI5LTQ2LjgxMTQyOS00Ni44MTE0MjgtNDYuODExNDI5SDQxMy45ODg1NzFjLTI1LjYgMC00Ni44MTE0MjkgMjAuNDgtNDYuODExNDI4IDQ2LjgxMTQyOXYxNzEuODg1NzE0SDQ2LjgxMTQyOWE0Ni4wOCA0Ni4wOCAwIDAgMC00Ni44MTE0MjkgNDYuMDh2MjE4LjY5NzE0M2MwIDI1LjYgMjAuNDggNDYuODExNDI5IDQ2LjgxMTQyOSA0Ni44MTE0MjhINzMuMTQyODU3djQ0Ny42MzQyODZjMCAyNS42IDIxLjIxMTQyOSA0Ni4wOCA0Ni44MTE0MjkgNDYuMDhIOTA0LjA0NTcxNGMyNS42IDAgNDYuODExNDI5LTIwLjQ4IDQ2LjgxMTQyOS00Ni44MTE0MjlWNTMwLjI4NTcxNGgyNy4wNjI4NTdjMjUuNiAwIDQ2LjgxMTQyOS0yMC40OCA0Ni44MTE0MjktNDYuODExNDI4VjI2NC43NzcxNDNhNDcuMTc3MTQzIDQ3LjE3NzE0MyAwIDAgMC00Ny41NDI4NTgtNDYuMDh6IG0tMTE5Ljk1NDI4NSA3MTIuNDExNDI4aC0xMDIuNHYtMTE5Ljk1NDI4NWMwLTI1LjYtMjAuNDgtNDYuODExNDI5LTQ2LjgxMTQyOS00Ni44MTE0MjktMjUuNiAwLTQ2LjgxMTQyOSAyMC40OC00Ni44MTE0MjggNDYuODExNDI5djExOS45NTQyODVoLTEwMi40di0xMTkuOTU0Mjg1YzAtMjUuNi0yMC40OC00Ni44MTE0MjktNDYuODExNDI5LTQ2LjgxMTQyOXMtNDYuODExNDI5IDIwLjQ4LTQ2LjgxMTQyOSA0Ni44MTE0Mjl2MTE5Ljk1NDI4NWgtMTAyLjR2LTExOS45NTQyODVjMC0yNS42LTIwLjQ4LTQ2LjgxMTQyOS00Ni44MTE0MjgtNDYuODExNDI5LTI1LjYgMC00Ni44MTE0MjkgMjAuNDgtNDYuODExNDI5IDQ2LjgxMTQyOXYxMTkuOTU0Mjg1aC0xMDIuNFY1MzIuNDhoNjkxLjJ2Mzk4LjYyODU3MXpNOTIuODkxNDI5IDMxMS41ODg1NzFoMzIxLjA5NzE0MmMyNS42IDAgNDYuODExNDI5LTIwLjQ4IDQ2LjgxMTQyOS00Ni44MTE0MjhWOTIuODkxNDI5aDEwMi40djE3MS44ODU3MTRjMCAyNS42IDIwLjQ4IDQ2LjgxMTQyOSA0Ni44MTE0MjkgNDYuODExNDI4aDMyMS4wOTcxNDJ2MTI1LjA3NDI4Nkg5Mi44OTE0MjlWMzExLjU4ODU3MXoiIGZpbGw9IiMzMzMzMzMiIHAtaWQ9IjE5MzQ2Ij48L3BhdGg+PC9zdmc+');
}
</style>
