import { computed, defineComponent, h, type PropType } from 'vue';
import { ElButton, ElPopconfirm, ElTooltip } from 'element-plus';
import { i18n } from '@/i18n';
import { runRowAction, actionLabel } from './rowAction';
import { actionTooltip, isIconOnlyAction, type ActionItem, type ActionOptions } from './index';

/**
 * 单格行内操作：按声明决定要不要「悬浮提示 / 就地气泡确认」两层包裹，
 * 以及这一格是「图标+文字」还是「只显图标」。
 *
 * 用渲染函数而非模板，是为了让按钮本体只写一遍——两种包装组合各写一份模板的话，
 * 以后给 ActionItem 加字段就得同步改多处，正是「渲染与计宽分叉」的老毛病。
 */
export default defineComponent({
    name: 'RowActionEntry',
    props: {
        item: { type: Object as PropType<ActionItem>, required: true },
        data: { type: Object as PropType<Record<string, unknown>>, required: true },
        /** 整列展示选项（图标模式 / 尺寸 / 外观）；本格没 icon 时依旧显文字 */
        options: { type: Object as PropType<ActionOptions>, default: undefined },
    },
    setup(props) {
        // 取全局 t 而非 useI18n：本组件不依赖插件注入的组件实例上下文，与列宽估算（index.ts）同一口径
        const t = i18n.global.t;

        const labelText = computed(() => actionLabel(props.item, props.data));

        /** 只显图标与否、以及该显示什么提示，都由 index.ts 的同一个判定给出（与计宽同源） */
        const iconOnly = computed(() => isIconOnlyAction(props.item, props.options));
        const tooltip = computed(() => actionTooltip(props.item, props.options));

        /** 外观→el-button 属性：link/text/plain 各自一个开关，不写就是 EP 默认带边框按钮 */
        const appearanceProps = computed(() => {
            switch (props.options?.appearance ?? 'link') {
                case 'text':
                    return { text: true };
                case 'plain':
                    return { plain: true };
                case 'bordered':
                    return {};
                default:
                    return { link: true };
            }
        });

        const button = () =>
            h(
                ElButton,
                {
                    ...appearanceProps.value,
                    // type 不预设默认值：页面没写就保持 el-button 默认色，迁移时不改变原有观感
                    type: props.item.type,
                    size: props.options?.size,
                    icon: props.item.icon,
                    loading: props.item.loading?.(props.data),
                    disabled: props.item.disabled?.(props.data),
                    // 纯图标格没有可见文案，用无障碍名称兼作名称；带文字的按钮文本已是无障碍名称，不重复声明
                    'aria-label': iconOnly.value ? labelText.value : undefined,
                    // 就地气泡确认：点按钮只负责弹气泡，执行由气泡的 confirm 事件触发，这里不能又跑一遍
                    onClick: (event: MouseEvent) => (props.item.confirm?.pop ? undefined : runRowAction(props.item, props.data, { event })),
                },
                () => (iconOnly.value ? undefined : labelText.value)
            );

        const withTooltip = () =>
            tooltip.value
                ? h(ElTooltip, { content: t(tooltip.value), placement: 'top', showAfter: 500 }, { default: button })
                : button();

        const withPopConfirm = () => {
            const confirm = props.item.confirm;
            if (!confirm?.pop) {
                return withTooltip();
            }
            return h(
                ElPopconfirm,
                { title: t(confirm.title, confirm.params?.(props.data) ?? {}), width: '220px', onConfirm: () => runRowAction(props.item, props.data) },
                { reference: withTooltip }
            );
        };

        return () => withPopConfirm();
    },
});
