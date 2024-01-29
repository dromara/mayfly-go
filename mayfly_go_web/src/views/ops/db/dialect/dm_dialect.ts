import { DbInst } from '../db';
import {
    commonCustomKeywords,
    DataType,
    DbDialect,
    DialectInfo,
    EditorCompletion,
    EditorCompletionItem,
    IndexDefinition,
    RowDefinition,
    sqlColumnType,
} from './index';
import { language as sqlLanguage } from 'monaco-editor/esm/vs/basic-languages/sql/sql.js';

export { DMDialect, DM_TYPE_LIST };

// 参考文档:https://eco.dameng.com/document/dm/zh-cn/sql-dev/dmpl-sql-datatype.html#%E5%AD%97%E7%AC%A6%E6%95%B0%E6%8D%AE%E7%B1%BB%E5%9E%8B
const DM_TYPE_LIST: sqlColumnType[] = [
    // 字符数据类型
    { udtName: 'CHAR', dataType: 'VARCHAR', desc: '定长字符串', space: '', range: '1 - 32767' },
    { udtName: 'VARCHAR', dataType: 'VARCHAR', desc: '变长字符串', space: '', range: '1 - 32767' },

    // 精确数值数据类型 NUMERIC、DECIMAL、DEC 类型、NUMBER 类型、INTEGER 类型、INT 类型、BIGINT 类型、TINYINT 类型、BYTE 类型、SMALLINT
    { udtName: 'NUMERIC', dataType: 'NUMERIC', desc: '零、正负定点数', space: '1-38', range: '' },
    { udtName: 'DECIMAL', dataType: 'DECIMAL', desc: '与NUMERIC相似', space: '1-38', range: '' },
    { udtName: 'NUMBER', dataType: 'NUMBER', desc: '同NUMERIC', space: '1-38', range: '' },
    { udtName: 'INTEGER', dataType: 'INTEGER', desc: '有符号整数', space: '10', range: '-2^31-1 ~ 2^31-1' },
    { udtName: 'INT', dataType: 'INT', desc: '同INTEGER', space: '10', range: '' },
    { udtName: 'BIGINT', dataType: 'BIGINT', desc: '有符号整数', space: '19', range: '-2^63-1 ~ 2^63-1' },
    { udtName: 'TINYINT', dataType: 'TINYINT', desc: '有符号整数', space: '3', range: '-128~+127' },
    { udtName: 'BYTE', dataType: 'BYTE', desc: '与 TINYINT 相似', space: '3', range: '' },
    { udtName: 'SMALLINT', dataType: 'SMALLINT', desc: '有符号整数', space: '5', range: '-2^15-1 ~ 2^15-1' },
    // (用得少，忽略)近似数值类型包括：FLOAT 类型、DOUBLE 类型、REAL 类型、DOUBLE PRECISION 类型。
    // 位串数据类型 BIT 用于存储整数数据 1、0 或 NULL，只有 0 才转换为假，其他非空、非 0 值都会自动转换为真
    { udtName: 'BIT', dataType: 'BIT', desc: '用于存储整数数据 1、0 或 NULL', space: '1', range: '1' },
    // 一般日期时间数据类型 DATE TIME TIMESTAMP 默认精度 6
    // 多媒体数据类型 TEXT/LONG/LONGVARCHAR 类型：变长字符串类型  IMAGE/LONGVARBINARY 类型  BLOB CLOB BFILE  100G-1
    { udtName: 'DATE', dataType: 'DATE', desc: '年、月、日', space: '', range: '' },
    { udtName: 'TIME', dataType: 'TIME', desc: '时、分、秒', space: '', range: '' },
    {
        udtName: 'TIMESTAMP',
        dataType: 'TIMESTAMP',
        desc: '年、月、日、时、分、秒',
        space: '',
        range: '-4712-01-01 00:00:00.000000000 ~ 9999-12-31 23:59:59.999999999',
    },
    { udtName: 'TEXT', dataType: 'TEXT', desc: '变长字符串', space: '', range: '100G-1' },
    { udtName: 'LONG', dataType: 'LONG', desc: '同TEXT', space: '', range: '100G-1' },
    { udtName: 'LONGVARCHAR', dataType: 'LONGVARCHAR', desc: '同TEXT', space: '', range: '100G-1' },
    { udtName: 'IMAGE', dataType: 'IMAGE', desc: '图像二进制类型', space: '', range: '100G-1' },
    { udtName: 'LONGVARBINARY', dataType: 'LONGVARBINARY', desc: '同IMAGE', space: '', range: '100G-1' },
    { udtName: 'BLOB', dataType: 'BLOB', desc: '变长的二进制大对象', space: '', range: '100G-1' },
    { udtName: 'CLOB', dataType: 'CLOB', desc: '同TEXT', space: '', range: '100G-1' },
    { udtName: 'BFILE', dataType: 'BFILE', desc: '二进制文件', space: '', range: '100G-1' },
];

