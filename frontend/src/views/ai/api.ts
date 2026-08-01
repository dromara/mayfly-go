import Api from '@/common/Api';
import type { Session, SessionMessage, ToolCall } from './types';

export type { Session, SessionMessage, ToolCall } from './types';

export const aiApi = {
    // 获取权限列表
    listSessions: Api.newGet<Session[]>('/ai/chat/sessions'),
    deleteSession: Api.newDelete('/ai/chat/sessions/{sessionKey}'),
    renameSession: Api.newPost('/ai/chat/sessions/rename'),
    listMessages: Api.newGet<SessionMessage[]>('/ai/chat/messages'),
};

export function getMachineTerminalSocketUrl(authCertName: string) {
    return `/machines/terminal/${authCertName}`;
}

export function getMachineRdpSocketUrl(authCertName: string) {
    return `/api/machines/rdp/${authCertName}`;
}
