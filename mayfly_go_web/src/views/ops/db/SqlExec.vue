<template>
    <div>
        <div class="toolbar">
            <el-row type="flex" justify="space-between">
                <el-col :span="24">
                    <project-env-select @changeProjectEnv="changeProjectEnv">
                        <template #default>
                            <el-form-item label="资源">
                                <el-select v-model="dbId" placeholder="请选择资源实例" @change="changeDbInstance" filterable style="width: 150px">
                                    <el-option v-for="item in dbs" :key="item.id" :label="item.name" :value="item.id">
                                        <span style="float: left">{{ item.name }}</span>
                                        <span style="float: right; color: #8492a6; margin-left: 6px; font-size: 13px">{{
                                            `${item.host}:${item.port} ${item.type}`
                                        }}</span>
                                    </el-option>
                                </el-select>
                            </el-form-item>

                            <el-form-item label="数据库">
                                <el-select
                                    v-model="db"
                                    placeholder="请选择数据库"
                                    @change="changeDb"
                                    @clear="clearDb"
                                    clearable
                                    filterable
                                    style="width: 150px"
                                >
                                    <el-option v-for="item in databaseList" :key="item" :label="item" :value="item"> </el-option>
                                </el-select>
                            </el-form-item>

                            <el-form-item label-width="20" label="表">
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
                                    <el-link @click="onRunSql" :underline="false" class="ml15" icon="VideoPlay"></el-link>
                                    <el-divider direction="vertical" border-style="dashed" />

                                    <el-tooltip class="box-item" effect="dark" content="format sql" placement="top">
                                        <el-link @click="formatSql" type="primary" :underline="false" icon="MagicStick"></el-link>
                                    </el-tooltip>
                                    <el-divider direction="vertical" border-style="dashed" />

                                    <el-tooltip class="box-item" effect="dark" content="commit" placement="top">
                                        <el-link @click="onCommit" type="success" :underline="false" icon="CircleCheck"></el-link>
                                    </el-tooltip>
                                    <el-divider direction="vertical" border-style="dashed" />

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
                                        <el-tooltip class="box-item" effect="dark" content="SQL脚本执行" placement="top">
                                            <el-link type="success" :underline="false" icon="Document"></el-link>
                                        </el-tooltip>
                                    </el-upload>
                                </div>

                                <div style="float: right" class="fl">
                                    <el-select
                                        v-model="sqlName"
                                        placeholder="选择or输入SQL模板名"
                                        @change="changeSqlTemplate"
                                        filterable
                                        allow-create
                                        default-first-option
                                        size="small"
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

                        <div class="mt5 sqlEditor">
                            <textarea ref="codeTextarea"></textarea>
                        </div>

                        <div class="mt5">
                            <el-row>
                                <el-link
                                    v-if="queryTab.nowTableName"
                                    @click="onDeleteData"
                                    class="ml5"
                                    type="danger"
                                    icon="delete"
                                    :underline="false"
                                ></el-link>

                                <span v-if="queryTab.execRes.data.length > 0">
                                    <el-divider direction="vertical" border-style="dashed" />
                                    <el-link type="success" :underline="false" @click="exportData"><span style="font-size: 12px">导出</span></el-link>
                                </span>
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
                        <el-col :span="8">
                            <el-link @click="onRefresh(dt.name)" icon="refresh" :underline="false" class="ml5"></el-link>
                            <el-divider direction="vertical" border-style="dashed" />

                            <el-link @click="addRow" type="primary" icon="plus" :underline="false"></el-link>
                            <el-divider direction="vertical" border-style="dashed" />

                            <el-link @click="onDeleteData" type="danger" icon="delete" :underline="false"></el-link>
                            <el-divider direction="vertical" border-style="dashed" />

                            <el-tooltip class="box-item" effect="dark" content="commit" placement="top">
                                <el-link @click="onCommit" type="success" icon="CircleCheck" :underline="false"></el-link>
                            </el-tooltip>
                            <el-divider direction="vertical" border-style="dashed" />

                            <el-tooltip class="box-item" effect="dark" content="生成insert sql" placement="top">
                                <el-link @click="onGenerateInsertSql" type="success" :underline="false">gi</el-link>
                            </el-tooltip>
                        </el-col>
                        <el-col :span="16">
                            <el-input
                                v-model="dt.condition"
                                placeholder="若需条件过滤，可选择列并点击对应的字段并输入需要过滤的内容点击查询按钮即可"
                                clearable
                                size="small"
                                style="width: 100%"
                            >
                                <template #prepend>
                                    <el-popover trigger="click" :width="320" placement="right">
                                        <template #reference>
                                            <el-link type="success" :underline="false">选择列</el-link>
                                        </template>
                                        <el-table
                                            :data="getColumns4Map(dt.name)"
                                            max-height="500"
                                            size="small"
                                            @row-click="
                                                (...event) => {
                                                    onConditionRowClick(event, dt);
                                                }
                                            "
                                            style="cursor: pointer"
                                        >
                                            <el-table-column property="columnName" label="列名" show-overflow-tooltip> </el-table-column>
                                            <el-table-column property="columnComment" label="备注" show-overflow-tooltip> </el-table-column>
                                        </el-table>
                                    </el-popover>
                                </template>

                                <template #append>
                                    <el-button @click="selectByCondition(dt.name, dt.condition)" icon="search" size="small"></el-button>
                                </template>
                            </el-input>
                        </el-col>
                    </el-row>
                    <el-table
                        @cell-dblclick="cellClick"
                        @sort-change="onTableSortChange"
                        @selection-change="onDataSelectionChange"
                        :data="dt.datas"
                        size="small"
                        :max-height="dataTabsTableHeight"
                        v-loading="dt.loading"
                        element-loading-text="查询中..."
                        empty-text="暂无数据"
                        stripe
                        border
                        class="mt5"
                    >
                        <el-table-column v-if="dt.datas.length > 0" type="selection" width="35" />
                        <el-table-column
                            min-width="100"
                            :width="flexColumnWidth(item, dt.datas)"
                            align="center"
                            v-for="item in dt.columnNames"
                            :key="item"
                            :prop="item"
                            :label="item"
                            show-overflow-tooltip
                            :sortable="nowTableName != '' ? 'custom' : false"
                        >
                            <template #header>
                                <el-tooltip raw-content placement="top" effect="customized">
                                    <template #content> {{ getColumnTip(dt.name, item) }} </template>
                                    {{ item }}
                                </el-tooltip>
                            </template>
                        </el-table-column>
                    </el-table>
                    <el-row type="flex" class="mt5" justify="center">
                        <el-pagination
                            small
                            :total="dt.count"
                            @current-change="handlePageChange(dt)"
                            layout="prev, pager, next, total, jumper"
                            v-model:current-page="dt.pageNum"
                            :page-size="defalutLimit"
                        ></el-pagination>
                    </el-row>
                </el-tab-pane>
            </el-tabs>
        </el-container>

        <el-dialog v-model="conditionDialog.visible" :title="conditionDialog.title" width="420px">
            <el-row>
                <el-col :span="5">
                    <el-select v-model="conditionDialog.condition">
                        <el-option label="=" value="="> </el-option>
                        <el-option label="LIKE" value="LIKE"> </el-option>
                        <el-option label=">" value=">"> </el-option>
                        <el-option label=">=" value=">="> </el-option>
                        <el-option label="<" value="<"> </el-option>
                        <el-option label="<=" value="<="> </el-option>
                    </el-select>
                </el-col>
                <el-col :span="19">
                    <el-input v-model="conditionDialog.value" :placeholder="conditionDialog.placeholder" />
                </el-col>
            </el-row>
            <template #footer>
                <span class="dialog-footer">
                    <el-button @click="onCancelCondition">取消</el-button>
                    <el-button type="primary" @click="onConfirmCondition">确定</el-button>
                </span>
            </template>
        </el-dialog>

        <el-dialog @close="genSqlDialog.visible = false" v-model="genSqlDialog.visible" title="SQL" width="1000px">
            <el-input v-model="genSqlDialog.sql" type="textarea" rows="20" />
        </el-dialog>
    </div>
