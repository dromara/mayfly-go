export class TagTreeNode {
    /**
     * 节点id
     */
    key: any;

    /**
     * 节点名称
     */
    label: string;

    /**
     * 树节点类型
     */
    type: any;

    isLeaf: boolean = false;

    params: any;

    static TagPath = -1;

    constructor(key: any, label: string, type?: any) {
        this.key = key;
        this.label = label;
        this.type = type || TagTreeNode.TagPath;
    }

    withIsLeaf(isLeaf: boolean) {
        this.isLeaf = isLeaf;
        return this;
    }

    withParams(params: any) {
        this.params = params;
        return this;
    }
}
