<template>
    <el-dialog :title="t('es.makeSearchParam')" v-model="visible" :width="state.searchBoxWidth" class="es-search-form-inline">
        <el-tabs v-model="state.activeName">
            <el-tab-pane :label="t('es.standardSearch')" name="standard">
                <el-card>
                    <template #header>
                        <el-space>
                            <span>{{ t('es.searchParams') }}</span>
                            <el-text type="info" size="small">{{ t('es.searchParamsDesc') }}</el-text>
                        </el-space>
                    </template>
                    <el-button v-if="state.queryParams.length == 0" size="small" @click="onAddParam" type="primary" icon="plus">{{
                        t('common.add')
                    }}</el-button>
                    <div v-for="item in state.queryParams" :key="item.uuid">
                        <el-form :inline="true" :model="item">
                            <el-form-item>
                                <el-switch v-model="item.enable" active-text="on" inactive-text="off" inline-prompt />
                            </el-form-item>
                            <el-form-item>
                                <el-select v-model="item.type">
                                    <el-option v-for="p in paramTypes" :key="p" :label="p" :value="p" />
                                </el-select>
                            </el-form-item>
                            <el-form-item>
                                <el-select filterable v-model="item.field" class="field-select">
                                    <el-option v-for="f in fields" :key="f" :label="f" :value="f" />
                                </el-select>
                            </el-form-item>
                            <el-form-item>
                                <el-select filterable v-model="item.matchType">
                                    <el-option v-for="d in matchTypes" :key="d" :label="d" :value="d" />
                                </el-select>
                            </el-form-item>
                            <el-form-item v-if="item.matchType !== 'range'">
                                <el-input
                                    v-model.trim="item.value"
                                    :placeholder="item.matchType === 'terms' || item.type === 'should' ? t('common.MultiPlaceholder') : ''"
                                />
                            </el-form-item>
                            <el-form-item>
                                <el-button link @click="onAddParam" type="primary" icon="plus" />
                            </el-form-item>
                            <el-form-item>
                                <el-button link @click="onCopyParam(item)" type="primary" icon="CopyDocument" />
                            </el-form-item>
                            <el-form-item>
                                <el-button link @click="onDelParam(item.uuid)" type="danger" icon="delete" />
                            </el-form-item>
                            <div v-if="item.matchType === 'range'">
                                <el-form-item>
                                    <el-select v-model="item.gtType" class="es-range-select">
                                        <el-option value="gt">gt ></el-option>
                                        <el-option value="gte">gte >=</el-option>
                                    </el-select>
                                </el-form-item>
                                <el-form-item>
                                    <el-input class="es-range-input" v-model.trim="item.gtValue" placeholder="> or >=" />
                                </el-form-item>
                                <el-form-item>
                                    <el-select v-model="item.ltType" class="es-range-select">
                                        <el-option value="lt">lt <</el-option>
                                        <el-option value="lte">lte <=</el-option>
                                    </el-select>
                                </el-form-item>
                                <el-form-item>
                                    <el-input class="es-range-input" v-model.trim="item.ltValue" placeholder="< or <=" />
                                </el-form-item>
                            </div>
                        </el-form>
                    </div>
                </el-card>

                <el-card :header="t('es.sortParams')">
                    <el-button v-if="state.sortParams.length == 0" size="small" @click="onAddSort" type="primary" icon="plus">{{ t('common.add') }}</el-button>

                    <div v-for="item in state.sortParams" :key="item.uuid">
                        <el-form :inline="true" :model="item">
                            <el-form-item>
                                <el-switch v-model="item.enable" active-text="on" inactive-text="off" inline-prompt></el-switch>
                            </el-form-item>
                            <el-form-item>
                                <el-select filterable v-model="item.field" class="field-select">
                                    <el-option v-for="f in fields" :key="f" :label="f" :value="f" />
                                </el-select>
                            </el-form-item>
                            <el-form-item>
                                <el-select filterable v-model="item.order">
                                    <el-option v-for="t in orderTypes" :key="t" :label="t" :value="t" />
                                </el-select>
                            </el-form-item>
                            <el-form-item>
                                <el-button link @click="onAddSort" type="primary" icon="plus" />
                            </el-form-item>
                            <el-form-item>
                                <el-button link @click="onDelSort(item.uuid)" type="danger" icon="delete" />
                            </el-form-item>
                        </el-form>
                    </div>
                </el-card>

                <el-card :header="t('es.otherParams')">
                    <el-form label-width="200px" label-position="left">
                        <el-form-item label="track_total_hits">
                            <el-checkbox v-model="state.track_total_hits" />
                        </el-form-item>
                        <el-form-item label="minimum_should_match">
                            <el-input-number size="small" v-model="state.minimum_should_match" :min="1" :max="10" />
                        </el-form-item>
                    </el-form>
                </el-card>
            </el-tab-pane>
            <el-tab-pane :label="t('es.AggregationSearch')" name="aggs"> developing... </el-tab-pane>
            <el-tab-pane :label="t('es.SqlSearch')" name="sql"> developing... </el-tab-pane>
        </el-tabs>

        <template #footer>
            <div>
                <el-button size="small" @click="onClearParam" icon="refresh">{{ t('common.reset') }}</el-button>
                <!-- <el-button size="small" @click="onSaveParam" type="primary" icon="check">{{ t('common.save') }}</el-button>-->

                <el-button size="small" @click="visible = false" icon="close">{{ t('common.close') }}</el-button>
                <el-button size="small" @click="onSearch" type="primary" icon="search">{{ t('common.search') }}</el-button>
            </div>
        </template>
    </el-dialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { reactive, ref, watch } from 'vue';
