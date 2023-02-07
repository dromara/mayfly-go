<template>
  <div class="instances-box layout-aside">
    <el-row type="flex" justify="space-between">
      <el-col :span="24" :style="{
        maxHeight: state.instanceMenuMaxHeight,
        height: state.instanceMenuMaxHeight, 
        overflow:'auto'
      }" class="el-scrollbar flex-auto">
        
        <el-menu background-color="transparent" ref="menuRef">
          <!-- 第一级：tag -->
          <el-sub-menu v-for="tag of instances.tags" :index="tag.tagPath" :key="tag.tagPath">
            <template #title>
              <el-icon><FolderOpened color="#e6a23c"/></el-icon>
              <span>{{ tag.tagPath }}</span>
            </template>
            <!-- 第二级：数据库实例 -->
            <el-sub-menu v-for="inst in instances.tree[tag.tagId]"
                         :index="'redis-instance-' + inst.id"
                         :key="'redis-instance-' + inst.id"
                         @click.prevent="changeInstance(inst)"
            >
              <template #title>
                <el-popover
                    placement="right-start"
                    title="mongo数据库实例信息"
                    trigger="hover"
                    :width="210"
                >
                  <template #reference>
                    <span>&nbsp;&nbsp;<el-icon><MostlyCloudy color="#409eff"/></el-icon>{{ inst.name }}</span>
                  </template>
                  <template #default>
                    <el-form class="instances-pop-form" label-width="55px" :size="'small'">
                      <el-form-item label="名称:">{{inst.name}}</el-form-item>
                      <el-form-item label="链接:">{{inst.host}}</el-form-item>
                    </el-form>
                  </template>
                </el-popover>
              </template>
              <!-- 第三级：数据库 -->
              <el-sub-menu v-for="db in instances.dbs[inst.id]" 
                           :index="inst.id + db.name" 
                           :key="inst.id + db.name"
                           :class="state.nowSchema === (inst.id+db.name) && 'checked'"
                           @click.prevent="changeSchema(inst, db.name)"
              >
                <template #title>
                  &nbsp;&nbsp;&nbsp;&nbsp;<el-icon><Coin color="#67c23a"/></el-icon>
                  <span class="checked-schema">
                    {{ db.name  }} [{{db.keys}}]
                  </span>
                </template>
              </el-sub-menu>
            </el-sub-menu>
          </el-sub-menu>
        </el-menu>
      </el-col>
    </el-row>
  </div>
</template>

<script lang="ts" setup>
import {onBeforeMount, onMounted, reactive, Ref, ref, watch} from 'vue';
import {store} from '@/store';

defineProps({
  instances: {
    type: Object, required: true
  },
})

const emits = defineEmits(['initLoadInstances','changeInstance','changeSchema'])

onBeforeMount(async ()=>{
  await initLoadInstances()
  setHeight()
})

const setHeight = () => {
  state.instanceMenuMaxHeight = window.innerHeight - 140 + 'px';
}

const menuRef = ref(null) as Ref;

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
 * @param fn 选中的实例后的回调函数
 */
const changeInstance = (inst : any, fn?:Function) => {
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

const selectDb = async (val?: any) => {
  const info = val || store.state.redisDbOptInfo.dbOptInfo
  if (info && info.dbId) {
    const {tagPath, dbId} = info
    menuRef.value.open(tagPath);
    menuRef.value.open('redis-instance-' + dbId);
    await changeInstance({id: dbId}, async (dbs: any[]) => {
      await changeSchema({id: dbId}, dbs[0]?.name)
    })
  }
}

onMounted(()=>{
  selectDb();
})

watch(()=>store.state.redisDbOptInfo.dbOptInfo, async newValue =>{
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