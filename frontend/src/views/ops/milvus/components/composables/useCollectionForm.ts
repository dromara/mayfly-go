/**
 * Milvus Collection 表单状态管理 composable
 * 职责：表单状态、字段操作、动态字段管理、索引操作、校验、提交、数据转换
 * 通过 provide/inject 与子组件 (SchemaFieldEditor / FieldConfigPanel) 共享状态
 */
import { Rules } from '@/common/rule';
import MonacoEditorBox from '@/components/monaco/MonacoEditorBox';
import { Msg } from '@/hooks/useI18n';
import type { FormInstance } from 'element-plus';
import { computed, inject, provide, ref, watch, type InjectionKey, type Ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { milvusApi } from '../../api';
import { DataTypeMap, fieldTypeGroups, IndexConfigByDataType, VectorTypes } from '../collection/constants';
import type { ApiCollection, ApiField, CollectionField, DynamicFieldIndex, FieldIndexParams, IndexConfig } from '../collection/types';

export interface UseCollectionFormOptions {
    milvusId: number;
    mode?: 'create' | 'edit' | 'copy';
    editData?: ApiCollection | null;
    visible: Ref<boolean>;
    onSuccess: () => void;
}

export function useCollectionForm(options: UseCollectionFormOptions) {
    const { milvusId, visible, onSuccess } = options;
    // 使用闭包捕获 props 值（mode/editData 在 drawer 打开期间不变）
    const getMode = () => options.mode;
    const getEditData = () => options.editData;

    const { t } = useI18n();

    const formRef = ref<FormInstance>();
    const loading = ref(false);
    const activeTab = ref('properties');
    const selectedFieldIndex = ref<number>(-1);
    const fieldTypePopoverVisible = ref(false);
    const selectedCategory = ref<string>('向量');
    const dynamicFieldEnabled = ref(false);

    // 动态字段
    const dynamicField = ref<{
        indexes: DynamicFieldIndex[];
        selectedIdx: number;
    }>({
        indexes: [], // 多个索引
        selectedIdx: 0, // 当前选中的索引
    });

    const form = ref({
        name: '',
        description: '',
        shardsNum: 1,
        consistency_level: 'Bounded',
        fields: [] as CollectionField[],
    });

    // 编辑模式下记录原始 collection 名称
    const originalName = ref('');

    const rules = {
        name: [Rules.requiredInput('common.name')],
    };

    // 计算属性：普通字段列表
    const normalFields = computed(() => {
        return form.value.fields || [];
    });

    // 是否选中动态字段
    const isDynamicFieldSelected = computed(() => selectedFieldIndex.value === -2);

    // 计算抽屉标题
    const drawerTitle = computed(() => {
        if (getMode() === 'edit') return t('milvus.editCollection');
        if (getMode() === 'copy') return t('milvus.copyCollection');
        return t('milvus.createCollection');
    });

    // 是否为编辑模式
    const isEditMode = computed(() => getMode() === 'edit');

    // 当前选中的字段
    const selectedField = computed(() => {
        if (selectedFieldIndex.value >= 0 && selectedFieldIndex.value < form.value.fields.length) {
            return form.value.fields[selectedFieldIndex.value];
        }
        return null;
    });

    // ---- 类型判断辅助函数 ----

    const isVectorType = (type: number) => VectorTypes.has(type);
    const isGeometryType = (type: number) => type === DataTypeMap.Geometry;
    const isClockType = (type: number) => type === DataTypeMap.Timestamptz;
    const isArrayType = (type: number) => type === DataTypeMap.Array;
    const isVarCharType = (type: number) => type === DataTypeMap.VarChar;
    const isInt64Type = (type: number) => type === DataTypeMap.Int64;
    const canBePrimaryKey = (type: number) => type === DataTypeMap.Int64 || type === DataTypeMap.VarChar;

    // 获取类型名称
    const getTypeName = (type: number): string => {
        const entry = Object.entries(fieldTypeGroups)
            .flatMap(([_, types]) => types)
            .find((t) => t.value === type);
        return entry?.label || String(type);
    };

    // 格式化字段类型显示
    const formatFieldType = (field: CollectionField) => {
        const typeName = getTypeName(field.dataType);
        if (isVectorType(field.dataType) && field.dim) {
            return `${typeName}(${field.dim})`;
        }
        if (isArrayType(field.dataType) && field.elementType) {
            return `Array<${getTypeName(field.elementType)}>`;
        }
        return typeName;
    };

    // ---- 索引相关 ----

    const getPrimaryKeyField = () => {
        return form.value.fields.find((f) => f.isPrimaryKey);
    };

    // 获取字段类型支持的索引配置
    const getFieldIndexConfig = (dataType: number): IndexConfig | null => {
        return (IndexConfigByDataType as Record<number, IndexConfig>)[dataType] || null;
    };

    // 判断字段是否支持索引
    const supportsIndex = (dataType: number) => {
        return !!getFieldIndexConfig(dataType);
    };

    // 按组组织索引类型
    const getIndexesByGroup = (dataType: number) => {
        const config = getFieldIndexConfig(dataType);
        if (!config) return {};

        const groups: Record<string, { value: string; label: string }[]> = {};
        for (const [key, index] of Object.entries(config.indexes)) {
            const group = index.group;
            if (!groups[group]) {
                groups[group] = [];
            }
            groups[group].push({ value: key, label: index.label });
        }
        return groups;
    };

    // 获取度量类型选项
    const getMetricOptions = (dataType: number) => {
        const config = getFieldIndexConfig(dataType);
        if (!config) return [];
        return config.metrics || [];
    };

    // 添加索引到字段
    const handleAddIndex = (field: CollectionField) => {
        const config = getFieldIndexConfig(field.dataType);
        if (!config) return;

        const firstIndexKey = Object.keys(config.indexes)[0];
        const firstIndex = config.indexes[firstIndexKey];

        field.indexType = firstIndexKey;
        field.metricType = config.metrics?.[0] || '';
        field.indexParams = { ...firstIndex.params };
    };

    // 删除字段索引
    const handleRemoveIndex = (field: CollectionField) => {
        field.indexType = undefined;
        field.metricType = undefined;
        field.indexParams = {};
    };

    // ---- 动态字段操作 ----

    const selectDynamicField = () => {
        selectedFieldIndex.value = -2;
        activeTab.value = 'properties';
    };

    // 动态字段开关切换
    const handleDynamicFieldToggle = (val: boolean) => {
        if (val) {
            // 启用动态字段
            if (!dynamicField.value.indexes || dynamicField.value.indexes.length === 0) {
                dynamicField.value.indexes = [];
            }
        }
    };

    // 删除动态字段
    const handleDeleteDynamicField = () => {
        dynamicFieldEnabled.value = false;
        dynamicField.value = {
            indexes: [],
            selectedIdx: 0,
        };
        selectedFieldIndex.value = -1;
    };

    // 添加动态字段索引
    const handleAddDynamicIndex = () => {
        dynamicField.value.indexes.push({
            indexName: '',
            indexType: 'AUTOINDEX',
            json_path: '$meta["key_name"]',
            json_cast_type: 'varchar',
            json_cast_function: '',
            field_name: '',
        });
        dynamicField.value.selectedIdx = dynamicField.value.indexes.length - 1;
    };

    // 删除动态字段索引
    const handleDeleteDynamicIndex = (index: number) => {
        dynamicField.value.indexes.splice(index, 1);
        if (dynamicField.value.selectedIdx >= dynamicField.value.indexes.length) {
            dynamicField.value.selectedIdx = Math.max(0, dynamicField.value.indexes.length - 1);
        }
    };

    // ---- 字段操作 ----

    // 选择字段
    const selectField = (index: number) => {
        selectedFieldIndex.value = index;
        activeTab.value = 'properties';
    };

    // 添加字段
    const handleAddField = (dataType: number, name?: string) => {
        const isVector = isVectorType(dataType);
        if (name) {
            // 如果名字存在，则添加后缀_1
            const existField = form.value.fields.find((f) => f.name === name);
            if (existField) {
                name = `${name}_${form.value.fields.filter((f) => f.name === name).length}`;
            }
        }

        if (!dataType) return;
        const fieldIndex = form.value.fields.length + 1;
        const newField = {
            name: name || `field${fieldIndex}`,
            dataType: dataType,
            isPrimaryKey: false,
            autoID: false,
            description: '',
            dim: isVector ? 768 : undefined,
            elementType: isArrayType(dataType) ? DataTypeMap.Int64 : undefined,
            isDynamic: false,
            isPartitionKey: false,
            isClusteringKey: false,
            nullable: false,
            mmap: false,
            defaultValue: '',
            maxLength: isVarCharType(dataType) ? 65535 : undefined,
            maxCapacity: isArrayType(dataType) ? 1024 : undefined,
            // 默认不添加索引
            indexType: undefined,
            metricType: undefined,
            indexParams: {},
        };

        // 如果是第一个字段且是 Int64，自动设为主键和自增
        if (form.value.fields.length === 0 && isInt64Type(dataType)) {
            newField.name = 'id';
            newField.isPrimaryKey = true;
            newField.autoID = true;
        }

        form.value.fields.push(newField);
        selectedFieldIndex.value = form.value.fields.length - 1;
        activeTab.value = 'properties';
    };

    // 删除字段
    const handleDeleteField = (index: number) => {
        const field = form.value.fields[index];
        if (field.isPrimaryKey && field.autoID) {
            Msg.warning('milvus.cannotDeletePrimaryKey');
            return;
        }
        form.value.fields.splice(index, 1);
        if (selectedFieldIndex.value >= form.value.fields.length) {
            selectedFieldIndex.value = form.value.fields.length - 1;
        }
    };

    const handleCopyField = (index: number) => {
        const field = form.value.fields[index];

        // 匹配字段名的基础部分和序号(如 field5_1 -> baseName="field5_", num=1)
        const match = field.name.match(/^(.+?)(\d+)$/);

        let baseName: string;
        let startNum: number;

        if (match) {
            baseName = match[1]; // 如 "field5_" 或 "field"
            startNum = parseInt(match[2]); // 如 1 或 2
        } else {
            baseName = field.name + '_';
            startNum = 0;
        }

        // 在所有字段中查找相同基础名称的最大序号
        let maxNum = startNum;
        const escapedBaseName = baseName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        const regex = new RegExp(`^${escapedBaseName}(\\d+)$`);

        form.value.fields.forEach((f) => {
            const m = f.name.match(regex);
            if (m) {
                const num = parseInt(m[1]);
                if (num > maxNum) {
                    maxNum = num;
                }
            }
        });

        // 使用最大序号+1作为新字段名
        const newName = baseName + (maxNum + 1);
        handleAddField(field.dataType, newName);
    };

    // 选择字段类型
    const handleSelectFieldType = (dataType: number) => {
        handleAddField(dataType);
        fieldTypePopoverVisible.value = false;
    };

    // ---- 表单提交 ----

    const handleClose = () => {
        visible.value = false;
    };

    const getFormData = async () => {
        await formRef.value?.validate();
        if (form.value.fields.length === 0) {
            Msg.warning('milvus.addFieldRequired');
            return '';
        }

        // 检查是否有主键
        if (!getPrimaryKeyField()) {
            Msg.warning('milvus.primaryKeyRequired');
            return '';
        }

        // 检查所有向量字段是否都有索引
        const vectorFieldsWithoutIndex = form.value.fields.filter((field) => isVectorType(field.dataType) && !field.indexType);
        if (vectorFieldsWithoutIndex.length > 0) {
            const fieldNames = vectorFieldsWithoutIndex.map((f) => f.name).join('、');
            Msg.warning('milvus.vectorFieldIndexRequired', { fields: fieldNames });
            return '';
        }

        // 构建提交数据
        const fieldsData = form.value.fields.map((field) => {
            const fieldData: Record<string, unknown> = {
                name: field.name,
                dataType: field.dataType,
                isPrimaryKey: field.isPrimaryKey,
                autoID: field.autoID,
                description: field.description,
                isDynamic: field.isDynamic,
                isPartitionKey: field.isPartitionKey,
                isClusteringKey: field.isClusteringKey,
                nullable: field.nullable,
                mmap: field.mmap,
            };

            // 向量类型添加 dim
            if (isVectorType(field.dataType) && field.dim) {
                fieldData.dim = field.dim;
            }

            // VarChar 类型添加 maxLength
            if (isVarCharType(field.dataType) && field.maxLength) {
                fieldData.maxLength = field.maxLength;
            }

            // Array 类型添加 elementType 和 maxCapacity
            if (isArrayType(field.dataType)) {
                fieldData.elementType = field.elementType;
                fieldData.maxCapacity = field.maxCapacity;
            }

            // 如果设置了 defaultValue
            if (field.defaultValue) {
                fieldData.defaultValue = field.defaultValue;
            }

            // 如果有索引配置
            if (field.indexType && supportsIndex(field.dataType)) {
                fieldData.indexParams = {
                    index_type: field.indexType,
                    ...(field.metricType && { metric_type: field.metricType }),
                    ...field.indexParams,
                };
            }

            // 清空空值
            let obj: Record<string, unknown> = {};
            for (const prop in fieldData) {
                if (fieldData[prop] !== undefined && fieldData[prop] !== '' && fieldData[prop] !== null) {
                    obj[prop] = fieldData[prop];
                }
            }
            return obj;
        });

        // 添加动态字段
        if (dynamicFieldEnabled.value && dynamicField.value.indexes && dynamicField.value.indexes.length > 0) {
            const dynamicFieldData: Record<string, unknown> = {
                name: '$meta',
                dataType: DataTypeMap.JSON,
                isDynamic: true,
                indexes: dynamicField.value.indexes.map((idx) => {
                    const idxData: Record<string, unknown> = {
                        index_name: idx.indexName || '',
                        index_type: idx.indexType,
                    };
                    // 动态字段索引特有参数
                    if (idx.json_path) idxData.json_path = idx.json_path;
                    if (idx.json_cast_type) idxData.json_cast_type = idx.json_cast_type;
                    if (idx.json_cast_function) idxData.json_cast_function = idx.json_cast_function;
                    if (idx.field_name) idxData.field_name = idx.field_name;
                    // 清理空值
                    const cleanIdx: Record<string, unknown> = {};
                    for (const prop in idxData) {
                        if (idxData[prop] !== undefined && idxData[prop] !== '' && idxData[prop] !== null) {
                            cleanIdx[prop] = idxData[prop];
                        }
                    }
                    return cleanIdx;
                }),
            };
            fieldsData.push(dynamicFieldData);
        }

        const submitData = {
            name: form.value.name,
            description: form.value.description,
            shardsNum: form.value.shardsNum,
            consistency_level: form.value.consistency_level,
            fields: fieldsData,
        };

        return submitData;
    };

    // 提交表单
    const handleSubmit = async () => {
        try {
            const submitData = await getFormData();
            if (!submitData) {
                return;
            }
            loading.value = true;

            if (getMode() === 'edit') {
                // 编辑模式：调用修改接口，提交所有可修改的字段
                const alterData: Record<string, unknown> = {};

                // 1. 重命名：如果名称发生变化
                if (form.value.name !== originalName.value) {
                    alterData.newName = form.value.name;
                }

                // 2. 描述
                if (submitData.description !== undefined) {
                    alterData.description = submitData.description;
                }

                // 3. 一致性级别
                if (submitData.consistency_level !== undefined) {
                    // 将字符串一致性级别转换为数字
                    const consistencyMap: Record<string, number> = {
                        Strong: 0,
                        Bounded: 1,
                        Session: 2,
                        Eventually: 3,
                    };
                    alterData.consistency_level = consistencyMap[submitData.consistency_level] ?? 1;
                }

                // 4. 新增字段：只提交非只读（新添加）的字段
                const newFields = form.value.fields.filter((f) => !f.readonly);
                if (newFields.length > 0) {
                    alterData.fields = newFields.map((f) => ({
                        name: f.name,
                        data_type: f.dataType,
                        is_primary_key: f.isPrimaryKey || false,
                        auto_id: f.autoID || false,
                        description: f.description || '',
                        dim: f.dim,
                        nullable: f.nullable || false,
                        element_type: f.elementType,
                        is_dynamic: f.isDynamic || false,
                        is_partition_key: f.isPartitionKey || false,
                        is_clustering_key: f.isClusteringKey || false,
                        max_length: f.maxLength,
                        max_capacity: f.maxCapacity,
                        type_params: f.typeParams || {},
                        index_params: f.indexParams || {},
                    }));
                }

                await milvusApi.alterCollection(milvusId, originalName.value, alterData);
                Msg.success('milvus.updatedSuccess');
            } else {
                // 创建/复制模式：调用创建接口
                await milvusApi.createCollection(milvusId, submitData);
                Msg.success('milvus.createdSuccess');
            }
            visible.value = false;
            onSuccess();
        } finally {
            loading.value = false;
        }
    };

    const handlePreview = async () => {
        const submitData = await getFormData();
        if (!submitData) {
            return;
        }
        MonacoEditorBox({
            content: JSON.stringify(submitData, null, 2),
            title: 'JSON预览',
            language: 'json',
            canChangeLang: false,
            useDrawer: true,
            options: { wordWrap: 'on', tabSize: 2, readOnly: true }, // 自动换行
        });
    };

    // ---- 数据转换与初始化 ----

    // 转换 API 返回的字段数据为表单格式
    const transformApiFieldToForm = (f: ApiField): CollectionField => {
        const dim = f.TypeParams?.dim ? parseInt(f.TypeParams.dim) : undefined;
        const maxLength = f.TypeParams?.max_length ? parseInt(f.TypeParams.max_length) : undefined;

        // 解析索引信息
        let indexType: string | undefined = undefined;
        let metricType: string | undefined = undefined;
        let indexParams: FieldIndexParams | undefined = undefined;

        // 优先从 IndexParams 中解析（后端 DescribeCollection 返回的结构）
        if (f.IndexParams && Object.keys(f.IndexParams).length > 0) {
            const ip = f.IndexParams;
            indexType = ip.index_type || ip.indexType;
            metricType = ip.metric_type || ip.metricType;

            // 解析 params 字段（可能是 JSON 字符串）
            let paramsObj: Record<string, unknown> = {};
            if (ip.params) {
                try {
                    paramsObj = typeof ip.params === 'string' ? JSON.parse(ip.params) : ip.params;
                } catch (e) {
                    console.warn('Failed to parse index params:', e);
                }
            }

            // 提取其他参数（如 mmap.enabled, nlist 等）
            for (const key in ip) {
                if (['index_type', 'indexType', 'metric_type', 'metricType', 'params'].includes(key)) continue;

                const value = ip[key];
                // 尝试转换为数字
                const numValue = Number(value);
                paramsObj[key] = isNaN(numValue) ? value : numValue;
            }

            if (Object.keys(paramsObj).length > 0) {
                indexParams = paramsObj as FieldIndexParams;
            }
        } else if (f.Indexes && f.Indexes.length > 0) {
            // 兼容旧格式：从 Indexes 数组中获取
            const index = f.Indexes[0];
            indexType = index.IndexType || index.index_type;
            metricType = index.MetricType || index.metric_type;

            const params = index.Params || index.indexParams || {};
            indexParams = {};
            for (const key in params) {
                const value = params[key];
                const numValue = Number(value);
                indexParams[key] = isNaN(numValue) ? value : numValue;
            }
        }

        return {
            name: f.Name || '',
            dataType: f.DataType ?? 0,
            isPrimaryKey: f.PrimaryKey || false,
            autoID: f.AutoID || false,
            description: f.Description || '',
            isDynamic: f.IsDynamic || false,
            isPartitionKey: f.IsPartitionKey || false,
            isClusteringKey: f.IsClusteringKey || false,
            nullable: f.Nullable || false,
            mmap: false,
            defaultValue: f.DefaultValue || '',
            dim: dim,
            maxLength: maxLength,
            elementType: f.ElementType,
            maxCapacity: undefined,
            readonly: false,
            // 索引信息
            indexType: indexType,
            metricType: metricType,
            indexParams: indexParams ?? {},
        };
    };

    // 重置表单
    const resetForm = () => {
        form.value = {
            name: '',
            description: '',
            shardsNum: 1,
            consistency_level: 'Bounded',
            fields: [],
        };
        selectedFieldIndex.value = -1;
        activeTab.value = 'properties';
        dynamicFieldEnabled.value = false;
        dynamicField.value = {
            indexes: [],
            selectedIdx: 0,
        };
        if (formRef.value) {
            formRef.value?.clearValidate();
        }
    };

    // 监听 visible 变化，初始化表单
    watch(visible, (newVal) => {
        if (newVal) {
            resetForm();
            const editData = getEditData();

            if (getMode() === 'edit' && editData) {
                // 编辑模式：填充现有数据
                const schema: NonNullable<ApiCollection['Schema']> = editData.Schema || editData.schema || {};
                // 记录原始名称
                originalName.value = editData.Name || editData.name || '';
                form.value = {
                    name: editData.Name || editData.name || '',
                    description: schema.Description || editData.description || '',
                    shardsNum: editData.ShardNum || 1,
                    consistency_level: ['Bounded', 'Strong', 'Session', 'Eventually'][editData.ConsistencyLevel ?? 0] || 'Bounded',
                    fields: (schema.Fields || []).map((f) => ({
                        ...transformApiFieldToForm(f),
                        readonly: true, // 编辑模式下现有字段只读
                    })),
                };
                // 处理动态字段
                if (schema.EnableDynamicField) {
                    dynamicFieldEnabled.value = true;
                }
            } else if (getMode() === 'copy' && editData) {
                // 复制模式：填充数据，表名加 _copy 后缀
                const schema: NonNullable<ApiCollection['Schema']> = editData.Schema || editData.schema || {};
                const srcName = editData.Name || editData.name || '';
                const copyName = srcName ? `${srcName}_copy` : '';
                form.value = {
                    name: copyName,
                    description: schema.Description || editData.description || '',
                    shardsNum: editData.ShardNum || 1,
                    consistency_level: ['Bounded', 'Strong', 'Session', 'Eventually'][editData.ConsistencyLevel ?? 0] || 'Bounded',
                    fields: (schema.Fields || []).map((f) => transformApiFieldToForm(f)),
                };
                // 处理动态字段
                if (schema.EnableDynamicField) {
                    dynamicFieldEnabled.value = true;
                }
            } else {
                // 新增模式：默认添加一个主键字段和一个向量字段
                handleAddField(DataTypeMap.Int64);
                handleAddField(DataTypeMap.FloatVector, 'vector');
            }
        }
    });

    return {
        // state
        formRef,
        loading,
        activeTab,
        selectedFieldIndex,
        fieldTypePopoverVisible,
        selectedCategory,
        dynamicFieldEnabled,
        dynamicField,
        form,
        rules,
        // computed
        normalFields,
        isDynamicFieldSelected,
        drawerTitle,
        isEditMode,
        selectedField,
        // type helpers
        isVectorType,
        isGeometryType,
        isClockType,
        isArrayType,
        isVarCharType,
        isInt64Type,
        canBePrimaryKey,
        formatFieldType,
        // index helpers
        getPrimaryKeyField,
        supportsIndex,
        getIndexesByGroup,
        getMetricOptions,
        handleAddIndex,
        handleRemoveIndex,
        // dynamic field ops
        selectDynamicField,
        handleDynamicFieldToggle,
        handleDeleteDynamicField,
        handleAddDynamicIndex,
        handleDeleteDynamicIndex,
        // field ops
        selectField,
        handleAddField,
        handleDeleteField,
        handleCopyField,
        handleSelectFieldType,
        // form lifecycle
        handleClose,
        handleSubmit,
        handlePreview,
    };
}

export type CollectionFormContext = ReturnType<typeof useCollectionForm>;

export const CollectionFormKey: InjectionKey<CollectionFormContext> = Symbol('CollectionFormContext');

export const provideCollectionForm = (ctx: CollectionFormContext) => provide(CollectionFormKey, ctx);

export const useCollectionFormInject = (): CollectionFormContext => {
    const ctx = inject(CollectionFormKey);
    if (!ctx) throw new Error('useCollectionFormInject must be used within CollectionsCreate');
    return ctx;
};
