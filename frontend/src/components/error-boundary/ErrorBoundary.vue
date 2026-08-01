<script lang="ts" setup>
import { ref, onErrorCaptured } from 'vue';

const props = defineProps<{ fallbackTitle?: string }>();
const error = ref<Error | null>(null);

onErrorCaptured((err: Error) => {
    error.value = err;
    return false;
});

const retry = () => {
    error.value = null;
};
</script>

<template>
    <slot v-if="!error" />
    <div v-else class="error-boundary">
        <el-result icon="error" :title="fallbackTitle || 'Error'" :sub-title="error.message">
            <template #extra>
                <el-button type="primary" @click="retry">Retry</el-button>
            </template>
        </el-result>
    </div>
</template>

<style scoped>
.error-boundary {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 200px;
}
</style>
