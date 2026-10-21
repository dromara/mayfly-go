<template>
    <div class="personal-page">
        <div class="personal-layout">
            <!-- 左：身份名片 -->
            <el-card shadow="never" class="profile-card">
                <div class="profile-card__decor" aria-hidden="true"></div>
                <div class="profile-card__body">
                    <UserAvatarUpload :username="userInfo.username || ''" :name="userInfo.name" :photo="userInfo.photo" :size="88" />

                    <h1 class="profile-card__name">{{ userInfo.name || userInfo.username || '-' }}</h1>
                    <div class="profile-card__account">@{{ userInfo.username || '-' }}</div>

                    <div class="profile-card__roles">
                        <span v-for="role of state.accountInfo.roles" :key="role.roleId" class="role-pill" :class="{ 'is-disabled': role.status !== 1 }">
                            {{ role.roleName }}
                        </span>
                        <el-tooltip v-if="state.accountInfo.roles.length === 0" :content="$t('personal.noRoleHint')" placement="top">
                            <span class="profile-card__empty">{{ $t('personal.noRole') }}</span>
                        </el-tooltip>
                    </div>

                    <div class="profile-card__divider"></div>

                    <dl class="profile-card__meta">
                        <div class="meta-row">
                            <dt>
                                <SvgIcon name="Clock" :size="13" />
                                {{ $t('personal.lastLoginAt') }}
                            </dt>
                            <dd>{{ formatDate(userInfo.lastLoginTime, 'YYYY-MM-DD HH:mm') || '-' }}</dd>
                        </div>
                        <div class="meta-row">
                            <dt>
                                <SvgIcon name="Link" :size="13" />
                                {{ $t('personal.loginIp') }}
                            </dt>
                            <dd>{{ userInfo.lastLoginIp || '-' }}</dd>
                        </div>
                        <div class="meta-row">
                            <dt>
                                <SvgIcon name="Key" :size="13" />
                                {{ $t('personal.permissionCount') }}
                                <el-tooltip :content="$t('personal.permissionCountHint')" placement="top">
                                    <SvgIcon name="QuestionFilled" class="meta-row__hint" :size="12" />
                                </el-tooltip>
                            </dt>
                            <dd>{{ permissionCount }} {{ $t('personal.permissionUnit') }}</dd>
                        </div>
                    </dl>
                </div>
            </el-card>

            <!-- 右：安全设置 -->
            <SectionCard :title="$t('personal.securitySettings')" icon="Lock" class="security-card">
                <div class="security-block">
                    <div class="security-block__head">
                        <h2 class="security-block__title">{{ $t('personal.changePassword') }}</h2>
                        <p class="security-block__desc">{{ $t('login.passwordRuleTip') }}</p>
                    </div>

                    <el-form ref="pwdFormRef" :model="pwdForm" :rules="pwdRules" label-position="top" class="pwd-form" @submit.prevent>
                        <div class="pwd-form__grid">
                            <el-form-item :label="$t('personal.newPassword')" prop="password">
                                <el-input
                                    v-model="pwdForm.password"
                                    type="password"
                                    show-password
                                    autocomplete="new-password"
                                    :placeholder="$t('personal.inputNewPasswordPlaceholder')"
                                />
                            </el-form-item>
                            <el-form-item :label="$t('personal.confirmPassword')" prop="confirmPassword">
                                <el-input
                                    v-model="pwdForm.confirmPassword"
                                    type="password"
                                    show-password
                                    autocomplete="new-password"
                                    :placeholder="$t('personal.confirmPasswordPlaceholder')"
                                />
                            </el-form-item>
                        </div>
                        <el-button type="primary" icon="Check" :loading="saving" @click="submitPassword">{{ $t('personal.updatePassword') }}</el-button>
                    </el-form>
                </div>

                <div v-if="state.authStatus.enable" class="security-divider"></div>

                <div v-if="state.authStatus.enable" class="oauth-block">
                    <div class="oauth-block__text">
                        <div class="oauth-block__title">OAuth2</div>
                        <div class="oauth-block__desc">{{ $t('personal.oauth2Desc') }}</div>
                    </div>
                    <div class="oauth-block__side">
                        <span class="status-pill" :class="state.authStatus.bind ? 'is-on' : 'is-off'">
                            {{ state.authStatus.bind ? $t('personal.boundUp') : $t('personal.notBound') }}
                        </span>
                        <el-button v-if="!state.authStatus.bind" type="primary" link @click="bindOAuth2">{{ $t('personal.immediateBinding') }}</el-button>
                        <el-button v-else type="warning" link @click="unbindOAuth2">{{ $t('personal.unbundle') }}</el-button>
                    </div>
                </div>
            </SectionCard>
        </div>
    </div>
