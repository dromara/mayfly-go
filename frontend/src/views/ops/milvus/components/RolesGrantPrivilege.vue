<template>
    <!-- 授权弹窗 -->
    <el-dialog v-model="privilegeDialog.visible" :title="$t('milvus.privilegeManagement')" width="900px">
        <el-form :model="privilegeForm" label-width="120px">
            <el-form-item :label="$t('milvus.roleName')">
                <el-input v-model="privilegeDialog.roleName" disabled></el-input>
            </el-form-item>

            <!-- 授权方式切换 -->
            <el-form-item :label="$t('milvus.grantType')">
                <el-radio-group v-model="privilegeForm.grantType">
                    <el-radio value="privilegeGroup">
                        {{ $t('milvus.grantByPrivilegeGroup') }}
                        <el-badge v-if="privilegeGroupAuthCount > 0" :value="privilegeGroupAuthCount" class="tab-badge" />
                    </el-radio>
                    <el-radio value="privilege">
                        {{ $t('milvus.grantByPrivilege') }}
                        <el-badge v-if="privilegeAuthCount > 0" :value="privilegeAuthCount" class="tab-badge" />
                    </el-radio>
                </el-radio-group>
            </el-form-item>

            <!-- 权限组类型Tab页签 -->
            <el-form-item :label="$t('milvus.privilegeGroupType')" v-if="privilegeForm.grantType === 'privilegeGroup'">
                <el-radio-group v-model="privilegeGroupTypeTab">
                    <el-radio value="cluster">
                        {{ $t('milvus.clusterPrivilege') }}
                        <el-badge v-if="clusterAuthCount > 0" :value="clusterAuthCount" class="tab-badge" />
                    </el-radio>
                    <el-radio value="database">
                        {{ $t('milvus.databasePrivilege') }}
                        <el-badge v-if="databaseAuthCount > 0" :value="databaseAuthCount" class="tab-badge" />
                    </el-radio>
                    <el-radio value="collection">
                        {{ $t('milvus.collectionPrivilege') }}
                        <el-badge v-if="collectionAuthCount > 0" :value="collectionAuthCount" class="tab-badge" />
                    </el-radio>
                    <el-radio value="custom">
                        {{ $t('milvus.customPrivilegeGroup') }}
                        <el-badge v-if="customAuthCount > 0" :value="customAuthCount" class="tab-badge" />
                    </el-radio>
                </el-radio-group>
            </el-form-item>

            <!-- 数据库列表选择 - 仅数据库权限且为权限组授权模式显示 -->
            <el-form-item
                :label="$t('milvus.databaseScope')"
                v-if="privilegeGroupTypeTab === 'database' && privilegeForm.grantType === 'privilegeGroup'"
                style="max-height: 400px"
            >
                <el-row :gutter="16" style="width: 100%">
                    <el-col :span="24">
                        <el-card shadow="never" :body-style="{ padding: '0' }">
                            <el-alert :closable="false" type="primary">{{ $t('milvus.databaseManagement') }}</el-alert>
                            <div style="padding: 10px">
                                <el-input v-model="dbFilterText" :placeholder="$t('common.search')" clearable size="small" style="margin-bottom: 10px" />
                                <div style="max-height: 300px; overflow-y: auto">
                                    <!-- 所有数据库选项 -->
                                    <div
                                        @click="selectDatabase('*')"
                                        style="padding: 2px 12px; cursor: pointer; border-radius: 4px; margin-bottom: 4px"
                                        :style="{
                                            backgroundColor: privilegeForm.selectedDatabase === '*' ? 'var(--el-color-primary-light-9)' : 'transparent',
                                        }"
                                    >
                                        <div style="display: flex; align-items: center; justify-content: space-between">
                                            <span
                                                ><strong>{{ $t('milvus.allDatabases') }} (*)</strong></span
                                            >
                                            <el-tag v-if="isDatabaseAuthorized('*')" size="small" type="success">
                                                {{ $t('milvus.authorized') }}
                                            </el-tag>
                                        </div>
                                    </div>
                                    <!-- 数据库列表 -->
                                    <div
                                        v-for="db in filteredDatabases"
                                        :key="db.name"
                                        @click="selectDatabase(db.name)"
                                        style="padding: 2px 12px; cursor: pointer; border-radius: 4px; margin-bottom: 4px"
                                        :style="{
                                            backgroundColor: privilegeForm.selectedDatabase === db.name ? 'var(--el-color-primary-light-9)' : 'transparent',
                                        }"
                                    >
                                        <div style="display: flex; align-items: center; justify-content: space-between">
                                            <span>{{ db.name }}</span>
                                            <el-tag v-if="isDatabaseAuthorized(db.name)" size="small" type="success">
                                                {{ $t('milvus.authorized') }}
                                            </el-tag>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </el-card>
                    </el-col>
                </el-row>
            </el-form-item>

            <!-- 数据库和Collection左右列表选择 - Collection权限、自定义权限组、或具体权限授权模式显示 -->
            <el-form-item
                :label="$t('milvus.databaseScope')"
                v-if="
                    ((privilegeGroupTypeTab === 'collection' || privilegeGroupTypeTab === 'custom') && privilegeForm.grantType === 'privilegeGroup') ||
                    privilegeForm.grantType === 'privilege'
                "
                style="max-height: 400px"
            >
                <el-row :gutter="16" style="width: 100%">
                    <!-- 左侧：数据库列表 -->
                    <el-col :span="12">
                        <el-card shadow="never" :body-style="{ padding: '0' }">
                            <el-alert :closable="false" type="primary">{{ $t('milvus.databaseManagement') }}</el-alert>
                            <div style="padding: 10px">
                                <el-input v-model="dbFilterText" :placeholder="$t('common.search')" clearable size="small" style="margin-bottom: 10px" />
                                <div style="max-height: 300px; overflow-y: auto">
                                    <!-- 所有数据库选项 -->
                                    <div
                                        @click="selectDatabase('*')"
                                        style="padding: 2px 12px; cursor: pointer; border-radius: 4px; margin-bottom: 4px"
                                        :style="{
                                            backgroundColor: privilegeForm.selectedDatabase === '*' ? 'var(--el-color-primary-light-9)' : 'transparent',
                                        }"
                                    >
                                        <div style="display: flex; align-items: center; justify-content: space-between">
                                            <span
                                                ><strong>{{ $t('milvus.allDatabases') }} (*)</strong></span
                                            >
                                            <el-tag v-if="isDatabaseAuthorized('*')" size="small" type="success">
                                                {{ $t('milvus.authorized') }}
                                            </el-tag>
                                        </div>
                                    </div>
                                    <!-- 数据库列表 -->
                                    <div
                                        v-for="db in filteredDatabases"
                                        :key="db.name"
                                        @click="selectDatabase(db.name)"
                                        style="padding: 2px 12px; cursor: pointer; border-radius: 4px; margin-bottom: 4px"
                                        :style="{
                                            backgroundColor: privilegeForm.selectedDatabase === db.name ? 'var(--el-color-primary-light-9)' : 'transparent',
                                        }"
                                    >
                                        <div style="display: flex; align-items: center; justify-content: space-between">
                                            <span>{{ db.name }}</span>
                                            <el-tag v-if="isDatabaseAuthorized(db.name)" size="small" type="success">
                                                {{ $t('milvus.authorized') }}
                                            </el-tag>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </el-card>
                    </el-col>

                    <!-- 右侧：Collection列表 -->
                    <el-col :span="12">
                        <el-card shadow="never" :body-style="{ padding: '0' }">
                            <el-alert :closable="false" type="primary">{{ $t('milvus.collectionManagement') }}</el-alert>

                            <div style="padding: 10px">
                                <el-input
                                    v-model="collFilterText"
                                    :placeholder="$t('common.search')"
                                    clearable
                                    size="small"
                                    style="margin-bottom: 10px"
                                    :disabled="!privilegeForm.selectedDatabase || privilegeForm.selectedDatabase === '*'"
                                />
                                <div style="max-height: 300px; overflow-y: auto">
                                    <!-- 所有 Collection 选项 -->
                                    <div
                                        @click="selectCollection('*')"
                                        style="padding: 2px 12px; cursor: pointer; border-radius: 4px; margin-bottom: 4px"
                                        :style="{
                                            backgroundColor: privilegeForm.selectedCollection === '*' ? 'var(--el-color-primary-light-9)' : 'transparent',
                                        }"
                                    >
                                        <div style="display: flex; align-items: center; justify-content: space-between">
                                            <span
                                                ><strong>{{ $t('milvus.allCollections') }} (*)</strong></span
                                            >
                                            <el-tag v-if="isCollectionAuthorized('*')" size="small" type="success">
                                                {{ $t('milvus.authorized') }}
                                            </el-tag>
                                        </div>
                                    </div>
                                    <!-- Collection 列表 -->
                                    <div
                                        v-for="coll in filteredCollections"
                                        :key="coll.name"
                                        @click="selectCollection(coll.name)"
                                        style="padding: 2px 12px; cursor: pointer; border-radius: 4px; margin-bottom: 4px"
                                        :style="{
                                            backgroundColor: privilegeForm.selectedCollection === coll.name ? 'var(--el-color-primary-light-9)' : 'transparent',
                                        }"
                                    >
                                        <div style="display: flex; align-items: center; justify-content: space-between">
                                            <span>{{ coll.name }}</span>
                                            <el-tag v-if="isCollectionAuthorized(coll.name)" size="small" type="success">
                                                {{ $t('milvus.authorized') }}
                                            </el-tag>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </el-card>
                    </el-col>
                </el-row>
            </el-form-item>

            <!-- 权限组授权 -->
            <template v-if="privilegeForm.grantType === 'privilegeGroup'">
                <el-divider content-position="left">{{ $t('milvus.privilegeGroup') }}</el-divider>
                <el-checkbox-group v-model="privilegeForm.selectedPrivilegeGroups">
                    <el-row :gutter="10">
                        <el-col :span="8" v-for="pg in filteredPrivilegeGroups" :key="pg.GroupName">
                            <el-checkbox :value="pg.GroupName">
                                <span>{{ pg.GroupName }}</span>
                            </el-checkbox>
                        </el-col>
                    </el-row>
                </el-checkbox-group>
            </template>

            <template v-if="privilegeForm.grantType === 'privilege'">
                <el-divider content-position="left">{{ $t('milvus.privilege') }}</el-divider>
                <div class="privilege-groups">
                    <div v-for="group in privilegeGroupList" :key="group.name" class="privilege-group-item">
                        <el-alert :closable="false" type="primary">
                            <el-checkbox
                                :model-value="isGroupAllSelected(group)"
                                :indeterminate="isGroupIndeterminate(group)"
                                @change="(val: boolean | string | number) => handleGroupChange(group, val)"
                            >
                                <strong>{{ group.label }}</strong>
                            </el-checkbox>
                            <el-badge v-if="getGroupAuthCount(group) > 0" :value="getGroupAuthCount(group)" class="tab-badge" />
                        </el-alert>
                        <el-checkbox-group v-model="privilegeForm.selectedPrivileges" class="group-checkboxes">
                            <el-row :gutter="10">
                                <el-col :span="8" v-for="priv in group.privileges" :key="priv">
                                    <el-checkbox :value="priv" :label="priv">
                                        <span>{{ priv }}</span>
                                    </el-checkbox>
                                </el-col>
                            </el-row>
                        </el-checkbox-group>
                    </div>
                </div>
            </template>
        </el-form>

        <template #footer>
            <el-button @click="privilegeDialog.visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="submitPrivilege" :loading="privilegeLoading">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>
