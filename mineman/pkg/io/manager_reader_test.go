package io

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

func TestManagedReader(t *testing.T) {

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	managedReader := NewManagedReader(r)
	managedReader.Start(context.Background())
	defer managedReader.Close()

	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {
		time.Sleep(2 * time.Second)

		w.WriteString("halo dunia\n")
		w.WriteString("apa kabarmu\n")

		time.Sleep(1 * time.Second)
		w.WriteString("baik baik saja kah\n")
		wg.Done()
	}()

	if err := managedReader.WaitForText("kabar", 3*time.Second); err != nil {
		t.Fatal(err)
	}

	if err := managedReader.WaitForText("kan", 1*time.Second); err == nil {
		t.Fatal("expect waiting for kan error")
	}

	wg.Wait()
}
