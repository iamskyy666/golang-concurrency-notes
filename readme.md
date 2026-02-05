Goroutines are Go’s core concurrency primitive. They let us run functions **concurrently** with very low overhead, making it practical to write highly concurrent programs without complex thread management.

**what they are, how they work, how they’re scheduled, how they communicate, and common pitfalls**.

---

## 1. What is a goroutine?

A **goroutine** is a lightweight, independently executing function managed by the Go runtime.

```go
go doWork()
```

This starts `doWork()` **concurrently** with the rest of the program.

Key points:

* Goroutines are **not OS threads**
* They are **much cheaper** than threads
* We can have **thousands or millions** of goroutines

---

## 2. Goroutines vs OS threads

| Feature        | OS Thread   | Goroutine            |
| -------------- | ----------- | -------------------- |
| Creation cost  | High        | Very low             |
| Stack size     | Large (MBs) | Small (starts ~2 KB) |
| Scheduling     | OS kernel   | Go runtime           |
| Context switch | Expensive   | Cheap                |

The Go runtime **multiplexes** many goroutines onto a smaller number of OS threads.

---

## 3. How goroutines are created

Any function call can become a goroutine using the `go` keyword:

```go
go fmt.Println("Hello")
```

Important:

* The function starts **asynchronously**
* The caller does **not wait**
* No return values (use channels instead)

This will likely print nothing unless the program waits.

---

## 4. The Go scheduler (G–M–P model)

Go uses a **user-space scheduler**, not the OS scheduler.

### Core components

| Component | Meaning                       |
| --------- | ----------------------------- |
| **G**     | Goroutine                     |
| **M**     | OS thread (Machine)           |
| **P**     | Processor (scheduler context) |

### How it works

* Each **P** has a run queue of goroutines
* **M** executes goroutines from a **P**
* `GOMAXPROCS` controls number of Ps (default = CPU cores)

```go
runtime.GOMAXPROCS(4)
```

This design:

* Avoids excessive thread creation
* Enables work stealing
* Keeps CPUs busy efficiently

---

## 5. Goroutine stacks (important detail)

Goroutines use **growable stacks**.

* Start small (~2 KB)
* Automatically grow and shrink
* No fixed size like threads

This is a major reason goroutines are so lightweight.

---

## 6. Concurrency vs parallelism

**Concurrency** ≠ **Parallelism**

* **Concurrency**: managing multiple tasks at once
* **Parallelism**: executing tasks at the same time

Goroutines enable concurrency.
Parallelism happens only if:

* Multiple CPUs
* `GOMAXPROCS > 1`

We can have concurrency on a single core.

---

## 7. Synchronization and communication

Goroutines **should not share memory directly**.

> “Do not communicate by sharing memory; share memory by communicating.”

### Channels (preferred)

```go
ch := make(chan int)

go func() {
    ch <- 42
}()

value := <-ch
```

Channels:

* Synchronize goroutines
* Pass data safely
* Block by default

---

### WaitGroups (for coordination)

```go
var wg sync.WaitGroup

wg.Add(1)
go func() {
    defer wg.Done()
    doWork()
}()

wg.Wait()
```

Use this to:

* Wait for goroutines to finish
* Avoid premature program exit

---

### Mutexes (when needed)

```go
var mu sync.Mutex
mu.Lock()
counter++
mu.Unlock()
```

Use mutexes when:

* Sharing mutable state
* Channels would complicate logic

---

## 8. Common goroutine patterns

### Fan-out / Fan-in

```go
for i := 0; i < 10; i++ {
    go worker(jobs)
}
```

Multiple workers consume from a shared channel.

---

### Worker pool

Limits concurrency to avoid overload.

```go
sem := make(chan struct{}, 5)

for _, task := range tasks {
    sem <- struct{}{}
    go func(t Task) {
        defer func() { <-sem }()
        process(t)
    }(task)
}
```

---

### Fire-and-forget (dangerous)

```go
go logEvent(e)
```

Risk:

* Goroutine may never finish
* Silent failures

