import { ElLink, ElText } from 'element-plus';
import { defineAsyncComponent, defineComponent, h } from 'vue';

type Size = 'large' | 'default' | 'small';

interface ComponentConfig {
    component: any;
    getDefaultProps?: (size: Size) => Record<string, any>;
}

const linkConf = {
    component: ElLink,
    getDefaultProps: (size: Size) => {
        return {
            type: 'primary',
            verticalAlign: 'baseline',
            style: {
                fontSize: size === 'small' ? '12px' : size === 'large' ? '16px' : '14px',
                verticalAlign: 'baseline',
            },
        };
    },
};

const components = {
    'el-link': linkConf,
    a: linkConf,

    'error-text': {
        component: ElText,
        getDefaultProps: (size: Size) => {
            return {
                type: 'danger',
                size,
            };
        },
    },

    'machine-info': {
        component: defineAsyncComponent(() => import('@/views/ops/machine/component/MachineDetail.vue')),
        getDefaultProps: (size: Size) => {
            return {
                size,
            };
        },
    },

    'db-info': {
        component: defineAsyncComponent(() => import('@/views/ops/db/component/DbDetail.vue')),
        getDefaultProps: (size: Size) => {
            return {
                size,
            };
        },
    },
} as Record<string, ComponentConfig>;

export const MessageRenderer = defineComponent({
    props: {
        content: String,
        size: {
            type: String as () => Size,
            default: 'default',
        },
    },
    setup(props) {
        const parseContent = (content: string) => {
            if (!content) {
                return [h('span', '')];
            }

            // 创建一个包装容器来处理HTML内容
            const container = document.createElement('div');
            container.innerHTML = content;

            const parseNode = (node: Node): any => {
                if (node.nodeType === Node.TEXT_NODE) {
                    return node.textContent;
                }

                if (node.nodeType === Node.ELEMENT_NODE) {
                    const element = node as HTMLElement;
                    const tagName = element.tagName.toLowerCase();
                    let attrs: Record<string, any> = {};

                    // 提取属性
                    for (let i = 0; i < element.attributes.length; i++) {
                        const attr = element.attributes[i];
                        attrs[attr.name] = attr.value;
                    }

                    const componentConf = components[tagName];
                    if (!componentConf) {
                        return h(tagName, attrs, Array.from(element.childNodes).map(parseNode));
                    }

                    // 存在默认组件配置，则合并
                    if (componentConf.getDefaultProps) {
                        const defaultProps = componentConf.getDefaultProps(props.size);
                        attrs = {
                            ...defaultProps,
                            ...attrs,
                        };
                    }

                    return h(componentConf.component, attrs, {
                        default: () => Array.from(element.childNodes).map(parseNode),
                    });
                }

                return '';
            };

            return Array.from(container.childNodes).map(parseNode);
        };

        return () => {
            // 根据 size 属性确定根元素的 class
            const rootClass = props.size === 'small' ? 'text-sm' : props.size === 'large' ? 'text-lg' : 'text-base';
            try {
                const elements = parseContent(props.content || '');
                return h('div', { class: rootClass }, elements);
            } catch (e) {
                console.error('消息渲染失败:', e);
                return h('div', { class: rootClass }, props.content || '');
            }
        };
    },
});
