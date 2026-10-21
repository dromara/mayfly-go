/**
 * 编辑器就绪契约（M7）
 *
 * 内核的 setText/setSegments 在编辑器未创建时是静默 no-op：回填类调用若不先判就绪就返回「成功」，
 * 调用方会顺手清空草稿信箱，草稿被吞且不留任何痕迹（跨页「带上下文追问」正是这条路径）。
 * 这里同时钉住两件事：未就绪时回填安全无副作用、就绪后回填真的落地。
 */
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';
import { mount } from '@vue/test-utils';
import { usePromptEditor } from '../usePromptEditor';

describe('usePromptEditor 就绪判定', () => {
    it('未就绪时回填是安全 no-op，就绪后回填真正落地', async () => {
        // 用持有对象而不是裸变量：TS 会把闭包内的赋值窄化为 never，拿不到真实类型
        const holder: { prompt?: ReturnType<typeof usePromptEditor> } = {};
        let readyBeforeMount = true;

        const host = defineComponent({
            setup() {
                holder.prompt = usePromptEditor({ placeholder: () => 'ph', skills: () => [] });
                readyBeforeMount = holder.prompt.ready();
                // onMounted 之前编辑器尚未创建：回填不得抛错，也不得产生内容
                expect(() => holder.prompt?.setText('draft before ready')).not.toThrow();
                expect(holder.prompt?.text.value).toBe('');
                // 容器 ref 必须绑到 setup 作用域里的同一个 Ref（写成字符串 ref 不解析路径，编辑器永不创建）
                return () => h('div', { ref: holder.prompt!.editorContainer });
            },
        });

        const wrapper = mount(host);
        await wrapper.vm.$nextTick();

        expect(readyBeforeMount).toBe(false);
        expect(holder.prompt?.ready()).toBe(true);

        holder.prompt?.setText('draft after ready');
        expect(holder.prompt?.text.value).toBe('draft after ready');
    });
});
