package insert

import (
	"sort"
	"container/list"
)

type IntList struct {
	list []int
}

func (il IntList)Insert(x int) {
  pos := sort.SearchInts(il.list, x)
  il.list = append(il.list, 0)
  for i := len(il.list)-1; i > pos; i-- {
    il.list[i] = il.list[i-1]
  }
  il.list[pos] = x
  return
}

type ContainerList struct {
	list list.List
}

func (cl ContainerList)Insert(x int) {
	for e := cl.list.Front(); e != nil; e = e.Next() {
		if e.Value.(int) > x {
			cl.list.InsertBefore(x, e)
		}
	}
}
