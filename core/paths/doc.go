// Package paths is the path predicates AutoDoc's core packages share, so each is written once.
// Paths are slash-separated and clean: a workspace's relative paths, or absolute ones.
//
// Concurrency: the package is pure functions; any number of goroutines may call them.
package paths
