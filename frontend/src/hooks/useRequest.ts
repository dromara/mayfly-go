import router from '@/router';
import { clearUser, getClientId, getRefreshToken, getToken, saveRefreshToken, saveToken } from '@/common/utils/storage';
import { templateResolve } from '@/common/utils/string';
import { ElMessage } from 'element-plus';
import { createFetch, UseFetchReturn } from '@vueuse/core';
import Api from '@/common/Api';
import { Result, ResultEnum } from '@/common/request';
import config from '@/common/config';
import { ref, unref } from 'vue';
import { URL_401 } from '@/router/staticRouter';
import openApi from '@/common/openApi';
import { useThemeConfig } from '@/store/themeConfig';

const baseUrl: string = config.baseApiUrl;

const useCustomFetch = createFetch({
    baseUrl: baseUrl,
    combination: 'chain',
    options: {
        immediate: false,
        timeout: 600000,
        // beforeFetch in pre-configured instance will only run when the newly spawned instance do not pass beforeFetch
        async beforeFetch({ options }) {
            const token = getToken();

            const headers = new Headers(options.headers || {});
            if (token) {
                headers.set('Authorization', token);
                headers.set('ClientId', getClientId());
            }

            const themeConfig = useThemeConfig().themeConfig;

            headers.set('Content-Type', 'application/json');
            headers.set('Accept-Language', themeConfig?.globalI18n);
            options.headers = headers;

            return { options };
        },
        async afterFetch(ctx) {
            ctx.data = await ctx.response.json();
            return ctx;
        },
    },
});

interface EsReq {
    esProxyReq: boolean;
}

export interface RequestOptions extends RequestInit, EsReq {}

export function useApiFetch<T>(api: Api, params: any = null, reqOptions?: RequestOptions) {
    const uaf = useCustomFetch<T>(api.url, {
        async beforeFetch({ url, options }) {
            options.method = api.method;
            if (!params) {
                return;
            }

            let paramsValue = unref(params);

            let apiUrl = url;
            // 简单判断该url是否是restful风格
            if (apiUrl.indexOf('{') != -1) {
                apiUrl = templateResolve(apiUrl, paramsValue);
            }

            if (api.beforeHandler) {
                paramsValue = await api.beforeHandler(paramsValue);
            }

            if (paramsValue) {
                const method = options.method?.toLowerCase();
                // post和put使用json格式传参
                if (method === 'post' || method === 'put') {
                    options.body = JSON.stringify(paramsValue);
                } else {
                    const searchParam = new URLSearchParams();
                    Object.keys(paramsValue).forEach((key) => {
                        const val = paramsValue[key];
                        if (val) {
                            searchParam.append(key, val);
                        }
                    });
                    apiUrl = `${apiUrl}?${searchParam.toString()}`;
                }
            }

            return {
                url: apiUrl,
                options: {
                    ...options,
                    ...reqOptions,
                },
            };
        },
        onFetchError: (ctx) => {
            if (reqOptions?.esProxyReq) {
                uaf.data = { value: JSON.parse(ctx.data) };
                return Promise.resolve(uaf.data);
            }
            return ctx;
        },
    }) as any;

    // 统一处理后的返回结果，如果直接使用uaf.data，则数据会出现由{code: x, data: {}} -> data 的变化导致某些结果绑定报错
    const data = ref<T | null>(null);
    return {
        execute: async function () {
            await execCustomFetch(uaf, reqOptions);
            data.value = uaf.data.value;
        },
        isFetching: uaf.isFetching,
        data: data,
        abort: uaf.abort,
    };
}

let refreshingToken = false;
let queue: any[] = [];

async function execCustomFetch(uaf: UseFetchReturn<any>, reqOptions?: RequestOptions) {
    try {
        await uaf.execute(true);
    } catch (e: any) {
        if (!reqOptions?.esProxyReq) {
            const rejectPromise = Promise.reject(e);

            if (e?.name == 'AbortError') {
                console.log('请求已取消');
                return rejectPromise;
            }

            const respStatus = uaf.response.value?.status;
            if (respStatus == 404) {
                ElMessage.error('url not found');
                return rejectPromise;
            }
            if (respStatus == 500) {
                ElMessage.error('server error');
                return rejectPromise;
            }

            console.error(e);
            ElMessage.error('network error');
            return rejectPromise;
        }
    }

    const result: Result & { error: any; status: number } = uaf.data.value as any;
    if (!result) {
        ElMessage.error('network request failed');
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
            const res = await openApi.refreshToken({ refresh_token: getRefreshToken() });
            saveToken(res.token);
            saveRefreshToken(res.refresh_token);
            // 重新缓存后端用户权限code
            await openApi.getPermissions();

            // 执行accessToken失效的请求
            queue.forEach((resolve: any) => {
                resolve();
            });
        } catch (e: any) {
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
        ElMessage.error(result.msg);
        uaf.error.value = new Error(result.msg);
    }

    return Promise.reject(result);
}