Avoid unless intentionally detached.

---

## 9. Common pitfalls (very important)

### 1. Program exits too early

```go
go work()
```

Main exits → goroutine is killed.

Fix: `WaitGroup`, channel, or sleep (not recommended).

---

### 2. Loop variable capture bug

❌ Bug:

```go
for i := 0; i < 5; i++ {
    go func() {
        fmt.Println(i)
    }()
}
```

✅ Fix:

```go
for i := 0; i < 5; i++ {
    go func(i int) {
        fmt.Println(i)
    }(i)
}
```

---

### 3. Goroutine leaks

Blocked forever:

```go
ch := make(chan int)
go func() {
    ch <- 1 // blocks forever if no receiver
}()
```

Always ensure:

* Channels are read
* Goroutines can exit

---

### 4. Unbounded goroutines

```go
for {
    go handleRequest()
}
```

This can:

* Exhaust memory
* Kill performance

Use worker pools or rate limiting.

---

## 10. Debugging goroutines

### Stack dump

```go
runtime.Stack(buf, true)
```

### Deadlock detection

Go runtime will panic on:

* All goroutines asleep
* Channel deadlocks

---

## 11. When to use goroutines (and when not)

### Use goroutines when:

* I/O-bound work
* Independent tasks
* Concurrent pipelines

### Avoid goroutines when:

* Tight CPU loops with no blocking
* Simpler sequential code is enough
* Shared state is complex and fragile

Concurrency adds complexity. Use it deliberately.

---

## 12. Mental model to keep us safe

Think of goroutines as:

> “Cheap, cancellable units of work that must be **owned, synchronized, and stopped**.”

If we can’t answer:

* Who starts it?
* Who stops it?
* Who waits for it?

We’re setting ourself up for bugs.

---
Let’s go deep on **`sync.WaitGroup`**, because it’s one of the most important (and commonly misused) synchronization tools in Go.

---

## 1. What is a WaitGroup?

A **WaitGroup** is a synchronization primitive that lets **one or more goroutines wait until a set of other goroutines finishes**.

In plain terms:

> We use a WaitGroup when we start multiple goroutines and need to **wait for all of them to complete** before moving on.

It lives in the `sync` package:

```go
import "sync"
```

---

## 2. The core idea (mental model)

A WaitGroup maintains an **internal counter**.

* We **increment** the counter when we start work
* We **decrement** the counter when work finishes
* We **block** until the counter reaches zero

That’s it. No magic beyond that.

---

## 3. The three methods

A `sync.WaitGroup` has exactly **three methods**:

### 1. `Add(delta int)`

Adjusts the counter.

```go
wg.Add(1)
```

* Positive value → increase counter
* Negative value → decrease counter
* Counter **must never go negative**

---

### 2. `Done()`

Signals that one unit of work is finished.

```go
wg.Done()
```

This is exactly the same as:

```go
wg.Add(-1)
```

---

### 3. `Wait()`

Blocks until the counter becomes zero.

```go
wg.Wait()
```

---

## 4. Basic example

```go
var wg sync.WaitGroup

wg.Add(1)

go func() {
    defer wg.Done()
    doWork()
}()

wg.Wait()
fmt.Println("All done")
```

Execution flow:

1. We set counter to `1`
2. We start a goroutine
3. The goroutine finishes and calls `Done()`
4. Counter becomes `0`
5. `Wait()` unblocks

---

## 5. Why WaitGroups exist

Without a WaitGroup:

```go
go work()
```

* `main()` exits
* Program terminates
* Goroutine is killed mid-execution

WaitGroups give us **lifecycle control** over goroutines.

---

## 6. Multiple goroutines

```go
var wg sync.WaitGroup

for i := 0; i < 5; i++ {
    wg.Add(1)
    go func(i int) {
        defer wg.Done()
        process(i)
    }(i)
}

wg.Wait()
```

Key idea:

* Each goroutine owns **one `Done()`**
* The main goroutine owns **one `Wait()`**

---

## 7. Rules we must follow (non-negotiable)

