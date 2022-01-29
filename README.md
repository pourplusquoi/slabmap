# slabmap

Ported from Rust library [slabmap](https://github.com/frozenlib/slabmap)

## Examples

```golang
import "github.com/pourplusquoi/slabmap"

slab := slabmap.NewSlabMap()
keyA := slab.Insert("aaa")
keyB := slab.Insert("bbb")

valueA, existsA := slab.Get(keyA)
fmt.Println(valueA, existsA)
valueB, existsB := slab.Get(keyB)
fmt.Println(valueB, existsB)

valueA, removedA := slab.Remove(keyA)
fmt.Println(valueA, removedA)
valueA, removedA = slab.Remove(keyA)
fmt.Println(valueA, removedA)
```

