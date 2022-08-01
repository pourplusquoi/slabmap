package slabmap_test

import (
	"testing"

	"github.com/pourplusquoi/slabmap"
	"github.com/stretchr/testify/assert"
)

func TestSlabMap_Basics(t *testing.T) {
	slab := slabmap.NewSlabMap[string]()
	assert.Equal(t, 0, slab.Len())
	assert.Equal(t, 0, slab.Capacity())

	slab.Reserve(1)
	assert.Equal(t, 0, slab.Len())
	assert.Equal(t, 1, slab.Capacity())

	key1 := slab.Insert("aaa")
	key2 := slab.Insert("bbb")
	assert.Equal(t, 2, slab.Len())
	assert.Equal(t, 2, slab.Capacity())

	value1, exists1 := slab.Get(key1)
	value2, exists2 := slab.Get(key2)
	value3, exists3 := slab.Get(999)
	assert.Equal(t, true, exists1)
	assert.Equal(t, true, exists2)
	assert.Equal(t, false, exists3)
	assert.Equal(t, "aaa", value1)
	assert.Equal(t, "bbb", value2)
	assert.Equal(t, "", value3)
	assert.Equal(t, true, slab.Contains(key1))
	assert.Equal(t, true, slab.Contains(key2))
	assert.Equal(t, false, slab.Contains(999))

	values := make([]string, 0)
	slab.Range(func(_ int, value string) bool {
		values = append(values, value)
		return true
	})
	assert.Equal(t, 2, len(values))
	assert.Equal(t, "aaa", values[0])
	assert.Equal(t, "bbb", values[1])

	value1, removed1 := slab.Remove(key1)
	value2, removed2 := slab.Remove(key1)
	value3, removed3 := slab.Remove(999)
	assert.Equal(t, 1, slab.Len())
	assert.Equal(t, true, removed1)
	assert.Equal(t, false, removed2)
	assert.Equal(t, false, removed3)
	assert.Equal(t, "aaa", value1)
	assert.Equal(t, "", value2)
	assert.Equal(t, "", value3)
	assert.Equal(t, false, slab.Contains(key1))
	assert.Equal(t, true, slab.Contains(key2))
	assert.Equal(t, false, slab.Contains(999))

	slab.Clear()
	assert.Equal(t, 0, slab.Len())
	assert.Equal(t, 2, slab.Capacity())
}

func TestSlabMap_Compaction(t *testing.T) {
	slab := slabmap.NewSlabMap[int]()

	keys := make([]int, 0)
	for i := 0; i < 100; i++ {
		keys = append(keys, slab.Insert(i))
	}
	slab.Optimize()

	for i := 0; i < 50; i++ {
		value, exists := slab.Remove(keys[i])
		assert.Equal(t, true, exists)
		assert.Equal(t, i, value)
	}
	slab.Reserve(1024)
	slab.Optimize()

	assert.Equal(t, 50, slab.Len())
	assert.Greater(t, slab.Capacity(), 1024)

	count := 0
	slab.Range(func(int, int) bool {
		count++
		return true
	})
	assert.Equal(t, 50, count)

	for i := 0; i < 100; i++ {
		keys = append(keys, slab.Insert(i))
	}
	slab.Optimize()

	for i := 50; i < 100; i++ {
		value, exists := slab.Remove(keys[i])
		assert.Equal(t, true, exists)
		assert.Equal(t, i, value)
	}
	slab.Reserve(2048)
	slab.Optimize()

	assert.Equal(t, 100, slab.Len())
	assert.Greater(t, slab.Capacity(), 2048)
}
