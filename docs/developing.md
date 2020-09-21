## Developer's workflow

### Code checking

Before committing any code, you should check the code style and syntax by running:

```sh
make check-style
```

### Running tests

You can run all the available tests by executing:

```sh
make test
```

### Building

You can create a new packaged build with:

```sh
make package
```

### Releasing

We use a combination of `Makefile` and [Goreleser](https://goreleaser.com/) to automate our release process.
To build, package and publish a new release you can issue the following command:

```sh
make release NEXT_VER=v1.1.1
```