import { randomUuid } from '@/common/utils/string';
import MonacoEditorBox from '@/components/monaco/MonacoEditorBox';
import { esApi } from '@/views/ops/es/api';

const { t } = useI18n();

/**
 *
 * 类型      是否参与评分   必须满足   说明
 * must     ✅ 是         ✅ 是      所有条件都必须满足，类似 SQL 的 AND
 * should   ✅ 是（默认）   ❌ 否     至少满足一个，类似 SQL 的 OR，可通过 minimum_should_match 控制
 * must_not ❌ 否         ✅ 是      所有条件都不满足，不参与评分
 *
 *
 * 匹配方式        适用类型   特点                            示例
 * match          text     对文本字段进行分词后匹配。         "match": { "content": "elasticsearch search"  }  匹配包含 "elasticsearch" 或 "search" 的文档。
 * match_phrase   text     短语匹配，要求关键词按顺序连续出现。 "match_phrase": { "content": "今天天气不错"  }    只有当该短语作为一个整体出现在内容中时才匹配。
 * term           keyword、boolean、number 等不分词字段  对字段进行精确匹配，不进行分词。   "term": { "status": "published" 匹配 status 字段等于 "published" 的文档。
 * terms        用于匹配多个值中的任意一个（类似 SQL 中的 IN）  "terms": { "category": ["tech", "science", "ai"]  }
 * exists       判断某个字段是否存在。
 * wildcard     支持通配符匹配（* 匹配任意字符序列，? 匹配单个字符）。
 * range        用于匹配数值或日期范围   "range": { "age": { "gte": 18,  "lte": 30 } }
 *
 * 使用建议
 * 对于需要全文检索的字段，使用 text 类型 + match。
 * 对于精确匹配（如 ID、状态码、标签等），使用 keyword 类型 + term。
 * 如果要提高性能，可以将不关心相关度的条件放在 bool.filter 中。
 * 尽量避免在大数据集中频繁使用 wildcard，它会显著影响性能。
 *
 * 查询示例
 * {
 *   "query": {
 *     "bool": {
 *       "must": [
 *         { "match": { "title": "搜索测试" }}
 *       ],
 *       "should": [
 *         { "term": { "category": "技术" }},
 *         { "match_phrase": { "content": "性能优化" }}
 *       ],
 *       "must_not": [
 *         { "term": { "status": "草稿" }}
 *       ],
 *       "minimum_should_match": 1
 *     }
 *   },
 *   "sort": { "etlTime": { "order": "desc" } },
 *   "aggs": {}
 *   "from": 0,
 *   "size": 25
 * }
 *
 * 聚合查询（Aggregations）
 * 是一种强大的数据分析功能，用于对数据进行分类、统计、分析和分组。它类似于 SQL 中的 GROUP BY 和 COUNT()、SUM() 等操作，但更加强大灵活。
 * 一、聚合的基本结构
 *
 * {
 *   "size": 0,
 *   "aggs": {
 *     "自定义聚合名称": {
 *       "聚合类型": {
 *         // 聚合参数
 *       }
 *     }
 *   }
 * }
 * - "size": 0：表示不返回具体的文档内容，只返回聚合结果。
 * - "aggs"：是聚合的入口，你可以在这里定义多个聚合项。
 *
 * 二、常见的聚合类型
 * 1. 指标聚合（Metric Aggregations）
 * 用于计算字段的统计指标
 *
 * 聚合类型           描述
 * avg            计算某个字段的平均值
 * sum            计算字段总和
 * min / max      获取最小值或最大值
 * value_count    统计非空值的数量
 * cardinality    去重统计（类似 SQL 的 COUNT(DISTINCT)）
 *
 * 示例：
 *
 * "aggs": {
 *   "avg_salary": { "avg": { "field": "salary" } },
 *   "unique_users": { "cardinality": { "field": "user_id.keyword" } }
 * }
 *
 * 2. 桶聚合（Bucket Aggregations）
 * 用于将文档分组（类似 SQL 的 GROUP BY），每个桶是一个分组。
 * (1) terms：按字段值分组统计
 * "aggs": {
 *   "group_by_status": {
 *     "terms": { "field": "status.keyword" }
 *   }
 * }
 * 按 status 字段的不同值进行分组，并统计每组数量。
 * (2) date_histogram：按时间间隔分组
 * "aggs": {
 *   "articles_over_time": {
 *     "date_histogram": {
 *       "field": "publish_date",
 *       "calendar_interval": "day"
 *     }
 *   }
 * }
 * 按天统计文章发布数量。
 *
 * (3) range / date_range：按数值/日期范围分组
 *
 * "aggs": {
 *   "age_distribution": {
 *     "range": {
 *       "field": "age",
 *       "ranges": [
 *         { "from": 0, "to": 18 },
 *         { "from": 18, "to": 35 },
 *         { "from": 35, "to": 60 }
 *       ]
 *     }
 *   }
 * }
 * 按年龄段区间统计人数。
 * (4) histogram：按固定数值步长分组
 * "aggs": {
 *   "price_distribution": {
 *     "histogram": {
 *       "field": "price",
 *       "interval": 100
 *     }
 *   }
 * }
 * 每 100 元为一个价格区间进行分组统计。
 *
 * 3. 嵌套聚合（组合使用）
 * 你可以在一个聚合中嵌套其他聚合，实现多维分析。
 * 示例：先按状态分组，再按平均工资排序
 * "aggs": {
 *   "group_by_status": {
 *     "terms": {
 *       "field": "status.keyword",
 *       "order": { "avg_salary": "desc" }
 *     },
 *     "aggs": {
 *       "avg_salary": { "avg": { "field": "salary" } }
 *     }
 *   }
 * }
 *
 *
 *
 *
 *
 * es 数据类型
 *
 * 一、基础数据类型
 * 类型   描述
 * text  用于全文本搜索，会被分析器分词处理。适用于长文本内容。
 * keyword  不分词，作为完整字符串存储和匹配，适用于精确查询、聚合、排序等。
 * long  64位整数。
 * integer  32位整数。
 * short  16位整数。
 * byte  8位整数。
 * double  双精度浮点数。
 * float  单精度浮点数。
 * half_float  半精度浮点数（占用更少空间）。
 * scaled_float  以长整型形式存储浮点数（如：1.99 存为 199，缩放因子为 100）。
 * date  日期类型，可接受格式化字符串或时间戳。
 * boolean  布尔值，true 或 false。
 * binary  Base64 编码的二进制数据（不存储，仅用于传输）。
 *
 * 二、复杂数据类型
 * 类型  描述
 * object  默认嵌套 JSON 对象类型，适合嵌套结构但不支持嵌套查询。
 * nested  特殊的 object 类型，支持嵌套查询（需使用 nested query）。
 * flattened  将整个对象视为单个字段，适用于动态结构但只支持精确匹配。
 * join  用于父子文档关系（Parent-Child），实现文档间逻辑关联。
 * percolator  用于预注册查询，然后对新文档进行匹配测试。
 *
 * 三、地理空间数据类型
 * 类型  描述
 * geo_point  表示经纬度坐标，可用于距离查询、地理围栏等。
 * geo_shape  支持复杂的地理形状（如多边形、线段等），用于高级地理查询。
 *
 * 四、特殊用途数据类型
 * 类型  描述
 * ip   用于 IPv4/IPv6 地址，支持范围查询。
 * token_count  统计某个 text 字段分词后的词项数量。
 * murmur3  自动计算字段的哈希值（需显式开启）。
 * attachment  用于解析 Base64 编码的文件（如 PDF、Word 等）。
 * search_as_you_type  优化自动补全搜索体验的字段类型。
 * rank_feature / rank_features  用于基于机器学习的相关性评分优化。
 *
 * 五、数组类型
 * 在 ES 中，没有单独的数组类型。任何字段都可以包含多个值，只要它们的类型一致。
 *
 * 六、字段映射示例
 *
 * {
 *   "mappings": {
 *     "properties": {
 *       "title": { "type": "text" },
 *       "status": { "type": "keyword" },
 *       "views": { "type": "integer" },
 *       "created_at": { "type": "date" },
 *       "location": { "type": "geo_point" },
 *       "tags": { "type": "keyword" },
 *       "user": {
 *         "type": "nested",
 *         "properties": {
 *           "name": { "type": "text" },
 *           "email": { "type": "keyword" }
 *         }
 *       }
 *     }
 *   }
 * }
 *
 */