</template>

<script lang="ts" setup>
import config from '@/common/config';
import { joinClientParams } from '@/common/request';
import { formatDate } from '@/common/utils/format';
import { Rules } from '@/common/rule';
import SectionCard from '@/components/section-card/SectionCard.vue';
import SvgIcon from '@/components/svg-icon/index.vue';
import UserAvatarUpload from '@/components/user-avatar-upload/UserAvatarUpload.vue';
import { Msg } from '@/hooks/useI18n';
import { useUserInfo } from '@/store/userInfo';
import type { FormInstance, FormRules } from 'element-plus';
import { storeToRefs } from 'pinia';
import { computed, onMounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { personApi, type AccountPersonVO } from './api';
import { validatePasswordChange } from './validators';

defineOptions({
    name: 'Personal',
});

const { t } = useI18n();
const { userInfo } = storeToRefs(useUserInfo());

const state = reactive({
    accountInfo: { roles: [] } as AccountPersonVO,
    authStatus: {
        enable: false,
        bind: false,
    },
});

const pwdForm = reactive({
    password: '',
    confirmPassword: '',
});
const pwdFormRef = ref<FormInstance>();
const saving = ref(false);

const permissionCount = computed(() => userInfo.value.permissions?.length ?? 0);

// 与登录页改密共用同一份口令强度规则，避免两处规则漂移；确认密码只做一致性校验
const pwdRules: FormRules = {
    password: [Rules.requiredInput('personal.newPassword'), Rules.accountPassword],
    confirmPassword: [
        Rules.requiredInput('personal.confirmPassword'),
        {
            validator: (_rule, value: string, callback: (error?: Error) => void) => {
                if (value !== pwdForm.password) {
                    callback(new Error(t('personal.passwordMismatch')));
                    return;
                }
                callback();
            },
            trigger: 'blur',
        },
    ],
};

onMounted(() => {
    loadAccountInfo();
    loadAuthStatus();
});

const loadAccountInfo = async () => {
    try {
        state.accountInfo = (await personApi.accountInfo.request()) ?? { roles: [] };
    } catch (error) {
        console.error('Failed to load account info:', error);
    }
};

const loadAuthStatus = async () => {
    try {
        state.authStatus = await personApi.authStatus.request();
    } catch (error) {
        console.error('Failed to load oauth2 status:', error);
    }
};

const submitPassword = async () => {
    // 字段级错误提示交给 el-form（视觉反馈）；是否放行由 validatePasswordChange 兜底判定
    const formValid = pwdFormRef.value ? await pwdFormRef.value.validate().then(() => true).catch(() => false) : false;
    if (!formValid) {
        return;
    }

    // 表单校验链路异常时 EP 会静默放行（见 validators 注释），这里用同一份规则再判一次
    const result = validatePasswordChange(pwdForm);
    if (result === 'strength') {
        Msg.error(t('login.passwordRuleTip'));
        return;
    }
    if (result === 'mismatch') {
        Msg.error(t('personal.passwordMismatch'));
        return;
    }

    saving.value = true;
    try {
        await personApi.updateAccount.request({ password: pwdForm.password });
        Msg.success(t('personal.passwordUpdated'));
        pwdForm.password = '';
        pwdForm.confirmPassword = '';
        pwdFormRef.value?.clearValidate();
    } catch (error) {
        console.error('Failed to update password:', error);
    } finally {
        saving.value = false;
    }
};

const bindOAuth2 = () => {
    const width = 700;
    const height = 500;
    const iTop = (window.screen.height - 30 - height) / 2;
    const iLeft = (window.screen.width - 10 - width) / 2;
    // 小窗口打开 oauth2 鉴权，绑定结果由回调页 postMessage 通知
    const oauthWindow = window.open(
        `${config.baseApiUrl}/auth/oauth2/bind?${joinClientParams()}`,
        'oauth2',
        `height=${height},width=${width},top=${iTop},left=${iLeft},location=no`
    );
    if (!oauthWindow) {
        return;
    }

    let pollTimer = 0;
    const handler = (e: MessageEvent) => {
        if (e.data?.action === 'oauthBind') {
            window.removeEventListener('message', handler);
            window.clearInterval(pollTimer);
            Msg.success('personal.bindingSuccess');
            setTimeout(() => {
                location.reload();
            }, 1000);
        }
    };
    window.addEventListener('message', handler);
    // 用户直接关掉授权窗口时同步收尾，否则监听器与轮询会常驻页面
    pollTimer = window.setInterval(() => {
        if (oauthWindow.closed) {
            window.removeEventListener('message', handler);
            window.clearInterval(pollTimer);
        }
    }, 1000);
};

const unbindOAuth2 = async () => {
    try {
        await personApi.unbindOauth2.request();
        Msg.success('personal.unbundleSuccess');
        await loadAuthStatus();
    } catch (error) {
        console.error('Failed to unbind oauth2:', error);
    }
};
</script>

<style scoped lang="scss">
.personal-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
}

