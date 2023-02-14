<template>
    <tag-menu :instanceMenuMaxHeight="instanceMenuMaxHeight" :tags="tags" ref="menuRef">
        <template #submenu="props">
            <!-- 第二级：数据库实例 -->
            <el-sub-menu v-for="inst in tree[props.tag.tagId]" :index="'instance-' + inst.id"
                :key="'instance-' + inst.id" @click.stop="changeInstance(inst, () => { })">
                <template #title>
                    <el-popover placement="right-start" title="数据库实例信息" trigger="hover" :width="210">
                        <template #reference>
                            <span class="ml10">
                                <el-icon>
                                    <MostlyCloudy color="#409eff" />
                                </el-icon>{{ inst.name }}
                            </span>
                        </template>
                        <template #default>
                            <el-form class="instances-pop-form" label-width="55px" :size="'small'">
                                <el-form-item label="类型:">{{ inst.type }}</el-form-item>
                                <el-form-item label="链接:">{{ inst.host }}:{{ inst.port }}</el-form-item>
                                <el-form-item label="用户:">{{ inst.username }}</el-form-item>
                                <el-form-item v-if="inst.remark" label="备注:">{{ inst.remark }}</el-form-item>
                            </el-form>
                        </template>
                    </el-popover>
                </template>
                <el-menu-item v-if="dbs[inst.id]?.length> 20" :index="'schema-filter-' + inst.id" :key="'schema-filter-' + inst.id">
                    <template #title>
                        &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
                        <el-input size="small" placeholder="过滤数据库" clearable
                                  @change="filterSchemaName(inst.id)"
                                  @keyup="(e: any) => filterSchemaName(inst.id, e)"
                                  v-model="state.schemaFilterParam[inst.id]" />
                    </template>
                </el-menu-item>
                <!-- 第三级：数据库 -->
                <el-sub-menu v-show="schema.show" v-for="schema in dbs[inst.id]" :index="inst.id + schema.name" :key="inst.id + schema.name"
                    :class="state.nowSchema === (inst.id + schema.name) && 'checked'"
                    @click.stop="changeSchema(inst, schema.name)">
                    <template #title>
                        <span class="checked-schema ml20">
                            <el-icon>
                                <Coin color="#67c23a" />
                            </el-icon>
                            <span v-html="schema.showName || schema.name"></span>
                        </span>
                    </template>
                    <!-- 第四级 01：表 -->
                    <el-sub-menu :index="inst.id + schema.name + '-table'">
                        <template #title>
                            <div class="ml30" style="width: 100%" @click="loadSchemaTables(inst, schema.name)">
                                <el-icon>
                                    <Calendar color="#409eff" />
                                </el-icon>
                                <span>表</span>
                                <el-icon v-show="state.loading[inst.id + schema.name]" class="is-loading">
                                    <Loading />
                                </el-icon>
                            </div>
                        </template>
                        <el-menu-item v-if="tables[inst.id + schema.name]?.length> 20"  :index="inst.id + schema.name + '-tableSearch'"
                            :key="inst.id + schema.name + '-tableSearch'">
                            <template #title>
                                <span class="ml35">
                                    <el-input size="small" placeholder="表名、备注过滤表" clearable
                                        @change="filterTableName(inst.id, schema.name)"
                                        @keyup="(e: any) => filterTableName(inst.id, schema.name, e)"
                                        v-model="state.filterParam[inst.id + schema.name]" />
                                </span>
                            </template>
                        </el-menu-item>

                        <template v-for="tb in tables[inst.id + schema.name]">
                            <el-menu-item :index="inst.id + schema.name + tb.tableName"
                                :key="inst.id + schema.name + tb.tableName" v-if="tb.show"
                                @click="clickSchemaTable(inst, schema.name, tb.tableName)">
                                <template #title>
                                    <div class="ml35" style="width: 100%">
                                        <el-icon>
                                            <Calendar color="#409eff" />
                                        </el-icon>
                                        <el-tooltip v-if="tb.tableComment" effect="customized"
                                            :content="tb.tableComment" placement="right" >
                                          <span v-html="tb.showName || tb.tableName"></span>
                                        </el-tooltip>
                                        <span v-else v-html="tb.showName || tb.tableName"></span>
                                    </div>
                                </template>
                            </el-menu-item>
                        </template>
                    </el-sub-menu>
                    <!-- 第四级 02：sql -->
                    <el-sub-menu @click.stop="loadSqls(inst, schema.name)" :index="inst.id + schema.name + '-sql'">
                        <template #title>
                            <span class="ml30">
                                <el-icon>
                                    <List color="#f56c6c" />
                                </el-icon>
                                <span>sql</span>
                            </span>
                        </template>

                        <template v-for="sql in sqls[inst.id + schema.name]">
                            <el-menu-item v-if="sql.show" :index="inst.id + schema.name + sql.name"
                                :key="inst.id + schema.name + sql.name" @click="clickSqlName(inst, schema.name, sql.name)">
                                <template #title>
                                    <div class="ml35" style="width: 100%">
                                        <el-icon>
                                            <Document />
                                        </el-icon>
                                        <span>{{ sql.name }}</span>
                                    </div>
                                </template>
                            </el-menu-item>
                        </template>
                    </el-sub-menu>
                </el-sub-menu>
            </el-sub-menu>
        </template>
    </tag-menu>
