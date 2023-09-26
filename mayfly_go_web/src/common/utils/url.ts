const mode = import.meta.env.VITE_ROUTER_MODE;

/**
 * @description 获取不同路由模式所对应的 url
 * @returns {String}
 */
export function getNowUrl() {
    const url = {
        hash: location.hash.substring(1),
        history: location.pathname + location.search,
    };
    return url[mode];
}
