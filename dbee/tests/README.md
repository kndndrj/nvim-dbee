# Raison d'être

This directory contains tests for the dbee project that are not unit tests.

## Tests:

Try to follow the uber-go style guide for tests, which can be found
[here](https://github.com/uber-go/guide/blob/master/style.md#test-tables).

### How to run tests

[Go testcontainers](https://golang.testcontainers.org/modules) is used to run integration tests
against the [adapters](./../adapters) package.

Testcontainers support two types of provider, docker and podman. If `podman` executable is detected,
then it will be used as the default provider. Otherwise `docker` will be used. When using podman,
the ryuk container (repear) need to be run as privileged, set env variable
`TESTCONTAINERS_RYUK_CONTAINER_PRIVILEGED=true` before running any tests to enable it.

### How to run tests

Tests are run via (from dbee pwd):

```bash
go test ./tests/... -v
```

If you want to disable cache add the `-count=1` flag:

```bash
go test ./tests/... -v -count=1
```

To run a specific adapter, you can use the `-run` flag:

```bash
go test ./tests/... -v -run Test<AdapterName>
```

For example, to run the `postgres` adapter tests:

```bash
go test ./tests/... -v -run TestPostgres
```

### Add new tests

Take a look at the `postgres` adapter example on how to add a new integration. Otherwise, the
default documentation from [testcontainers](https://golang.testcontainers.org/modules) is always
very helpful to look at.
