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
                        </template>
                    </project-env-select>
                </el-col>
            </el-row>
        </div>

        <el-container style="border: 1px solid #eee; margin-top: 1px; height: 549px">
            <el-container style="margin-left: 2px">
                <el-header style="text-align: left; height: 45px; font-size: 12px; padding: 0px">
                    <el-select v-model="tableName" placeholder="请选择表" @change="changeTable" filterable style="width: 99%">
                        <el-option
                            v-for="item in tableMetadata"
                            :key="item.tableName"
                            :label="item.tableName + (item.tableComment != '' ? `【${item.tableComment}】` : '')"
                            :value="item.tableName"
                        >
                        </el-option>
                    </el-select>
                </el-header>

                <el-main style="padding: 0px; overflow: hidden">
                    <el-table :data="columnMetadata" height="100%" size="mini">
                        <el-table-column prop="columnName" label="名称" show-overflow-tooltip> </el-table-column>
                        <el-table-column prop="columnComment" label="备注" show-overflow-tooltip> </el-table-column>
                        <el-table-column width="120" prop="columnType" label="类型" show-overflow-tooltip> </el-table-column>
                    </el-table>
                </el-main>
            </el-container>
            <el-aside id="sqlcontent" width="75%" style="background-color: rgb(238, 241, 246)">
                <div class="toolbar">
                    <div class="fl">
                        <!-- <el-button v-waves @click="runSql" type="success" icon="video-play"  plain>执行</el-button>

                        <el-button v-waves @click="formatSql" type="primary" icon="magic-stick"  plain>格式化</el-button> -->

                        <el-upload
                            style="display: inline-block; margin-left: 10px"
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
                            <el-button v-waves type="success" icon="video-play" plain>sql脚本执行</el-button>
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
                            class="mr10"
                        >
                            <el-option v-for="item in sqlNames" :key="item" :label="item.database" :value="item">
                                {{ item }}
                            </el-option>
                        </el-select>

                        <el-button v-waves @click="saveSql" type="primary" icon="document-add" plain>保存</el-button>
                        <el-button v-waves @click="deleteSql" type="danger" icon="delete" plain>删除</el-button>
                    </div>
                </div>
                <codemirror
                    @mousemove="listenMouse"
                    @beforeChange="onBeforeChange"
                    class="codesql"
                    ref="cmEditor"
                    language="sql"
                    v-model="sql"
                    :options="cmOptions"
                />
                <el-button-group :style="btnStyle">
                    <el-button v-waves @click="runSql" type="success" icon="video-play" size="small" plain>执行</el-button>
                    <el-button v-waves @click="formatSql" type="primary" icon="magic-stick" size="small" plain>格式化</el-button>
                </el-button-group>
            </el-aside>
        </el-container>
        <el-table
            @cell-dblclick="cellClick"
            style="margin-top: 1px"
            :data="execRes.data"
            size="mini"
            max-height="300"
            :empty-text="execRes.emptyResText"
            stripe
            border
        >
            <el-table-column
                min-width="100"
                :width="flexColumnWidth(item, execRes.data)"
                align="center"
                v-for="item in execRes.tableColumn"
                :key="item"
                :prop="item"
                :label="item"
                show-overflow-tooltip
            >
            </el-table-column>
        </el-table>
        <!-- <el-row v-if="dbId">
            <el-button @click="addRow" type="text" icon="el-icon-plus"></el-button>
        </el-row> -->
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

