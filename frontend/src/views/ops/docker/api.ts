import Api from '@/common/Api';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';

export const dockerApi = {
    info: Api.newGet('/docker/info'),

    containers: Api.newGet('/docker/containers'),
    containersStats: Api.newGet('/docker/containers/stats'),
    containerStop: Api.newPost('/docker/containers/stop'),
    containerRemove: Api.newPost('/docker/containers/remove'),
    containerRestart: Api.newPost('/docker/containers/restart'),
    containerCreate: Api.newPost('/docker/containers/create'),

    images: Api.newGet('/docker/images'),
    imageRemove: Api.newPost('/docker/images/remove'),
    imageSave: Api.newPost('/docker/images/save'),
    imageUpload: Api.newPost('/docker/images/load'),
};

export function getDockerExecSocketUrl(host: any, containerId: string) {
    return `/docker/containers/exec?host=${host}&containerId=${containerId}`;
}

export function getContainerLogSocketUrl(host: any, containerId: string) {
    return `${config.baseWsUrl}/docker/containers/logs?${joinClientParams()}&host=${host}&containerId=${containerId}`;
}