### Rule 1: Call `Add()` before starting the goroutine

❌ Wrong (race condition):

```go
go func() {
    wg.Add(1)
    defer wg.Done()
}()
```

Why this is bad:

* The goroutine might start **after** `Wait()` runs
* This causes undefined behavior or panic

✅ Correct:

```go
wg.Add(1)
go func() {
    defer wg.Done()
}()
```

---

### Rule 2: Every `Add(1)` must have exactly one `Done()`

If we miss a `Done()`:

* `Wait()` blocks forever (deadlock)

If we call too many `Done()`:

* Panic: **negative WaitGroup counter**

---

### Rule 3: Never copy a WaitGroup

❌ Very dangerous:

```go
func work(wg sync.WaitGroup) {
    wg.Done()
}
```

This copies the internal state.

✅ Always pass by pointer:

```go
func work(wg *sync.WaitGroup) {
    defer wg.Done()
}
```

---

## 8. Using `defer wg.Done()` (best practice)

We almost always do this:

```go
go func() {
    defer wg.Done()
    doWork()
}()
```

Why:

* Guarantees `Done()` runs
* Safe even if:

  * We return early
  * We panic

---

## 9. WaitGroup does NOT pass data

This is critical to understand.

❌ Wrong use case:

```go
// WaitGroup is NOT for communication
```

WaitGroups:

* Only wait for completion
* Do NOT transfer values
* Do NOT signal errors

For data:

* Use **channels**
* Or shared state + mutex

---

## 10. WaitGroup vs Channels

| Use case                 | WaitGroup | Channel    |
| ------------------------ | --------- | ---------- |
| Wait for completion      | ✅         | ⚠️ awkward |
| Pass data                | ❌         | ✅          |
| Signal multiple workers  | ⚠️        | ✅          |
| Simple lifecycle control | ✅         | ❌          |

Rule of thumb:

* **WaitGroup = waiting**
* **Channel = communication**

---

## 11. Common mistakes (very important)

### 1. Calling `Add()` concurrently with `Wait()`

This is illegal.

```go
go wg.Add(1)
wg.Wait()
```

This can panic.

Correct pattern:

* All `Add()` calls happen **before** `Wait()`
* Or are strictly ordered

---

### 2. Forgetting to wait

```go
wg.Add(1)
go work()
```

No `Wait()` → program exits → goroutine dies.

---

### 3. Goroutine leaks disguised as WaitGroup bugs

WaitGroup doesn’t cancel goroutines.

If a goroutine blocks forever:

* `Done()` is never called
* `Wait()` blocks forever

WaitGroup is **not a cancellation mechanism**.

---

## 12. WaitGroup + Context (real-world pattern)

This is the **correct production pattern**:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

var wg sync.WaitGroup

wg.Add(1)
go func() {
    defer wg.Done()
    worker(ctx)
}()

