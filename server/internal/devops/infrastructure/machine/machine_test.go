package machine

import (
	"fmt"
	"mayfly-go/pkg/utils"
	"strings"
	"testing"
)

func TestSSH(t *testing.T) {
	//ssh.ListenAndServe("148.70.36.197")
	//cli := New("148.70.36.197", "root", "g..91mn#", 22)
	////output, err := cli.Run("free -h")
	////fmt.Printf("%v\n%v", output, err)
	//err := cli.RunTerminal("tail -f /usr/local/java/logs/eatlife-info.log", os.Stdout, os.Stdin)
	//fmt.Println(err)

	res := "top - 17:14:07 up 5 days,  6:30,  2 users,  load average: 0.03, 0.04, 0.05\nTasks: 101 total,   1 running, 100 sleeping,   0 stopped,   0 zombie\n%Cpu(s):  6.2 us,  0.0 sy,  0.0 ni, 93.8 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st\nKiB Mem :  1882012 total,    73892 free,   770360 used,  1037760 buff/cache\nKiB Swap:        0 total,        0 free,        0 used.   933492 avail Mem"
	split := strings.Split(res, "\n")
	//var firstLine string
	//for i := 0; i < len(split); i++ {
	//	if i == 0 {
	//		val := strings.Split(split[i], "top -")[1]
	//		vals := strings.Split(val, ",")
	//
	//	}
	//}
	firstLine := strings.Split(strings.Split(split[0], "top -")[1], ",")
	//  17:14:07 up 5 days
	up := strings.Trim(strings.Split(firstLine[0], "up")[1], " ") + firstLine[1]
	//   2 users
	users := strings.Split(strings.Trim(firstLine[2], " "), " ")[0]
	//   load average: 0.03
	oneMinLa := strings.Trim(strings.Split(strings.Trim(firstLine[3], " "), ":")[1], " ")
	fiveMinLa := strings.Trim(firstLine[4], " ")
	fietMinLa := strings.Trim(firstLine[5], " ")
	fmt.Println(firstLine, up, users, oneMinLa, fiveMinLa, fietMinLa)
	tasks := Parse(strings.Split(split[1], "Tasks:")[1])
	cpu := Parse(strings.Split(split[2], "%Cpu(s):")[1])
	mem := Parse(strings.Split(split[3], "KiB Mem :")[1])
	fmt.Println(tasks, cpu, mem)
}

func Parse(val string) map[string]string {
	res := make(map[string]string)
	vals := strings.Split(val, ",")
	for i := 0; i < len(vals); i++ {
		trimData := strings.Trim(vals[i], " ")
		keyValue := strings.Split(trimData, " ")
		res[keyValue[1]] = keyValue[0]
	}
	return res
}

func TestTemplateRev(t *testing.T) {
	temp := "hello my name is {name} hahahaha lihaiba {age} years old {public}"
	str := "hello my name is   hmlhmlhm  慌慌信息    hahahaha lihaiba   15   years old private  protected"

	//temp1 := " top - {up},  {users} users,  load average: {loadavg}"
	//str1 := " top - 17:14:07 up 5 days,  6:30,  2 users,  load average: 0.03, 0.04, 0.05"

	//taskTemp := "Tasks: {total} total,   {running} running, {sleeping} sleeping,   {stopped} stopped,   {zombie} zombie"
	//taskVal := "Tasks:   101  total,   1 running, 100   sleeping,    0   stopped,   0  zombie"

	//nameRunne := []rune(str)
	//index := strings.Index(temp, "{")
	//ei := strings.Index(temp, "}") + 1
	//next := temp[ei:]
	//key := temp[index+1 : ei-1]
	//value := SubString(str, index, UnicodeIndex(str, next))
	res := make(map[string]interface{})
	utils.ReverStrTemplate(temp, str, res)
	fmt.Println(res)
}

//func ReverStrTemplate(temp, str string, res map[string]string) {
//	index := UnicodeIndex(temp, "{")
//	ei := UnicodeIndex(temp, "}") + 1
//	next := temp[ei:]
//	nextContain := UnicodeIndex(next, "{")
//	nextIndexValue := next
//	if nextContain != -1 {
//		nextIndexValue = SubString(next, 0, nextContain)
//	}
//	key := temp[index+1 : ei-1]
//	// 如果后面没有内容了，则取字符串的长度即可
//	var valueLastIndex int
//	if nextIndexValue == "" {
//		valueLastIndex = StrLen(str)
//	} else {
//		valueLastIndex = UnicodeIndex(str, nextIndexValue)
//	}
//	value := SubString(str, index, valueLastIndex)
//	res[key] = value
//
//	if nextContain != -1 {
//		ReverStrTemplate(next, SubString(str, UnicodeIndex(str, value)+StrLen(value), StrLen(str)), res)
//	}
//}
//
//func StrLen(str string) int {
//	return len([]rune(str))
//}
//
//func SubString(str string, begin, end int) (substr string) {
//	// 将字符串的转换成[]rune
//	rs := []rune(str)
//	lth := len(rs)
//
//	// 简单的越界判断
//	if begin < 0 {
//		begin = 0
//	}
//	if begin >= lth {
//		begin = lth
//	}
//	if end > lth {
//		end = lth
//	}
//
//	// 返回子串
//	return string(rs[begin:end])
//}
//
//func UnicodeIndex(str, substr string) int {
//	// 子串在字符串的字节位置
//	result := strings.Index(str, substr)
//	if result >= 0 {
//		// 获得子串之前的字符串并转换成[]byte
//		prefix := []byte(str)[0:result]
//		// 将子串之前的字符串转换成[]rune
//		rs := []rune(string(prefix))
//		// 获得子串之前的字符串的长度，便是子串在字符串的字符位置
//		result = len(rs)
//	}
//
//	return result
//}

func TestTerminal(t *testing.T) {

	// ioutil.ReadAll(file)
}
