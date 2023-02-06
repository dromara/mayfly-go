<template>
    <div class="instances-box layout-aside">
        <el-row type="flex" justify="space-between">
            <el-col :span="24" :style="{
                maxHeight: state.instanceMenuMaxHeight,
                height: state.instanceMenuMaxHeight,
                overflow: 'auto'
            }" class="el-scrollbar flex-auto">

                <el-menu background-color="transparent" :collapse-transition="false">
                    <!-- 第一级：tag -->
                    <el-sub-menu v-for="tag of instances.tags" :index="tag.tagPath" :key="tag.tagPath">
                        <template #title>
                            <el-icon>
                                <FolderOpened color="#e6a23c" />
                            </el-icon>
                            <span>{{ tag.tagPath }}</span>
                        </template>
                        <!-- 第二级：数据库实例 -->
                        <el-sub-menu v-for="inst in instances.tree[tag.tagId]" :index="'mongo-instance-' + inst.id"
                            :key="'mongo-instance-' + inst.id" @click.prevent="changeInstance(inst)">
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
                            <el-sub-menu v-for="db in instances.dbs[inst.id]" :index="inst.id + db.Name"
                                :key="inst.id + db.Name" :class="state.nowSchema === (inst.id + db.Name) && 'checked'"
                                @click.prevent="changeSchema(inst, db.Name)">
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
                                        <div style="width: 100%" @click="loadTableNames(inst, db.Name)">
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
                    </el-sub-menu>
                </el-menu>
            </el-col>
        </el-row>
    </div>
</template>

<script lang="ts" setup>
import { onBeforeMount, reactive } from 'vue';
import { formatByteSize } from '@/common/utils/format';

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
    state.instanceMenuMaxHeight = window.innerHeight - 140 + 'px';
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
 */
const changeInstance = (inst: any) => {
    emits('changeInstance', inst)
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
 * @param schema database名
 */
const loadTableNames = async (inst: any, schema: string) => {
    state.loading[inst.id + schema] = true
    await emits('loadTableNames', inst, schema, () => {
        state.loading[inst.id + schema] = false
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
.instances-box {
    .el-menu {
        width: 100%;
    }

    .el-sub-menu {
        .checked {
            .checked-schema {
                color: var(--el-color-primary);
            }
        }
    }

    .el-sub-menu__title {
        padding-left: 0 !important;
        height: 30px !important;
        line-height: 30px !important;
    }

    .el-menu--vertical:not(.el-menu--collapse):not(.el-menu--popup-container) .el-sub-menu__title {
        padding-right: 10px;
    }

    .el-menu-item {
        padding-left: 0 !important;
        height: 20px !important;
        line-height: 20px !important;
    }

    .el-icon {
        margin: 0;
    }

    .el-sub-menu__icon-arrow {
        top: inherit;
        right: 10px;
    }

}

.instances-pop-form {
    .el-form-item {
        margin-bottom: unset;
    }
}
</style>