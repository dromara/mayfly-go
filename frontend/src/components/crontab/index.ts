// 表单选项的子组件校验数字格式
export function checkNumber(value: any, minLimit: number, maxLimit: number) {
    //检查必须为整数
    value = Math.floor(value);
    if (value < minLimit) {
        value = minLimit;
    } else if (value > maxLimit) {
        value = maxLimit;
    }
    return value;
}

export interface CrontabValueObj {
    second: string;
    min: string;
    hour: string;
    day: string;
    mouth: string;
    week: string;
    year: string;
}
