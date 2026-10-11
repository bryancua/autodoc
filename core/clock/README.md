# core/clock

AutoDoc's time. The daemon builds one `Clock` and hands it to every core package that records a
time or compares with one, through that package's `WithClock` option.

| Name | What |
| --- | --- |
| `Clock` | `func() time.Time`; `Now()` reads it, and the zero `Clock` is `time.Now` |
| `Sleep(ctx, d)` | waits for `d` or `ctx`, and reports whether `d` elapsed |

**What goes through the Clock:** a time that is stored (an `updated_at`, an event's time), or
compared with one (a retry deadline, a change log's age).

**What stays on the monotonic clock:** a latency measured for a log line (`time.Since(start)`), and
the timers a loop waits on. A test's `Clock` does not move them.

Safe for concurrent use. A test's `Clock` must be too.
