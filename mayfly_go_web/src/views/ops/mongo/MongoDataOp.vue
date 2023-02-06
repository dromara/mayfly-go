<template>
    <div>
      <el-row>
        <el-col :span="4">
          <mongo-instance-tree
              @init-load-instances="loadInstances"
              @change-instance="changeInstance"
              @change-schema="changeDatabase"
              @load-table-names="loadTableNames"
              @load-table-data="changeCollection"
              :instances="state.instances"/>
        </el-col>
        <el-col :span="20">
          <el-container id="data-exec" style="border: 1px solid #eee; margin-top: 1px">
            <el-tabs @tab-remove="removeDataTab" @tab-click="onDataTabClick" style="width: 100%; margin-left: 5px"
                     v-model="state.activeName">
              <el-tab-pane closable v-for="dt in state.dataTabs" :key="dt.key" :label="dt.label" :name="dt.key">
                
                <el-row class="mt5 mb5">
                  <el-col :span="2">
                  <el-link @click="findCommand(state.activeName)" icon="refresh" :underline="false" class="">
                  </el-link>
                  <el-link @click="showInsertDocDialog" class="" type="primary" icon="plus" :underline="false">
                  </el-link>
                  </el-col>
                  <el-col :span="22">
                  <el-input ref="findParamInputRef" v-model="dt.findParamStr" placeholder="点击输入相应查询条件"
                            @focus="showFindDialog(dt.key)">
                    <template #prepend>查询参数</template>
                  </el-input>
                  </el-col>
                </el-row>
                <el-row>
                  <el-col :span="6" v-for="item in dt.datas" :key="item">
                    <el-card :body-style="{ padding: '0px', position: 'relative' }">
                      <el-input type="textarea" v-model="item.value" :rows="10" />
                      <div style="padding: 3px; float: right" class="mr5 mongo-doc-btns">
                        <div>
                          <el-link @click="onJsonEditor(item)" :underline="false" type="success"
                                   icon="MagicStick"></el-link>

                          <el-divider direction="vertical" border-style="dashed" />

                          <el-link @click="onSaveDoc(item.value)" :underline="false" type="warning"
                                   icon="DocumentChecked"></el-link>

                          <el-divider direction="vertical" border-style="dashed" />

                          <el-popconfirm @confirm="onDeleteDoc(item.value)" title="确定删除该文档?">
                            <template #reference>
                              <el-link :underline="false" type="danger" icon="DocumentDelete">
                              </el-link>
                            </template>
                          </el-popconfirm>
                        </div>
                      </div>
                    </el-card>
                  </el-col>
                </el-row>
              </el-tab-pane>
            </el-tabs>
          </el-container>
        </el-col>

      </el-row>
        
        <el-dialog width="600px" title="find参数" v-model="findDialog.visible">
            <el-form label-width="70px">
                <el-form-item label="filter">
                    <el-input v-model="findDialog.findParam.filter" type="textarea" :rows="6" clearable
                        auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item label="sort">
                    <el-input v-model="findDialog.findParam.sort" type="textarea" :rows="3" clearable
                        auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item label="limit">
                    <el-input v-model.number="findDialog.findParam.limit" type="number" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item label="skip">
                    <el-input v-model.number="findDialog.findParam.skip" type="number" auto-complete="off"></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div>
                    <el-button @click="findDialog.visible = false">取 消</el-button>
                    <el-button @click="confirmFindDialog" type="primary">确 定</el-button>
                </div>
            </template>
        </el-dialog>

        <el-dialog width="60%" :title="`新增'${state.activeName}'集合文档`" v-model="insertDocDialog.visible"
            :close-on-click-modal="false">
            <monaco-editor v-model="insertDocDialog.doc" language="json" />
            <template #footer>
                <div>
                    <el-button @click="insertDocDialog.visible = false">取 消</el-button>
                    <el-button @click="onInsertDoc" type="primary">确 定</el-button>
                </div>
            </template>
        </el-dialog>

        <el-dialog width="60%" title="json编辑器" v-model="jsonEditorDialog.visible" @close="onCloseJsonEditDialog"
                   :close-on-click-modal="false">
            <monaco-editor v-model="jsonEditorDialog.doc" language="json" />
        </el-dialog>

        <div style="text-align: center; margin-top: 10px"></div>
    </div>
</template>

