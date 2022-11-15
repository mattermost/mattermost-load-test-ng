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

NOTE: Makefile no longer supports building the binary file using Windows on AMD64

The release process consists of two steps:
1. Prepare the release and get the automatically created PR merged.
2. Do the actual release.

#### Prepare the release

There are certain changes that we need to do for every release (for now, simply updating the default load-test URL in the config sample). You can automatically apply those changes with the following command:

```sh
make prepare-release NEXT_VER=v1.1.1
```

Follow the instructions in the output to create the PR. Once it's merged, continue with the next step.

#### Actual release

With the PR already merged, you can now finish the process with the following command:

```sh
make release NEXT_VER=v1.1.1
```

`goreleaser` needs to use a Github token to create the release for you. You can define it inline with the `GITHUB_TOKEN` variable or write it to the default `~/.config/goreleaser/github_token` file. Read [`goreleaser` docs](https://goreleaser.com/scm/github/) for more info.
