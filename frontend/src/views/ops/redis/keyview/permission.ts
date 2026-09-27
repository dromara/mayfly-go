/**
 * redis 数据面权限码。
 *
 * 后端接口与前端按钮共用同一份码值：这里只声明常量，权限判定仍由 `v-auth` 与
 * `hasPerm` 负责，避免出现「后端另有一套字面量」的漂移
 */
export const PERM_DATA_SAVE = 'redis:data:save';
export const PERM_DATA_DEL = 'redis:data:del';
