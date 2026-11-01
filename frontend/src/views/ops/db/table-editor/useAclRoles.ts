import { onMounted, ref } from 'vue';
import { roleApi } from '@/views/system/api';

/**
 * 角色下拉与角色名回显。
 *
 * 规则里存的是角色 id：界面直接显示 id 等于让管理员去别处对照，因此两个面板共用
 * 这一份「id → 名称」映射，而不是各自写一遍。
 */
export function useAclRoles() {
    const roles = ref<any[]>([]);

    onMounted(async () => {
        const res = await roleApi.list.request();
        roles.value = res.list;
    });

    const roleNames = (roleIds: number[]) =>
        roleIds.map((id) => roles.value.find((role) => role.id === id)?.name ?? String(id)).join('、');

    return { roles, roleNames };
}
