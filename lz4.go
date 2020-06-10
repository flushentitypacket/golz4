package lz4

// #cgo pkg-config: liblz4
// #include <lz4.h>
// #include <stdlib.h>
import "C"

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"unsafe"
)

const (
	streamingBlockSize        = 1024 * 64
	blockHeaderSize           = 4
	boundedStreamingBlockSize = streamingBlockSize + streamingBlockSize/255 + 16
)

// p gets a char pointer to the first byte of a []byte slice
func p(in []byte) *C.char {
	if len(in) == 0 {
		return (*C.char)(unsafe.Pointer(nil))
	}
	return (*C.char)(unsafe.Pointer(&in[0]))
}

// clen gets the length of a []byte slice as a char *
func clen(s []byte) C.int {
	return C.int(len(s))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Uncompress with a known output size. len(out) should be equal to
// the length of the uncompressed out.
func Uncompress(out, in []byte) (outSize int, err error) {
	outSize = int(C.LZ4_decompress_safe(p(in), p(out), clen(in), clen(out)))
	if outSize < 0 {
		err = errors.New("Malformed compression stream")
	}
	return
}

// CompressBound calculates the size of the output buffer needed by
// Compress. This is based on the following macro:
//
// #define LZ4_COMPRESSBOUND(isize)
//      ((unsigned int)(isize) > (unsigned int)LZ4_MAX_INPUT_SIZE ? 0 : (isize) + ((isize)/255) + 16)
func CompressBound(in []byte) int {
	return len(in) + ((len(in) / 255) + 16)
}

// Compress compresses in and puts the content in out. len(out)
// should have enough space for the compressed data (use CompressBound
// to calculate). Returns the number of bytes in the out slice.
func Compress(out, in []byte) (outSize int, err error) {
	outSize = int(C.LZ4_compress_default(p(in), p(out), clen(in), clen(out)))
	if outSize == 0 {
		err = errors.New("Insufficient space for compression")
	}
	return
}

// Writer is an io.WriteCloser that lz4 compress its input.
type Writer struct {
	compressionBuffer      [2]unsafe.Pointer
	lz4Stream              *C.LZ4_stream_t
	underlyingWriter       io.Writer
	inpBufIndex            int
	totalCompressedWritten int
}

// NewWriter creates a new Writer. Writes to
// the writer will be written in compressed form to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		compressionBuffer: [2]unsafe.Pointer{
			C.malloc(streamingBlockSize),
			C.malloc(streamingBlockSize),
		},
		lz4Stream:        C.LZ4_createStream(),
		underlyingWriter: w,
	}
}

// Write writes a compressed form of src to the underlying io.Writer.
func (w *Writer) Write(src []byte) (int, error) {
	remainingBytes := len(src)
	totalWritten := 0

	for remainingBytes > 0 {
		endIdx := totalWritten + streamingBlockSize
		if endIdx > len(src) {
			endIdx = len(src)
		}
		written, err := w.writeFrame(src[totalWritten:endIdx])
		if err != nil {
			return totalWritten, err
		}
		totalWritten += written
		remainingBytes -= written
	}

	return totalWritten, nil
}

func (w *Writer) writeFrame(src []byte) (int, error) {
	var compressedBuf [boundedStreamingBlockSize]byte
	inpPtr := w.nextInputBuffer()

	copy(inpPtr, src)

	written := int(C.LZ4_compress_fast_continue(
		w.lz4Stream,
		p(inpPtr),
		p(compressedBuf[:]),
		C.int(len(src)),
		C.int(len(compressedBuf)),
		1))
	if written <= 0 {
		return 0, errors.New("error compressing")
	}

	// Write "header" to the buffer for decompression
	var header [4]byte
	binary.LittleEndian.PutUint32(header[:], uint32(written))
	_, err := w.underlyingWriter.Write(header[:])
	if err != nil {
		return 0, err
	}

	// Write to underlying buffer
	_, err = w.underlyingWriter.Write(compressedBuf[:written])
	if err != nil {
		return 0, err
	}

	w.totalCompressedWritten += written + 4
	return len(src), nil
}

