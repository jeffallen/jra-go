#	Copyright 2009 Jeff R. Allen. All rights reserved.
#	Use of this source code is governed by a BSD-style
#	license that can be found in the LICENSE file of the Go
#	distribution.

DIRS=server client

all: default

default:
	for i in $(DIRS); do $(MAKE) -C $$i ; done

install:
	for i in $(DIRS); do $(MAKE) -C $$i install; done

clean:
	for i in $(DIRS); do $(MAKE) -C $$i clean; done

