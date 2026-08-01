<template>
    <div class="home-container personal overflow-x-hidden">
        <!-- 个人信息卡片 -->
        <div class="mb-4">
            <el-card shadow="hover" :header="$t('home.personalInfo')">
                <div class="flex flex-col sm:flex-row gap-4 items-center sm:items-start">
                    <div class="w-25 rounded overflow-hidden shrink-0">
                        <el-upload
                            class="h-full!"
                            :action="getUploadFileUrl(`avatar_${userInfo.username}`)"
                            :limit="1"
                            :show-file-list="false"
                            :before-upload="beforeAvatarUpload"
                            :on-success="handleAvatarSuccess"
                            accept=".png,.jpg,.jpeg"
                        >
                            <img :src="userInfo.photo" class="w-full h-full rounded transition-transform duration-300 hover:scale-110" />
                        </el-upload>
                    </div>

                    <div class="flex-1 px-3.75">
                        <div class="mb-4 text-lg truncate">{{ $t('home.welcomeMsg', { name: userInfo.name }) }}</div>
                        <div class="grid grid-cols-1 sm:grid-cols-2 gap-x-4 gap-y-1.5 text-[13px]">
                            <div class="flex items-center">
                                <span class="text-gray-500 mr-2 truncate">{{ $t('common.username') }}：</span>
                                <span class="truncate">{{ userInfo.username }}</span>
                            </div>
                            <div class="flex items-center">
                                <span class="text-gray-500 mr-2 truncate">{{ $t('common.role') }}：</span>
                                <span class="truncate">{{ roleInfo }}</span>
                            </div>
                            <div class="flex items-center">
                                <span class="text-gray-500 mr-2 truncate">{{ $t('home.lastLoginIp') }}：</span>
                                <span class="truncate">{{ userInfo.lastLoginIp }}</span>
                            </div>
                            <div class="flex items-center">
                                <span class="text-gray-500 mr-2 truncate">{{ $t('home.lastLoginTime') }}：</span>
                                <span class="truncate">{{ formatDate(userInfo.lastLoginTime) }}</span>
                            </div>
                        </div>
                    </div>
                </div>
            </el-card>
        </div>

        <div class="mt-4 grid grid-cols-1 sm:grid-cols-2 gap-4">
            <!-- 快捷入口卡片 -->
            <div>
                <el-card shadow="hover" class="h-105">
                    <template #header>
                        <div class="flex justify-between items-center font-medium">
                            <span>{{ $t('home.quickAccess') }}</span>
                        </div>
                    </template>
                    <el-scrollbar :max-height="400">
                        <div class="p-3">
                            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
                                <component v-for="comp of resourceComponents" :is="comp" @navigate="navigateToResource" />
                            </div>
                        </div>
                    </el-scrollbar>
                </el-card>
            </div>

            <!-- 最近操作记录 -->
            <div>
                <el-card shadow="hover" class="h-105">
                    <template #header>
                        <div class="flex justify-between items-center font-medium">
                            <span>{{ $t('home.recentOperations') }}</span>
                        </div>
                    </template>
                    <el-table :data="state.recentOpLogs" :height="270" stripe size="small" :empty-text="$t('home.noOpRecord')">
                        <el-table-column prop="createTime" :label="$t('common.time')" show-overflow-tooltip width="140">
                            <template #default="scope">
                                {{ formatDate(scope.row.createTime) }}
                            </template>
                        </el-table-column>

                        <el-table-column prop="codePath" :label="$t('common.path')" min-width="150" show-overflow-tooltip>
                            <template #default="scope">
                                <TagCodePath :path="scope.row.codePath" />
                            </template>
                        </el-table-column>
                        <el-table-column :label="$t('common.operation')" width="60">
                            <template #default="scope">
                                <el-link @click="navigateToResource(scope.row.codePath)" type="primary" icon="Position"></el-link>
                            </template>
                        </el-table-column>
                    </el-table>
                </el-card>
            </div>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { getFileUrl, getUploadFileUrl } from '@/common/request';
import { formatAxis, formatDate } from '@/common/utils/format';
import { saveUser } from '@/common/utils/storage';
import { Msg } from '@/hooks/useI18n';
import { useAutoOpenResource } from '@/store/autoOpenResource';
import { useUserInfo } from '@/store/userInfo';
import { storeToRefs } from 'pinia';
import { computed, onMounted, reactive } from 'vue';
import { useRouter } from 'vue-router';
import TagCodePath from '../ops/component/TagCodePath.vue';
import { resourceOpLogApi } from '../ops/tag/api';
import type { ResourceOpLog } from '../ops/tag/types';
import type { SysRole } from '../system/types';
import { personApi } from '../personal/api';
import { resourceComponents } from './resources';

const router = useRouter();
const { userInfo } = storeToRefs(useUserInfo());

const state = reactive({
    accountInfo: {
        roles: [] as SysRole[],
    },
    msgs: [],
    defaultLogSize: 20,
    recentOpLogs: [] as ResourceOpLog[],
});

const roleInfo = computed(() => {
    if (state.accountInfo.roles.length == 0) {
        return '';
    }
    return state.accountInfo.roles.map((val: SysRole) => val.name).join('、');
});

// 当前时间提示语
const currentTime = computed(() => {
    return formatAxis(new Date());
});

// 页面加载时
onMounted(() => {
    initData();
    getAccountInfo();
});

const getAccountInfo = async () => {
    state.accountInfo = await personApi.accountInfo.request();
};

const beforeAvatarUpload = (rawFile: File) => {
    if (rawFile.size >= 512 * 1024) {
        Msg.error('头像不能超过512KB!');
        return false;
    }
    return true;
};

const handleAvatarSuccess = (_response: unknown, uploadFile: import('element-plus').UploadFile) => {
    if (uploadFile.raw) {
        userInfo.value.photo = URL.createObjectURL(uploadFile.raw);
    }

    const newUser = { ...userInfo.value };
    newUser.photo = getFileUrl(`avatar_${userInfo.value.username}`);
    // 存储用户信息到浏览器缓存
    saveUser(newUser);
};

// 初始化数据
const initData = async () => {
    // 获取最近操作记录（不区分资源类型）
    try {
        const opLogsRes = await resourceOpLogApi.getAccountResourceOpLogs.request({
            pageSize: state.defaultLogSize,
        });
        state.recentOpLogs = opLogsRes.list || [];
    } catch (error) {
        console.error('Failed to load recent operation logs:', error);
    }
};

// 快捷跳转
const toPage = (item: string, codePath = '') => {
    let path;
    useAutoOpenResource().setCodePath(codePath);
    switch (item) {
        case 'personal': {
            router.push('/personal');
            break;
        }
        default: {
            path = '/my-resource';
        }
    }

    router.push({ path });
};

// 资源导航
const navigateToResource = (codePath: string) => {
    toPage('resource', codePath);
};
</script>

<style scoped></style>
