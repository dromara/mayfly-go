<template>
    <div id="search-card" v-show="search.visible" @keydown.esc="closeSearch">
        <el-card title="搜索" size="small">
            <!-- 搜索框 -->
            <el-input
                class="search-input"
                ref="searchInputRef"
                placeholder="请输入查找内容,回车搜索"
                v-model="search.value"
                @keyup.enter.native="searchKeywords(true)"
                clearable
            >
            </el-input>
            <!-- 选项 -->
            <div class="search-options">
                <el-row>
                    <el-col :span="12">
                        <el-checkbox class="usn" v-model="search.regex"> 正则匹配 </el-checkbox>
                    </el-col>
                    <el-col :span="12">
                        <el-checkbox class="usn" v-model="search.words"> 单词全匹配 </el-checkbox>
                    </el-col>
                    <el-col :span="12">
                        <el-checkbox class="usn" v-model="search.matchCase"> 区分大小写 </el-checkbox>
                    </el-col>
                    <el-col :span="12">
                        <el-checkbox class="usn" v-model="search.incremental"> 增量查找 </el-checkbox>
                    </el-col>
                </el-row>
            </div>
            <!-- 按钮 -->
            <div class="search-buttons">
                <el-button class="terminal-search-button search-button-prev" type="primary" size="small" @click="searchKeywords(false)"> 上一个 </el-button>
                <el-button class="terminal-search-button search-button-next" type="primary" size="small" @click="searchKeywords(true)"> 下一个 </el-button>
                <el-button class="terminal-search-button search-button-next" type="primary" size="small" @click="closeSearch"> 关闭 </el-button>
            </div>
        </el-card>
    </div>
</template>
<script lang="ts" setup>
import { ref, toRefs, nextTick, reactive } from 'vue';
import { ElMessage } from 'element-plus';
import { SearchAddon, ISearchOptions } from 'xterm-addon-search';

const props = defineProps({
    searchAddon: {
        type: [SearchAddon],
        require: true,
    },
});

const state = reactive({
    search: {
        visible: false,
        value: '',
        regex: false,
        words: false,
        matchCase: false,
        incremental: false,
    },
});

const { search } = toRefs(state);

const emit = defineEmits(['close']);

const searchInputRef: any = ref(null);

function open() {
    const visible = state.search.visible;
    state.search.visible = !visible;
    console.log(state.search.visible);
    if (!visible) {
        nextTick(() => {
            searchInputRef.value.focus();
        });
    }
}

function closeSearch() {
    state.search.visible = false;
    state.search.value = '';
    props.searchAddon?.clearDecorations();
    emit('close');
}

function searchKeywords(direction: any) {
    if (!state.search.value) {
        return;
    }
    const option = {
        regex: state.search.regex,
        wholeWord: state.search.words,
        caseSensitive: state.search.matchCase,
        incremental: state.search.incremental,
    };
    let res;
    if (direction) {
        res = props.searchAddon?.findNext(state.search.value, getSearchOptions(option));
    } else {
        res = props.searchAddon?.findPrevious(state.search.value, getSearchOptions(option));
    }
    if (!res) {
        ElMessage.info('未查询到匹配项');
    }
}

const getSearchOptions = (searchOptions?: ISearchOptions): ISearchOptions => {
    return {
        ...searchOptions,
        decorations: {
            matchOverviewRuler: '#888888',
            activeMatchColorOverviewRuler: '#ffff00',
            matchBackground: '#888888',
            activeMatchBackground: '#ffff00',
        },
    };
};

defineExpose({ open });
</script>

<style lang="scss" scoped>
#search-card {
    position: absolute;
    top: 60px;
    right: 20px;
    z-index: 1200;
    width: 270px;

    .search-input {
        width: 240px;
    }

    .search-options {
        margin: 12px 0;
    }

    .search-buttons {
        margin-top: 5px;
        display: flex;
        justify-content: flex-end;
    }

    .terminal-search-button {
        margin-left: 10px;
    }
}
</style>
