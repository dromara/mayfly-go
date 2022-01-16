<template>
	<div class="dynamic-form">
		<el-form
			:model="form"
			ref="dynamicForm"
			:label-width="formInfo.labelWidth ? formInfo.labelWidth : '100px'"
			:size="formInfo.size ? formInfo.size : 'small'"
		>
			<el-row v-for="fr in formInfo.formRows" :key="fr.key">
				<el-col v-for="item in fr" :key="item.key" :span="item.span ? item.span : 24 / fr.length">
					<el-form-item :prop="item.name" :label="item.label" :label-width="item.labelWidth" :required="item.required" :rules="item.rules">
						<!-- input输入框 -->
						<el-input
							v-if="item.type === 'input'"
							v-model.trim="form[item.name]"
							:placeholder="item.placeholder"
							:type="item.inputType"
							clearable

							@change="item.change ? item.change(form) : ''"
						></el-input>

						<!-- 普通文本信息（可用于不可修改字段等） -->
						<span v-else-if="item.type === 'text'">{{ form[item.name] }}</span>

						<!-- select选择框 -->
						<!-- optionProps.label: 指定option中的label为options对象的某个属性值，默认就是label字段 -->
						<!-- optionProps.value: 指定option中的value为options对象的某个属性值，默认就是value字段 -->
						<el-select
							v-else-if="item.type === 'select'"
							v-model.trim="form[item.name]"
							:placeholder="item.placeholder"
							:filterable="item.filterable"
							:remote="item.remote"
							:remote-method="item.remoteMethod"
							@focus="item.focus ? item.focus(form) : ''"
							clearable
							:disabled="item.updateDisabled && form.id != null"
							style="width: 100%"
						>
							<el-option
								v-for="i in item.options"
								:key="i.key"
								:label="i[item.optionProps ? item.optionProps.label || 'label' : 'label']"
								:value="i[item.optionProps ? item.optionProps.value || 'value' : 'value']"
							></el-option>
						</el-select>
					</el-form-item>
				</el-col>
			</el-row>

			<el-row type="flex" justify="center">
				<slot name="btns" :submitDisabled="submitDisabled" :data="form" :submit="submit">
					<el-button @click="reset" size="small">重 置</el-button>
					<el-button type="primary" @click="submit" size="small">保 存</el-button>
				</slot>
			</el-row>
		</el-form>
	</div>
</template>

<script lang="ts">
import { watch, ref, toRefs, reactive, onMounted, defineComponent } from 'vue';
import { ElMessage } from 'element-plus';

export default defineComponent({
	name: 'DynamicForm',

	props: {
		formInfo: { type: Object },
		formData: { type: [Object, Boolean] },
	},

	setup(props: any, context) {
		const dynamicForm: any = ref();
		const state = reactive({
			form: {},
			submitDisabled: false,
		});

		watch(props.formData, (newValue, oldValue) => {
			if (props.formData) {
				state.form = { ...props.formData };
			}
		});

		const submit = () => {
			dynamicForm.value.validate((valid: boolean) => {
				if (valid) {
					// 提交的表单数据
					const subform = { ...state.form };
					const operation = state.form['id'] ? props.formInfo['updateApi'] : props.formInfo['createApi'];
					if (operation) {
						state.submitDisabled = true;
						operation.request(state.form).then(
							(res: any) => {
								ElMessage.success('保存成功');
								context.emit('submitSuccess', subform);
								state.submitDisabled = false;
								// this.cancel()
							},
							(e: any) => {
								state.submitDisabled = false;
							}
						);
					} else {
						ElMessage.error('表单未设置对应的提交权限');
					}
				} else {
					return false;
				}
			});
		};

		const reset = () => {
			context.emit('reset');
			resetFieldsAndData();
		};

		/**
		 * 重置表单以及表单数据
		 */
		const resetFieldsAndData = () => {
			// 对整个表单进行重置，将所有字段值重置为初始值并移除校验结果
			const df: any = dynamicForm;
			df.resetFields();
			// 重置表单数据
			state.form = {};
		};

		return {
			...toRefs(state),
			dynamicForm,
			submit,
			reset,
			resetFieldsAndData,
		};
	},
});
// @Component({
//   name: 'DynamicForm'
// })
// export default class DynamicForm extends Vue {
//   @Prop()
//   formInfo: object
//   @Prop()
//   formData: [object,boolean]|undefined

//   form = {}
//   submitDisabled = false

//   @Watch('formData', { deep: true })
//   onRoleChange() {
//     if (this.formData) {
//       this.form = { ...this.formData }
//     }
//   }

//   submit() {
//     const dynamicForm: any = this.$refs['dynamicForm']
//     dynamicForm.validate((valid: boolean) => {
//       if (valid) {
//         // 提交的表单数据
//         const subform = { ...this.form }
//         const operation = this.form['id']
//           ? this.formInfo['updateApi']
//           : this.formInfo['createApi']
//         if (operation) {
//           this.submitDisabled = true
//           operation.request(this.form).then(
//             (res: any) => {
//               ElMessage.success('保存成功')
//               this.$emit('submitSuccess', subform)
//               this.submitDisabled = false
//               // this.cancel()
//             },
//             (e: any) => {
//               this.submitDisabled = false
//             }
//           )
//         } else {
//           ElMessage.error('表单未设置对应的提交权限')
//         }
//       } else {
//         return false
//       }
//     })
//   }

//   reset() {
//     this.$emit('reset')
//     this.resetFieldsAndData()
//   }

//   /**
//    * 重置表单以及表单数据
//    */
//   resetFieldsAndData() {
//     // 对整个表单进行重置，将所有字段值重置为初始值并移除校验结果
//     const df: any = this.$refs['dynamicForm']
//     df.resetFields()
//     // 重置表单数据
//     this.form = {}
//   }

//   mounted() {
//     // 组件可能还没有初始化，第一次初始化的时候无法watch对象
//     this.form = { ...this.formData }
//   }

// }
</script>
