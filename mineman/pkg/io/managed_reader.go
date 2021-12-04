package io

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"time"
)

var (
	ErrWaitForTextTimeout = errors.New("error wait for text has timeout without receive expected text")
)

type (
	ReadHook func(text string)

	ManagedReader struct {
		// only these
		rd io.ReadCloser

		// some internal state
		ctx     context.Context
		cancel  func()
		cmdChan chan manageCommand

		// managed inside loop
		hookIdOffset uint64
		readHooks    map[uint64]ReadHook
	}

	manageCommand        interface{}
	manageCommandAddHook struct {
		hook    ReadHook
		retChan chan uint64
	}
	manageCommandRemoveHook struct {
		id uint64
	}
)

func (r *ManagedReader) WaitForText(text string, timeout time.Duration) error {
	errChan := make(chan error, 1) // make a small buffer
	defer close(errChan)

	doneChan := make(chan struct{})
	defer close(doneChan)

	ctx, cancel := context.WithTimeout(r.ctx, timeout)
	defer cancel()

	sendResult := func(err error) {
		select {
		case <-doneChan:
			// do nothing
		default:
			errChan <- err
		}
	}

	go func() {
		<-ctx.Done()
		sendResult(ErrWaitForTextTimeout)
	}()

	id := r.AddOnReadHook(func(t string) {
		select {
		case <-ctx.Done():
			return
		default:
			if strings.Contains(t, text) {
				sendResult(nil)
			}
		}
	})
	defer r.RemoveOnReadHook(id)

	err := <-errChan
	return err
}

func (r *ManagedReader) AddOnReadHook(hook ReadHook) uint64 {
	retChan := make(chan uint64)
	defer close(retChan)

	cmd := new(manageCommandAddHook)
	cmd.hook = hook
	cmd.retChan = retChan

	r.cmdChan <- cmd

	return <-retChan
}

func (r *ManagedReader) RemoveOnReadHook(id uint64) {
	cmd := new(manageCommandRemoveHook)
	cmd.id = id
}

func (r *ManagedReader) Start(ctx context.Context) error {
	if r.ctx != nil {
		return errors.New("managed reader already started")
	}

	r.ctx, r.cancel = context.WithCancel(ctx)
	go r.start()
	return nil
}

func (r *ManagedReader) Close() error {
	if r.cancel == nil {
		return errors.New("managed reader not yet started")
	}

	// cancel context and reset state
	r.cancel()
	r.ctx = nil
	r.cancel = nil

	// close reader
	return r.rd.Close()
}

func (r *ManagedReader) addReadHook(cmd *manageCommandAddHook) {
	id := r.hookIdOffset + 1
	r.readHooks[id] = cmd.hook
	r.hookIdOffset = id
	cmd.retChan <- id
}

func (r *ManagedReader) removeReadHook(cmd *manageCommandRemoveHook) {
	delete(r.readHooks, cmd.id)
}

func (r *ManagedReader) start() {
	textChan := make(chan string, 10)
	defer close(textChan)
	go r.listen(textChan)

	for {
		select {
		case <-r.ctx.Done():
			return
		case cmd := <-r.cmdChan:
			switch cmd.(type) {
			case *manageCommandAddHook:
				r.addReadHook(cmd.(*manageCommandAddHook))
			case *manageCommandRemoveHook:
				r.removeReadHook(cmd.(*manageCommandRemoveHook))
			}
		case text := <-textChan:
			// call all hook
			for _, h := range r.readHooks {
				h(text)
			}
		}
	}
}

func (r *ManagedReader) listen(out chan string) {
	scanner := bufio.NewScanner(r.rd)
	defer close(out)
	for scanner.Scan() {
		select {
		case <-r.ctx.Done():
			return
		default:
			out <- scanner.Text()
		}
	}
}

func NewManagedReader(r io.ReadCloser) *ManagedReader {
	return &ManagedReader{
		rd:        r,
		cmdChan:   make(chan manageCommand, 10),
		readHooks: make(map[uint64]ReadHook),
	}
}
