<template>
  <div class="file-manage">
    <el-dialog
      :title="title"
      :visible.sync="visible"
      :show-close="true"
      :before-close="handleClose"
      width="800px"
    >
      <div class="toolbar">
        <div style="float: right">
          <el-button
            type="primary"
            @click="add"
            icon="el-icon-plus"
            size="mini"
            plain
            >添加</el-button
          >
        </div>
      </div>
      <!-- <div style="float: right;">
       
      </div> -->
      <el-table :data="fileTable" stripe style="width: 100%">
        <el-table-column prop="name" label="名称" width>
          <template slot-scope="scope">
            <el-input
              v-model="scope.row.name"
              size="mini"
              :disabled="scope.row.id != null"
              clearable
            ></el-input>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="类型" width>
          <template slot-scope="scope">
            <el-select
              :disabled="scope.row.id != null"
              size="mini"
              v-model="scope.row.type"
              style="width: 100px"
              placeholder="请选择"
            >
              <el-option
                v-for="item in enums.FileTypeEnum"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              ></el-option>
            </el-select>
          </template>
        </el-table-column>
        <el-table-column prop="path" label="路径" width>
          <template slot-scope="scope">
            <el-input
              v-model="scope.row.path"
              :disabled="scope.row.id != null"
              size="mini"
              clearable
            ></el-input>
          </template>
        </el-table-column>
        <el-table-column label="操作" width>
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
              @click="getConf(scope.row)"
              type="primary"
              :ref="scope.row"
              icon="el-icon-tickets"
              size="mini"
              plain
              >查看</el-button
            >
            <el-button
              type="danger"
              :ref="scope.row"
              @click="deleteRow(scope.$index, scope.row)"
              icon="el-icon-delete"
              size="mini"
              plain
              >删除</el-button
            >
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog
      :title="tree.title"
      :visible.sync="tree.visible"
      :close-on-click-modal="false"
      width="650px"
    >
      <div style="height: 45vh; overflow: auto">
        <el-tree
          ref="fileTree"
          :load="loadNode"
          :props="props"
          lazy
          node-key="id"
          :expand-on-click-node="false"
        >
          <span class="custom-tree-node" slot-scope="{ node, data }">
            <span v-if="data.type == 'd' && !node.expanded">
              <i class="el-icon-folder"></i>
            </span>
            <span v-if="data.type == 'd' && node.expanded">
              <i class="el-icon-folder-opened"></i>
            </span>
            <span v-if="data.type == '-'">
              <i class="el-icon-document"></i>
            </span>

            <span style="display: inline-block; width: 400px">
              {{ node.label }}
              <span style="color: #67c23a" v-if="data.type == '-'"
                >&nbsp;&nbsp;[{{ formatFileSize(data.size) }}]</span
              >
            </span>

            <span>
              <el-link
                @click="getFileContent(tree.folder.id, data.path)"
                v-if="data.type == '-' && data.size < 5 * 1024 * 1024"
                type="info"
                icon="el-icon-view"
                :underline="false"
              />

              <el-upload
                :on-success="uploadSuccess"
                :headers="{ token }"
                :data="{
                  fileId: tree.folder.id,
                  path: data.path,
                  machineId: machineId,
                }"
                :action="getUploadFile({ path: data.path })"
                :show-file-list="false"
                name="file"
                multiple
                :limit="100"
                style="display: inline-block; margin-left: 2px"
              >
                <el-link
                  v-if="data.type == 'd'"
                  icon="el-icon-upload"
                  :underline="false"
                />
              </el-upload>

              <el-link
                v-if="data.type == '-'"
                @click="downloadFile(node, data)"
                type="danger"
                icon="el-icon-download"
                :underline="false"
                style="margin-left: 2px"
              />

              <el-link
                v-if="!dontOperate(data)"
                @click="deleteFile(node, data)"
                type="danger"
                icon="el-icon-delete"
                :underline="false"
                style="margin-left: 2px"
              />
            </span>
          </span>
        </el-tree>
      </div>
    </el-dialog>

    <el-dialog
      :title="fileContent.dialogTitle"
      :visible.sync="fileContent.contentVisible"
      :close-on-click-modal="false"
      width="900px"
    >
      <div>
        <codemirror
          style="height: 500px"
          ref="cmEditor"
          v-model="fileContent.content"
          :options="cmOptions"
        />
      </div>

      <div slot="footer" class="dialog-footer">
        <el-button @click="fileContent.contentVisible = false" size="mini"
          >取 消</el-button
        >
        <el-button type="primary" @click="updateContent" size="mini"
          >确 定</el-button
        >
      </div>
    </el-dialog>
  </div>