</template>

<script setup lang="ts">
import { Msg } from '@/hooks/useI18n';
import { computed, ref, watch } from 'vue';
import { milvusApi } from '../api';
import type { IDatabase, ICollection } from '../types';

/** Milvus 权限组 (API 返回) */
interface MilvusPrivilegeGroup {
    GroupName: string;
    Privileges?: string[];
}

/** Milvus 权限项 (角色权限详情) */
interface MilvusPrivilegeItem {
    DbName?: string;
    ObjectName?: string;
    Privilege?: string;
}

/** 权限 scope 映射 (dbName -> collections -> collName -> privName -> 是否授权) */
interface PrivilegeScopeMap {
    collections: Record<string, Record<string, boolean>>;
}

/** 角色权限数据 */
interface RolePrivilegeData {
    privileges: MilvusPrivilegeItem[];
}

/** 权限分组UI模型 */
interface PrivilegeGroupUI {
    name: string;
    label: string;
    privileges: string[];
}

type GrantType = 'privilegeGroup' | 'privilege';

const props = defineProps<{
    milvusId: number;
}>();

const emit = defineEmits<{
    'privilege-saved': [];
}>();

// 权限组分类定义
const PRIVILEGE_GROUP_CATEGORIES = {
    CLUSTER: {
        name: 'cluster',
        label: '集群权限',
        groupNames: ['ClusterReadOnly', 'ClusterReadWrite', 'ClusterAdmin'],
    },
    DATABASE: {
        name: 'database',
        label: '数据库权限',
        groupNames: ['DatabaseReadOnly', 'DatabaseReadWrite', 'DatabaseAdmin'],
    },
    COLLECTION: {
        name: 'collection',
        label: 'Collection权限',
        groupNames: ['CollectionReadOnly', 'CollectionReadWrite', 'CollectionAdmin'],
    },
    CUSTOM: {
        name: 'custom',
        label: '自定义权限组',
        groupNames: [], // 其余所有权限组
    },
};

