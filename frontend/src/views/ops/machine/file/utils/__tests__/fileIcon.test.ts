import { describe, expect, it } from 'vitest';
import { getFileIcon } from '../fileIcon';

describe('getFileIcon', () => {
    it('Word 文档', () => {
        expect(getFileIcon('report.doc')).toBe('icon file/word');
        expect(getFileIcon('report.docx')).toBe('icon file/word');
    });

    it('Excel 文件', () => {
        expect(getFileIcon('data.xls')).toBe('icon file/excel');
        expect(getFileIcon('data.xlsx')).toBe('icon file/excel');
    });

    it('PPT 文件', () => {
        expect(getFileIcon('slides.pptx')).toBe('icon file/ppt');
    });

    it('PDF 文件', () => {
        expect(getFileIcon('manual.pdf')).toBe('icon file/pdf');
    });

    it('配置文件', () => {
        expect(getFileIcon('config.yaml')).toBe('icon file/yaml');
        expect(getFileIcon('config.yml')).toBe('icon file/yaml');
        expect(getFileIcon('page.html')).toBe('icon file/html');
        expect(getFileIcon('style.css')).toBe('icon file/css');
    });

    it('脚本文件', () => {
        expect(getFileIcon('app.js')).toBe('icon file/js');
        expect(getFileIcon('app.ts')).toBe('icon file/js');
    });

    it('媒体文件', () => {
        expect(getFileIcon('movie.mp4')).toBe('icon file/video');
        expect(getFileIcon('song.mp3')).toBe('icon file/audio');
    });

    it('图片文件', () => {
        expect(getFileIcon('photo.jpg')).toBe('icon file/image');
        expect(getFileIcon('photo.PNG')).toBe('icon file/image');
        expect(getFileIcon('icon.svg')).toBe('icon file/image');
    });

    it('文本文件', () => {
        expect(getFileIcon('README.md')).toBe('icon file/md');
        expect(getFileIcon('notes.txt')).toBe('icon file/txt');
    });

    it('压缩文件', () => {
        expect(getFileIcon('archive.zip')).toBe('icon file/zip');
        expect(getFileIcon('archive.tar.gz')).toBe('icon file/zip');
        expect(getFileIcon('archive.7z')).toBe('icon file/zip');
    });

    it('未知扩展名返回默认图标', () => {
        expect(getFileIcon('file.unknown')).toBe('icon file/file');
        expect(getFileIcon('noextension')).toBe('icon file/file');
    });

    it('扩展名大小写不敏感', () => {
        expect(getFileIcon('DOC.PDF')).toBe('icon file/pdf');
    });
});
