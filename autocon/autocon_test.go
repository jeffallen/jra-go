// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package autocon

import (
	"fmt"
	"runtime"
	"testing"
)

func TestUnblock(t *testing.T) {
	results := make(chan string, 100)
	ch := NewAutoConsumer(3,
		func(ch chan interface {}) {
			for job := range ch {
				if closed(ch) {
					return
				}
				results <- job.(string)
			}
		})
	for i := 0; i < 10; i++ {
		ch <- fmt.Sprintf("item %d", i)
	}
	for i := 0; i < 10; i++ {
		runtime.Gosched()
		if _, ok := <- results; !ok {
			t.Error("missing result ", i)
		}
	}
	_, ok := <- results
	if ok {
		t.Error("too many results")
	}
}

func TestSlow(t *testing.T) {
	workers := make(chan bool, 100)
	sleepch := make(chan bool)
	ch := NewAutoConsumer(10,
		func(ch chan interface {}) {
			workers <- true
			for _ = range ch {
				if closed(ch) {
					return
				}
				_ = <- sleepch
			}
		})
	for i := 0; i < 10; i++ {
		ch <- true
	}
	for i := 0; i < 10; i++ {
		runtime.Gosched()
		if _, ok := <- workers; !ok {
			t.Error("missing worker ", i)
		}
	}
	_, ok := <- workers
	if ok {
		t.Error("too many workers")
	}
}

