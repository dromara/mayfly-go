import Api from '@/common/Api';

export const projectApi = {
    // 获取账号可访问的项目列表
    accountProjects: Api.create("/accounts/projects", 'get'),
    projects: Api.create("/projects", 'get'),
    saveProject: Api.create("/projects", 'post'),
    // 获取项目下的环境信息
    projectEnvs:  Api.create("/projects/{projectId}/envs", 'get'),
    saveProjectEnv:  Api.create("/projects/{projectId}/envs", 'post'),
    // 获取项目下的成员信息
    projectMems:  Api.create("/projects/{projectId}/members", 'get'),
    saveProjectMem:  Api.create("/projects/{projectId}/members", 'post'),
    deleteProjectMem:  Api.create("/projects/{projectId}/members/{accountId}", 'delete'),
}   