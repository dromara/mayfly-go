/**
 * 根据文件扩展名返回对应的图标名称
 * @param filename 文件名
 * @returns 图标名称
 */
export function getFileIcon(filename: string): string {
    const fileExtension = filename.split('.').pop()?.toLowerCase() ?? '';

    switch (fileExtension) {
        case 'doc':
        case 'docx':
            return 'icon file/word';
        case 'xls':
        case 'xlsx':
            return 'icon file/excel';
        case 'ppt':
        case 'pptx':
            return 'icon file/ppt';
        case 'pdf':
            return 'icon file/pdf';
        case 'xml':
            return 'icon file/xml';
        case 'html':
            return 'icon file/html';
        case 'yaml':
        case 'yml':
            return 'icon file/yaml';
        case 'css':
            return 'icon file/css';
        case 'js':
        case 'ts':
            return 'icon file/js';
        case 'mp4':
        case 'rmvb':
            return 'icon file/video';
        case 'mp3':
            return 'icon file/audio';
        case 'bmp':
        case 'jpg':
        case 'jpeg':
        case 'png':
        case 'tif':
        case 'gif':
        case 'pcx':
        case 'tga':
        case 'exif':
        case 'svg':
        case 'psd':
        case 'ai':
        case 'webp':
            return 'icon file/image';
        case 'md':
            return 'icon file/md';
        case 'txt':
            return 'icon file/txt';
        case 'zip':
        case 'rar':
        case '7z':
        case 'gz':
        case 'tar':
        case 'tgz':
            return 'icon file/zip';
        default:
            return 'icon file/file';
    }
}
