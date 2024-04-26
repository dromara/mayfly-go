import Api from '@/common/Api';

export const tagApi = {
    listByQuery: Api.newGet('/tag-trees/query'),
    getTagTrees: Api.newGet('/tag-trees'),
    saveTagTree: Api.newPost('/tag-trees'),
    delTagTree: Api.newDelete('/tag-trees/{id}'),
    movingTag: Api.newPost('/tag-trees/moving'),

    getResourceTagPaths: Api.newGet('/tag-trees/resources/{resourceType}/tag-paths'),
    countTagResource: Api.newGet('/tag-trees/resources/count'),
    getRelateTagIds: Api.newGet('/tag-trees/relate/{relateType}/{relateId}'),

    getTeams: Api.newGet('/teams'),
    saveTeam: Api.newPost('/teams'),
    delTeam: Api.newDelete('/teams/{id}'),

    getTeamMem: Api.newGet('/teams/{teamId}/members'),
    saveTeamMem: Api.newPost('/teams/{teamId}/members'),
    delTeamMem: Api.newDelete('/teams/{teamId}/members/{accountId}'),
};

export const resourceAuthCertApi = {
    detail: Api.newGet('/auth-certs/detail'),
    listByQuery: Api.newGet('/auth-certs'),
    save: Api.newPost('/auth-certs'),
    delete: Api.newDelete('/auth-certs/{id}'),
};

export const resourceOpLogApi = {
    getAccountResourceOpLogs: Api.newGet('/resource-op-logs/account'),
};
