golz4
=====

[![Godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/DataDog/golz4) [![license](http://img.shields.io/badge/license-BSD-red.svg?style=flat)](https://raw.githubusercontent.com/DataDog/golz4/master/LICENSE) [![CircleCI](https://circleci.com/gh/DataDog/golz4.svg?style=svg)](https://circleci.com/gh/DataDog/golz4)

Golang interface to LZ4 compression.

Forked from `github.com/cloudflare/golz4` but with significant differences:

* input/output arg order has been swapped to follow Go convention, ie `Compress(in, out)` -> `Compress(out, in)`
* builds against the liblz4 that it detects using pkgconfig

Benchmark
```
BenchmarkCompress-4                 	 4002601	       284 ns/op	 151.65 MB/s	       0 B/op	       0 allocs/op
BenchmarkCompressUncompress-4       	14668696	        75.6 ns/op	 568.66 MB/s	       0 B/op	       0 allocs/op
BenchmarkStreamCompress-4           	     306	   4032676 ns/op	2600.20 MB/s	23627182 B/op	     643 allocs/op
BenchmarkStreamCompressReader-4     	    1617	    700174 ns/op	14975.94 MB/s	    7856 B/op	     163 allocs/op
BenchmarkStreamUncompress-4         	     100	  10385084 ns/op	1009.69 MB/s	22283872 B/op	     485 allocs/op
BenchmarkStreamDecompressReader-4   	     236	   5060225 ns/op	2072.19 MB/s	    8557 B/op	     324 allocs/op`
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
