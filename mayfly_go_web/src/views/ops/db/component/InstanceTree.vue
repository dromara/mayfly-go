<template>
  <div class="instances-box layout-aside">
    <el-row type="flex" justify="space-between">
      <el-col :span="24" :style="{maxHeight: instanceMenuMaxHeight,height: instanceMenuMaxHeight, overflow:'auto'}" class="el-scrollbar flex-auto">
        <el-menu background-color="transparent" ref="menuRef">
          <!-- 第一级：tag -->
          <el-sub-menu v-for="tag of instances.tags" :index="tag.tagPath" :key="tag.tagPath">
            <template #title>
              <el-icon><FolderOpened color="#e6a23c"/></el-icon>
              <span>{{ tag.tagPath }}</span>
            </template>
            <!-- 第二级：数据库实例 -->
            <el-sub-menu v-for="inst in instances.tree[tag.tagId]"
                         :index="'instance-' + inst.id"
                         :key="'instance-' + inst.id"
                         @click="changeInstance(inst, ()=>{})"
            >
              <template #title>
                <el-popover
                    placement="right-start"
                    title="数据库实例信息"
                    trigger="hover"
                    :width="210"
                >
                  <template #reference>
                    <span>&nbsp;&nbsp;<el-icon><MostlyCloudy color="#409eff"/></el-icon>{{ inst.name }}</span>
                  </template>
                  <template #default>
                    <el-form class="instances-pop-form" label-width="55px" :size="'small'">
                      <el-form-item label="类型:">{{inst.type}}</el-form-item>
                      <el-form-item label="链接:">{{inst.host}}:{{inst.port}}</el-form-item>
                      <el-form-item label="用户:">{{inst.username}}</el-form-item>
                      <el-form-item v-if="inst.remark" label="备注:">{{inst.remark}}</el-form-item>
                    </el-form>
                  </template>
                </el-popover>
              </template>
              <!-- 第三级：数据库 -->
              <el-sub-menu v-for="schema in instances.dbs[inst.id]" 
                           :index="inst.id + schema" 
                           :key="inst.id + schema"
                           :class="state.nowSchema === (inst.id+schema) && 'checked'"
                           @click="changeSchema(inst, schema)"
              >
                <template #title>
                  &nbsp;&nbsp;&nbsp;&nbsp;<el-icon><Coin color="#67c23a"/></el-icon>
                  <span class="checked-schema">{{ schema }}</span>
                </template>
                <!-- 第四级 01：表 -->
                <el-sub-menu :index="inst.id + schema + '-table'" >
                  <template #title>
                    <div style="width: 100%" @click="loadTableNames(inst, schema, ()=>{})">
                      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<el-icon><Calendar color="#409eff"/></el-icon>
                      <span>表</span>
                      <el-icon v-show="state.loading[inst.id + schema]" class="is-loading"><Loading /></el-icon>
                    </div>
                  </template>
                  <el-menu-item :index="inst.id + schema + '-tableSearch'"
                                :key="inst.id + schema + '-tableSearch'">
                    <template #title>
                      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
                      <el-input size="small" placeholder="过滤表" clearable
                                @change="filterTableName(inst.id, schema)"
                                @keyup="e => filterTableName(inst.id, schema, e)"
                                v-model="state.filterParam[inst.id+schema]"/>
                    </template>
                  </el-menu-item>
                  
                  <template v-for="tb in instances.tables[inst.id+schema]" >
                    <el-menu-item :index="inst.id + schema + tb.tableName"
                                  :key="inst.id + schema + tb.tableName"
                                  v-if="tb.show"
                                  @click="loadTableData(inst, schema, tb.tableName)"
                    >
                      <template #title>
                        <div style="width: 100%" >
                          &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<el-icon><Calendar color="#409eff"/></el-icon>
                          <span :title="tb.tableComment||''">{{tb.tableName}}</span>
                        </div>
                      </template>
                    </el-menu-item>
                  </template>
                </el-sub-menu>
                <!-- 第四级 02：sql -->
                <el-sub-menu :index="inst.id + schema + '-sql'">
                  <template #title>
                    &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<el-icon><List color="#f56c6c"/></el-icon>
                    <span>sql</span>
                  </template>

                  <template v-for="sql in instances.sqls[inst.id+schema]" >
                    <el-menu-item :index="inst.id + schema + sql.name"
                                  :key="inst.id + schema + sql.name"
                                  v-if="sql.show"
                                  @click="loadSql(inst, schema, sql.name)"
                    >
                      <template #title>
                        <div style="width: 100%" >
                          &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<el-icon><Calendar color="#409eff"/></el-icon>
                          <span>{{sql.name}}</span>
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
import {nextTick, onBeforeMount, onMounted, reactive, ref, Ref, watch} from 'vue';
import {store} from '@/store';

