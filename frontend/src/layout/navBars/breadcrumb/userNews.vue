<template>
    <div class="flex flex-col w-full rounded-md shadow-sm">
        <!-- Header -->
        <div class="flex items-center justify-between border-b border-gray-200 px-4 py-2 text-sm text-gray-700">
            <div class="font-medium">{{ $t('layout.user.newTitle') }}</div>
            <div v-if="unreadCount > 0" class="color-primary cursor-pointer opacity-80 transition-opacity hover:opacity-100" @click="onRead()">
                {{ $t('layout.user.newBtn') }}
            </div>
        </div>

        <!-- Content -->
        <el-scrollbar height="350px" v-loading="loadingMsgs" class="px-4 py-2 text-sm">
            <template v-if="msgs.length > 0">
                <div
                    v-for="(v, k) in msgs"
                    :key="k"
                    class="pt-1 mt-0.5"
                    :style="{ backgroundColor: v.status == 1 ? 'var(--el-color-info-light-9)' : 'transparent' }"
                    @click="onRead(v)"
                >
                    <div class="flex justify-between items-start">
                        <el-text size="small" tag="b" :type="EnumValue.getEnumByValue(MsgSubtypeEnum, v.subtype)?.extra?.notifyType">
                            {{ $t(EnumValue.getEnumByValue(MsgSubtypeEnum, v.subtype)?.label || '') }}
                        </el-text>
                    </div>
                    <div class="text-gray-500 mt-1 mb-1">{{ v.msg }}</div>
                    <div class="text-gray-500">{{ formatDate(v.createTime) }}</div>
                    <div class="mt-2 border-t border-gray-200"></div>
                </div>

                <el-button class="w-full mt-1" size="small" @click="loadMsgs()" v-if="!loadMoreDisable"> {{ $t('redis.loadMore') }} </el-button>
            </template>

            <el-empty v-if="msgs.length == 0 && !loadingMsgs" :image-size="100" :description="$t('layout.user.newDesc')" />
        </el-scrollbar>

        <!-- Footer -->
        <!-- <div
            v-if="msgs.length > 0"
            class="color-primary flex h-9 items-center justify-center border-t border-gray-200 text-sm cursor-pointer opacity-80 transition-opacity hover:opacity-100"
            @click="toMsgCenter"
        >
            {{ $t('layout.user.newGo') }}
        </div> -->
    </div>
</template>

<script lang="ts" setup>
import { MsgSubtypeEnum } from '@/common/commonEnum';
import EnumValue from '@/common/Enum';
import { formatDate } from '@/common/utils/format';
import { personApi } from '@/views/personal/api';
import { useIntervalFn } from '@vueuse/core';
import { onMounted, ref, watchEffect } from 'vue';

const emit = defineEmits(['update:count']);

const msgQuery = ref<any>({
    pageNum: 1,
    pageSize: 10,
});

const loadMoreDisable = ref(true);
const loadingMsgs = ref(true);
const msgs = ref<Array<any>>([]);
const unreadCount = ref(0);

onMounted(() => {
    useIntervalFn(
        () => {
            // 定时更新未读消息数
            personApi.getUnreadMsgCount.request().then((res) => {
                unreadCount.value = res;
            });
        },
        10 * 1000,
        { immediate: true, immediateCallback: true }
    );
});

watchEffect(() => {
    emit('update:count', unreadCount.value);
});

const loadMsgs = async (research: boolean = false) => {
    if (research) {
        msgQuery.value.pageNum = 1;
        msgs.value = [];
    }
    const msgList = await getMsgs();
    msgs.value.push(...msgList.list);
    msgQuery.value.pageNum += 1;

    loadMoreDisable.value = msgList.total <= msgs.value.length;
};

const getMsgs = async () => {
    try {
        loadingMsgs.value = true;
        return await personApi.getMsgs.request(msgQuery.value);
    } catch (e) {
        //
    } finally {
        loadingMsgs.value = false;
    }
};

const onRead = async (msg: any = null) => {
    if (msg && (msg.status == 1 || !msg.status)) {
        return;
    }

    await personApi.readMsg.request({ id: msg?.id || 0 });
    loadMsgs(true);

    if (!msg) {
        // 如果是全部已读，重置未读消息数
        unreadCount.value = 0;
    } else {
        // 如果是单条已读，减少未读消息数
        unreadCount.value = Math.max(unreadCount.value - 1, 0);
    }
};

defineExpose({
    loadMsgs,
});

const toMsgCenter = () => {};
</script>

<style scoped lang="scss"></style>
