/**
 * DB 资源树 - 模块入口
 * 命令注册 → commands.ts，贡献者注册 → contributors.ts
 * 共享常量与辅助函数 → helpers.ts（避免循环依赖）
 */
import { defineAsyncComponent } from 'vue';
import { ResourceTypeEnum } from '@/common/commonEnum';
import { defineResourceConfig } from '@/views/ops/resource/resourceRegistry';
import { DbIcon } from './helpers';

const DbInstList = defineAsyncComponent(() => import('../pages/InstanceList.vue'));

// 副作用导入：注册命令与贡献者
import './commands';
import './contributors';

// ---------------------------------- 默认导出 ----------------------------------

export default defineResourceConfig({
    order: 2,
    resourceType: ResourceTypeEnum.Db.value,
    manager: {
        componentConf: {
            component: DbInstList,
            icon: DbIcon,
            name: 'tag.db',
        },
        countKey: 'db',
        permCode: 'db:instance',
    },
});
