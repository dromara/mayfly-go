<script lang="ts">
// 渲染函数：https://v3.cn.vuejs.org/guide/render-function.html
import { h, resolveComponent, defineComponent } from 'vue';
export default defineComponent({
	name: 'svgIcon',
	props: {
		// svg 图标组件名字
		name: {
			type: String,
		},
		// svg 大小
		size: {
			type: Number,
		},
		// svg 颜色
		color: {
			type: String,
		},
	},
	setup(props: any) {
        return () => h('i', { class: 'el-icon', style: `--font-size: ${props.size};--color: ${props.color}` }, [h(resolveComponent(`${props.name}`))]);
	},
});
</script>