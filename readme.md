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

`select` is one of the most powerful—and most misunderstood—tools in Go. It’s where goroutines, channels, cancellation, and timeouts all come together.

---

## 1. What is `select`?

`select` lets us **wait on multiple channel operations at the same time** and proceed with **exactly one** of them.

It’s Go’s equivalent of:

* “Wait for whichever happens first”
* “React to multiple concurrent events”

Basic form:

```go
select {
case v := <-ch1:
    handle(v)
case ch2 <- x:
    sent(x)
}
```

---

## 2. Why `select` exists

Without `select`, we would:

* Block on one channel at a time
* Write complex coordination logic
* Miss cancellation signals
* Deadlock easily

`select` gives us:

* Multiplexing over channels
* Cancellation and timeouts
* Non-blocking operations
* Clean concurrent state machines

---

## 3. How `select` works (core rules)

### Rule 1: Each `case` must be a channel operation

Valid operations:

* Receive: `<-ch`
* Send: `ch <- v`

Invalid:

* Function calls
* Conditionals
* Arbitrary expressions

---

### Rule 2: `select` blocks until a case is ready

If:

* At least one case is ready → one is chosen
* No cases are ready → `select` blocks
* `default` exists → it runs immediately

---

### Rule 3: If multiple cases are ready, one is chosen randomly

This is **pseudo-random**, but fair over time.

This prevents starvation.

---

## 4. Simple receive example

```go
select {
case v := <-ch:
    fmt.Println(v)
}
```

This behaves like a normal receive, but it scales when we add more cases.

---

## 5. Send and receive together

```go
select {
case ch <- x:
    fmt.Println("sent")
case v := <-ch:
    fmt.Println("received", v)
}
```

We don’t know which one will run—only that one will.

---

## 6. `default`: non-blocking select

Adding `default` makes `select` **non-blocking**.

```go
select {
case v := <-ch:
    use(v)
default:
    // no value available
}
```

Behavior:

* If no channel is ready → `default` executes immediately
* No blocking occurs

This is useful but dangerous if overused.

---

## 7. Busy loops (common mistake)

```go
for {
    select {
    case v := <-ch:
        handle(v)
    default:
        // do nothing
    }
}
```

This causes:

* 100% CPU usage
* Tight polling
* Performance collapse

Fix:

* Add blocking
* Use `time.Sleep`
* Or remove `default`

---

## 8. Timeouts with `select`

One of the most important uses.

```go
select {
case v := <-ch:
    process(v)
case <-time.After(time.Second):
    timeout()
}
```

We wait for:

* A value from `ch`
* OR a timeout

Whichever happens first wins.

---

## 9. Cancellation with `context.Context`

This is the **production-grade pattern**.

```go
select {
case v := <-ch:
    handle(v)
case <-ctx.Done():
    return
}
```

This ensures:

* Goroutines don’t leak
* Work can be stopped cleanly
* Shutdowns are predictable

---

## 10. Closed channels and `select`

Receiving from a closed channel is **always ready**.

```go
select {
case v, ok := <-ch:
    if !ok {
        ch = nil
    }
}
```

This is subtle and very important.

---

## 11. Disabling cases with `nil` channels

A `nil` channel:

* Blocks forever on send and receive
* Is never selected

We can use this to **dynamically enable/disable cases**.

```go
if done {
    ch = nil
}
```

This is a powerful state-machine technique.

---

## 12. Select in loops (typical pattern)

```go
for {
    select {
    case v := <-jobs:
        process(v)
    case <-ctx.Done():
        return
    }
}
```

This is the canonical worker loop.

---

## 13. Fan-in with `select`

```go
select {
case v := <-ch1:
    out <- v
case v := <-ch2:
    out <- v
}
```

We merge multiple input channels into one output channel.

---

## 14. Select fairness and starvation

When multiple cases are ready:

* Go randomizes selection
* Over time, all cases get chances

This avoids starvation but does **not** guarantee strict fairness.

We must not rely on order.

---

