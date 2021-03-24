<template>
  <div>
    <div class="toolbar">
      <div class="fl">
        <el-button
          type="primary"
          icon="el-icon-plus"
          size="mini"
          @click="openFormDialog(false)"
          plain
          >添加</el-button
        >
        <el-button
          type="primary"
          icon="el-icon-edit"
          size="mini"
          :disabled="currentId == null"
          @click="openFormDialog(currentData)"
          plain
          >编辑</el-button
        >
        <el-button
          :disabled="currentId == null"
          @click="deleteMachine(currentId)"
          type="danger"
          icon="el-icon-delete"
          size="mini"
          >删除</el-button
        >
        <el-button
          type="success"
          :disabled="currentId == null"
          @click="fileManage(currentData)"
          size="mini"
          plain
          >文件管理</el-button
        >
      </div>

      <div style="float: right">
        <el-input
          placeholder="host"
          size="mini"
          style="width: 140px"
          v-model="params.host"
          @clear="search"
          plain
          clearable
        ></el-input>
        <el-button
          @click="search"
          type="success"
          icon="el-icon-search"
          size="mini"
        ></el-button>
      </div>
    </div>

    <el-table
      :data="data.list"
      stripe
      style="width: 100%"
      @current-change="choose"
    >
      <el-table-column label="选择" width="55px">
        <template slot-scope="scope">
          <el-radio v-model="currentId" :label="scope.row.id">
            <i></i>
          </el-radio>
        </template>
      </el-table-column>
      <el-table-column prop="name" label="名称" width></el-table-column>
      <el-table-column prop="ip" label="IP" width></el-table-column>
      <el-table-column prop="port" label="端口" width></el-table-column>
      <el-table-column prop="username" label="用户名"></el-table-column>
      <el-table-column prop="createTime" label="创建时间"></el-table-column>
      <el-table-column prop="updateTime" label="更新时间"></el-table-column>
      <el-table-column label="操作" min-width="200px">
        <template slot-scope="scope">
          <el-button
            type="primary"
            @click="info(scope.row.id)"
            :ref="scope.row"
            icom="el-icon-tickets"
            size="mini"
            plain
            >基本信息</el-button
          >
          <el-button
            type="primary"
            @click="monitor(scope.row.id)"
            :ref="scope.row"
            icom="el-icon-tickets"
            size="mini"
            plain
            >监控</el-button
          >
          <el-button
            type="success"
            @click="serviceManager(scope.row)"
            :ref="scope.row"
            size="mini"
            plain
            >服务管理</el-button
          >
          <el-button
            type="success"
            @click="showTerminal(scope.row)"
            :ref="scope.row"
            size="mini"
            plain
            >终端</el-button
          >
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      style="text-align: center"
      background
      layout="prev, pager, next, total, jumper"
      :total="data.total"
      :current-page.sync="params.pageNum"
      :page-size="params.pageSize"
    />

    <el-dialog title="基本信息" :visible.sync="infoDialog.visible" width="30%">
      <div style="white-space: pre-line">{{ infoDialog.info }}</div>
      <!-- <span slot="footer" class="dialog-footer">
        <el-button @click="dialogVisible = false">取 消</el-button>
        <el-button type="primary" @click="dialogVisible = false">确 定</el-button>
      </span>-->
    </el-dialog>

    <el-dialog
      @close="closeMonitor"
      title="监控信息"
      :visible.sync="monitorDialog.visible"
      width="60%"
    >
      <monitor ref="monitorDialog" :machineId="monitorDialog.machineId" />
    </el-dialog>

    <el-dialog
      title="终端"
      :visible.sync="terminalDialog.visible"
      width="70%"
      :close-on-click-modal="false"
      :modal="false"
      @close="closeTermnial"
    >
      <ssh-terminal ref="terminal" :socketURI="terminalDialog.socketUri" />
    </el-dialog>

    <service-manage
      :title="serviceDialog.title"
      :visible.sync="serviceDialog.visible"
      :machineId.sync="serviceDialog.machineId"
    />

    <dynamic-form-dialog
      :visible.sync="formDialog.visible"
      :title="formDialog.title"
      :formInfo="formDialog.formInfo"
      :formData.sync="formDialog.formData"
      @submitSuccess="submitSuccess"
    ></dynamic-form-dialog>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator'
import { DynamicFormDialog } from '@/components/dynamic-form'
import Monitor from './Monitor.vue'
import { machineApi } from './api'
import SshTerminal from './SshTerminal.vue'
import ServiceManage from './ServiceManage.vue';