// later
cancel()
wg.Wait()
```

* `context` → controls stopping
* `WaitGroup` → controls waiting

They solve different problems and work together.

---

## 13. Internals (useful to know)

Internally, a WaitGroup:

* Uses atomic counters
* Has a semaphore-like mechanism
* Is optimized and lock-free in common cases

This is why:

* It’s fast
* It’s safe
* But also strict about misuse

---

## 14. When we should use WaitGroups

Use them when:

* We launch goroutines
* We need to wait for all of them
* No data needs to be returned

Avoid them when:

* We need streaming results
* We need error propagation
* We need cancellation (use context)

---

## 15. Final mental checklist

Before using a WaitGroup, we should be able to answer:

* Who calls `Add()`?
* Who calls `Done()`?
* Who calls `Wait()`?
* What guarantees that `Done()` is always called?

If we can’t answer these clearly, the design needs rework.

---

Channels are the **other half of Go’s concurrency story**. Goroutines give us concurrency; **channels give us safe coordination and communication**.

---

## 1. What is a channel?

A **channel** is a typed conduit that lets goroutines **send and receive values safely**.

```go
ch := make(chan int)
```

Conceptually:

> A channel is a **thread-safe queue** managed by the Go runtime that also provides **synchronization**.

---

## 2. Why channels exist

Without channels, goroutines would have to:

* Share memory
* Protect everything with mutexes
* Coordinate timing manually

Channels let us:

* Transfer ownership of data
* Synchronize execution
* Avoid most explicit locking

This leads to Go’s famous rule:

> **Do not communicate by sharing memory; share memory by communicating.**

---

## 3. Basic channel operations

### Creating a channel

```go
ch := make(chan int)
```

This creates an **unbuffered channel**.

---

### Sending

```go
ch <- 42
```

Send blocks until someone receives.

---

### Receiving

```go
v := <-ch
```

Receive blocks until someone sends.

---

### Directional types (optional)

```go
var sendOnly chan<- int
var recvOnly <-chan int
```

We often use these in function signatures to enforce correctness.

---

## 4. Unbuffered channels (synchronous)

Unbuffered channels have **no capacity**.

```go
ch := make(chan int)
```

Behavior:

* Send blocks until a receiver is ready
* Receive blocks until a sender is ready

This creates a **handshake**.

Example:

```go
go func() {
    ch <- 10
}()

fmt.Println(<-ch)
```

Execution:

1. Sender blocks
2. Receiver arrives
3. Value transfers
4. Both proceed

---

## 5. Buffered channels (asynchronous)

Buffered channels have capacity.

```go
ch := make(chan int, 3)
```

Behavior:

* Send blocks only when buffer is full
* Receive blocks only when buffer is empty

Example:

```go
ch <- 1
ch <- 2
ch <- 3
// ch <- 4 // blocks
```

Buffering trades **synchronization** for **throughput**.

---

## 6. Choosing buffer size

Rule of thumb:

* `0` → strict synchronization
* `1` → signal / semaphore behavior
* `N` → bounded queue / worker pool

We should **never use unbounded buffering** (which Go doesn’t allow anyway).

---

## 7. Closing channels

```go
close(ch)
```

Closing means:

* No more sends allowed
* Receivers can continue draining values
* Further receives return zero value + `ok=false`

```go
v, ok := <-ch
```

* `ok == false` → channel is closed and empty

---

### Who should close a channel?

**Only the sender. Always.**

Receivers must **never** close a channel.

Reason:

* Closing is a signal that no more values will arrive
* Only the producer knows when production is done

---

## 8. Ranging over channels

```go
for v := range ch {
    fmt.Println(v)
}
```

This loop:

* Receives values
* Stops automatically when channel is closed

This is the cleanest consumption pattern.

---

## 9. Channels as synchronization tools

Channels are not just for data.

### Signaling completion

```go
done := make(chan struct{})

go func() {
    work()
    close(done)
}()

<-done
```

We use `struct{}` because it allocates nothing.

---

### Semaphore / concurrency limiting

```go
sem := make(chan struct{}, 5)

for _, task := range tasks {
    sem <- struct{}{}
    go func(t Task) {
        defer func() { <-sem }()
        process(t)
    }(task)
}
```

This limits concurrency to 5 goroutines.

---

## 10. Select statement

`select` lets us wait on **multiple channel operations**.

```go
select {
case v := <-ch1:
    handle(v)
case ch2 <- x:
    sent()
case <-time.After(time.Second):
    timeout()
}
```

Key rules:

* One ready case is chosen randomly
* If none are ready, `select` blocks
* `default` makes it non-blocking

---

## 11. Default case (non-blocking)

```go
select {
case v := <-ch:
    use(v)
default:
    // no value available
}
```

This prevents blocking but must be used carefully.

---

## 12. Channel direction in APIs (best practice)

```go
func producer(out chan<- int) {
    out <- 1
}

