package utils

import "fmt"

// 数组比较
// 依次返回，新增值，删除值，以及不变值
func ArrayCompare(newArr []interface{}, oldArr []interface{}, compareFun func(interface{}, interface{}) bool) ([]interface{}, []interface{}, []interface{}) {
	var unmodifierValue []interface{}
	ni, oi := 0, 0
	for {
		if ni >= len(newArr) {
			break
		}
		nv := newArr[ni]
		for {
			if oi >= len(oldArr) {
				oi = 0
				break
			}
			ov := oldArr[oi]
			if compareFun(nv, ov) {
				unmodifierValue = append(unmodifierValue, nv)
				// 新数组移除该位置值
				if len(newArr) > ni {
					newArr = append(newArr[:ni], newArr[ni+1:]...)
					ni = ni - 1
				}
				if len(oldArr) > oi {
					oldArr = append(oldArr[:oi], oldArr[oi+1:]...)
					oi = oi - 1
				}
			}
			oi = oi + 1
		}
		ni = ni + 1
	}

	return newArr, oldArr, unmodifierValue
}

type NumT interface {
	~int | ~int32 | ~uint64
}

func NumberArr2StrArr[T NumT](numberArr []T) []string {
	strArr := make([]string, 0)
	for _, v := range numberArr {
		strArr = append(strArr, fmt.Sprintf("%d", v))
	}
	return strArr

}