const defaultSearch = {
    sort: {} as any, // etlTime: { order: 'desc' }
    query: { bool: { must: [], should: [], must_not: [] } },
    aggs: {},
} as any;

interface Props {
    instId: string;
    idxName: string;
}

const fields = ref<string[]>();

const props = defineProps<Props>();

const emit = defineEmits(['search']);

const visible = defineModel<boolean>('visible');
watch(visible, async (v) => {
    if (v) {
        // 通过mapping获取所有字段信息
        if (fields.value?.length) {
            return;
        }
        let mp = await esApi.proxyReq('get', props.instId, `/${props.idxName}/_mappings`);
        let properties = mp[props.idxName].mappings.properties;
        let data = ['_id'];
        for (let key in properties) {
            data.push(key);
            let item = properties[key];
            if (item.fields) {
                for (let f in item.fields) {
                    data.push(`${key}.${f}`);
                }
            }
        }
        fields.value = data;
    }
});

const paramTypes = ['must', 'should', 'must_not'] as const;
const matchTypes = ['match', 'match_phrase', 'term', 'terms', 'exists', 'wildcard', 'range'] as const;
const orderTypes = ['asc', 'desc'] as const;
const gtTypes = ['gt', 'gte'] as const;
const ltTypes = ['lt', 'lte'] as const;