// Tab页签控制
const privilegeGroupTypeTab = ref('cluster'); // 当前激活的Tab

// 授权弹窗
const privilegeDialog = ref({
    visible: false,
    roleName: '',
    currentDb: '',
    currentRoleData: null as RolePrivilegeData | null,
});
const privilegeLoading = ref(false);
const privilegeForm = ref({
    grantType: 'privilegeGroup', // 'privilegeGroup' | 'privilege'
    selectedPrivilegeGroups: [] as string[],
    selectedDatabase: '*',
    selectedCollection: '*',
    selectedPrivileges: [] as string[],
});

// 存储每个 database + collection + tabType + grantType 的权限勾选状态
// key 格式: "tabType:grantType:dbName:collName"
const dbCollectionPrivileges = ref<
    Map<
        string,
        {
            grantType: 'privilegeGroup' | 'privilege';
            selectedPrivilegeGroups: string[];
            selectedPrivileges: string[];
        }
    >
>(new Map());

// 版本号：用于强制触发 computed 重新计算
const privilegeStateVersion = ref(0);

// 标志位：是否正在恢复 scope 状态（恢复时不触发 watch）
const isRestoringScope = ref(false);

const getScopeTabType = () => {
    return privilegeForm.value.grantType === 'privilege' ? 'privilege' : privilegeGroupTypeTab.value;
};

const getCurrentScopeKey = () => {
    const dbName = privilegeForm.value.selectedDatabase;
    const collName = privilegeForm.value.selectedCollection;
    return `${getScopeTabType()}:${privilegeForm.value.grantType}:${dbName}:${collName}`;
};

// 计算每个权限组类型的已授权数量（统计所有 scope 中权限组勾选的总次数，不去重）
const getAuthorizedCountByType = (type: 'cluster' | 'database' | 'collection' | 'custom') => {
    // 访问版本号，确保 computed 能追踪到状态变化
    const version = privilegeStateVersion.value;

    let groupNames: string[];

    if (type === 'cluster') {
        groupNames = PRIVILEGE_GROUP_CATEGORIES.CLUSTER.groupNames;
    } else if (type === 'database') {
        groupNames = PRIVILEGE_GROUP_CATEGORIES.DATABASE.groupNames;
    } else if (type === 'collection') {
        groupNames = PRIVILEGE_GROUP_CATEGORIES.COLLECTION.groupNames;
    } else {
        // 自定义权限组：排除固定权限组
        const allGroups = privilegeGroups.value || [];
        groupNames = allGroups
            .map((g: MilvusPrivilegeGroup) => g.GroupName)
            .filter(
                (name: string) =>
                    !PRIVILEGE_GROUP_CATEGORIES.CLUSTER.groupNames.includes(name) &&
                    !PRIVILEGE_GROUP_CATEGORIES.DATABASE.groupNames.includes(name) &&
                    !PRIVILEGE_GROUP_CATEGORIES.COLLECTION.groupNames.includes(name)
            );
    }

    let totalCount = 0;

    // 只统计缓存中的所有 scope（当前表单状态已经通过 watch 保存到缓存中）
    dbCollectionPrivileges.value.forEach((state, key) => {
        if (state.grantType === 'privilegeGroup') {
            state.selectedPrivilegeGroups.forEach((groupName) => {
                if (groupNames.includes(groupName)) {
                    totalCount++;
                }
            });
        }
    });

    return totalCount;
};

// 计算属性：每个类型的已授权数量（基于当前表单状态）
const clusterAuthCount = computed(() => getAuthorizedCountByType('cluster'));
const databaseAuthCount = computed(() => getAuthorizedCountByType('database'));
const collectionAuthCount = computed(() => getAuthorizedCountByType('collection'));
const customAuthCount = computed(() => getAuthorizedCountByType('custom'));

const getPrivilegeAuthCount = () => {
    const version = privilegeStateVersion.value;
    let totalCount = 0;

    // 统计所有缓存的 scope 中权限授权的总次数（不去重，每次 scope 的权限都单独计数）
    dbCollectionPrivileges.value.forEach((state, key) => {
        const parts = key.split(':');
        const keyGrantType = parts[1];
        if (keyGrantType === 'privilege') {
            totalCount += state.selectedPrivileges.length;
        }
    });

    // 统计当前表单中已选择但还未保存到缓存的权限
    if (privilegeForm.value.grantType === 'privilege') {
        const scopeKey = getCurrentScopeKey();
        const saved = dbCollectionPrivileges.value.get(scopeKey);
        // 只统计缓存中没有的权限
        privilegeForm.value.selectedPrivileges.forEach((priv) => {
            if (!saved || !saved.selectedPrivileges.includes(priv)) {
                totalCount++;
            }
        });
    }

    return totalCount;
};

const privilegeGroupAuthCount = computed(() => clusterAuthCount.value + databaseAuthCount.value + collectionAuthCount.value + customAuthCount.value);
const privilegeAuthCount = computed(() => getPrivilegeAuthCount());

