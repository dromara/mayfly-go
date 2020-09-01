import request from './request'

export default {
  login: (param: any) => request.request('POST', '/accounts/login', param, null),
  captcha: () => request.request('GET', '/open/captcha', null, null),
  logout: (param: any) => request.request('POST', '/sys/accounts/logout/{token}', param, null)
}