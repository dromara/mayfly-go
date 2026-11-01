<template>
    <el-dialog v-model="visible" :title="$t('db.tableAcl.title')" width="780px" append-to-body destroy-on-close>
        <el-tabs v-model="activeTab">
            <el-tab-pane :label="$t('db.tableAcl.tabTable')" name="table">
                <DbTableAclPanel :db-id="dbId" :db-name="dbName" />
            </el-tab-pane>
            <el-tab-pane :label="$t('db.tableAcl.tabColumn')" name="column">
                <DbColumnAclPanel :db-id="dbId" :db-name="dbName" />
            </el-tab-pane>
        </el-tabs>
    </el-dialog>
</template>

<script lang="ts" setup>
import { defineAsyncComponent, ref } from 'vue';

defineProps<{ dbId: number; dbName: string }>();
const visible = defineModel<boolean>('visible', { default: false });

// 两个面板都只在打开弹窗后加载：低频动作，不必占用表清单页的首次渲染
const DbTableAclPanel = defineAsyncComponent(() => import('./DbTableAclPanel.vue'));
const DbColumnAclPanel = defineAsyncComponent(() => import('./DbColumnAclPanel.vue'));

// 打开时停在表保护页签：先去配表、再配列是绝大多数人的顺序
const activeTab = ref('table');
</script>
