<template>
    <div class="personal">
        <el-row>
            <!-- 更新信息 -->
            <el-col :span="24">
                <el-card shadow="hover" class="!mt-3.5 personal-edit" :header="$t('personal.updateInfo')">
                    <div class="personal-edit-title">{{ $t('personal.basicInfo') }}</div>
                    <el-form :model="accountForm" label-width="auto" class="mt-8 mb-8">
                        <el-row :gutter="35">
                            <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="4" class="!mb-4">
                                <el-form-item :label="$t('common.password')">
                                    <el-input
                                        type="password"
                                        show-password
                                        v-model="accountForm.password"
                                        :placeholder="$t('personal.inputNewPasswordPlaceholder')"
                                        clearable
                                    ></el-input>
                                </el-form-item>
                            </el-col>
                            <!--  -->
                            <el-col :xs="24" :sm="24" :md="24" :lg="24" :xl="24">
                                <el-form-item>
                                    <el-button @click="updateAccount" type="primary" icon="position">{{ $t('personal.updatePersonalInfo') }}</el-button>
                                </el-form-item>
                            </el-col>
                        </el-row>
                    </el-form>

                    <span v-show="authStatus.enable">
                        <div class="personal-edit-title mb-2">{{ $t('personal.accountInfo') }}</div>
                        <div class="personal-edit-safe-box">
                            <div class="personal-edit-safe-item">
                                <div class="personal-edit-safe-item-left">
                                    <div class="personal-edit-safe-item-left-label">Oauth2</div>
                                    <div class="personal-edit-safe-item-left-value">
                                        {{ $t('personal.currentStatus') }}：{{ authStatus.bind ? $t('personal.boundUp') : $t('personal.notBound') }}
                                    </div>
                                </div>
                                <div class="personal-edit-safe-item-right">
                                    <el-button v-if="!authStatus.bind" link @click="bindOAuth2" type="primary">{{ $t('personal.immediateBinding') }}</el-button>
                                    <el-button v-else link @click="unbindOAuth2()" type="warning">{{ $t('personal.unbundle') }}</el-button>
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
import { useI18n } from 'vue-i18n';

defineOptions({
    name: 'Personal',
});

const { t } = useI18n();

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
    ElMessage.success(t('personal.updateSuccess'));
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
                ElMessage.success(t('personal.bindingSuccess'));
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
    ElMessage.success(t('personal.unbundleSuccess'));
    state.authStatus = await personApi.authStatus.request();
};
</script>

<style scoped lang="scss">
@use '@/theme/mixins/index.scss' as mixins;
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
                        @include mixins.text-ellipsis(1);
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