## 15. Common `select` mistakes (critical)

### 1. Forgetting to handle cancellation

```go
select {
case v := <-ch:
    process(v)
}
```

This goroutine can never stop.

Always include:

```go
case <-ctx.Done():
```

---

### 2. Assuming order

```go
select {
case <-ch1:
case <-ch2:
}
```

Order in code ≠ order in execution.

---

### 3. Blocking sends inside select chains

```go
select {
case v := <-ch:
    out <- v // can block!
}
```

This can deadlock. We often need nested selects.

---

## 16. Nested `select` (advanced but real)

```go
select {
case v := <-in:
    select {
    case out <- v:
    case <-ctx.Done():
        return
    }
case <-ctx.Done():
    return
}
```

This ensures:

* No blocking sends
* Clean cancellation

---

## 17. `select` vs `switch`

They look similar but are conceptually different:

| `select`           | `switch`         |
| ------------------ | ---------------- |
| Channel readiness  | Value comparison |
| Concurrent         | Sequential       |
| Random case choice | Deterministic    |

`select` is about **events**, not logic.

---

## 18. Final mental model

We should think of `select` as:

> “We wait until one of these channel events happens, then we react.”

If we can’t answer:

* What events are we waiting for?
* What happens if none occur?
* How do we stop?

Then the `select` logic is incomplete.

---

## 19. Practical rule to stay safe

Inside long-running goroutines:

* Always have a `select`
* Always handle cancellation
* Avoid `default` unless we truly want non-blocking behavior

---

Let’s go deep on **context timeouts** in Go concurrency.

This topic is critical because **timeouts are how we prevent goroutine leaks, runaway work, and stuck systems**. If we understand context timeouts well, our concurrent Go code becomes predictable and safe.

---

## 1. What is `context.Context` (quick grounding)

A `context.Context` is a value we pass through our call stack to carry:

* Cancellation signals
* Deadlines / timeouts
* Request-scoped values (used sparingly)

In concurrency, we mostly care about **cancellation and timeouts**.

---

## 2. What is a context timeout?

A **context timeout** is a deadline after which:

* The context is automatically canceled
* All goroutines observing that context are notified
* Blocking operations should stop

We create one using:

```go
ctx, cancel := context.WithTimeout(parent, 2*time.Second)
defer cancel()
```

This means:

> “All work using this context must finish within 2 seconds, or stop.”

---

## 3. What actually happens when the timeout expires

When the timeout is reached:

1. The context’s `Done()` channel is closed
2. `ctx.Err()` returns `context.DeadlineExceeded`
3. Any goroutine selecting on `ctx.Done()` unblocks
4. Cancellation propagates to all child contexts

No goroutine is killed automatically — **we must cooperate**.

---

## 4. The `Done()` channel (the heart of it)

Every context has:

```go
Done() <-chan struct{}
```

This channel:

* Is open initially
* Is closed on timeout or cancellation
* Never sends values
* Closes exactly once

This makes it perfect for `select`.

---

## 5. Basic timeout pattern (canonical)

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()

