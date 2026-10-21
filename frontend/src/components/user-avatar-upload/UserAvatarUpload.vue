<template>
    <el-upload
        class="user-avatar-upload"
        :style="{ '--avatar-size': `${size}px` }"
        :action="getUploadFileUrl(`avatar_${username}`)"
        :limit="1"
        :show-file-list="false"
        :before-upload="beforeUpload"
        :on-success="handleSuccess"
        accept=".png,.jpg,.jpeg"
    >
        <img v-if="photo && !imageFailed" :src="photo" class="user-avatar-upload__img" :alt="name || username" @error="imageFailed = true" />
        <span v-else class="user-avatar-upload__fallback">{{ avatarLetter }}</span>
        <span class="user-avatar-upload__mask">
            <SvgIcon name="Camera" :size="16" />
        </span>
    </el-upload>
</template>

<script lang="ts" setup>
import { getFileUrl, getUploadFileUrl } from '@/common/request';
import { saveUser } from '@/common/utils/storage';
import SvgIcon from '@/components/svg-icon/index.vue';
import { Msg } from '@/hooks/useI18n';
import { useUserInfo } from '@/store/userInfo';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

/**
 * 用户头像上传：登录信息里的头像与首页/个人中心共用同一份写入路径。
 * 抽成组件是因为两处都要做「512KB 限制 + 上传后刷新 store 与本地缓存」，各写一份必然漂移
 */
const props = withDefaults(
    defineProps<{
        username: string;
        name?: string;
        photo?: string;
        /** 头像直径（px） */
        size?: number;
    }>(),
    {
        name: '',
        photo: '',
        size: 64,
    }
);

const { t } = useI18n();
const { userInfo } = useUserInfo();

const avatarLetter = computed(() => (props.name || props.username || '?').slice(0, 1).toUpperCase());

// 头像文件被删除或存储不可达时回退到字母头像，不展示破图；photo 变了（如重新上传）重试加载
const imageFailed = ref(false);
watch(
    () => props.photo,
    () => {
        imageFailed.value = false;
    }
);

const beforeUpload = (rawFile: File) => {
    if (rawFile.size >= 512 * 1024) {
        Msg.error(t('personal.avatarTooLarge'));
        return false;
    }
    return true;
};

const handleSuccess = (_response: unknown, uploadFile: import('element-plus').UploadFile) => {
    if (uploadFile.raw) {
        // 先给本地 blob 预览，随后以服务端回读地址为准（blob 地址刷新后失效）
        userInfo.photo = URL.createObjectURL(uploadFile.raw);
    }
    userInfo.photo = getFileUrl(`avatar_${props.username}`);
    saveUser({ ...userInfo });
};
</script>

<style scoped lang="scss">
.user-avatar-upload {
    position: relative;
    display: block;
    width: var(--avatar-size, 64px);
    height: var(--avatar-size, 64px);
    border-radius: 50%;
    overflow: hidden;
    cursor: pointer;
    background: color-mix(in srgb, var(--el-color-primary) 16%, transparent);
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--el-color-primary) 14%, transparent);
}

.user-avatar-upload__img {
    position: absolute;
    inset: 0;
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
}

.user-avatar-upload__fallback {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
    font-size: calc(var(--avatar-size, 64px) * 0.4);
    font-weight: 600;
    color: #fff;
    background: linear-gradient(135deg, var(--el-color-primary), color-mix(in srgb, var(--el-color-primary) 60%, #10b981));
}

.user-avatar-upload__mask {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #fff;
    background: rgb(0 0 0 / 42%);
    opacity: 0;
    transition: opacity 0.2s ease;
    pointer-events: none;
}

.user-avatar-upload:hover .user-avatar-upload__mask {
    opacity: 1;
}

@media (prefers-reduced-motion: reduce) {
    .user-avatar-upload__mask {
        transition: none;
    }
}
</style>
