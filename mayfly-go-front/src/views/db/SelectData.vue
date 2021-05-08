<template>
  <div>
    <div class="toolbar">
      <div class="fl">
        <el-select
          size="small"
          v-model="dbId"
          placeholder="请选择数据库"
          @change="changeDb"
          @clear="clearDb"
          clearable
          filterable
        >
          <el-option
            v-for="item in dbs.list"
            :key="item.id"
            :label="`${item.name} [${item.type}]`"
            :value="item.id"
          >
          </el-option>
        </el-select>
      </div>
    </div>

    <el-container style="height: 50%; border: 1px solid #eee; margin-top: 1px">
      <el-aside width="70%" style="background-color: rgb(238, 241, 246)">
        <div class="toolbar">
          <div class="fl">
            <el-button
              @click="runSql"
              type="success"
              icon="el-icon-video-play"
              size="mini"
              plain
              >执行</el-button
            >

            <el-button
              @click="formatSql"
              type="primary"
              icon="el-icon-magic-stick"
              size="mini"
              plain
              >格式化</el-button
            >

            <el-button
              @click="saveSql"
              type="primary"
              icon="el-icon-document-add"
              size="mini"
              plain
              >保存</el-button
            >
          </div>
        </div>
        <codemirror
          class="codesql"
          ref="cmEditor"
          v-model="sql"
          :placeholder="placeholder"
          :options="cmOptions"
          @inputRead="inputRead"
        />
      </el-aside>

      <el-container style="margin-left: 2px">
        <el-header
          style="text-align: left; height: 45px; font-size: 12px; padding: 0px"
        >
          <el-select
            v-model="tableName"
            placeholder="请选择表"
            @change="changeTable"
            clearable
            filterable
            style="width: 99%"
          >
            <el-option
              v-for="item in tableMetadata"
              :key="item.tableName"
              :label="
                item.tableName +
                (item.tableComment != '' ? `【${item.tableComment}】` : '')
              "
              :value="item.tableName"
            >
            </el-option>
          </el-select>
        </el-header>

        <el-main style="padding: 0px; height: 100%; overflow: hidden">
          <el-table :data="columnMetadata" height="100%" size="mini">
            <el-table-column prop="columnName" label="名称"> </el-table-column>
            <el-table-column prop="columnType" label="类型"> </el-table-column>
            <el-table-column prop="columnComment" label="备注">
            </el-table-column>
          </el-table>
        </el-main>
      </el-container>
    </el-container>
    <el-table
      style="margin-top: 1px"
      :data="selectRes.data"
      size="mini"
      max-height="300"
      stripe
      border
    >
      <el-table-column
        min-width="100"
        align="center"
        v-for="item in selectRes.tableColumn"
        :key="item"
        :prop="item"
        :label="item"
        show-overflow-tooltip
      >
      </el-table-column>
    </el-table>

    <!-- <el-pagination
      style="text-align: center"
      background
      layout="prev, pager, next, total, jumper"
      :total="data.total"
      :current-page.sync="params.pageNum"
      :page-size="params.pageSize"
    /> -->
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator'
import { dbApi } from './api'

import 'codemirror/theme/ambiance.css'
import 'codemirror/addon/hint/show-hint.css'
// import base style
import 'codemirror/lib/codemirror.css'
// 引入主题后还需要在 options 中指定主题才会生效
import 'codemirror/theme/base16-light.css'

// require('codemirror/addon/edit/matchbrackets')
require('codemirror/addon/selection/active-line')
import { codemirror } from 'vue-codemirror'
import 'codemirror/mode/sql/sql.js'
import 'codemirror/addon/hint/show-hint.js'
import 'codemirror/addon/hint/sql-hint.js'

import sqlFormatter from 'sql-formatter'
import { notEmpty } from '@/common/assert'
import { Message } from 'element-ui'


