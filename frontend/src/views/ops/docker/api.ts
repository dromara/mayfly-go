import Api from '@/common/Api';
import config from '@/common/config';
import { joinClientParams } from '@/common/request';
import type { PageResult } from '@/types/common';
import type { Container, ContainerListParam, ContainerListItem, DockerContainerStats, DockerImageItem } from './types';

export const dockerApi = {
    page: Api.newGet<PageResult<Container>, ContainerListParam>('/docker/container-conf/page'),
    saveConf: Api.newPost<void>('/docker/container-conf/save'),
    delConf: Api.newDelete<void>('/docker/container-conf/del/{id}'),

    info: Api.newGet<Record<string, unknown>>('/docker/{id}/info'),

    containers: Api.newGet<ContainerListItem[]>('/docker/{id}/containers'),
    containersStats: Api.newGet<DockerContainerStats[]>('/docker/{id}/containers/stats'),
    containerStop: Api.newPost<void>('/docker/{id}/containers/stop'),
    containerRemove: Api.newPost<void>('/docker/{id}/containers/remove'),
    containerRestart: Api.newPost<void>('/docker/{id}/containers/restart'),
    containerCreate: Api.newPost<void>('/docker/{id}/containers/create'),

    images: Api.newGet<DockerImageItem[]>('/docker/{id}/images'),
    imageRemove: Api.newPost<void>('/docker/{id}/images/remove'),
    imageSave: Api.newPost<void>('/docker/{id}/images/save'),
    imageUpload: Api.newPost<void>('/docker/{id}/images/load'),
};

export function getDockerExecSocketUrl(id: number, containerId: string) {
    return `/docker/${id}/containers/exec?id=${id}&containerId=${containerId}`;
}

export function getContainerLogSocketUrl(id: number, containerId: string) {
    return `${config.baseWsUrl}/docker/${id}/containers/logs?${joinClientParams()}&id=${id}&containerId=${containerId}`;
}
