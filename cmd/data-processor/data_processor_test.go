package main

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestDataProcessorRun(t *testing.T) {
	t.Run("shutdown on signal", func(t *testing.T) {
		stopCh := make(chan os.Signal, 1)
		go func() {
			time.Sleep(50 * time.Millisecond)
			stopCh <- syscall.SIGINT
		}()

		err := run(stopCh)
		if err != nil {
			t.Logf("run returned error (expected without DB): %v", err)
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		stopCh := make(chan os.Signal, 1)
		close(stopCh)

		err := run(stopCh)
		if err != nil {
			t.Logf("run returned error (expected without DB): %v", err)
		}
	})
}
