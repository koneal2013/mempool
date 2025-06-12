## Instructions

Start by cloning this repository.
In a terminal window, change the working directory to ``<PATH WHERE YOU CLONED THE REPO>/mempool``.

Running ``make start`` will:

1. Clean the build directory
2. Install Mockgen and generate the mocks for testing
3. Run all test in the ``/mempool/pkg`` directory
4. Compile and install the applicaton
5. Execute the application

In the root directory of the project, you will find a ``.env`` file. To decrease the log output level, set the ``DEBUG`` value to ``false``. To change the capacity of the mempool, set ``MAX_MEMPOOL_SIZE`` to the desired value (the default value is 5000).

The program output 'prioritized-transactions.txt' can be found in the project root directory after execution has completed.

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

### Troubleshooting

If you encounter the following error:
``mockgen: command not found``
run the following command:

```bash
export GOPATH="$HOME/go"
    PATH="$GOPATH/bin:$PATH"
    go install github.com/golang/mock/mockgen
```