// 参考官方文档：https://eco.dameng.com/document/dm/zh-cn/pm/function.html
const replaceFunctions: EditorCompletionItem[] = [
    //  数值函数
    { label: 'ABS', insertText: 'ABS(n)', description: '求数值 n 的绝对值' },
    { label: 'ACOS', insertText: 'ACOS(n)', description: '求数值 n 的反余弦值' },
    { label: 'ASIN', insertText: 'ASIN(n)', description: '求数值 n 的反正弦值' },
    { label: 'ATAN', insertText: 'ATAN(n)', description: '求数值 n 的反正切值' },
    { label: 'ATAN2', insertText: 'ATAN2(n1,n2)', description: '求数值 n1/n2 的反正切值' },
    { label: 'CEIL', insertText: 'CEIL(n)', description: '求大于或等于数值 n 的最小整数' },
    { label: 'CEILING', insertText: 'CEILING(n)', description: '求大于或等于数值 n 的最小整数，等价于 CEIL(n)' },
    { label: 'COS', insertText: 'COS(n)', description: '求数值 n 的余弦值' },
    { label: 'COSH', insertText: 'COSH(n)', description: '求数值 n 的双曲余弦值' },
    { label: 'COT', insertText: 'COT(n)', description: '求数值 n 的余切值' },
    { label: 'DEGREES', insertText: 'DEGREES(n)', description: '求弧度 n 对应的角度值' },
    { label: 'EXP', insertText: 'EXP(n)', description: '求数值 n 的自然指数' },
    { label: 'FLOOR', insertText: 'FLOOR(n)', description: '求小于或等于数值 n 的最大整数' },
    { label: 'GREATEST', insertText: 'GREATEST(n, n2)', description: '求一个或多个数中最大的一个' },
    { label: 'GREAT', insertText: 'GREAT(n1, n2)', description: '求 n1、n2 两个数中最大的一个' },
    { label: 'LEAST', insertText: 'LEAST(n, n2)', description: '求一个或多个数中最小的一个' },
    { label: 'LN', insertText: 'LN(n)', description: '求数值 n 的自然对数' },
    { label: 'LOG', insertText: 'LOG(n1,n2)', description: '求数值 n2 以 n1 为底数的对数' },
    { label: 'LOG10', insertText: 'LOG10(n)', description: '求数值 n 以 10 为底的对数' },
    { label: 'MOD', insertText: 'MOD(m,n)', description: '求数值 m 被数值 n 除的余数' },
    { label: 'PI', insertText: 'PI()', description: '得到常数 π' },
    { label: 'POWER', insertText: 'POWER(n1,n2)', description: '求数值 n2 以 n1 为基数的指数' },
    { label: 'RADIANS', insertText: 'RADIANS(n)', description: '求角度 n 对应的弧度值' },
    { label: 'RAND', insertText: 'RAND', description: '求一个 0 到 1 之间的随机浮点数' },
    { label: 'ROUND', insertText: 'ROUND(n [,m]})', description: '返回四舍五入到小数点后面 m 位的 n 值' },
    { label: 'SIGN', insertText: 'SIGN(n)', description: '判断数值的数学符号' },
    { label: 'SIN', insertText: 'SIN(n)', description: '求数值 n 的正弦值' },
    { label: 'SINH', insertText: 'SINH(n)', description: '求数值 n 的双曲正弦值' },
    { label: 'SQRT', insertText: 'SQRT(n)', description: '求数值 n 的平方根' },
    { label: 'TAN', insertText: 'TAN(n)', description: '求数值 n 的正切值' },
    { label: 'TANH', insertText: 'TANH(n)', description: '求数值 n 的双曲正切值' },
    { label: 'TO_NUMBER', insertText: 'TO_NUMBER (char [,fmt])', description: '将 CHAR、VARCHAR、VARCHAR2 等类型的字符串转换为 DECIMAL 类型的数值' },
    { label: 'TRUNC', insertText: 'TRUNC(n [,m])', description: "截取数值函数，str 内只能为数字和'-', '+', '.' 的组合" },
    { label: 'TRUNCATE', insertText: 'TRUNCATE(n [,m])', description: '截取数值函数，等价于 TRUNC 函数' },
    { label: 'TO_CHAR', insertText: 'TO_CHAR(n [, fmt])', description: '将数值类型的数据转换为 VARCHAR 类型输出' },
    { label: 'BITAND', insertText: 'BITAND(n1, n2)', description: '求两个数值型数值按位进行 AND 运算的结果' },
    { label: 'NANVL', insertText: 'NANVL(n1, n2)', description: '有一个参数为空则返回空，否则返回 n1 的值' },
    { label: 'REMAINDER', insertText: 'REMAINDER(n1, n2)', description: '计算 n1 除 n2 的余数，余数取绝对值更小的那一个' },
    { label: 'TO_BINARY_FLOAT', insertText: 'TO_BINARY_FLOAT(n)', description: '将 number、real 或 double 类型数值转换成 binary float 类型' },
    { label: 'TO_BINARY_DOUBLE', insertText: 'TO_BINARY_DOUBLE(n)', description: '将 number、real 或 float 类型数值转换成 binary double 类型' },
    // 字符串函数
    { label: 'ASCII', insertText: 'ASCII(char)', description: '返回字符对应的整数' },
    { label: 'ASCIISTR', insertText: 'ASCIISTR(char)', description: '将字符串 char 中，非 ASCII 的字符转成 \\XXXX(UTF-16)格式，ASCII 字符保持不变' },
    { label: 'BIT_LENGTH', insertText: 'BIT_LENGTH(char)', description: '求字符串的位长度' },
    { label: 'CHAR', insertText: 'CHAR(n)', description: '返回整数 n 对应的字符' },
    { label: 'CHAR_LENGTH', insertText: 'CHAR_LENGTH(char)', description: '求字符串的串长度' },
    { label: 'CHARACTER_LENGTH', insertText: 'CHARACTER_LENGTH(char)', description: '求字符串的串长度' },
    { label: 'CHR', insertText: 'CHR(n)', description: '返回整数 n 对应的字符，等价于 CHAR(n)' },
    { label: 'CONCAT', insertText: 'CONCAT(char1,char2,char3,...)', description: '顺序联结多个字符串成为一个字符串' },
    {
        label: 'DIFFERENCE',
        insertText: 'DIFFERENCE(char1,char2)',
        description: '返回两个值串同一位置出现相同字符的个数。',
    },
    { label: 'INITCAP', insertText: 'INITCAP(char)', description: '将字符串中单词的首字符转换成大写的字符' },
    {
        label: 'INS',
        insertText: 'INS(char1,begin,n,char2)',
        description: '删除在字符串 char1 中以 begin 参数所指位置开始的 n 个字符, 再把 char2 插入到 char1 串的 begin 所指位置',
    },
    {
        label: 'INSERT',
        insertText: 'INSERT(char1,n1,n2,char2)',
        description: '将字符串 char1 从 n1 的位置开始删除 n2 个字符，并将 char2 插入到 char1 中 n1 的位置',
    },
    {
        label: 'INSSTR',
        insertText: 'INSERT(char1,n1,n2,char2)',
        description: '将字符串 char1 从 n1 的位置开始删除 n2 个字符，并将 char2 插入到 char1 中 n1 的位置',
    },
    {
        label: 'INSTR',
        insertText: 'INSTR(char1,char2[,n,[m]])',
        description: '从输入字符串char1的第n个字符开始查找字符串 char2 的第 m 次出现的位置，以字符计算',
    },
    { label: 'INSTRB', insertText: 'INSTRB(char1,char2[,n,[m]])', description: '从 char1 的第 n 个字节开始查找字符串 char2 的第 m 次出现的位置，以字节计算' },
    { label: 'LCASE', insertText: 'LCASE(char)', description: '将大写的字符串转换为小写的字符串' },
    { label: 'LEFT', insertText: 'LEFT(char,n)', description: '返回字符串最左边的 n 个字符组成的字符串' },
    { label: 'LEFTSTR', insertText: 'LEFTSTR(char,n)', description: '返回字符串最左边的 n 个字符组成的字符串' },
    { label: 'LEN', insertText: 'LEN(char)', description: '返回给定字符串表达式的字符(而不是字节)个数（汉字为一个字符），其中不包含尾随空格' },
    { label: 'LENGTH', insertText: 'LENGTH(clob)', description: '返回给定字符串表达式的字符(而不是字节)个数（汉字为一个字符），其中包含尾随空格' },
    { label: 'OCTET_LENGTH', insertText: 'OCTET_LENGTH(char)', description: '返回输入字符串的字节数' },
    { label: 'LOCATE', insertText: 'LOCATE(char1,char2[,n])', description: '返回 char1 在 char2 中首次出现的位置' },
    { label: 'LOWER', insertText: 'LOWER(char)', description: '将大写的字符串转换为小写的字符串' },
    { label: 'LPAD', insertText: 'LPAD(char1,n,char2)', description: '在输入字符串的左边填充上 char2 指定的字符，将其拉伸至 n 个字节长度' },
    { label: 'LTRIM', insertText: 'LTRIM(str[,set])', description: '删除字符串 str 左边起，出现在 set 中的任何字符，当遇到不在 set 中的第一个字符时返回结果' },
    { label: 'POSITION', insertText: 'POSITION(char1, char2)', description: '求串 1 在串 2 中第一次出现的位置' },
    { label: 'REPEAT', insertText: 'REPEAT(char,n)', description: '返回将字符串重复 n 次形成的字符串' },
    { label: 'REPEATSTR', insertText: 'REPEATSTR(char,n)', description: '返回将字符串重复 n 次形成的字符串' },
    {
        label: 'REPLACE',
        insertText: 'REPLACE(str, search [,replace])',
        description: '将输入字符串 STR 中所有出现的字符串 search 都替换成字符串 replace ,其中 str 为 char、clob 或 text 类型',
    },
    { label: 'REPLICATE', insertText: 'REPLICATE(char,times)', description: '把字符串 char 自己复制 times 份' },
    { label: 'REVERSE', insertText: 'REVERSE(char)', description: '将字符串反序' },
    { label: 'RIGHT', insertText: 'RIGHT', description: '返回字符串最右边 n 个字符组成的字符串' },
    { label: 'RIGHTSTR', insertText: 'RIGHTSTR(char,n)', description: '返回字符串最右边 n 个字符组成的字符串' },
    { label: 'RPAD', insertText: 'RPAD(char1,n[,char2])', description: '类似 LPAD 函数，只是向右拉伸该字符串使之达到 n 个字节长度' },
    { label: 'RTRIM', insertText: 'RTRIM(str[,set])', description: '删除字符串str右边起出现的set中的任何字符，当遇到不在 set 中的第一个字符时返回结果' },
    { label: 'SOUNDEX', insertText: 'SOUNDEX(char)', description: '返回一个表示字符串发音的字符串' },
    { label: 'SPACE', insertText: 'SPACE(n)', description: '返回一个包含 n 个空格的字符串' },
    { label: 'STRPOSDEC', insertText: 'STRPOSDEC(char[,pos])', description: '把字符串 char 中指定位置 pos 上的字节值减一' },
    { label: 'STRPOSINC', insertText: 'STRPOSINC(char)', description: '把字符串 char 中最后一个字节的值加一' },
    { label: 'STRPOSINC', insertText: 'STRPOSINC(char,pos)', description: '把字符串 char 中指定位置 pos 上的字节值加一' },
    {
        label: 'STUFF',
        insertText: 'STUFF(char1,begin,n,char2)',
        description: '删除在字符串 char1 中以 begin 参数所指位置开始的 n 个字符, 再把 char2 插入到 char1 串的 begin 所指位置',
    },
    { label: 'SUBSTR', insertText: 'SUBSTR(char[,m[,n]])', description: '返回 char 中从字符位置 m 开始的 n 个字符' },
    { label: 'SUBSTRING', insertText: 'SUBSTRING(char [FROM m [FOR n]])', description: '返回 char 中从字符位置 m 开始的 n 个字符' },
    { label: 'SUBSTRB', insertText: 'SUBSTRB(char,m[,n])', description: 'SUBSTR 函数等价的单字节形式' },
    { label: 'TO_CHAR', insertText: 'TO_CHAR(character)', description: '将 VARCHAR、CLOB、TEXT 类型的数据转化为 VARCHAR 类型输出' },
    { label: 'TRANSLATE', insertText: 'TRANSLATE(char,from,to)', description: '将所有出现在搜索字符集中的字符转换成字符集中的相应字符' },
    { label: 'TRIM', insertText: 'TRIM([<<LEADING|TRAILING|BOTH>[char] | char> FROM] str)', description: '删去字符串 str 中由 char 指定的字符' },
    { label: 'UCASE', insertText: 'UCASE(char)', description: '将小写的字符串转换为大写的字符串' },
    { label: 'UPPER', insertText: 'UPPER(char)', description: '将小写的字符串转换为大写的字符串' },
    { label: 'NLS_UPPER', insertText: 'NLS_UPPER(char)', description: '将小写的字符串转换为大写的字符串' },
    { label: 'REGEXP', insertText: 'REGEXP', description: '根据符合 POSIX 标准的正则表达式进行字符串匹配' },
    {
        label: 'OVERLAY',
        insertText: 'OVERLAY(char1 PLACING char2 FROM int [FOR int])',
        description: '字符串覆盖函数，用 char2 覆盖 char1 中指定的子串，返回修改后的 char1',
    },
    { label: 'TEXT_EQUAL', insertText: 'TEXT_EQUAL(n1,n2)', description: '返回两个 LONGVARCHAR 类型的值的比较结果，相同返回 1，否则返回 0' },
    { label: 'BLOB_EQUAL', insertText: 'BLOB_EQUAL(n1,n2)', description: '返回两个 LONGVARBINARY 类型的值的比较结果，相同返回 1，否则返回 0' },
    { label: 'NLSSORT', insertText: 'NLSSORT(str1 [,nls_sort=str2])', description: '返回对自然语言排序的编码' },
    { label: 'GREATEST', insertText: 'GREATEST(char {,char})', description: '求一个或多个字符串中最大的字符串' },
    { label: 'GREAT', insertText: 'GREAT (char1, char2)', description: '求 char 1、char 2 中最大的字符串' },
    { label: 'TO_SINGLE_BYTE', insertText: 'TO_SINGLE_BYTE (char)', description: '将多字节形式的字符（串）转换为对应的单字节形式' },
    { label: 'TO_MULTI_BYTE', insertText: 'TO_MULTI_BYTE (char)', description: '将单字节形式的字符（串）转换为对应的多字节形式' },
    { label: 'EMPTY_CLOB', insertText: 'EMPTY_CLOB ()', description: '初始化 clob 字段' },
    { label: 'EMPTY_BLOB', insertText: 'EMPTY_BLOB ()', description: '初始化 blob 字段' },
    {
        label: 'UNISTR',
        insertText: 'UNISTR (char)',
        description: '将字符串 char 中，ASCII 编码或 Unicode 编码（‗XXXX‘4 个 16 进制字符格式）转成本地字符。对于其他字符保持不变',
    },
    { label: 'ISNULL', insertText: 'ISNULL(char)', description: '判断表达式是否为 NULL' },
    { label: 'CONCAT_WS', insertText: 'CONCAT_WS(delim, char1,char2,char3,…)', description: '顺序联结多个字符串成为一个字符串，并用 delim 分割' },
    { label: 'SUBSTRING_INDEX', insertText: 'SUBSTRING_INDEX(char, delim, count)', description: '按关键字截取字符串，截取到指定分隔符出现指定次数位置之前' },
    { label: 'COMPOSE', insertText: 'COMPOSE(varchar str)', description: '在 UTF8 库下，将 str 以本地编码的形式返回' },
    {
        label: 'FIND_IN_SET',
        insertText: 'FIND_IN_SET(str, strlist[,separator])',
        description: '查询 strlist 中是否包含 str，返回 str 在 strlist 中第一次出现的位置或 NULL',
    },
    { label: 'TRUNC', insertText: 'TRUNC(str1, str2)', description: '截取字符串函数' },
    //日期时间函数
    { label: 'ADD_DAYS', insertText: 'ADD_DAYS(date,n)', description: '返回日期加上 n 天后的新日期' },
    { label: 'ADD_MONTHS', insertText: 'ADD_MONTHS(date,n)', description: '在输入日期上加上指定的几个月返回一个新日期' },
    { label: 'ADD_WEEKS', insertText: 'ADD_WEEKS(date,n)', description: '返回日期加上 n 个星期后的新日期' },
    { label: 'CURDATE', insertText: 'CURDATE()', description: '返回系统当前日期' },
    { label: 'CURTIME', insertText: 'CURTIME(n)', description: '返回系统当前时间' },
    { label: 'CURRENT_DATE', insertText: 'CURRENT_DATE()', description: '返回系统当前日期' },
    { label: 'CURRENT_TIME', insertText: 'CURRENT_TIME(n)', description: '返回系统当前时间' },
    { label: 'CURRENT_TIMESTAMP', insertText: 'CURRENT_TIMESTAMP(n)', description: '返回系统当前带会话时区信息的时间戳' },
    { label: 'DATEADD', insertText: 'DATEADD(datepart,n,date)', description: '向指定的日期加上一段时间' },
    { label: 'DATEDIFF', insertText: 'DATEDIFF(datepart,date1,date2)', description: '返回跨两个指定日期的日期和时间边界数' },
    { label: 'DATEPART', insertText: 'DATEPART(datepart,date)', description: '返回代表日期的指定部分的整数' },
    { label: 'DAY', insertText: 'DAY(date)', description: '返回日期中的天数' },
    { label: 'DAYNAME', insertText: 'DAYNAME(date)', description: '返回日期的星期名称' },
    { label: 'DAYOFMONTH', insertText: 'DAYOFMONTH(date)', description: '返回日期为所在月份中的第几天' },
    { label: 'DAYOFWEEK', insertText: 'DAYOFWEEK(date)', description: '返回日期为所在星期中的第几天' },
    { label: 'DAYOFYEAR', insertText: 'DAYOFYEAR(date)', description: '返回日期为所在年中的第几天' },
    { label: 'DAYS_BETWEEN', insertText: 'DAYS_BETWEEN(date1,date2)', description: '返回两个日期之间的天数' },
    { label: 'EXTRACT', insertText: 'EXTRACT(时间字段 FROM date)', description: '抽取日期时间或时间间隔类型中某一个字段的值' },
    { label: 'GETDATE', insertText: 'GETDATE(n)', description: '返回系统当前时间戳' },
    { label: 'GREATEST', insertText: 'GREATEST(date {,date})', description: '求一个或多个日期中的最大日期' },
    { label: 'GREAT', insertText: 'GREAT(date1,date2)', description: '求 date1、date2 中的最大日期' },
    { label: 'HOUR', insertText: 'HOUR(time)', description: '返回时间中的小时分量' },
    { label: 'LAST_DAY', insertText: 'LAST_DAY(date)', description: '返回输入日期所在月份最后一天的日期' },
    { label: 'LEAST', insertText: 'LEAST(date {,date})', description: '求一个或多个日期中的最小日期' },
    { label: 'MINUTE', insertText: 'MINUTE(time)', description: '返回时间中的分钟分量' },
    { label: 'MONTH', insertText: 'MONTH(date)', description: '返回日期中的月份分量' },
    { label: 'MONTHNAME', insertText: 'MONTHNAME(date)', description: '返回日期中月分量的名称' },
    { label: 'MONTHS_BETWEEN', insertText: 'MONTHS_BETWEEN(date1,date2)', description: '返回两个日期之间的月份数' },
    { label: 'NEXT_DAY', insertText: 'NEXT_DAY(date1,char2)', description: '返回输入日期指定若干天后的日期' },
    { label: 'NOW', insertText: 'NOW(n)', description: '返回系统当前时间戳' },
    { label: 'QUARTER', insertText: 'QUARTER(date)', description: '返回日期在所处年中的季节数' },
    { label: 'SECOND', insertText: 'SECOND(time)', description: '返回时间中的秒分量' },
    { label: 'ROUND', insertText: 'ROUND(date1[, fmt])', description: '把日期四舍五入到最接近格式元素指定的形式' },
    { label: 'TIMESTAMPADD', insertText: 'TIMESTAMPADD(datepart,n,timestamp)', description: '返回时间戳 timestamp 加上 n 个 datepart 指定的时间段的结果' },
    {
        label: 'TIMESTAMPDIFF',
        insertText: 'TIMESTAMPDIFF(datepart,timestamp1,timestamp2)',
        description: '返回一个表明timestamp2与timestamp1之间的指定 datepart 类型时间间隔的整数',
    },
    { label: 'SYSDATE', insertText: 'SYSDATE()', description: '返回系统的当前日期' },
    { label: 'TO_DATE', insertText: "TO_DATE(CHAR[,fmt[,'nls']])", description: '字符串转换为日期时间数据类型' },
    {
        label: 'FROM_TZ',
        insertText: 'FROM_TZ(timestamp,timezonetz_name])',
        description: '将时间戳类型 timestamp 和时区类型 timezone（或时区名称 tz_name ） 转 化 为 timestamp with timezone 类型',
    },
    { label: 'TZ_OFFSET', insertText: 'TZ_OFFSET(timezone|[tz_name])', description: '返回给定的时区或时区名和标准时区(UTC)的偏移量' },
    { label: 'TRUNC', insertText: 'TRUNC(date[,fmt])', description: '把日期截断到最接近格式元素指定的形式' },
    { label: 'WEEK', insertText: 'WEEK(date)', description: '返回日期为所在年中的第几周' },
    { label: 'WEEKDAY', insertText: 'WEEKDAY(date)', description: '返回当前日期的星期值' },
    { label: 'WEEKS_BETWEEN', insertText: 'WEEKS_BETWEEN(date1,date2)', description: '返回两个日期之间相差周数' },
    { label: 'YEAR', insertText: 'YEAR(date)', description: '返回日期的年分量' },
    { label: 'YEARS_BETWEEN', insertText: 'YEARS_BETWEEN(date1,date2)', description: '返回两个日期之间相差年数' },
    { label: 'LOCALTIME', insertText: 'LOCALTIME(n)', description: '返回系统当前时间' },
    { label: 'LOCALTIMESTAMP', insertText: 'LOCALTIMESTAMP(n)', description: '返回系统当前时间戳' },
    { label: 'OVERLAPS', insertText: 'OVERLAPS', description: '返回两个时间段是否存在重叠' },
    {
        label: 'TO_CHAR',
        insertText: 'TO_CHAR(date[,fmt[,nls]])',
        description: '将日期数据类型 DATE 转换为一个在日期语法 fmt 中指定语法的 VARCHAR 类型字符串。',
    },
    { label: 'SYSTIMESTAMP', insertText: 'SYSTIMESTAMP(n)', description: '返回系统当前带数据库时区信息的时间戳' },
    { label: 'NUMTODSINTERVAL', insertText: 'NUMTODSINTERVAL(dec,interval_unit)', description: '转换一个指定的 DEC 类型到 INTERVAL DAY TO SECOND' },
    { label: 'NUMTOYMINTERVAL', insertText: 'NUMTOYMINTERVAL (dec,interval_unit)', description: '转换一个指定的 DEC 类型值到 INTERVAL YEAR TO MONTH' },
    { label: 'WEEK', insertText: 'WEEK(date, mode)', description: '根据指定的 mode 计算日期为年中的第几周' },
    {
        label: 'UNIX_TIMESTAMP',
        insertText: 'UNIX_TIMESTAMP (datetime)',
        description: "返回自标准时区的'1970-01-01 00:00:00 +0:00'的到本地会话时区的指定时间的秒数差",
    },
    { label: 'FROM_UNIXTIME', insertText: 'FROM_UNIXTIME(unixtime)', description: "返回将自'1970-01-01 00:00:00'的秒数差转成本地会话时区的时间戳类型" },
    {
        label: 'FROM_UNIXTIME',
        insertText: 'FROM_UNIXTIME(unixtime, fmt)',
        description: "将自'1970-01-01 00:00:00'的秒数差转成本地会话时区的指定 fmt 格式的时间串",
    },
    { label: 'SESSIONTIMEZONE', insertText: 'SESSIONTIMEZONE', description: '返回当前会话的时区' },
    { label: 'DBTIMEZONE', insertText: 'DBTIMEZONE', description: '返回当前数据库的时区' },
    { label: 'DATE_FORMAT', insertText: 'DATE_FORMAT(d, format)', description: '以不同的格式显示日期/时间数据' },
    { label: 'TIME_TO_SEC', insertText: 'TIME_TO_SEC(d)', description: '将时间换算成秒' },
    { label: 'SEC_TO_TIME', insertText: 'SEC_TO_TIME(sec)', description: '将秒换算成时间' },
    { label: 'TO_DAYS', insertText: 'TO_DAYS(timestamp)', description: '转换成公元 0 年 1 月 1 日的天数差' },
    { label: 'DATE_ADD', insertText: 'DATE_ADD(datetime, interval)', description: '返回一个日期或时间值加上一个时间间隔的时间值' },
    { label: 'DATE_SUB', insertText: 'DATE_SUB(datetime, interval)', description: '返回一个日期或时间值减去一个时间间隔的时间值' },
    { label: 'SYS_EXTRACT_UTC', insertText: 'SYS_EXTRACT_UTC(d timestamp)', description: '将所给时区信息转换为 UTC 时区信息' },
    { label: 'TO_DSINTERVAL', insertText: 'TO_DSINTERVAL(d timestamp)', description: '转换一个 timestamp 类型值到 INTERVAL DAY TO SECOND' },
    { label: 'TO_YMINTERVAL', insertText: 'TO_YMINTERVAL(d timestamp)', description: '转换一个 timestamp 类型值到 INTERVAL YEAR TO MONTH' },
    // 空值判断函数
    { label: 'COALESCE', insertText: 'COALESCE(n1,n2,…nx)', description: '返回第一个非空的值' },
    { label: 'IFNULL', insertText: 'IFNULL(n1,n2)', description: '当 n1 为非空时，返回 n1；若 n1 为空，则返回 n2' },
    { label: 'ISNULL', insertText: 'ISNULL(n1,n2)', description: '当 n1 为非空时，返回 n1；若 n1 为空，则返回 n2' },
    { label: 'NULLIF', insertText: 'NULLIF(n1,n2)', description: '如果 n1=n2 返回 NULL，否则返回 n1' },
    { label: 'NVL', insertText: 'NVL(n1,n2)', description: '返回第一个非空的值' },
    { label: 'NULL_EQU', insertText: 'NULL_EQU', description: '返回两个类型相同的值的比较' },
    // 类型转换函数
    { label: 'CAST', insertText: 'CAST(value AS 类型说明)', description: '将 value 转换为指定的类型' },
    {
        label: 'CONVERT',
        insertText: 'CONVERT(类型说明,value)',
        description:
            '用于当 INI 参数 ENABLE_CS_CVT=0 时，将 value 转换为指定的类型；用于当 INI 参数 ENABLE_CS_CVT=1 时，将字符串从原串编码格式转换成目的编码格式',
    },
    {
        label: 'CONVERT',
        insertText: 'CONVERT(char, dest_char_set [,source_char_set ] )',
        description:
            '用于当 INI 参数 ENABLE_CS_CVT=0 时，将 value 转换为指定的类型；用于当 INI 参数 ENABLE_CS_CVT=1 时，将字符串从原串编码格式转换成目的编码格式',
    },
    { label: 'HEXTORAW', insertText: 'HEXTORAW(exp)', description: '将 exp 转换为 BLOB 类型' },
    { label: 'RAWTOHEX', insertText: 'RAWTOHEX(exp)', description: '将 exp 转换为 VARCHAR 类型' },
    { label: 'BINTOCHAR', insertText: 'BINTOCHAR(exp)', description: '将 exp 转换为 CHAR' },
    { label: 'TO_BLOB', insertText: 'TO_BLOB(value)', description: '将 value 转换为 blob' },
    { label: 'UNHEX', insertText: 'UNHEX(exp)', description: '将十六进制的 exp 转换为格式字符串' },
    { label: 'HEX', insertText: 'HEX(exp)', description: '将字符串的 exp 转换为十六进制字符串' },
    // 杂类函数
    { label: 'DECODE', insertText: 'DECODE(exp, search1, result1, … searchn, resultn [,default])', description: '查表译码' },
    { label: 'ISDATE', insertText: 'ISDATE(exp)', description: '判断表达式是否为有效的日期' },
    { label: 'ISNUMERIC', insertText: 'ISNUMERIC(exp)', description: '判断表达式是否为有效的数值' },
    { label: 'DM_HASH', insertText: 'DM_HASH(exp)', description: '根据给定表达式生成 HASH 值' },
    { label: 'LNNVL', insertText: 'LNNVL(condition)', description: '根据表达式计算结果返回布尔值' },
    { label: 'LENGTHB', insertText: 'LENGTHB(value)', description: '返回 value 的字节数' },
    {
        label: 'FIELD',
        insertText: 'FIELD(value, e1, e2, e3, e4...en)',
        description: '返回 value 在列表 e1, e2, e3, e4...en 中的位置序号，不在输入列表时则返回 0',
    },
    { label: 'ORA_HASH', insertText: 'ORA_HASH(exp [,max_bucket [,seed_value]])', description: '为表达式 exp 生成 HASH 桶值' },
];

