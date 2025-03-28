[![Package Docs](https://pkg.go.dev/badge/git.sr.ht/~nelsam/hel)](https://pkg.go.dev/git.sr.ht/~nelsam/hel) [![Build Status](https://builds.sr.ht/~nelsam/hel.svg)](https://builds.sr.ht/~nelsam/hel)

# Hel

A go mock generator that you don't have to think about.

## Quick Start

In your shell:

``` sh
$ go install git.sr.ht/~nelsam/hel@latest
$ hel ./...
```

In your tests:

``` go
package foo_test

import foo "."

func TestFoo(t *testing.T) {
  seq := pers.CallSequence(t)
  mock := newMockFoo(t, time.Second)
  pers.Expect(seq, mock.method.Baz,
    pers.WithArgs("the argument I expect"),
    pers.Return(errors.New("an error")),
  )
  foo.Bar(mock)
}
```

## About

In Norse mythology, Hel cares for the souls of people who didn't die in battle.

This little tool cares for mock implementations of interface types.

## Goals

- Generate mocks without a whole bunch of arguments and options.
  - My business logic usually makes it clear which interface types are going to
    need mocks to test a given package. Why do I have to tell the generator
    which types need mocks?
- Keep mocks in test files.
  - Interfaces generally belong in the package that uses them, not in the
    package that implements them.
    - If that's true, interfaces will (usually) only be used in one package - so
      why do mock generators always export them in non-test files?
- Make mocks that are difficult to use incorrectly.
  - I hate it when I see a data race in a test and find out it's because the
    test and the business logic both accessed a mock at the same time.
- Generate code that is easy to prove correct.
  - Tests are still code, and we still need to prove correctness. But we can't
    write tests to prove the correctness of the tests. So any test code
    (including mocks) should be provably correct with our eyeballs.

## Installation

Hel is go-installable.

`go install git.sr.ht/~nelsam/hel@latest`

We got to v4 before realizing that modules work a lot
better if you stay at v0, so we're back at v0.x now. All versions above v0 have
been retracted.

## Usage

At its simplest, you can just run `hel` without any options in the directory you
want to generate mocks for. By default, mocks are saved in `mock_helheim_test.go`.

See `hel -h` or `hel --help` for command line options.

### Dependency Tracking

Hel's greatest feature is that it understands dependencies, generating mocks for
anything that your code depends on. If you write a function that uses an
`http.Handler` as a parameter, hel will generate a `mockHTTPHandler`. It also
generates mocks for the interfaces' method parameters, so the `http.Handler`
would cause hel to generate a `mockHTTPResponseWriter` as well.

By default, `hel` will _not_ generate mocks for local interface types, instead
only mocking types which the business logic uses.

## Go Generate

The recommended way to use `hel` is to add one or more `//go:generate` comments
in projects which use it. Here are some examples:

#### In a file (e.g. `generate.go`) in the root of your project:

```go
//go:generate hel ./...
```

The above command would find all interface types depended on in the project and
generate mocks in `mock_helheim_test.go` in any packages it finds dependencies
for.

#### In a file (e.g. `generate.go`) in each package you want mocks to be generated for:

```go
//go:generate hel
```

The above command would generate mocks for dependencies of the current package
in `mock_helheim_test.go`

#### Above each interface type you want a mock for

```go
//go:generate hel --type Foo --output mock_foo_test.go

type Foo interface {
   Foo() string
}
```

The above command would generate a mock for the Foo type in
`mock_foo_test.go`