</template>

<script lang="ts">
import { onMounted, toRefs, reactive, defineComponent, ref } from 'vue';
import { dbApi } from './api';

import 'codemirror/addon/hint/show-hint.css';
// import base style
import 'codemirror/lib/codemirror.css';
// 引入主题后还需要在 options 中指定主题才会生效
import 'codemirror/theme/base16-light.css';

import 'codemirror/addon/selection/active-line';
import _CodeMirror from 'codemirror';
import 'codemirror/addon/hint/show-hint.js';
import 'codemirror/addon/hint/sql-hint.js';

import { format as sqlFormatter } from 'sql-formatter';
import { notBlank, notEmpty, isTrue } from '@/common/assert';
import { ElMessage, ElMessageBox } from 'element-plus';
import ProjectEnvSelect from '../component/ProjectEnvSelect.vue';
import config from '@/common/config';
import { getSession } from '@/common/utils/storage';
import SqlExecBox from './component/SqlExecBox';
import { dateStrFormat } from '@/common/utils/date.ts';

export default defineComponent({
    name: 'SqlExec',
    components: {
        ProjectEnvSelect,
    },
    setup() {
        const codeTextarea: any = ref(null);
        const token = getSession('token');
        let codemirror = null as any;
        const tableMap = new Map();

        const state = reactive({
            token: token,
            defalutLimit: 20, // 默认查询数量
            dbs: [], // 数据库实例列表
            databaseList: [], // 数据库实例拥有的数据库列表，1数据库实例  -> 多数据库
            db: '', // 当前操作的数据库
            dbType: '',
            tables: [],
            dbId: null, // 当前选中操作的数据库实例
            tableName: '',
            tableMetadata: [],
            sqlName: '', // 当前sql模板名
            sqlNames: [], // 所有sql模板名
            activeName: 'Query',
            queryTabName: 'Query',
            nowTableName: '', // 当前表格数据操作的数据库表名，用于双击编辑表内容使用
            dataTabs: {}, // 点击表信息后执行结果数据展示tabs
            dataTabsTableHeight: 600,
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
            conditionDialog: {
                title: '',
                placeholder: '',
                columnRow: null,
                dataTab: null,
                visible: false,
                condition: '=',
                value: null,
            },
            genSqlDialog: {
                visible: false,
                sql: '',
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

        const initCodemirror = () => {
            // 初始化编辑器实例，传入需要被实例化的文本域对象和默认配置
            codemirror = _CodeMirror.fromTextArea(codeTextarea.value, state.cmOptions);
            codemirror.on('inputRead', (instance: any, changeObj: any) => {
                if (/^[a-zA-Z]/.test(changeObj.text[0])) {
                    instance.showHint();
                }
            });

            codemirror.on('beforeChange', (instance: any, changeObj: any) => {
                var text = changeObj.text[0];
                // 将sql提示去除
                changeObj.text[0] = text.split('  ')[0];
            });
        };

        onMounted(() => {
            initCodemirror();
            setHeight();
            // 监听浏览器窗口大小变化,更新对应组件高度
            window.onresize = () =>
                (() => {
                    setHeight();
                })();
        });

        /**
         * 设置codemirror高度和数据表高度
         */
        const setHeight = () => {
            // 默认300px
            codemirror.setSize('auto', `${window.innerHeight - 538}px`);
            state.dataTabsTableHeight = window.innerHeight - 274;
        };

        /**
         * 项目及环境更改后的回调事件
         */
        const changeProjectEnv = (projectId: any, envId: any) => {
            state.dbs = [];
            state.dbId = null;
            state.db = '';
            state.databaseList = [];
            clearDb();
            if (envId != null) {
                state.params.envId = envId;
                search();
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
            isTrue(sql && sql.trim(), '请选中需要执行的sql');

            // 去除字符串前的空格、换行等
            sql = sql.replace(/(^\s*)/g, '');
            let execRemark = '';
            let canRun = true;
            if (
                sql.startsWith('update') ||
                sql.startsWith('UPDATE') ||
                sql.startsWith('INSERT') ||
                sql.startsWith('insert') ||
                sql.startsWith('DELETE') ||
                sql.startsWith('delete')
            ) {
                const res: any = await ElMessageBox.prompt('请输入备注', 'Tip', {
                    confirmButtonText: '确定',
                    cancelButtonText: '取消',
                    inputPattern: /^[\s\S]*.*[^\s][\s\S]*$/,
                    inputErrorMessage: '请输入执行该sql的备注信息',
                });
                execRemark = res.value;
                if (!execRemark) {
                    canRun = false;
                }
            }
            if (!canRun) {
                return;
            }

            try {
                state.queryTab.loading = true;
                const colAndData: any = await runSql(sql, execRemark);
                state.queryTab.execRes.data = colAndData.res;
                state.queryTab.execRes.tableColumn = colAndData.colNames;
                state.queryTab.loading = false;
            } catch (e: any) {
                state.queryTab.loading = false;
            }

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
        };

        const exportData = () => {
            const dataList = state.queryTab.execRes.data;
            isTrue(dataList.length > 0, '没有数据可导出');

            const tableColumn = state.queryTab.execRes.tableColumn;
            // 二维数组
            const cvsData = [tableColumn];
            for (let data of dataList) {
                // 数据值组成的一维数组
                let dataValueArr: any = [];
                for (let column of tableColumn) {
                    dataValueArr.push(data[column]);
                }
                cvsData.push(dataValueArr);
            }
            const csvString = cvsData.map((e) => e.join(',')).join('\n');

            // 导出
            let link = document.createElement('a');
            let exportContent = '\uFEFF';
            let blob = new Blob([exportContent + csvString], {
                type: 'text/plain;charset=utrf-8',
            });
            link.id = 'download-csv';
            link.setAttribute('href', URL.createObjectURL(blob));
            link.setAttribute('download', `查询数据导出-${dateStrFormat('yyyyMMddHHmmss', new Date())}.csv`);
            document.body.appendChild(link);
            link.click();
        };

        /**
         * 执行sql str
         *
         * @param sql 执行的sql
         */
        const runSql = async (sql: string, remark: string = '') => {
            return await dbApi.sqlExec.request({
                id: state.dbId,
                db: state.db,
                sql: sql.trim(),
                remark,
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
                // 修改选择框绑定的表信息
                state.tableName = name;
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
            return `${config.baseApiUrl}/dbs/${state.dbId}/exec-sql-file?db=${state.db}`;
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
                    // 转为字符串后比较
                    if ((tableData[i][str] + '').length > 0) {
                        columnContent = tableData[i][str] + '';
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
                columnContent = tableData[index][str] + '';
            }
            const contentWidth: number = getContentWidth(columnContent);
            // 获取列名称的长度 加上排序图标长度
            const columnWidth: number = getContentWidth(str) + 43;
            const flexWidth: number = contentWidth > columnWidth ? contentWidth : columnWidth;
            return flexWidth + 'px';
        };

        /**
         * 获取内容所需要占用的宽度
         */
        const getContentWidth = (content: any): number => {
            // 以下分配的单位长度可根据实际需求进行调整
            let flexWidth = 0;
            for (const char of content) {
                if (flexWidth > 500) {
                    break;
                }
                if ((char >= '0' && char <= '9') || (char >= 'a' && char <= 'z')) {
                    // 如果是小写字母、数字字符，分配8个单位宽度
                    flexWidth += 8.5;
                    continue;
                }
                if (char >= 'A' && char <= 'Z') {
                    flexWidth += 9;
                    continue;
                }
                if (char >= '\u4e00' && char <= '\u9fa5') {
                    // 如果是中文字符，为字符分配16个单位宽度
                    flexWidth += 16;
                } else {
                    // 其他种类字符，为字符分配9个单位宽度
                    flexWidth += 8;
                }
            }
            if (flexWidth > 500) {
                // 设置最大宽度
                flexWidth = 500;
            }
            return flexWidth;
        };

        const getColumnTip = (tableName: string, columnName: string) => {
            // 优先从 table map中获取
            let columns = getColumns4Map(tableName);
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
            let selectSql = codemirror.getSelection();
            if (!selectSql) {
                selectSql = getCodermirrorValue();
            }
            return selectSql;
        };

        /**
         * 选择数据库实例事件
         */
        const changeDbInstance = (dbId: any) => {
            state.db = '';
            const dbInfo = state.dbs.find((e: any) => e.id == dbId) as any;
            state.dbType = dbInfo.type;
            state.databaseList = dbInfo.database.split(' ');
            clearDb();
        };

        /**
         * 更改数据库事件
         */
        const changeDb = (db: string) => {
            if (!db) {
                return;
            }
            clearDb();
            dbApi.tableMetadata.request({ id: state.dbId, db }).then((res) => {
                state.tableMetadata = res;
            });

            dbApi.hintTables
                .request({
                    id: state.dbId,
                    db,
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
                datas: [],
                columnNames: [],
                pageNum: 1,
                count: 0,
            };
            tab.columnNames = await getColumnNames(tableName);
            state.dataTabs[tableName] = tab;

            onRefresh(tableName);
        };

        /**
         * 获取表的所有列信息
         */
        const getColumns = async (tableName: string) => {
            // 优先从 table map中获取
            let columns = getColumns4Map(tableName);
            if (columns) {
                return columns;
            }
            columns = await dbApi.columnMetadata.request({
                id: state.dbId,
                db: state.db,
                tableName: tableName,
            });
            tableMap.set(tableName, columns);
            return columns;
        };

        // 从缓存map获取列信息
        const getColumns4Map = (tableName: string) => {
            return tableMap.get(tableName);
        };

        const getColumnNames = async (tableName: string) => {
            const columns = await getColumns(tableName);
            return columns.map((t: any) => t.columnName);
        };

        /**
         * 条件查询，点击列信息后显示输入对应的值
         */
        const onConditionRowClick = (event: any, dataTab: any) => {
            const row = event[0];
            state.conditionDialog.title = `请输入 [${row.columnName}] 的值`;
            state.conditionDialog.placeholder = `${row.columnType}  ${row.columnComment}`;
            state.conditionDialog.columnRow = row;
            state.conditionDialog.dataTab = dataTab;
            state.conditionDialog.visible = true;
        };

        // 确认条件
        const onConfirmCondition = () => {
            const conditionDialog = state.conditionDialog;
            const dataTab = state.conditionDialog.dataTab as any;
            let condition = dataTab.condition;
            if (condition) {
                condition += ` AND `;
            }
            const row = conditionDialog.columnRow as any;
            condition += `${row.columnName} ${conditionDialog.condition} `;
            dataTab.condition = condition + wrapColumnValue(row, conditionDialog.value);
            onCancelCondition();
        };

        const onCancelCondition = () => {
            state.conditionDialog.visible = false;
            state.conditionDialog.title = ``;
            state.conditionDialog.placeholder = ``;
            state.conditionDialog.value = null;
            state.conditionDialog.columnRow = null;
            state.conditionDialog.dataTab = null;
        };

        const onRefresh = async (tableName: string) => {
            const dataTab = state.dataTabs[tableName];
            // 查询条件置空
            dataTab.condition = '';
            dataTab.pageNum = 1;
            setDataTabDatas(dataTab);
        };

        /**
         * 数据tab修改页数
         */
        const handlePageChange = async (dataTab: any) => {
            setDataTabDatas(dataTab);
        };

        /**
         * 根据条件查询数据
         */
        const selectByCondition = async (tableName: string, condition: string) => {
            notEmpty(condition, '条件不能为空');
            const dataTab = state.dataTabs[tableName];
            dataTab.pageNum = 1;
            setDataTabDatas(dataTab);
        };

        /**
         * 设置data tab的表数据
         */
        const setDataTabDatas = async (dataTab: any) => {
            dataTab.loading = true;
            try {
                dataTab.count = await getTableCount(dataTab.name, dataTab.condition);
                if (dataTab.count > 0) {
                    const colAndData: any = await runSql(getDefaultSelectSql(dataTab.name, dataTab.condition, dataTab.orderBy, dataTab.pageNum));
                    dataTab.datas = colAndData.res;
                } else {
                    dataTab.datas = [];
                }
            } finally {
                dataTab.loading = false;
            }
        };

        /**
         * 获取表的统计数量
         */
        const getTableCount = async (tableName: string, condition: string = '') => {
            const countRes = await runSql(getDefaultCountSql(tableName, condition));
            return countRes.res[0].count;
        };

        /**
         * 获取默认查询语句
         */
        const getDefaultSelectSql = (tableName: string, where: string = '', orderBy: string = '', pageNum: number = 1) => {
            const baseSql = `SELECT * FROM ${tableName} ${where ? 'WHERE ' + where : ''} ${orderBy ? orderBy : ''}`;
            if (state.dbType == 'mysql') {
                return `${baseSql} LIMIT ${(pageNum - 1) * state.defalutLimit}, ${state.defalutLimit};`;
            }
            if (state.dbType == 'postgres') {
                return `${baseSql} OFFSET ${(pageNum - 1) * state.defalutLimit} LIMIT ${state.defalutLimit};`;
            }
            return baseSql;
        };

        /**
         * 获取默认查询统计语句
         */
        const getDefaultCountSql = (tableName: string, where: string = '') => {
            return `SELECT COUNT(*) count FROM ${tableName} ${where ? 'WHERE ' + where : ''}`;
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

            const orderBy = `ORDER BY ${sort.prop} ${sortType}`;
            state.dataTabs[state.activeName].orderBy = orderBy;

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
            dbApi.getSql.request({ id: state.dbId, type: 1, name: state.sqlName, db: state.db }).then((res) => {
                if (res) {
                    setCodermirrorValue(res.sql);
                } else {
                    setCodermirrorValue('');
                }
            });
        };

        const setCodermirrorValue = (value: string) => {
            codemirror.setValue(value);
        };

        const getCodermirrorValue = () => {
            codemirror.getValue();
        };

        /**
         * 获取用户保存的sql模板名称
         */
        const getSqlNames = () => {
            dbApi.getSqlNames
                .request({
                    id: state.dbId,
                    db: state.db,
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
            const sql = codemirror.getValue();
            notEmpty(sql, 'sql内容不能为空');
            notBlank(state.dbId, '请先选择数据库实例');
            await dbApi.saveSql.request({ id: state.dbId, db: state.db, sql: sql, type: 1, name: state.sqlName });
            ElMessage.success('保存成功');

            dbApi.getSqlNames
                .request({
                    id: state.dbId,
                    db: state.db,
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
                await dbApi.deleteDbSql.request({ id: state.dbId, name: state.sqlName, db: state.db });
                ElMessage.success('删除成功');
                getSqlNames();
            } catch (err) {}
        };

        // 清空数据库事件
        const clearDb = () => {
            state.tableName = '';
            state.nowTableName = '';
            state.tableMetadata = [];
            state.dataTabs = {};
            setCodermirrorValue('');
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
                    onRefresh(state.activeName);
                } else {
                    state.queryTab.execRes.data = state.queryTab.execRes.data.filter(
                        (d: any) => !(deleteDatas.findIndex((x: any) => x[primaryKeyColumnName] == d[primaryKeyColumnName]) != -1)
                    );
                    state.queryTab.selectionDatas = [];
                }
            });
        };

        const onGenerateInsertSql = async () => {
            const queryTab = isQueryTab();
            const datas = queryTab ? state.queryTab.selectionDatas : state.dataTabs[state.activeName].selectionDatas;
            isTrue(datas && datas.length > 0, '请先选择要生成insert语句的数据');
            const tableName = state.nowTableName;
            const columns: any = await getColumns(tableName);

            const sqls = [];
            for (let data of datas) {
                let colNames = [];
                let values = [];
                for (let column of columns) {
                    const colName = column.columnName;
                    colNames.push(colName);
                    values.push(wrapValueByType(data[colName]));
                }
                sqls.push(`INSERT INTO ${tableName} (${colNames.join(', ')}) VALUES(${values.join(', ')})`);
            }
            state.genSqlDialog.sql = sqls.join(';\n') + ';';
            state.genSqlDialog.visible = true;
        };

        const wrapValueByType = (val: any) => {
            if (val == null) {
                return 'NULL';
            }
            if (typeof val == 'number') {
                return val;
            }
            return `'${val}'`;
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
            // 转为字符串比较,可能存在数字等
            let text = (row[property] ? row[property] : '') + '';
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
                db: state.db,
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
         * 格式化sql
         */
        const formatSql = () => {
            let selectSql = codemirror.getSelection();
            isTrue(selectSql, '请选中需要格式化的sql');
            codemirror.replaceSelection(sqlFormatter(selectSql));
        };

        const search = async () => {
            const res = await dbApi.dbs.request(state.params);
            state.dbs = res.list;
        };

        return {
            ...toRefs(state),
            codeTextarea,
            changeProjectEnv,
            changeTable,
            cellClick,
            onRunSql,
            exportData,
            removeDataTab,
            onDataTabClick,
            beforeUpload,
            getUploadSqlFileUrl,
            execSqlFileSuccess,
            flexColumnWidth,
            getColumnTip,
            getColumns4Map,
            onConditionRowClick,
            onConfirmCondition,
            onCancelCondition,
            changeSqlTemplate,
            deleteSql,
            saveSql,
            changeDbInstance,
            changeDb,
            clearDb,
            formatSql,
            onBeforeChange,
            onRefresh,
            handlePageChange,
            selectByCondition,
            onCommit,
            addRow,
            onDataSelectionChange,
            onDeleteData,
            onTableSortChange,
            onGenerateInsertSql,
        };
    },
});
</script>

<style lang="scss">
.sqlEditor {
    font-size: 8pt;
    font-weight: 600;
    border: 1px solid #ccc;
    .CodeMirror {
        flex-grow: 1;
        z-index: 1;
        .CodeMirror-code {
            line-height: 19px;
        }
        font-family: 'JetBrainsMono';
    }
}
.el-tabs__header {
    padding: 0 10px;
    background-color: #fff;
}

#data-exec {
    min-height: calc(100vh - 155px);
}
</style>
