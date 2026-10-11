// Package clock is AutoDoc's time: the one Clock a daemon builds and hands to every core package
// that records a time or compares with one, and a context-aware Sleep.
//
// A Clock reads wall time for what is stored or compared with what is stored (an updated_at, a
// retry deadline). A latency measured for a log line (time.Since a start) stays on the monotonic
// clock: a test's Clock does not move it.
//
// Concurrency: a Clock is a function value; time.Now is safe for concurrent use, and a test's
// Clock must be too. Sleep may be called from any goroutine.
package clock
