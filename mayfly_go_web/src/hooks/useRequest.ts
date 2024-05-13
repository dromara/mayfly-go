import router from '@/router';
import { clearUser, getClientId, getRefreshToken, getToken, saveRefreshToken, saveToken } from '@/common/utils/storage';
import { templateResolve } from '@/common/utils/string';
import { ElMessage } from 'element-plus';
import { createFetch } from '@vueuse/core';
import Api from '@/common/Api';
import { Result, ResultEnum } from '@/common/request';
import config from '@/common/config';
import { unref } from 'vue';
import { URL_401 } from '@/router/staticRouter';
import openApi from '@/common/openApi';

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
            headers.set('Content-Type', 'application/json');
            options.headers = headers;

            return { options };
        },
        async afterFetch(ctx) {
            const result: Result = await ctx.response.json();
            ctx.data = result;
            return ctx;
        },
    },
});

export function useApiFetch<T>(api: Api, params: any = null, reqOptions: RequestInit = {}) {
    const uaf = useCustomFetch<T>(api.url, {
        beforeFetch({ url, options }) {
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
                paramsValue = api.beforeHandler(paramsValue);
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
    });

    return {
        execute: async function () {
            return execUaf(uaf);
        },
        isFetching: uaf.isFetching,
        data: uaf.data,
        abort: uaf.abort,
    };
}

let refreshingToken = false;
let queue: any[] = [];

async function execUaf(uaf: any) {
    try {
        await uaf.execute(true);
    } catch (e: any) {
        const rejectPromise = Promise.reject(e);

        if (e?.name == 'AbortError') {
            console.log('请求已取消');
            return rejectPromise;
        }

        const respStatus = uaf.response.value?.status;
        if (respStatus == 404) {
            ElMessage.error('请求接口不存在');
            return rejectPromise;
        }
        if (respStatus == 500) {
            ElMessage.error('服务器响应异常');
            return rejectPromise;
        }

        console.error(e);
        ElMessage.error('网络请求错误');
        return rejectPromise;
    }

    const result: Result = uaf.data.value as any;
    if (!result) {
        ElMessage.error('网络请求失败');
        return Promise.reject(result);
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
                    resolve(execUaf(uaf));
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

        await execUaf(uaf);
        return;
    }

    // 如果提示没有权限，则跳转至无权限页面
    if (resultCode === ResultEnum.NO_PERMISSION) {
        router.push({
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
