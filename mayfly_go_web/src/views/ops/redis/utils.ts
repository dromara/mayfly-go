export function keysToTree(keys: any, separator: string = ':', openStatus: any = null, forceCut = 20000) {
    const tree = {};
    keys.forEach((key: any) => {
        let currentNode = tree;
        const keyStr = key;
        const keySplited = keyStr.split(separator);
        const lastIndex = keySplited.length - 1;

        keySplited.forEach((value: string, index: number) => {
            // key node
            if (index === lastIndex) {
                currentNode[`${keyStr}\`k\``] = {
                    keyNode: true,
                    nameBuffer: key,
                };
            }
            // folder node
            else {
                currentNode[value] === undefined && (currentNode[value] = {});
            }

            currentNode = currentNode[value];
        });
    });

    // to tree format
    return formatTreeData(tree, '', separator, openStatus, forceCut);
}

export function keysToList(keys: any, separator: string = ':', openStatus: any = null, forceCut = 20000) {
    return keys.map((x: string) => {
        return {
            key: x,
            name: x,
        };
    });
}

function formatTreeData(tree: any, previousKey: string = '', separator: string = ':', openStatus: any = null, forceCut: number = 20000) {
    return Object.keys(tree).map((key) => {
        const node = { name: key || '[Empty]' } as any;

        // folder node
        if (!tree[key].keyNode && Object.keys(tree[key]).length > 0) {
            // fullName
            const tillNowKeyName = previousKey + key + separator;

            node.type = 1;
            // folder's fullName may same with key name, such as 'aa-'
            node.key = tillNowKeyName;
            if (openStatus) {
                node.open = openStatus?.has(node.key);
            }
            node.children = formatTreeData(tree[key], tillNowKeyName, separator, openStatus, forceCut);
            node.keyCount = node.children.reduce((a: any, b: any) => a + (b.keyCount || 1), 0);
            // too many children, force cut, do not incluence keyCount display
            // node.open && node.children.length > forceCut && node.children.splice(forceCut);
            // keep folder node in front of the tree and sorted(not include the outest list)
            // async sort, only for opened folders
            node.open && sortKeysAndFolder(node.children);
            node.fullName = tillNowKeyName;
            return node;
        }

        node.type = 2;
        // key node
        node.name = key.replace(/`k`$/, '');
        // node.nameBuffer = tree[key].nameBuffer.toJSON();
        node.key = node.name;

        return node;
    });
}

export function sortKeysAndFolder(nodes: any) {
    nodes.sort((a: any, b: any) => {
        // a & b are all keys
        if (!a.children && !b.children) {
            return a.name > b.name ? 1 : -1;
        }
        // a & b are all folder
        if (a.children && b.children) {
            return a.name > b.name ? 1 : -1;
        }

        // a is folder, b is key
        if (a.children) {
            return -1;
        }
        // a is key, b is folder

        return 1;
    });
}

// sortByTreeNode
export function sortByTreeNodes(nodes: any) {
    nodes.sort((a: any, b: any) => {
        // a & b are all keys
        if (a.isLeaf && b.isLeaf) {
            return a.label > b.label ? 1 : -1;
        }
        // a & b are all folder
        if (!a.isLeaf && !b.isLeaf) {
            return a.label > b.label ? 1 : -1;
        }

        // a is folder, b is key
        if (!a.isLeaf) {
            return -1;
        }
        // a is key, b is folder

        return 1;
    });
}