// 计算属性：根据分类过滤权限组
const filteredPrivilegeGroups = computed(() => {
    const allGroups = privilegeGroups.value || [];

    if (privilegeGroupTypeTab.value === 'cluster') {
        return allGroups.filter((g: MilvusPrivilegeGroup) => PRIVILEGE_GROUP_CATEGORIES.CLUSTER.groupNames.includes(g.GroupName));
    } else if (privilegeGroupTypeTab.value === 'database') {
        return allGroups.filter((g: MilvusPrivilegeGroup) => PRIVILEGE_GROUP_CATEGORIES.DATABASE.groupNames.includes(g.GroupName));
    } else if (privilegeGroupTypeTab.value === 'collection') {
        return allGroups.filter((g: MilvusPrivilegeGroup) => PRIVILEGE_GROUP_CATEGORIES.COLLECTION.groupNames.includes(g.GroupName));
    } else {
        // 自定义权限组：排除上述所有固定权限组
        const fixedGroupNames = [
            ...PRIVILEGE_GROUP_CATEGORIES.CLUSTER.groupNames,
            ...PRIVILEGE_GROUP_CATEGORIES.DATABASE.groupNames,
            ...PRIVILEGE_GROUP_CATEGORIES.COLLECTION.groupNames,
        ];
        return allGroups.filter((g: MilvusPrivilegeGroup) => !fixedGroupNames.includes(g.GroupName));
    }
});

// 筛选文本
const dbFilterText = ref('');
const collFilterText = ref('');

// 权限组
const privilegeGroups = ref<MilvusPrivilegeGroup[]>([]);

const databases = ref<IDatabase[]>([]);
const collections = ref<ICollection[]>([]);

// 过滤后的数据库列表
const filteredDatabases = computed(() => {
    if (!dbFilterText.value) return databases.value;
    return databases.value.filter((db) => db.name.toLowerCase().includes(dbFilterText.value.toLowerCase()));
});

// 过滤后的 Collection 列表
const filteredCollections = computed(() => {
    if (!collFilterText.value) return collections.value;
    return collections.value.filter((coll) => coll.name.toLowerCase().includes(collFilterText.value.toLowerCase()));
});

// 检查角色是否拥有 * 通配符权限
const hasWildcardPrivilege = (type: 'database' | 'collection') => {
    const privs = privilegeDialog.value.currentRoleData?.privileges || [];
    return privs.some((p: MilvusPrivilegeItem) => {
        if (type === 'database') {
            return p.DbName === '*';
        } else if (type === 'collection') {
            return p.DbName === privilegeDialog.value.currentDb && p.ObjectName === '*';
        }
        return false;
    });
};

// 检查数据库是否已授权（按当前授权方式和权限组类型独立判断）
const isDatabaseAuthorized = (dbName: string) => {
    const currentTab = getScopeTabType();
    const currentGrantType = privilegeForm.value.grantType;

    // 检查当前 Tab 和授权方式下，该数据库本身是否已授权（collName 为 *）
    const dbKey = `${currentTab}:${currentGrantType}:${dbName}:*`;
    const dbSaved = dbCollectionPrivileges.value.get(dbKey);
    if (dbSaved) {
        const hasValidGroup = dbSaved.selectedPrivilegeGroups.some((groupName) => isPrivilegeGroupBelongsToTab(groupName, currentTab));
        if (hasValidGroup || dbSaved.selectedPrivileges.length > 0) {
            return true;
        }
    }

    // 检查当前 Tab 和授权方式下，该数据库是否有任何 collection 已授权
    for (const [key, saved] of dbCollectionPrivileges.value.entries()) {
        const parts = key.split(':');
        const keyTab = parts[0];
        const keyGrantType = parts[1];
        const keyDbName = parts[2];
        const keyCollName = parts[3];

        // 只检查当前 Tab 和授权方式下，该数据库的 collection
        if (keyTab === currentTab && keyGrantType === currentGrantType && keyDbName === dbName && keyCollName !== '*') {
            const hasValidGroup = saved.selectedPrivilegeGroups.some((groupName) => isPrivilegeGroupBelongsToTab(groupName, currentTab));
            if (hasValidGroup || saved.selectedPrivileges.length > 0) {
                return true;
            }
        }
    }

    // 如果当前正在选择该数据库，检查当前表单状态
    if (privilegeForm.value.selectedDatabase === dbName) {
        const hasValidGroup = privilegeForm.value.selectedPrivilegeGroups.some((groupName) => isPrivilegeGroupBelongsToTab(groupName, currentTab));
        if (hasValidGroup || privilegeForm.value.selectedPrivileges.length > 0) {
            return true;
        }
    }

    return false;
};

// 检查权限组是否属于当前 Tab 类型
const isPrivilegeGroupBelongsToTab = (groupName: string, tabType: string) => {
    if (tabType === 'cluster') {
        return PRIVILEGE_GROUP_CATEGORIES.CLUSTER.groupNames.includes(groupName);
    } else if (tabType === 'database') {
        return PRIVILEGE_GROUP_CATEGORIES.DATABASE.groupNames.includes(groupName);
    } else if (tabType === 'collection') {
        return PRIVILEGE_GROUP_CATEGORIES.COLLECTION.groupNames.includes(groupName);
    } else {
        // custom: 排除所有固定权限组
        return (
            !PRIVILEGE_GROUP_CATEGORIES.CLUSTER.groupNames.includes(groupName) &&
            !PRIVILEGE_GROUP_CATEGORIES.DATABASE.groupNames.includes(groupName) &&
            !PRIVILEGE_GROUP_CATEGORIES.COLLECTION.groupNames.includes(groupName)
        );
    }
};

