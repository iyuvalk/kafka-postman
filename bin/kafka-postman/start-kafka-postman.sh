#!/bin/bash

if [ $# -eq 2 ] && [[ "$1" == "--wait" ]] && [[ "$2" =~ [0-9]+ ]]; then
  sleep $2
fi
./kafka-postman &
while true; do
  sleep 10s
done
