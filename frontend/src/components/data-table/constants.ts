/**
 * 列宽自适应的共享度量常量。
 *
 * 单独成模块，是为了让「列模型（table-column）」与「行内操作（actions）」两处引用同一份真源
 * 而不互相 import 实现——它们只依赖这几个数字，不依赖彼此（DOM 测量侧改用 getComputedStyle 取真实内边距）。
 * 本模块是 data-table 的内部件，不经 index barrel 对外暴露。
 */

// 单元格左右内间距等（内容需求宽 → minWidth 的固定加项）
export const CELL_PADDING = 30;

// 表头离线估宽的额外留白：仅覆盖 .cell 左右内边距 + 少量缓冲。排序箭头/表头图标等「文字之外的占位」
// 不在此预算内，改由渲染后的表头实测回填（columnWidthMeasure.measureHeaderCellWidth）自适应，
// 因此不再需要像过去那样给每列一律留 60px（那会把短值列撑得明显偏空）。
export const HEADER_PADDING = 30;

// 自动计宽上限
export const MAX_AUTO_WIDTH = 400;

// el-tag 相对纯文本额外占用的边框 + 内边距
export const TAG_CHROME_WIDTH = 22;
