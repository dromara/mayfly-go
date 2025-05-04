<template>
    <div class="home-container personal">
        <el-row :gutter="15">
            <!-- 个人信息 -->
            <el-col :xs="24" :sm="16">
                <el-card shadow="hover" :header="$t('home.personalInfo')">
                    <div class="personal-user">
                        <div class="personal-user-left">
                            <el-upload
                                class="!h-full personal-user-left-upload"
                                :action="getUploadFileUrl(`avatar_${userInfo.username}`)"
                                :limit="1"
                                :show-file-list="false"
                                :before-upload="beforeAvatarUpload"
                                :on-success="handleAvatarSuccess"
                                accept=".png,.jpg,.jpeg"
                            >
                                <img :src="userInfo.photo" />
                            </el-upload>
                        </div>
                        <div class="personal-user-right">
                            <el-row>
                                <el-col :span="24" class="personal-title mb-4">
                                    {{ $t('home.welcomeMsg', { name: userInfo.name }) }}
                                </el-col>
                                <el-col :span="24">
                                    <el-row>
                                        <el-col :xs="24" :sm="12" class="personal-item !mb-1.5">
                                            <div class="personal-item-label">{{ $t('common.username') }}：</div>
                                            <div class="personal-item-value">{{ userInfo.username }}</div>
                                        </el-col>
                                        <el-col :xs="24" :sm="12" class="personal-item !mb-1.5">
                                            <div class="personal-item-label">{{ $t('common.role') }}：</div>
                                            <div class="personal-item-value">{{ roleInfo }}</div>
                                        </el-col>
                                    </el-row>
                                </el-col>
                                <el-col :span="24">
                                    <el-row>
                                        <el-col :xs="24" :sm="12" class="personal-item !mb-1.5">
                                            <div class="personal-item-label">{{ $t('home.lastLoginIp') }}：</div>
                                            <div class="personal-item-value">{{ userInfo.lastLoginIp }}</div>
                                        </el-col>
                                        <el-col :xs="24" :sm="12" class="personal-item !mb-1.5">
                                            <div class="personal-item-label">{{ $t('home.lastLoginTime') }}：</div>
                                            <div class="personal-item-value">{{ formatDate(userInfo.lastLoginTime) }}</div>
                                        </el-col>
                                    </el-row>
                                </el-col>
                            </el-row>
                        </div>
                    </div>
                </el-card>
            </el-col>

            <!-- 消息通知 -->
            <el-col :xs="24" :sm="8" class="pl15 personal-info">
                <el-card shadow="hover">
                    <template #header>
                        <span>{{ $t('home.msgNotify') }}</span>
                        <span @click="showMsgs" class="personal-info-more">{{ $t('common.more') }}</span>
                    </template>
                    <div class="personal-info-box">
                        <ul class="personal-info-ul">
                            <li v-for="(v, k) in state.msgs as any" :key="k" class="personal-info-li">
                                <a class="personal-info-li-title">{{ `[${$t(EnumValue.getLabelByValue(MsgTypeEnum, v.type))}] ${v.msg}` }}</a>
                            </li>
                        </ul>
                    </div>
                </el-card>
            </el-col>
        </el-row>

        <el-row :gutter="20" class="!mt-4 resource-info">
            <el-col :sm="12">
                <el-card shadow="hover">
                    <template #header>
                        <el-row justify="center">
                            <div class="resource-num pointer-icon" @click="toPage('machine')">
                                <SvgIcon
                                    class="mb-1 mr-1"
                                    :size="28"
                                    :name="TagResourceTypeEnum.Machine.extra.icon"
                                    :color="TagResourceTypeEnum.Machine.extra.iconColor"
                                />
                                <span class="">{{ state.machine.num }}</span>
                            </div>
                        </el-row>
                    </template>
                    <el-row>
                        <el-col :sm="24">
                            <el-table
                                :data="state.machine.opLogs"
                                :height="state.resourceOpTableHeight"
                                stripe
                                size="small"
                                :empty-text="$t('home.noOpRecord')"
                            >
                                <el-table-column prop="createTime" show-overflow-tooltip width="135">
                                    <template #default="scope">
                                        {{ formatDate(scope.row.createTime) }}
                                    </template>
                                </el-table-column>
                                <el-table-column prop="codePath" min-width="400" show-overflow-tooltip>
                                    <template #default="scope">
                                        <TagCodePath :path="scope.row.codePath" :tagInfos="state.machine.tagInfos" />
                                    </template>
                                </el-table-column>
                                <el-table-column width="30">
                                    <template #default="scope">
                                        <el-link @click="toPage('machine', scope.row.codePath)" type="primary" icon="Position"></el-link>
                                    </template>
                                </el-table-column>
                            </el-table>
                        </el-col>
                    </el-row>
                </el-card>
            </el-col>

            <el-col :sm="12">
                <el-card shadow="hover">
                    <template #header>
                        <el-row justify="center">
                            <div class="resource-num pointer-icon" @click="toPage('db')">
                                <SvgIcon
                                    class="mb-1 mr-1"
                                    :size="28"
                                    :name="TagResourceTypeEnum.DbInstance.extra.icon"
                                    :color="TagResourceTypeEnum.DbInstance.extra.iconColor"
                                />
                                <span class="">{{ state.db.num }}</span>
                            </div>
                        </el-row>
                    </template>
                    <el-row>
                        <el-col :sm="24">
                            <el-table :data="state.db.opLogs" :height="state.resourceOpTableHeight" stripe size="small" :empty-text="$t('home.noOpRecord')">
                                <el-table-column prop="createTime" show-overflow-tooltip min-width="135">
                                    <template #default="scope">
                                        {{ formatDate(scope.row.createTime) }}
                                    </template>
                                </el-table-column>
                                <el-table-column prop="codePath" min-width="380" show-overflow-tooltip>
                                    <template #default="scope">
                                        <TagCodePath :path="scope.row.codePath" :tagInfos="state.db.tagInfos" />
                                    </template>
                                </el-table-column>
                                <el-table-column width="30">
                                    <template #default="scope">
                                        <el-link @click="toPage('db', scope.row.codePath)" type="primary" icon="Position"></el-link>
                                    </template>
                                </el-table-column>
                            </el-table>
                        </el-col>
                    </el-row>
                </el-card>
            </el-col>
        </el-row>

        <el-row :gutter="20" class="!mt-4 resource-info">
            <el-col :sm="12">
                <el-card shadow="hover">
                    <template #header>
                        <el-row justify="center">
                            <div class="resource-num pointer-icon" @click="toPage('redis')">
                                <SvgIcon
                                    class="mb-1 mr-1"
                                    :size="28"
                                    :name="TagResourceTypeEnum.Redis.extra.icon"
                                    :color="TagResourceTypeEnum.Redis.extra.iconColor"
                                />
                                <span class="">{{ state.redis.num }}</span>
                            </div>
                        </el-row>
                    </template>
                    <el-row>
                        <el-col :sm="24">
                            <el-table :data="state.redis.opLogs" :height="state.resourceOpTableHeight" stripe size="small" :empty-text="$t('home.noOpRecord')">
                                <el-table-column prop="createTime" show-overflow-tooltip min-width="135">
                                    <template #default="scope">
                                        {{ formatDate(scope.row.createTime) }}
                                    </template>
                                </el-table-column>
                                <el-table-column prop="codePath" min-width="380" show-overflow-tooltip>
                                    <template #default="scope">
                                        <TagCodePath :path="scope.row.codePath" :tagInfos="state.redis.tagInfos" />
                                    </template>
                                </el-table-column>
                                <el-table-column width="30">
                                    <template #default="scope">
                                        <el-link @click="toPage('redis', scope.row.codePath)" type="primary" icon="Position"></el-link>
                                    </template>
                                </el-table-column>
                            </el-table>
                        </el-col>
                    </el-row>
                </el-card>
            </el-col>

            <el-col :sm="12">
                <el-card shadow="hover">
                    <template #header>
                        <el-row justify="center">
                            <div class="resource-num pointer-icon" @click="toPage('mongo')">
                                <SvgIcon
                                    class="mb-1 mr-1"
                                    :size="28"
                                    :name="TagResourceTypeEnum.Mongo.extra.icon"
                                    :color="TagResourceTypeEnum.Mongo.extra.iconColor"
                                />
                                <span class="">{{ state.mongo.num }}</span>
                            </div>
                        </el-row>
                    </template>
                    <el-row>
                        <el-col :sm="24">
                            <el-table :data="state.mongo.opLogs" :height="state.resourceOpTableHeight" stripe size="small" :empty-text="$t('home.noOpRecord')">
                                <el-table-column prop="createTime" show-overflow-tooltip min-width="135">
                                    <template #default="scope">
                                        {{ formatDate(scope.row.createTime) }}
                                    </template>
                                </el-table-column>
                                <el-table-column prop="codePath" min-width="380" show-overflow-tooltip>
                                    <template #default="scope">
                                        <TagCodePath :path="scope.row.codePath" :tagInfos="state.mongo.tagInfos" />
                                    </template>
                                </el-table-column>
                                <el-table-column width="30">
                                    <template #default="scope">
                                        <el-link @click="toPage('mongo', scope.row.codePath)" type="primary" icon="Position"></el-link>
                                    </template>
                                </el-table-column>
                            </el-table>
                        </el-col>
                    </el-row>
                </el-card>
            </el-col>
        </el-row>

        <el-dialog width="900px" :title="$t('common.msg')" v-model="msgDialog.visible">
            <el-table border :data="msgDialog.msgs.list" size="small">
                <el-table-column property="type" :label="$t('common.type')" width="60">
                    <template #default="scope">
                        {{ $t(EnumValue.getLabelByValue(MsgTypeEnum, scope.row.type)) }}
                    </template>
                </el-table-column>
                <el-table-column property="msg" :label="$t('common.msg')"></el-table-column>
                <el-table-column property="createTime" :label="$t('common.time')" width="150">
                    <template #default="scope">
                        {{ formatDate(scope.row.createTime) }}
                    </template>
                </el-table-column>
            </el-table>
            <el-row type="flex" class="mt-1" justify="center">
                <el-pagination
                    small
                    @current-change="searchMsg"
                    style="text-align: center"
                    background
                    layout="prev, pager, next, total, jumper"
                    :total="msgDialog.msgs.total"
                    v-model:current-page="msgDialog.query.pageNum"
                    :page-size="msgDialog.query.pageSize"
                />
            </el-row>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, toRefs } from 'vue';
