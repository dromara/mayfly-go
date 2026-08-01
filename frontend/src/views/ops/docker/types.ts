/**
 * Docker 模块类型定义
 * 对应后端: docker/domain/entity/contrainer.go
 */
import type { BaseModel, PageParam } from '@/types/common';

/** Docker 容器配置实体 (对应 entity.Container) */
export interface Container extends BaseModel {
    id: number;
    code: string;
    name: string;
    addr: string;
    remark: string;
    extra?: Record<string, unknown>;
}

/** Docker 容器配置列表查询参数 */
export interface ContainerListParam extends PageParam {
    tagPath?: string;
}

/** Docker 容器信息 */
export interface DockerContainer {
    id: string;
    name: string;
    image: string;
    imageId: string;
    command: string;
    created: string;
    state: string;
    status: string;
    ports: DockerPort[];
    labels: Record<string, string>;
}

/** Docker 端口映射 */
export interface DockerPort {
    privatePort: number;
    publicPort?: number;
    type: string;
    ip?: string;
}

/** Docker 容器统计信息 */
export interface DockerContainerStats {
    id: string;
    containerId: string;
    name: string;
    cpuPercent: number;
    cpuTotalUsage: number;
    systemUsage: number;
    memUsage: number;
    memLimit: number;
    memPercent: number;
    netIO: { input: number; output: number };
    blockIO: { read: number; write: number };
    pids: number;
}

/** Docker 容器列表项 (包含统计信息，对应后端 vo.Container) */
export interface ContainerListItem {
    containerId: string;
    name: string;
    imageId: string;
    imageName: string;
    state: string;
    status: string;
    ports: string[];
    networks: string[];
    createTime: string;
    stats?: DockerContainerStats;
}

/** Docker 镜像列表项 */
export interface DockerImageItem {
    id: string;
    tags: string[];
    size: number;
    createTime: string;
    isUse: number;
    containerId?: string;
    name?: string;
}

/** Docker 镜像信息 */
export interface DockerImage {
    id: string;
    repository: string;
    tag: string;
    size: number;
    created: string;
}

/** Docker 容器详情 */
export interface DockerContainerInfo {
    id: string;
    name: string;
    image: string;
    state: Record<string, unknown>;
    config: Record<string, unknown>;
    networkSettings: Record<string, unknown>;
    mounts: Record<string, unknown>[];
}
