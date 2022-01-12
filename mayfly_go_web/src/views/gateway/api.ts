import Api from '@/common/Api';

export const serviceApi = {
    services: Api.create("/gw/services", 'get'),
    saveService: Api.create("/gw/services", 'post'),
    syncService: Api.create("/gw/services/{id}/sync", 'post'),
    // 获取服务下的api信息
    serviceApis:  Api.create("/gw/services/{serviceId}/apis", 'get'),
    saveServiceApi:  Api.create("/gw/services/{serviceId}/apis", 'post'),
    syncServiceApi: Api.create("/gw/services/{id}/apis/{apiId}/sync", 'post'),
    // 获取项目下的成员信息
    projectMems:  Api.create("/gw/projects/{projectId}/members", 'get'),
    saveProjectMem:  Api.create("/gw/projects/{projectId}/members", 'post'),
    deleteProjectMem:  Api.create("/gw/projects/{projectId}/members/{accountId}", 'delete'),
}   