<template>
    <tag-menu :instanceMenuMaxHeight="state.instanceMenuMaxHeight" :tags="instances.tags" ref="menuRef">
        <template #submenu="props">
            <el-sub-menu v-for="inst in instances.tree[props.tag.tagId]" :index="'mongo-instance-' + inst.id"
                :key="'mongo-instance-' + inst.id" @click.stop="changeInstance(inst, () => { })">
                <template #title>
                    <el-popover placement="right-start" title="mongo数据库实例信息" trigger="hover" :width="210">
                        <template #reference>
                            <span>&nbsp;&nbsp;<el-icon>
                                    <MostlyCloudy color="#409eff" />
                                </el-icon>{{ inst.name }}</span>
                        </template>
                        <template #default>
                            <el-form class="instances-pop-form" label-width="55px" :size="'small'">
                                <el-form-item label="名称:">{{ inst.name }}</el-form-item>
                                <el-form-item label="链接:">{{ inst.uri }}</el-form-item>
                            </el-form>
                        </template>
                    </el-popover>
                </template>
                <!-- 第三级：数据库 -->
                <el-sub-menu v-for="db in instances.dbs[inst.id]" :index="inst.id + db.Name" :key="inst.id + db.Name"
                    :class="state.nowSchema === (inst.id + db.Name) && 'checked'"
                    @click.stop="changeSchema(inst, db.Name)">
                    <template #title>
                        &nbsp;&nbsp;&nbsp;&nbsp;<el-icon>
                            <Coin color="#67c23a" />
                        </el-icon>
                        <span class="checked-schema">
                            {{ db.Name }}
                            <span style="color: #8492a6;font-size: 13px">[{{
                                formatByteSize(db.SizeOnDisk)
                            }}]</span>
                        </span>
                    </template>
                    <!-- 第四级 01：表 -->
                    <el-sub-menu :index="inst.id + db.Name + '-table'">
                        <template #title>
                            <div style="width: 100%" @click="loadTableNames(inst, db.Name, () => { })">
                                &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<el-icon>
                                    <Calendar color="#409eff" />
                                </el-icon>
                                <span>集合</span>
                                <el-icon v-show="state.loading[inst.id + db.Name]" class="is-loading">
                                    <Loading />
                                </el-icon>
                            </div>
                        </template>
                        <el-menu-item :index="inst.id + db.Name + '-tableSearch'"
                            :key="inst.id + db.Name + '-tableSearch'">
                            <template #title>
                                &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
                                <el-input size="small" placeholder="过滤" clearable
                                    @change="filterTableName(inst.id, db.Name)"
                                    @keyup="(e: any) => filterTableName(inst.id, db.Name, e)"
                                    v-model="state.filterParam[inst.id + db.Name]" />
                            </template>
                        </el-menu-item>

                        <template v-for="tb in instances.tables[inst.id + db.Name]">
                            <el-menu-item :index="inst.id + db.Name + tb.tableName"
                                :key="inst.id + db.Name + tb.tableName" v-if="tb.show"
                                @click="loadTableData(inst, db.Name, tb.tableName)">
                                <template #title>
                                    <div style="width: 100%">
                                        &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<el-icon>
                                            <Calendar color="#409eff" />
                                        </el-icon>
                                        <span :title="tb.tableComment || ''">{{ tb.tableName }}</span>
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
import { onBeforeMount, reactive } from 'vue';
import { formatByteSize } from '@/common/utils/format';
import TagMenu from '../component/TagMenu.vue';

const props = defineProps({
    instances: {
        type: Object, required: true
    },
})

const emits = defineEmits(['initLoadInstances', 'changeInstance', 'loadTableNames', 'loadTableData', 'changeSchema'])

onBeforeMount(async () => {
    await initLoadInstances()
    setHeight()
})

const setHeight = () => {
    state.instanceMenuMaxHeight = window.innerHeight - 115 + 'px';
}

const state = reactive({
    instanceMenuMaxHeight: '800px',
    nowSchema: '',
    filterParam: {},
    loading: {},

})

/**
 * 初始化加载实例数据
 */
const initLoadInstances = () => {
    emits('initLoadInstances')
}

/**
 * 改变选中的数据库实例
 * @param inst 选中的实例对象
 * @param fn 选中的实例对象后的回调事件
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
/**
 * 加载schema下所有表
 * @param inst 数据库实例
 * @param fn 加载表名后的回调函数，参数：表名list
 * @param schema database名
 */
const loadTableNames = async (inst: any, schema: string, fn: Function) => {
    state.loading[inst.id + schema] = true
    await emits('loadTableNames', inst, schema, (res: any) => {
        state.loading[inst.id + schema] = false
        fn && fn(res)
    })
}
/**
 * 加载选中表数据
 * @param inst 数据库实例
 * @param schema database名
 * @param tableName 表名
 */
const loadTableData = (inst: any, schema: string, tableName: string) => {
    emits('loadTableData', inst, schema, tableName)
}

const filterTableName = (instId: number, schema: string, event?: any) => {
    if (event) {
        state.filterParam[instId + schema] = event.target.value
    }
    let param = state.filterParam[instId + schema] as string
    param = param?.replace('/', '\/')
    const key = instId + schema;
    props.instances.tables[key].forEach((a: any) => {
        a.show = param ? eval('/' + param.split('').join('[_\w]*') + '[_\w]*/ig').test(a.tableName) : true
    })
}

</script>

<style lang="scss">
.instances-pop-form {
    .el-form-item {
        margin-bottom: unset;
    }
}
</style>