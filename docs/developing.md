## Developer's workflow
### Notes
The load-test tool does not support Windows.

### Code checking

Before committing any code, you should check the code style and syntax by running:

```sh
make check-style
```

Also, if you happen to have modified any file under the `deployment/terraform/assets/` directory, you need to regenerate the corresponding code by running:

```sh
make assets
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