select {
case v := <-ch:
    process(v)
case <-ctx.Done():
    return ctx.Err()
}
```

This ensures:

* We don’t block forever
* We stop when time runs out
* We don’t leak goroutines

---

## 6. Why `defer cancel()` matters (even with timeouts)

Even if a timeout exists, we **must still call `cancel()`**.

Why?

* Frees timers early
* Releases internal resources
* Cancels child contexts immediately

Rule:

> If we call `WithTimeout`, we must call `cancel`.

Always.

---

## 7. Timeout vs manual cancellation

### Timeout

```go
ctx, cancel := context.WithTimeout(parent, d)
```

* Automatic cancellation after duration
* Used for bounded operations

---

### Manual cancellation

```go
ctx, cancel := context.WithCancel(parent)
```

* Cancellation happens when we call `cancel()`
* Used for lifecycle control (shutdowns)

In practice, we often combine both.

---

## 8. Context timeout in goroutines

A goroutine must **observe** the context.

```go
func worker(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            doWork()
        }
    }
}
```

If we don’t check `ctx.Done()`:

* Timeout does nothing
* Goroutine leaks
* System degrades over time

---

## 9. Context timeout with blocking operations

### Channel receive

```go
select {
case v := <-ch:
    handle(v)
case <-ctx.Done():
    return
}
```

---

### Channel send (often forgotten)

```go
select {
case out <- v:
case <-ctx.Done():
    return
}
```

Blocking sends are a **common leak source**.

---

## 10. Context timeout with I/O

Many standard library calls respect context:

```go
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
resp, err := http.DefaultClient.Do(req)
```

When timeout expires:

* Request is canceled
* Socket is closed
* Goroutine unblocks

This is why context exists in the first place.

---

## 11. `WithTimeout` vs `WithDeadline`

They are equivalent in behavior.

```go
context.WithTimeout(parent, 2*time.Second)
context.WithDeadline(parent, time.Now().Add(2*time.Second))
```

Use:

* `WithTimeout` → relative duration
* `WithDeadline` → absolute time

---

## 12. Context hierarchy and propagation

Contexts form a tree.

```go
parent
  └── child (timeout)
        └── grandchild
```

If:

* Parent is canceled → all children cancel
* Child times out → grandchildren cancel
* Grandchild cancels → parent is unaffected

This makes cancellation **structured and predictable**.

---

## 13. Timeout errors (`ctx.Err()`)

After cancellation:

```go
err := ctx.Err()
```

Returns:

* `context.DeadlineExceeded` → timeout
* `context.Canceled` → manual cancellation

We should check this to understand **why** work stopped.

---

## 14. Context timeout vs `time.After`

### `time.After`

```go
select {
case <-time.After(time.Second):
}
```

Problems:

* Timer can’t be canceled
* Leaks timers in loops
* No propagation

---

### Context timeout

```go
ctx, cancel := context.WithTimeout(...)
defer cancel()
```

Benefits:

* Cancelable
* Propagates
* Standardized
* Safer in long-lived systems

Use `time.After` only for very short, local waits.

---

## 15. Timeout in worker pools (real-world)

```go
for {
    select {
    case job := <-jobs:
        process(job)
    case <-ctx.Done():
        return
    }
}
```

This ensures:

* Workers exit cleanly
* Shutdown respects deadlines
* No dangling goroutines

---

## 16. Common mistakes (critical)

### 1. Creating context inside a loop

```go
for {
    ctx, cancel := context.WithTimeout(...)
    // leak if cancel not called
}
```

Fix:

* Always call `cancel`
* Or move context creation outside

---

### 2. Ignoring the context

```go
func work(ctx context.Context) {
    doBlockingThing() // ignores ctx
}
```

This defeats the entire purpose.

---

### 3. Passing `nil` context

Never do this.

Always use:

```go
context.Background()
```

or

```go
context.TODO()
```

---

## 17. Context timeout is NOT

Context timeout is **not**:

* A forceful kill
* A goroutine terminator
* A replacement for synchronization
* A timeout on CPU usage

It is a **cooperative cancellation signal**.

---

## 18. How context timeout fits with other primitives

| Tool            | Role        |
| --------------- | ----------- |
| Goroutines      | Run work    |
| Channels        | Communicate |
| Select          | React       |
| WaitGroup       | Wait        |
| Context timeout | Stop        |

They each solve **one problem**. Context handles stopping.

---

## 19. Final mental model (this matters)

We should think of a context timeout as:

> “A shared clock that tells all goroutines when their time is up.”

If a goroutine does not:

* Select on `ctx.Done()`
* Or pass context to blocking calls

It is ignoring the clock.

---

## 20. Practical rule to stay safe

In any goroutine that:

* Blocks
* Loops
* Talks to I/O
* Waits on channels

We must ask:

> “What context stops this?”

If there is no answer, we have a bug — just not one we’ve seen yet.

---

`sync.Once` is simple on the surface, but it solves a **very specific and critical concurrency problem**: **safe, one-time initialization**.

---

## 1. What is `sync.Once`?

`sync.Once` is a synchronization primitive that guarantees **a function runs exactly once**, no matter how many goroutines call it — and it does so **safely and efficiently**.

```go
var once sync.Once

