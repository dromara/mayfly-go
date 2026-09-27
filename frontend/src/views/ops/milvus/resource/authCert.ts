// 当前选中的 Milvus 授权凭证名称
// 独立文件避免 api.ts 与 store.ts 之间的循环依赖
export let currentAcName = '';

export function setCurrentAcName(name: string) {
    currentAcName = name;
}
