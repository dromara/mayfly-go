<template>
    <div class="personal">
        <el-row>
            <!-- 个人信息 -->
            <el-col :xs="24" :sm="16">
                <el-card shadow="hover" header="个人信息">
                    <div class="personal-user">
                        <div class="personal-user-left">
                            <el-upload
                                class="h100 personal-user-left-upload"
                                action="https://jsonplaceholder.typicode.com/posts/"
                                multiple
                                :limit="1"
                            >
                                <img :src="getUserInfos.photo" />
                            </el-upload>
                        </div>
                        <div class="personal-user-right">
                            <el-row>
                                <el-col :span="24" class="personal-title mb18"
                                    >{{ currentTime }}，admin，生活变的再糟糕，也不妨碍我变得更好！
                                </el-col>
                                <el-col :span="24">
                                    <el-row>
                                        <el-col :xs="24" :sm="8" class="personal-item mb6">
                                            <div class="personal-item-label">昵称：</div>
                                            <div class="personal-item-value">小柒</div>
                                        </el-col>
                                        <el-col :xs="24" :sm="16" class="personal-item mb6">
                                            <div class="personal-item-label">身份：</div>
                                            <div class="personal-item-value">超级管理</div>
                                        </el-col>
                                    </el-row>
                                </el-col>
                                <el-col :span="24">
                                    <el-row>
                                        <el-col :xs="24" :sm="8" class="personal-item mb6">
                                            <div class="personal-item-label">登录IP：</div>
                                            <div class="personal-item-value">192.168.1.1</div>
                                        </el-col>
                                        <el-col :xs="24" :sm="16" class="personal-item mb6">
                                            <div class="personal-item-label">登录时间：</div>
                                            <div class="personal-item-value">2021-02-05 18:47:26</div>
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
                        <span>消息通知</span>
                        <span class="personal-info-more">更多</span>
                    </template>
                    <div class="personal-info-box">
                        <ul class="personal-info-ul">
                            <li v-for="(v, k) in newsInfoList" :key="k" class="personal-info-li">
                                <a :href="v.link" target="_block" class="personal-info-li-title">{{ v.title }}</a>
                            </li>
                        </ul>
                    </div>
                </el-card>
            </el-col>

            <!-- 营销推荐 -->
            <el-col :span="24">
                <el-card shadow="hover" class="mt15" header="营销推荐">
                    <el-row :gutter="15" class="personal-recommend-row">
                        <el-col :sm="6" v-for="(v, k) in recommendList" :key="k" class="personal-recommend-col">
                            <div class="personal-recommend" :style="{ 'background-color': v.bg }">
                                <i :class="v.icon" :style="{ color: v.iconColor }"></i>
                                <div class="personal-recommend-auto">
                                    <div>{{ v.title }}</div>
                                    <div class="personal-recommend-msg">{{ v.msg }}</div>
                                </div>
                            </div>
                        </el-col>
                    </el-row>
                </el-card>
            </el-col>

            <!-- 更新信息 -->
            <el-col :span="24">
                <el-card shadow="hover" class="mt15 personal-edit" header="更新信息">
                    <div class="personal-edit-title">基本信息</div>
                    <el-form :model="personalForm" size="small" label-width="40px" class="mt35 mb35">
                        <el-row :gutter="35">
                            <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="4" class="mb20">
                                <el-form-item label="昵称">
                                    <el-input v-model="personalForm.name" placeholder="请输入昵称" clearable></el-input>
                                </el-form-item>
                            </el-col>
                            <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="4" class="mb20">
                                <el-form-item label="邮箱">
                                    <el-input v-model="personalForm.email" placeholder="请输入邮箱" clearable></el-input>
                                </el-form-item>
                            </el-col>
                            <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="4" class="mb20">
                                <el-form-item label="签名">
                                    <el-input v-model="personalForm.autograph" placeholder="请输入签名" clearable></el-input>
                                </el-form-item>
                            </el-col>
                            <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="4" class="mb20">
                                <el-form-item label="职业">
                                    <el-select v-model="personalForm.occupation" placeholder="请选择职业" clearable class="w100">
                                        <el-option label="计算机 / 互联网 / 通信" value="1"></el-option>
                                        <el-option label="生产 / 工艺 / 制造" value="2"></el-option>
                                        <el-option label="医疗 / 护理 / 制药" value="3"></el-option>
                                    </el-select>
                                </el-form-item>
                            </el-col>
                            <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="4" class="mb20">
                                <el-form-item label="手机">
                                    <el-input v-model="personalForm.phone" placeholder="请输入手机" clearable></el-input>
                                </el-form-item>
                            </el-col>
                            <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="4" class="mb20">
                                <el-form-item label="性别">
                                    <el-select v-model="personalForm.sex" placeholder="请选择性别" clearable class="w100">
                                        <el-option label="男" value="1"></el-option>
                                        <el-option label="女" value="2"></el-option>
                                    </el-select>
                                </el-form-item>
                            </el-col>
                            <el-col :xs="24" :sm="24" :md="24" :lg="24" :xl="24">
                                <el-form-item>
                                    <el-button type="primary" icon="el-icon-position">更新个人信息</el-button>
                                </el-form-item>
                            </el-col>
                        </el-row>
                    </el-form>
                    <div class="personal-edit-title mb15">账号安全</div>
                    <div class="personal-edit-safe-box">
                        <div class="personal-edit-safe-item">
                            <div class="personal-edit-safe-item-left">
                                <div class="personal-edit-safe-item-left-label">账户密码</div>
                                <div class="personal-edit-safe-item-left-value">当前密码强度：强</div>
                            </div>
                            <div class="personal-edit-safe-item-right">
                                <el-button type="text">立即修改</el-button>
                            </div>
                        </div>
                    </div>
                    <div class="personal-edit-safe-box">
                        <div class="personal-edit-safe-item">
                            <div class="personal-edit-safe-item-left">
                                <div class="personal-edit-safe-item-left-label">密保手机</div>
                                <div class="personal-edit-safe-item-left-value">已绑定手机：132****4108</div>
                            </div>
                            <div class="personal-edit-safe-item-right">
                                <el-button type="text">立即修改</el-button>
                            </div>
                        </div>
                    </div>
                    <div class="personal-edit-safe-box">
                        <div class="personal-edit-safe-item">
                            <div class="personal-edit-safe-item-left">
                                <div class="personal-edit-safe-item-left-label">密保问题</div>
                                <div class="personal-edit-safe-item-left-value">已设置密保问题，账号安全大幅度提升</div>
                            </div>
                            <div class="personal-edit-safe-item-right">
                                <el-button type="text">立即设置</el-button>
                            </div>
                        </div>
                    </div>
                    <div class="personal-edit-safe-box">
                        <div class="personal-edit-safe-item">
                            <div class="personal-edit-safe-item-left">
                                <div class="personal-edit-safe-item-left-label">绑定QQ</div>
                                <div class="personal-edit-safe-item-left-value">已绑定QQ：110****566</div>
                            </div>
                            <div class="personal-edit-safe-item-right">
                                <el-button type="text">立即设置</el-button>
                            </div>
                        </div>
                    </div>
                </el-card>
            </el-col>
        </el-row>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, computed } from 'vue';
