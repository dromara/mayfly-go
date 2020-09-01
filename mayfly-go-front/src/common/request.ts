import router from "../router";
import Axios from 'axios';
import { ResultEnum } from './enums'
import Api from './Api';
import { AuthUtils } from './AuthUtils'
import config from './config';
import ElementUI from 'element-ui';

export interface Result {
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
  data?: any;
}

const baseUrl = config.baseApiUrl

/**
 * 通知错误消息
 * @param msg 错误消息
 */
function notifyErrorMsg(msg: string) {
  // 危险通知
  ElementUI.Message.error(msg);
}

// create an axios instance
const service = Axios.create({
  baseURL: baseUrl, // url = base url + request url
  timeout: 20000 // request timeout
})

// request interceptor
service.interceptors.request.use(
  config => {
    // do something before request is sent
    const token = AuthUtils.getToken()
    if (token) {
      // 设置token
      config.headers['Authorization'] = token
    }
    return config
  },
  error => {
    console.log(error) // for debug
    return Promise.reject(error)
  }
)

// response interceptor
service.interceptors.response.use(
  response => {
    // 获取请求返回结果
    const data: Result = response.data;
    // 如果提示没有权限，则移除token，使其重新登录
    if (data.code === ResultEnum.NO_PERMISSION) {
      AuthUtils.removeToken()
      notifyErrorMsg('登录超时')
      setTimeout(() => {
        router.push({
          path: '/login',
        });
      }, 1000)
      return;
    }
    if (data.code === ResultEnum.SUCCESS) {
      return data.data;
    } else {
      return Promise.reject(data);
    }
  },
  (  error: any) => {
    return Promise.reject(error)
  }
)

/**
 * @author: hml
 *
 * 将带有{id}的url替换为真实值；
 * 若restUrl:/category/{categoryId}/product/{productId}  param:{categoryId:1, productId:2}
 * 则返回 /category/1/product/2 的url
 */
function parseRestUrl(restUrl: string, param: any) {
  return restUrl.replace(/\{\w+\}/g, (word) => {
    const key = word.substring(1, word.length - 1);
    const value = param[key];
    if (value != null || value != undefined) {
      // delete param[key]
      return value;
    }
    return "";
  });
}

/**
 * 请求uri
 * 该方法已处理请求结果中code != 200的message提示,如需其他错误处理(取消加载状态,重置对象状态等等),可catch继续处理
 * 
 * @param {Object} method 请求方法(GET,POST,PUT,DELTE等)
 * @param {Object} uri    uri
 * @param {Object} params 参数
 */
function request(method: string, url: string, params: any, headers: any): Promise<any> {
  if (!url)
    throw new Error('请求url不能为空');
  // 简单判断该url是否是restful风格
  if (url.indexOf("{") != -1) {
    url = parseRestUrl(url, params);
  }
  const query: any = {
    method,
    url: url,
  };
  if (headers) {
    query.headers = headers
  } 
  // else {
  //   query.headers = {}
  // }
  const lowMethod = method.toLowerCase();
  // const signKey = 'sd8mow3RPMDS0PMPmMP98AS2RG43T'
  // if (params) { 
  //   delete params.sign
  //   query.headers = headers || {}
  //   // query.headers.sign = md5(Object.keys(params).sort().map(key => `${key}=${params[key]}`).join('&') + signKey)
  // } else {
  //   query.headers = headers || {}
  //   query.headers.sign = {'sign': md5(signKey)}
  // }
  // post和put使用json格式传参
  if (lowMethod === 'post' || lowMethod === 'put') {
    query.data = params;
    // query.headers.sign = md5(JSON.stringify(params) + signKey)
  } else {
    query.params = params;
    // query.headers.sign = md5(Object.keys(params).sort().map(key => `${key}=${params[key]}`).join('&') + signKey)
  }
  return service.request(query).then(res => res)
    .catch(e => {
      notifyErrorMsg(e.msg || e.message)
      return Promise.reject(e);
    });
}

/**
 * 根据api执行对应接口
 * @param api Api实例
 * @param params 请求参数
 */
function send(api: Api, params: any): Promise<any> {
  return request(api.method, api.url, params, null);
}

/**
 * 根据api执行对应接口
 * @param api Api实例
 * @param params 请求参数
 */
function sendWithHeaders(api: Api, params: any, headers: any): Promise<any> {
  return request(api.method, api.url, params, headers);
}

function getApiUrl(url: string) {
  // 只是返回api地址而不做请求，用在上传组件之类的
  return baseUrl + url + '?token=' + AuthUtils.getToken();
}

export default {
  request,
  send,
  sendWithHeaders,
  parseRestUrl,
  getApiUrl
}
