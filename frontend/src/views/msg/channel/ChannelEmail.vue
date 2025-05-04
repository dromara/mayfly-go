<template>
    <el-form-item prop="extra.smtpAccount" :rules="[Rules.requiredInput('msg.smtpAccount')]" :label="$t('msg.smtpAccount')">
        <el-input v-model.trim="extra.smtpAccount" auto-complete="off" clearable></el-input>
    </el-form-item>

    <el-form-item prop="extra.smtpPassword" :rules="[Rules.requiredInput('msg.smtpPassword')]" :label="$t('msg.smtpPassword')">
        <el-input v-model.trim="extra.smtpPassword" auto-complete="off" clearable></el-input>
    </el-form-item>
</template>

<script lang="ts" setup>
import { Rules } from '@/common/rule';

const extra = defineModel<any>('extra', { default: {} });
</script>
<style lang="scss"></style>
