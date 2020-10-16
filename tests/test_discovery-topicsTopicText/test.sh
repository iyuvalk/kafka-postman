#!/bin/bash

#This file is the actual test. If it returns a non zero result the test will be considered failed.
  #cd "$FOLDER/docker-compose"
  #sudo docker stack deploy --compose-file docker-compose.yml kafka-postman

#TODO: For some reason it seems that sometimes some of the earlier entries in the raw log are missing from the specific (e.g Sylvester)'s log - WEIRD!

TESTS_COUNT=50
while true; do
  if grep 'sylvester' /tmp/data-poller-raw/sample_poller.log>/dev/null 2>&1 && ! grep 'sylvester' /tmp/data-poller-sylvester/sample_poller.log>/dev/null 2>&1; then
    echo "Failed to find Sylvester's logs in the poller logs but they exist in the raw poller."
    exit 1
  fi
  if grep 'bugs_bunny' /tmp/data-poller-raw/sample_poller.log>/dev/null 2>&1 && ! grep 'bugs_bunny' /tmp/data-poller-bugs_bunny/sample_poller.log>/dev/null 2>&1; then
    echo "Failed to find Bugs Bunny's logs in the poller logs but they exist in the raw poller."
    exit 1
  fi
  if grep 'daffy_duck' /tmp/data-poller-raw/sample_poller.log>/dev/null 2>&1 && grep 'daffy_duck' /tmp/data-poller-daffy_duck/sample_poller.log>/dev/null 2>&1; then
    echo "Found Daffy Duck's logs in the poller logs but they should ONLY exist in the raw poller. (the daffy_duck poller doesn't report to the topics topic and should not be discovered)"
    exit 1
  fi
  if grep 'sylvester' /tmp/data-poller-raw/sample_poller.log>/dev/null 2>&1 && grep 'bugs_bunny' /tmp/data-poller-raw/sample_poller.log>/dev/null 2>&1 && grep 'daffy_duck' /tmp/data-poller-raw/sample_poller.log>/dev/null 2>&1; then
    TESTS_COUNT=$(( TESTS_COUNT - 1 ))
    if [[ $TESTS_COUNT -le 0 ]]; then
      exit 0
    else
      sleep 0.5
    fi
  fi
done
