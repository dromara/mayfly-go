// src/basic-languages/mysql/mysql.ts
var language = {
  keywords: [
    "GROUP BY",
    "ORDER BY",
    "LEFT JOIN",
    "RIGHT JOIN",
    "INNER JOIN",
    "SELECT * FROM",
  ],
  operators: [
  ],
  builtinFunctions: [
  ],
  builtinVariables: [],
  replaceFunctions:[ // 自定义修改函数提示
      
    /**  字符串相关函数  */
    { label: 'CONCAT',     insertText:'CONCAT(str1,str2,...)',         description: '多字符串合并' },
    { label: 'ASCII',      insertText:'ASCII(char)',                   description: '返回字符的ASCII值' },
    { label: 'BIT_LENGTH', insertText:'BIT_LENGTH(str1)',              description: '多字符串合并' },
    { label: 'INSTR',      insertText:'INSTR(str,substr)',             description: '返回字符串substr所在str位置' },
    { label: 'LEFT',       insertText:'LEFT(str,len)',                 description: '返回字符串str的左端len个字符' },
    { label: 'RIGHT',      insertText:'RIGHT(str,len)',                description: '返回字符串str的右端len个字符' },
    { label: 'MID',        insertText:'MID(str,pos,len)',              description: '返回字符串str的位置pos起len个字符' },
    { label: 'SUBSTRING',  insertText:'SUBSTRING(exp, start, length)', description: '截取字符串' },
    { label: 'REPLACE',    insertText:'REPLACE(str,from_str,to_str)',  description: '替换字符串' },
    { label: 'REPEAT',     insertText:'REPEAT(str,count)',             description: '重复字符串count遍' },
    { label: 'UPPER',      insertText:'UPPER(str)',                    description: '返回大写的字符串' },
    { label: 'LOWER',      insertText:'LOWER(str)',                    description: '返回小写的字符串' },
    { label: 'TRIM',       insertText:'TRIM(str)',                     description: '去除字符串首尾空格' },
    /**  数学相关函数  */
    { label: 'ABS',                insertText:'ABS(n)',         description: '返回n的绝对值' },
    { label: 'FLOOR',              insertText:'FLOOR(n)',       description: '返回不大于n的最大整数' },
    { label: 'CEILING',            insertText:'CEILING(n)',     description: '返回不小于n的最小整数值' },
    { label: 'ROUND',              insertText:'ROUND(n,d)',     description: '返回n的四舍五入值,保留d(默认0)位小数' },
    { label: 'RAND',               insertText:'RAND()',         description: '返回在范围0到1.0内的随机浮点值' },
    
    /** 日期函数 */  
    { label: 'DATE',              insertText:'DATE(\'date\')',                      description: '返回指定表达式的日期部分' },
    { label: 'WEEK',              insertText:'WEEK(\'date\')',                      description: '返回指定日期是一年中的第几周' },
    { label: 'MONTH',             insertText:'MONTH(\'date\')',                     description: '返回指定日期的月份' },
    { label: 'QUARTER',           insertText:'QUARTER(\'date\')',                   description: '返回指定日期是一年的第几个季度' },
    { label: 'YEAR',              insertText:'YEAR(\'date\')',                      description: '返回指定日期的年份' },
    { label: 'DATE_ADD',          insertText:'DATE_ADD(\'date\', interval 1 day)',  description: '日期函数加减运算' },
    { label: 'DATE_SUB',          insertText:'DATE_SUB(\'date\', interval 1 day)',  description: '日期函数加减运算' },
    { label: 'DATE_FORMAT',       insertText:'DATE_FORMAT(\'date\', \'%Y-%m-%d %h:%i:%s\')',       description: '' }, 
    { label: 'CURDATE',           insertText:'CURDATE()',                      description: '返回当前日期' },
    { label: 'CURTIME',           insertText:'CURTIME()',                      description: '返回当前时间' },
    { label: 'NOW',               insertText:'NOW()',                          description: '返回当前日期时间' },
    { label: 'DATEDIFF',          insertText:'DATEDIFF(expr1,expr2)',          description: '返回结束日expr1和起始日expr2之间的天数' },
    { label: 'UNIX_TIMESTAMP',    insertText:'UNIX_TIMESTAMP()',               description: '返回指定时间(默认当前)unix时间戳' },
    { label: 'FROM_UNIXTIME',     insertText:'FROM_UNIXTIME(timestamp)',       description: '把时间戳格式为年月日时分秒' },
      
    /**  逻辑函数 */  
    { label: 'IFNULL',            insertText:'IFNULL(expression, alt_value)',  description: '表达式为空取第二个参数值,否则取表达式值' },
    { label: 'IF',                insertText:'IF(expr1, expr2, expr3)',          description: 'expr1为true则取expr2，否则取expr3' },
    { label: 'CASE',              insertText:'(CASE \n WHEN expr1 THEN expr2 \n ELSE expr3) col',       description: 'CASE WHEN THEN ELSE' },
  ]
};
export {
  language
};
