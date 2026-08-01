/**
 * Mongo 模块类型定义
 * 对应后端: mongo/domain/entity/mongo.go
 */
import type { BaseModel } from '@/types/common';

/** Mongo 实体 (对应 entity.Mongo) */
export interface Mongo extends BaseModel {
    id: number;
    code: string;
    name: string;
    uri: string;
    sshTunnelMachineId: number;
    remark?: string;
}

/**
 * Mongo 数据库信息。
 * 后端直接序列化 go.mongodb.org/mongo-driver 的 DatabaseSpecification，
 * 该结构体无 json tag，故字段名为大写。
 */
export interface MongoDatabase {
    Name: string;
    SizeOnDisk: number;
    Empty: boolean;
}

/**
 * 列出数据库结果。
 * 后端直接序列化 mongo.ListDatabasesResult（无 json tag，字段大写）。
 */
export interface MongoDatabasesResult {
    Databases: MongoDatabase[];
    TotalSize: number;
}

/** 按 id 更新文档结果（对应 mongo.UpdateResult，无 json tag） */
export interface MongoUpdateResult {
    MatchedCount: number;
    ModifiedCount: number;
    UpsertedCount: number;
    UpsertedID: unknown;
}

/** 按 id 删除文档结果（对应 mongo.DeleteResult，无 json tag） */
export interface MongoDeleteResult {
    DeletedCount: number;
}

/** 插入文档结果（对应 mongo.InsertOneResult，无 json tag） */
export interface MongoInsertResult {
    InsertedID: unknown;
}
