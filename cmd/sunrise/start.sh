#!/bin/sh

stream-start
sunrise -dir /var/www/pics >> /tmp/sunrise.log 2>&1 &