once.Do(initFunc)
```

Guarantee:

> `initFunc` will run **once and only once** across the entire program lifetime.

---

## 2. Why `sync.Once` exists

In concurrent programs, multiple goroutines often need:

* Shared initialization
* Lazy setup
* Expensive one-time work

Without `sync.Once`, we would need:

* Mutexes
* Condition variables
* Error-prone flags

And we would still risk:

* Double initialization
* Data races
* Deadlocks

`sync.Once` exists to **make the correct solution trivial**.

---

## 3. Basic usage

```go
var once sync.Once

func initConfig() {
    loadConfig()
}

func handler() {
    once.Do(initConfig)
    useConfig()
}
```

No matter how many goroutines call `handler()`:

* `initConfig()` runs once
* All callers see initialized state

---

## 4. What `Once.Do` actually guarantees

`once.Do(f)` guarantees:

1. `f` runs **at most once**
2. If multiple goroutines call `Do` concurrently:

   * One runs `f`
   * Others block
3. When `Do` returns:

   * `f` has completed successfully or panicked

This means `Do` is a **full memory barrier**.

---

## 5. Memory visibility (very important)

After `once.Do(f)` returns:

* All writes performed inside `f`
* Are visible to all goroutines

This is stronger than just “runs once” — it’s **safe publication**.

This is why `sync.Once` is commonly used for:

* Lazy global variables
* Singleton initialization

---

## 6. What happens if `f` panics?

This is a critical detail.

If `f` panics:

* The panic propagates
* `sync.Once` considers `f` **done**
* `f` will **never run again**

```go
once.Do(func() {
    panic("boom")
})

once.Do(initFunc) // will NOT run
```

This is intentional.

Implication:

* Initialization functions must be robust
* Panics during init are usually fatal

---

## 7. `sync.Once` vs mutex + flag

### Without `sync.Once`

```go
var (
    mu   sync.Mutex
    done bool
)

func initOnce() {
    mu.Lock()
    defer mu.Unlock()

    if done {
        return
    }
    setup()
    done = true
}
```

This is:

* Verbose
* Easy to get wrong
* Hard to maintain

---

### With `sync.Once`

```go
var once sync.Once

func initOnce() {
    once.Do(setup)
}
```

This is:

* Clear
* Correct
* Fast
* Idiomatic

---

## 8. `sync.Once` is NOT resettable

This is by design.

```go
once.Do(f)
once.Do(f) // no-op
```

We cannot:

* Reset it
* Reuse it
* “Run again”

If we need reset behavior, we need a different design.

---

## 9. Common use cases

### 1. Lazy initialization

```go
var (
    once sync.Once
    db   *DB
)

func getDB() *DB {
    once.Do(func() {
        db = connect()
    })
    return db
}
```

---

### 2. Global setup

```go
func init() {
    once.Do(setupLogging)
}
```

---

### 3. One-time expensive computation

```go
once.Do(buildCache)
```

---

## 10. What `sync.Once` is NOT for

`sync.Once` is **not** for:

* Running something once per request
* Protecting mutable state
* Repeated lifecycle management
* Conditional execution

If we need:

* Multiple executions
* Reset
* State transitions

We should use:

* Mutexes
* Channels
* State machines

---

## 11. `sync.Once` and errors (common trap)

`sync.Once` does not return errors.

Bad pattern:

```go
once.Do(func() {
    err = setup()
})
```

If `setup()` fails:

* `once` is “done”
* We cannot retry
* We’re stuck with partial state

Correct approaches:

* Panic on fatal init failure
* Pre-check before `Do`
* Use a custom `Once` with error handling

---

## 12. Safe pattern for error-aware init

```go
var (
    once sync.Once
    initErr error
)

