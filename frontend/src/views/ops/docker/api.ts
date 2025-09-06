import Api from '@/common/Api';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';

export const dockerApi = {
    page: Api.newGet('/docker/container-conf/page'),
    saveConf: Api.newPost('/docker/container-conf/save'),
    delConf: Api.newDelete('/docker/container-conf/del/{id}'),

    info: Api.newGet('/docker/{id}/info'),

    containers: Api.newGet('/docker/{id}/containers'),
    containersStats: Api.newGet('/docker/{id}/containers/stats'),
    containerStop: Api.newPost('/docker/{id}/containers/stop'),
    containerRemove: Api.newPost('/docker/{id}/containers/remove'),
    containerRestart: Api.newPost('/docker/{id}/containers/restart'),
    containerCreate: Api.newPost('/docker/{id}/containers/create'),

    images: Api.newGet('/docker/{id}/images'),
    imageRemove: Api.newPost('/docker/{id}/images/remove'),
    imageSave: Api.newPost('/docker/{id}/images/save'),
    imageUpload: Api.newPost('/docker/{id}/images/load'),
};

export function getDockerExecSocketUrl(id: number, containerId: string) {
    return `/docker/${id}/containers/exec?id=${id}&containerId=${containerId}`;
}

export function getContainerLogSocketUrl(id: number, containerId: string) {
    return `${config.baseWsUrl}/docker/${id}/containers/logs?${joinClientParams()}&id=${id}&containerId=${containerId}`;
}
