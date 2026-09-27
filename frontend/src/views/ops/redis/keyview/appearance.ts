/**
 * 数据视角与视角操作的呈现皮肤：类型主色、类型速记字母、图标。
 *
 * 分工边界：后端描述符负责「这个类型能做什么、数据长什么样」（能力位、列结构、表单、操作清单），
 * 这里负责「它在界面上长什么样」。皮肤改动不必重编后端；新增视角或操作时漏登记，
 * 由 __tests__/keyview.test.ts 里「皮肤必须覆盖后端声明的每个视角与操作」的用例拦住。
 */

/** 单个视角的皮肤 */
export type RedisViewAppearance = {
    /** 类型主色：只用于加速扫读，类型识别靠 badge 本身 */
    color: string;
    /** 类型速记字母，固定 2 字符：窄位置（key 列表）放不下文字标签 */
    badge: string;
    icon: string;
};

/** 视角未登记时的中性皮肤：不假装知道类型，也不让列表列宽跳动 */
const UNKNOWN: RedisViewAppearance = { color: 'var(--el-border-color-darker)', badge: '--', icon: 'QuestionFilled' };

const views: Record<string, RedisViewAppearance> = {
    string: { color: '#f56c6c', badge: 'St', icon: 'Document' },
    bitmap: { color: '#8b5cf6', badge: 'Bi', icon: 'Grid' },
    hyperloglog: { color: '#f59e0b', badge: 'Hl', icon: 'Odometer' },
    hash: { color: '#409eff', badge: 'Ha', icon: 'Grid' },
    list: { color: '#67c23a', badge: 'Li', icon: 'Tickets' },
    set: { color: '#e6a23c', badge: 'Se', icon: 'PieChart' },
    zset: { color: '#a855f7', badge: 'Zs', icon: 'Histogram' },
    geo: { color: '#14b8a6', badge: 'Ge', icon: 'MapLocation' },
    stream: { color: '#0ea5e9', badge: 'Xs', icon: 'DataLine' },
};

/** 视角操作的图标，键为 `${view}:${op}`：op 名是视角内的局部标识，跨视角可能重名 */
const opIcons: Record<string, string> = {
    'bitmap:bitcount': 'DataAnalysis',
    'bitmap:bitpos': 'Aim',
    'geo:geodist': 'Connection',
    'geo:geosearch': 'Search',
    'hash:hincrby': 'Sort',
    'hash:hexpire': 'AlarmClock',
    'hash:hrandfield': 'MagicStick',
    'list:ltrim': 'Scissor',
    'list:lpop': 'Back',
    'list:rpop': 'Right',
    'set:spop': 'Sell',
    'set:combine': 'Link',
    'stream:xtrim': 'Scissor',
    'stream:xgroupcreate': 'Plus',
    'stream:xack': 'Check',
    'stream:xinfo': 'InfoFilled',
    'zset:zincrby': 'Sort',
    'zset:zpopmax': 'Top',
    'zset:zpopmin': 'Bottom',
    'zset:zrank': 'View',
    'zset:zremrangebyscore': 'Delete',
};

/** 取视角皮肤；未登记（新类型还没补皮肤）时返回中性皮肤而不是崩在界面上 */
export function viewAppearance(view?: string): RedisViewAppearance {
    return (view && views[view]) || UNKNOWN;
}

/** 取视角操作图标 */
export function opIcon(view: string, op: string): string {
    return opIcons[`${view}:${op}`] ?? UNKNOWN.icon;
}

/** 只读皮肤表：测试据此核对「后端声明的每个视角/操作都登记了皮肤」 */
export const viewSkins: Readonly<Record<string, RedisViewAppearance>> = views;
export const opSkins: Readonly<Record<string, string>> = opIcons;
