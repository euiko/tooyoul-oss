package io

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/euiko/tooyoul/mineman/pkg/log"
)

var (
	ErrWaitForTextTimeout = errors.New("error wait for text has timeout without receive expected text")
)

type (
	ReadHook func(text string)

	ManagedReader struct {
		// only these
		rds []io.ReadCloser

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
	errChan := make(chan error) // make a small buffer
	defer close(errChan)

	// use context also to synchronize whether result already send
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sendResult := func(err error) {
		select {
		case <-ctx.Done():
			// do nothing
		default:
			// early cancel, for one of result send
			cancel()
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
				log.Debug("waiting text completed, text found")
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
	go r.start(r.ctx)
	return nil
}

func (r *ManagedReader) StartAndWait(ctx context.Context, text string, timeout time.Duration) error {
	if r.ctx != nil {
		return errors.New("managed reader already started")
	}

	errChan := make(chan error, 1)
	defer close(errChan)

	r.ctx, r.cancel = context.WithCancel(ctx)
	go func() {
		errChan <- r.WaitForText(text, timeout)
	}()
	go r.start(r.ctx)
	err := <-errChan
	return err
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
	for _, rd := range r.rds {
		if err := rd.Close(); err != nil {
			return err
		}
	}
	return nil
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

func (r *ManagedReader) start(ctx context.Context) {
	textChan := make(chan string, len(r.rds)*10)
	defer close(textChan)

	// listen all reader
	for _, rd := range r.rds {
		go r.listen(rd, textChan)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case cmd := <-r.cmdChan:
			switch cmd := cmd.(type) {
			case *manageCommandAddHook:
				r.addReadHook(cmd)
			case *manageCommandRemoveHook:
				r.removeReadHook(cmd)
			}
		case text := <-textChan:
			log.Debug("teamredminer stdout : %s", log.WithValues(text))
			// call all hook
			for _, h := range r.readHooks {
				h(text)
			}
		}
	}
}

func (r *ManagedReader) listen(rd io.Reader, out chan string) {
	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		select {
		case <-r.ctx.Done():
			return
		default:
			out <- scanner.Text()
		}
	}
}

func NewManagedReader(rds ...io.ReadCloser) *ManagedReader {
	return &ManagedReader{
		rds:       rds,
		cmdChan:   make(chan manageCommand, 10),
		readHooks: make(map[uint64]ReadHook),
	}
}
