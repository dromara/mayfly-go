<template>
    <el-input v-model="cron" :placeholder="$t('components.crontab.crontabInputPlaceholder')">
        <template #prepend>
            <el-button @click="showCron = true" icon="Pointer"></el-button>
        </template>
    </el-input>
    <el-dialog :title="$t('components.crontab.crontabTitle')" v-model="showCron" width="660px">
        <Crontab :expression="cron" @hide="showCron = false" @fill="(ex: any) => (cron = ex)" />
    </el-dialog>
</template>

<script lang="ts" setup>
import { ref } from 'vue';
import Crontab from '@/components/crontab/Crontab.vue';

const cron = defineModel<string>('modelValue', { required: true });

const showCron = ref(false);
</script>
