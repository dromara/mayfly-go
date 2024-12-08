export const ClientState = {
    /**
     * The client is idle, with no active connection.
     *
     * @type number
     */
    IDLE: 0,

    /**
     * The client is in the process of establishing a connection.
     *
     * @type {!number}
     */
    CONNECTING: 1,

    /**
     * The client is waiting on further information or a remote server to
     * establish the connection.
     *
     * @type {!number}
     */
    WAITING: 2,

    /**
     * The client is actively connected to a remote server.
     *
     * @type {!number}
     */
    CONNECTED: 3,

    /**
     * The client is in the process of disconnecting from the remote server.
     *
     * @type {!number}
     */
    DISCONNECTING: 4,

    /**
     * The client has completed the connection and is no longer connected.
     *
     * @type {!number}
     */
    DISCONNECTED: 5,
};

export const TunnelState = {
    /**
     * A connection is in pending. It is not yet known whether connection was
     * successful.
     *
     * @type {!number}
     */
    CONNECTING: 0,

    /**
     * Connection was successful, and data is being received.
     *
     * @type {!number}
     */
    OPEN: 1,

    /**
     * The connection is closed. Connection may not have been successful, the
     * tunnel may have been explicitly closed by either side, or an error may
     * have occurred.
     *
     * @type {!number}
     */
    CLOSED: 2,

    /**
     * The connection is open, but communication through the tunnel appears to
     * be disrupted, and the connection may close as a result.
     *
     * @type {!number}
     */
    UNSTABLE: 3,
};
