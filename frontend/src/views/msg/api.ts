import Api from '@/common/Api';

export const channelApi = {
    list: Api.newGet('/msg/channels'),
    save: Api.newPost('/msg/channels'),
    del: Api.newDelete('/msg/channels'),
};

export const tmplApi = {
    list: Api.newGet('/msg/tmpls'),
    relateChannels: Api.newGet('/msg/tmpls/{id}/channels'),
    save: Api.newPost('/msg/tmpls'),
    del: Api.newDelete('/msg/tmpls'),
    sendMsg: Api.newPost('/msg/tmpls/{code}/send'),
};
