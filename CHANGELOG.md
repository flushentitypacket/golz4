# golz4 CHANGELOG

## v1.0.3

* Writer now supports any input size, not just blocks smaller than 65 KB. [PR 10](https://github.com/DataDog/golz4/pull/10)
* Writer ensures the double buffer used for writing do not move in memory. [PR 11](https://github.com/DataDog/golz4/pull/11)

## v1.0.2

* Fix panic with read when provided a buffer smaller than the decompressed data. The new version buffers the inflated data for later read calls when this happens. [PR 9](https://github.com/DataDog/golz4/pull/9)

## v1.0.1

Do not use deprecated LZ4 functions anymore. This removes the warnings that show up during compilation. The API or its behavior remains unchanged.

## v1.0.0

While this release **does not break API compatibility**, it changes the way the library is built.
Starting with this version, the C source code for `liblz4` **is not included in the Go package anymore**.

**The `liblz4` needs to be provided externally**, using a package manager or a manual, from source installation, for example.

Detection of `liblz4` now relies on `pkg-config` to add the correct `CFLAGS` and `LDFLAGS`.

## v0.0.131

* Initial release, using lz4 source code version r131
