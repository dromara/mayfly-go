import Api from '@/common/Api';

export const mongoApi = {
    mongoList: Api.newGet('/mongos'),
    mongoTags: Api.newGet('/mongos/tags'),
    testConn: Api.newPost('/mongos/test-conn'),
    saveMongo: Api.newPost('/mongos'),
    deleteMongo: Api.newDelete('/mongos/{id}'),
    databases: Api.newGet('/mongos/{id}/databases'),
    collections: Api.newGet('/mongos/{id}/collections'),
    runCommand: Api.newPost('/mongos/{id}/run-command'),
    findCommand: Api.newPost('/mongos/{id}/command/find'),
    updateByIdCommand: Api.newPost('/mongos/{id}/command/update-by-id'),
    deleteByIdCommand: Api.newPost('/mongos/{id}/command/delete-by-id'),
    insertCommand: Api.newPost('/mongos/{id}/command/insert'),
};