<script lang="ts" setup>
import {mongoApi} from './api';
import {reactive, ref, toRefs} from 'vue';
import {ElMessage} from 'element-plus';

import {isTrue, notBlank} from '@/common/assert';
import {useStore} from '@/store/index.ts';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import MongoInstanceTree from '@/views/ops/mongo/MongoInstanceTree.vue';

const store = useStore();
const findParamInputRef: any = ref(null);
const state = reactive({
    tags: [],
    mongoList: [] as any,
    query: {
        tagPath: null,
    },
    mongoId: null, // 当前选择操作的mongo
    database: '', // 当前选择操作的库
    collection: '', //当前选中的collection
    activeName: '', // 当前操作的tab
    dataTabs: {} as any, // 数据tabs
    findDialog: {
        visible: false,
        findParam: {
            limit: 0,
            skip: 0,
            filter: '',
            sort: '',
        },
    },
    insertDocDialog: {
        visible: false,
        doc: '',
    },
    jsonEditorDialog: {
        visible: false,
        doc: '',
        item: {} as any,
    },
    instances:{tags:{}, tree:{}, dbs:{}, tables:{}}
});

const {
    findDialog,
    insertDocDialog,
    jsonEditorDialog,
} = toRefs(state)

const changeInstance = async (inst: any) => {
  if (inst) {
    if (!state.instances.dbs[inst.id]) {
      const res = await mongoApi.databases.request({id: inst.id});
      state.instances.dbs[inst.id] = res.Databases;
      console.log(res.Databases)
    }
  }
}

const changeDatabase = async (inst: any, database: string) => {
};

const loadTableNames = async (inst: any, database: string, fn:Function) => {
  let tbs = await mongoApi.collections.request({ id: inst.id, database });
  let tables = [];
  for(let tb of tbs){
    tables.push({tableName: tb, show: true})
  }
  state.instances.tables[inst.id+database] = tables
  fn()
}

const changeCollection = (inst: any, schema: string, collection: string) => {
  state.collection = collection
  state.mongoId = inst.id
  state.database = schema
  let key = inst.id + schema +collection
  let dataTab = state.dataTabs[key];
  if (!dataTab) {
    // 默认查询参数
    const findParam = {
      filter: '{}',
      sort: '{"_id": -1}',
      skip: 0,
      limit: 12,
    };
    state.dataTabs[key] = {
      key: key,
      label: schema+'.'+collection,
      name: inst.id+schema+collection,
      datas: [],
      findParamStr: JSON.stringify(findParam),
      findParam,
    };
  }
  state.activeName = key;
  findCommand(key);
};

const showFindDialog = (key: string) => {
    // 获取当前tab的索引位置，将其输入框失去焦点，防止输入以及重复获取焦点
    const dataTabNames = Object.keys(state.dataTabs);
    for (let i = 0; i < dataTabNames.length; i++) {
        if (key == dataTabNames[i]) {
            findParamInputRef.value[i].blur();
        }
    }

    state.findDialog.findParam = state.dataTabs[key].findParam;
    state.findDialog.visible = true;
};

const confirmFindDialog = () => {
    state.dataTabs[state.activeName].findParam = state.findDialog.findParam;
    state.dataTabs[state.activeName].findParamStr = JSON.stringify(state.findDialog.findParam);
    state.findDialog.visible = false;
    findCommand(state.activeName);
};

const findCommand = async (key: string) => {
    const dataTab = state.dataTabs[key];
    const findParma = dataTab.findParam;
    let filter, sort;
    try {
        filter = findParma.filter ? JSON.parse(findParma.filter) : {};
        sort = findParma.sort ? JSON.parse(findParma.sort) : {};
    } catch (e) {
        ElMessage.error('filter或sort字段json字符串值错误。注意: json key需双引号');
        return;
    }
    const datas = await mongoApi.findCommand.request({
        id: state.mongoId,
        database: state.database,
        collection: state.collection,
        filter,
        sort,
        limit: findParma.limit || 12,
        skip: findParma.skip || 0,
    });
    state.dataTabs[key].datas = wrapDatas(datas);
};

/**
 * 包装mongo查询回来的对象，即将其都转为json字符串并用value属性值描述，方便显示
 */
const wrapDatas = (datas: any) => {
    const wrapDatas = [] as any;
    if (!datas) {
        return wrapDatas;
    }
    for (let data of datas) {
        wrapDatas.push({ value: JSON.stringify(data, null, 4) });
    }
    return wrapDatas;
};