func consumer(in <-chan int) {
    fmt.Println(<-in)
}
```

This:

* Documents intent
* Prevents misuse
* Improves maintainability

---

## 13. Common channel patterns

### Fan-out

```go
for i := 0; i < workers; i++ {
    go worker(jobs)
}
```

Multiple goroutines read from the same channel.

---

### Fan-in

```go
func merge(cs ...<-chan int) <-chan int
```

Multiple channels merged into one.

---

### Pipeline

```go
gen -> square -> sum
```

Each stage:

* Receives from input channel
* Sends to output channel
* Closes output when done

---

## 14. Common mistakes (critical)

### 1. Sending on a closed channel

```go
panic: send on closed channel
```

This is always a bug.

---

### 2. Closing a channel too early

If we close while goroutines are still sending → panic.

---

### 3. Forgetting to close a channel

Receivers block forever.

---

### 4. Goroutine leaks

```go
for v := range ch {
    // blocks forever if ch is never closed
}
```

We must always ensure:

* Channels are closed
* Or receivers can exit another way (context)

---

## 15. Channels vs Mutexes

| Problem                 | Channels | Mutex |
| ----------------------- | -------- | ----- |
| Ownership transfer      | ✅        | ❌     |
| Simple state protection | ❌        | ✅     |
| Pipelines               | ✅        | ❌     |
| Shared counters         | ❌        | ✅     |

Rule:

* **Channels for coordination**
* **Mutexes for state**

---

## 16. Channels + Context (production-grade)

```go
select {
case v := <-ch:
    process(v)
case <-ctx.Done():
    return
}
```

This prevents goroutine leaks and allows clean shutdowns.

---

## 17. Final mental model

We should think of channels as:

> “A safe handoff point where one goroutine gives responsibility for a value to another.”

If multiple goroutines:

* Send without receivers → deadlock
* Receive without senders → deadlock

Channels force us to design concurrency **explicitly**.

---

Buffered vs unbuffered channels is one of those topics that *looks simple* but quietly determines whether our Go programs are clean, fast, or full of deadlocks.


---

## 1. What “buffered” and “unbuffered” really mean

At the core, the difference is **where a value waits** when it’s sent.

* **Unbuffered channel** → value waits in the *sender* until a receiver is ready
* **Buffered channel** → value waits in the *channel buffer* until a receiver takes it

That single difference affects:

* Blocking behavior
* Synchronization
* Performance
* Correctness

---

## 2. Unbuffered channels (capacity = 0)

### Definition

```go
ch := make(chan int)
```

No capacity. No queue. No storage.

---

### Blocking behavior

With an unbuffered channel:

* Send blocks until a receiver is ready
* Receive blocks until a sender is ready

This creates a **synchronous rendezvous**.

```go
ch <- 10   // blocks
v := <-ch // blocks
```

The send and receive complete **at the same time**.

---

### Timeline view

```
Sender:   ch <- 10  ────────┐
                            ├── value transfers
Receiver:        <- ch ─────┘
```

Neither side can proceed alone.

---

### What this guarantees

Unbuffered channels guarantee:

* The receiver has started before the sender continues
* Precise handoff
* Strong ordering

This makes them **synchronization primitives**, not queues.

---

### Example: strict sequencing

```go
ch := make(chan struct{})

go func() {
    fmt.Println("step 1")
    ch <- struct{}{}
}()

<-ch
fmt.Println("step 2")
```

We are guaranteed:

```
step 1
step 2
```

No races. No guessing.

---

### When unbuffered channels shine

We use unbuffered channels when:

* Ordering matters
* We want backpressure
* We want explicit synchronization
* We want to detect misuse early (deadlocks show up fast)

They are safer by default.

---

## 3. Buffered channels (capacity > 0)

### Definition

```go
ch := make(chan int, 3)
```

This channel has **space for 3 values**.

---

### Blocking behavior

With a buffered channel:

* Send blocks only when buffer is full
* Receive blocks only when buffer is empty

```go
ch <- 1 // does not block
ch <- 2 // does not block
ch <- 3 // does not block
// ch <- 4 // blocks
```

---

### Timeline view

```
Sender:   ch <- 1   ch <- 2   ch <- 3
Channel: [ 1 ][ 2 ][ 3 ]
Receiver:                    <- ch
```

Senders and receivers are **decoupled** (up to capacity).

---

### What buffering changes

Buffering:

* Increases throughput
* Reduces synchronization
* Hides timing dependencies

But it also:

* Hides bugs
* Allows bursts
* Delays backpressure

---

## 4. Capacity = 1 (the special case)

```go
ch := make(chan int, 1)
```

This is extremely common.

Why?

* Acts like a **binary semaphore**
* Allows one value “in flight”
* Reduces blocking while keeping control

Example:

```go
lock := make(chan struct{}, 1)
lock <- struct{}{} // acquire