func (w *Writer) nextInputBuffer() []byte {
	w.inpBufIndex = (w.inpBufIndex + 1) % 2
	tmpSlice := reflect.SliceHeader{
		Data: uintptr(w.compressionBuffer[w.inpBufIndex]),
		Len:  streamingBlockSize,
		Cap:  streamingBlockSize,
	}
	return *(*[]byte)(unsafe.Pointer(&tmpSlice))
}

// Close releases all the resources occupied by Writer.
// w cannot be used after the release.
func (w *Writer) Close() error {
	if w.lz4Stream != nil {
		C.LZ4_freeStream(w.lz4Stream)
		w.lz4Stream = nil
	}
	C.free(w.compressionBuffer[0])
	C.free(w.compressionBuffer[1])
	return nil
}

// reader is an io.ReadCloser that decompresses when read from.
type reader struct {
	lz4Stream        *C.LZ4_streamDecode_t
	pending          []byte
	left             unsafe.Pointer
	right            unsafe.Pointer
	underlyingReader io.Reader
	isLeft           bool
}

// NewReader creates a new io.ReadCloser.  Reads from the returned ReadCloser
// read and decompress data from r.  It is the caller's responsibility to call
// Close on the ReadCloser when done.  If this is not done, underlying objects
// in the lz4 library will not be freed.
func NewReader(r io.Reader) io.ReadCloser {
	return &reader{
		lz4Stream:        C.LZ4_createStreamDecode(),
		underlyingReader: r,
		isLeft:           true,
		// As per lz4 docs:
		//
		//   *_continue() :
		//     These decoding functions allow decompression of multiple blocks in "streaming" mode.
		//     Previously decoded blocks must still be available at the memory position where they were decoded.
		//
		// double buffer needs to use C.malloc to make sure the same memory address
		// allocate buffers in go memory will fail randomly since GC may move the memory
		left:  C.malloc(boundedStreamingBlockSize),
		right: C.malloc(boundedStreamingBlockSize),
	}
}

// Close releases all the resources occupied by r.
// r cannot be used after the release.
func (r *reader) Close() error {
	if r.lz4Stream != nil {
		C.LZ4_freeStreamDecode(r.lz4Stream)
		r.lz4Stream = nil
	}

	C.free(r.left)
	C.free(r.right)
	return nil
}

// Read decompresses `compressionBuffer` into `dst`.
// dst buffer must of at least streamingBlockSize bytes large
func (r *reader) Read(dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}
	// Write data read from a previous call
	if r.pending != nil {
		return r.readFromPending(dst)
	}

	blockSize, err := r.readSize(r.underlyingReader)
	if err != nil {
		return 0, err
	}

	// read blockSize from r.underlyingReader --> readBuffer
	var uncompressedBuf [boundedStreamingBlockSize]byte
	_, err = io.ReadFull(r.underlyingReader, uncompressedBuf[:blockSize])
	if err != nil {
		return 0, err
	}

	var ptr unsafe.Pointer
	if r.isLeft {
		ptr = r.left
		r.isLeft = false
	} else {
		ptr = r.right
		r.isLeft = true
	}

	decompressed := int(C.LZ4_decompress_safe_continue(
		r.lz4Stream,
		(*C.char)(unsafe.Pointer(&uncompressedBuf[0])),
		(*C.char)(ptr),
		C.int(blockSize),
		C.int(streamingBlockSize),
	))

	if decompressed < 0 {
		return decompressed, errors.New("error decompressing")
	}

	mySlice := C.GoBytes(ptr, C.int(decompressed))
	copySize := min(decompressed, len(dst))

	copied := copy(dst, mySlice[:copySize])

	if decompressed > len(dst) {
		// Save data for future reads
		r.pending = mySlice[copied:]
	}

	return copied, nil
}

// read the 4-byte little endian size from the head of each stream compressed block
func (r *reader) readSize(rdr io.Reader) (int, error) {
	var temp [4]byte
	_, err := io.ReadFull(rdr, temp[:])
	if err != nil {
		return 0, err
	}

	return int(binary.LittleEndian.Uint32(temp[:])), nil
}

