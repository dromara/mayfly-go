<template>
    <div>
        <div ref="jsoneditorVue" :style="{ height: height, width: width }"></div>
    </div>
</template>

<script lang="ts">
import { ref, toRefs, reactive, nextTick, watch, onMounted, onUnmounted, defineComponent } from 'vue';
import JSONEditor from 'jsoneditor';
import 'jsoneditor/dist/jsoneditor.min.css';

export default defineComponent({
    name: 'JsonEdit',
    components: {},
    props: {
        modelValue: {
            type: [String, Object],
        },
        height: {
            type: String,
            default: '500px',
        },
        width: {
            type: String,
            default: 'auto',
        },
        options: {
            type: Object,
            default: null,
        },
        currentMode: {
            type: String,
            default: 'tree',
        },
        modeList: {
            type: Array,
            default() {
                return ['tree', 'code', 'form', 'text', 'view'];
            },
        },
    },
    setup(props: any, { emit }) {
        let { modelValue, options, modeList, currentMode } = toRefs(props);

        const jsoneditorVue = ref(null)
        // 编辑器实例
        let editor = null as any;
        // 值类型
        let valueType = 'string';
        // 是否内部改变(即onChange事件双向绑定)，内部改变则不需要重新赋值给editor
        let internalChange = false;

        const state = reactive({
            height: '500px',
            width: 'auto',
        });

        onMounted(() => {
            state.width = props.width;
            state.height = props.height;

            init();
            setJson(modelValue.value);
        });

        onUnmounted(() => {
            editor?.destroy();
            editor = null;
        });

        watch(
            () => props.modelValue,
            (newValue) => {
                if (!editor) {
                    init();
                }
                setJson(newValue);
            }
        );

        const setJson = (value: any) => {
            if (internalChange) {
                return;
            }
            if (typeof value == 'string') {
                valueType = 'string';
                editor.set(JSON.parse(value));
            } else {
                valueType = 'object';
                editor.set(value);
            }
        };

        const onChange = () => {
            try {
                const json = editor.get();
                if (valueType == 'string') {
                    emit('update:modelValue', JSON.stringify(json));
                } else {
                    emit('update:modelValue', json);
                }
                emit('onChange', json);
                internalChange = true;
                nextTick(() => {
                    internalChange = false;
                });
            } catch (error) {}
        };

        const init = () => {
            console.log('init json editor');
            const finalOptions = {
                ...options.value,
                mode: currentMode.value,
                modes: modeList.value,
                onChange,
            };
            editor = new JSONEditor(jsoneditorVue.value, finalOptions);
        };

        return {
            ...toRefs(state),
            jsoneditorVue,
        };
    },
});
</script>

<style lang="scss">
div.jsoneditor-menu a.jsoneditor-poweredBy {
    display: none;
}
</style>