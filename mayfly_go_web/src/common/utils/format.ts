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

/**
 * 格式化json字符串
 * @param txt  json字符串
 * @param compress 是否压缩
 * @returns 格式化后的字符串
 */
export function formatJsonString(txt: string, compress: boolean) {
    var indentChar = '    ';
    if (/^\s*$/.test(txt)) {
        console.log('数据为空,无法格式化! ');
        return txt;
    }
    try {
        var data = JSON.parse(txt);
    } catch (e: any) {
        console.log('数据源语法错误,格式化失败! 错误信息: ' + e.description, 'err');
        return txt;
    }
    var draw: any = [],
        line = compress ? '' : '\n',
        // eslint-disable-next-line no-unused-vars
        nodeCount: number = 0,
        // eslint-disable-next-line no-unused-vars
        maxDepth: number = 0;

    var notify = function (name: any, value: any, isLast: any, indent: any, formObj: any) {
        nodeCount++; /*节点计数*/
        for (var i = 0, tab = ''; i < indent; i++) tab += indentChar; /* 缩进HTML */
        tab = compress ? '' : tab; /*压缩模式忽略缩进*/
        maxDepth = ++indent; /*缩进递增并记录*/
        if (value && value.constructor == Array) {
            /*处理数组*/
            draw.push(tab + (formObj ? '"' + name + '": ' : '') + '[' + line); /*缩进'[' 然后换行*/
            for (var i = 0; i < value.length; i++) notify(i, value[i], i == value.length - 1, indent, false);
            draw.push(tab + ']' + (isLast ? line : ',' + line)); /*缩进']'换行,若非尾元素则添加逗号*/
        } else if (value && typeof value == 'object') {
            /*处理对象*/
            draw.push(tab + (formObj ? '"' + name + '": ' : '') + '{' + line); /*缩进'{' 然后换行*/
            var len = 0,
                i = 0;
            for (var key in value) len++;
            for (var key in value) notify(key, value[key], ++i == len, indent, true);
            draw.push(tab + '}' + (isLast ? line : ',' + line)); /*缩进'}'换行,若非尾元素则添加逗号*/
        } else {
            if (typeof value == 'string') value = '"' + value + '"';
            draw.push(tab + (formObj ? '"' + name + '": ' : '') + value + (isLast ? '' : ',') + line);
        }
    };
    var isLast = true,
        indent = 0;
    notify('', data, isLast, indent, false);
    return draw.join('');
}