import Api from '@/common/Api';

export const machineApi = {
    // 获取权限列表
    list: Api.create("/machines", 'get'),
    info: Api.create("/machines/{id}/sysinfo", 'get'),
    top: Api.create("/machines/{id}/top", 'get'),
    // 保存按钮
    save: Api.create("/devops/machines", 'post'),
    update: Api.create("/devops/machines/{id}", 'put'),
    // 删除机器
    del: Api.create("/devops/machines/{id}", 'delete'),
    // 获取配置文件列表
    files: Api.create("/devops/machines/{id}/files", 'get'),
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