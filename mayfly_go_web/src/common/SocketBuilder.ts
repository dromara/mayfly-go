class SocketBuilder {
    websocket: WebSocket;

    constructor(url: string) {
        if (typeof WebSocket === 'undefined') {
            throw new Error('不支持websocket');
        }
        if (!url) {
            throw new Error('websocket url不能为空');
        }
        this.websocket = new WebSocket(url);
    }

    static builder(url: string) {
        return new SocketBuilder(url);
    }

    open(onopen: any) {
        this.websocket.onopen = onopen;
        return this;
    }

    error(onerror: any) {
        this.websocket.onerror = onerror;
        return this;
    }

    message(onmessage: any) {
        this.websocket.onmessage = onmessage;
        return this;
    }

    close(onclose: any) {
        this.websocket.onclose = onclose;
        return this;
    }

    build() {
        return this.websocket;
    }
}

export default SocketBuilder;
