import { useApiFetch } from '../hooks/useRequest';
import Api from './Api';
import config from './config';
import { getClientId, getToken } from './utils/storage';

export default {
    request,
    get,
    post,
    put,
    del,
    getApiUrl,
};

export interface Result<T = unknown> {
    /**
     * 响应码
     */
    code: number;
    /**
     * 响应消息
     */
    msg: string;
    /**
     * 数据
     */
    data?: T;
}

export enum ResultEnum {
    SUCCESS = 200,
    ERROR = 400,
    PARAM_ERROR = 405,
    SERVER_ERROR = 500,
    NO_PERMISSION = 501,
    ACCESS_TOKEN_INVALID = 502, // accessToken失效
}

export const baseUrl: string = config.baseApiUrl;
// const baseUrl: string = 'http://localhost:18888/api';
// const baseWsUrl: string = config.baseWsUrl;

/**
 * fetch请求url
 *
 * 该方法已处理请求结果中code != 200的message提示,如需其他错误处理(取消加载状态,重置对象状态等等),可catch继续处理
 *
 * @param {Object} method 请求方法(GET,POST,PUT,DELTE等)
 * @param {Object} uri    uri
 * @param {Object} params 参数
 */
async function request<T = unknown>(method: string, url: string, params: Record<string, unknown> | object | null = null, options: Record<string, unknown> = {}): Promise<T> {
    const { execute, data } = useApiFetch(Api.create<T>(url, method) as any, params, options);
    await execute();
    return data.value as T;
}

/**
 * get请求uri
 * 该方法已处理请求结果中code != 200的message提示,如需其他错误处理(取消加载状态,重置对象状态等等),可catch继续处理
 *
 * @param {Object} url   uri
 * @param {Object} params 参数
 */
function get<T = unknown>(url: string, params: Record<string, unknown> | object | null = null, options: Record<string, unknown> = {}): Promise<T> {
    return request<T>('get', url, params, options);
}

function post<T = unknown>(url: string, params: Record<string, unknown> | object | null = null, options: Record<string, unknown> = {}): Promise<T> {
    return request<T>('post', url, params, options);
}

function put<T = unknown>(url: string, params: Record<string, unknown> | object | null = null, options: Record<string, unknown> = {}): Promise<T> {
    return request<T>('put', url, params, options);
}

function del<T = unknown>(url: string, params: Record<string, unknown> | object | null = null, options: Record<string, unknown> = {}): Promise<T> {
    return request<T>('delete', url, params, options);
}

function getApiUrl(url: string) {
    // 只是返回api地址而不做请求，用在上传组件之类的
    return baseUrl + url + '?' + joinClientParams();
}

/**
 * 创建 websocket
 */
export const createWebSocket = (url: string): Promise<WebSocket> => {
    return new Promise<WebSocket>((resolve, reject) => {
        const clientParam = (url.includes('?') ? '&' : '?') + joinClientParams();
        const socket = new WebSocket(`${config.baseWsUrl}${url}${clientParam}`);

        socket.onopen = () => {
            resolve(socket);
        };

        socket.onerror = (e) => {
            reject(e);
        };
    });
};

// 组装客户端参数，包括 token 和 clientId
export function joinClientParams(): string {
    return `token=${getToken()}&clientId=${getClientId()}`;
}

/**
 * 获取文件url地址
 * @param key 文件key
 * @returns 文件url
 */
export function getFileUrl(key: string) {
    return `${baseUrl}/sys/files/${key}`;
}

/**
 * 获取系统文件上传url
 * @param key 文件key
 * @returns 文件上传url
 */
export function getUploadFileUrl(key: string = '') {
    return `${baseUrl}/sys/files/upload?token=${getToken()}&fileKey=${key}`;
}

/**
 * 下载文件
 * @param key 文件key
 */
export function downloadFile(key: string) {
    const a = document.createElement('a');
    a.setAttribute('href', `${getFileUrl(key)}`);
    a.setAttribute('target', '_blank');
    a.click();
    a.remove();
}