// import * as echarts from 'echarts';
import { formatAxis, formatDate } from '@/common/utils/format';
import { indexApi } from './api';
import { useRouter } from 'vue-router';
import { storeToRefs } from 'pinia';
import { useUserInfo } from '@/store/userInfo';
import { personApi } from '../personal/api';
import SvgIcon from '@/components/svgIcon/index.vue';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { resourceOpLogApi } from '../ops/tag/api';
import TagCodePath from '../ops/component/TagCodePath.vue';
import { useAutoOpenResource } from '@/store/autoOpenResource';
import { getAllTagInfoByCodePaths } from '../ops/component/tag';
import { ElMessage } from 'element-plus';
import { getFileUrl, getUploadFileUrl } from '@/common/request';
import { saveUser } from '@/common/utils/storage';
import EnumValue from '../../common/Enum';
import { MsgTypeEnum } from './enums';

const router = useRouter();
const { userInfo } = storeToRefs(useUserInfo());

const state = reactive({
    accountInfo: {
        roles: [],
    },
    msgs: [],
    msgDialog: {
        visible: false,
        query: {
            pageSize: 10,
            pageNum: 1,
        },
        msgs: {
            list: [],
            total: null,
        },
    },
    resourceOpTableHeight: 180,
    defaultLogSize: 5,
    machine: {
        num: 0,
        opLogs: [],
        tagInfos: {},
    },
    db: {
        num: 0,
        opLogs: [],
        tagInfos: {},
    },
    redis: {
        num: 0,
        opLogs: [],
        tagInfos: {},
    },
    mongo: {
        num: 0,
        opLogs: [],
        tagInfos: {},
    },
});