const props = defineProps({
  instanceMenuMaxHeight: {
    type: [Number, String],
  },
  instances: {
    type: Object, required: true
  },
})

const emits = defineEmits(['initLoadInstances','changeInstance','loadTableNames','loadTableData','changeSchema'])

onBeforeMount(async ()=>{
  await initLoadInstances()
})

const menuRef = ref(null) as Ref

const state = reactive({
  nowSchema: '',
  filterParam: {},
  loading: {}
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
 * @param fn 选中的实例对象后的回调函数
 */
const changeInstance = (inst : any, fn: Function) => {
  emits('changeInstance', inst, fn)
}
/**
 * 改变选中的数据库schema
 * @param inst 选中的实例对象
 * @param schema 选中的数据库schema
 */
const changeSchema = (inst : any, schema: string) => {
  state.nowSchema = inst.id + schema
  emits('changeSchema', inst, schema)
}
/**
 * 加载schema下所有表
 * @param inst 数据库实例
 * @param schema database名
 * @param fn 加载表集合后的回调函数，参数：res 表集合
 */
const loadTableNames = async (inst: any, schema: string, fn: Function) => {
  state.loading[inst.id+schema] = true
  await emits('loadTableNames', inst, schema, (res: any[])=>{
    state.loading[inst.id+schema] = false
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
  if(event){
    state.filterParam[instId+schema] = event.target.value
  }
  let param = state.filterParam[instId+schema] as string 
  param = param?.replace('/','\/')
  const key = instId + schema;
  props.instances.tables[key].forEach((a:any) =>{
    a.show = param?eval('/'+param.split('').join('[_\w]*')+'[_\w]*/ig').test(a.tableName):true
  })
}

const selectDb = async (val?: any) => {
  let info = val || store.state.sqlExecInfo.dbOptInfo;
  if (info && info.dbId) {
    const {tagPath, dbId, db} = info
    menuRef.value.open(tagPath);
    menuRef.value.open('instance-' + dbId);
    await changeInstance({id: dbId}, () => {
      // 加载数据库
      nextTick(async () => {
        menuRef.value.open(dbId + db)
        state.nowSchema = (dbId+db)
        // 加载集合列表
        await nextTick(async () => {
          await loadTableNames({id: dbId}, db, (res: any[]) => {
            // 展开集合列表
            menuRef.value.open(dbId + db + '-table')
            console.log(res)
          })
        })
      })
    })
  }
}

onMounted(()=>{
  selectDb();
})

watch(()=>store.state.sqlExecInfo.dbOptInfo, async newValue => {
  await selectDb(newValue)
})


</script>

<style lang="scss">
.instances-box {
  .el-menu{
    width: 275px;
  }
  .el-sub-menu{
    .checked{
      .checked-schema{
        color: var(--el-color-primary);
      }
    }
  }
  .el-sub-menu__title{
    padding-left: 0 !important;
    height: 30px !important;
    line-height: 30px !important;
  }
  .el-menu--vertical:not(.el-menu--collapse):not(.el-menu--popup-container) .el-sub-menu__title{
    padding-right: 10px;
  }
  .el-menu-item{
    padding-left: 0 !important;
    height: 20px !important;
    line-height: 20px !important;
  }
  .el-icon{
    margin: 0;
  }
  .el-sub-menu__icon-arrow{
    top:inherit;
    right: 10px;
  }

}
.instances-pop-form{
  .el-form-item{
    margin-bottom: unset;
  }
}
</style>