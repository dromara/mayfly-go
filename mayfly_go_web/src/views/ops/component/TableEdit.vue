<template>
    <div>
        <el-dialog :title="`${title} 详情`" v-model="dialogVisible" :before-close="cancel" width="90%">
            <el-table @cell-click="cellClick" :data="data.res">
                <el-table-column :width="200" :prop="item" :label="item" v-for="item in data.colNames" :key="item"> </el-table-column>
            </el-table>
        </el-dialog>
    </div>
</template>


<script lang="ts">
import { watch, toRefs, reactive, defineComponent } from 'vue';

export default defineComponent({
    name: 'tableEdit',
    props: {
        visible: {
            type: Boolean,
        },
        title: {
            type: String,
        },
        data: {
            type: Object,
        },
    },
    setup(props: any, { emit }) {
        const state = reactive({
            dialogVisible: false,
            data: {
                res: [],
                colNames: [],
            },
        });

        watch(props, async (newValue) => {
            state.dialogVisible = newValue.visible;
            state.data.res = newValue.data.res;
            state.data.colNames = newValue.data.colNames;
        });
        const cellClick = (row: any, column: any, cell: any, event: any) => {
            console.log(cell.children[0].tagName);
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
        return {
            ...toRefs(state),
            cancel,
            cellClick,
        };
    },
});
</script>

