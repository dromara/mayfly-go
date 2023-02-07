<template>
    <tag-menu :instanceMenuMaxHeight="state.instanceMenuMaxHeight" :tags="instances.tags" ref="menuRef">
        <template #submenu="props">
            <el-sub-menu v-for="inst in instances.tree[props.tag.tagId]" :index="'redis-instance-' + inst.id"
                :key="'redis-instance-' + inst.id" @click.stop="changeInstance(inst)">
                <template #title>
                    <el-popover placement="right-start" title="redis实例信息" trigger="hover" :width="210">
                        <template #reference>
                            <span>&nbsp;&nbsp;<el-icon>
                                    <MostlyCloudy color="#409eff" />
                                </el-icon>{{ inst.name }}</span>
                        </template>
                        <template #default>
                            <el-form class="instances-pop-form" label-width="55px" :size="'small'">
                                <el-form-item label="名称:">{{ inst.name }}</el-form-item>
                                <el-form-item label="链接:">{{ inst.host }}</el-form-item>
                                <el-form-item label="备注:">{{ inst.remark }}</el-form-item>
                            </el-form>
                        </template>
                    </el-popover>
                </template>
                <!-- 第三级：数据库 -->
                <el-menu-item v-for="db in instances.dbs[inst.id]" :index="inst.id + db.name" :key="inst.id + db.name"
                    :class="state.nowSchema === (inst.id + db.name) && 'checked'"
                    @click="changeSchema(inst, db.name)">
                    <template #title>
                        &nbsp;&nbsp;&nbsp;&nbsp;<el-icon>
                            <Coin color="#67c23a" />
                        </el-icon>
                        <span class="checked-schema">
                            {{ db.name }} [{{ db.keys }}]
                        </span>
                    </template>
                </el-menu-item>
            </el-sub-menu>
        </template>
    </tag-menu>
</template>

<script lang="ts" setup>
import { onBeforeMount, onMounted, reactive, Ref, ref, watch } from 'vue';
import { store } from '@/store';
import TagMenu from '../component/TagMenu.vue';

defineProps({
    instances: {
        type: Object, required: true
    },
})

const emits = defineEmits(['initLoadInstances', 'changeInstance', 'changeSchema'])

onBeforeMount(async () => {
    await initLoadInstances()
    setHeight()
})

const setHeight = () => {
    state.instanceMenuMaxHeight = window.innerHeight - 115 + 'px';
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
const changeInstance = (inst: any, fn?: Function) => {
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

const selectDb = async (val?: any) => {
    const info = val || store.state.redisDbOptInfo.dbOptInfo
    if (info && info.dbId) {
        const { tagPath, dbId } = info
        menuRef.value.open(tagPath);
        menuRef.value.open('redis-instance-' + dbId);
        await changeInstance({ id: dbId }, async (dbs: any[]) => {
            await changeSchema({ id: dbId }, dbs[0]?.name)
        })
    }
}

onMounted(() => {
    selectDb();
})

watch(() => store.state.redisDbOptInfo.dbOptInfo, async newValue => {
    await selectDb(newValue)
})

</script>

<style lang="scss">
.instances-pop-form {
    .el-form-item {
        margin-bottom: unset;
    }
}
</style>