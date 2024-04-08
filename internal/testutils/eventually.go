package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TryTilCountIs(t *testing.T, condition func() bool, cnt uint64, tick time.Duration, msgAndArgs ...interface{}) {
	ch := make(chan bool, 1)
	done := make(chan bool, 1)

	count := uint64(0)
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for tick := ticker.C; ; {
		select {
		case <-tick:
			tick = nil
			go func() { ch <- condition() }()
			if count++; count == cnt {
				done <- true
			}
		case v := <-ch:
			if v {
				return
			}
			tick = ticker.C
		case _ = <-done:
			assert.Fail(t, "Condition never satisfied", msgAndArgs...)
			t.FailNow()
		}
	}
}
