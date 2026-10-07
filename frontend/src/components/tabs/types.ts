/**
 * 标签条的稳定契约（Contract）。
 *
 * 任何标签条实现（自研 VS Code 风格、Element Plus 适配器、或未来第三方实现）
 * 都必须满足同一组 props / emits / slots，页面只依赖本契约、不依赖具体实现，
 * 因此替换实现无需改动任何业务页面（开闭原则）。
 */
export interface TabItem {
    key: string;
    label: string;
    /** 悬浮提示全文（如「表名 | 表备注」）；缺省回落到 label */
    title?: string;
    closable?: boolean;
}

export interface TabBarProps {
    tabs: TabItem[];
    modelValue: string;
}

export type TabBarEmits = {
    'update:modelValue': [key: string];
    close: [key: string];
    contextmenu: [event: MouseEvent, tab: TabItem];
};

/**
 * Tabs 高层面板的 emits：在标签条契约之上补充 change。
 * v-model 只负责双向绑定；change 是「激活项确实切换」的通知，页面挂切换副作用用它，
 * 不必再监听较难记的 update:model-value。
 */
export interface TabsEmits extends TabBarEmits {
    change: [key: string];
}
