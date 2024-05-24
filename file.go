package shmspmc

import (
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/unix"
)

type File[T comparable] struct {
	zero     T
	path     string
	osFile   *os.File
	isWriter bool
	typeSize int64
	mem      []byte

	fileSize  int64
	next      int64
	nextPunch int64
}

const (
	extendSize      = 2 * (1 << 20)
	dataBeginOffset = 4 * (1 << 10) // must be 4k-aligned to allow madvise
)

func New[T comparable](name string, isWriter bool) (*File[T], error) {
	path := filepath.Join("/dev/shm", name)

	// open file
	flags := os.O_RDONLY
	if isWriter {
		flags = os.O_CREATE | os.O_RDWR
	}
	osFile, err := os.OpenFile(path, flags, 0644)
	if err != nil {
		return nil, err
	}

	// map
	prot := unix.PROT_READ
	if isWriter {
		prot |= unix.PROT_WRITE
	}
	mem, err := unix.Mmap(
		int(osFile.Fd()),
		0,
		1<<40,
		prot,
		unix.MAP_SHARED,
	)
	if err != nil {
		return nil, err
	}

	var zero T
	return &File[T]{
		osFile:    osFile,
		isWriter:  isWriter,
		typeSize:  int64(unsafe.Sizeof(zero)),
		mem:       mem,
		next:      dataBeginOffset,
		nextPunch: dataBeginOffset,
	}, nil
}

func (p *File[T]) extend() error {
	if p.next+p.typeSize < p.fileSize {
		return nil
	}

	// truncate
	p.fileSize += extendSize
	if p.fileSize > int64(len(p.mem)) {
		// reset
		p.fileSize = extendSize
		p.next = dataBeginOffset
		p.nextPunch = dataBeginOffset
	}
	if err := p.osFile.Truncate(int64(p.fileSize)); err != nil {
		return err
	}

	// punch hole
	if p.fileSize-p.nextPunch > extendSize*2 {
		if err := unix.Madvise(
			unsafe.Slice(
				(*byte)(unsafe.Pointer(&p.mem[p.nextPunch])),
				extendSize,
			),
			unix.MADV_REMOVE,
		); err != nil {
			return err
		}
		p.nextPunch += extendSize
	}

	return nil
}

func (p *File[T]) Write(value T) error {
	if err := p.extend(); err != nil {
		return err
	}
	copy(
		unsafe.Slice((*byte)(unsafe.Pointer(&p.mem[p.next])), p.typeSize),
		unsafe.Slice((*byte)(unsafe.Pointer(&value)), p.typeSize),
	)
	atomic.StoreInt64(
		(*int64)(unsafe.Pointer(&p.mem[0])),
		p.next,
	)
	p.next += p.typeSize
	return nil
}

func (p *File[T]) Read() T {
	for {
		index := atomic.LoadInt64(
			(*int64)(unsafe.Pointer(&p.mem[0])),
		)
		value := *(*T)(unsafe.Pointer(&p.mem[index]))
		// MADV_REMOVE may cause read value to be zero, read again
		if value != p.zero {
			return value
		}
	}
}

func (p *File[T]) Close() error {
	if p.isWriter {
		return errors.Join(
			p.osFile.Close(),
			os.Remove(p.path),
		)
	}
	return p.osFile.Close()
}
