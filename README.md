## Instructions

Start by cloning this repository.
In a terminal window, change the working directory to `<PATH WHERE YOU CLONED THE REPO>/mempool`.

Running `make start` will:

1. Clean the build directory
2. Install Mockgen and generate the mocks for testing
3. Run all tests in the `/mempool/pkg` directory
4. Compile and install the application (binary will be placed in `/bin`)
5. Execute the application

In the root directory of the project, you will find a `.env` file. To decrease the log output level, set the `DEBUG` value to `false`. To change the capacity of the mempool, set `MAX_MEMPOOL_SIZE` to the desired value (the default value is 5000). The prioritized transactions output file is set by `PRIORITIZED_TX_FILE_PATH` (default: `./prioritized_transactions.txt`).

The program output 'prioritized_transactions.txt' can be found in the project root directory after execution has completed.

### Set Up and Run

```bash
source .env
make start
```

### Test

generate all mocks & run tests:

```bash
make test
```

generate all mocks & run tests (verbose):

```bash
make testv
```

run tests w/o generating mocks (quickly)

```bash
make testq
```

run tests verbose w/o generating mocks

```bash
make testqv
```

run code coverage

```bash
make covero
```

### Environment Variables

- `DEBUG`: Set to `true` for verbose logging (decreases performance).
- `TRANSACTIONS_FILE_PATH`: Path to the input transactions file (default: `./transactions.txt`).
- `MAX_MEMPOOL_SIZE`: Maximum number of transactions in the mempool (default: `5000`).
- `PRIORITIZED_TX_FILE_PATH`: Output file for prioritized transactions (default: `./prioritized_transactions.txt`).

### Mempool Changes

- The mempool now uses an explicit `uint32` for `MAX_MEMPOOL_SIZE`.
- The mempool must be started with a call to `StartProcessors(wg, numProcessors)` before adding transactions.
- Transactions are added using `AddTx`, which now manages the WaitGroup internally.
- The binary is output to the `/bin` directory.
- The `.env` file must be present and correctly configured for the application to run.

## Project Challenges and Solutions

### 1. Challenges Faced

- **Concurrency and Race Conditions:** Ensuring safe concurrent access to the mempool data structures (map, heap) and avoiding race conditions, especially with multiple goroutines processing transactions.
- **Duplicate Transaction Handling:** Preventing duplicate transactions from being processed or added, both in the main pool and in-flight (pending) state.
- **Channel Management:** Avoiding panics from sending on closed channels and managing the lifecycle of processor goroutines.
- **Test Synchronization:** Ensuring tests reliably wait for all transaction processing to complete, without negative WaitGroup counters or deadlocks.
- **Efficient Export:** Ensuring that exporting the mempool to a file is efficient and does not block or slow down the main application, even with a large number of transactions.

### 2. Solutions Implemented

- **Mutexes and Pending Checks:** Used mutexes to protect shared state and a `pendingChecks` map to track in-flight transactions, preventing duplicates.
- **Explicit Processor Startup:** Refactored to require explicit starting of mempool processors, giving more control and predictability in both main code and tests.
- **WaitGroup Management:** Centralized `WaitGroup` incrementing inside `AddTx` to ensure it only tracks successfully queued transactions, preventing negative counters.
- **Direct Transaction Submission:** Removed unnecessary goroutines in main transaction ingestion, relying on the mempool's own concurrency for safety and performance.
- **Optimized ExportToFile:** The `ExportToFile` function now minimizes lock duration, pops the transactions from the `TxHeap`, and writes all output in a single system call using a buffer. This greatly improves performance and reliability for large mempools.

### 3. Constraints

- **Go Standard Library Only:** Used only the Go standard library and a few well-known logging/testing packages.
- **Resource Limits:** Designed to handle a large number of transactions efficiently, but with a configurable maximum pool size to avoid unbounded memory usage.
- **Testability:** All concurrency and state management had to be testable and deterministic for CI.

### 4. Metrics & Extra Effort for Availability, Stability, Performance

- **WaitGroup and Channel Safety:** Extra care was taken to ensure that all goroutines complete and that channels are closed only when safe, to maintain stability.
- **Heap Operations:** Used Go's `container/heap` for efficient transaction prioritization.
- **Logging:** Added structured logging for all major events and errors to aid in debugging and monitoring.
- **Test Coverage:** Added unit and benchmark tests for heap and mempool operations to measure and maintain performance.
- **ExportToFile Performance:** Benchmarked and optimized the export process to ensure it scales with large mempools and does not block other operations.

### 5. Unique Architectural Decisions

- **Explicit Processor Control:** Processors are started explicitly via a `StartProcessors` method, allowing tests and main code to control concurrency and lifecycle.
- **Pending Transaction Tracking:** The use of a `pendingChecks` map to track transactions in-flight (not yet in the main pool) is a unique solution to the duplicate problem in a concurrent environment.
- **Type-Safe Heap Operations:** Added type-safe `PushTx` and `PopTx` methods to the heap for better performance and code clarity.
- **Efficient Export:** The export logic is now optimized for both correctness and speed, using a buffer and minimal locking.

For more details, see the code and comments in the `pkg/types/mempool.go` and `cmd/mempool/main.go` files.