const showInsertDocDialog = () => {
    // tab数据中的第一个文档，因为该集合的文档都类似，故使用第一个文档赋值至需要新增的文档输入框，方便直接修改新增
    const datasFirstDoc = state.dataTabs[state.activeName].datas[0];
    let doc = '';
    if (datasFirstDoc) {
        // 移除_id字段，因为新增无需该字段
        const docObj = JSON.parse(datasFirstDoc.value);
        delete docObj['_id'];
        doc = JSON.stringify(docObj, null, 4);
    }
    state.insertDocDialog.doc = doc;
    state.insertDocDialog.visible = true;
};

const onInsertDoc = async () => {
    let docObj;
    try {
        docObj = JSON.parse(state.insertDocDialog.doc);
    } catch (e) {
        ElMessage.error('文档内容错误,无法解析为json对象');
    }
    const res = await mongoApi.insertCommand.request({
        id: state.mongoId,
        database: state.database,
        collection: state.activeName,
        doc: docObj,
    });
    isTrue(res.InsertedID, '新增失败');
    ElMessage.success('新增成功');
    findCommand(state.activeName);
    state.insertDocDialog.visible = false;
};

const onJsonEditor = (item: any) => {
    state.jsonEditorDialog.item = item;
    state.jsonEditorDialog.doc = item.value;
    state.jsonEditorDialog.visible = true;
};

const onCloseJsonEditDialog = () => {
    state.jsonEditorDialog.item.value = JSON.stringify(JSON.parse(state.jsonEditorDialog.doc), null, 4);
};

const onSaveDoc = async (doc: string) => {
    const docObj = parseDocJsonString(doc);
    const id = docObj._id;
    notBlank(id, '文档的_id属性不存在');
    delete docObj['_id'];
    const res = await mongoApi.updateByIdCommand.request({
        id: state.mongoId,
        database: state.database,
        collection: state.collection,
        docId: id,
        update: { $set: docObj },
    });
    isTrue(res.ModifiedCount == 1, '修改失败');
    ElMessage.success('保存成功');
};

const onDeleteDoc = async (doc: string) => {
    const docObj = parseDocJsonString(doc);
    const id = docObj._id;
    notBlank(id, '文档的_id属性不存在');
    const res = await mongoApi.deleteByIdCommand.request({
        id: state.mongoId,
        database: state.database,
        collection: state.collection,
        docId: id,
    });
    isTrue(res.DeletedCount == 1, '删除失败');
    ElMessage.success('删除成功');
    findCommand(state.activeName);
};

/**
 * 将json字符串解析为json对象
 */
const parseDocJsonString = (doc: string) => {
    try {
        return JSON.parse(doc);
    } catch (e) {
        ElMessage.error('文档内容解析为json对象失败');
        throw e;
    }
};

/**
 * 数据tab点击
 */
const onDataTabClick = (tab: any) => {
    const name = tab.props.name;
    // 修改选择框绑定的表信息
    state.collection = name;
};

const removeDataTab = (targetName: string) => {
    const tabNames = Object.keys(state.dataTabs);
    let activeName = state.activeName;
    tabNames.forEach((name, index) => {
        if (name === targetName) {
            const nextTab = tabNames[index + 1] || tabNames[index - 1];
            if (nextTab) {
                activeName = nextTab;
            }
        }
    });
    state.activeName = activeName;
    // 如果移除最后一个数据tab，则将选择框绑定的collection置空
    if (activeName == targetName) {
        state.collection = '';
    } else {
        state.collection = activeName;
    }

    delete state.dataTabs[targetName];
};

const loadInstances = async () => {
  const res = await mongoApi.mongoList.request({pageNum: 1, pageSize: 1000,});
  if(!res.total) return
  state.instances = {tags:{}, tree:{}, dbs:{}, tables:{}} ; // 初始化变量
  for (const db of res.list) {
    let arr = state.instances.tree[db.tagId] || []
    const {tagId, tagPath} = db
    // tags
    state.instances.tags[db.tagId]={tagId, tagPath}
    // 实例
    arr.push(db)
    state.instances.tree[db.tagId] = arr;
  }
}

</script>

<style>
.mongo-doc-btns {
    position: absolute;
    z-index: 2;
    right: 3px;
    top: 2px;
    max-width: 120px;
}
</style>
