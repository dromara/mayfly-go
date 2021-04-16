import Api from '@/common/Api';

export const dbApi = {
    // 获取权限列表
    dbs: Api.create("/dbs", 'get'),
    tableMetadata: Api.create("/dbs/{id}/t-metadata", 'get'),
    columnMetadata: Api.create("/dbs/{id}/c-metadata", 'get'),
    // 获取表即列提示
    hintTables: Api.create("/dbs/{id}/hint-tables", 'get'),
    selectData: Api.create("/dbs/{id}/select", 'get'),
    // 保存sql
    saveSql: Api.create("/dbs/{id}/sql", 'post'),
    // 获取保存的sql
    getSql: Api.create("/dbs/{id}/sql", 'get'),
    lsFile: Api.create("/devops/machines/files/{fileId}/ls", 'get'),
    rmFile: Api.create("/devops/machines/files/{fileId}/rm", 'delete'),
    uploadFile: Api.create("/devops/machines/files/upload", 'post'),
    fileContent: Api.create("/devops/machines/files/{fileId}/cat", 'get'),
    // 修改文件内容
    updateFileContent: Api.create("/devops/machines/files/{id}", 'put'),
    // 添加文件or目录
    addConf: Api.create("/devops/machines/{machineId}/files", 'post'),
    // 删除配置的文件or目录
    delConf: Api.create("/devops/machines/files/{id}", 'delete'),
}