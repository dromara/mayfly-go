<template>
    <div class="personal">
        <el-row>
            <!-- 更新信息 -->
            <el-col :span="24">
                <el-card shadow="hover" class="mt15 personal-edit" header="更新信息">
                    <div class="personal-edit-title">基本信息</div>
                    <el-form :model="accountForm" label-width="auto" class="mt35 mb35">
                        <el-row :gutter="35">
                            <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="4" class="mb20">
                                <el-form-item label="密码">
                                    <el-input type="password" show-password v-model="accountForm.password" placeholder="请输入新密码" clearable></el-input>
                                </el-form-item>
                            </el-col>
                            <!--  -->
                            <el-col :xs="24" :sm="24" :md="24" :lg="24" :xl="24">
                                <el-form-item>
                                    <el-button @click="updateAccount" type="primary" icon="position">更新个人信息</el-button>
                                </el-form-item>
                            </el-col>
                        </el-row>
                    </el-form>

                    <span v-show="authStatus.enable">
                        <div class="personal-edit-title mb15">账号信息</div>
                        <div class="personal-edit-safe-box">
                            <div class="personal-edit-safe-item">
                                <div class="personal-edit-safe-item-left">
                                    <div class="personal-edit-safe-item-left-label">Oauth2</div>
                                    <div class="personal-edit-safe-item-left-value">当前状态：{{ authStatus.bind ? '已绑定' : '未绑定' }}</div>
                                </div>
                                <div class="personal-edit-safe-item-right">
                                    <el-button v-if="!authStatus.bind" link @click="bindOAuth2" type="primary">立即绑定</el-button>
                                    <el-button v-else link @click="unbindOAuth2()" type="warning">解绑</el-button>
                                </div>
                            </div>
                        </div>
                    </span>
                </el-card>
            </el-col>
        </el-row>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, onMounted } from 'vue';
import { ElMessage } from 'element-plus';
import { personApi } from './api';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';

const state = reactive({
    accountInfo: {
        roles: [],
    },
    recommendList: [],
    accountForm: {
        password: '',
    },
    authStatus: {
        enable: false,
        bind: false,
    },
});

const { accountForm, authStatus } = toRefs(state);

onMounted(async () => {
    getAccountInfo();
    state.authStatus = await personApi.authStatus.request();
});

const getAccountInfo = async () => {
    state.accountInfo = await personApi.accountInfo.request();
};

const updateAccount = async () => {
    await personApi.updateAccount.request(state.accountForm);
    ElMessage.success('更新成功');
};

const bindOAuth2 = () => {
    const width = 700;
    const height = 500;
    var iTop = (window.screen.height - 30 - height) / 2; //获得窗口的垂直位置;
    var iLeft = (window.screen.width - 10 - width) / 2; //获得窗口的水平位置;
    // 小窗口打开oauth2鉴权
    let oauthWindow = window.open(
        `${config.baseApiUrl}/auth/oauth2/bind?${joinClientParams()}`,
        'oauth2',
        `height=${height},width=${width},top=${iTop},left=${iLeft},location=no`
    );
    if (oauthWindow) {
        const handler = (e: any) => {
            if (e.data.action === 'oauthBind') {
                window.removeEventListener('message', handler);
                // 处理登录token
                ElMessage.success('绑定成功');
                setTimeout(() => {
                    location.reload();
                }, 1000);
            }
        };
        window.addEventListener('message', handler);
        setInterval(() => {
            if (oauthWindow!.closed) {
                window.removeEventListener('message', handler);
            }
        }, 1000);
    }
};

const unbindOAuth2 = async () => {
    await personApi.unbindOauth2.request();
    ElMessage.success('解绑成功');
    state.authStatus = await personApi.authStatus.request();
};
</script>

<style scoped lang="scss">
@import '../../theme/mixins/index.scss';
.personal {
    .personal-edit {
        .personal-edit-title {
            position: relative;
            padding-left: 10px;
            color: #606266;

            &::after {
                content: '';
                width: 2px;
                height: 10px;
                position: absolute;
                left: 0;
                top: 50%;
                transform: translateY(-50%);
                background: var(--el-color-primary);
            }
        }

        .personal-edit-safe-box {
            border-bottom: 1px solid #ebeef5;
            padding: 15px 0;

            .personal-edit-safe-item {
                width: 100%;
                display: flex;
                align-items: center;
                justify-content: space-between;

                .personal-edit-safe-item-left {
                    flex: 1;
                    overflow: hidden;

                    .personal-edit-safe-item-left-label {
                        color: #606266;
                        margin-bottom: 5px;
                    }

                    .personal-edit-safe-item-left-value {
                        color: gray;
                        @include text-ellipsis(1);
                        margin-right: 15px;
                    }
                }
            }

            &:last-of-type {
                padding-bottom: 0;
                border-bottom: none;
            }
        }
    }
}
</style>
