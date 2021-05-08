<template>
  <div class="mock-data-dialog">
    <el-dialog
      :title="title"
      :visible="visible"
      :show-close="false"
      :destroy-on-close="true"
      width="800px"
    >
      <el-form :model="form" ref="mockDataForm" label-width="70px" size="small">
        <el-form-item prop="method" label="名称">
          <el-input
            v-model.trim="form.name"
            placeholder="请输入名称"
          ></el-input>
        </el-form-item>

        <el-form-item prop="description" label="描述">
          <el-input
            v-model.trim="form.description"
            placeholder="请输入描述"
          ></el-input>
        </el-form-item>

        <el-form-item prop="type" label="类型">
          <el-select
            v-model="form.type"
            default-first-option
            style="width: 100%"
            placeholder="请选择类型"
          >
          <el-option
                v-for="item in enums.scriptTypeEnum"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              ></el-option>
          </el-select>
        </el-form-item>

        <el-form-item prop="script" label="内容" id="jsonedit">
          <codemirror
            style="height: 400px"
            ref="cmEditor"
            v-model="form.script"
            :options="cmOptions"
            @inputRead="inputRead"
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
          >关 闭</el-button
        >
      </div>
    </el-dialog>
  </div>
</template>

<script lang="ts">
import { Component, Vue, Prop, Watch } from 'vue-property-decorator'
import { machineApi } from './api'
import enums from './enums'
import { notEmpty } from '@/common/assert'
import Utils from '../../common/Utils'

import { codemirror } from 'vue-codemirror'
import 'codemirror/lib/codemirror.css'
import 'codemirror/theme/panda-syntax.css'
// import base style
require('codemirror/addon/selection/active-line')
import 'codemirror/mode/shell/shell.js'
import 'codemirror/addon/selection/active-line.js'
// 匹配括号
import 'codemirror/addon/edit/matchbrackets.js'

@Component({
  name: 'ScriptEdit',
  components: {
    codemirror,
  },
})
export default class ScriptEdit extends Vue {
  @Prop()
  visible: boolean
  @Prop()
  data: [object, boolean]
  @Prop()
  title: string
  @Prop()
  machineId: number
  // 是否公共脚本
  @Prop()
  isCommon: boolean

  enums= enums
  cmOptions = {
    tabSize: 2,
    mode: 'text/x-sh',
    theme: 'panda-syntax',
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
      completeSingle: true,
    },
  }

  submitDisabled = false

  form = {
    id: null,
    name: '',
    machineId: 0,
    description: '',
    script: '',
    type: null,
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
      this.form.script = ''
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
    this.form.machineId = this.isCommon ? 9999999 : this.machineId
    const mockDataForm: any = this.$refs['mockDataForm']
    mockDataForm.validate((valid: any) => {
      if (valid) {
        notEmpty(this.form.name, '名称不能为空')
        notEmpty(this.form.description, '描述不能为空')
        notEmpty(this.form.script, '内容不能为空')
        machineApi.saveScript.request(this.form).then(
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

  // 输入字符给提示
  inputRead(instance: any, changeObj: any) {
    // if (/^[a-zA-Z]/.test(changeObj.text[0])) {
    //   this.showHint()
    // }
    // const oldCuror = this.codemirror.getCursor()
    let otherChar
    const inputChar = changeObj.text[0]
    switch (inputChar) {
      case "'":
        otherChar = "'"
        break
      case '"':
        otherChar = '"'
        break
      case '{':
        otherChar = '}'
        break
      case '[':
        otherChar = ']'
        break
      default:
        return
    }
    this.codemirror.replaceRange(otherChar, this.codemirror.getCursor())
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
