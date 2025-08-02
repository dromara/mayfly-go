<template>
    <div class="rounded-xl shadow-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 overflow-hidden w-full">
        <!-- Header -->
        <div class="flex items-center justify-between px-5 py-4 border-b border-gray-100 dark:border-gray-700">
            <h3 class="font-semibold text-lg text-gray-800 dark:text-gray-100 flex items-center">
                <SvgIcon class="mr-2" name="Bell" :size="16" />
                {{ $t('layout.user.newTitle') }}
            </h3>
            <el-badge :value="unreadCount" :max="99" :hidden="unreadCount === 0" type="primary">
                <el-button v-if="unreadCount > 0" size="small" type="primary" link @click="onRead()" class="text-sm">
                    {{ $t('layout.user.newBtn') }}
                </el-button>
            </el-badge>
        </div>

        <!-- Content -->
        <el-scrollbar height="360px" v-loading="loadingMsgs" class="px-3 py-2" :class="{ 'py-8': msgs.length === 0 }">
            <template v-if="msgs.length > 0">
                <div
                    v-for="(v, k) in msgs"
                    :key="k"
                    class="px-3 py-3 my-1 rounded-lg transition-all duration-200 cursor-pointer hover:shadow-sm hover:bg-gray-100 dark:hover:bg-gray-700"
                    :class="{
                        ' hover:bg-gray-100  dark:hover:bg-gray-200 border border-blue-100 dark:border-blue-800/50': v.status == -1,
                        'bg-gray-50 hover:bg-gray-100 dark:bg-gray-600/20 dark:hover:bg-gray-200 border border-transparent': v.status == 1,
                    }"
                    @click="onRead(v)"
                >
                    <div class="flex justify-between items-start">
                        <el-tag
                            size="small"
                            :type="EnumValue.getEnumByValue(MsgSubtypeEnum, v.subtype)?.extra?.notifyType || 'info'"
                            effect="light"
                            class="rounded-full"
                        >
                            {{ $t(EnumValue.getEnumByValue(MsgSubtypeEnum, v.subtype)?.label || '') }}
                        </el-tag>
                        <el-text size="small" type="info" class="text-xs whitespace-nowrap ml-2">
                            {{ formatDate(v.createTime) }}
                        </el-text>
                    </div>
                    <div class="mt-2 text-gray-700 dark:text-gray-300 text-sm leading-relaxed">
                        <MessageRenderer :content="v.msg" size="small" />
                    </div>
                </div>

                <div class="text-center py-3" v-if="!loadMoreDisable">
                    <el-button link type="primary" size="small" @click="loadMsgs()">
                        {{ $t('redis.loadMore') }}
                        <SvgIcon name="ArrowDown" />
                    </el-button>
                </div>
            </template>

            <div v-else-if="!loadingMsgs" class="text-center py-6">
                <SvgIcon name="ChatLineRound" :size="36" class="mb-3 text-gray-300 dark:text-gray-600" />
                <p class="text-gray-500 dark:text-gray-400 text-2xl">{{ $t('layout.user.newDesc') }}</p>
            </div>
        </el-scrollbar>
    </div>
</template>

<script lang="ts" setup>
import { MsgSubtypeEnum } from '@/common/commonEnum';
import EnumValue from '@/common/Enum';
import { formatDate } from '@/common/utils/format';
import { MessageRenderer } from '@/components/message/message';
import { personApi } from '@/views/personal/api';
import { useIntervalFn } from '@vueuse/core';
import { onMounted, ref, watchEffect } from 'vue';

const emit = defineEmits(['update:count']);

const msgQuery = ref({
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

    if (!msg) {
        loadMsgs(true);
        // 如果是全部已读，重置未读消息数
        unreadCount.value = 0;
    } else {
        msg.status = 1;
        // 如果是单条已读，减少未读消息数
        unreadCount.value = Math.max(unreadCount.value - 1, 0);
    }
};

defineExpose({
    loadMsgs,
    clearMsg: function () {
        msgQuery.value.pageNum = 1;
        msgs.value = [];
        loadingMsgs.value = true;
    },
});

const toMsgCenter = () => {};
</script>

<style scoped lang="scss">
:deep(.el-scrollbar__view) {
    padding-left: 0.5rem;
    padding-right: 0.5rem;
}

:deep(.el-tag) {
    border: none;
}
</style>
