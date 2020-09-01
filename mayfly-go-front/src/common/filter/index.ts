/*
 * @Date: 2020-05-23 09:55:10
 * @LastEditors: JOU(wx: huzhen555)
 * @LastEditTime: 2020-05-27 15:34:15
 */ 
import { time2Date } from '@/common/util';


/**
 * @description: 格式化时间过滤器
 * @author: JOU(wx: huzhen555)
 * @param {any} value 过滤器参数
 * @return: 转换后的参数
 */
function timeStr2Date(value: string) {
  return time2Date(value);
}

/**
 * @description: 以一个分隔符替换为另一个分隔符，常用于数组字符串转换为某格式
 * @author: JOU(wx: huzhen555)
 * @param {any} value 过滤器参数
 * @return: 转换后的参数
 */
function replaceTag(value: string, newSep = '', oldSep = ',') {
  return value.replace(new RegExp(oldSep, 'g'), () => newSep);
}

/**
 * @description: 字符串转数组
 * @author: JOU(wx: huzhen555)
 * @param {string} value 待转换字符串
 * @return: 转换后的数组
 */
function str2Ary(value: string, sep = ',') {
  return (value || '').split(sep);
}

/**
 * @description: 按shopName(subName)格式化店名
 * @author: JOU(wx: huzhen555)
 * @param {string} value 待转化字符串
 * @return: 格式化后的店名
 */
function formatShopName(value: string, subName = '') {
  if (subName) {
    return `${value}(${subName})`;
  }
  return value;
}


export const vueFilters = {
  timeStr2Date, replaceTag, formatShopName,
};

// /**
//  * @description: 返回数据的格式化，如有些数据需要以逗号隔开转换成数组等
//  * @author: JOU(wx: huzhen555)
//  * @param {any}  data 格式化的数据
//  * @param {any} rules 转换规则，可对传入object，string，function
//  *                    object时data必需为array，格式为 { key1: ['filterName', 'arg1', 'arg2'], key2: function <= [自定义过滤器] }
//  *                    string时，表示某个过滤器的方法名
//  *                    function时，表示某个自定义过滤器
//  * @return: 转换后的数据
//  */
// const filterHandlers = { ...vueFilters, str2Ary };
// type TCustomerFilter = (...args: any[]) => any;
// type TRuleMap = IGeneralObject<[string, ...any[]]|TCustomerFilter>
// export function formatResp(data: any, rules: TRuleMap|TCustomerFilter|string) {
//   const ruleHandler = (rule: TCustomerFilter|string, dataItem: any, origin: any[]) => {
//     if (typeof rule === 'string' && typeof filterHandlers[rule] === 'function') {
//       dataItem = filterHandlers[rule](dataItem);
//     }
//     else if (Array.isArray(rule) && rule.length > 0 && typeof filterHandlers[rule[0]] === 'function') {
//       dataItem = filterHandlers[rule[0]].apply([dataItem, ...rule.slice(1)]);
//     }
//     else if (typeof rule  === 'function') {
//       dataItem = rule(dataItem, origin);
//     }
//     return dataItem;
//   }
  
  
//   if (Array.isArray(data)) {
//     if (data.length <= 0 || Object.keys(rules).length <= 0) {
//       return data;
//     }

//     return data.map(dataItem => {
//       rules = rules as TRuleMap;
//       for (let ruleKey in rules) {
//         let rule = rules[ruleKey];
//         dataItem[ruleKey] = ruleHandler(rule, dataItem[ruleKey], dataItem);
//       }
//       return dataItem;
//     });
//   }
//   else if (typeof rules === 'string' || typeof rules === 'function') {
//     return ruleHandler(rules, data, data);
//   }
//   else {
//     return data;
//   }
// }