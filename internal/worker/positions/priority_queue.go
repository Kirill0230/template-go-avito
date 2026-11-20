package positions

import (
	"bytes"
	"container/heap"
)

type priorityQueue []*task

func (q priorityQueue) Len() int { return len(q) }

func (q priorityQueue) Less(i, j int) bool {
	return taskOlder(q[i], q[j])
}

func (q priorityQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *priorityQueue) Push(value any) {
	item := value.(*task)
	item.index = len(*q)
	*q = append(*q, item)
}

func (q *priorityQueue) Pop() any {
	old := *q
	last := len(old) - 1
	item := old[last]
	old[last] = nil
	item.index = -1
	*q = old[:last]
	return item
}

func (q priorityQueue) freshestIndex() int {
	freshest := 0
	for i := 1; i < len(q); i++ {
		if taskOlder(q[freshest], q[i]) {
			freshest = i
		}
	}
	return freshest
}

func taskOlder(left, right *task) bool {
	if !left.trip.SilentSince.Equal(right.trip.SilentSince) {
		return left.trip.SilentSince.Before(right.trip.SilentSince)
	}
	return bytes.Compare(left.trip.ID[:], right.trip.ID[:]) < 0
}

func (q *priorityQueue) remove(index int) *task {
	return heap.Remove(q, index).(*task)
}