@Component({
  name: 'SelectData',
  components: {
    codemirror,
  },
})
export default class SelectData extends Vue {
  dbs = []
  tables = []
  dbId = ''
  tableName = ''
  tableMetadata = []
  columnMetadata = []
  sql = ''
  selectRes = {
    tableColumn: [],
    data: [],
  }
  params = {
    pageNum: 1,
    pageSize: 10,
  }
  placeholder = 'sqlshuru'
  cmOptions = {
    tabSize: 4,
    mode: 'text/x-sql',
    // theme: 'cobalt',
    lineNumbers: true,
    line: true,
    indentWithTabs: true,
    smartIndent: true,
    // matchBrackets: true,
    theme: 'base16-light',
    autofocus: true,
    extraKeys: { Tab: 'autocomplete' }, // 自定义快捷键
    hintOptions: {
      completeSingle: false,
      // 自定义提示选项
      tables: {},
    },
    // more CodeMirror options...
  }

  get codemirror() {
    return this.$refs.cmEditor['codemirror']
  }

  mounted() {
    this.search()
  }

  // 输入字符给提示
  inputRead(instance: any, changeObj: any) {
    if (/^[a-zA-Z]/.test(changeObj.text[0])) {
      this.showHint()
    }
  }

  // 执行sql
  async runSql() {
    notEmpty(this.dbId, '请先选择数据库')
    // 没有选中的文本，则为全部文本
    let selectSql = this.codemirror.getSelection()
    if (selectSql == '') {
      selectSql = this.sql
    }
    notEmpty(this.sql, '内容不能为空')
    const res = await dbApi.selectData.request({
      id: this.dbId,
      selectSql,
    })
    let tableColumn: any
    let data
    if (res.length > 0) {
      tableColumn = Object.keys(res[0])
      data = res
    } else {
      tableColumn = []
      data = []
    }
    this.selectRes.tableColumn = tableColumn
    this.selectRes.data = data
  }

  async saveSql() {
    notEmpty(this.sql, 'sql内容不能为空')
    notEmpty(this.dbId, '请先选择数据库')
    await dbApi.saveSql.request({ id: this.dbId, sql: this.sql, type: 1 })
    Message.success('保存成功')
  }

  // 更改数据库事件
  async changeDb(id: number) {
    if (!id) {
      return
    }
    this.clearDb()
    this.tableMetadata = await dbApi.tableMetadata.request({ id })
    // 赋值第一个表信息
    if (this.tableMetadata.length > 0) {
      this.tableName = this.tableMetadata[0]['tableName']
      this.changeTable(this.tableName)
    }
    const dbSql = await dbApi.getSql.request({ id, type: 1 })
    if (dbSql) {
      this.sql = dbSql.sql
    }
    this.cmOptions.hintOptions.tables = await dbApi.hintTables.request({
      id: this.dbId,
    })
  }

  // 清空数据库事件
  clearDb() {
    this.tableName = ''
    this.tableMetadata = []
    this.columnMetadata = []
    this.selectRes.data = []
    this.selectRes.tableColumn = []
    this.sql = ''
  }

  // 选择表事件
  async changeTable(tableName: string) {
    if (tableName == '') {
      return
    }
    this.columnMetadata = await dbApi.columnMetadata.request({
      id: this.dbId,
      tableName,
    })
  }

  // 自动提示功能
  showHint() {
    this.codemirror.showHint()
  }

  formatSql() {
    /* 获取文本编辑器内容*/
    let sqlContent = ''
    sqlContent = this.codemirror.getValue()
    /* 将sql内容进行格式后放入编辑器中*/
    this.codemirror.setValue(sqlFormatter.format(sqlContent))
  }

  async search() {
    this.dbs = await dbApi.dbs.request(this.params)
  }
}
</script>

<style>
.codesql {
  font-size: 10pt;
  font-family: Consolas, Menlo, Monaco, Lucida Console, Liberation Mono,
    DejaVu Sans Mono, Bitstream Vera Sans Mono, Courier New, monospace, serif;
}
</style>
