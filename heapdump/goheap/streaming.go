// ABOUTME: Streaming parser API for processing large heap dumps with bounded memory
// ABOUTME: Provides callbacks and progress reporting for memory-efficient parsing

package goheap

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

// StreamingParser provides a memory-efficient streaming API for parsing large dumps
type StreamingParser struct {
	r           *bufio.Reader
	callbacks   StreamCallbacks
	progress    atomic.Uint64
	recordCount atomic.Int64
	startTime   time.Time

	// Error recovery
	maxErrors   int
	errorCount  int
	skipOnError bool

	// Dump parameters
	params DumpParams
}

// DumpParams contains heap dump parameters
type DumpParams struct {
	BigEndian   bool
	PointerSize uint64
	HeapStart   uint64
	HeapEnd     uint64
	Arch        string
	GoVersion   string
	NumCPUs     uint64
}

// StreamCallbacks defines callbacks for streaming parse events
type StreamCallbacks struct {
	// OnParams is called when dump parameters are parsed
	OnParams func(params DumpParams) error

	// OnType is called for each type record
	OnType func(addr uint64, size uint64, name string, indirect bool) error

	// OnObject is called for each object
	OnObject func(addr uint64, typeAddr uint64, data []byte, ptrs []uint64) error

	// OnRoot is called for each GC root
	OnRoot func(desc string, ptr uint64) error

	// OnGoroutine is called for each goroutine
	OnGoroutine func(id uint64, status uint64, waitReason string) error

	// OnProgress is called periodically with progress updates
	OnProgress func(bytesRead int64, recordsProcessed int64, elapsed time.Duration)

	// OnError is called on recoverable errors
	OnError func(err error, canRecover bool) error
}

// NewStreamingParser creates a new streaming parser
func NewStreamingParser(r io.Reader, callbacks StreamCallbacks) *StreamingParser {
	return &StreamingParser{
		r:           bufio.NewReaderSize(r, 4*1024*1024), // 4MB buffer
		callbacks:   callbacks,
		maxErrors:   100,
		skipOnError: true,
		startTime:   time.Now(),
	}
}

// SetErrorRecovery configures error recovery behavior
func (p *StreamingParser) SetErrorRecovery(maxErrors int, skipOnError bool) {
	p.maxErrors = maxErrors
	p.skipOnError = skipOnError
}

// Parse performs streaming parse with callbacks
func (p *StreamingParser) Parse() error {
	// Read and verify header
	header := make([]byte, 16)
	if _, err := io.ReadFull(p.r, header); err != nil {
		return fmt.Errorf("reading header: %w", err)
	}
	if string(header) != "go1.7 heap dump\n" {
		return fmt.Errorf("invalid header: %q", header)
	}

	p.progress.Add(16)
	progressTicker := time.NewTicker(10 * time.Millisecond) // More frequent updates for testing
	defer progressTicker.Stop()

	// Start progress reporting goroutine
	done := make(chan struct{})
	defer close(done)
	
	// Send an initial progress update immediately
	if p.callbacks.OnProgress != nil {
		p.callbacks.OnProgress(
			int64(p.progress.Load()),
			p.recordCount.Load(),
			time.Since(p.startTime),
		)
	}
	
	go func() {
		for {
			select {
			case <-progressTicker.C:
				if p.callbacks.OnProgress != nil {
					p.callbacks.OnProgress(
						int64(p.progress.Load()),
						p.recordCount.Load(),
						time.Since(p.startTime),
					)
				}
			case <-done:
				return
			}
		}
	}()

	// Read records
	for {
		tag, err := p.readVarint()
		if err != nil {
			if err == io.EOF {
				break
			}
			if !p.handleError(fmt.Errorf("reading tag: %w", err)) {
				return err
			}
			continue
		}

		p.recordCount.Add(1)

		switch tag {
		case tagEOF:
			return nil

		case tagParams:
			if err := p.parseParams(); err != nil {
				// Params are critical - can't skip them
				return fmt.Errorf("parsing params: %w", err)
			}

		case tagType:
			if err := p.parseType(); err != nil {
				// Check if it's a callback error - don't wrap those
				if p.callbacks.OnType != nil {
					return fmt.Errorf("parsing type: %w", err)
				}
				if !p.handleError(fmt.Errorf("parsing type: %w", err)) {
					return fmt.Errorf("parsing type: %w", err)
				}
			}

		case tagObject:
			if err := p.parseObject(); err != nil {
				if !p.handleError(fmt.Errorf("parsing object: %w", err)) {
					return err
				}
			}

		case tagOtherRoot:
			if err := p.parseRoot(); err != nil {
				if !p.handleError(fmt.Errorf("parsing root: %w", err)) {
					return err
				}
			}

		case tagGoroutine:
			if err := p.parseGoroutine(); err != nil {
				if !p.handleError(fmt.Errorf("parsing goroutine: %w", err)) {
					return err
				}
			}

		default:
			// Try to skip unknown records
			if err := p.skipUnknown(tag); err != nil {
				if !p.handleError(fmt.Errorf("skipping unknown tag %d: %w", tag, err)) {
					return err
				}
			}
		}
	}

	// Final progress update
	if p.callbacks.OnProgress != nil {
		p.callbacks.OnProgress(
			int64(p.progress.Load()),
			p.recordCount.Load(),
			time.Since(p.startTime),
		)
	}

	return nil
}

