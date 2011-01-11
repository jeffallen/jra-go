# Copyright 2011 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

package autocon

type autoConsumer struct {
	worker	func(chan interface {})
	consumers []chan interface {}
	ch chan interface {}
}

func NewAutoConsumer(maxConsumers int, worker func(chan interface {})) chan interface {} {
	ac := new(autoConsumer)
	ac.worker = worker
	ac.consumers = make([]chan interface {}, maxConsumers)
	ac.ch = make(chan interface {})
	go ac.run()
	return ac.ch
}

func (ac* autoConsumer)run() {
	for job := range ac.ch {
		var ok bool
		if closed(ac.ch) {
			break
		}
	 assign:
		for i, c := range ac.consumers {
			if c != nil {
				if ok = c <- job; ok {
					break assign
				}
			}
			ac.consumers[i] = make(chan interface {})
			go ac.worker(ac.consumers[i])
			ac.consumers[i] <- job
			ok = true
			break assign
		}
		if ! ok {
			// we didn't manage to find/create a worker to handle the job, so panic
			panic("no worker available for job")
		}
	}

	// close all the consumers' input channels to make them quit
	for _, c := range ac.consumers {
		close(c)
	}
}
