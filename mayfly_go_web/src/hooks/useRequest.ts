import router from '@/router';
import { getClientId, getToken } from '@/common/utils/storage';
import { templateResolve } from '@/common/utils/string';
import { ElMessage } from 'element-plus';
import { createFetch } from '@vueuse/core';
import Api from '@/common/Api';
import { Result, ResultEnum } from '@/common/request';
import config from '@/common/config';
import { unref } from 'vue';
import { URL_401 } from '@/router/staticRouter';

const baseUrl: string = config.baseApiUrl;

const useCustomFetch = createFetch({
    baseUrl: baseUrl,
    combination: 'chain',
    options: {
        immediate: false,
        timeout: 60000,
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
            if (api.beforeHandler) {
                paramsValue = api.beforeHandler(paramsValue);
            }

            let apiUrl = url;
            // 简单判断该url是否是restful风格
            if (apiUrl.indexOf('{') != -1) {
                apiUrl = templateResolve(apiUrl, paramsValue);
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

            // 如果返回为成功结果，则将结果的data赋值给响应式data
            if (result.code === ResultEnum.SUCCESS) {
                uaf.data.value = result.data;
                return;
            }

            // 如果提示没有权限，则跳转至无权限页面
            if (result.code === ResultEnum.NO_PERMISSION) {
                router.push({
                    path: URL_401,
                });
                return Promise.reject(result);
            }

            // 如果返回的code不为成功，则会返回对应的错误msg，则直接统一通知即可。忽略登录超时或没有权限的提示（直接跳转至401页面）
            if (result.msg && result?.code != ResultEnum.NO_PERMISSION) {
                ElMessage.error(result.msg);
                uaf.error.value = new Error(result.msg);
            }

            return Promise.reject(result);
        },
        isFetching: uaf.isFetching,
        data: uaf.data,
        abort: uaf.abort,
    };
}
