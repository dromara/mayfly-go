/**
 * Auth 模块类型定义
 * 对应后端: auth/domain/entity/oauth2.go
 */

/** OAuth2 账户关联实体 (对应 entity.Oauth2Account) */
export interface Oauth2Account {
    id: number;
    accountId: number;
    identity: string;
    createTime: string;
    updateTime: string;
}

/** OAuth2 登录配置 */
export interface OAuth2LoginConfig {
    enabled: boolean;
    clientId: string;
    authorizationUrl: string;
    scopes: string;
}

/** LDAP 配置 */
export interface LdapConfig {
    enabled: boolean;
    host: string;
    port: number;
    baseDn: string;
    bindDn: string;
    bindPassword: string;
    userFilter: string;
}
