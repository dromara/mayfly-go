import Api from '@/common/Api';
import config from '@/common/config';
import openApi from '@/common/openApi';
import { Result, ResultEnum } from '@/common/request';
import { clearUser, getClientId, getRefreshToken, getToken, saveRefreshToken, saveToken } from '@/common/utils/storage';
import { templateResolve } from '@/common/utils/string';
import router from '@/router';
import { URL_401 } from '@/router/staticRouter';
import { useThemeConfig } from '@/store/themeConfig';
import { createFetch, UseFetchReturn } from '@vueuse/core';
import JSONBig from 'json-bigint';
import { ref, unref } from 'vue';
import { Msg } from './useI18n';

const baseUrl: string = config.baseApiUrl;

// 配置 JSONBig：将大数（int64/uint64）转为字符串，避免精度丢失
const JSONBigString = JSONBig({ storeAsString: true });

const useCustomFetch = createFetch({
    baseUrl: baseUrl,
    combination: 'chain',
    options: {
        immediate: false,
        timeout: 600000,
        // beforeFetch in pre-configured instance will only run when the newly spawned instance do not pass beforeFetch
        async beforeFetch({ url, options }: { url: string; options: RequestInit }) {
            const token = getToken();

            const headers = new Headers(options.headers || {});
            if (token) {
                headers.set('Authorization', token);
                headers.set('ClientId', getClientId());
            }

            const themeConfig = useThemeConfig().themeConfig;

            // 如果不是 FormData，才设置 Content-Type
            if (!(options.body instanceof FormData)) {
                headers.set('Content-Type', 'application/json');
            }
            headers.set('Accept-Language', themeConfig?.globalI18n);
            options.headers = headers;

            return { url, options };
        },
        async afterFetch(ctx: { response: Response; data: unknown }) {
            // 使用 json-bigint 解析响应数据，解决 int64/uint64 精度丢失问题
            const responseText = await ctx.response.text();
            try {
                ctx.data = JSONBigString.parse(responseText);
            } catch (err) {
                // 如果解析失败，尝试使用原生 JSON.parse
                try {
                    ctx.data = JSON.parse(responseText);
                } catch {
                    ctx.data = responseText;
                }
            }
            return ctx;
        },
    },
});

interface EsReq {
    esProxyReq?: boolean;
}

export interface RequestOptions extends RequestInit, EsReq {}

export function useApiFetch<T, P = unknown>(api: Api, params?: P, reqOptions?: RequestOptions) {
    const currentParam = ref(params);

    const uaf: any = useCustomFetch<T>(api.url, {
        async beforeFetch({ url, options }: { url: string; options: RequestInit }) {
            options.method = api.method;
            let paramsValue = unref(currentParam);

            let apiUrl = url;
            // 提取 ac（授权凭证名）参数，始终以查询参数方式传递
            let acValue: string | undefined;
            if (paramsValue && paramsValue.ac) {
                acValue = paramsValue.ac;
                const { ac: _ac, ...restParams } = paramsValue;
                paramsValue = restParams;
            }

            // 简单判断该url是否是restful风格
            if (apiUrl.indexOf('{') != -1 && paramsValue) {
                apiUrl = templateResolve(apiUrl, paramsValue);
            }

            if (api.beforeHandler) {
                paramsValue = await api.beforeHandler(paramsValue);
            }

            // post和put使用json格式传参（如果是FormData则直接使用）
            const method = options.method?.toLowerCase();
            if ((method === 'post' || method === 'put') && paramsValue) {
                if (paramsValue instanceof FormData) {
                    options.body = paramsValue;
                    // 对于 FormData，删除 Content-Type header，让浏览器自动设置 multipart/form-data 和 boundary
                    if (options.headers instanceof Headers) {
                        options.headers.delete('Content-Type');
                    } else if (options.headers && typeof options.headers === 'object') {
                        delete (options.headers as Record<string, string>)['Content-Type'];
                    }
                } else {
                    options.body = JSON.stringify(paramsValue);
                }
            } else if (paramsValue && method !== 'post' && method !== 'put') {
                const searchParam = new URLSearchParams();
                Object.keys(paramsValue).forEach((key) => {
                    const val = paramsValue[key];
                    if (val) {
                        searchParam.append(key, val);
                    }
                });
                const qs = searchParam.toString();
                if (qs) {
                    apiUrl = `${apiUrl}?${qs}`;
                }
            }

            // ac 参数始终以查询参数形式附加到 URL
            if (acValue) {
                const separator = apiUrl.includes('?') ? '&' : '?';
                apiUrl = `${apiUrl}${separator}ac=${encodeURIComponent(acValue)}`;
            }

            // 确保 FormData 的 body 不被 reqOptions 覆盖
            const finalOptions = {
                ...options,
                ...reqOptions,
            };
            // 如果原始 options.body 是 FormData，优先保留
            if (options.body instanceof FormData) {
                finalOptions.body = options.body;
                // 对于 FormData，不要设置 Content-Type，让浏览器自动设置 multipart/form-data 和 boundary
                if (finalOptions.headers instanceof Headers) {
                    finalOptions.headers.delete('Content-Type');
                } else if (finalOptions.headers && typeof finalOptions.headers === 'object') {
                    delete (finalOptions.headers as Record<string, string>)['Content-Type'];
                }
            }

            return {
                url: apiUrl,
                options: finalOptions,
            };
        },
        onFetchError: (ctx: { data: unknown }) => {
            if (reqOptions?.esProxyReq) {
                // 使用 json-bigint 解析错误响应
                try {
                    const errorText = typeof ctx.data === 'string' ? ctx.data : JSON.stringify(ctx.data);
                    uaf.data = { value: JSONBigString.parse(errorText) };
                } catch {
                    uaf.data = { value: ctx.data };
                }
                return Promise.resolve(uaf.data);
            }
            return ctx;
        },
    });

    // 统一处理后的返回结果，如果直接使用uaf.data，则数据会出现由{code: x, data: {}} -> data 的变化导致某些结果绑定报错
    const data = ref<T | null>(null);
    return {
        execute: async function (executeParam?: P) {
            if (executeParam !== undefined) {
                currentParam.value = executeParam;
            }

            await execCustomFetch(uaf, reqOptions);
            data.value = uaf.data.value;
        },
        isFetching: uaf.isFetching,
        data: data,
        abort: uaf.abort,
    };
}

