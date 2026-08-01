import Api from '@/common/Api';
import type { PageParam, PageResult } from '@/types/common';
import type { TagTree, Team, TeamMember, ResourceOpLog, ResourceTagQueryParam, AuthCertQueryParam, ResourceTag } from './types';
import type { ResourceAuthCert } from '@/types/common';

export const tagApi = {
    listByQuery: Api.newGet<TagTree[]>('/tag-trees/query'),
    getTagTrees: Api.newGet<TagTree[]>('/tag-trees'),
    saveTagTree: Api.newPost<void>('/tag-trees'),
    delTagTree: Api.newDelete<void>('/tag-trees/{id}'),

    listResourceTags: Api.newGet<ResourceTag[], ResourceTagQueryParam>('/tag-trees/resource-tags'),
    getResourceTagPaths: Api.newGet<string[]>('/tag-trees/resources/tag-paths'),
    countTagResource: Api.newGet<Record<string, number>>('/tag-trees/resources/count'),
    getRelateTagIds: Api.newGet<number[]>('/tag-trees/relate/{relateType}/{relateId}'),

    getTeams: Api.newGet<PageResult<Team>, PageParam>('/teams'),
    saveTeam: Api.newPost<void>('/teams'),
    delTeam: Api.newDelete<void>('/teams/{id}'),

    getTeamMem: Api.newGet<PageResult<TeamMember>, PageParam>('/teams/{teamId}/members'),
    saveTeamMem: Api.newPost<void>('/teams/{teamId}/members'),
    delTeamMem: Api.newDelete<void>('/teams/{teamId}/members/{accountId}'),
};

export const resourceAuthCertApi = {
    detail: Api.newGet<ResourceAuthCert>('/auth-certs/detail'),
    listByQuery: Api.newGet<PageResult<ResourceAuthCert>, AuthCertQueryParam>('/auth-certs'),
    getByCodes: Api.newGet<ResourceAuthCert[]>('/auth-certs/simple'),
    save: Api.newPost<void>('/auth-certs'),
    delete: Api.newDelete<void>('/auth-certs/{id}'),
};

export const resourceOpLogApi = {
    getAccountResourceOpLogs: Api.newGet<PageResult<ResourceOpLog>, PageParam>('/resource-op-logs/account'),
};