let dmDialectInfo: DialectInfo;
class DMDialect implements DbDialect {
    getInfo(): DialectInfo {
        if (dmDialectInfo) {
            return dmDialectInfo;
        }

        let { keywords, operators, builtinVariables } = sqlLanguage;
        let functionNames = replaceFunctions.map((a) => a.label);
        let excludeKeywords = new Set(functionNames.concat(operators));

        let editorCompletions: EditorCompletion = {
            keywords: keywords
                .filter((a: string) => !excludeKeywords.has(a)) // 移除已存在的operator、function
                .map((a: string): EditorCompletionItem => ({ label: a, description: 'keyword' }))
                .concat(
                    // 加上自定义的关键字
                    commonCustomKeywords.map(
                        (a): EditorCompletionItem => ({
                            label: a,
                            description: 'keyword',
                        })
                    )
                ),
            operators: operators.map((a: string): EditorCompletionItem => ({ label: a, description: 'operator' })),
            functions: replaceFunctions,
            variables: builtinVariables.map((a: string): EditorCompletionItem => ({ label: a, description: 'var' })),
        };

        dmDialectInfo = {
            name: 'DM',
            icon: 'iconfont icon-db-dm',
            defaultPort: 5236,
            formatSqlDialect: 'plsql',
            columnTypes: DM_TYPE_LIST.sort((a, b) => a.udtName.localeCompare(b.udtName)),
            editorCompletions,
        };
        return dmDialectInfo;
    }