@Component({
  name: 'MachineList',
  components: {
    DynamicFormDialog,
    Monitor,
    SshTerminal,
    ServiceManage
  },
})
export default class MachineList extends Vue {
  data = {
    list: [],
    total: 10,
  }
  infoDialog = {
    visible: false,
    info: '',
  }
  serviceDialog = {
    visible: false,
    machineId: 0,
    title: ''
  }
  monitorDialog = {
    visible: false,
    machineId: 0,
  }
  currentId = null
  currentData: any = null
  params = {
    pageNum: 1,
    pageSize: 10,
    host: null,
    clusterId: null,
  }
  dialog = {
    machineId: null,
    visible: false,
    title: '',
  }
  terminalDialog = {
    visible: false,
    socketUri: '',
  }
  formDialog = {
    visible: false,
    title: '',
    formInfo: {
      createApi: machineApi.save,
      updateApi: machineApi.update,
      formRows: [
        [
          {
            type: 'input',
            label: '名称：',
            name: 'name',
            placeholder: '请输入名称',
            rules: [
              {
                required: true,
                message: '请输入名称',
                trigger: ['blur', 'change'],
              },
            ],
          },
        ],
        [
          {
            type: 'input',
            label: 'ip：',
            name: 'ip',
            placeholder: '请输入ip',
            rules: [
              {
                required: true,
                message: '请输入ip',
                trigger: ['blur', 'change'],
              },
            ],
          },
        ],
        [
          {
            type: 'input',
            label: '端口号：',
            name: 'port',
            placeholder: '请输入端口号',
            inputType: 'number',
            rules: [
              {
                required: true,
                message: '请输入ip',
                trigger: ['blur', 'change'],
              },
            ],
          },
        ],
        [
          {
            type: 'input',
            label: '用户名：',
            name: 'username',
            placeholder: '请输入用户名',
            rules: [
              {
                required: true,
                message: '请输入用户名',
                trigger: ['blur', 'change'],
              },
            ],
          },
        ],
        [
          {
            type: 'input',
            label: '密码：',
            name: 'password',
            placeholder: '请输入密码',
            inputType: 'password',
          },
        ],
      ],
    },
    formData: { port: 22 },
  }

  mounted() {
    this.search()
  }

  choose(item: any) {
    if (!item) {
      return
    }
    this.currentId = item.id
    this.currentData = item
  }

  async info(id: number) {
    const res = await machineApi.info.request({ id })
    this.infoDialog.info = res
    this.infoDialog.visible = true
    // res.data
    // this.$alert(res, '机器基本信息', {
    //   type: 'info',
    //   dangerouslyUseHTMLString: false,
    //   closeOnClickModal: true,
    //   showConfirmButton: false,
    // }).catch((r) => {
    //   console.log(r)
    // })
  }

  monitor(id: number) {
    this.monitorDialog.machineId = id
    this.monitorDialog.visible = true
    // 如果重复打开同一个则开启定时任务
    const md: any = this.$refs['monitorDialog']
    if (md) {
      md.startInterval()
    }
  }

  closeMonitor() {
    // 关闭窗口，取消定时任务
    const md: any = this.$refs['monitorDialog']
    md.cancelInterval()
  }

  showTerminal(row: any) {
    this.terminalDialog.visible = true
    this.terminalDialog.socketUri = `ws://localhost:8888/api/machines/${row.id}/terminal`
  }

  closeTermnial() {
    this.terminalDialog.visible = false
    this.terminalDialog.socketUri = ''
    const t: any = this.$refs['terminal']
    t.closeAll()
  }

  openFormDialog(redis: any) {
    let dialogTitle
    if (redis) {
      this.formDialog.formData = this.currentData
      dialogTitle = '编辑机器'
    } else {
      this.formDialog.formData = { port: 22 }
      dialogTitle = '添加机器'
    }

    this.formDialog.title = dialogTitle
    this.formDialog.visible = true
  }

  async deleteMachine(id: number) {
    await machineApi.del.request({ id })
    this.$message.success('操作成功')
    this.search()
  }

  serviceManager(row: any) {
    this.serviceDialog.machineId = row.id
    this.serviceDialog.visible = true
    this.serviceDialog.title = `${row.name} => ${row.ip}`
  }

  submitSuccess() {
    this.currentId = null
    ;(this.currentData = null), this.search()
  }

  async search() {
    const res = await machineApi.list.request(this.params)
    this.data = res
  }
}
</script>

<style>
.el-dialog__body {
  padding: 2px 2px;
}
</style>