type searchParam = {
    uuid: string; // 唯一id
    enable: boolean; // 是否启用，启用后才应用到搜索
    type: (typeof paramTypes)[number];
    field: string;
    matchType: (typeof matchTypes)[number];
    value: any;
    gtType: (typeof gtTypes)[number];
    gtValue: string;
    ltType: (typeof ltTypes)[number];
    ltValue: string;
};

type sortParam = {
    uuid: string; // 唯一id
    enable: boolean; // 是否启用，启用后才应用到搜索
    field: string;
    order: (typeof orderTypes)[number];
};

const state = reactive({
    searchBoxWidth: '720px',
    queryParams: [] as searchParam[],
    sortParams: [] as sortParam[],
    search: defaultSearch,
    minimum_should_match: 1,
    track_total_hits: false,

    activeName: 'standard',
});

const onAddParam = () => {
    state.queryParams.push({
        uuid: randomUuid(),
        enable: true,
        type: 'must',
        field: '',
        matchType: 'term',
        value: '',
        gtType: 'gt',
        gtValue: '',
        ltType: 'lt',
        ltValue: '',
    });
};

const onCopyParam = (item: searchParam) => {
    let newItem = JSON.parse(JSON.stringify(item));
    newItem.uuid = randomUuid();
    state.queryParams.push(newItem);
};

