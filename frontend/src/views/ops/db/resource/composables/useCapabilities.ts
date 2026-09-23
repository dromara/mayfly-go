import { DbInst } from '../../db';
import type { NegotiatedCapabilities } from '../../dialect/registry';

// 方言能力协商：统一走 DbInst 按库缓存（首次请求 /capabilities 并协商，后续复用）。
//
// 协商语义：前端方言静态声明（天花板）∩ 后端实际能力（约束），
// 返回 NegotiatedCapabilities 同时携带 backendFeatures（数据驱动渲染）与 namespace（命名空间层次）。
//
// 不做跨会话缓存：capabilities 是实例方言/版本的静态声明，但可能被【其他用户/会话】改实例类型或数据库升级而改变；
// 客户端跨会话缓存无法被他人失效（本地清除只清本标签页），会造成陈旧渲染。
// DbInst 缓存为进程级（标签页生命周期），标签页刷新即清空，保证每次打开取最新。
export async function getDbCapabilities(id: number, db: string): Promise<NegotiatedCapabilities> {
    // 资源树按库节点懒加载能力时，该 DbInst 可能尚未被缓存（树展开路径不像打开表工作区那样预先建实例）。
    // 必须用 getInstA（缓存未命中则回源获取并缓存），否则用严格的 getInst 会在缺失时直接抛错，
    // 导致展开任意物理库节点都「loadChildren failed / 重试」。
    const inst = await DbInst.getInstA(id);
    return await inst.loadCapabilities(db);
}
