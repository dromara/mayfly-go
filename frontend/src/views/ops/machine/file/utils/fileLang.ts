import { baseName, fileExt } from './fileExt';

// 无扩展名但有确定语法的运维高频文件，按完整文件名判定
const LANGUAGE_BY_NAME: Record<string, string> = {
    Dockerfile: 'dockerfile',
    Makefile: 'makefile',
    Jenkinsfile: 'groovy',
    '.env': 'ini',
    '.gitignore': 'ini',
    '.editorconfig': 'ini',
    crontab: 'shell',
};

// 扩展名 → Monaco 语言 id。取值以运维机器上真正常见的配置文件为准
const LANGUAGE_BY_EXT: Record<string, string> = {
    sh: 'shell',
    bash: 'shell',
    zsh: 'shell',
    // nginx / redis / mysqld / php-fpm 等绝大多数机器配置都是 ini 家族
    conf: 'ini',
    ini: 'ini',
    cnf: 'ini',
    properties: 'properties',
    toml: 'ini',
    env: 'ini',
    yml: 'yaml',
    yaml: 'yaml',
    json: 'json',
    log: 'plaintext',
    txt: 'plaintext',
    md: 'markdown',
    sql: 'sql',
    xml: 'xml',
    html: 'html',
    htm: 'html',
    css: 'css',
    scss: 'scss',
    js: 'javascript',
    cjs: 'javascript',
    mjs: 'javascript',
    ts: 'typescript',
    jsx: 'javascript',
    tsx: 'typescript',
    py: 'python',
    go: 'go',
    java: 'java',
    php: 'php',
    rb: 'ruby',
    lua: 'lua',
};

/**
 * 按文件路径推断编辑器语言。
 *
 * 判定顺序：完整文件名 → 扩展名 → 兜底 plaintext。扩展名解析与文件图标共用
 * utils/fileExt，避免同一件事在两处用两套规则。
 */
export function getFileLanguage(path: string): string {
    const name = baseName(path);
    if (LANGUAGE_BY_NAME[name]) {
        return LANGUAGE_BY_NAME[name];
    }
    const ext = fileExt(name);
    return LANGUAGE_BY_EXT[ext] ?? 'plaintext';
}