    getDefaultSelectSql(db: string, table: string, condition: string, orderBy: string, pageNum: number, limit: number) {
        return `SELECT * FROM "${table}" ${condition ? 'WHERE ' + condition : ''} ${orderBy ? orderBy : ''} ${this.getPageSql(pageNum, limit)};`;
    }

    getPageSql(pageNum: number, limit: number) {
        return ` OFFSET ${(pageNum - 1) * limit} LIMIT ${limit}`;
    }

    getDefaultRows(): RowDefinition[] {
        return [
            { name: 'id', type: 'BIGINT', length: '', numScale: '', value: '', notNull: true, pri: true, auto_increment: true, remark: '主键ID' },
            { name: 'creator_id', type: 'BIGINT', length: '', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '创建人id' },
            {
                name: 'creator',
                type: 'VARCHAR',
                length: '100',
                numScale: '',
                value: '',
                notNull: true,
                pri: false,
                auto_increment: false,
                remark: '创建人姓名',
            },
            {
                name: 'create_time',
                type: 'TIMESTAMP',
                length: '',
                numScale: '',
                value: 'SYSDATE',
                notNull: true,
                pri: false,
                auto_increment: false,
                remark: '创建时间',
            },
            { name: 'updator_id', type: 'BIGINT', length: '', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '修改人id' },
            {
                name: 'updator',
                type: 'VARCHAR',
                length: '100',
                numScale: '',
                value: '',
                notNull: true,
                pri: false,
                auto_increment: false,
                remark: '修改人姓名',
            },
            {
                name: 'update_time',
                type: 'TIMESTAMP',
                length: '',
                numScale: '',
                value: 'SYSDATE',
                notNull: true,
                pri: false,
                auto_increment: false,
                remark: '修改时间',
            },
        ];
    }

