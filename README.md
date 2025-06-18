# Mempool Project

## Recent Changes

### Min-Heap Transaction Prioritization
- The mempool now uses a **min-heap** (`TxHeap`) for transaction prioritization.
- The transaction with the **lowest** `TotalFee` is always at the top of the heap.
- When the mempool reaches its maximum size, incoming transactions are compared against the lowest-fee transaction. If the new transaction has a higher fee, it replaces the lowest-fee transaction. This ensures the mempool always contains the highest-fee transactions.

### Exporting Transactions in Descending Order
- The `ExportToFile` function now pops all transactions from the min-heap into a slice and reverses the slice before writing to the output file.
- This guarantees that the exported file lists transactions from **highest to lowest TotalFee**.

### Explicit Processor Control
- Processors are now explicitly started via the `StartProcessors(wg, numProcessors)` method, providing clear control over concurrency and lifecycle management.

### WaitGroup Management
- The `WaitGroup` incrementing logic is centralized within the `AddTx` method, ensuring accurate tracking of transaction processing and preventing synchronization issues.

### Optimized Export Performance
- The export logic has been optimized to minimize lock duration and efficiently write transactions to the output file using buffered writes, significantly improving performance for large mempools.

### Updated Tests
- All heap-related tests have been updated to reflect min-heap behavior.
- Integration and mempool tests now correctly validate the prioritization and eviction logic.
- ---

## Instructions

Start by cloning this repository.  
In a terminal window, change the working directory to `<PATH WHERE YOU CLONED THE REPO>/mempool`.

Running `make start` will:

1. Clean the build directory.
2. Install Mockgen and generate mocks for testing.
3. Run all tests in the `/mempool/pkg` directory.
4. Compile and install the application (binary will be placed in `/bin`).
5. Execute the application.

In the root directory of the project, you will find a `.env` file. To decrease the log output level, set the `DEBUG` value to `false`. To change the capacity of the mempool, set `MAX_MEMPOOL_SIZE` to the desired value (default is `5000`). The prioritized transactions output file is set by `PRIORITIZED_TX_FILE_PATH` (default: `./prioritized_transactions.txt`).

The program output `prioritized_transactions.txt` can be found in the project root directory after execution has completed.

### Set Up and Run

```bash
source .env
make start
```

### Test

Generate all mocks & run tests:

```bash
make test
```

Generate all mocks & run tests (verbose):

```bash
make testv
```

Run tests without generating mocks (quickly):

```bash
make testq
```

Run tests verbose without generating mocks:

```bash
make testqv
```

Run code coverage:

```bash
make covero
```

### Environment Variables

- `DEBUG`: Set to `true` for verbose logging (decreases performance).
- `TRANSACTIONS_FILE_PATH`: Path to the input transactions file (default: `./transactions.txt`).
- `MAX_MEMPOOL_SIZE`: Maximum number of transactions in the mempool (default: `5000`).
- `PRIORITIZED_TX_FILE_PATH`: Output file for prioritized transactions (default: `./prioritized_transactions.txt`).

---

## Project Challenges and Solutions

### Challenges Faced
- **Concurrency and Race Conditions:** Ensuring safe concurrent access to mempool data structures and avoiding race conditions.
- **Duplicate Transaction Handling:** Preventing duplicate transactions from being processed or added.
- **Channel Management:** Avoiding panics from sending on closed channels and managing goroutine lifecycles.
- **Efficient Export:** Ensuring efficient export of transactions without blocking or slowing down the main application.

### Solutions Implemented
- **Mutexes and Pending Checks:** Used mutexes and a `pendingChecks` map to track in-flight transactions, preventing duplicates.
- **Explicit Processor Startup:** Refactored to explicitly start mempool processors, improving control and predictability.
- **WaitGroup Management:** Centralized `WaitGroup` incrementing inside `AddTx` to prevent synchronization issues.
- **Optimized ExportToFile:** Minimized lock duration and optimized file writing for performance.

### Constraints
- **Go Standard Library Only:** Limited external dependencies to standard library and well-known logging/testing packages.
- **Resource Limits:** Configurable maximum pool size to manage memory usage effectively.
- **Testability:** Ensured deterministic and testable concurrency and state management.

### Metrics & Extra Effort for Availability, Stability, Performance
- **WaitGroup and Channel Safety:** Ensured goroutines complete safely and channels close appropriately.
- **Heap Operations:** Leveraged Go's `container/heap` for efficient prioritization.
- **Logging:** Implemented structured logging for debugging and monitoring.
- **Test Coverage:** Added comprehensive unit and benchmark tests to maintain performance.

### Unique Architectural Decisions
- **Explicit Processor Control:** Provided explicit control over processor concurrency and lifecycle.
- **Pending Transaction Tracking:** Implemented a `pendingChecks` map to manage in-flight transactions.
- **Type-Safe Heap Operations:** Added type-safe heap methods for clarity and performance.
- **Efficient Export:** Optimized export logic for correctness and speed.

For more details, see the code and comments in the `pkg/types/mempool.go` and `cmd/mempool/main.go`