const onDelParam = (uuid: string) => {
    state.queryParams = state.queryParams.filter((item) => item.uuid !== uuid);
};
const onAddSort = () => {
    state.sortParams.push({
        uuid: randomUuid(),
        enable: true,
        field: '',
        order: 'asc',
    });
};

const onDelSort = (uuid: string) => {
    state.sortParams = state.sortParams.filter((item) => item.uuid !== uuid);
};

const onClearParam = () => {
    // 清空查询条件
    state.queryParams = [];
    state.sortParams = [];
};
const onSaveParam = () => {
    // 保存查询条件
};

const onSearch = () => {
    parseParams();
    MonacoEditorBox({
        content: JSON.stringify(state.search, null, 2),
        title: t('es.searchParamsPreview'),
        language: 'json',
        width: state.searchBoxWidth,
        canChangeLang: false,
        options: { wordWrap: 'on', tabSize: 2, readOnly: false }, // 自动换行
        confirmFn: (val: string) => {
            emit('search', JSON.parse(val));
        },
    });
};

const parseParams = () => {
    // 组装查询条件并emit search事件
    let must = [] as any;
    let should = [] as any;
    let must_not = [] as any;
    let sort = {} as any;

    for (let item of state.queryParams) {
        if (!item.enable || !item.field || (!item.value.trim() && !item.gtValue.trim() && !item.ltValue.trim())) {
            continue;
        }
        // wildcard 自动添加通配符
        if (item.matchType === 'wildcard' && !item.value.includes('*') && !item.value.includes('?')) {
            item.value = `*${item.value}*`;
        }

        let value = item.value;
        if (item.matchType === 'terms') {
            value = item.value.split(',').map((item: string) => item.trim());
        }

        let match = {
            [item.matchType]: {
                [item.field]: value,
            },
        } as any;

        // 处理range
        if (item.matchType == 'range') {
            let gtType = item.gtType;
            let ltType = item.ltType;
            let gtValue = item.gtValue.trim();
            let ltValue = item.ltValue.trim();
            if (!gtValue && !ltValue) {
                continue;
            }
            let range = {} as any;
            if (gtValue) {
                range[gtType] = gtValue;
            }
            if (ltValue) {
                range[ltType] = ltValue;
            }

            match = {
                range: {
                    [item.field]: range,
                },
            };
        }

        switch (item.type) {
            case 'must':
                must.push(match);
                break;
            case 'should':
                should.push(match);
                break;
            case 'must_not':
                must_not.push(match);
                break;
        }
    }

    state.search.query = { bool: { must, should, must_not } };

    // 排序
    state.sortParams.forEach((item) => {
        if (item.enable && item.field) {
            sort[item.field] = { order: item.order };
        }
    });
    state.search.sort = sort;

    // track_total_hits
    if (state.track_total_hits) {
        state.search['track_total_hits'] = true;
    } else {
        delete state.search['track_total_hits'];
    }

    // minimum_should_match 需要结合should使用，默认为1，表示至少一个should条件满足
    if (should.length > 0) {
        state.search['minimum_should_match'] = Math.max(1, state.minimum_should_match);
    } else {
        delete state.search['minimum_should_match'];
    }
};
</script>

<style lang="scss" scoped>
.es-search-form-inline {
    * {
        border-radius: 3px;
    }
    .el-card {
        margin-bottom: 10px;
        .el-card__header {
            padding: 10px;
        }
    }

    .el-input {
        --el-input-width: 150px;
    }
    .es-range-input {
        --el-input-width: 240px;
    }
    .el-select {
        --el-select-width: 100px;
        font-size: var(--font-size);
    }
    .es-range-select {
        --el-select-width: 70px;
    }
    .field-select {
        --el-select-width: 150px;
    }
    .el-form {
        margin-bottom: 10px;
    }
    .el-form-item {
        margin-right: 5px;
        margin-bottom: 5px;
    }
    .el-input-number {
        width: 80px;
    }
}
</style>
