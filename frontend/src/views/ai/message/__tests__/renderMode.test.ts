/**
 * 消息渲染模式的缺省契约（M5）
 *
 * 只读运行轨迹与可对话会话共用同一套渲染组件，判定靠 provide/inject 穿透两层纯转发组件。
 * 漏接 provide 时缺省必须是「只读」：若缺省成 chat，只读轨迹里就会冒出可点的审批/编辑控件，
 * 而它只在服务端拒绝后表现为一条错误提示，界面上看不出任何异常——这类缺陷必须由单测钉住。
 */
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';
import { mount } from '@vue/test-utils';
import { provideMessageRenderMode, useMessageRenderMode, useReadonlyTranscript } from '../renderMode';

/** 探针组件：把两个判定结果渲染成文本，便于断言 */
const probe = defineComponent({
    setup() {
        const mode = useMessageRenderMode();
        const readonly = useReadonlyTranscript();
        return () => h('span', `${mode}:${readonly}`);
    },
});

describe('消息渲染模式缺省值', () => {
    it('宿主未 provide 时按只读轨迹处理（fail-closed）', () => {
        const wrapper = mount(probe);
        expect(wrapper.text()).toBe('transcript:true');
    });

    it('宿主 provide chat 后恢复可交互（缺省改严不能顺手关掉聊天侧）', () => {
        const host = defineComponent({
            setup() {
                provideMessageRenderMode('chat');
                return () => h(probe);
            },
        });
        const wrapper = mount(host);
        expect(wrapper.text()).toBe('chat:false');
    });

    it('宿主 provide transcript 时判定为只读', () => {
        const host = defineComponent({
            setup() {
                provideMessageRenderMode('transcript');
                return () => h(probe);
            },
        });
        const wrapper = mount(host);
        expect(wrapper.text()).toBe('transcript:true');
    });
});
