#!/bin/bash

#This file is the actual test. If it returns a non zero result the test will be considered failed.
  #cd "$FOLDER/docker-compose"
  #sudo docker stack deploy --compose-file docker-compose.yml kafka-postman

TESTS_COUNT=10
while true; do
  if grep 'sylvester' /tmp/data-poller-raw/sample_poller.log>/dev/null 2>&1 && ! grep 'sylvester' /tmp/data-poller-sylvester/sample_poller.log>/dev/null 2>&1; then
    echo "Failed to find Sylvester's logs in the poller logs but they exist in the raw poller."
    exit 1
  fi
  if grep 'bugs_bunny' /tmp/data-poller-raw/sample_poller.log>/dev/null 2>&1 && ! grep 'bugs_bunny' /tmp/data-poller-bugs_bunny/sample_poller.log>/dev/null 2>&1; then
    echo "Failed to find Bugs Bunny's logs in the poller logs but they exist in the raw poller."
    exit 1
  fi
  if grep 'porky_pig' /tmp/data-poller-raw/sample_poller.log>/dev/null 2>&1 && ! grep 'porky_pig' /tmp/data-poller-porky_pig/sample_poller.log>/dev/null 2>&1; then
    echo "Failed to find Porky Pig's logs in the poller logs but they exist in the raw poller."
    exit 1
  fi
  if grep 'sylvester' /tmp/data-poller-raw/sample_poller.log>/dev/null 2>&1 && grep 'bugs_bunny' /tmp/data-poller-raw/sample_poller.log>/dev/null 2>&1 && grep 'porky_pig' /tmp/data-poller-raw/sample_poller.log>/dev/null 2>&1; then
    TESTS_COUNT=$(( TESTS_COUNT - 1 ))
    if [[ $TESTS_COUNT -le 0 ]]; then
      exit 0
    else
      sleep 0.5
    fi
  fi
done