// 检查 Collection 是否已授权（按当前授权方式和权限组类型独立判断）
const isCollectionAuthorized = (collName: string) => {
    const dbName = privilegeForm.value.selectedDatabase;
    const currentTab = getScopeTabType();
    const currentGrantType = privilegeForm.value.grantType;

    // 使用当前 activeTab 和 grantType 作为 key 前缀，确保不同 Tab 和授权方式下的状态独立
    const key = `${currentTab}:${currentGrantType}:${dbName}:${collName}`;
    const saved = dbCollectionPrivileges.value.get(key);

    // 如果缓存中有该 scope 的记录，需要验证权限组是否属于当前 Tab 类型
    if (saved) {
        const hasValidPrivilegeGroup = saved.selectedPrivilegeGroups.some((groupName) => isPrivilegeGroupBelongsToTab(groupName, currentTab));

        if (currentTab === 'privilege') {
            return saved.selectedPrivileges.length > 0;
        }

        if (currentTab === 'collection' || currentTab === 'custom') {
            return hasValidPrivilegeGroup || saved.selectedPrivileges.length > 0;
        }

        return false;
    }

    // 如果当前正在选择该 Collection，检查当前表单状态
    if (privilegeForm.value.selectedCollection === collName) {
        if (currentTab === 'privilege') {
            return privilegeForm.value.selectedPrivileges.length > 0;
        }
        const hasValidPrivilegeGroup = privilegeForm.value.selectedPrivilegeGroups.some((groupName) => isPrivilegeGroupBelongsToTab(groupName, currentTab));
        return hasValidPrivilegeGroup || privilegeForm.value.selectedPrivileges.length > 0;
    }

    // 对于通配符 '*'，检查缓存中是否有当前 tabType + grantType + dbName 下 '*' scope 的授权记录
    if (collName === '*') {
        for (const [key, saved] of dbCollectionPrivileges.value.entries()) {
            const parts = key.split(':');
            const keyTab = parts[0];
            const keyGrantType = parts[1];
            const keyDbName = parts[2];
            const keyCollName = parts[3];
            if (keyTab === currentTab && keyGrantType === currentGrantType && keyDbName === dbName && keyCollName === '*') {
                if (currentTab === 'privilege') {
                    if (saved.selectedPrivileges.length > 0) {
                        return true;
                    }
                } else {
                    const hasValidPrivilegeGroup = saved.selectedPrivilegeGroups.some((groupName) => isPrivilegeGroupBelongsToTab(groupName, currentTab));
                    if (hasValidPrivilegeGroup || saved.selectedPrivileges.length > 0) {
                        return true;
                    }
                }
            }
        }
        return false;
    }

    // 对于具体的 Collection，如果没有缓存记录且不是当前选中项，返回 false
    return false;
};

// 根据当前勾选的权限/权限组，获取已授权的范围
const getAuthorizedScopes = () => {
    const selectedPrivileges = new Set<string>();

    // 收集所有选中的权限
    if (privilegeForm.value.grantType === 'privilegeGroup') {
        // 权限组模式：展开权限组为具体权限
        privilegeForm.value.selectedPrivilegeGroups.forEach((groupName) => {
            const group = privilegeGroups.value.find((g) => g.GroupName === groupName);
            if (group && group.Privileges) {
                group.Privileges.forEach((p: string) => selectedPrivileges.add(p));
            }
        });
    } else {
        // 具体权限模式
        privilegeForm.value.selectedPrivileges.forEach((p) => selectedPrivileges.add(p));
    }

    return selectedPrivileges;
};

// 保存当前 scope 的权限勾选状态（不触发版本号更新）
const saveCurrentScopeState = () => {
    const scopeKey = getCurrentScopeKey();
    dbCollectionPrivileges.value.set(scopeKey, {
        grantType: privilegeForm.value.grantType as GrantType,
        selectedPrivilegeGroups: [...privilegeForm.value.selectedPrivilegeGroups],
        selectedPrivileges: [...privilegeForm.value.selectedPrivileges],
    });
};

// 更新版本号，触发 computed 重新计算（仅在权限组勾选变化时调用）
const notifyPrivilegeChange = () => {
    privilegeStateVersion.value++;
};

// 选择数据库
const selectDatabase = async (dbName: string) => {
    // 保存当前 scope 的状态
    saveCurrentScopeState();

    privilegeForm.value.selectedDatabase = dbName;
    privilegeForm.value.selectedCollection = '*';
    collections.value = [];
    collFilterText.value = '';
    privilegeDialog.value.currentDb = dbName;

    if (dbName !== '*' && privilegeGroupTypeTab.value !== 'database') {
        // 非数据库权限 Tab 时，加载 Collection 列表
        const res = await milvusApi.listCollections(props.milvusId, dbName);
        collections.value = res || [];
    }

    // 恢复新 scope 的权限状态
    restoreScopePrivileges();
};

// 选择 Collection
const selectCollection = (collName: string) => {
    // 保存当前 scope 的状态
    saveCurrentScopeState();

    privilegeForm.value.selectedCollection = collName;
    // 恢复新 scope 的权限状态
    restoreScopePrivileges();
};

// 恢复当前 scope 的权限勾选状态
const restoreScopePrivileges = () => {
    const scopeKey = getCurrentScopeKey();
    const state = dbCollectionPrivileges.value.get(scopeKey);

    // 设置标志位，阻止 watch 触发
    isRestoringScope.value = true;

    if (state) {
        // 如果已有保存的状态，直接恢复
        privilegeForm.value.selectedPrivilegeGroups = [...state.selectedPrivilegeGroups];
        privilegeForm.value.selectedPrivileges = [...state.selectedPrivileges];
    } else {
        // 如果没有保存过状态，从角色已有权限加载
        loadCurrentScopePrivileges();
    }

    // 恢复标志位
    isRestoringScope.value = false;

    // 通知 computed 重新计算，确保 Badge 显示正确
    notifyPrivilegeChange();
};

// 监听授权方式切换和权限勾选变化，保存当前 scope 状态
watch(
    () => [privilegeForm.value.grantType, privilegeForm.value.selectedPrivilegeGroups, privilegeForm.value.selectedPrivileges],
    ([newGrantType, newGroups, newPrivs], [oldGrantType, oldGroups, oldPrivs]) => {
        // 如果是恢复状态过程中，不触发保存和通知
        if (isRestoringScope.value) {
            return;
        }

        // 如果是授权方式切换，不触发保存（由第二个 watch 处理）
        if (oldGrantType && newGrantType !== oldGrantType) {
            return;
        }

        saveCurrentScopeState();
        // 权限勾选变化时，通知 computed 重新计算
        notifyPrivilegeChange();
    },
    { deep: true }
);

