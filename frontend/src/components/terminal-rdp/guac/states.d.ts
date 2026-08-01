/**
 * guac/states.js 类型声明：Guacamole 客户端与隧道状态常量。
 */
export declare const ClientState: {
    IDLE: number;
    CONNECTING: number;
    WAITING: number;
    CONNECTED: number;
    DISCONNECTING: number;
    DISCONNECTED: number;
};

export declare const TunnelState: {
    CONNECTING: number;
    OPEN: number;
    CLOSED: number;
    UNSTABLE: number;
};
