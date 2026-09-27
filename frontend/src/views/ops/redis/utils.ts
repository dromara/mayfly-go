/**
 * key 名列表 → 分组树：把 `a:b:c` 这类 key 按分隔符折叠成目录层。
 *
 * 树模型里有两种节点：目录（继续嵌套）与 key（叶子）。同一段名既是目录又是 key 的场景
 * （目录 `aa` 与 key `aa`）靠 key 叶子的魔法键后缀区分，段名本身不会带上后缀，因此判别无歧义；
 * 每层目录用无原型对象承载，key 里出现 `__proto__`、`constructor` 这类段名时不会污染对象原型
 */

interface TreeNode {
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

    return formatTreeData(tree, '', separator, openStatus);
}

export function keysToList(keys: string[]) {
    return keys.map((key: string) => ({ key, name: key }));
}

function formatTreeData(tree: TreeFolder, previousKey: string = '', separator: string = ':', openStatus: Set<string> | null = null): TreeNode[] {
    return Object.keys(tree).map((name) => {
        if (isFolderName(name) && Object.keys(tree[name]).length > 0) {
            const node = tree[name] as TreeFolder;
            const tillNowKeyName = previousKey + name + separator;
            const children = formatTreeData(node, tillNowKeyName, separator, openStatus);
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
                // 只有展开过的目录需要立即有序（el-tree 渲染读的就是这份顺序）；
                // 未展开的等展开时由 RedisDataOp 对渲染节点补排，避免整棵树白排一遍
                children.sort(
                    compareNodeByName(
                        (child) => !!child.children,
                        (child) => child.name
                    )
                );
            }
            return treeNode;
        }

        const keyName = isFolderName(name) ? name : name.slice(0, -KEY_SUFFIX.length);
        return { name: keyName || '[Empty]', type: 2, key: keyName };
    });
}

/**
 * 目录在前、key 在后，同组内按名字典序：构建树时与 el-tree 已展开的子节点共用同一排序语义
 */
function compareNodeByName<T>(isFolder: (node: T) => boolean, nameOf: (node: T) => string) {
    return (a: T, b: T): number => {
        const folderDiff = Number(isFolder(b)) - Number(isFolder(a));
        return folderDiff !== 0 ? folderDiff : nameOf(a) > nameOf(b) ? 1 : -1;
    };
}

/** el-tree 展开后子节点是渲染好的 node（isLeaf/label），排序必须作用到它身上才能改变显示顺序 */
export function sortByTreeNodes(nodes: { isLeaf: boolean; label: string }[]) {
    nodes.sort(
        compareNodeByName(
            (node) => !node.isLeaf,
            (node) => node.label
        )
    );
}
