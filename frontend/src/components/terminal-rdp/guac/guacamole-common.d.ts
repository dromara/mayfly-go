/**
 * Guacamole 客户端库（guacamole-common.js）最小类型声明。
 * 仅为 MachineRdp 使用到的成员建模，返回类型保持宽松以兼容组件内的本地接口。
 */
declare const Guacamole: {
    Client: new (tunnel: unknown) => any;
    Keyboard: new (element: unknown) => any;
    Mouse: (new (element: unknown) => any) & {
        Touchpad: new (element: unknown) => any;
    };
    WebSocketTunnel: new (url: string) => any;
    StringReader: new (stream: unknown) => any;
    StringWriter: new (stream: unknown) => any;
    BlobReader: new (stream: unknown, mimetype: string) => any;
    BlobWriter: new (stream: unknown) => any;
};

export default Guacamole;