func initOnce() error {
    once.Do(func() {
        initErr = setup()
    })
    return initErr
}
```

We must accept:

* No retries
* First result wins

---

## 13. Internals (useful intuition)

Internally, `sync.Once` uses:

* An atomic flag
* A mutex on slow paths
* Memory barriers

Fast path:

* Already done → almost zero cost

This is why it’s extremely efficient.

---

## 14. `sync.Once` vs `init()` function

| `init()`        | `sync.Once`     |
| --------------- | --------------- |
| Runs at startup | Runs lazily     |
| Automatic       | Explicit        |
| Single-threaded | Concurrent-safe |
| No control      | Controlled      |

We use:

* `init()` for mandatory setup
* `sync.Once` for optional or lazy setup

---

## 15. Common mistakes (important)

### 1. Doing too much work inside `Do`

Long-running init:

* Blocks all callers
* Delays startup paths

Keep init short or isolate heavy work.

---

### 2. Calling `Do` inside `f`

```go
once.Do(func() {
    once.Do(other) // deadlock risk
})
```

Never do this.

---

### 3. Assuming retries

Once means once. No retries. Ever.

---

## 16. Final mental model

We should think of `sync.Once` as:

> “A one-time gate that closes forever after the first execution.”

Once the gate closes:

* No one else gets through
* Success or failure is final

---

## 17. Practical rule

If we need:

* Exactly-once initialization
* Concurrency safety
* Minimal overhead

`sync.Once` is the correct tool.

If we need:

* Retry
* Reset
* State changes

`sync.Once` is the wrong tool.

---

Mutexes are one of the **oldest and most fundamental concurrency tools**, and in Go they’re still essential — even with channels, contexts, and select. If we misunderstand mutexes, we end up with data races, deadlocks, or performance collapse.

---

## 1. What is a mutex?

A **mutex** (mutual exclusion lock) ensures that **only one goroutine at a time** can access a critical section of code or data.

In Go, mutexes live in the `sync` package:

```go
var mu sync.Mutex
```

We use it like this:

```go
mu.Lock()
defer mu.Unlock()
```

---

## 2. Why mutexes exist

Goroutines run concurrently. If multiple goroutines:

* Read and write shared memory
* Modify maps, slices, counters, structs

We get **data races**, which lead to:

* Undefined behavior
* Corrupted state
* Impossible-to-debug bugs

Mutexes give us:

* Safety
* Memory visibility guarantees
* Deterministic behavior

---

## 3. The critical section concept

A **critical section** is the smallest piece of code that must not run concurrently.

```go
mu.Lock()
counter++
mu.Unlock()
```

Key idea:

> We lock **data**, not code.

If goroutines touch the same mutable data, they must share the same mutex.

---

## 4. How Go mutexes work (behavioral guarantees)

When we call:

```go
mu.Lock()
```

* If unlocked → we acquire immediately
* If locked → we block until unlocked

When we call:

```go
mu.Unlock()
```

* Exactly one waiting goroutine is woken up
* Unlock must be called by the goroutine that locked

Unlocking an unlocked mutex **panics**.

---

## 5. Memory visibility (very important)

Mutexes are **memory barriers**.

Guarantee:

> All writes made before `Unlock()`
> are visible to all goroutines after `Lock()`.

This is why mutexes are safe for shared state.

---

## 6. Basic example

```go
type Counter struct {
    mu sync.Mutex
    n  int
}

func (c *Counter) Inc() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.n++
}

