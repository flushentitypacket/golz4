golz4
=====

[![Godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/DataDog/golz4) [![license](http://img.shields.io/badge/license-BSD-red.svg?style=flat)](https://raw.githubusercontent.com/DataDog/golz4/master/LICENSE) [![CircleCI](https://circleci.com/gh/DataDog/golz4.svg?style=svg)](https://circleci.com/gh/DataDog/golz4)

Golang interface to LZ4 compression.

Forked from `github.com/cloudflare/golz4` but with significant differences:

* input/output arg order has been swapped to follow Go convention, ie `Compress(in, out)` -> `Compress(out, in)`
* builds against the liblz4 that it detects using pkgconfig

Benchmark 
```
BenchmarkCompress-8             	 5000000	       234 ns/op	 183.73 MB/s	       0 B/op	       0 allocs/op
BenchmarkCompressUncompress-8   	20000000	        62.4 ns/op	 688.60 MB/s	       0 B/op	       0 allocs/op
BenchmarkStreamCompress-8       	   50000	     32842 ns/op	2003.41 MB/s	  278537 B/op	       4 allocs/op
BenchmarkStreamUncompress-8     	  500000	      2867 ns/op	22855.34 MB/s	      52 B/op	       2 allocs/op
```

Building
--------

Building `golz4` requires that [lz4](https://github.com/lz4/lz4) library be available.

On Debian or Ubuntu, this is as easy as running

```
$ sudo apt-get install liblz4-dev
```

On MacOS

```
$ brew install lz4
```

If the library version provided for your OS is too old and does not include a `liblz4.pc` pkg-config file, the [upstream documentation](https://github.com/lz4/lz4#installation) describes how to build and install from source.

_NOTE_: if `lz4` is not installed in standard directories, setting `PKG_CONFIG_PATH` environment variable with the directory containing the `liblz4.pc` file will help.
