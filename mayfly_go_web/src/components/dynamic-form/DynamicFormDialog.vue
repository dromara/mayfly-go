<template>
	<div class="form-dialog">
		<el-dialog :title="title" v-model="visible" :width="dialogWidth ? dialogWidth : '500px'">
			<dynamic-form ref="df" :form-info="formInfo" :form-data="formData" @submitSuccess="submitSuccess">
				<template #btns="props">
					<slot name="btns">
						<el-button :disabled="props.submitDisabled" type="primary" @click="props.submit" size="small">保 存</el-button>
						<el-button :disabled="props.submitDisabled" @click="close()" size="small">取 消</el-button>
					</slot>
				</template>
			</dynamic-form>
		</el-dialog>
	</div>
</template>

<script lang="ts">
import { watch, ref, toRefs, reactive, onMounted, defineComponent } from 'vue';
import DynamicForm from './DynamicForm.vue';
export default defineComponent({
	name: 'DynamicFormDialog',
	components: {
		DynamicForm,
	},
  
	props: {
		visible: { type: Boolean },
		dialogWidth: { type: String },
		title: { type: String },
		formInfo: { type: Object },
		formData: { type: [Object, Boolean] },
	},

	setup(props: any, context) {
		const df: any = ref();

		const close = () => {
			// 更新父组件visible prop对应的值为false
			context.emit('update:visible', false);
			// 关闭窗口，则将表单数据置为null
			context.emit('update:formData', null);
			context.emit('close');
			// 取消动态表单的校验以及form数据
			setTimeout(() => {
				df.resetFieldsAndData();
			}, 200);
		};

		const submitSuccess = (form: any) => {
			context.emit('submitSuccess', form);
			close();
		};

		return {
			df,
			close,
			submitSuccess,
		};
	},
});
</script>