// critical section

<-lock // release
```

This is valid, though mutexes are usually clearer.

---

## 5. Comparing behavior side-by-side

### Same code, different behavior

#### Unbuffered

```go
ch := make(chan int)

go func() {
    ch <- 1
    fmt.Println("sent")
}()

fmt.Println(<-ch)
```

Output order is guaranteed:

```
1
sent
```

---

#### Buffered

```go
ch := make(chan int, 1)

go func() {
    ch <- 1
    fmt.Println("sent")
}()

fmt.Println(<-ch)
```

Possible output:

```
sent
1
```

Buffering changes **ordering guarantees**.

---

## 6. Backpressure (critical concept)

### Unbuffered channels enforce backpressure

Sender cannot outrun receiver.

This is ideal for:

* Pipelines
* Resource-limited systems
* Preventing overload

---

### Buffered channels soften backpressure

Sender can run ahead until buffer fills.

This is ideal for:

* Burst handling
* I/O smoothing
* Worker queues

But dangerous if unbounded work is possible.

---

## 7. Deadlocks and debugging

### Unbuffered: fail fast

```go
ch := make(chan int)
ch <- 1 // deadlock
```

This deadlocks immediately. That’s good — the bug is obvious.

---

### Buffered: fail late

```go
ch := make(chan int, 1000)
for i := 0; i < 1000; i++ {
    ch <- i
}
```

This might work in tests but deadlock in production.

Buffered channels can **delay failures**, making bugs harder to find.

---

## 8. Buffered channels are NOT queues (by default)

While they look like queues, they:

* Have fixed capacity
* Block instead of growing
* Require explicit closing
* Do not support peeking or length guarantees for logic

If we treat them as general-purpose queues, we will eventually get stuck.

---

## 9. Choosing buffer size (practical rules)

### Rule 1: Start with unbuffered

Unbuffered channels force us to reason about synchronization.

---

### Rule 2: Add buffering deliberately

Add a buffer only when:

* We measured a bottleneck
* We need burst tolerance
* We understand the timing relationship

---

### Rule 3: Small buffers are usually enough

Common sizes:

* `1` → signal / semaphore
* `N workers` → worker pool
* `CPU count` → throughput tuning

Large buffers are a smell.

---

## 10. Channels, buffering, and ownership

* Unbuffered channel → ownership transfers immediately
* Buffered channel → ownership transfers when received, not when sent

This matters when values contain:

* Pointers
* Mutable state
* Resources

We must not mutate data after sending unless we know it hasn’t been received yet.

---

## 11. Closing behavior (same for both)

Closing rules:

* Closing does **not** clear the buffer
* Receivers drain buffered values first
* After buffer is empty, receives return zero values

```go
close(ch)
for v := range ch {
    fmt.Println(v)
}
```

Works the same for buffered and unbuffered channels.

---

## 12. When to use which (clear guidance)

### Use **unbuffered channels** when:

* Correctness > performance
* Ordering matters
* We want strict synchronization
* We want bugs to surface early

### Use **buffered channels** when:

* We need burst absorption
* We want throughput over strict ordering
* We control capacity carefully
* We understand the blocking behavior

---

## 13. Final mental model (the one to remember)

We should think:

* **Unbuffered channel**
  → “We meet, exchange the value, then continue together.”

* **Buffered channel**
  → “We drop the value here and move on—up to a limit.”

If we don’t know *where the value waits*, we don’t understand the channel.

---



