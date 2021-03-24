<template>
  <div>
    <div class="toolbar">
      <div class="fl">
        <el-button
          @click="search"
          type="primary"
          icon="el-icon-refresh"
          size="mini"
          plain
          >刷新</el-button
        >
        <el-button
          @click="editData(null)"
          type="primary"
          icon="el-icon-plus"
          size="mini"
          plain
          >添加</el-button
        >
        <el-button
          type="primary"
          icon="el-icon-edit"
          size="mini"
          :disabled="currentData == null"
          @click="editData(currentData)"
          plain
          >查看&编辑</el-button
        >
        <el-button
          :disabled="currentData == null"
          type="danger"
          icon="el-icon-delete"
          size="mini"
          @click="deleteData()"
          >删除</el-button
        >
      </div>

      <div style="float: right">
        <el-input
          placeholder="方法名过滤"
          size="mini"
          style="width: 140px"
          @clear="search"
          plain
          v-model="queryParam"
          clearable
        ></el-input>
        <el-button
          @click="filterData"
          type="success"
          icon="el-icon-search"
          size="mini"
        ></el-button>
      </div>
    </div>

    <el-table
      :data="data"
      @current-change="choose"
      border
      stripe
      style="width: 100%"
    >
      <el-table-column label="选择" width="55px">
        <template slot-scope="scope">
          <el-radio v-model="currentMethod" :label="scope.row.method">
            <i></i>
          </el-radio>
        </template>
      </el-table-column>
      <el-table-column
        prop="method"
        label="方法名"
        :min-width="50"
      ></el-table-column>
      <el-table-column
        prop="description"
        label="描述"
        :min-width="50"
      ></el-table-column>
      <el-table-column prop="description" label="状态" :width="75">
        <template slot-scope="scope">
          <el-tooltip
            :content="scope.row.enable == 1 ? '启用' : '禁用'"
            placement="top"
          >
            <el-switch
              v-model="scope.row.enable"
              :active-value="1"
              active-color="#13ce66"
              inactive-color="#ff4949"
              @change="changeStatus(scope.row)"
            ></el-switch>
          </el-tooltip>
        </template>
      </el-table-column>
      <el-table-column
        prop="effectiveUser"
        label="生效用户"
        :min-width="70"
        show-overflow-tooltip
      >
        <template slot-scope="scope">
          {{ showEffectiveUser(scope.row.effectiveUser) }}
        </template>
      </el-table-column>
      <el-table-column
        prop="data"
        label="数据"
        min-width="100"
        show-overflow-tooltip
      ></el-table-column>
    </el-table>

    <mock-data-edit
      :visible.sync="editDialog.visible"
      :data.sync="editDialog.data"
      :title="editDialog.title"
      :type="editDialog.type"
      @cancel="closeEditDialog"
      @submitSuccess="submitSuccess"
    />
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator'
import { mockApi } from './api'
import MockDataEdit from './MockDataEdit.vue'

@Component({
  name: 'MockDataList',
  components: {
    MockDataEdit,
  },
})
export default class MockDataList extends Vue {
  data: Array<any> = []

  currentData: any = null
  currentMethod = null
  queryParam = null

  editDialog = {
    data: {},
    visible: false,
    title: '',
    type: 0,
  }

  choose(item: any) {
    if (!item) {
      return
    }
    this.currentData = item
    this.currentMethod = item.method
  }

  editData(data: any) {
    this.editDialog.data = data
    if (data) {
      this.editDialog.title = '修改mock数据'
      this.editDialog.type = 1
    } else {
      this.editDialog.title = '新增mock数据'
      this.editDialog.type = 0
      // this.delChoose()
    }
    this.editDialog.visible = true
  }

  closeEditDialog() {
    // this.currentData.opentime = Date.now()
    // this.editDialog.data = this.currentData
  }

  delChoose() {
    this.currentMethod = null
    this.currentData = null
  }

  changeStatus(row: any) {
    const enable = row.enable
    row.enable = enable ? 1 : 0
    mockApi.update
      .request(row)
      .then((res) => {
        this.$message.success('操作成功')
      })
      .catch((e) => {
        row.enable = enable
        this.$message.success('操作失败')
      })
  }

  showEffectiveUser(users: string[]) {
    if (!users || users.length == 0) {
      return '全部用户'
    }
    return users.join(', ')
  }

  deleteData() {
    mockApi.delete.request({ method: this.currentMethod }).then((res) => {
      this.$message.success('删除成功')
      this.search()
    })
  }

  submitSuccess() {
    this.delChoose()
    this.search()
  }

  mounted() {
    this.search()
  }

  filterData() {
    this.data = this.data.filter(
      (item) => item.method.indexOf(this.queryParam) != -1
    )
  }

  async search() {
    if (this.data.length != 0) {
      this.data = []
    }
    const res = await mockApi.list.request(null)
    const values: string[] = Object.values(res)
    for (const value of values) {
      this.data.push(JSON.parse(value))
    }
  }
}
</script>

<style>
.el-dialog__body {
  padding: 2px 2px;
}
</style>
