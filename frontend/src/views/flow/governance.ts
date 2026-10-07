/**
 * 把后端下发的「治理路径」换成「生效资源」标签树的可勾选节点。
 *
 * 治理粒度由后端各场景在自己的注册声明里给出（如数据库按库治理，粒度在 实例/凭证/库 这条路径上），
 * 前端只做拼接：这里曾经写过 `实例类型 → 换成库那条路径`、`库名类型 → 跳过` 两条分支，
 * 那是把「谁能被治理、在哪一层」这份事实复制到了前端——新接入 ES 索引、Mongo 集合这类
 * 粒度不同的资源时必须回来加分支，否则要么选不到节点、要么同一个资源在树上出现两个可勾点。
 *
 * 资源树接口的 type 参数同时接受单个资源类型与 `2/5/22` 这样的类型路径，
 * 因此单段路径直接给数字，多段路径给拼接后的字符串。
 *
 * @param governPaths schema 下发的治理路径集合；为空（请求还没回来）时返回 null，由调用方等 schema
 */
export function governTagTypesOf(governPaths?: number[][]): (number | string)[] | null {
    if (!governPathList(governPaths).length) {
        return null;
    }

    const tagTypes: (number | string)[] = [];
    for (const path of governPathList(governPaths)) {
        const nodeType = path.length === 1 ? path[0] : path.join('/');
        if (!tagTypes.includes(nodeType)) {
            tagTypes.push(nodeType);
        }
    }
    return tagTypes;
}

/** 后端可能给出 null 元素（历史数据或手工改库），拼接前先滤掉，免得 join 时抛错 */
function governPathList(governPaths?: number[][]): number[][] {
    return (governPaths ?? []).filter((path) => path?.length);
}
