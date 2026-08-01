/**
 * 公共类型定义，对应后端 pkg/model 和各模块 entity/form/vo
 */

// ==================== 基础实体模型 ====================

/** 基础创建信息 (对应后端 model.CreateModel) */
export interface CreateModel {
    createTime: string;
    creatorId?: number;
    creator: string;
}

/** 基础实体模型，含创建+修改信息 (对应后端 model.Model) */
export interface BaseModel extends CreateModel {
    updateTime: string;
    modifierId?: number;
    modifier: string;
}

// ==================== 分页相关 ====================

/** 分页请求参数 (对应后端 model.PageParam) */
export interface PageParam {
    pageNum?: number;
    pageSize?: number;
}

/** 分页响应 (对应后端 model.PageResult[T]) */
export interface PageResult<T> {
    total: number;
    list: T[];
}

/** 系统消息 (对应后端 msgentity.Msg) */
export interface Msg {
    id: number;
    type: number;
    subtype: string;
    status: number;
    msg: string;
    recipientId?: number;
    createTime: string;
    extra?: Record<string, any>;
    [key: string]: unknown;
}

// ==================== 标签相关 ====================

/** 资源标签 (对应后端 tagentity.ResourceTag) */
export interface ResourceTag {
    tagId: number;
    codePath: string;
}

/** 关联标签信息 (对应后端 tagentity.RelateTags) */
export interface RelateTags {
    tags: ResourceTag[];
}

/** 资源标签路径列表 (对应后端 tagentity.ResourceTags) */
export interface ResourceTags {
    tags: ResourceTag[];
}

// ==================== 授权凭证相关 ====================

/** 授权凭证基本信息 (对应后端 tagentity.AuthCert) */
export interface AuthCertInfo {
    name: string;
    username: string;
    ciphertextType: number;
    type: number;
}

/** 授权凭证列表 (对应后端 tagentity.AuthCerts) */
export interface AuthCerts {
    authCerts: AuthCertInfo[];
}

/** 资源授权凭证完整信息 (对应后端 tagentity.ResourceAuthCert) */
export interface ResourceAuthCert {
    id?: number | null;
    name: string;
    resourceCode: string;
    resourceType: number;
    type: number;
    username: string;
    ciphertext: string;
    ciphertextType: number;
    remark?: string;
    extra?: Record<string, unknown>;
    createTime?: string;
    creator?: string;
}

// ==================== 通用工具类型 ====================

/** 可选的日期范围参数 */
export interface DateRangeParam {
    startTime?: string;
    endTime?: string;
}