// 监听Tab切换，自动设置对应的授权对象
watch(privilegeGroupTypeTab, (newTab, oldTab) => {
    if (privilegeForm.value.grantType === 'privilege') return;

    const oldKey = `${oldTab}:${privilegeForm.value.grantType}:${privilegeForm.value.selectedDatabase}:${privilegeForm.value.selectedCollection}`;
    dbCollectionPrivileges.value.set(oldKey, {
        grantType: privilegeForm.value.grantType as GrantType,
        selectedPrivilegeGroups: [...privilegeForm.value.selectedPrivilegeGroups],
        selectedPrivileges: [...privilegeForm.value.selectedPrivileges],
    });

    if (newTab === 'cluster') {
        privilegeForm.value.selectedDatabase = '*';
        privilegeForm.value.selectedCollection = '*';
    } else if (newTab === 'database') {
        privilegeForm.value.selectedCollection = '*';
        if (!privilegeForm.value.selectedDatabase) {
            privilegeForm.value.selectedDatabase = '*';
        }
    }

    restoreScopePrivileges();
});

watch(
    () => privilegeForm.value.grantType,
    (newType, oldType) => {
        // 先设置标志位，阻止第一个 watch 触发保存空数据
        isRestoringScope.value = true;

        if (oldType) {
            const oldTabType = oldType === 'privilege' ? 'privilege' : privilegeGroupTypeTab.value;
            const oldKey = `${oldTabType}:${oldType}:${privilegeForm.value.selectedDatabase}:${privilegeForm.value.selectedCollection}`;
            dbCollectionPrivileges.value.set(oldKey, {
                grantType: oldType as GrantType,
                selectedPrivilegeGroups: [...privilegeForm.value.selectedPrivilegeGroups],
                selectedPrivileges: [...privilegeForm.value.selectedPrivileges],
            });
        }

        // 恢复新授权方式的状态
        restoreScopePrivileges();

        // 恢复标志位
        isRestoringScope.value = false;
    }
);

const initAllScopePrivileges = (privs: MilvusPrivilegeItem[]) => {
    const scopeMap = new Map<string, MilvusPrivilegeItem[]>();
    const privilegeGroupNames = privilegeGroups.value.map((pg) => pg.GroupName);

    privs.forEach((p) => {
        const dbName = p.DbName || '*';
        const objName = p.ObjectName || '*';
        const privilegeName = p.Privilege || '';

        const isGroup = privilegeGroupNames.includes(privilegeName);

        let tabType: string;
        let grantType: string;

        if (isGroup) {
            grantType = 'privilegeGroup';
            if (PRIVILEGE_GROUP_CATEGORIES.CLUSTER.groupNames.includes(privilegeName)) {
                tabType = 'cluster';
            } else if (PRIVILEGE_GROUP_CATEGORIES.DATABASE.groupNames.includes(privilegeName)) {
                tabType = 'database';
            } else if (PRIVILEGE_GROUP_CATEGORIES.COLLECTION.groupNames.includes(privilegeName)) {
                tabType = 'collection';
            } else {
                tabType = 'custom';
            }
        } else {
            grantType = 'privilege';
            tabType = 'privilege';
        }

        const key = `${tabType}:${grantType}:${dbName}:${objName}`;

        if (!scopeMap.has(key)) {
            scopeMap.set(key, []);
        }
        scopeMap.get(key)!.push(p);
    });

    scopeMap.forEach((scopePrivs, key) => {
        const allPrivs = scopePrivs.map((p) => p.Privilege || '').filter((p) => p !== '*');

        const selectedGroups: string[] = [];
        const selectedIndividualPrivs: string[] = [];

        allPrivs.forEach((priv) => {
            if (privilegeGroupNames.includes(priv)) {
                selectedGroups.push(priv);
            } else {
                selectedIndividualPrivs.push(priv);
            }
        });

        // 关键修复：将权限组和单独权限分别存储到不同的 key 下
        // 权限组存储到：tabType:privilegeGroup:dbName:collName
        if (selectedGroups.length > 0) {
            dbCollectionPrivileges.value.set(key, {
                grantType: 'privilegeGroup' as const,
                selectedPrivilegeGroups: selectedGroups,
                selectedPrivileges: [],
            });
        }
        // 单独权限存储到：privilege:privilege:dbName:collName
        if (selectedIndividualPrivs.length > 0) {
            const privilegeKey = `privilege:privilege:${key.split(':')[2]}:${key.split(':')[3]}`;
            dbCollectionPrivileges.value.set(privilegeKey, {
                grantType: 'privilege' as const,
                selectedPrivilegeGroups: [],
                selectedPrivileges: selectedIndividualPrivs,
            });
        }
    });
};

// 加载当前 database+collection 的授权状态
const loadCurrentScopePrivileges = () => {
    const dbName = privilegeForm.value.selectedDatabase;
    const collName = privilegeForm.value.selectedCollection;
    const privs = privilegeDialog.value.currentRoleData?.privileges || [];
    const currentGrantType = privilegeForm.value.grantType;

    const matchedPrivs = privs.filter((p: MilvusPrivilegeItem) => {
        const matchDb = (p.DbName || '*') === dbName;
        const matchColl = (p.ObjectName || '*') === collName;
        return matchDb && matchColl;
    });

    const allPrivs = matchedPrivs.map((p) => p.Privilege || '').filter((p) => p !== '*');

    const privilegeGroupNames = privilegeGroups.value.map((pg) => pg.GroupName);
    const selectedGroups: string[] = [];
    const selectedIndividualPrivs: string[] = [];

    allPrivs.forEach((priv) => {
        if (privilegeGroupNames.includes(priv)) {
            selectedGroups.push(priv);
        } else {
            selectedIndividualPrivs.push(priv);
        }
    });

    if (currentGrantType === 'privilegeGroup') {
        // 只保留属于当前 Tab 类型的权限组，避免跨 Tab 类型的权限组误显
        const currentTab = getScopeTabType();
        privilegeForm.value.selectedPrivilegeGroups = selectedGroups.filter((groupName) => isPrivilegeGroupBelongsToTab(groupName, currentTab));
        privilegeForm.value.selectedPrivileges = [];
    } else {
        privilegeForm.value.selectedPrivilegeGroups = [];
        privilegeForm.value.selectedPrivileges = selectedIndividualPrivs;
    }

    saveCurrentScopeState();
};