</template>

<script lang="ts" setup>
import { onBeforeMount, reactive, toRefs } from 'vue';
import TagMenu from '../../component/TagMenu.vue';
import { dbApi } from '../api';
import { DbInst } from '../db';

const emits = defineEmits(['changeInstance', 'clickSqlName', 'clickSchemaTable', 'changeSchema', 'loadSqlNames'])

onBeforeMount(async () => {
    await loadInstances();
    state.instanceMenuMaxHeight = window.innerHeight - 140 + 'px';
})

const state = reactive({
    tags: {},
    tree: {},
    dbs: {},
    tables: {},
    sqls: {},
    nowSchema: '',
    filterParam: {},
    schemaFilterParam: {},
    loading: {},
    instanceMenuMaxHeight: '850px',
})

const {
    instanceMenuMaxHeight,
    tags,
    tree,
    dbs,
    sqls,
    tables,
} = toRefs(state)

// 加载实例数据
const loadInstances = async () => {
    const res = await dbApi.dbs.request({ pageNum: 1, pageSize: 1000, })
    if (!res.total) return
    // state.instances = { tags: {}, tree: {}, dbs: {}, tables: {}, sqls: {} }; // 初始化变量
    for (const db of res.list) {
        let arr = state.tree[db.tagId] || []
        const { tagId, tagPath } = db
        // tags
        state.tags[db.tagId] = { tagId, tagPath }

        // tree
        arr.push(db)
        state.tree[db.tagId] = arr;

        // dbs
        let databases = db.database.split(' ')
        let dbs = [] as any [];
        databases.forEach((a: string) =>dbs.push({name: a, show: true}))
        state.dbs[db.id] = dbs
    }
}

/**
 * 改变选中的数据库实例
 * @param inst 选中的实例对象
 * @param fn 选中的实例对象后的回调函数
 */
const changeInstance = (inst: any, fn: Function) => {
    emits('changeInstance', inst, fn)
}
/**
 * 改变选中的数据库schema
 * @param inst 选中的实例对象
 * @param schema 选中的数据库schema
 */
const changeSchema = (inst: any, schema: string) => {
    state.nowSchema = inst.id + schema
    emits('changeSchema', inst, schema)
}

/** 加载schema下所有表
 * 
 * @param inst 数据库实例
 * @param schema database名
 */
