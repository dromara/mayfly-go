<template>
    <el-input v-model="cron" placeholder="可点击左边按钮配置">
        <template #prepend>
            <el-button @click="showCron = true" icon="Pointer"></el-button>
        </template>
    </el-input>
    <el-dialog title="生成 cron" v-model="showCron" width="600px">
        <Crontab :expression="cron" @hide="showCron = false" @fill="(ex: any) => (cron = ex)" />
    </el-dialog>
</template>

<script lang="ts" setup>
import { ref } from 'vue';
import Crontab from '@/components/crontab/Crontab.vue';

const cron = defineModel<string>('modelValue', { required: true });

const showCron = ref(false);
</script>