const { msgDialog } = toRefs(state);

const roleInfo = computed(() => {
    if (state.accountInfo.roles.length == 0) {
        return '';
    }
    return state.accountInfo.roles.map((val: any) => val.roleName).join('、');
});

// 当前时间提示语
const currentTime = computed(() => {
    return formatAxis(new Date());
});

// 页面加载时
onMounted(() => {
    initData();
    getAccountInfo();

    getMsgs().then((res) => {
        state.msgs = res.list;
    });
});

const showMsgs = async () => {
    state.msgDialog.query.pageNum = 1;
    searchMsg();
    state.msgDialog.visible = true;
};

const searchMsg = async () => {
    state.msgDialog.msgs = await getMsgs();
};

const getAccountInfo = async () => {
    state.accountInfo = await personApi.accountInfo.request();
};

const getMsgs = async () => {
    return await personApi.getMsgs.request(state.msgDialog.query);
};

const beforeAvatarUpload = (rawFile: any) => {
    if (rawFile.size >= 512 * 1024) {
        ElMessage.error('头像不能超过512KB!');
        return false;
    }
    return true;
};

const handleAvatarSuccess = (response: any, uploadFile: any) => {
    userInfo.value.photo = URL.createObjectURL(uploadFile.raw);

    const newUser = { ...userInfo.value };
    newUser.photo = getFileUrl(`avatar_${userInfo.value.username}`);
    // 存储用户信息到浏览器缓存
    saveUser(newUser);
};

