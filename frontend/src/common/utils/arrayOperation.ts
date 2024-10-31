// 判断两数组是否相同
export function judementSameArr(news: Array<string>, old: Array<string>) {
    let count = 0;
    const leng = old.length;
    for (let i in old) {
        for (let j in news) {
            if (old[i] === news[j]) count++;
        }
    }
    return count === leng ? true : false;
}