import sqlFormatter from 'sql-formatter';
import { notNull, notEmpty } from '@/common/assert';
import { ElMessage, ElMessageBox } from 'element-plus';
import ProjectEnvSelect from '../component/ProjectEnvSelect.vue';
import config from '@/common/config';
import { getSession } from '@/common/utils/storage';

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
            dbs: [],
            tables: [],
            dbId: null,
            tableName: '',
            tableMetadata: [],
            columnMetadata: [],
            sqlName: '',
            sqlNames: [],
            sql: '',
            sqlTabs: {
                tabs: [] as any,
                active: '',
                index: 1,
            },
            execRes: {
                tableColumn: [],
                data: [],
                emptyResText: '没有数据',
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
        const runSql = async () => {
            notNull(state.dbId, '请先选择数据库');
            // 没有选中的文本，则为全部文本
            let sql = getSql();
            notNull(sql, '内容不能为空');
            runSqlStr(sql);
        };

        /**
         * 执行sql str
         */
        const runSqlStr = async (sql: string) => {
            state.execRes.tableColumn = [];
            state.execRes.data = [];
            state.execRes.emptyResText = '查询中...';
            console.log(sql);
            const res = await dbApi.sqlExec.request({
                id: state.dbId,
                sql: sql,
            });
            state.execRes.emptyResText = '没有数据';
            state.execRes.tableColumn = res.colNames;
            state.execRes.data = res.res;
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
                // 赋值第一个表信息
                if (state.tableMetadata.length > 0) {
                    state.tableName = state.tableMetadata[0]['tableName'];
                    changeTable(state.tableName, true);
                }
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

        const changeSqlTemplate = () => {
            getUserSql();
        };

        const getUserSql = () => {
            notNull(state.dbId, '请先选择数据库');
            dbApi.getSql.request({ id: state.dbId, type: 1, name: state.sqlName }).then((res) => {
                if (res) {
                    state.sql = res.sql;
                } else {
                    state.sql = '';
                }
            });
        };

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
            notNull(state.dbId, '请先选择数据库');
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
            state.tableMetadata = [];
            state.columnMetadata = [];
            state.execRes.data = [];
            state.execRes.tableColumn = [];
            state.sql = '';
            state.cmOptions.hintOptions.tables = [];
        };

        // 选择表事件
        const changeTable = (tableName: string, execSelectSql: boolean = true) => {
            if (tableName == '') {
                return;
            }
            dbApi.columnMetadata
                .request({
                    id: state.dbId,
                    tableName: tableName,
                })
                .then((res) => {
                    state.columnMetadata = res;
                });

            if (execSelectSql) {
                runSqlStr(`SELECT * FROM ${tableName} ORDER BY create_time DESC LIMIT 25`);
            }
        };
        // 监听单元格点击事件
        const cellClick = (row: any, column: any, cell: any, event: any) => {
            console.log(cell.children[0].tagName);
            let isDiv = cell.children[0].tagName === 'DIV';
            let text = cell.children[0].innerText;
            let div = cell.children[0];
            if (isDiv) {
                let input = document.createElement('input');
                input.setAttribute('value', text);
                input.setAttribute('style', 'height:30px');
                cell.replaceChildren(input);
                input.focus();
                input.addEventListener('blur', () => {
                    div.innerText = input.value;
                    cell.replaceChildren(div);
                    console.log(input.value, column.rawColumnKey, text, 42242);
                    if (input.value !== text) {
                        dbApi.sqlExec.request({
                            id: state.dbId,
                            sql: `UPDATE ${state.tableName} SET ${column.rawColumnKey} = '${input.value}' WHERE ${column.rawColumnKey} = '${text}'`,
                        });
                    }
                });
            }
        };
        // 添加新数据行
        // const addRow = () => {
        //     let obj: any = {};
        //     (state.execRes.tableColumn as any) = state.columnMetadata.map((i) => (i as any).columnName);
        //     state.execRes.tableColumn.forEach((item) => {
        //         obj[item] = 'NULL';
        //     });
        //     (state.execRes.data as any).push(obj);
        //     let values = Object.values(obj).join(',');
        //     console.log(values, 4343);
        //     let sql = `INSERT INTO \`${state.tableName}\` VALUES (${values});`;
        //     // runSqlStr(sql);
        // };

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
                codemirror.value.replaceSelection(sqlFormatter.format(selectSql));
            } else {
                /* 将sql内容进行格式后放入编辑器中*/
                state.sql = sqlFormatter.format(sqlFormatter.format(state.sql));
            }
        };

        const search = async () => {
            const res = await dbApi.dbs.request(state.params);
            state.dbs = res.list;
        };
        /**
         * 获取选择文字，显示隐藏按钮，防抖
         */
        const getSelection = _.debounce((e) => {
            let temp = codemirror.value.getSelection();
            if (temp) {
                state.btnStyle.display = 'block';
                if (!state.btnStyle.left) {
                    state.btnStyle.left = e.target.getBoundingClientRect().left - 150 + 'px';
                    state.btnStyle.top = e.target.getBoundingClientRect().top - 60 + 'px';
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
            runSql,
            beforeUpload,
            getUploadSqlFileUrl,
            execSqlFileSuccess,
            flexColumnWidth,
            changeSqlTemplate,
            deleteSql,
            saveSql,
            changeDb,
            clearDb,
            formatSql,
            onBeforeChange,
            listenMouse,
        };
    },
});
</script>

<style scoped lang="scss">
.codesql {
    font-size: 10pt;
    font-weight: 600;
    font-family: Consolas, Menlo, Monaco, Lucida Console, Liberation Mono, DejaVu Sans Mono, Bitstream Vera Sans Mono, Courier New, monospace, serif;
}
#sqlcontent {
    .CodeMirror {
        height: 300px !important;
    }
}
</style>
