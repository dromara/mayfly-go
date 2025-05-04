<template>
    <el-form-item prop="extra.secret" :rules="[Rules.requiredInput('secret')]" :label="$t('Secret')">
        <el-input v-model.trim="extra.secret" auto-complete="off" clearable></el-input>
    </el-form-item>
</template>

<script lang="ts" setup>
import { Rules } from '@/common/rule';

const extra = defineModel<any>('extra', { default: {} });
</script>
<style lang="scss"></style>
