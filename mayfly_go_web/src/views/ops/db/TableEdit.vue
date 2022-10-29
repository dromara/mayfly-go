<template>
    <div>
        <el-dialog :title="`${title} 详情`" v-model="dialogVisible" :before-close="cancel" width="90%">
            <el-table @cell-click="cellClick" :data="data.res">
                <el-table-column :width="200" :prop="item" :label="item" v-for="item in data.colNames" :key="item">
                </el-table-column>
            </el-table>
        </el-dialog>
    </div>
</template>


<script lang="ts" setup>
import { watch, toRefs, reactive } from 'vue';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    title: {
        type: String,
    },
    data: {
        type: Object,
    },
})

//定义事件
const emit = defineEmits(['update:visible'])

const state = reactive({
    dialogVisible: false,
    data: {
        res: [],
        colNames: [],
    },
});

const {
    dialogVisible,
    data,
} = toRefs(state)

watch(props, async (newValue: any) => {
    state.dialogVisible = newValue.visible;
    state.data.res = newValue.data.res;
    state.data.colNames = newValue.data.colNames;
});

const cellClick = (row: any, column: any, cell: any) => {
    let isDiv = cell.children[0].tagName === 'DIV';
    let text = cell.children[0].innerText;
    let div = cell.children[0];
    if (isDiv) {
        let input = document.createElement('input');
        input.setAttribute('value', text);
        cell.replaceChildren(input);
        input.focus();
        input.addEventListener('blur', () => {
            div.innerText = input.value;
            cell.replaceChildren(div);
        });
    }
};

const cancel = () => {
    emit('update:visible', false);
};

</script>

