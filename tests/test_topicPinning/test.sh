#!/bin/bash

#This file is the actual test. If it returns a non zero result the test will be considered failed.
  #cd "$FOLDER/docker-compose"
  #sudo docker stack deploy --compose-file docker-compose.yml kafka-postman

TESTS_COUNT=50
while true; do
  for datacenter in sylvester bugs_bunny daffy_duck; do
    pollers_count=$(grep -r "$datacenter" /tmp/data-poller-* | awk -F: '{print $1}' | uniq | grep -v "/tmp/data-poller-raw/sample_poller.log" | wc -l)
    if [ "$pollers_count" -gt 1 ]; then
      echo "$datacenter data was found in more than one file"
      exit 1
    fi
    if [ "$pollers_count" -lt 1 ]; then
      echo "$datacenter data was NOT found"
      exit 1
    fi
  done
  TESTS_COUNT=$(( TESTS_COUNT - 1 ))
  if [[ $TESTS_COUNT -le 0 ]]; then
    exit 0
  else
    sleep 0.5
  fi
done