func (c *Counter) Value() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.n
}
```

This is the idiomatic pattern.

---

## 7. `defer mu.Unlock()` (best practice)

We almost always write:

```go
mu.Lock()
defer mu.Unlock()
```

Why:

* Prevents forgetting to unlock
* Safe on early returns
* Safe on panics

Exception:

* Extremely hot paths where performance is critical
* Then we must unlock manually with care

---

## 8. `sync.RWMutex` (read–write mutex)

### Why it exists

If:

* Many goroutines read
* Few goroutines write

A regular mutex becomes a bottleneck.

---

### RWMutex behavior

```go
var mu sync.RWMutex
```

* `RLock()` → multiple readers allowed
* `Lock()` → exclusive writer
* Writers block readers and writers
* Readers block writers

Example:

```go
mu.RLock()
v := data
mu.RUnlock()
```

---

### When RWMutex helps (and when it doesn’t)

Helps when:

* Reads are frequent
* Writes are rare
* Critical sections are short

Hurts when:

* Writes are frequent
* Critical sections are long
* Read/write ratio is unpredictable

RWMutex is **not always faster**.

---

## 9. Mutex vs channel (important comparison)

| Problem              | Mutex | Channel |
| -------------------- | ----- | ------- |
| Protect shared state | ✅     | ❌       |
| Ownership transfer   | ❌     | ✅       |
| Simple counters      | ✅     | ❌       |
| Pipelines            | ❌     | ✅       |
| Fine-grained locking | ✅     | ❌       |

Rule:

* **Mutexes protect memory**
* **Channels coordinate goroutines**

---

## 10. Common mutex patterns

### Protecting a map

```go
type SafeMap struct {
    mu sync.Mutex
    m  map[string]int
}
```

Never access maps concurrently without protection.

---

### Lazy initialization with mutex

```go
if v == nil {
    mu.Lock()
    if v == nil {
        v = init()
    }
    mu.Unlock()
}
```

(Double-checked locking — use carefully; `sync.Once` is usually better.)

---

## 11. Deadlocks (critical to understand)

### Self-deadlock

```go
mu.Lock()
mu.Lock() // deadlock
```

A goroutine cannot lock the same mutex twice.

---

### Lock ordering deadlock

```go
mu1.Lock()
mu2.Lock()
```

Another goroutine does:

```go
mu2.Lock()
mu1.Lock()
```

Both block forever.

Fix:

* Always acquire locks in a consistent order

---

## 12. Holding locks too long

```go
mu.Lock()
time.Sleep(time.Second)
mu.Unlock()
```

This:

* Kills concurrency
* Causes latency spikes
* Blocks unrelated work

Locks must:

* Be short
* Avoid I/O
* Avoid blocking operations

---

## 13. Mutexes and panic safety

If we forget to unlock during panic → deadlock.

This is why `defer mu.Unlock()` matters.

---

## 14. Zero value usability

A mutex’s zero value is usable:

```go
var mu sync.Mutex
```

No initialization required.

---

## 15. Copying mutexes (very dangerous)

❌ Never copy a mutex after first use:

```go
type Bad struct {
    mu sync.Mutex
}
b2 := b1 // copies mutex state
```

This leads to undefined behavior.

Always pass structs containing mutexes by pointer.

---

## 16. Mutex fairness and starvation

Go mutexes:

* Are not strictly fair
* Favor throughput
* Avoid convoying

This is good for performance but means:

* We must not assume fairness
* We must not depend on order

---

## 17. Mutexes and performance

Mutex cost is low but not free.

Performance tips:

* Minimize lock scope
* Avoid locking in tight loops
* Prefer local variables
* Avoid contention

---

## 18. When mutexes are the right tool

Use mutexes when:

* We have shared mutable state
* We need fast access
* Channels would complicate logic

Avoid mutexes when:

* Data can be passed by ownership
* Flow is naturally event-based
* We need cancellation signaling

---

## 19. Mutex + context (important reminder)

Mutexes do **not** support cancellation.

If a goroutine blocks on `Lock()`:

* Context cannot unblock it
* We must design around this

This is a major reason to keep locks short.

---

## 20. Final mental model

We should think of a mutex as:

> “A guard standing in front of our data, allowing only one goroutine through at a time.”

If we don’t know:

* What data is protected
* Who owns the lock
* How long it’s held

Then the mutex is being misused.

---

## 21. Practical rule to stay safe

If we can replace a mutex with:

* Clear ownership
* Channels
* Immutable data

We should.

But when we truly share memory:
**mutexes are the correct and necessary tool**.

---





