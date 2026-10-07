/**
 * cloneNodeProperties 深拷贝流程图节点的属性集合。
 *
 * 属性里含条件树、审批人数组等嵌套结构：抽屉表单若与画布节点共享这些引用，
 * 编辑中途的就地改动（改逻辑关系、加删条件行）就会立刻污染节点数据，
 * 点「取消」也回不去，之后一次保存流程会把这些已取消的改动静默写进流程定义。
 * 流程定义本身就是这份数据的 JSON 序列化，因此 JSON 往返足以覆盖全部嵌套结构。
 */
export function cloneNodeProperties(properties?: Record<string, unknown> | null): Record<string, unknown> {
    return properties ? (JSON.parse(JSON.stringify(properties)) as Record<string, unknown>) : {};
}