    getDefaultIndex(): IndexDefinition {
        return {
            indexName: '',
            columnNames: [],
            unique: false,
            indexType: 'NORMAL',
            indexComment: '',
        };
    }

    quoteIdentifier = (name: string) => {
        return `"${name}"`;
    };

    matchType(text: string, arr: string[]): boolean {
        if (!text || !arr || arr.length === 0) {
            return false;
        }
        for (let i = 0; i < arr.length; i++) {
            if (text.indexOf(arr[i]) > -1) {
                return true;
            }
        }
        return false;
    }

    getDefaultValueSql(cl: any): string {
        if (cl.value && cl.value.length > 0) {
            // 哪些字段默认值需要加引号
            let marks = false;
            if (this.matchType(cl.type, ['CHAR', 'TIME', 'DATE', 'TEXT'])) {
                // 默认值是now()的time或date不需要加引号
                let val = cl.value.toUpperCase().replace(' ', '');
                if (this.matchType(cl.type, ['TIME', 'DATE']) && ['CURRENT_DATE', 'SYSDATE', 'CURDATE', 'CURTIME'].includes(val)) {
                    marks = false;
                } else {
                    marks = true;
                }
            }
            // 哪些函数不需要加引号
            if (this.matchType(cl.value, ['nextval'])) {
                marks = false;
            }
            return ` DEFAULT ${marks ? "'" : ''}${cl.value}${marks ? "'" : ''}`;
        }
        return '';
    }

