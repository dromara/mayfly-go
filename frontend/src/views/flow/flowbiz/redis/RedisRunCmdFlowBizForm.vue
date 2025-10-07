<template>
    <el-form :model="bizForm" ref="formRef" :rules="rules" label-width="auto">
        <el-form-item prop="id" label="DB" required>
            <ResourceSelect
                v-bind="$attrs"
                v-model="selectRedis"
                @change="changeRedis"
                :resource-type="TagResourceTypeEnum.Redis.value"
                :tag-path-node-type="NodeTypeTagPath"
                :placeholder="$t('flow.selectRedisPlaceholder')"
            >
            </ResourceSelect>
        </el-form-item>

        <el-form-item prop="cmd" label="CMD" required>
            <el-input type="textarea" v-model="bizForm.cmd" :placeholder="$t('flow.cmdPlaceholder')" :rows="5" />
        </el-form-item>
    </el-form>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import ResourceSelect from '@/views/ops/resource/ResourceSelect.vue';
import { NodeType, TagTreeNode } from '@/views/ops/component/tag';
import { redisApi } from '@/views/ops/redis/api';
import { sleep } from '@/common/utils/loading';
import { useI18n } from 'vue-i18n';
import { Rules } from '@/common/rule';
import { RedisIcon } from '@/views/ops/redis/resource';

const { t } = useI18n();

const rules = {
    id: [
        {
            required: true,
            message: t('flow.selectRedisPlaceholder'),
            trigger: ['change', 'blur'],
        },
    ],
    cmd: [Rules.requiredInput('flow.runCmd')],
};

// tagpath 节点类型
const NodeTypeTagPath = new NodeType(TagTreeNode.TagPath).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const res = await redisApi.redisList.request({ tagPath: parentNode.key });
    if (!res.total) {
        return [];
    }

    const redisInfos = res.list;
    await sleep(100);
    return redisInfos.map((x: any) => {
        x.tagPath = parentNode.key;
        return new TagTreeNode(`${x.code}`, x.name, NodeTypeRedis).withParams(x).withIcon(RedisIcon);
    });
});

// redis实例节点类型
const NodeTypeRedis = new NodeType(1).withLoadNodesFunc(async (parentNode: TagTreeNode) => {
    const redisInfo = parentNode.params;

    let dbs: TagTreeNode[] = redisInfo.db.split(',').map((x: string) => {
        return new TagTreeNode(x, `db${x}`, 2 as any)
            .withIsLeaf(true)
            .withParams({
                id: redisInfo.id,
                db: x,
                name: `db${x}`,
                keys: 0,
                tagPath: redisInfo.tagPath,
                redisName: redisInfo.name,
                code: redisInfo.code,
            })
            .withIcon({ name: 'Coin', color: '#67c23a' });
    });

    if (redisInfo.mode == 'cluster') {
        return dbs;
    }

    const res = await redisApi.redisInfo.request({ id: redisInfo.id, host: redisInfo.host, section: 'Keyspace' });
    for (let db in res.Keyspace) {
        for (let d of dbs) {
            if (db == d.params.name) {
                d.params.keys = res.Keyspace[db]?.split(',')[0]?.split('=')[1] || 0;
            }
        }
    }
    // 替换label
    dbs.forEach((e: any) => {
        e.label = `${e.params.name}`;
    });
    return dbs;
});

const emit = defineEmits(['changeResourceCode']);

const formRef: any = ref(null);

const bizForm = defineModel<any>('bizForm', {
    default: {
        id: 0,
        db: 0,
        cmd: '',
    },
});

const redisName = ref('');
const tagPath = ref('');

const selectRedis = computed({
    get: () => {
        return redisName.value ? `${tagPath.value} > ${redisName.value} > db${bizForm.value.db}` : '';
    },
    set: () => {
        //
    },
});

const changeRedis = (nodeData: TagTreeNode) => {
    const params = nodeData.params;
    tagPath.value = params.tagPath;
    redisName.value = params.redisName;
    bizForm.value.id = params.id;
    bizForm.value.db = parseInt(params.db);

    changeResourceCode(params.code);
};

const changeResourceCode = async (redisCode: any) => {
    emit('changeResourceCode', TagResourceTypeEnum.Redis.value, redisCode);
};

const validateBizForm = async () => {
    return formRef.value.validate();
};

const resetBizForm = () => {
    //重置表单域
    formRef.value.resetFields();
    bizForm.value.id = 0;
    bizForm.value.db = 0;
    bizForm.value.cmd = '';
};

defineExpose({ validateBizForm, resetBizForm });
</script>
<style lang="scss"></style>