// 当前角色已授权的数据库和Collection（用于醒目标注）
const authorizedDatabases = computed(() => {
    const privs = privilegeDialog.value.currentRoleData?.privileges || [];
    const dbSet = new Set<string>();
    privs.forEach((p: MilvusPrivilegeItem) => {
        if (p.DbName) {
            dbSet.add(p.DbName);
        }
    });
    return Array.from(dbSet);
});

const authorizedCollections = computed(() => {
    const privs = privilegeDialog.value.currentRoleData?.privileges || [];
    const collSet = new Set<string>();
    privs.forEach((p: MilvusPrivilegeItem) => {
        if (p.ObjectName) {
            collSet.add(p.ObjectName);
        }
    });
    return Array.from(collSet);
});

const getPrivilegeAuthScope = (privilegeName: string) => {
    const privs = privilegeDialog.value.currentRoleData?.privileges || [];
    const scopes: string[] = [];

    privs.forEach((p: MilvusPrivilegeItem) => {
        if (p.Privilege === privilegeName) {
            const scope = [];
            if (p.DbName) scope.push(`db:${p.DbName}`);
            if (p.ObjectName && p.ObjectName !== '*') scope.push(`coll:${p.ObjectName}`);
            scopes.push(scope.length > 0 ? scope.join(', ') : '全局');
        }
    });

    return scopes;
};

const getGroupAuthCount = (group: PrivilegeGroupUI) => {
    let count = 0;

    // 优先检查当前 scope 的缓存
    const scopeKey = getCurrentScopeKey();
    const saved = dbCollectionPrivileges.value.get(scopeKey);
    if (saved) {
        group.privileges.forEach((priv: string) => {
            if (saved.selectedPrivileges.includes(priv)) {
                count++;
            }
        });
    }

    // 如果当前正在选择该 scope，检查当前表单状态（如果缓存中没有或缓存与表单不一致）
    if (privilegeForm.value.grantType === 'privilege') {
        group.privileges.forEach((priv: string) => {
            if (privilegeForm.value.selectedPrivileges.includes(priv)) {
                // 如果缓存中没有记录，或者缓存中没有该权限，则计数
                if (!saved || !saved.selectedPrivileges.includes(priv)) {
                    count++;
                }
            }
        });
    }

    return count;
};

// 权限分组配置
const privilegeGroupList = ref([
    {
        name: 'collection',
        label: 'Collection 相关权限',
        privileges: [
            'CreateCollection',
            'DescribeCollection',
            'ShowCollections',
            'DropCollection',
            'RenameCollection',
            'CreateAlias',
            'DescribeAlias',
            'DropAlias',
            'ListAliases',
            'Load',
            'GetLoadingProgress',
            'GetLoadState',
            'Release',
            'Flush',
            'GetFlushState',
            'GetStatistics',
            'Compaction',
            'FlushAll',
        ],
    },
    {
        name: 'partition',
        label: 'Partition 相关权限',
        privileges: ['CreatePartition', 'DropPartition', 'ShowPartitions', 'HasPartition'],
    },
    {
        name: 'index',
        label: 'Index 相关权限',
        privileges: ['CreateIndex', 'DropIndex', 'IndexDetail'],
    },
    {
        name: 'entity',
        label: 'Entity 相关权限',
        privileges: ['Query', 'Insert', 'Upsert', 'Delete', 'Search', 'Import'],
    },
    {
        name: 'user',
        label: 'User 相关权限',
        privileges: ['CreateUser', 'DropUser', 'UpdateUser', 'SelectUser'],
    },
    {
        name: 'database',
        label: 'Database 相关权限',
        privileges: ['CreateDatabase', 'DropDatabase', 'ListDatabase', 'DescribeDatabase'],
    },
    {
        name: 'resourceGroup',
        label: 'ResourceGroup 相关权限',
        privileges: ['DescribeResourceGroup', 'ListResourceGroup', 'UpdateResourceGroups', 'CreateResourceGroup', 'DropResourceGroup'],
    },
]);

// 计算当前选中的权限组包含的所有权限
const currentPrivilegeGroupPrivileges = computed(() => {
    const allPrivs = new Set<string>();
    privilegeForm.value.selectedPrivilegeGroups.forEach((groupName) => {
        const group = privilegeGroups.value.find((g) => g.GroupName === groupName);
        if (group && group.Privileges) {
            group.Privileges.forEach((p: string) => allPrivs.add(p));
        }
    });
    return Array.from(allPrivs);
});

// 判断分组是否全选
const isGroupAllSelected = (group: PrivilegeGroupUI) => {
    if (group.privileges.length === 0) return false;
    return group.privileges.every((priv: string) => privilegeForm.value.selectedPrivileges.includes(priv));
};

// 判断分组是否半选
const isGroupIndeterminate = (group: PrivilegeGroupUI) => {
    if (group.privileges.length === 0) return false;
    const selectedCount = group.privileges.filter((priv: string) => privilegeForm.value.selectedPrivileges.includes(priv)).length;
    return selectedCount > 0 && selectedCount < group.privileges.length;
};

// 分组全选/取消全选
const handleGroupChange = (group: PrivilegeGroupUI, val: boolean | string | number) => {
    if (val) {
        // 全选：添加该组所有权限
        group.privileges.forEach((priv: string) => {
            if (!privilegeForm.value.selectedPrivileges.includes(priv)) {
                privilegeForm.value.selectedPrivileges.push(priv);
            }
        });
    } else {
        // 取消全选：移除该组所有权限
        privilegeForm.value.selectedPrivileges = privilegeForm.value.selectedPrivileges.filter((priv) => !group.privileges.includes(priv));
    }
};

// 当前角色拥有的权限
const currentRolePrivileges = computed(() => {
    return privilegeDialog.value.currentRoleData?.privileges || [];
});

// 加载权限组
const loadPrivilegeGroups = async () => {
    if (privilegeGroups.value.length) {
        return;
    }
    try {
        const res = await milvusApi.getPrivilegeGroups(props.milvusId);
        privilegeGroups.value = res || [];
    } catch (error: unknown) {
        privilegeGroups.value = [];
    }
};

// 加载数据库列表
const loadDatabases = async () => {
    try {
        const res = await milvusApi.listDatabases(props.milvusId);
        databases.value = res || [];
    } catch (error: unknown) {
        databases.value = [];
    }
};

