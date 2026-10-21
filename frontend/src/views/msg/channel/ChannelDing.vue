<template>
    <!-- 标签与必填提示同一个真源（msg.secret）：之前标签走 $t('Secret')（字典里根本没这个键，
         界面直出「Secret」），提示又是 Rules.requiredInput('secret')（渲成「请输入secret」中英混排）；
         凭据字段默认遮文展示，不把它常驻在明文输入框里 -->
    <el-form-item prop="extra.secret" :rules="[Rules.requiredInput(t('msg.secret'))]" :label="t('msg.secret')">
        <el-input v-model.trim="extra.secret" auto-complete="off" show-password clearable></el-input>
    </el-form-item>
</template>

<script lang="ts" setup>
import { Rules } from '@/common/rule';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
const extra = defineModel<any>('extra', { default: () => ({}) });
</script>
<style lang="scss"></style>
