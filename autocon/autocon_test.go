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
		func(ch chan interface{}) {
			for {
				job, ok := <-ch
				if !ok {
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
		select {
		case <-results:
		default:
			t.Error("missing result ", i)
		}
	}
	select {
	case _ = <-results:
		t.Error("too many results")
	default:
		// ok, we expected this branch
	}
}

func TestSlow(t *testing.T) {
	workers := make(chan bool, 100)
	sleepch := make(chan bool)
	ch := NewAutoConsumer(10,
		func(ch chan interface{}) {
			workers <- true
			for {
				_, ok := <-ch
				if !ok {
					return
				}
				_ = <-sleepch
			}
		})
	for i := 0; i < 10; i++ {
		ch <- true
	}
	for i := 0; i < 10; i++ {
		runtime.Gosched()
		select {
		case _ = <-workers:
			// ok
		default:
			t.Error("missing worker ", i)
		}
	}
	select {
	case _ = <-workers:
		t.Error("too many workers")
	default:
		// ok, should be no more results
	}
}

// ex: ts=2