// 打开授权弹窗
const handleGrantPrivilege = async (row: { roleName: string }) => {
    privilegeDialog.value.roleName = row.roleName;
    privilegeDialog.value.visible = true;
    privilegeLoading.value = true;

    try {
        // 重置表单
        privilegeForm.value = {
            grantType: 'privilegeGroup',
            selectedPrivilegeGroups: [],
            selectedDatabase: '*',
            selectedCollection: '*',
            selectedPrivileges: [],
        };

        // 清空之前的权限状态缓存
        dbCollectionPrivileges.value = new Map();

        // 重置Tab为默认
        privilegeGroupTypeTab.value = 'cluster';

        // 加载必要数据
        await Promise.all([loadPrivilegeGroups(), loadDatabases()]);

        // 获取角色最新的权限详情
        const roleData = await milvusApi.describeRole(props.milvusId, row.roleName);
        const currentPrivs = roleData?.Privileges || [];
        privilegeDialog.value.currentRoleData = { privileges: currentPrivs };

        // 初始化所有 scope 的权限状态缓存
        initAllScopePrivileges(currentPrivs);

        // 恢复当前 scope（默认为 cluster）的权限勾选状态
        restoreScopePrivileges();

        // 根据Tab类型设置默认值
        if (privilegeGroupTypeTab.value !== 'cluster') {
            // 默认选中第一个数据库
            if (databases.value.length > 0) {
                await selectDatabase(databases.value[0].name);
            }
        }
    } finally {
        privilegeLoading.value = false;
    }
};

// 提交授权
const submitPrivilege = async () => {
    const roleName = privilegeDialog.value.roleName;

    // 构建本次提交的所有 scope 的权限变更
    const allPrivileges: Record<string, PrivilegeScopeMap> = {};

    // 遍历所有已缓存的 scope 和当前表单，收集所有需要提交的权限
    const allScopes = new Set<string>();

    // 添加所有缓存的 scope
    dbCollectionPrivileges.value.forEach((state, key) => {
        allScopes.add(key);
    });

    // 添加当前 scope
    allScopes.add(getCurrentScopeKey());

    // 处理每个 scope 的权限变更
    allScopes.forEach((scopeKey) => {
        // 解析 scope key: "tabType:grantType:dbName:collName"
        const parts = scopeKey.split(':');
        const dbName = parts[2];
        const collName = parts[3];

        // 获取该 scope 的状态（缓存或当前表单）
        let state;
        if (scopeKey === getCurrentScopeKey()) {
            // 当前 scope 使用当前表单状态
            state = {
                grantType: privilegeForm.value.grantType,
                selectedPrivilegeGroups: privilegeForm.value.selectedPrivilegeGroups,
                selectedPrivileges: privilegeForm.value.selectedPrivileges,
            };
        } else {
            state = dbCollectionPrivileges.value.get(scopeKey);
        }

        if (!state) return;

        // 初始化该 scope 的权限结构
        if (!allPrivileges[dbName]) {
            allPrivileges[dbName] = { collections: {} };
        }
        if (!allPrivileges[dbName].collections[collName]) {
            allPrivileges[dbName].collections[collName] = {};
        }

        // 收集该 scope 的所有权限（权限组 + 具体权限）
        const allPrivsForScope = [...state.selectedPrivilegeGroups, ...state.selectedPrivileges];
        allPrivsForScope.forEach((priv) => {
            allPrivileges[dbName].collections[collName][priv] = true;
        });
    });

    // 对比角色当前权限，计算授权(true)和取消授权(false)
    const currentPrivs = privilegeDialog.value.currentRoleData?.privileges || [];
    const existingPrivilegeMap = new Map<string, boolean>();

    // 构建现有权限的映射
    currentPrivs.forEach((p: MilvusPrivilegeItem) => {
        const dbName = p.DbName || '*';
        const collName = p.ObjectName || '*';
        const privName = p.Privilege;
        const key = `${dbName}:${collName}:${privName}`;
        existingPrivilegeMap.set(key, true);
    });

    // 构建最终提交数据
    const finalPrivileges: Record<string, PrivilegeScopeMap> = {};

    // 处理新授权的权限（值为 true）
    Object.keys(allPrivileges).forEach((dbName) => {
        Object.keys(allPrivileges[dbName].collections).forEach((collName) => {
            Object.keys(allPrivileges[dbName].collections[collName]).forEach((privName) => {
                const key = `${dbName}:${collName}:${privName}`;
                if (!existingPrivilegeMap.has(key)) {
                    // 新授权
                    if (!finalPrivileges[dbName]) {
                        finalPrivileges[dbName] = { collections: {} };
                    }
                    if (!finalPrivileges[dbName].collections[collName]) {
                        finalPrivileges[dbName].collections[collName] = {};
                    }
                    finalPrivileges[dbName].collections[collName][privName] = true;
                }
                // 从 existingPrivilegeMap 中移除已处理的权限
                existingPrivilegeMap.delete(key);
            });
        });
    });

    // 处理取消授权的权限（值为 false）
    existingPrivilegeMap.forEach((_, key) => {
        const [dbName, collName, privName] = key.split(':');
        if (!finalPrivileges[dbName]) {
            finalPrivileges[dbName] = { collections: {} };
        }
        if (!finalPrivileges[dbName].collections[collName]) {
            finalPrivileges[dbName].collections[collName] = {};
        }
        finalPrivileges[dbName].collections[collName][privName] = false;
    });

    // 检查是否有变更
    const hasChanges = Object.keys(finalPrivileges).length > 0;
    if (!hasChanges) {
        Msg.warning('没有权限变更');
        return;
    }

    privilegeLoading.value = true;
    try {
        await milvusApi.updateRole(props.milvusId, {
            roleName,
            privileges: finalPrivileges,
        });

        Msg.success('milvus.grantSuccess');
        privilegeDialog.value.visible = false;
        emit('privilege-saved');
    } finally {
        privilegeLoading.value = false;
    }
};

// 暴露方法供父组件调用
defineExpose({
    handleGrantPrivilege,
});
</script>

<style scoped>
.group-checkboxes {
    padding: 0 16px;
}

.tab-badge {
    margin-left: 8px;
}
</style>
