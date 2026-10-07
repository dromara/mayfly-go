/**
 * key 名列表 → 分组树：把 `a:b:c` 这类 key 按分隔符折叠成目录层。
 *
 * 树模型里有两种节点：目录（继续嵌套）与 key（叶子）。同一段名既是目录又是 key 的场景
 * （目录 `aa` 与 key `aa`）靠 key 叶子的魔法键后缀区分，段名本身不会带上后缀，因此判别无歧义；
 * 每层目录用无原型对象承载，key 里出现 `__proto__`、`constructor` 这类段名时不会污染对象原型
 */

export interface TreeNode {
    name: string;
    /** 1 目录，2 key（与资源树渲染层的图标判断约定一致） */
    type?: number;
    key?: string;
    open?: boolean;
    keyCount?: number;
    children?: TreeNode[];
    [key: string]: unknown;
}

/** key 叶子在目录层里挂载用的魔法键后缀：段名里带反引号的概率极低，撞名即视作目录 */
const KEY_SUFFIX = '`k`';

interface TreeKeyNode {
    keyNode: true;
}

interface TreeFolder {
    [key: string]: TreeFolder | TreeKeyNode;
}

const isFolderName = (name: string): boolean => !name.endsWith(KEY_SUFFIX);

export function keysToTree(keys: string[], separator: string = ':', openStatus: Set<string> | null = null) {
    const tree: TreeFolder = Object.create(null) as TreeFolder;
    keys.forEach((key: string) => {
        let currentNode = tree;
        const segments = key.split(separator);
        const lastIndex = segments.length - 1;

        segments.forEach((segment: string, index: number) => {
            if (index === lastIndex) {
                currentNode[`${key}${KEY_SUFFIX}`] = { keyNode: true };
                return;
            }
            if (!isFolderName(segment) || currentNode[segment] === undefined) {
                currentNode[segment] = Object.create(null) as TreeFolder;
            }
            currentNode = currentNode[segment] as TreeFolder;
        });
    });

    return formatTreeData(tree, '', separator, openStatus).sort(compareTreeNode);
}

export function keysToList(keys: string[]) {
    return keys.map((key: string) => ({ key, name: key }));
}

/**
 * 增量插入：把新增的一批 key 插入已构建好的分组树（「加载更多」用），
 * 避免每批都全量重建整棵树。落已有目录则插叶子并沿路 keyCount++，缺目录则按序建目录链；
 * 插入位用二分定位，维持与 keysToTree 完全相同的有序输出。
 *
 * 就地修改并返回传入的 tree：消费方持有的正是这份引用，重建反而会丢掉展开/勾选态。
 */
export function insertKeysToTree(tree: TreeNode[], keys: string[], separator = ':', openStatus: Set<string> | null = null): TreeNode[] {
    keys.forEach((key) => insertOneKey(tree, key, separator, openStatus));
    return tree;
}

function insertOneKey(root: TreeNode[], key: string, separator: string, openStatus: Set<string> | null) {
    const segments = key.split(separator);
    const lastIndex = segments.length - 1;
    let siblings = root;
    // 沿途经过的目录：叶子确认新增后，再给它们补 keyCount（重复 key 不动计数）
    const folders: TreeNode[] = [];
    let prefix = '';

    for (let index = 0; index < lastIndex; index++) {
        const segment = segments[index];
        prefix += segment + separator;
        // 目录用带分隔符结尾的全路径 key 匹配，与同名 key 叶子（如目录 `aa:` 与 key `aa`）区分开
        let folder = siblings.find((node) => node.type === 1 && node.key === prefix);
        if (!folder) {
            folder = { name: segment || '[Empty]', type: 1, key: prefix, children: [], keyCount: 0 };
            if (openStatus?.has(prefix)) {
                folder.open = true;
            }
            sortedInsert(siblings, folder);
        }
        folders.push(folder);
        siblings = folder.children as TreeNode[];
    }

    // 叶子已存在即重复 key（其祖先目录必然已存在，不会留下空目录），直接返回
    if (siblings.some((node) => node.type === 2 && node.key === key)) {
        return;
    }
    sortedInsert(siblings, { name: key || '[Empty]', type: 2, key });
    folders.forEach((folder) => {
        folder.keyCount = (folder.keyCount ?? 0) + 1;
    });
}

/** 二分定位有序插入位（compareTreeNode 语义），比线性 findIndex 更稳（大目录下追加不退化成 O(n^2) 扫描） */
function sortedInsert(siblings: TreeNode[], node: TreeNode) {
    let lo = 0;
    let hi = siblings.length;
    while (lo < hi) {
        const mid = (lo + hi) >> 1;
        if (compareTreeNode(siblings[mid], node) < 0) {
            lo = mid + 1;
        } else {
            hi = mid;
        }
    }
    siblings.splice(lo, 0, node);
}

function formatTreeData(tree: TreeFolder, previousKey: string = '', separator: string = ':', openStatus: Set<string> | null = null): TreeNode[] {
    return Object.keys(tree).map((name) => {
        if (isFolderName(name) && Object.keys(tree[name]).length > 0) {
            const node = tree[name] as TreeFolder;
            const tillNowKeyName = previousKey + name + separator;
            const children = formatTreeData(node, tillNowKeyName, separator, openStatus);
            // 排序下沉数据层：输出即有序（目录在前、key 在后，同组按名字典序）。
            // 虚拟树按 data 顺序渲染，不再依赖 el-tree v1 的 root.childNodes DOM 补排
            children.sort(compareTreeNode);
            // 目录的 key 带分隔符结尾，与同名 key 的全名（如目录 `aa-` 与 key `aa`）区分开
            const treeNode: TreeNode = {
                name: name || '[Empty]',
                type: 1,
                key: tillNowKeyName,
                children,
                keyCount: children.reduce((sum: number, child: TreeNode) => sum + (child.keyCount || 1), 0),
            };
            if (openStatus?.has(tillNowKeyName)) {
                treeNode.open = true;
            }
            return treeNode;
        }

        const keyName = isFolderName(name) ? name : name.slice(0, -KEY_SUFFIX.length);
        return { name: keyName || '[Empty]', type: 2, key: keyName };
    });
}

/** 目录在前、key 在后，同组内按名字典序：分组树的统一排序语义 */
function compareNodeByName<T>(isFolder: (node: T) => boolean, nameOf: (node: T) => string) {
    return (a: T, b: T): number => {
        const folderDiff = Number(isFolder(b)) - Number(isFolder(a));
        return folderDiff !== 0 ? folderDiff : nameOf(a) > nameOf(b) ? 1 : -1;
    };
}

/** keysToTree 全量构建与 insertKeysToTree 增量插入共用，保证两者输出同序 */
const compareTreeNode = compareNodeByName<TreeNode>(
    (node) => node.type === 1,
    (node) => node.name
);
