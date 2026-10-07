/**
 * 选择域：文档多选的状态与有效性。
 *
 * 勾选态与「批量动作」分开维护，是因为结果一旦换页或重查，之前勾中的文档可能已经不在列表里：
 * 不清掉勾选就会让「删除选中」打到用户看不见的文档上。这里把裁剪做成一次调用，
 * 组件只需在拿到新结果时 prune 一次。
 */
import { computed, ref } from 'vue';

import type { MongoDoc } from '../../types';
import { docKey } from '../../docview/fields';

export function useDocSelection() {
    /** 勾选列是否显示：默认收起，避免日常阅读时多占一列 */
    const enabled = ref(false);
    const selected = ref<MongoDoc[]>([]);

    const selectedCount = computed(() => selected.value.length);

    /** 退出选择态即清空：留着上一次的勾选，下次进来会以为「这些是我刚选的」 */
    function toggleMode() {
        enabled.value = !enabled.value;
        if (!enabled.value) {
            selected.value = [];
        }
    }

    function setSelected(docs: MongoDoc[]) {
        selected.value = docs ?? [];
    }

    function clear() {
        selected.value = [];
    }

    /** 按新的结果集裁剪勾选（只按主键令牌比对，与渲染共用同一身份判据） */
    function prune(docs: MongoDoc[]) {
        if (!selected.value.length) {
            return;
        }
        const visible = new Set((docs ?? []).map((doc) => docKey(doc)));
        selected.value = selected.value.filter((doc) => visible.has(docKey(doc)));
    }

    return { enabled, selected, selectedCount, toggleMode, setSelected, clear, prune };
}
