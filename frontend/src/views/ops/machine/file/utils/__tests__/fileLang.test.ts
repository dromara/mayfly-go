import { describe, expect, it } from 'vitest';
import { baseName, fileExt } from '../fileExt';
import { getFileLanguage } from '../fileLang';

describe('fileExt / baseName', () => {
    it('取最后一个点之后的部分并小写', () => {
        expect(fileExt('report.DOC')).toBe('doc');
        expect(fileExt('archive.tar.gz')).toBe('gz');
        expect(fileExt('/etc/my.cnf')).toBe('cnf');
    });

    it('无扩展名与隐藏文件返回空串，不把整个文件名当扩展名', () => {
        // 旧实现用 split('.').pop()，'Makefile' 会被当成扩展名 'makefile'
        expect(fileExt('Makefile')).toBe('');
        expect(fileExt('.gitignore')).toBe('');
        expect(fileExt('LICENSE')).toBe('');
    });

    it('baseName 去掉目录部分', () => {
        expect(baseName('/usr/local/nginx/conf/nginx.conf')).toBe('nginx.conf');
        expect(baseName('nginx.conf')).toBe('nginx.conf');
    });
});

describe('getFileLanguage', () => {
    it('运维高频配置文件按扩展名命中，不再全落回纯文本', () => {
        expect(getFileLanguage('redis.conf')).toBe('ini');
        expect(getFileLanguage('/etc/nginx/nginx.conf')).toBe('ini');
        expect(getFileLanguage('my.cnf')).toBe('ini');
        expect(getFileLanguage('app.yml')).toBe('yaml');
        expect(getFileLanguage('crontab.bak.sh')).toBe('shell');
        expect(getFileLanguage('main.go')).toBe('go');
        expect(getFileLanguage('App.tsx')).toBe('typescript');
    });

    it('无扩展名文件按完整文件名判定', () => {
        expect(getFileLanguage('/build/Dockerfile')).toBe('dockerfile');
        expect(getFileLanguage('Makefile')).toBe('makefile');
        expect(getFileLanguage('.env')).toBe('ini');
        expect(getFileLanguage('/etc/crontab')).toBe('shell');
    });

    it('不再用 endsWith 松散匹配：以 js 结尾但不是 js 的文件不误判', () => {
        // 旧实现 path.endsWith('js') 会把 'myjs' 判成 javascript
        expect(getFileLanguage('myjs')).toBe('plaintext');
        expect(getFileLanguage('notes')).toBe('plaintext');
        expect(getFileLanguage('app.js')).toBe('javascript');
    });

    it('未知类型兜底 plaintext', () => {
        expect(getFileLanguage('binary.bin')).toBe('plaintext');
        expect(getFileLanguage('README')).toBe('plaintext');
    });
});
