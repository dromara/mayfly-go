import Api from '@/common/Api';

export const tagApi = {
    getAccountTags: Api.create("/tag-trees/account-has", 'get'),
    getTagTrees: Api.create("/tag-trees", 'get'),
    saveTagTree: Api.create("/tag-trees", 'post'),
    delTagTree: Api.create("/tag-trees/{id}", 'delete'),

    getTeams: Api.create("/teams", 'get'),
    saveTeam: Api.create("/teams", 'post'),
    delTeam: Api.create("/teams/{id}", 'delete'),

    getTeamMem: Api.create("/teams/{teamId}/members", 'get'),
    saveTeamMem:  Api.create("/teams/{teamId}/members", 'post'),
    delTeamMem:  Api.create("/teams/{teamId}/members/{accountId}", 'delete'),

    getTeamTagIds: Api.create("/teams/{teamId}/tags", 'get'),
    saveTeamTags: Api.create("/teams/{teamId}/tags", 'post'),
}   