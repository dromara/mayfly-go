<template>
    <div v-if="props.authCerts">
        <el-select value-key="name" v-model="selectAuthCert" size="small">
            <el-option v-for="item in props.authCerts" :key="item.name" :label="item.username" :value="item">
                {{ item.username }}
                <el-divider direction="vertical" border-style="dashed" />
                <EnumTag :value="item.type" :enums="AuthCertTypeEnum" />
                <el-divider direction="vertical" border-style="dashed" />
                <EnumTag :value="item.ciphertextType" :enums="AuthCertCiphertextTypeEnum" />
            </el-option>
        </el-select>
    </div>
</template>

<script lang="ts" setup>
import EnumTag from '@/components/enumtag/EnumTag.vue';
import { AuthCertTypeEnum, AuthCertCiphertextTypeEnum } from '../tag/enums';
const props = defineProps({
    authCerts: {
        type: [Array<any>],
        required: true,
        default: () => [],
    },
});

const selectAuthCert = defineModel('selectAuthCert');
</script>

<style lang="scss"></style>
