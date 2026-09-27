/**
 * 取文件名的扩展名（小写，不含点）。
 *
 * 图标与编辑器语法两套映射都必须走这里：此前内容编辑器用 `path.endsWith('js')` 这类
 * 松散判断，会命中任何以 js 结尾的名字，且把 `.conf`、`.env` 等运维高频文件漏掉。
 *
 * 无扩展名（Dockerfile、Makefile）与隐藏文件（.gitignore，首个点不是分隔符）返回空串，
 * 由调用方按完整文件名兜底判定。
 */
export function fileExt(filename: string): string {
    const name = filename.split('/').pop() ?? filename;
    const dotIdx = name.lastIndexOf('.');
    // 首个点不是分隔符：无扩展名（Makefile）或隐藏文件（.gitignore）
    if (dotIdx <= 0) {
        return '';
    }
    return name.slice(dotIdx + 1).toLowerCase();
}

/** 取文件名本身（去掉目录），用于按 Dockerfile / Makefile 等完整文件名判定 */
export function baseName(path: string): string {
    return path.split('/').pop() ?? path;
}
