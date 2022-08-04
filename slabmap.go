package slabmap

type SlabMap[T any] struct {
	entries       []*entry[T]
	nextVacantIdx int
	len           int
	nonOptimized  int
}

type entryTag uint8

const invalidIndex = int(^uint(0) >> 1)

const (
	_ entryTag = iota
	entryTagOccupied
	entryTagVacantHead
	entryTagVacantTail
)

type entry[T any] struct {
	tag           entryTag
	value         T
	vacantBodyLen int
	nextVacantIdx int
}

func (e *entry[T]) isOccupied() bool {
	return e.tag == entryTagOccupied
}

func occupied[T any](value T) *entry[T] {
	return &entry[T]{
		tag:   entryTagOccupied,
		value: value,
	}
}

func vacantHead[T any](vacantBodyLen int) *entry[T] {
	return &entry[T]{
		tag:           entryTagVacantHead,
		vacantBodyLen: vacantBodyLen,
	}
}

func vacantTail[T any](nextVacantIdx int) *entry[T] {
	return &entry[T]{
		tag:           entryTagVacantTail,
		nextVacantIdx: nextVacantIdx,
	}
}

func NewSlabMap[T any]() *SlabMap[T] {
	return NewSlabMapWithCapacity[T](0)
}

func NewSlabMapWithCapacity[T any](capacity int) *SlabMap[T] {
	return &SlabMap[T]{
		entries:       make([]*entry[T], 0, capacity),
		nextVacantIdx: invalidIndex,
		len:           0,
		nonOptimized:  0,
	}
}

func (m *SlabMap[T]) Capacity() int {
	return cap(m.entries)
}

func (m *SlabMap[T]) Reserve(additional int) {
	entries := make([]*entry[T], 0, len(m.entries)+m.entriesAdditional(additional))
	for _, entry := range m.entries {
		entries = append(entries, entry)
	}
	m.entries = entries
}

func (m *SlabMap[T]) Len() int {
	return m.len
}

func (m *SlabMap[T]) Get(key int) (value T, exists bool) {
	if key < 0 || key >= len(m.entries) {
		return value, false
	}
	entry := m.entries[key]
	if entry.isOccupied() {
		return entry.value, true
	}
	return value, false
}

func (m *SlabMap[T]) Contains(key int) bool {
	_, exists := m.Get(key)
	return exists
}

func (m *SlabMap[T]) Insert(value T) int {
	return m.InsertWithKey(func(int) T { return value })
}

func (m *SlabMap[T]) InsertWithKey(f func(int) T) int {
	var idx int
	if m.nextVacantIdx < len(m.entries) {
		idx = m.nextVacantIdx
		current := m.entries[idx]
		switch current.tag {
		case entryTagVacantHead:
			if current.vacantBodyLen > 0 {
				m.entries[idx+1] = vacantHead[T](current.vacantBodyLen - 1)
			}
			m.nextVacantIdx = idx + 1
		case entryTagVacantTail:
			m.nextVacantIdx = current.nextVacantIdx
		default:
			// unreachable
		}
		m.entries[idx] = occupied(f(idx))
		m.nonOptimized = saturatingSub(m.nonOptimized, 1, 0)
	} else {
		idx = len(m.entries)
		m.entries = append(m.entries, occupied(f(idx)))
	}
	m.len++
	return idx
}

func (m *SlabMap[T]) Remove(key int) (value T, removed bool) {
	if key < 0 || key >= len(m.entries) {
		return value, false
	}
	isLast := (key+1 == len(m.entries))
	current := m.entries[key]
	if !current.isOccupied() {
		return value, false
	}
	m.len--
	if isLast {
		m.entries = m.entries[0 : len(m.entries)-1]
	} else {
		m.entries[key] = vacantTail[T](m.nextVacantIdx)
		m.nextVacantIdx = key
		m.nonOptimized++
	}
	if m.Len() == 0 {
		m.Clear()
	}
	return current.value, true
}

func (m *SlabMap[T]) Clear() {
	for i := 0; i < len(m.entries); i++ {
		m.entries[i] = nil
	}
	m.len = 0
	m.nextVacantIdx = invalidIndex
	m.nonOptimized = 0
}

func (m *SlabMap[T]) Retain(f func(int, T) bool) {
	idxVacantStart := 0
	m.nextVacantIdx = invalidIndex
	for idx := 0; idx < len(m.entries); {
		current := m.entries[idx]
		switch current.tag {
		case entryTagVacantTail:
			idx++
		case entryTagVacantHead:
			idx += current.vacantBodyLen + 2
		case entryTagOccupied:
			if f(idx, current.value) {
				m.mergeVacant(idxVacantStart, idx)
				idx++
				idxVacantStart = idx
			} else {
				current.tag = entryTagVacantTail
				current.nextVacantIdx = invalidIndex
				idx++
			}
		default:
			// unreachable
		}
	}
	m.entries = m.entries[0:idxVacantStart]
	m.nonOptimized = 0
}

func (m *SlabMap[T]) Optimize() {
	if !m.isOptimized() {
		m.Retain(func(int, T) bool { return true })
	}
}

func (m *SlabMap[T]) Range(f func(int, T) bool) {
	for idx := 0; idx < len(m.entries); idx++ {
		current := m.entries[idx]
		if !current.isOccupied() {
			continue
		}
		if !f(idx, current.value) {
			break
		}
	}
}

func (m *SlabMap[T]) entriesAdditional(additional int) int {
	return saturatingSub(additional, len(m.entries)-m.len, 0)
}

func (m *SlabMap[T]) isOptimized() bool {
	return m.nonOptimized == 0
}

func (m *SlabMap[T]) mergeVacant(start, end int) {
	if start < end {
		if start < end-1 {
			m.entries[start] = vacantHead[T](end - start - 2)
		}
		m.entries[end-1] = vacantTail[T](m.nextVacantIdx)
		m.nextVacantIdx = start
	}
}

func saturatingSub(val int, sub int, low int) int {
	res := val - sub
	if res > low {
		return res
	}
	return low
}
