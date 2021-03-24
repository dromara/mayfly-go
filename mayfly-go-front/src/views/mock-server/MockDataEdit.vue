<template>
  <div class="mock-data-dialog">
    <el-dialog
      :title="title"
      :visible="visible"
      :show-close="false"
      width="800px"
    >
      <el-form :model="form" ref="mockDataForm" label-width="70px" size="small">
        <el-form-item prop="method" label="方法名">
          <el-input
            :disabled="type == 1"
            v-model.trim="form.method"
            placeholder="请输入方法名"
          ></el-input>
        </el-form-item>

        <el-form-item prop="description" label="描述">
          <el-input
            v-model.trim="form.description"
            placeholder="请输入方法描述"
          ></el-input>
        </el-form-item>

        <el-form-item prop="description" label="生效用户">
          <el-select
            v-model="form.effectiveUser"
            multiple
            filterable
            allow-create
            default-first-option
            style="width:100%"
            placeholder="请选择或创建生效用户，空为所有用户都生效"
          > 
          </el-select>
        </el-form-item>

        <el-form-item prop="data" label="数据" id="jsonedit">
          <codemirror
            style="height: 400px"
            ref="cmEditor"
            v-model="form.data"
            :options="cmOptions"
          />
        </el-form-item>
      </el-form>

      <div style="text-align: center" class="dialog-footer">
        <el-button
          type="primary"
          :loading="btnLoading"
          @click="btnOk"
          size="mini"
          :disabled="submitDisabled"
          >确 定</el-button
        >
        <el-button @click="cancel()" :disabled="submitDisabled" size="mini"
          >取 消</el-button
        >
      </div>
    </el-dialog>
  </div>
</template>

<script lang="ts">
import { Component, Vue, Prop, Watch } from 'vue-property-decorator'
import { mockApi } from './api'
import { notEmpty } from '@/common/assert'
import Utils from '../../common/Utils'

import { codemirror } from 'vue-codemirror'
import 'codemirror/lib/codemirror.css'
import 'codemirror/theme/panda-syntax.css'
// import base style
require('codemirror/addon/selection/active-line')
import 'codemirror/mode/javascript/javascript.js'
import 'codemirror/addon/selection/active-line.js'
// 匹配括号
import 'codemirror/addon/edit/matchbrackets.js'

// json校验
require('script-loader!jsonlint')
import 'codemirror/addon/lint/lint'
import 'codemirror/addon/lint/lint.css'
import 'codemirror/addon/lint/json-lint'

@Component({
  name: 'MockDataEdit',
  components: {
    codemirror,
  },
})
export default class MockDataEdit extends Vue {
  @Prop()
  visible: boolean
  @Prop()
  data: [object, boolean]
  @Prop()
  title: string
  @Prop()
  type: number

  cmOptions = {
    tabSize: 2,
    mode: 'application/json',
    theme: 'panda-syntax',
    // mode: {
    //   // 模式, 可查看 codemirror/mode 中的所有模式
    //   name: 'javascript',
    //   json: true,
    // },
    line: true,
    // 开启校验
    lint: true,
    gutters: ['CodeMirror-lint-markers'],
    indentWithTabs: true,
    smartIndent: true,
    matchBrackets: true,
    autofocus: true,
    styleSelectedText: true,
    styleActiveLine: true, // 高亮选中行
    foldGutter: true, // 块槽
    hintOptions: {
      // 当匹配只有一项的时候是否自动补全
      completeSingle: false,
    },
  }

  submitDisabled = false

  form = {
    method: '',
    description: '',
    data: '',
    effectiveUser: null
  }

  btnLoading = false

  // rules = {
  //   method: [
  //     {
  //       required: true,
  //       message: '请输入方法名',
  //       trigger: ['change', 'blur'],
  //     },
  //   ],
  //   description: [
  //     {
  //       required: true,
  //       message: '请输入方法描述',
  //       trigger: ['change', 'blur'],
  //     },
  //   ],
  // }

  @Watch('data', { deep: true })
  onDataChange() {
    if (this.data) {
      Utils.copyProperties(this.data, this.form)
    } else {
      Utils.resetProperties(this.form)
      this.form.data = ''
    }
  }

  // mounted() {
  //   Utils.copyProperties(this.data, this.form)
  //   this.codemirror.setValue(
  //     JSON.stringify(JSON.parse(this.data['data']), null, 2)
  //   )
  // }

  get codemirror() {
    return this.$refs.cmEditor['codemirror']
  }

  btnOk() {
    const mockDataForm: any = this.$refs['mockDataForm']
    mockDataForm.validate((valid: any) => {
      if (valid) {
        notEmpty(this.form.method, '方法名不能为空')
        notEmpty(this.form.description, '描述不能为空')
        notEmpty(this.form.data, '数据不能为空')
        let res
        if (this.type == 1) {
          res = mockApi.update.request(this.form)
        } else {
          res = mockApi.create.request(this.form)
        }
        res.then(
          (res: any) => {
            this.$message.success('保存成功')
            this.$emit('submitSuccess')
            this.submitDisabled = false
            this.cancel()
          },
          (e: any) => {
            this.submitDisabled = false
          }
        )
      } else {
        return false
      }
    })
  }
  cancel() {
    this.$emit('update:visible', false)
    this.$emit('cancel')
    // this.codemirror.setValue('')
    // setTimeout(() => {
    //   this.resetForm()
    // }, 200)
  }

  resetForm() {
    const mockDataForm: any = this.$refs['mockDataForm']
    if (mockDataForm) {                                       
      mockDataForm.clearValidate()
    }
  }
}
</script>
<style lang="less">
// 	.m-dialog {
// 		.el-cascader {
// 			width: 100%;
// 		}
// 	}
#jsonedit {
  .CodeMirror {
    overflow-y: scroll !important;
    height: 400px !important;
  }
}
</style>
