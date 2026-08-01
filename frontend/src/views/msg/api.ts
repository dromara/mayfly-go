import Api from '@/common/Api';
import type { PageResult } from '@/types/common';
import type { MsgChannel, MsgTemplate } from '@/views/system/msg/types';

export const channelApi = {
    list: Api.newGet<PageResult<MsgChannel>>('/msg/channels'),
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
