export class AuthUtils {

    private static tokenName = 'token'

    /**
     * 保存token
     * @param token token
     */
    static saveToken(token: string) {
        sessionStorage.setItem(this.tokenName, token)
    }

    /**
     * 获取token
     */
    static getToken() {
        return sessionStorage.getItem(this.tokenName)
    }

    /**
     * 移除token
     */
    static removeToken() {
        sessionStorage.removeItem(this.tokenName)
    }
}