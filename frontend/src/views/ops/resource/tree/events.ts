/**
 * 资源树事件总线（零依赖的类型化发布订阅）：
 * 节点操作完成后只发布失效/定位事件，由容器订阅统一处理，
 * 替代 handler 点对点调用树 API（新增操作类型无需理解刷新范围）。
 */

type Handler<T> = (payload: T) => void;

/**
 * 事件作用域约定：target 缺省视为主资源树（'ops'）；
 * 容器通过 eventScope 声明自己的作用域，不匹配的事件忽略（防多容器并存互相误响应）。
 */
export const DEFAULT_TREE_SCOPE = 'ops';

export interface TreeInvalidateEvent {
    /** 待失效的节点 key，缺省为根（容器重载该节点子树） */
    key?: string;
    /** 目标容器作用域，缺省为主资源树 */
    target?: string;
}

export interface TreeLocateEvent {
    key: string;
    /** 目标容器作用域，缺省为主资源树 */
    target?: string;
}

export interface TreeEventMap {
    'node:invalidate': TreeInvalidateEvent;
    'node:locate': TreeLocateEvent;
}

class TypedEmitter<M extends object> {
    private handlers = new Map<keyof M, Set<Handler<never>>>();

    on<K extends keyof M>(event: K, handler: Handler<M[K]>): () => void {
        let set = this.handlers.get(event);
        if (!set) {
            set = new Set();
            this.handlers.set(event, set);
        }
        set.add(handler as Handler<never>);
        return () => this.off(event, handler);
    }

    off<K extends keyof M>(event: K, handler: Handler<M[K]>) {
        this.handlers.get(event)?.delete(handler as Handler<never>);
    }

    emit<K extends keyof M>(event: K, payload: M[K]) {
        this.handlers.get(event)?.forEach((h) => (h as Handler<M[K]>)(payload));
    }
}

export const treeEvents = new TypedEmitter<TreeEventMap>();
