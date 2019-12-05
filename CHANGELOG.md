# golz4 CHANGELOG

## v1.0.0

While this release **does not break API compatibility**, it changes the way the library is built.
Starting with this version, the C source code for `liblz4` **is not included in the Go package anymore**.

**The `liblz4` needs to be provided externally**, using a package manager or a manual, from source installation, for example.

Detection of `liblz4` now relies on `pkg-config` to add the correct `CFLAGS` and `LDFLAGS`.

## v0.0.131

* Initial release, using lz4 source code version r131