func (r *reader) readFromPending(dst []byte) (int, error) {
	copySize := min(len(dst), len(r.pending))
	copied := copy(dst, r.pending[:copySize])

	if copied == len(r.pending) {
		r.pending = nil
	} else {
		r.pending = r.pending[copied:]
	}
	return copied, nil
}

// CompressReader reads input and creates an io.ReadCloser for reading
// compressed output
type CompressReader struct {
	underlyingReader       io.Reader
	compressionBuffer      [2]unsafe.Pointer
	outputBuffer           *bytes.Buffer
	lz4Stream              *C.LZ4_stream_t
	inpBufIndex            int
	totalCompressedWritten int
}

// NewCompressReader creates a new io.ReadCloser.  Reads from the returned ReadCloser
// read and compress data from r.  It is the caller's responsibility to call
// Close on the ReadCloser when done.  If this is not done, underlying objects
// in the lz4 library will not be freed.
func NewCompressReader(r io.Reader) *CompressReader {
	return &CompressReader{
		compressionBuffer: [2]unsafe.Pointer{
			C.malloc(streamingBlockSize),
			C.malloc(streamingBlockSize),
		},
		lz4Stream:        C.LZ4_createStream(),
		underlyingReader: r,
		outputBuffer:     bytes.NewBuffer(nil),
	}
}

func (r *CompressReader) Read(dst []byte) (int, error) {
	// try to consume from the buffer
	// If the buffer contains anything it's leftover from a previous call
	n, err := r.outputBuffer.Read(dst)
	if n > 0 {
		return n, nil
	}

	// the buffer is empty, we are going to write into it so we reset it first
	var compressedBuf [boundedStreamingBlockSize + blockHeaderSize]byte
	inpPtr := r.nextInputBuffer()

	bytesRead, err := io.ReadFull(r.underlyingReader, inpPtr)
	if err == io.EOF {
		// nothing left to read from the source
		return 0, err
	}
	if err != nil && err != io.ErrUnexpectedEOF {
		// ErrUnexpectedEOF occurs when some bytes are read but not all the bytes (n > 0)
		return 0, fmt.Errorf("error reading source: %s", err)
	}
	// fmt.Printf("ReadFull: read %d bytes\n", bytesRead)

	// compress and write the data into compressedBuf, leaving space for the
	// 4 byte header
	written := int(C.LZ4_compress_fast_continue(
		r.lz4Stream,
		p(inpPtr),
		p(compressedBuf[blockHeaderSize:]),
		C.int(bytesRead),
		C.int(len(compressedBuf)-blockHeaderSize),
		1))
	if written <= 0 {
		return 0, errors.New("error compressing")
	}

	// fmt.Printf("wrote %d compressed bytes\n", written)
	// fmt.Printf("compressedBuf %v\n", compressedBuf)

	// Write "header" to the buffer for decompression at the first 4 bytes
	binary.LittleEndian.PutUint32(compressedBuf[:blockHeaderSize], uint32(written))

	// populate the buffer with our internal slice and consume from it
	r.outputBuffer = bytes.NewBuffer(compressedBuf[:written+blockHeaderSize])
	n, _ = r.outputBuffer.Read(dst)
	// here we ignore any EOF because the buffer contains partial data only
	// EOF will be communicated on the next call if the underlying Reader is exhausted

	r.totalCompressedWritten += written + 4
	return n, nil
}

func (r *CompressReader) nextInputBuffer() []byte {
	r.inpBufIndex = (r.inpBufIndex + 1) % 2
	tmpSlice := reflect.SliceHeader{
		Data: uintptr(r.compressionBuffer[r.inpBufIndex]),
		Len:  streamingBlockSize,
		Cap:  streamingBlockSize,
	}
	return *(*[]byte)(unsafe.Pointer(&tmpSlice))
}

// Close releases all the resources occupied by Reader.
// r cannot be used after the release.
func (r *CompressReader) Close() error {
	if r.lz4Stream != nil {
		C.LZ4_freeStream(r.lz4Stream)
		r.lz4Stream = nil
	}
	C.free(r.compressionBuffer[0])
	C.free(r.compressionBuffer[1])
	return nil
}