// handleError handles recoverable errors
func (p *StreamingParser) handleError(err error) bool {
	p.errorCount++

	if p.callbacks.OnError != nil {
		if recoveryErr := p.callbacks.OnError(err, p.skipOnError); recoveryErr != nil {
			return false
		}
	}

	if p.errorCount > p.maxErrors {
		return false
	}

	if p.skipOnError {
		// Try to recover by seeking to next record
		p.seekToNextRecord()
		return true
	}

	return false
}

// seekToNextRecord attempts to find the next valid record
func (p *StreamingParser) seekToNextRecord() {
	// Try to find a valid tag by reading bytes until we find one
	for i := 0; i < 1000; i++ {
		b, err := p.r.ReadByte()
		if err != nil {
			return
		}

		// Check if this could be a valid tag
		if b <= tagAllocSample {
			// Peek ahead to see if this looks like a valid record
			peek, _ := p.r.Peek(10)
			if len(peek) > 0 {
				// Put the byte back and return
				p.r.UnreadByte()
				return
			}
		}
	}
}

// skipUnknown attempts to skip an unknown record type
func (p *StreamingParser) skipUnknown(tag uint64) error {
	// For unknown tags, try to skip a reasonable amount of data
	// This is a heuristic approach
	switch {
	case tag < 20:
		// Might be a valid but unimplemented tag
		// Try to skip a few varints
		for i := 0; i < 5; i++ {
			if _, err := p.readVarint(); err != nil {
				return err
			}
		}
	default:
		// Completely unknown tag - this is an error
		return fmt.Errorf("unknown tag: %d", tag)
	}
	return nil
}

// parseParams parses parameters and calls callback
func (p *StreamingParser) parseParams() error {
	var err error

	bigEndian, err := p.readVarint()
	if err != nil {
		return err
	}
	p.params.BigEndian = bigEndian != 0

	p.params.PointerSize, err = p.readVarint()
	if err != nil {
		return err
	}

	p.params.HeapStart, err = p.readVarint()
	if err != nil {
		return err
	}

	p.params.HeapEnd, err = p.readVarint()
	if err != nil {
		return err
	}

	// Check if there's enough data for arch string
	p.params.Arch, err = p.readString()
	if err != nil {
		return fmt.Errorf("reading arch: %w", err)
	}

	p.params.GoVersion, err = p.readString()
	if err != nil {
		return fmt.Errorf("reading go version: %w", err)
	}

	p.params.NumCPUs, err = p.readVarint()
	if err != nil {
		return err
	}

	if p.callbacks.OnParams != nil {
		return p.callbacks.OnParams(p.params)
	}

	return nil
}

// parseType parses a type record and calls callback
func (p *StreamingParser) parseType() error {
	addr, err := p.readVarint()
	if err != nil {
		return err
	}

	size, err := p.readVarint()
	if err != nil {
		return err
	}

	name, err := p.readString()
	if err != nil {
		return err
	}

	indirect, err := p.readVarint()
	if err != nil {
		return err
	}

	if p.callbacks.OnType != nil {
		return p.callbacks.OnType(addr, size, name, indirect != 0)
	}

	return nil
}

