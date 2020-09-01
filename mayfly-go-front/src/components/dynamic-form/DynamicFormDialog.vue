<template>
  <div class="form-dialog">
    <el-dialog :title="title" :visible="visible" :width="dialogWidth ? dialogWidth : '500px'">
      <dynamic-form
        ref="df"
        :form-info="formInfo"
        :form-data="formData"
        @submitSuccess="submitSuccess"
      >
        <template slot="btns" slot-scope="props">
          <slot name="btns">
            <el-button
              :disabled="props.submitDisabled"
              type="primary"
              @click="props.submit"
              size="mini"
            >保 存</el-button>
            <el-button :disabled="props.submitDisabled" @click="close()" size="mini">取 消</el-button>
          </slot>
        </template>
      </dynamic-form>
    </el-dialog>
  </div>
</template>

<script lang="ts">
import { Component, Vue, Prop } from 'vue-property-decorator'
import DynamicForm from './DynamicForm.vue'

@Component({
  name: 'DynamicFormDialog',
  components: {
    DynamicForm
  }
})
export default class DynamicFormDialog extends Vue {
  @Prop()
  visible: boolean|undefined
  @Prop()
  dialogWidth: string|undefined
  @Prop()
  title: string|undefined
  @Prop()
  formInfo: object|undefined
  @Prop()
  formData: [object,boolean]|undefined

  close() {
    // 更新父组件visible prop对应的值为false
    this.$emit('update:visible', false)
    // 关闭窗口，则将表单数据置为null
    this.$emit('update:formData', null)
    this.$emit('close')
    // 取消动态表单的校验以及form数据
    setTimeout(() => {
      const df: any = this.$refs.df
      df.resetFieldsAndData()
    }, 200)
  }

  submitSuccess(form: any) {
    this.$emit('submitSuccess', form)
    this.close()
  }
  
}
</script>