    getTypeLengthSql(cl: any) {
        // 哪些字段可以指定长度  VARCHAR/VARCHAR2/CHAR/BIT/NUMBER/NUMERIC/TIME、TIMESTAMP(可以指定小数秒精度)
        if (cl.length && this.matchType(cl.type, ['CHAR', 'BIT', 'TIME', 'NUM', 'DEC'])) {
            // 哪些字段类型可以指定小数点
            if (cl.numScale && this.matchType(cl.type, ['NUM', 'DEC'])) {
                return `(${cl.length}, ${cl.numScale})`;
            } else {
                return `(${cl.length})`;
            }
        }
        return '';
    }

    genColumnBasicSql(cl: RowDefinition): string {
        let length = this.getTypeLengthSql(cl);
        // 默认值
        let defVal = this.getDefaultValueSql(cl);
        let incr = cl.auto_increment ? 'IDENTITY' : '';
        // 如果有原名以原名为准
        let name = cl.oldName && cl.name !== cl.oldName ? cl.oldName : cl.name;
        return ` ${this.quoteIdentifier(name)} ${cl.type}${length} ${incr} ${cl.notNull ? 'NOT NULL' : ''} ${defVal} `;
    }

    getCreateTableSql(data: any): string {
        let createSql = '';
        let tableCommentSql = '';
        let columCommentSql = '';

        // 创建表结构
        let pks = [] as string[];
        let fields: string[] = [];
        data.fields.res.forEach((item: any) => {
            item.name && fields.push(this.genColumnBasicSql(item));
            if (item.pri) {
                pks.push(item.name);
            }
            // 列注释
            if (item.remark) {
                columCommentSql += ` comment on column "${data.tableName}"."${item.name}" is '${item.remark}'; `;
            }
        });
        // 建表
        createSql = `CREATE TABLE "${data.tableName}"
                     (
                         ${fields.join(',')}
                             ${pks ? `, PRIMARY KEY (${pks.join(',')})` : ''}
                     );`;
        // 表注释
        if (data.tableComment) {
            tableCommentSql = ` comment on table "${data.tableName}" is '${data.tableComment}'; `;
        }

        return createSql + tableCommentSql + columCommentSql;
    }