</template>

<script lang="ts">
import { Component, Vue, Prop, Watch } from 'vue-property-decorator'
import enums from './enums'
import { machineApi } from './api'

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
  name: 'FileManage',
  components: {
    codemirror,
  },
})
export default class FileManage extends Vue {
  @Prop()
  visible: boolean
  @Prop()
  machineId: [number]
  @Prop()
  title: string

  addFile = machineApi.addConf
  delFile = machineApi.delConf
  updateFileContent = machineApi.updateFileContent
  uploadFile = machineApi.uploadFile
  files = machineApi.files
  enums = enums
  activeName = 'conf-file'
  token = sessionStorage.getItem('token')
  form = {
    id: null,
    type: null,
    name: '',
    remark: '',
  }
  fileTable: any[] = []
  btnLoading = false
  fileContent = {
    fileId: 0,
    content: '',
    contentVisible: false,
    dialogTitle: '',
    path: '',
  }
  tree = {
    title: '',
    visible: false,
    folder: { id: 0 },
    node: {
      childNodes: [],
    },
    resolve: {},
  }
  props = {
    label: 'name',
    children: 'zones',
    isLeaf: 'leaf',
  }

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

  get codemirror() {
    return this.$refs.cmEditor['codemirror']
  }

  @Watch('machineId', { deep: true })
  onDataChange() {
    if (this.machineId) {
      this.getFiles()
    }
  }

  async getFiles() {
    const res = await this.files.request({ id: this.machineId })
    this.fileTable = res.list
  }

  /**
   * tab切换触发事件
   * @param {Object} tab
   * @param {Object} event
   */
  // handleClick(tab, event) {
  //   // if (tab.name == 'file-manage') {
  //   //   this.fileManage.node.childNodes = [];
  //   //   this.loadNode(this.fileManage.node, this.fileManage.resolve);
  //   // }
  // }

  add() {
    // 往数组头部添加元素
    this.fileTable = [{}].concat(this.fileTable)
  }

  async addFiles(row: any) {
    row.machineId = this.machineId
    await this.addFile.request(row)
    this.$message.success('添加成功')
    this.getFiles()
  }

