/**
 * 机器文件模块的协议常量。
 *
 * 这些取值来自后端 `ls` 的输出形态与权限码，视图层与 composable 必须共用同一份：
 * 各写一份时改一处漏一处（路径分隔符曾同时在两处定义），而漏改表现为拼出错误路径而非报错。
 */

/** 机器侧路径分隔符（当前仅支持 Linux 语义） */
export const PATH_SEP = '/';

/** `ls -l` 首列类型标记：目录 */
export const DIR_TYPE = 'd';

/** `ls -l` 首列类型标记：普通文件 */
export const FILE_TYPE = '-';

/**
 * 系统关键目录：禁止删除/复制/移动。
 *
 * 精确匹配（不是前缀匹配），因此 `/usr` 受保护但 `/usr/local` 可操作；
 * 尾部带斜杠与不带的两种形态都列出，与后端返回的路径保持一致。
 */
export const PROTECTED_PATHS: ReadonlySet<string> = new Set([
    '/',
    '//',
    '/usr',
    '/usr/',
    '/usr/bin',
    '/opt',
    '/run',
    '/etc',
    '/proc',
    '/var',
    '/mnt',
    '/boot',
    '/dev',
    '/home',
    '/media',
    '/root',
]);

/** 文件操作权限码，与 server/internal/machine/api/machine_file.go 的 RequiredPermissionCode 对应 */
export const FILE_PERM = {
    add: 'machine:file:add',
    upload: 'machine:file:upload',
    write: 'machine:file:write',
    rm: 'machine:file:rm',
    del: 'machine:file:del',
} as const;

/** 预览内容的大小上限，超过则提示下载查看（与后端读文件接口的大文件保护同数量级） */
export const PREVIEW_MAX_SIZE = 1 * 1024 * 1024;
