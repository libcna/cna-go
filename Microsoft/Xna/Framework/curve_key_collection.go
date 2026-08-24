package framework

import "fmt"

// CurveKeyCollection is a managed, position-sorted collection of CurveKey
// references.
type CurveKeyCollection struct {
	keys             []*CurveKey
	timeRange        float32
	inverseTimeRange float32
	cacheAvailable   bool
	version          uint64
}

func NewCurveKeyCollection() *CurveKeyCollection {
	return &CurveKeyCollection{cacheAvailable: true}
}

func (c *CurveKeyCollection) Item(index int32) (*CurveKey, error) {
	resolved, err := curveCollectionIndex(index, len(c.keys))
	if err != nil {
		return nil, err
	}
	return c.keys[resolved], nil
}

func (c *CurveKeyCollection) SetItem(index int32, value *CurveKey) error {
	if value == nil {
		return curveArgumentError("value is nil")
	}
	resolved, err := curveCollectionIndex(index, len(c.keys))
	if err != nil {
		return err
	}
	if c.keys[resolved].position == value.position {
		c.keys[resolved] = value
		c.version++
		return nil
	}
	c.removeAt(resolved)
	c.add(value)
	return nil
}

func (c *CurveKeyCollection) Count() int32 { return int32(len(c.keys)) }

func (c *CurveKeyCollection) IsReadOnly() bool { return false }

func (c *CurveKeyCollection) IndexOf(item *CurveKey) int32 {
	for index, key := range c.keys {
		if key.EqualsByCurveKey(item) {
			return int32(index)
		}
	}
	return -1
}

func (c *CurveKeyCollection) RemoveAt(index int32) error {
	resolved, err := curveCollectionIndex(index, len(c.keys))
	if err != nil {
		return err
	}
	c.removeAt(resolved)
	return nil
}

func (c *CurveKeyCollection) Add(item *CurveKey) error {
	if item == nil {
		return curveArgumentError("item is nil")
	}
	c.add(item)
	return nil
}

func (c *CurveKeyCollection) Clear() {
	clear(c.keys)
	c.keys = c.keys[:0]
	c.timeRange = 0
	c.inverseTimeRange = 0
	c.cacheAvailable = false
	c.version++
}

func (c *CurveKeyCollection) Contains(item *CurveKey) bool {
	return c.IndexOf(item) >= 0
}

func (c *CurveKeyCollection) CopyTo(destination []*CurveKey, arrayIndex int32) error {
	if destination == nil {
		return curveArgumentError("destination is nil")
	}
	if arrayIndex < 0 || int64(arrayIndex) > int64(len(destination)) {
		return curveArgumentError("array index is outside the destination")
	}
	if int64(len(destination))-int64(arrayIndex) < int64(len(c.keys)) {
		return curveArgumentError("destination is too small")
	}
	copy(destination[int(arrayIndex):], c.keys)
	c.cacheAvailable = false
	return nil
}

func (c *CurveKeyCollection) Remove(item *CurveKey) bool {
	index := c.IndexOf(item)
	c.cacheAvailable = false
	if index < 0 {
		return false
	}
	c.removeAt(int(index))
	return true
}

func (c *CurveKeyCollection) GetEnumerator() Iterator[*CurveKey] {
	return &curveKeyIterator{collection: c, version: c.version}
}

func (c *CurveKeyCollection) Clone() *CurveKeyCollection {
	return &CurveKeyCollection{
		keys:             append([]*CurveKey(nil), c.keys...),
		timeRange:        c.timeRange,
		inverseTimeRange: c.inverseTimeRange,
		cacheAvailable:   true,
	}
}

func (c *CurveKeyCollection) add(item *CurveKey) {
	index := c.binarySearch(item)
	if index >= 0 {
		for index < len(c.keys) && item.position == c.keys[index].position {
			index++
		}
	} else {
		index = ^index
	}
	c.keys = append(c.keys, nil)
	copy(c.keys[index+1:], c.keys[index:])
	c.keys[index] = item
	c.cacheAvailable = false
	c.version++
}

func (c *CurveKeyCollection) binarySearch(item *CurveKey) int {
	low, high := 0, len(c.keys)-1
	for low <= high {
		middle := low + (high-low)/2
		comparison := compareSingle(c.keys[middle].position, item.position)
		switch {
		case comparison == 0:
			return middle
		case comparison < 0:
			low = middle + 1
		default:
			high = middle - 1
		}
	}
	return ^low
}

func (c *CurveKeyCollection) removeAt(index int) {
	copy(c.keys[index:], c.keys[index+1:])
	c.keys[len(c.keys)-1] = nil
	c.keys = c.keys[:len(c.keys)-1]
	c.cacheAvailable = false
	c.version++
}

func (c *CurveKeyCollection) ranges() (float32, float32) {
	c.ensureCache()
	return c.timeRange, c.inverseTimeRange
}

func (c *CurveKeyCollection) ensureCache() {
	if c.cacheAvailable {
		return
	}
	c.timeRange = 0
	c.inverseTimeRange = 0
	if len(c.keys) > 1 {
		c.timeRange = c.keys[len(c.keys)-1].position - c.keys[0].position
		if c.timeRange > smallestPositiveSingle {
			c.inverseTimeRange = 1 / c.timeRange
		}
	}
	c.cacheAvailable = true
}

type curveKeyIterator struct {
	collection *CurveKeyCollection
	version    uint64
	index      int
}

func (i *curveKeyIterator) Next() (*CurveKey, bool, error) {
	if i.version != i.collection.version {
		return nil, false, fmt.Errorf("curve-key collection changed during enumeration")
	}
	if i.index >= len(i.collection.keys) {
		return nil, false, nil
	}
	value := i.collection.keys[i.index]
	i.index++
	return value, true, nil
}

func curveCollectionIndex(index int32, count int) (int, error) {
	if index < 0 || int64(index) >= int64(count) {
		return 0, curveArgumentError("index is outside the collection")
	}
	return int(index), nil
}
