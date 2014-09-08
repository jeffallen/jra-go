#!/bin/sh

PATH=$HOME/gopath/bin:$HOME/bin:$PATH
export PATH

stream-start
sunrise -dir /var/www/pics >> /tmp/sunrise.log 2>&1 &
