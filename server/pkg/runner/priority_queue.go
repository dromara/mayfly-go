package runner

// PriorityQueue 是一个基于小顶堆的优先队列
// 当capacity <= 0时，为无界队列，切片容量会动态扩缩容
// 当capacity > 0 时，为有界队列，初始化后就固定容量，不会扩缩容
type PriorityQueue[T any] struct {
	// 用于比较前一个元素是否小于后一个元素
	less Less[T]
	// 队列容量
	capacity int
	// 队列中的元素，为便于计算父子节点的index，0位置留空，根节点从1开始
	data []T

	zero T
}

func (p *PriorityQueue[T]) Len() int {
	return len(p.data) - 1
}

// Cap 无界队列返回0，有界队列返回创建队列时设置的值
func (p *PriorityQueue[T]) Cap() int {
	return p.capacity
}

func (p *PriorityQueue[T]) IsBoundless() bool {
	return p.capacity <= 0
}

func (p *PriorityQueue[T]) IsFull() bool {
	return p.capacity > 0 && len(p.data)-1 == p.capacity
}

func (p *PriorityQueue[T]) IsEmpty() bool {
	return len(p.data) < 2
}

func (p *PriorityQueue[T]) Peek(i int) (T, bool) {
	if p.IsEmpty() {
		return p.zero, false
	}
	if i >= p.Len() {
		return p.zero, false
	}
	return p.data[i+1], true
}

func (p *PriorityQueue[T]) Enqueue(t T) bool {
	if p.IsFull() {
		return false
	}

	p.data = append(p.data, t)
	node, parent := len(p.data)-1, (len(p.data)-1)/2
	for parent > 0 && p.less(p.data[node], p.data[parent]) {
		p.data[parent], p.data[node] = p.data[node], p.data[parent]
		node = parent
		parent = parent / 2
	}
	return true
}

func (p *PriorityQueue[T]) Dequeue() (T, bool) {
	if p.IsEmpty() {
		return p.zero, false
	}

	pop := p.data[1]
	// 假定说我拿到了堆顶，就是理论上优先级最低的
	// pop 的优先级
	p.data[1] = p.data[len(p.data)-1]
	p.data = p.data[:len(p.data)-1]
	p.shrinkIfNecessary()
	p.heapify(p.data, len(p.data)-1, 1)
	return pop, true
}

func (p *PriorityQueue[T]) shrinkIfNecessary() {
	if !p.IsBoundless() {
		return
	}
	if cap(p.data) > 1024 && len(p.data)*3 < cap(p.data)*2 {
		data := make([]T, len(p.data), cap(p.data)*5/6)
		copy(data, p.data)
		p.data = data
	}
}

func (p *PriorityQueue[T]) heapify(data []T, n, i int) {
	minPos := i
	for {
		if left := i * 2; left <= n && p.less(data[left], data[minPos]) {
			minPos = left
		}
		if right := i*2 + 1; right <= n && p.less(data[right], data[minPos]) {
			minPos = right
		}
		if minPos == i {
			break
		}
		data[i], data[minPos] = data[minPos], data[i]
		i = minPos
	}
}

func (p *PriorityQueue[T]) Remove(i int) (T, bool) {
	if p.IsEmpty() || i >= p.Len() || i < 0 {
		return p.zero, false
	}

	i += 1
	result := p.data[i]
	last := len(p.data) - 1
	p.data[i] = p.data[last]
	p.data = p.data[:last]
	p.shrinkIfNecessary()
	p.heapify(p.data, len(p.data)-1, i)
	return result, true
}

// NewPriorityQueue 创建优先队列 capacity <= 0 时，为无界队列，否则有有界队列
func NewPriorityQueue[T any](capacity int, less Less[T]) *PriorityQueue[T] {
	sliceCap := capacity + 1
	if capacity < 1 {
		capacity = 0
		sliceCap = 64
	}
	return &PriorityQueue[T]{
		capacity: capacity,
		data:     make([]T, 1, sliceCap),
		less:     less,
	}
}

// Less 用于比较两个对象的大小 src < dst, 返回 true，src >= dst, 返回 false
type Less[T any] func(src T, dst T) bool
