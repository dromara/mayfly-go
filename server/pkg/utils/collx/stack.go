package collx

type Stack[T any] struct {
	items []T
}

// 入栈
func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

// 出栈
func (s *Stack[T]) Pop() T {
	var item T
	if len(s.items) == 0 {
		return item
	}
	lastIndex := len(s.items) - 1
	item = s.items[lastIndex]
	s.items = s.items[:lastIndex]
	return item
}

// 获取栈顶元素
func (s *Stack[T]) Top() T {
	var item T
	if len(s.items) == 0 {
		return item
	}
	return s.items[len(s.items)-1]
}

// 检查栈是否为空
func (s *Stack[T]) IsEmpty() bool {
	return len(s.items) == 0
}

// 返回栈的大小
func (s *Stack[T]) Size() int {
	return len(s.items)
}
