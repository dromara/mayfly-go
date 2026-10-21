import Api from '@/common/Api';
import type { PageResult } from '@/types/common';
import type { MsgChannel, MsgTemplate } from '@/views/system/msg/types';

export const channelApi = {
    list: Api.newGet<PageResult<MsgChannel>>('/msg/channels'),
    // 单条详情：带完整 extra（含 bot 加签密钥）。列表里的 secret 已被后端脱敏，
    // 编辑回填必须走这个受 msg:channel:save 把守的出口，否则保存一次就把密钥抹空
    detail: Api.newGet<MsgChannel>('/msg/channels/{id}'),
    save: Api.newPost<void>('/msg/channels'),
    del: Api.newDelete<void>('/msg/channels'),
};

export const tmplApi = {
    list: Api.newGet<PageResult<MsgTemplate>>('/msg/tmpls'),
    relateChannels: Api.newGet<MsgChannel[]>('/msg/tmpls/{id}/channels'),
    save: Api.newPost<void>('/msg/tmpls'),
    del: Api.newDelete<void>('/msg/tmpls'),
    sendMsg: Api.newPost<void>('/msg/tmpls/{code}/send'),
};
