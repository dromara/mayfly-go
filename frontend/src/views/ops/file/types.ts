/**
 * File 模块类型定义
 * 对应后端: file/domain/entity/file.go
 */

/** 文件实体 (对应 entity.File) */
export interface FileRecord {
    id: number;
    fileKey: string;
    filename: string;
    path: string;
    size: number;
    createTime: string;
    creator: string;
}
