import type { SuggestionContributor } from '../types';
import { columnContributor } from './column';
import { keywordContributor } from './keyword';
import { schemaContributor } from './schema';
import { tableContributor } from './table';

/**
 * 默认补全贡献者链，按序执行：
 * schema（库名） -> column（字段，含 `.` 触发的专属场景） -> table（表名） -> keyword（关键字等）
 *
 * 扩展方式：新增建议类型时实现 SuggestionContributor 并加入本链即可，
 * 无需修改补全核心逻辑（开闭原则）。
 */
export function createDefaultContributors(): SuggestionContributor[] {
    return [schemaContributor, columnContributor, tableContributor, keywordContributor];
}