let refreshingToken = false;
let queue: (() => void)[] = [];

async function execCustomFetch(uaf: UseFetchReturn<unknown>, reqOptions?: RequestOptions) {
    try {
        await uaf.execute(true);
    } catch (e: unknown) {
        if (!reqOptions?.esProxyReq) {
            const rejectPromise = Promise.reject(e);

            if ((e as Error)?.name == 'AbortError') {
                return rejectPromise;
            }

            const respStatus = uaf.response.value?.status;
            if (respStatus == 404) {
                Msg.error('common.urlNotFoundError');
                return rejectPromise;
            }
            if (respStatus == 500) {
                Msg.error('common.serverError');
                return rejectPromise;
            }

            console.error(e);
            Msg.error('common.networkError');
            return rejectPromise;
        }
    }

    const result = uaf.data.value as (Result & { error: unknown; status: number }) | undefined;
    if (!result) {
        Msg.error('common.requestFailedError');
        return Promise.reject(result);
    }
    // es代理请求
    if (reqOptions?.esProxyReq) {
        uaf.data.value = result;
        return Promise.resolve(result);
    }

    const resultCode = result.code;

    // 如果返回为成功结果，则将结果的data赋值给响应式data
    if (resultCode === ResultEnum.SUCCESS) {
        uaf.data.value = result.data;
        return;
    }

    // 如果是accessToken失效，则使用refreshToken刷新token
    if (resultCode == ResultEnum.ACCESS_TOKEN_INVALID) {
        if (refreshingToken) {
            // 请求加入队列等待, 防止并发多次请求refreshToken
            return new Promise((resolve) => {
                queue.push(() => {
                    resolve(execCustomFetch(uaf, reqOptions));
                });
            });
        }

        try {
            refreshingToken = true;
            const res = await openApi.refreshToken({ refresh_token: getRefreshToken() }) as { token: string; refresh_token: string };
            saveToken(res.token);
            saveRefreshToken(res.refresh_token);
            // 重新缓存后端用户权限code
            await openApi.getPermissions();

            // 执行accessToken失效的请求
            queue.forEach((resolve: () => void) => {
                resolve();
            });
        } catch (e: unknown) {
            clearUser();
        } finally {
            refreshingToken = false;
            queue = [];
        }

        await execCustomFetch(uaf, reqOptions);
        return;
    }

    // 如果提示没有权限，则跳转至无权限页面
    if (resultCode === ResultEnum.NO_PERMISSION) {
        await router.push({
            path: URL_401,
        });
        return Promise.reject(result);
    }

    // 如果返回的code不为成功，则会返回对应的错误msg，则直接统一通知即可。忽略登录超时或没有权限的提示（直接跳转至401页面）
    if (result.msg && resultCode != ResultEnum.NO_PERMISSION) {
        Msg.error(result.msg);
        uaf.error.value = new Error(result.msg);
    }

    return Promise.reject(result);
}
