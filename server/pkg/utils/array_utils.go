package utils

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
				newArr = append(newArr[:ni], newArr[ni+1:]...)
				oldArr = append(oldArr[:oi], oldArr[oi+1:]...)
				ni = ni - 1
				oi = oi - 1
			}
			oi = oi + 1
		}
		ni = ni + 1
	}

	return newArr, oldArr, unmodifierValue
}
