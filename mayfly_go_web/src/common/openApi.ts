import Api from './Api'

export default {
    login: Api.newPost("/sys/accounts/login"),
    otpVerify: Api.newPost("/sys/accounts/otp-verify"),
    changePwd: Api.newPost("/sys/accounts/change-pwd"),
    getPublicKey: Api.newGet("/common/public-key"),
    getConfigValue: Api.newGet("/sys/configs/value"),
    captcha: Api.newGet("/sys/captcha"),
    logout: Api.newPost("/sys/accounts/logout/{token}"),
    getPermissions: Api.newGet("/sys/accounts/permissions")
}