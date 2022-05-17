import Api from '@/common/Api';

export const mongoApi = {
    mongoList : Api.create("/mongos", 'get'),
    saveMongo : Api.create("/mongos", 'post'),
    deleteMongo : Api.create("/mongos/{id}", 'delete'),
    databases: Api.create("/mongos/{id}/databases", 'get'),
    collections: Api.create("/mongos/{id}/collections", 'get'),
    runCommand: Api.create("/mongos/{id}/run-command", 'post'),
    findCommand: Api.create("/mongos/{id}/command/find", 'post'),
    updateByIdCommand: Api.create("/mongos/{id}/command/update-by-id", 'post'),
    deleteByIdCommand: Api.create("/mongos/{id}/command/delete-by-id", 'post'),
    insertCommand: Api.create("/mongos/{id}/command/insert", 'post'),
}