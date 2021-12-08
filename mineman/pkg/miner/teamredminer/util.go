package teamredminer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

func waitForRead(reader io.Reader, str string, timeout time.Duration) error {
	if reader == nil {
		return errors.New("waitForRead's reader is not valid")
	}

	textChan := make(chan string, 10)
	defer close(textChan)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	scanner := bufio.NewScanner(reader)
	go func() {
		for scanner.Scan() {
			textChan <- scanner.Text()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout reached and doesn't receive any '%s' text", str)
		case text := <-textChan:
			if strings.Contains(text, str) {
				return nil
			}
		}
	}
}