    getCreateIndexSql(tableData: any): string {
        // CREATE UNIQUE INDEX idx_column_name ON your_table (column1, column2);
        // COMMENT ON INDEX idx_column_name IS 'Your index comment here';
        // 创建索引
        let sql: string[] = [];
        tableData.indexs.res.forEach((a: any) => {
            sql.push(` CREATE ${a.unique ? 'UNIQUE' : ''} INDEX ${a.indexName} ON "${tableData.tableName}" ("${a.columnNames.join('","')})"`);
        });
        return sql.join(';');
    }

    getModifyColumnSql(tableData: any, tableName: string, changeData: { del: RowDefinition[]; add: RowDefinition[]; upd: RowDefinition[] }): string {
        let schemaArr = tableData.db.split('/');
        let schema = schemaArr.length > 1 ? schemaArr[schemaArr.length - 1] : schemaArr[0];

        let dbTable = `${this.quoteIdentifier(schema)}.${this.quoteIdentifier(tableName)}`;

        let modifySql = '';
        let dropSql = '';
        let renameSql = '';
        let commentSql = '';

        // 主键字段
        let priArr = new Set();

        if (changeData.add.length > 0) {
            changeData.add.forEach((a) => {
                modifySql += `ALTER TABLE ${dbTable} add COLUMN ${this.genColumnBasicSql(a)};`;
                if (a.remark) {
                    commentSql += `COMMENT ON COLUMN ${dbTable}.${this.quoteIdentifier(a.name)} IS '${a.remark}';`;
                }
                if (a.pri) {
                    priArr.add(`"${a.name}"`);
                }
            });
        }

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                let cmtSql = `COMMENT ON COLUMN ${dbTable}.${this.quoteIdentifier(a.name)} IS '${a.remark}';`;
                if (a.remark && a.oldName === a.name) {
                    commentSql += cmtSql;
                }
                // 修改了字段名
                if (a.oldName !== a.name) {
                    renameSql += `ALTER TABLE ${dbTable} RENAME COLUMN ${this.quoteIdentifier(a.oldName!)} TO ${this.quoteIdentifier(a.name)};`;
                    if (a.remark) {
                        commentSql += cmtSql;
                    }
                }
                modifySql += `ALTER TABLE ${dbTable} MODIFY ${this.genColumnBasicSql(a)};`;
                if (a.pri) {
                    priArr.add(`${this.quoteIdentifier(a.name)}`);
                }
            });
        }

        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                dropSql += `ALTER TABLE ${dbTable} DROP COLUMN ${a.name};`;
            });
        }

        // 编辑主键
        let dropPkSql = '';
        if (priArr.size > 0) {
            let resPri = tableData.fields.res.filter((a: RowDefinition) => a.pri);
            if (resPri) {
                priArr.add(`${this.quoteIdentifier(resPri.name)}`);
            }
            // 如果有编辑主键字段，则删除主键，再添加主键
            // 解析表字段中是否含有主键，有的话就删除主键
            if (tableData.fields.oldFields.find((a: RowDefinition) => a.pri)) {
                dropPkSql = `ALTER TABLE ${dbTable} DROP PRIMARY KEY;`;
            }
        }

        let addPkSql = priArr.size > 0 ? `ALTER TABLE ${dbTable} ADD PRIMARY KEY (${Array.from(priArr).join(',')});` : '';

        return dropPkSql + modifySql + dropSql + renameSql + addPkSql + commentSql;
    }

    getModifyIndexSql(tableData: any, tableName: string, changeData: { del: any[]; add: any[]; upd: any[] }): string {
        // 不能直接修改索引名或字段、需要先删后加
        let dropIndexNames: string[] = [];
        let addIndexs: any[] = [];

        if (changeData.upd.length > 0) {
            changeData.upd.forEach((a) => {
                dropIndexNames.push(a.indexName);
                addIndexs.push(a);
            });
        }

        if (changeData.del.length > 0) {
            changeData.del.forEach((a) => {
                dropIndexNames.push(a.indexName);
            });
        }

        if (changeData.add.length > 0) {
            changeData.add.forEach((a) => {
                addIndexs.push(a);
            });
        }

        if (dropIndexNames.length > 0 || addIndexs.length > 0) {
            let sql: string[] = [];
            if (dropIndexNames.length > 0) {
                dropIndexNames.forEach((a) => {
                    sql.push(`DROP INDEX ${a}`);
                });
            }

            if (addIndexs.length > 0) {
                addIndexs.forEach((a) => {
                    sql.push(`CREATE ${a.unique ? 'UNIQUE' : ''} INDEX ${a.indexName} ON "${tableName}" (${a.columnNames.join(',')})`);
                });
            }
            return sql.join(';');
        }
        return '';
    }

    getDataType(columnType: string): DataType {
        if (DbInst.isNumber(columnType)) {
            return DataType.Number;
        }
        // 日期时间类型
        if (/datetime|timestamp/gi.test(columnType)) {
            return DataType.DateTime;
        }
        // 日期类型
        if (/date/gi.test(columnType)) {
            return DataType.Date;
        }
        // 时间类型
        if (/time/gi.test(columnType)) {
            return DataType.Time;
        }
        return DataType.String;
    }

    // eslint-disable-next-line @typescript-eslint/no-unused-vars,no-unused-vars
    wrapStrValue(columnType: string, value: string): string {
        return `'${value}'`;
    }
}
