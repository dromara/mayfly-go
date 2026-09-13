/**
 * 表编辑服务层索引（table-editor 私有）
 *
 * 只 re-export 有真实消费方的单例：validationService / schemaDiffService 由 DbTableOp 消费，
 * templateService 由 TemplateSelector 消费。服务类为本目录私有（消费方只用单例），
 * 不经本桶文件对外暴露，避免出现「看似公共 API 实则无人调用」的死导出。
 */

export { validationService } from './validationService';
export { templateService } from './templateService';
export { schemaDiffService } from './schemaDiffService';
