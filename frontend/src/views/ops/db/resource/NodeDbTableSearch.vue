<template>
    <div class="db-table-search">
        <el-input
            v-model="keyword"
            size="small"
            clearable
            :placeholder="$t('db.tableSearchPlaceholder')"
            @input="scheduleSearch"
            @clear="onSearch"
            @keyup.enter="onSearch"
        />
    </div>
</template>

<script lang="ts" setup>
import { inject, ref } from 'vue';

import { TreeApiKey } from '@/views/ops/resource/tree/context';
import type { TreeNode } from '@/views/ops/resource/tree/types';
import { tableResultsKey } from './helpers';

// 表名搜索框节点：仅当某库表数超阈值时出现。输入去抖后把过滤词写入【并列的结果节点】并只刷新它，
// 搜索框自身不在被刷新的子树内 → 不重挂载，光标/焦点/输入值全程保留；结果由服务端 LIKE 下推填充。
const props = defineProps<{ data: TreeNode }>();

const tree = inject(TreeApiKey)!;
const menuKey = props.data.params?.menuKey as string;
const resultsKey = tableResultsKey(menuKey);

// 本地态为准：不从 params 反向同步，避免刷新时回填覆盖用户正在输入的内容
const keyword = ref('');

const onSearch = () => {
    const results = tree.getNode(resultsKey);
    if (results) {
        results.params.tableFilter = keyword.value;
        tree.refresh(resultsKey);
        return;
    }
    // 结果节点尚未水合等异常：兜底刷新表菜单本身
    tree.refresh(menuKey);
};

let timer: ReturnType<typeof setTimeout> | undefined;
const scheduleSearch = () => {
    if (timer) {
        clearTimeout(timer);
    }
    timer = setTimeout(onSearch, 300);
};
</script>

<style lang="scss" scoped>
.db-table-search {
    width: 100%;
    padding: 0 4px;
}
</style>