.personal-layout {
    display: grid;
    grid-template-columns: 340px minmax(0, 1fr);
    gap: 16px;
    align-items: stretch;
}

/* 身份名片：与首页 Hero 同一套渐变底，避免两个页面的"欢迎区"观感割裂 */
.profile-card {
    --el-card-border-radius: 18px;
    position: relative;
    height: 100%;
    overflow: hidden;
}

.profile-card__decor {
    position: absolute;
    inset: 0;
    pointer-events: none;
    background:
        radial-gradient(80% 60% at 20% 0%, color-mix(in srgb, var(--el-color-primary) 22%, transparent), transparent 65%),
        radial-gradient(70% 60% at 100% 10%, color-mix(in srgb, var(--el-color-success) 15%, transparent), transparent 70%);
}

.profile-card__body {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 8px 4px 4px;
    text-align: center;
}

.profile-card__name {
    margin: 14px 0 0;
    font-size: 20px;
    font-weight: 700;
    line-height: 1.3;
    letter-spacing: -0.01em;
    color: var(--el-text-color-primary);
}

.profile-card__account {
    margin-top: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
}

.profile-card__roles {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 6px;
    margin-top: 12px;
}

.role-pill {
    padding: 2px 10px;
    font-size: 11px;
    font-weight: 600;
    line-height: 18px;
    color: var(--el-color-primary);
    background: color-mix(in srgb, var(--el-color-primary) 13%, transparent);
    border: 1px solid color-mix(in srgb, var(--el-color-primary) 22%, transparent);
    border-radius: 999px;

    &.is-disabled {
        color: var(--el-text-color-placeholder);
        background: var(--el-fill-color);
        border-color: var(--el-border-color-lighter);
    }
}

.profile-card__empty {
    font-size: 12px;
    color: var(--el-text-color-placeholder);
}

.profile-card__divider {
    width: 100%;
    height: 1px;
    margin: 18px 0 14px;
    background: var(--el-border-color-extra-light);
}

.profile-card__meta {
    display: flex;
    flex-direction: column;
    gap: 10px;
    width: 100%;
    margin: 0;
}

.meta-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    font-size: 12px;

    dt {
        display: inline-flex;
        align-items: center;
        gap: 5px;
        flex-shrink: 0;
        color: var(--el-text-color-secondary);
    }

    dd {
        margin: 0;
        min-width: 0;
        font-weight: 500;
        font-variant-numeric: tabular-nums;
        color: var(--el-text-color-primary);
        overflow: hidden;
        white-space: nowrap;
        text-overflow: ellipsis;
    }
}

.meta-row__hint {
    color: var(--el-text-color-placeholder);
    cursor: help;
}

.security-block {
    padding: 20px 22px 22px;
}

.security-block__head {
    margin-bottom: 16px;
}

.security-block__title {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
}

.security-block__desc {
    margin: 6px 0 0;
    font-size: 12px;
    line-height: 1.5;
    color: var(--el-text-color-placeholder);
}

.pwd-form__grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0 16px;
    max-width: 640px;
}

.security-divider {
    height: 1px;
    margin: 0 22px;
    background: var(--el-border-color-extra-light);
}

.oauth-block {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 18px 22px;
}

.oauth-block__title {
    font-size: 13px;
    font-weight: 600;
    color: var(--el-text-color-primary);
}

.oauth-block__desc {
    margin-top: 4px;
    font-size: 12px;
    color: var(--el-text-color-placeholder);
}

.oauth-block__side {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-shrink: 0;
}

.status-pill {
    --pill-tone: var(--el-color-success);
    padding: 2px 10px;
    font-size: 11px;
    font-weight: 600;
    line-height: 18px;
    color: var(--pill-tone);
    background: color-mix(in srgb, var(--pill-tone) 12%, transparent);
    border-radius: 999px;

    &.is-off {
        --pill-tone: var(--el-text-color-placeholder);
        background: var(--el-fill-color);
    }
}

@media (max-width: 1100px) {
    .personal-layout {
        grid-template-columns: minmax(0, 1fr);
    }
}

@media (max-width: 640px) {
    .pwd-form__grid {
        grid-template-columns: minmax(0, 1fr);
    }

    .oauth-block {
        flex-direction: column;
        align-items: flex-start;
    }
}
</style>
