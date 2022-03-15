<template>
    <div>
        <div class="toolbar">
            <el-row type="flex" justify="space-between">
                <el-col :span="24">
                    <project-env-select @changeProjectEnv="changeProjectEnv" @clear="clearDb">
                        <template #default>
                            <el-form-item label="数据库">
                                <el-select v-model="dbId" placeholder="请选择数据库" @change="changeDb" @clear="clearDb" clearable filterable>
                                    <el-option v-for="item in dbs" :key="item.id" :label="item.database" :value="item.id">
                                        <span style="float: left">{{ item.database }}</span>
                                        <span style="float: right; color: #8492a6; margin-left: 6px; font-size: 13px">{{
                                            `${item.name}  [${item.type}]`
                                        }}</span>
                                    </el-option>
                                </el-select>
                            </el-form-item>

                            <el-form-item label-width="40" label="表">
                                <el-select v-model="tableName" placeholder="选择表查看表数据" @change="changeTable" filterable style="width: 250px">
                                    <el-option
                                        v-for="item in tableMetadata"
                                        :key="item.tableName"
                                        :label="item.tableName + (item.tableComment != '' ? `【${item.tableComment}】` : '')"
                                        :value="item.tableName"
                                    >
                                    </el-option>
                                </el-select>
                            </el-form-item>
                        </template>
                    </project-env-select>
                </el-col>
            </el-row>
        </div>

        <el-container id="data-exec" style="border: 1px solid #eee; margin-top: 1px">
            <el-tabs @tab-remove="removeDataTab" @tab-click="onDataTabClick" style="width: 100%" v-model="activeName">
                <el-tab-pane :label="queryTab.label" :name="queryTab.name">
                    <div>
                        <div>
                            <div class="toolbar">
                                <div class="fl">
                                    <el-upload
                                        style="display: inline-block"
                                        :before-upload="beforeUpload"
                                        :on-success="execSqlFileSuccess"
                                        :headers="{ Authorization: token }"
                                        :data="{
                                            dbId: 1,
                                        }"
                                        :action="getUploadSqlFileUrl()"
                                        :show-file-list="false"
                                        name="file"
                                        multiple
                                        :limit="100"
                                    >
                                        <el-button type="success" icon="video-play" plain size="small">sql脚本执行</el-button>
                                    </el-upload>
                                    <el-button @click="onCommit" class="ml5 mb5" type="success" icon="CircleCheck" plain size="small"
                                        >commit</el-button
                                    >
                                </div>

                                <div style="float: right" class="fl">
                                    <el-select
                                        v-model="sqlName"
                                        placeholder="选择or输入SQL模板名"
                                        @change="changeSqlTemplate"
                                        filterable
                                        allow-create
                                        default-first-option
                                        class="mr10"
                                    >
                                        <el-option v-for="item in sqlNames" :key="item" :label="item.database" :value="item">
                                            {{ item }}
                                        </el-option>
                                    </el-select>

                                    <el-button @click="saveSql" type="primary" icon="document-add" plain size="small">保存</el-button>
                                    <el-button @click="deleteSql" type="danger" icon="delete" plain size="small">删除</el-button>
                                </div>
                            </div>
                        </div>

                        <div class="mt5">
                            <codemirror
                                style="border: 1px solid #ccc"
                                @mousemove="listenMouse"
                                @beforeChange="onBeforeChange"
                                height="300px"
                                class="codesql"
                                ref="cmEditor"
                                language="sql"
                                v-model="sql"
                                :options="cmOptions"
                            />
                            <el-button-group :style="btnStyle">
                                <el-button @click="onRunSql" type="success" icon="video-play" size="small" plain>执行</el-button>
                                <el-button @click="formatSql" type="primary" icon="magic-stick" size="small" plain>格式化</el-button>
                            </el-button-group>
                        </div>

                        <div class="mt5">
                            <el-row v-if="queryTab.nowTableName">
                                <el-link @click="onDeleteData" class="ml5" type="danger" icon="delete" :underline="false"></el-link>
                            </el-row>
                            <el-table
                                @cell-dblclick="cellClick"
                                @selection-change="onDataSelectionChange"
                                :data="queryTab.execRes.data"
                                v-loading="queryTab.loading"
                                element-loading-text="查询中..."
                                size="small"
                                max-height="250"
                                empty-text="tips: select *开头的单表查询或点击表名默认查询的数据,可双击数据在线修改"
                                stripe
                                border
                                class="mt5"
                            >
                                <el-table-column
                                    v-if="queryTab.execRes.tableColumn.length > 0 && queryTab.nowTableName"
                                    type="selection"
                                    width="35"
                                />
                                <el-table-column
                                    min-width="100"
                                    :width="flexColumnWidth(item, queryTab.execRes.data)"
                                    align="center"
                                    v-for="item in queryTab.execRes.tableColumn"
                                    :key="item"
                                    :prop="item"
                                    :label="item"
                                    show-overflow-tooltip
                                >
                                </el-table-column>
                            </el-table>
                        </div>
                    </div>
                </el-tab-pane>

                <el-tab-pane closable v-for="dt in dataTabs" :key="dt.name" :label="dt.label" :name="dt.name">
                    <el-row v-if="dbId">
                        <el-link @click="onRefresh(dt.name)" icon="refresh" :underline="false" class="ml5"></el-link>
                        <el-link @click="addRow" class="ml5" type="primary" icon="plus" :underline="false"></el-link>
                        <el-link @click="onDeleteData" class="ml5" type="danger" icon="delete" :underline="false"></el-link>

                        <el-tooltip class="box-item" effect="dark" content="commit" placement="top">
                            <el-link @click="onCommit" class="ml5" type="success" icon="check" :underline="false"></el-link>
                        </el-tooltip>
                    </el-row>
                    <el-row class="mt5">
                        <el-input v-model="dt.condition" placeholder="若需条件过滤，输入WHERE之后查询条件点击查询按钮即可" clearable size="small">
                            <template #prepend>
                                <el-button @click="selectByCondition(dt.name, dt.condition)" icon="search" size="small"></el-button>
                            </template>
                        </el-input>
                    </el-row>
                    <el-table
                        @cell-dblclick="cellClick"
                        @sort-change="onTableSortChange"
                        @selection-change="onDataSelectionChange"
                        :data="dt.execRes.data"
                        size="small"
                        max-height="600"
                        v-loading="dt.loading"
                        element-loading-text="查询中..."
                        empty-text="暂无数据"
                        stripe
                        border
                        class="mt5"
                    >
                        <el-table-column v-if="dt.execRes.tableColumn.length > 0" type="selection" width="35" />
                        <el-table-column
                            min-width="100"
                            :width="flexColumnWidth(item, dt.execRes.data)"
                            align="center"
                            v-for="item in dt.execRes.tableColumn"
                            :key="item"
                            :prop="item"
                            :label="item"
                            show-overflow-tooltip
                            :sortable="nowTableName != '' ? 'custom' : false"
                        >
                            <template #header>
                                <el-tooltip raw-content effect="dark" placement="top">
                                    <template #content> {{ getColumnTip(dt.name, item) }} </template>
                                    <el-icon><question-filled /></el-icon>
                                </el-tooltip>
                                {{ item }}
                            </template>
                        </el-table-column>
                    </el-table>
                </el-tab-pane>
            </el-tabs>
        </el-container>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, computed, defineComponent, ref } from 'vue';
