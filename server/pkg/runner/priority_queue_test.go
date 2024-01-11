package runner

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestChangePriority(t *testing.T) {
	q := NewPriorityQueue[*priorityElement](100,
		func(src *priorityElement, dst *priorityElement) bool {
			return src.Priority < dst.Priority
		})
	e1 := &priorityElement{
		Data:     10,
		Priority: 200,
	}
	_ = q.Enqueue(e1)
	e2 := &priorityElement{
		Data:     10,
		Priority: 100,
	}
	_ = q.Enqueue(e2)
	//e1.Priority = 10
	val, _ := q.Dequeue()
	println(val)
}

type priorityElement struct {
	Data     any
	Priority int
}

func TestPriorityQueue_Remove(t *testing.T) {
	q := NewPriorityQueue[*priorityElement](100,
		func(src *priorityElement, dst *priorityElement) bool {
			return src.Priority < dst.Priority
		})

	for i := 8; i > 0; i-- {
		q.Enqueue(&priorityElement{Priority: i})
	}
	requirePriorities(t, q)

	q.Remove(8)
	requirePriorities(t, q)
	q.Remove(7)
	requirePriorities(t, q)

	q.Remove(2)
	requirePriorities(t, q)

	q.Remove(1)
	requirePriorities(t, q)

	q.Remove(0)
	requirePriorities(t, q)
}

func requirePriorities(t *testing.T, q *PriorityQueue[*priorityElement]) {
	ps := make([]int, 0, q.Len())
	for _, val := range q.data[1:] {
		ps = append(ps, val.Priority)
	}
	for i := q.Len(); i >= 2; i-- {
		require.False(t, q.less(q.data[i], q.data[i/2]), ps)
	}
}