const loadSchemaTables = async (inst: any, schema: string) => {
    const key = getSchemaKey(inst.id, schema);
    state.loading[key] = true
    try {
        let { id } = inst
        let tables = await DbInst.getInst(id, inst.type).loadTables(schema);
        tables && tables.forEach((a: any) => a.show = true)
        state.tables[key] = tables;
        changeSchema(inst, schema);
    } finally {
        state.loading[key] = false
    }
}

/**
 * 加载选中表数据
 * @param inst 数据库实例
 * @param schema database名
 * @param tableName 表名
 */
const clickSchemaTable = (inst: any, schema: string, tableName: string) => {
    emits('clickSchemaTable', inst, schema, tableName)
}

const filterTableName = (instId: number, schema: string, event?: any) => {
    const key = getSchemaKey(instId, schema)
    if (event) {
        state.filterParam[key] = event.target.value
    }
    let param = state.filterParam[key] as string
    state.tables[key].forEach((a: any) => {
        let {match, showName} = matchAndHighLight(param, a.tableName+a.tableComment, a.tableName)
        a.show = match;
        a.showName = showName
    })
}

const filterSchemaName = (instId: number, event?: any) => {
    if (event) {
        state.schemaFilterParam[instId] = event.target.value
    }
    let param = state.schemaFilterParam[instId] as string
    param = param?.replace('/', '\/')
    state.dbs[instId].forEach((a: any) => {
        let {match, showName} = matchAndHighLight(param, a.name, a.name)
        a.show = match
        a.showName = showName
    })
}

const matchAndHighLight = (searchParam: string, param: string, title: string): {match: boolean, showName: string} => {
    if(!searchParam){
        return  {match: true, showName: ''}
    }
    let str = '';
    for(let c of searchParam?.replace('/', '\/')){
        str += `(${c}).*`
    }
    let regex = eval(`/${str}/i`)
    let res = param.match(regex);
    if(res?.length){
        if(res?.length){
            let tmp = '', showName = '';
            for(let i =1; i<=res.length-1; i++){
                let head = (tmp || title).replace(res[i], `###${res[i]}!!!`);
                let idx = head.lastIndexOf('!!!')+3;
                tmp = head.substring(idx);
                showName += head.substring(0, idx)
                if(!tmp){
                    break
                }
            }
            showName += tmp;
            showName = showName.replaceAll('###','<span style="color: red">')
            showName = showName.replaceAll('!!!','</span>')
            return {match: true, showName}
        }
    }
  
    return {match: false, showName: ''}

}

/**
 * 加载用户保存的sql脚本
 * 
 * @param inst 
 * @param schema 
 */
const loadSqls = async (inst: any, schema: string) => {
    const key = getSchemaKey(inst.id, schema)
    let sqls = state.sqls[key];
    if (!sqls) {
        const sqls = await dbApi.getSqlNames.request({ id: inst.id, db: schema, })
        sqls && sqls.forEach((a: any) => a.show = true)
        state.sqls[key] = sqls;
    } else {
        sqls.forEach((a: any) => a.show = true);
    }
}

const reloadSqls = async (inst: any, schema: string) => {
    const sqls = await dbApi.getSqlNames.request({ id: inst.id, db: schema, })
    sqls && sqls.forEach((a: any) => a.show = true)
    state.sqls[getSchemaKey(inst.id, schema)] = sqls;
}

/**
 * 点击sql模板名称时间，加载用户保存的指定名称的sql内容，并回调子组件指定事件
 */
const clickSqlName = async (inst: any, schema: string, sqlName: string) => {
    emits('clickSqlName', inst, schema, sqlName)
    changeSchema(inst, schema);
}

/**
 * 根据实例以及库获取对应的唯一id
 * 
 * @param inst  数据库实例
 * @param schema 数据库
 */
const getSchemaKey = (instId: any, schema: string) => {
    return instId + schema;
}

const getSchemas = (dbId: any) => {
    return state.dbs[dbId] || []
}

defineExpose({
    getSchemas,
    reloadSqls,
})
</script>

<style lang="scss">
.instances-pop-form {
    .el-form-item {
        margin-bottom: unset;
    }
}
</style>