// 初始化数字滚动
const initData = async () => {
    resourceOpLogApi.getAccountResourceOpLogs
        .request({ resourceType: TagResourceTypeEnum.Machine.value, pageSize: state.defaultLogSize })
        .then(async (res: any) => {
            const tagInfos = await getAllTagInfoByCodePaths(res.list?.map((item: any) => item.codePath));
            state.machine.tagInfos = tagInfos;
            state.machine.opLogs = res.list;
        });

    resourceOpLogApi.getAccountResourceOpLogs
        .request({ resourceType: TagResourceTypeEnum.DbInstance.value, pageSize: state.defaultLogSize })
        .then(async (res: any) => {
            const tagInfos = await getAllTagInfoByCodePaths(res.list?.map((item: any) => item.codePath));
            state.db.tagInfos = tagInfos;
            state.db.opLogs = res.list;
        });

    resourceOpLogApi.getAccountResourceOpLogs
        .request({ resourceType: TagResourceTypeEnum.Redis.value, pageSize: state.defaultLogSize })
        .then(async (res: any) => {
            const tagInfos = await getAllTagInfoByCodePaths(res.list?.map((item: any) => item.codePath));
            state.redis.tagInfos = tagInfos;
            state.redis.opLogs = res.list;
        });

    resourceOpLogApi.getAccountResourceOpLogs
        .request({ resourceType: TagResourceTypeEnum.Mongo.value, pageSize: state.defaultLogSize })
        .then(async (res: any) => {
            const tagInfos = await getAllTagInfoByCodePaths(res.list?.map((item: any) => item.codePath));
            state.mongo.tagInfos = tagInfos;
            state.mongo.opLogs = res.list;
        });

    indexApi.machineDashbord.request().then((res: any) => {
        state.machine.num = res.machineNum;
    });

    indexApi.dbDashbord.request().then((res: any) => {
        state.db.num = res.dbNum;
    });

    indexApi.redisDashbord.request().then((res: any) => {
        state.redis.num = res.redisNum;
    });

    indexApi.mongoDashbord.request().then((res: any) => {
        state.mongo.num = res.mongoNum;
    });
};

