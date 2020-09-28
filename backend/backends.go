// Copyright 2016 Eleme. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package backend

import (
	"bytes"
	"io"
	"log"
	"sync"
	"time"
)

const (
	WriteQueue = 16
)

type Backend struct {
	*HttpBackend
	Interval        int
	RewriteInterval int
	MaxRowLimit     int32

	fileBackend     *FileBackend
	running         bool
	ticker          *time.Ticker
	chWrite         chan []byte
	buffer          *bytes.Buffer
	chTimer         <-chan time.Time
	writeCounter    int32
	rewriterRunning bool
	waitGroup       sync.WaitGroup
}

// maybe ch_timer is not the best way.
func NewBackend(cfg *BackendConfig, name string) (bs *Backend, err error) {
	bs = &Backend{
		HttpBackend: NewHttpBackend(cfg),
		// FIXME: path...
		Interval:        cfg.Interval,
		RewriteInterval: cfg.RewriteInterval,
		running:         true,
		ticker:          time.NewTicker(time.Millisecond * time.Duration(cfg.RewriteInterval)),
		chWrite:         make(chan []byte, WriteQueue),

		rewriterRunning: false,
		MaxRowLimit:     int32(cfg.MaxRowLimit),
	}
	bs.fileBackend, err = NewFileBackend(name)
	if err != nil {
		_ = bs.Close()
		return nil, err
	}
	go bs.worker()
	return bs, nil
}

func (bs *Backend) worker() {
	for bs.running {
		select {
		case p, ok := <-bs.chWrite:
			if !ok {
				// closed
				bs.Flush()
				bs.waitGroup.Wait()
				_ = bs.HttpBackend.Close()
				bs.fileBackend.Close()
				return
			}
			bs.WriteBuffer(p)

		case <-bs.chTimer:
			bs.Flush()
			if !bs.running {
				bs.waitGroup.Wait()
				_ = bs.HttpBackend.Close()
				bs.fileBackend.Close()
				return
			}

		case <-bs.ticker.C:
			bs.Idle()
		}
	}
}

func (bs *Backend) Write(p []byte) (err error) {
	if !bs.running {
		return io.ErrClosedPipe
	}

	bs.chWrite <- p
	return
}

func (bs *Backend) Close() (err error) {
	bs.running = false
	close(bs.chWrite)
	return
}

func (bs *Backend) WriteBuffer(p []byte) {
	bs.writeCounter++

	if bs.buffer == nil {
		bs.buffer = &bytes.Buffer{}
	}

	n, err := bs.buffer.Write(p)
	if err != nil {
		log.Printf("error: %s\n", err)
		return
	}
	if n != len(p) {
		err = io.ErrShortWrite
		log.Printf("error: %s\n", err)
		return
	}

	if p[len(p)-1] != '\n' {
		_, err = bs.buffer.Write([]byte{'\n'})
		if err != nil {
			log.Printf("error: %s\n", err)
			return
		}
	}

	switch {
	case bs.writeCounter >= bs.MaxRowLimit:
		bs.Flush()
	case bs.chTimer == nil:
		bs.chTimer = time.After(
			time.Millisecond * time.Duration(bs.Interval))
	}

	return
}

func (bs *Backend) Flush() {
	if bs.buffer == nil {
		return
	}

	p := bs.buffer.Bytes()
	bs.buffer = nil
	bs.chTimer = nil
	bs.writeCounter = 0

	if len(p) == 0 {
		return
	}

	// TODO: limitation
	bs.waitGroup.Add(1)
	go func() {
		defer bs.waitGroup.Done()
		var buf bytes.Buffer
		err := Compress(&buf, p)
		if err != nil {
			log.Printf("write file error: %s\n", err)
			return
		}

		p = buf.Bytes()

		// maybe blocked here, run in another goroutine
		if bs.HttpBackend.IsActive() {
			err = bs.HttpBackend.WriteCompressed(p)
			switch err {
			case nil:
				return
			case ErrBadRequest:
				log.Printf("bad request, drop all data.")
				return
			case ErrNotFound:
				log.Printf("bad backend, drop all data.")
				return
			default:
				log.Printf("unknown error %s, maybe overloaded.", err)
			}
			log.Printf("write http error: %s\n", err)
		}

		err = bs.fileBackend.Write(p)
		if err != nil {
			log.Printf("write file error: %s\n", err)
		}
		// don't try to run rewrite loop directly.
		// that need a lock.
	}()

	return
}

func (bs *Backend) Idle() {
	if !bs.rewriterRunning && bs.fileBackend.IsData() {
		bs.rewriterRunning = true
		go bs.RewriteLoop()
	}

	// TODO: report counter
}

func (bs *Backend) RewriteLoop() {
	for bs.fileBackend.IsData() {
		if !bs.running {
			return
		}
		if !bs.HttpBackend.IsActive() {
			time.Sleep(time.Millisecond * time.Duration(bs.RewriteInterval))
			continue
		}
		err := bs.Rewrite()
		if err != nil {
			time.Sleep(time.Millisecond * time.Duration(bs.RewriteInterval))
			continue
		}
	}
	bs.rewriterRunning = false
}

func (bs *Backend) Rewrite() (err error) {
	p, err := bs.fileBackend.Read()
	if err != nil {
		return
	}
	if p == nil { // why?
		return
	}

	err = bs.HttpBackend.WriteCompressed(p)

	switch err {
	case nil:
	case ErrBadRequest:
		log.Printf("bad request, drop all data.")
		err = nil
	case ErrNotFound:
		log.Printf("bad backend, drop all data.")
		err = nil
	default:
		log.Printf("unknown error %s, maybe overloaded.", err)

		err = bs.fileBackend.RollbackMeta()
		if err != nil {
			log.Printf("rollback meta error: %s\n", err)
		}
		return
	}

	err = bs.fileBackend.UpdateMeta()
	if err != nil {
		log.Printf("update meta error: %s\n", err)
		return
	}
	return
}
