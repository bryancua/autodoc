# core/paths

The path predicates AutoDoc's core packages share, written once. Paths are slash-separated and
clean, relative to a workspace or absolute.

| Function | Answers |
| --- | --- |
| `Under(p, dir)` | whether `p` is `dir` or inside it, by whole segments; every relative path is under `"."` |

Pure functions with no I/O, safe for concurrent use. An SQL form of `Under` (a `dao.Predicate`)
lives with the queries that need it, in `core/index` and `core/pgstore`.