const toPage = (item: any, codePath = '') => {
    let path;
    switch (item) {
        case 'personal': {
            router.push('/personal');
            break;
        }
        case 'mongo': {
            useAutoOpenResource().setMongoCodePath(codePath);
            path = '/mongo/mongo-data-operation';
            break;
        }
        case 'machine': {
            useAutoOpenResource().setMachineCodePath(codePath);
            path = '/machine/machines-op';
            break;
        }
        case 'db': {
            useAutoOpenResource().setDbCodePath(codePath);
            path = '/dbms/sql-exec';
            break;
        }
        case 'redis': {
            useAutoOpenResource().setRedisCodePath(codePath);
            path = '/redis/data-operation';
            break;
        }
    }

    router.push({ path });
};
</script>

<style scoped lang="scss">
@use '@/theme/mixins/index.scss' as mixins;

.personal {
    .personal-user {
        height: 130px;
        display: flex;
        align-items: center;

        .personal-user-left {
            width: 100px;
            height: 130px;
            border-radius: 3px;

            ::v-deep(.el-upload) {
                height: 100%;
            }

            .personal-user-left-upload {
                img {
                    width: 100%;
                    height: 100%;
                    border-radius: 3px;
                }

                &:hover {
                    img {
                        animation: logoAnimation 0.3s ease-in-out;
                    }
                }
            }
        }

        .personal-user-right {
            flex: 1;
            padding: 0 15px;

            .personal-title {
                font-size: 18px;
                @include mixins.text-ellipsis(1);
            }

            .personal-item {
                display: flex;
                align-items: center;
                font-size: 13px;

                .personal-item-label {
                    color: gray;
                    @include mixins.text-ellipsis(1);
                }

                .personal-item-value {
                    @include mixins.text-ellipsis(1);
                }
            }
        }
    }

    .personal-info {
        .personal-info-more {
            float: right;
            color: gray;
            font-size: 13px;

            &:hover {
                color: var(--el-color-primary);
                cursor: pointer;
            }
        }

        .personal-info-box {
            height: 130px;
            overflow: hidden;

            .personal-info-ul {
                list-style: none;

                .personal-info-li {
                    font-size: 13px;
                    padding-bottom: 10px;

                    .personal-info-li-title {
                        display: inline-block;
                        @include mixins.text-ellipsis(1);
                        color: grey;
                        text-decoration: none;
                    }

                    & a:hover {
                        color: var(--el-color-primary);
                        cursor: pointer;
                    }
                }
            }
        }
    }
}

.resource-info {
    text-align: center;

    ::v-deep(.el-card__header) {
        padding: 2px 20px;
    }

    .resource-num {
        font-weight: 700;
        font-size: 2vw;
    }
}

.home-container {
    overflow-x: hidden;

    .home-card-item {
        width: 100%;
        height: 103px;
        background: gray;
        border-radius: 4px;
        transition: all ease 0.3s;
        cursor: pointer;

        &:hover {
            box-shadow: 0 2px 12px 0 rgb(0 0 0 / 10%);
            transition: all ease 0.3s;
        }
    }

    .home-card-item-box {
        display: flex;
        align-items: center;
        position: relative;
        overflow: hidden;

        &:hover {
            i {
                right: 0px !important;
                bottom: 0px !important;
                transition: all ease 0.3s;
            }
        }

        i {
            position: absolute;
            right: -10px;
            bottom: -10px;
            font-size: 70px;
            transform: rotate(-30deg);
            transition: all ease 0.3s;
        }

        .home-card-item-flex {
            padding: 0 20px;
            color: white;

            .home-card-item-title,
            .home-card-item-tip {
                font-size: 13px;
            }

            .home-card-item-title-num {
                font-size: 2vw;
            }

            .home-card-item-tip-num {
                font-size: 13px;
            }
        }
    }
}
</style>