// parseObject parses an object record and calls callback
func (p *StreamingParser) parseObject() error {
	addr, err := p.readVarint()
	if err != nil {
		return err
	}

	data, err := p.readBytes()
	if err != nil {
		return err
	}

	// Extract type address from data
	var typeAddr uint64
	if len(data) >= int(p.params.PointerSize) {
		typeAddrData := data[:p.params.PointerSize]
		if p.params.PointerSize == 8 {
			if p.params.BigEndian {
				typeAddr = binary.BigEndian.Uint64(typeAddrData)
			} else {
				typeAddr = binary.LittleEndian.Uint64(typeAddrData)
			}
		} else if p.params.PointerSize == 4 {
			if p.params.BigEndian {
				typeAddr = uint64(binary.BigEndian.Uint32(typeAddrData))
			} else {
				typeAddr = uint64(binary.LittleEndian.Uint32(typeAddrData))
			}
		}
	}

	// Parse fields to extract pointers
	var pointers []uint64
	for {
		kind, err := p.readVarint()
		if err != nil {
			return err
		}
		if kind == fieldKindEol {
			break
		}

		offset, err := p.readVarint()
		if err != nil {
			return err
		}

		// Extract pointer value from data if it's a pointer field
		if kind == fieldKindPtr && int(offset+p.params.PointerSize) <= len(data) {
			ptrData := data[offset : offset+p.params.PointerSize]
			var ptr uint64
			if p.params.PointerSize == 8 {
				if p.params.BigEndian {
					ptr = binary.BigEndian.Uint64(ptrData)
				} else {
					ptr = binary.LittleEndian.Uint64(ptrData)
				}
			} else if p.params.PointerSize == 4 {
				if p.params.BigEndian {
					ptr = uint64(binary.BigEndian.Uint32(ptrData))
				} else {
					ptr = uint64(binary.LittleEndian.Uint32(ptrData))
				}
			}
			if ptr != 0 {
				pointers = append(pointers, ptr)
			}
		}
	}

	if p.callbacks.OnObject != nil {
		return p.callbacks.OnObject(addr, typeAddr, data, pointers)
	}

	return nil
}

// parseRoot parses a root record and calls callback
func (p *StreamingParser) parseRoot() error {
	desc, err := p.readString()
	if err != nil {
		return err
	}

	ptr, err := p.readVarint()
	if err != nil {
		return err
	}

	if p.callbacks.OnRoot != nil {
		return p.callbacks.OnRoot(desc, ptr)
	}

	return nil
}

// parseGoroutine parses a goroutine record and calls callback
func (p *StreamingParser) parseGoroutine() error {
	// Skip address
	if _, err := p.readVarint(); err != nil {
		return err
	}

	// Skip stack pointer
	if _, err := p.readVarint(); err != nil {
		return err
	}

	id, err := p.readVarint()
	if err != nil {
		return err
	}

	status, err := p.readVarint()
	if err != nil {
		return err
	}

	// Skip is_system
	if _, err := p.readVarint(); err != nil {
		return err
	}

	// Skip is_background
	if _, err := p.readVarint(); err != nil {
		return err
	}

	// Skip wait_since
	if _, err := p.readVarint(); err != nil {
		return err
	}

	waitReason, err := p.readString()
	if err != nil {
		return err
	}

	// Skip remaining fields
	for i := 0; i < 4; i++ {
		if _, err := p.readVarint(); err != nil {
			return err
		}
	}

	if p.callbacks.OnGoroutine != nil {
		return p.callbacks.OnGoroutine(id, status, waitReason)
	}

	return nil
}

// readVarint reads a variable-length integer
func (p *StreamingParser) readVarint() (uint64, error) {
	v, err := binary.ReadUvarint(p.r)
	if err == nil {
		p.progress.Add(1) // Approximate
	}
	return v, err
}

// readString reads a length-prefixed string
func (p *StreamingParser) readString() (string, error) {
	length, err := p.readVarint()
	if err != nil {
		return "", err
	}
	if length > 1<<20 { // Sanity check: 1MB max string
		return "", fmt.Errorf("string too long: %d", length)
	}

	data := make([]byte, length)
	n, err := io.ReadFull(p.r, data)
	p.progress.Add(uint64(n))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// readBytes reads a length-prefixed byte slice
func (p *StreamingParser) readBytes() ([]byte, error) {
	length, err := p.readVarint()
	if err != nil {
		return nil, err
	}
	if length > 1<<30 { // Sanity check: 1GB max
		return nil, fmt.Errorf("byte slice too long: %d", length)
	}

	data := make([]byte, length)
	n, err := io.ReadFull(p.r, data)
	p.progress.Add(uint64(n))
	if err != nil {
		return nil, err
	}
	return data, nil
}
