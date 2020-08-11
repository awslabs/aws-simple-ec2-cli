# AWS Simple EC2 CLI: Build Instructions

## Install Go version 1.13+

There are several options for installing go:

1. If you're on mac, you can simply `brew install go`
2. If you'd like a flexible go installation manager consider using gvm https://github.com/moovweb/gvm
3. For all other situations use the official go getting started guide: https://golang.org/doc/install

## Build

This project uses `make` to organize compilation, build, and test targets.

To build cmd/main.go, which will build the full static binary and pull in depedent packages, run:
```
$ make build
```

The resulting binary will be in the generated `build/` dir

```
$ make build

$ ls build/
ez-ec2
```

## Test

You can execute the unit tests for the instance selector with `make`:

```
$ make unit-test
```

### Run All Tests

The full suite includes license-test, go-report-card, and more. See the full list in the [makefile](./Makefile). NOTE: some tests require AWS Credentials to be configured on the system: 

```
$ make test
```

## Format

To keep our code readable with go conventions, we use `goimports` to format the source code.
Make sure to run `goimports` before you submit a PR or you'll be caught by our tests! 

You can use the `make fmt` target as a convenience
```
$ make fmt
```