import { dbApi } from './api';
import _ from 'lodash';

import 'codemirror/addon/hint/show-hint.css';
// import base style
import 'codemirror/lib/codemirror.css';
// 引入主题后还需要在 options 中指定主题才会生效
import 'codemirror/theme/base16-light.css';

import 'codemirror/addon/selection/active-line';
import { codemirror } from '@/components/codemirror';
// import 'codemirror/mode/sql/sql.js';
import 'codemirror/addon/hint/show-hint.js';
import 'codemirror/addon/hint/sql-hint.js';

import { format as sqlFormatter } from 'sql-formatter';
import { notBlank, notEmpty, isTrue } from '@/common/assert';
import { ElMessage, ElMessageBox } from 'element-plus';
import ProjectEnvSelect from '../component/ProjectEnvSelect.vue';
import config from '@/common/config';
import { getSession } from '@/common/utils/storage';
import SqlExecBox from './component/SqlExecBox';

export default defineComponent({
    name: 'SqlExec',
    components: {
        codemirror,
        ProjectEnvSelect,
    },
    setup() {
        const cmEditor: any = ref(null);
        const token = getSession('token');

        const state = reactive({
            token: token,
            defalutLimit: 25, // 默认查询数量
            dbs: [],
            tables: [],
            dbId: null,
            tableName: '',
            tableMetadata: [],
            columnMetadata: [],
            sqlName: '', // 当前sql模板名
            sqlNames: [], // 所有sql模板名
            sql: '',
            activeName: 'Query',
            queryTabName: 'Query',
            nowTableName: '', // 当前表格数据操作的数据库表名，用于双击编辑表内容使用
            dataTabs: {}, // 点击表信息后执行结果数据展示tabs
            // 查询tab
            queryTab: {
                label: '查询',
                name: 'Query',
                // 点击执行按钮执行结果信息
                execRes: {
                    data: [],
                    tableColumn: [],
                },
                loading: false,
                nowTableName: '', //当前表格数据操作的数据库表名，用于双击编辑表内容使用
                selectionDatas: [],
            },
            params: {
                pageNum: 1,
                pageSize: 10,
                envId: null,
            },
            btnStyle: {
                position: 'absolute',
                zIndex: 1000,
                display: 'none',
                left: '',
                top: '',
            },
            cmOptions: {
                tabSize: 4,
                mode: 'text/x-sql',
                lineNumbers: true,
                line: true,
                indentWithTabs: true,
                smartIndent: true,
                matchBrackets: true,
                theme: 'base16-light',
                autofocus: true,
                extraKeys: { Tab: 'autocomplete' }, // 自定义快捷键
                hintOptions: {
                    completeSingle: false,
                    // 自定义提示选项
                    tables: {},
                },
                // more CodeMirror options...
            },
        });

        const tableMap = new Map();

        const codemirror: any = computed(() => {
            return cmEditor.value.coder;
        });

        /**
         * 项目及环境更改后的回调事件
         */
        const changeProjectEnv = (projectId: any, envId: any) => {
            state.dbs = [];
            state.dbId = null;
            clearDb();
            if (envId != null) {
                state.params.envId = envId;
                search();
            }
        };

        /**
         * 输入字符给提示
         */
        const inputRead = (instance: any, changeObj: any) => {
            if (/^[a-zA-Z]/.test(changeObj.text[0])) {
                showHint();
            }
        };

        const onBeforeChange = (instance: any, changeObj: any) => {
            var text = changeObj.text[0];
            // 将sql提示去除
            changeObj.text[0] = text.split('  ')[0];
        };

        /**
         * 执行sql
         */
        const onRunSql = async () => {
            notBlank(state.dbId, '请先选择数据库');
            // 没有选中的文本，则为全部文本
            let sql = getSql();
            notBlank(sql.trim(), 'sql内容不能为空');

            state.queryTab.loading = true;
            // 即只有以该字符串开头的sql才可修改表数据内容
            if (sql.startsWith('SELECT *') || sql.startsWith('select *') || sql.startsWith('SELECT\n  *')) {
                state.queryTab.selectionDatas = [];
                const tableName = sql.split(/from/i)[1];
                if (tableName) {
                    const tn = tableName.trim().split(' ')[0];
                    state.queryTab.nowTableName = tn;
                    state.nowTableName = tn;
                } else {
                    state.queryTab.nowTableName = '';
                    state.nowTableName = '';
                }
            } else {
                state.queryTab.nowTableName = '';
                state.nowTableName = '';
            }

            const colAndData: any = await runSql(sql);
            state.queryTab.execRes.data = colAndData.res;
            state.queryTab.execRes.tableColumn = colAndData.colNames;
            state.queryTab.loading = false;
        };

        /**
         * 执行sql str
         *
         * @param sql 执行的sql
         */
        const runSql = (sql: string) => {
            return dbApi.sqlExec.request({
                id: state.dbId,
                sql: sql.trim(),
            });
        };

        const removeDataTab = (targetName: string) => {
            const tabNames = Object.keys(state.dataTabs);
            let activeName = state.activeName;
            tabNames.forEach((name, index) => {
                if (name === targetName) {
                    const nextTab = tabNames[index + 1] || tabNames[index - 1] || state.queryTab.name;
                    if (nextTab) {
                        activeName = nextTab;
                    }
                }
            });
            state.activeName = activeName;
            delete state.dataTabs[targetName];
        };

        /**
         * 数据tab点击
         */
        const onDataTabClick = (tab: any) => {
            const name = tab.props.name;
            // 不是查询tab，则为表数据tab，同时赋值当前表名，用于在线修改表数据等
            if (name != state.queryTab.name) {
                state.nowTableName = name;
            } else {
                state.nowTableName = state.queryTab.nowTableName;
            }
        };

        const beforeUpload = (file: File) => {
            if (!state.dbId) {
                ElMessage.error('请先选择数据库');
                return false;
            }
            ElMessage.success(`'${file.name}' 正在上传执行, 请关注结果通知`);
        };

        // 执行sql成功
        const execSqlFileSuccess = (res: any) => {
            if (res.code !== 200) {
                ElMessage.error(res.msg);
            }
        };

        // 获取sql文件上传执行url
        const getUploadSqlFileUrl = () => {
            return `${config.baseApiUrl}/dbs/${state.dbId}/exec-sql-file`;
        };

        const flexColumnWidth = (str: any, tableData: any, flag = 'equal') => {
            // str为该列的字段名(传字符串);tableData为该表格的数据源(传变量);
            // flag为可选值，可不传该参数,传参时可选'max'或'equal',默认为'max'
            // flag为'max'则设置列宽适配该列中最长的内容,flag为'equal'则设置列宽适配该列中第一行内容的长度。
            str = str + '';
            let columnContent = '';
            if (!tableData || !tableData.length || tableData.length === 0 || tableData === undefined) {
                return;
            }
            if (!str || !str.length || str.length === 0 || str === undefined) {
                return;
            }
            if (flag === 'equal') {
                // 获取该列中第一个不为空的数据(内容)
                for (let i = 0; i < tableData.length; i++) {
                    if (tableData[i][str].length > 0) {
                        columnContent = tableData[i][str];
                        break;
                    }
                }
            } else {
                // 获取该列中最长的数据(内容)
                let index = 0;
                for (let i = 0; i < tableData.length; i++) {
                    if (tableData[i][str] === null) {
                        return;
                    }
                    const now_temp = tableData[i][str] + '';
                    const max_temp = tableData[index][str] + '';
                    if (now_temp.length > max_temp.length) {
                        index = i;
                    }
                }
                columnContent = tableData[index][str];
            }
            // 以下分配的单位长度可根据实际需求进行调整
            let flexWidth = 0;
            for (const char of columnContent) {
                if ((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z')) {
                    // 如果是英文字符，为字符分配8个单位宽度
                    flexWidth += 8;
                } else if (char >= '\u4e00' && char <= '\u9fa5') {
                    // 如果是中文字符，为字符分配15个单位宽度
                    flexWidth += 16;
                } else {
                    // 其他种类字符，为字符分配10个单位宽度
                    flexWidth += 10;
                }
            }
            if (flexWidth < 80) {
                // 设置最小宽度
                flexWidth = 80;
            }
            if (flexWidth > 500) {
                // 设置最大宽度
                flexWidth = 500;
            }
            return flexWidth + 'px';
        };

        const getColumnTip = (tableName: string, columnName: string) => {
            // 优先从 table map中获取
            let columns = tableMap.get(tableName);
            if (!columns) {
                return '';
            }
            const column = columns.find((c: any) => c.columnName == columnName);
            const comment = column.columnComment;
            return `${column.columnType} ${comment ? ' |  ' + comment : ''}`;
        };

        /**
         * 获取sql，如果有鼠标选中，则返回选中内容，否则返回输入框内所有内容
         */
        const getSql = () => {
            // 没有选中的文本，则为全部文本
            let selectSql = codemirror.value.getSelection();
            if (selectSql == '') {
                selectSql = state.sql;
            }
            return selectSql;
        };

        /**
         * 更改数据库事件
         */
        const changeDb = (id: number) => {
            if (!id) {
                return;
            }
            clearDb();
            dbApi.tableMetadata.request({ id }).then((res) => {
                state.tableMetadata = res;
            });

            dbApi.hintTables
                .request({
                    id: state.dbId,
                })
                .then((res) => {
                    state.cmOptions.hintOptions.tables = res;
                });

            getSqlNames();
        };

        // 选择表事件
        const changeTable = async (tableName: string, execSelectSql: boolean = true) => {
            if (tableName == '') {
                return;
            }
            state.columnMetadata = (await getColumns(tableName)) as any;

            if (!execSelectSql) {
                return;
            }

            // 执行sql，并新增tab
            state.nowTableName = tableName;
            state.activeName = tableName;
            let tab = state.dataTabs[tableName];
            // 如果存在该表tab，则直接返回
            if (tab) {
                return;
            }

            tab = {
                label: tableName,
                name: tableName,
                execRes: {
                    tableColumn: [],
                    data: [],
                },
                querySql: getDefaultSelectSql(tableName),
            };
            state.dataTabs[tableName] = tab;

            state.dataTabs[tableName].execRes.tableColumn = [];
            state.dataTabs[tableName].execRes.data = [];

            onRefresh(tableName);
        };

        /**
         * 获取默认查询语句
         */
        const getDefaultSelectSql = (tableName: string, where: string = '', orderBy: string = '') => {
            return `SELECT * FROM \`${tableName}\` ${where ? 'WHERE ' + where : ''} ${orderBy ? orderBy : ''} LIMIT ${state.defalutLimit}`;
        };

        const selectByCondition = async (tableName: string, condition: string) => {
            notEmpty(condition, '条件不能为空');
            state.dataTabs[tableName].loading = true;
            try {
                const colAndData: any = await runSql(getDefaultSelectSql(tableName, condition));
                state.dataTabs[tableName].execRes.tableColumn = colAndData.colNames;
                state.dataTabs[tableName].execRes.data = colAndData.res;
                state.dataTabs[tableName].loading = false;
            } catch (err) {
                state.dataTabs[tableName].loading = false;
            }
        };

        /**
         * 获取表的所有列信息
         */
        const getColumns = async (tableName: string) => {
            // 优先从 table map中获取
            let columns = tableMap.get(tableName);
            if (columns) {
                return columns;
            }
            columns = await dbApi.columnMetadata.request({
                id: state.dbId,
                tableName: tableName,
            });
            tableMap.set(tableName, columns);
            return columns;
        };

        const onRefresh = async (tableName: string) => {
            // 查询条件置空
            state.dataTabs[tableName].condition = '';
            state.dataTabs[tableName].loading = true;
            const colAndData: any = await runSql(state.dataTabs[tableName].querySql);
            state.dataTabs[tableName].execRes.tableColumn = colAndData.colNames;
            state.dataTabs[tableName].execRes.data = colAndData.res;
            state.dataTabs[tableName].loading = false;
        };

        /**
         * 提交事务，用于没有开启自动提交事务
         */
        const onCommit = () => {
            notBlank(state.dbId, '请先选择数据库');
            runSql('COMMIT;');
            ElMessage.success('COMMIT success');
        };

        /**
         * 表排序字段变更
         */
        const onTableSortChange = async (sort: any) => {
            if (!state.nowTableName || !sort.prop) {
                return;
            }
            const tableName = state.activeName;
            const sortType = sort.order == 'descending' ? 'DESC' : 'ASC';

            state.dataTabs[state.activeName].querySql = getDefaultSelectSql(tableName, '', `ORDER BY \`${sort.prop}\` ${sortType}`);

            onRefresh(tableName);
        };

        const changeSqlTemplate = () => {
            getUserSql();
        };

        /**
         * 获取用户保存的sql模板内容
         */
        const getUserSql = () => {
            notBlank(state.dbId, '请先选择数据库');
            dbApi.getSql.request({ id: state.dbId, type: 1, name: state.sqlName }).then((res) => {
                if (res) {
                    state.sql = res.sql;
                } else {
                    state.sql = '';
                }
            });
        };

        /**
         * 获取用户保存的sql模板名称
         */
        const getSqlNames = () => {
            dbApi.getSqlNames
                .request({
                    id: state.dbId,
                })
                .then((res) => {
                    if (res && res.length > 0) {
                        state.sqlNames = res.map((r: any) => r.name);
                        state.sqlName = state.sqlNames[0];
                    } else {
                        state.sqlNames = ['default'] as any;
                        state.sqlName = 'default';
                    }

                    getUserSql();
                });
        };

        const saveSql = async () => {
            notEmpty(state.sql, 'sql内容不能为空');
            notBlank(state.dbId, '请先选择数据库');
            await dbApi.saveSql.request({ id: state.dbId, sql: state.sql, type: 1, name: state.sqlName });
            ElMessage.success('保存成功');

            dbApi.getSqlNames
                .request({
                    id: state.dbId,
                })
                .then((res) => {
                    if (res) {
                        state.sqlNames = res.map((r: any) => r.name);
                    }
                });
        };

        const deleteSql = async () => {
            notBlank(state.dbId, '请先选择数据库');
            try {
                await ElMessageBox.confirm(`确定删除【${state.sqlName}】该SQL模板?`, '提示', {
                    confirmButtonText: '确定',
                    cancelButtonText: '取消',
                    type: 'warning',
                });
                await dbApi.deleteDbSql.request({ id: state.dbId, name: state.sqlName });
                ElMessage.success('删除成功');
                getSqlNames();
            } catch (err) {}
        };

        // 清空数据库事件
        const clearDb = () => {
            state.tableName = '';
            state.nowTableName = '';
            state.tableMetadata = [];
            state.columnMetadata = [];
            state.dataTabs = {};
            state.sql = '';
            state.sqlNames = [];
            state.sqlName = '';
            state.activeName = state.queryTab.name;
            state.queryTab.execRes.data = [];
            state.queryTab.execRes.tableColumn = [];
            state.cmOptions.hintOptions.tables = [];
            tableMap.clear();
        };

        const onDataSelectionChange = (datas: []) => {
            if (isQueryTab()) {
                state.queryTab.selectionDatas = datas;
            } else {
                state.dataTabs[state.activeName].selectionDatas = datas;
            }
        };

        /**
         * 执行删除数据事件
         */
        const onDeleteData = async () => {
            const queryTab = isQueryTab();
            const deleteDatas = queryTab ? state.queryTab.selectionDatas : state.dataTabs[state.activeName].selectionDatas;
            isTrue(deleteDatas && deleteDatas.length > 0, '请先选择要删除的数据');
            const primaryKey = await getColumn(state.nowTableName);
            const primaryKeyColumnName = primaryKey.columnName;
            const ids = deleteDatas.map((d: any) => `${wrapColumnValue(primaryKey, d[primaryKeyColumnName])}`).join(',');
            const sql = `DELETE FROM ${state.nowTableName} WHERE ${primaryKeyColumnName} IN (${ids})`;

            promptExeSql(sql, null, () => {
                if (!queryTab) {
                    state.dataTabs[state.activeName].execRes.data = state.dataTabs[state.activeName].execRes.data.filter(
                        (d: any) => !(deleteDatas.findIndex((x: any) => x[primaryKeyColumnName] == d[primaryKeyColumnName]) != -1)
                    );
                    state.dataTabs[state.activeName].selectionDatas = [];
                } else {
                    state.queryTab.execRes.data = state.queryTab.execRes.data.filter(
                        (d: any) => !(deleteDatas.findIndex((x: any) => x[primaryKeyColumnName] == d[primaryKeyColumnName]) != -1)
                    );
                    state.queryTab.selectionDatas = [];
                }
            });
        };

        /**
         * 是否为查询tab
         */
        const isQueryTab = () => {
            return state.activeName == state.queryTab.name;
        };

        // 监听单元格点击事件
        const cellClick = (row: any, column: any, cell: any) => {
            const property = column.property;
            // 如果当前操作的表名不存在 或者 当前列的property不存在(如多选框)，则不允许修改当前单元格内容
            if (!state.nowTableName || !property) {
                return;
            }
            let text = row[property];
            let div = cell.children[0];
            if (div) {
                let input = document.createElement('input');
                input.setAttribute('value', text);
                // 将表格width也赋值于输入框，避免输入框长度超过表格长度
                input.setAttribute('style', 'height:30px;' + div.getAttribute('style'));
                cell.replaceChildren(input);
                input.focus();
                input.addEventListener('blur', async () => {
                    row[property] = input.value;
                    cell.replaceChildren(div);
                    if (input.value !== text) {
                        const primaryKey = await getColumn(state.nowTableName);
                        const primaryKeyColumnName = primaryKey.columnName;
                        // 更新字段列信息
                        const updateColumn = await getColumn(state.nowTableName, column.rawColumnKey);
                        const sql = `UPDATE ${state.nowTableName} SET ${column.rawColumnKey} = ${wrapColumnValue(updateColumn, input.value)} 
                                        WHERE ${primaryKeyColumnName} = ${wrapColumnValue(primaryKey, row[primaryKeyColumnName])}`;
                        promptExeSql(sql, () => {
                            row[property] = text;
                        });
                    }
                });
            }
        };

        /**
         * 根据字段列名获取字段列信息。
         * 若字段列名为空，则返回第一个字段列信息（用于获取主键等，目前先以默认表字段第一个字段）
         */
        const getColumn = async (tableName: string, columnName: string = '') => {
            const cols = await getColumns(tableName);
            if (!columnName) {
                return cols[0];
            }
            return cols.find((c: any) => c.columnName == columnName);
        };

        /**
         * 根据字段信息包装字段值，如为字符串等则添加‘’
         */
        const wrapColumnValue = (column: any, value: any) => {
            if (isNumber(column.columnType)) {
                return value;
            }
            return `'${value}'`;
        };

        /**
         * 判断字段类型是否为数字类型
         */
        const isNumber = (columnType: string) => {
            return columnType.match(/int|double|float|nubmer|decimal/gi);
        };

        /**
         * 弹框提示是否执行sql
         */
        const promptExeSql = (sql: string, cancelFunc: any = null, successFunc: any = null) => {
            SqlExecBox({
                sql: sql,
                dbId: state.dbId as any,
                runSuccessCallback: successFunc,
                cancelCallback: cancelFunc,
            });
        };

        // 添加新数据行
        const addRow = async () => {
            const tableName = state.nowTableName;
            const columns = await getColumns(tableName);

            // key: 字段名，value: 字段名提示
            let obj: any = {};
            columns.forEach((item: any) => {
                obj[`${item.columnName}`] = `'${item.columnName}[${item.columnType}]${item.nullable == 'YES' ? '' : '[not null]'}'`;
            });
            let columnNames = Object.keys(obj).join(',');
            let values = Object.values(obj).join(',');
            let sql = `INSERT INTO ${state.nowTableName} (${columnNames}) VALUES (${values});`;
            promptExeSql(sql, null, () => {
                onRefresh(tableName);
            });
        };

        /**
         * 自动提示功能
         */
        const showHint = () => {
            codemirror.value.showHint();
        };

        /**
         * 格式化sql
         */
        const formatSql = () => {
            let selectSql = codemirror.value.getSelection();
            // 有选中sql则只格式化选中部分，否则格式化全部
            if (selectSql != '') {
                codemirror.value.replaceSelection(sqlFormatter(selectSql));
            } else {
                /* 将sql内容进行格式后放入编辑器中*/
                state.sql = sqlFormatter(state.sql);
            }
        };

        const search = async () => {
            const res = await dbApi.dbs.request(state.params);
            state.dbs = res.list;
        };

        /**
         * 获取选择文字，显示隐藏按钮，防抖
         */
        const getSelection = _.debounce((e: any) => {
            let temp = codemirror.value.getSelection();
            if (temp) {
                state.btnStyle.display = 'block';
                if (!state.btnStyle.left) {
                    state.btnStyle.left = e.target.getBoundingClientRect().left;
                    state.btnStyle.top = e.target.getBoundingClientRect().top - 160 + 'px';
                }
            } else {
                state.btnStyle.display = 'none';
                state.btnStyle.left = '';
                state.btnStyle.top = '';
            }
        }, 100);

        const listenMouse = (e: any) => {
            getSelection(e);
        };

        return {
            ...toRefs(state),
            cmEditor,
            changeProjectEnv,
            inputRead,
            changeTable,
            cellClick,
            onRunSql,
            removeDataTab,
            onDataTabClick,
            beforeUpload,
            getUploadSqlFileUrl,
            execSqlFileSuccess,
            flexColumnWidth,
            getColumnTip,
            changeSqlTemplate,
            deleteSql,
            saveSql,
            changeDb,
            clearDb,
            formatSql,
            onBeforeChange,
            listenMouse,
            onRefresh,
            selectByCondition,
            onCommit,
            addRow,
            onDataSelectionChange,
            onDeleteData,
            onTableSortChange,
        };
    },
});
</script>

<style lang="scss">
.codesql {
    font-size: 9pt;
    font-weight: 600;
}
.el-tabs__header {
    padding: 0 10px;
    background-color: #fff;
}

#data-exec {
    min-height: calc(100vh - 155px);

    .el-table__empty-text {
        width: 100%;
        margin-left: 50px;
    }
}
</style>
