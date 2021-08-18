/**
 * 格式化字节单位
 * @param size byte size
 * @returns 
 */
export function formatByteSize(size: any) {
    const value = Number(size);
        if (size && !isNaN(value)) {
            const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB', 'BB'];
            let index = 0;
            let k = value;
            if (value >= 1024) {
                while (k > 1024) {
                    k = k / 1024;
                    index++;
                }
            }
            return `${k.toFixed(2)}${units[index]}`;
        }
        return '-';
}