<template>
  <div class="file-manage">
    <el-dialog
      :title="title"
      :visible.sync="visible"
      :show-close="true"
      :before-close="handleClose"
      width="60%"
    >
      <div class="toolbar">
        <div style="float: left">
          <el-select
            v-model="type"
            @change="getScripts"
            size="mini"
            placeholder="请选择"
          >
            <el-option :key="0" label="私有" :value="0"> </el-option>
            <el-option :key="1" label="公共" :value="1"> </el-option>
          </el-select>
        </div>
        <div style="float: right">
          <el-button
            @click="editScript(currentData)"
            :disabled="currentId == null"
            type="primary"
            :ref="currentData"
            icon="el-icon-tickets"
            size="mini"
            plain
            >查看</el-button
          >
          <el-button
            type="primary"
            @click="editScript(null)"
            icon="el-icon-plus"
            size="mini"
            plain
            >添加</el-button
          >
          <el-button
            :disabled="currentId == null"
            type="danger"
            :ref="currentData"
            @click="deleteRow(currentData)"
            icon="el-icon-delete"
            size="mini"
            plain
            >删除</el-button
          >
        </div>
      </div>

      <!-- <el-tabs type="border-card">
        <el-tab-pane label="私有"></el-tab-pane>
        <el-tab-pane label="公共"></el-tab-pane>
      </el-tabs> -->
      <el-table
        :data="scriptTable"
        @current-change="choose"
        stripe
        border
        size="mini"
        style="width: 100%"
      >
        <el-table-column label="选择" width="55px">
          <template slot-scope="scope">
            <el-radio v-model="currentId" :label="scope.row.id">
              <i></i>
            </el-radio>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="名称" :min-width="50">
        </el-table-column>
        <el-table-column
          prop="description"
          label="描述"
          :min-width="100"
          show-overflow-tooltip
        ></el-table-column>
        <el-table-column prop="name" label="类型" :min-width="50">
          <template slot-scope="scope">
            {{ enums.scriptTypeEnum.getLabelByValue(scope.row.type) }}
          </template>
        </el-table-column>
        <el-table-column label="操作">
          <template slot-scope="scope">
            <el-button
              v-if="scope.row.id == null"
              @click="addFiles(scope.row)"
              type="success"
              :ref="scope.row"
              icon="el-icon-success"
              size="mini"
              plain
              >确定</el-button
            >

            <el-button
              v-if="scope.row.id != null"
              @click="runScript(scope.row)"
              type="primary"
              :ref="scope.row"
              icon="el-icon-tickets"
              size="mini"
              plain
              >执行</el-button
            >
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog
      title="执行结果"
      :visible.sync="resultDialog.visible"
      width="40%"
    >
      <div style="white-space: pre-line; padding: 10px; color: #000000">
        {{ resultDialog.result }}
      </div>
    </el-dialog>

    <el-dialog
      v-if="terminalDialog.visible"
      title="终端"
      :visible.sync="terminalDialog.visible"
      width="70%"
      :close-on-click-modal="false"
      :modal="false"
      @close="closeTermnial"
    >
      <ssh-terminal
        ref="terminal"
        :cmd="terminalDialog.cmd"
        :machineId="terminalDialog.machineId"
        height="600px"
      />
    </el-dialog>

    <script-edit
      :visible.sync="editDialog.visible"
      :data.sync="editDialog.data"
      :title="editDialog.title"
      :machineId="editDialog.machineId"
      :isCommon="type == 1"
      @submitSuccess="submitSuccess"
    />
  </div>
</template>

<script lang="ts">
import { Component, Vue, Prop, Watch } from 'vue-property-decorator'
import SshTerminal from './SshTerminal.vue'
import { machineApi } from './api'
import enums from './enums'
import ScriptEdit from './ScriptEdit.vue'

@Component({
  name: 'ServiceManage',
  components: {
    SshTerminal,
    ScriptEdit,
  },
})
export default class ServiceManage extends Vue {
  @Prop()
  visible: boolean
  @Prop()
  machineId: number
  @Prop()
  title: string

  enums = enums
  type = 0
  currentId = null
  currentData = null
  editDialog = {
    visible: false,
    data: null,
    title: '',
    machineId: 9999999,
  }
  scriptTable: any[] = []
  resultDialog = {
    visible: false,
    result: '',
  }
  terminalDialog = {
    visible: false,
    cmd: '',
    machineId: 0,
  }

  @Watch('machineId', { deep: true })
  onDataChange() {
    if (this.machineId) {
      this.getScripts()
    }
  }

  async getScripts() {
    this.currentId = null
    this.currentData = null
    const machineId = this.type == 0 ? this.machineId : 9999999
    const res = await machineApi.scripts.request({ machineId: machineId })
    this.scriptTable = res.list
  }

  async runScript(script: any) {
    const noResult = script.type == enums.scriptTypeEnum['NO_RESULT'].value
    // 如果脚本类型为有结果类型，则显示结果信息
    if (script.type == enums.scriptTypeEnum['RESULT'].value || noResult) {
      const res = await machineApi.runScript.request({
        machineId: this.machineId,
        scriptId: script.id,
      })
      if (noResult) {
        this.$message.success('执行完成')
        return
      }
      this.resultDialog.result = res
      this.resultDialog.visible = true
      return
    }

    if (script.type == enums.scriptTypeEnum['REAL_TIME'].value) {
      this.terminalDialog.cmd = script.script
      this.terminalDialog.visible = true
      this.terminalDialog.machineId = this.machineId
      return
    }
  }

  closeTermnial() {
    this.terminalDialog.visible = false
    this.terminalDialog.machineId = 0
    // const t: any = this.$refs['terminal']
    // t.closeAll()
  }

  /**
   * 选择数据
   */
  choose(item: any) {
    if (!item) {
      return
    }
    this.currentId = item.id
    this.currentData = item
  }

  editScript(data: any) {
    this.editDialog.visible = true
    this.editDialog.machineId = this.machineId
    this.editDialog.data = data
  }

  submitSuccess() {
    // this.delChoose()
    // this.search()
    this.getScripts()
  }

  deleteRow(row: any) {
    this.$confirm(`此操作将删除 [${row.name}], 是否继续?`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }).then(() => {
      machineApi.deleteScript
        .request({
          machineId: this.machineId,
          scriptId: row.id,
        })
        .then((res) => {
          this.getScripts()
        })
      // 删除配置文件
    })
  }

  /**
   * 关闭取消按钮触发的事件
   */
  handleClose() {
    this.$emit('update:visible', false)
    this.$emit('update:machineId', null)
    this.$emit('cancel')
    this.scriptTable = []
  }
}
</script>
<style lang="less">
</style>
