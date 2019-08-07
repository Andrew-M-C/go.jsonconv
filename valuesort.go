package jsonconv
import (
	"sort"
	"strings"
)

type valuePair struct {
	K string
	V *JsonValue
}

type ascending []*valuePair
func (this ascending) Len() int {
	return len(this)
}
func (this ascending) Less(i, j int) bool {
	return strings.Compare(this[i].K, this[j].K) < 0
}
func (this ascending) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
	return
}

type descending []*valuePair
func (this descending) Len() int {
	return len(this)
}
func (this descending) Less(i, j int) bool {
	return strings.Compare(this[i].K, this[j].K) > 0
}
func (this descending) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
	return
}

func sortObjects(obj *JsonValue, mode Sort) []*valuePair {
	ret := make([]*valuePair, 0, obj.Length())
	obj.ObjectForeach(func (k string, v *JsonValue) error {
		ret = append(ret, &valuePair{K: k, V: v})
		return nil
	})
	switch mode {
	case DictAsc:
		sort.Sort(ascending(ret))
	case DictDesc:
		sort.Sort(descending(ret))
	default:
		// do nothing
	}
	return ret
}
