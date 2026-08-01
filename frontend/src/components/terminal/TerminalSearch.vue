<template>
    <div id="search-card" v-show="search.visible" @keydown.esc="closeSearch">
        <el-card :title="$t('components.terminal.search')" size="small">
            <!-- 搜索框 -->
            <el-input
                class="search-input"
                ref="searchInputRef"
                :placeholder="$t('components.terminal.serachPlaceholder')"
                v-model="search.value"
                @keyup.enter="searchKeywords('next')"
                clearable
            >
            </el-input>
            <!-- 选项 -->
            <div class="search-options">
                <el-row>
                    <el-col :span="12">
                        <el-checkbox class="usn" v-model="search.regex"> {{ $t('components.terminal.regexMatch') }} </el-checkbox>
                    </el-col>
                    <el-col :span="12">
                        <el-checkbox class="usn" v-model="search.words"> {{ $t('components.terminal.fullWordMatching') }} </el-checkbox>
                    </el-col>
                    <el-col :span="12">
                        <el-checkbox class="usn" v-model="search.matchCase"> {{ $t('components.terminal.caseSensitive') }} </el-checkbox>
                    </el-col>
                    <el-col :span="12">
                        <el-checkbox class="usn" v-model="search.incremental"> {{ $t('components.terminal.incrementalSearch') }} </el-checkbox>
                    </el-col>
                </el-row>
            </div>
            <!-- 按钮 -->
            <div class="search-buttons">
                <el-button class="terminal-search-button search-button-prev" type="primary" size="small" @click="searchKeywords('prev')">
                    {{ $t('components.terminal.previous') }}
                </el-button>
                <el-button class="terminal-search-button search-button-next" type="primary" size="small" @click="searchKeywords('next')">
                    {{ $t('components.terminal.next') }}
                </el-button>
                <el-button class="terminal-search-button search-button-next" type="primary" size="small" @click="closeSearch">
                    {{ $t('components.terminal.close') }}
                </el-button>
            </div>
        </el-card>
    </div>
</template>
<script lang="ts" setup>
import { ref, toRefs, nextTick, reactive, type PropType } from 'vue';
import { SearchAddon, ISearchOptions } from '@xterm/addon-search';
import { useI18n } from 'vue-i18n';
import { Msg } from '@/hooks/useI18n';

const { t } = useI18n();

const props = defineProps({
    searchAddon: {
        type: Object as PropType<SearchAddon | null>,
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

const searchInputRef = ref<HTMLInputElement | null>(null);

function open() {
    const visible = state.search.visible;
    state.search.visible = !visible;
    if (!visible) {
        nextTick(() => {
            searchInputRef.value?.focus();
        });
    }
}

function closeSearch() {
    state.search.visible = false;
    state.search.value = '';
    props.searchAddon?.clearDecorations();
    emit('close');
}

function searchKeywords(direction: 'next' | 'prev') {
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
    if (direction === 'next') {
        res = props.searchAddon?.findNext(state.search.value, getSearchOptions(option));
    } else {
        res = props.searchAddon?.findPrevious(state.search.value, getSearchOptions(option));
    }
    if (!res) {
        Msg.info('components.terminal.noMatchMsg');
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