import { formatAxis } from '@/common/utils/formatTime.ts';
import { newsInfoList, recommendList } from './mock.ts';
import { useStore } from '@/store/index.ts';
import { useRouter } from 'vue-router';
export default {
    name: 'personal',
    setup() {
        const store = useStore();
        const router = useRouter();
        const state = reactive({
            newsInfoList,
            recommendList,
            personalForm: {
                name: '',
                email: '',
                autograph: '',
                occupation: '',
                phone: '',
                sex: '',
            },
        });
        // 当前时间提示语
        const currentTime = computed(() => {
            return formatAxis(new Date());
        });

        // 获取用户信息 vuex
        const getUserInfos = computed(() => {
            return store.state.userInfos.userInfos;
        });

        return {
            getUserInfos,
            currentTime,
            ...toRefs(state),
        };
    },
};
</script>

<style scoped lang="scss">
@import '../../theme/mixins/mixins.scss';
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
                @include text-ellipsis(1);
            }
            .personal-item {
                display: flex;
                align-items: center;
                font-size: 13px;
                .personal-item-label {
                    color: gray;
                    @include text-ellipsis(1);
                }
                .personal-item-value {
                    @include text-ellipsis(1);
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
                color: var(--color-primary);
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
                        @include text-ellipsis(1);
                        color: grey;
                        text-decoration: none;
                    }
                    & a:hover {
                        color: var(--color-primary);
                        cursor: pointer;
                    }
                }
            }
        }
    }
    .personal-recommend-row {
        .personal-recommend-col {
            .personal-recommend {
                position: relative;
                height: 100px;
                color: #ffffff;
                border-radius: 3px;
                overflow: hidden;
                cursor: pointer;
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
                .personal-recommend-auto {
                    padding: 15px;
                    position: absolute;
                    left: 0;
                    top: 5%;
                    .personal-recommend-msg {
                        font-size: 12px;
                        margin-top: 10px;
                    }
                }
            }
        }
    }
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
                background: var(--color-primary);
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
