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

/**
 * 需提单才能执行的业务响应码，与后端 flow.application.CodeNeedWorkTicket 一致。
 *
 * 「需要审批」与「已被禁止」在界面上同为红色失败提示，但前者要给「提交工单」入口、
 * 后者绝不能给。靠匹配提示文案区分会在改文案或换语言时静默失效，所以让后端把它
 * 放进响应码，前端只认这个码
 */
export const NEED_WORK_TICKET_CODE = 4001;

/** 按响应码判断业务失败类型：错误码是唯一的分流依据，绝不用文案匹配 */
function isErrorCode(error: unknown, code: number): boolean {
    return (error as { code?: number } | null)?.code === code;
}

/** 该错误是否是「需提交工单审批」 */
export function isNeedWorkTicketError(error: unknown): boolean {
    return isErrorCode(error, NEED_WORK_TICKET_CODE);
}

/**
 * 「命中仅提醒、需操作者确认」的响应码，与后端 flowapp.CodeNeedWarnAck 对应。
 *
 * 与 4001 分开是必须的：两者后续动作完全不同（一个确认后重试、一个去提单），
 * 混用一个码会让「直接执行」和「转审批」无法分流
 */
export const WARN_ACK_CODE = 4002;

/** 该错误是否是「需要操作者确认策略提醒」 */
export function isWarnAckError(error: unknown): boolean {
    return isErrorCode(error, WARN_ACK_CODE);
}

/**
 * 业务失败的错误对象：文案已由请求层统一提示，code 供调用方按响应码分流。
 *
 * 单独抽成工厂是为了让「失败错误必须带码」这件事可被测试覆盖：
 * 请求层一旦漏挂 code，被拦下的命令就再也给不出提单入口，而且不会报错
 */
export function newBizFailureError(msg: string, code: number): Error & { code?: number } {
    const error = new Error(msg) as Error & { code?: number };
    error.code = code;
    return error;
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
async function request<T = unknown>(
    method: string,
    url: string,
    params: Record<string, unknown> | object | null = null,
    options: Record<string, unknown> = {}
): Promise<T> {
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
