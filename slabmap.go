package slabmap

type SlabMap struct {
	entries       []*entry
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

type entry struct {
	tag           entryTag
	value         interface{}
	vacantBodyLen int
	nextVacantIdx int
}

func (e *entry) isOccupied() bool {
	return e.tag == entryTagOccupied
}

func occupied(value interface{}) *entry {
	return &entry{
		tag:   entryTagOccupied,
		value: value,
	}
}

func vacantHead(vacantBodyLen int) *entry {
	return &entry{
		tag:           entryTagVacantHead,
		vacantBodyLen: vacantBodyLen,
	}
}

func vacantTail(nextVacantIdx int) *entry {
	return &entry{
		tag:           entryTagVacantTail,
		nextVacantIdx: nextVacantIdx,
	}
}

func NewSlabMap() *SlabMap {
	return NewSlabMapWithCapacity(0)
}

func NewSlabMapWithCapacity(capacity int) *SlabMap {
	return &SlabMap{
		entries:       make([]*entry, 0, capacity),
		nextVacantIdx: invalidIndex,
		len:           0,
		nonOptimized:  0,
	}
}

func (m *SlabMap) Capacity() int {
	return cap(m.entries)
}

func (m *SlabMap) Reserve(additional int) {
	entries := make([]*entry, 0, len(m.entries)+m.entriesAdditional(additional))
	for _, entry := range m.entries {
		entries = append(entries, entry)
	}
	m.entries = entries
}

func (m *SlabMap) Len() int {
	return m.len
}

func (m *SlabMap) Get(key int) (value interface{}, exists bool) {
	entry := m.entries[key]
	if entry.isOccupied() {
		return entry.value, true
	}
	return nil, false
}

func (m *SlabMap) Contains(key int) bool {
	_, exists := m.Get(key)
	return exists
}

func (m *SlabMap) Insert(value interface{}) int {
	return m.InsertWithKey(func(int) interface{} { return value })
}

func (m *SlabMap) InsertWithKey(f func(int) interface{}) int {
	var idx int
	if m.nextVacantIdx < len(m.entries) {
		idx = m.nextVacantIdx
		current := m.entries[idx]
		switch current.tag {
		case entryTagVacantHead:
			if current.vacantBodyLen > 0 {
				m.entries[idx+1] = vacantHead(current.vacantBodyLen - 1)
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

func (m *SlabMap) Remove(key int) (value interface{}, removed bool) {
	isLast := (key+1 == len(m.entries))
	current := m.entries[key]
	if !current.isOccupied() {
		return nil, false
	}
	m.len--
	if isLast {
		m.entries = m.entries[0:m.len]
	} else {
		m.entries[key] = vacantTail(m.nextVacantIdx)
		m.nextVacantIdx = key
		m.nonOptimized++
	}
	if m.Len() == 0 {
		m.Clear()
	}
	return current.value, true
}

func (m *SlabMap) Clear() {
	for i := 0; i < len(m.entries); i++ {
		m.entries[i] = nil
	}
	m.len = 0
	m.nextVacantIdx = invalidIndex
	m.nonOptimized = 0
}

func (m *SlabMap) Retain(f func(int, interface{}) bool) {
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
			}
		default:
			// unreachable
		}
	}
	m.entries = m.entries[0:idxVacantStart]
	m.nonOptimized = 0
}

func (m *SlabMap) Optimize() {
	if !m.isOptimized() {
		m.Retain(func(int, interface{}) bool { return true })
	}
}

func (m *SlabMap) Range(f func(int, interface{}) bool) {
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

func (m *SlabMap) entriesAdditional(additional int) int {
	return saturatingSub(additional, len(m.entries)-m.len, 0)
}

func (m *SlabMap) isOptimized() bool {
	return m.nonOptimized == 0
}

func (m *SlabMap) mergeVacant(start, end int) {
	if start < end {
		if start < end-1 {
			m.entries[start] = vacantHead(end - start - 2)
		}
		m.entries[end-1] = vacantTail(m.nextVacantIdx)
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
