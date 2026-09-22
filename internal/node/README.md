# internal/node — shared read-only structural access layer

`internal/node` is the single place in the module that classifies arbitrary
Go values into **scalars, object entries, array elements, nulls, and
unrepresentable values**. Before this layer existed, `gen`, `oj`, `alt`
(decompose/diff/checksum), and `jp` (walk) each maintained their own type
switches and reflection helpers, and the rules for nil handling, integer
width, and map keys drifted between entry points.

The layer has two parts:

- `node.Value` — a read-only *view* of one node in a value graph. It never
  copies the graph; children are adapted lazily through `Len`, `Index`,
  `Each`, and `Entries`.
- `node.Session` — a per-traversal cycle detector that returns a
  `*node.CycleError` carrying the path to the cycle instead of recursing
  until the stack overflows.

## Classification contract

`node.Of` classifies the generic data model directly:

| Go value | Kind |
|---|---|
| `nil` | `Null` |
| `bool` | `Bool` |
| `int` … `int64` | `Int` |
| `uint` … `uint64` | `Uint` |
| `float32`, `float64` | `Float` |
| `string` | `String` |
| `[]byte` | `Bytes` |
| `time.Time` | `Time` |
| `[]any`, `gen.Array` | `Array` |
| `map[string]any`, `gen.Object` | `Object` |
| anything else | `Other` |

`node.Reflect` adapts values outside the model without copying:

- **pointers and interfaces** are unwrapped; typed nils report `Null`;
- **named alias types** report the kind of their underlying type;
- **slices/arrays of any element type** report `Array` (a `[]byte` alias is
  an `Array`, only the exact `[]byte` type is `Bytes`);
- **maps with any key type** report `Object`; non-string keys are converted
  with `ojg.KeyString`, and `Entries` collapses keys that convert to the
  same string (last wins, matching `map[string]any` construction);
- **structs, complex numbers, channels, functions, unsafe pointers** report
  `Other` and are left to the caller, which keeps its own policy for them
  (e.g. `alt` keeps its `sinfo` based struct decomposition).

Signed and unsigned integers are distinct kinds so no precision is lost;
each entry point applies its own numeric policy on top (for example
`alt.Diff` compares through `int64` and `alt.Checksum` encodes the raw
`uint64` bits, exactly as before).

## Entry points and their policies

| Entry | Uses | Order policy | Cycle policy |
|---|---|---|---|
| `jp.Walk` | `Of` | native map order (unspecified) | unchanged (generic cycles still overflow) |
| `alt.Checksum` | `Of` + `Session` | object keys sorted | cyclic value encoded as null |
| `alt.Decompose`/`Alter` | `Reflect` + `Session` | n/a (builds maps) | cyclic value becomes nil |
| `alt.Diff`/`Compare` | `Reflect` + 2×`Session` | deterministic: sorted keys of the first map, then sorted keys unique to the second | cyclic value treated as nil |
| `alt.Match` | `Reflect` + 2×`Session` | n/a (boolean) | cyclic value treated as nil |
| `alt.Filter` match | `Reflect` + `Session` | n/a | cyclic value becomes nil |

`Diff` and `Match` compare two graphs, so each side gets its own
`Session`; otherwise the same map instance appearing on both sides would
look like a cycle.

## Complexity

- Classification is O(1) per node; `Each`/`Index` adapt children lazily, so
  walking a tree costs O(nodes) time and O(1) extra space per node.
- `Entries` (used by checksum of reflected maps) costs O(entries) space per
  object, as does key sorting. The generic `map[string]any` fast path in
  checksum collects keys on the stack for maps with up to 16 members.
- Cycle detection keeps a stack of the containers on the current path:
  O(depth) space and O(depth) per container enter. Scalars, empty
  containers, and nil-able nils are not tracked. Path segments are
  allocation-free value types (`node.Seg`); the `[]any` path is only built
  when a cycle is actually found.

## Compatibility notes

Preserved exactly (pinned by `alt/characterization_test.go` and
`jp/walk_characterization_test.go`):

- typed nil pointer/interface → null; typed nil slice/map → empty
  container in decompose but null in checksum; empty slice ≠ null in diff;
- integer width rules, including `uint64` wrapping to `int64` in decompose
  and diff comparisons, and named `uint` aliases staying `uint64`;
- `float32` rounding in decompose but not in checksum or for aliases;
- `[]byte` → string in decompose, raw bytes in checksum, int array for
  aliases; `time.Time` uses time options, `*time.Time` and aliases use
  struct reflection;
- struct checksums use `DefaultOptions` (`type` member, nil members
  omitted) while diff uses zero options;
- diff/match type-identity rules (named type ≠ underlying type) and the
  quirk that a typed nil pointer differs even from itself.

Intentional, documented changes:

- cyclic data no longer crashes with a stack overflow; the cyclic value is
  treated as nil (decompose/checksum/diff/match) and the traversal session
  records a `*node.CycleError` with the path;
- `alt.Diff`/`alt.Compare` report map member differences in a deterministic
  order instead of map iteration order;
- `alt.Filter` matching against cyclic data terminates instead of
  overflowing.

## Benchmarks

`jp` walk and `alt` checksum/diff benchmarks over a large nested document
(64 members × 16 sub-documents), before → after the migration
(Apple M-series, go1.26):

| Benchmark | before | after | allocs before | allocs after |
|---|---|---|---|---|
| WalkLarge | 119 µs | 128 µs | 5250 | 5250 |
| WalkLargeLeaves | 128 µs | 127 µs | 5250 | 5250 |
| WalkGenLarge | 92 µs | 95 µs | 3202 | 3202 |
| ChecksumLarge | 207 µs | 170 µs | 1047 | 27 |
| DiffLarge | 283 µs | 198 µs | 9 | 8 |
| Decompose (struct) | 167 ns | 184 ns | 7 | 8 |

Walk keeps identical allocation counts — the adapter adds no intermediate
tree. Checksum and diff are faster than before; decompose pays a small
fixed cost for the traversal session.
