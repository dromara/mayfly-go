/**
 * 资源树 key 协议（单一事实源，零依赖纯函数）
 *
 * 分组节点 key 的产出方（resource.ts groupByResourceType）与消费方（locateResource.ts codePath 定位解析）共用此协议，
 * 避免格式在两处手写产生漂移。单独成模块（不依赖 apis/contributors）以便定位解析逻辑纯函数化、可脱离运行时单测。
 */

/** 类型分组节点 key：标签路径 + 资源类型（如 `tag1/-1`） */
export const resGroupKey = (tagPath: string, resourceType: number | string) => `${tagPath}-${resourceType}`;