  deleteRow(idx: any, row: any) {
    if (row.id) {
      this.$confirm(`此操作将删除 [${row.name}], 是否继续?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }).then(() => {
        // 删除配置文件
        this.delFile
          .request({
            machineId: this.machineId,
            id: row.id,
          })
          .then((res) => {
            this.fileTable.splice(idx, 1)
          })
      })
    } else {
      this.fileTable.splice(idx, 1)
    }
  }

  getConf(row: any) {
    if (row.type == 1) {
      this.tree.folder = row
      this.tree.title = row.name
      const treeNode = (this.tree.node.childNodes = [])
      this.loadNode(this.tree.node, this.tree.resolve)
      this.tree.visible = true
      return
    }
    this.getFileContent(row.id, row.path)
  }

  async getFileContent(fileId: number, path: string) {
    const res = await machineApi.fileContent.request({
      fileId,
      path,
      machineId: this.machineId,
    })
    this.fileContent.content = res
    this.fileContent.fileId = fileId
    this.fileContent.dialogTitle = path
    this.fileContent.path = path
    this.fileContent.contentVisible = true
  }

  async updateContent() {
    await this.updateFileContent.request({
      content: this.fileContent.content,
      id: this.fileContent.fileId,
      path: this.fileContent.path,
      machineId: this.machineId,
    })
    this.$message.success('修改成功')
    this.fileContent.contentVisible = false
    this.fileContent.content = ''
  }

  /**
   * 关闭取消按钮触发的事件
   */
  handleClose() {
    this.$emit('update:visible', false)
    this.$emit('update:machineId', null)
    this.$emit('cancel')
    this.activeName = 'conf-file'
    this.fileTable = []
    this.tree.folder = { id: 0 }
  }

  /**
   * 加载文件树节点
   * @param {Object} node
   * @param {Object} resolve
   */
  async loadNode(node: any, resolve: any) {
    if (typeof resolve !== 'function') {
      return
    }

    const folder: any = this.tree.folder
    if (node.level === 0) {
      this.tree.node = node
      this.tree.resolve = resolve

      // let folder: any = this.tree.folder
      const path = folder ? folder.path : '/'
      return resolve([
        {
          name: path,
          type: 'd',
          path: path,
        },
      ])
    }

    let path
    const data = node.data
    // 只有在第一级节点时，name==path，即上述level==0时设置的
    if (!data || data.name == data.path) {
      path = folder.path
    } else {
      path = data.path
    }

    const res = await machineApi.lsFile.request({
      fileId: folder.id,
      machineId: this.machineId,
      path,
    })
    for (const file of res) {
      const type = file.type
      if (type != 'd') {
        file.leaf = true
      }
    }
    return resolve(res)
  }

  deleteFile(node: any, data: any) {
    const file = data.path
    this.$confirm(`此操作将删除 [${file}], 是否继续?`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
      .then(() => {
        machineApi.rmFile
          .request({
            fileId: this.tree.folder.id,
            path: file,
            machineId: this.machineId,
          })
          .then((res) => {
            this.$message.success('删除成功')
            const fileTree: any = this.$refs.fileTree
            fileTree.remove(node)
          })
      })
      .catch(() => {
        // skip
      })
  }

  downloadFile(node: any, data: any) {
    const a = document.createElement('a')
    // a.setAttribute('target', '_blank')
    a.setAttribute(
      'href',
      process.env.VUE_APP_BASE_API +
        `/machines/${this.machineId}/files/${this.tree.folder.id}/read?type=1&path=${data.path}`
    )
    a.click()
  }

  getUploadFile(data: any) {
    return (
      process.env.VUE_APP_BASE_API +
      `/machines/${this.machineId}/files/${
        this.tree.folder.id
      }/upload?token=${sessionStorage.getItem('token')}`
    )
  }

  uploadSuccess(res: any) {
    if (res.code == 200) {
      this.$message.success('文件上传中...')
    } else {
      this.$message.error(res.msg)
    }
  }

  dontOperate(data: any) {
    const path = data.path
    const ls = [
      '/',
      '//',
      '/usr',
      '/usr/',
      '/usr/bin',
      '/opt',
      '/run',
      '/etc',
      '/proc',
      '/var',
      '/mnt',
      '/boot',
      '/dev',
      '/home',
      '/media',
      '/root',
    ]
    return ls.indexOf(path) != -1
  }

  /**
   * 格式化文件大小
   * @param {*} value
   */
  formatFileSize(size: any) {
    const value = Number(size)
    if (size && !isNaN(value)) {
      const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB', 'BB']
      let index = 0
      let k = value
      if (value >= 1024) {
        while (k > 1024) {
          k = k / 1024
          index++
        }
      }
      return `${k.toFixed(2)}${units[index]}`
    }
    return '-'
  }
}
</script>
<style lang="less">
.CodeMirror {
  height: 500px;
}
</style>
