#!/bin/bash

#TODO: It seems that maybe there's a timing issue here... Sometimes it seems that the kafka server doesn't have all the topics configured (at least not when kafka postman pulls the list)


#This file will run at the beginning of this test
[ ! -d /tmp/data-poller-raw ] && mkdir /tmp/data-poller-raw
[ ! -d /tmp/data-poller-sylvester ] && mkdir /tmp/data-poller-sylvester
[ ! -d /tmp/data-poller-bugs_bunny ] && mkdir /tmp/data-poller-bugs_bunny
[ ! -d /tmp/data-poller-porky_pig ] && mkdir /tmp/data-poller-porky_pig

IS_KAFKA_POSTMAN_LOADED=0
TS_STARTED=$(date +%s)
WAITS=600
while true; do
  CONTINUE=0
  if [[ $IS_KAFKA_POSTMAN_LOADED -eq 0 ]]; then
    docker stack deploy --compose-file "$1/docker-compose/docker-compose.yml" kafka-postman
    sleep 1
    IS_KAFKA_POSTMAN_LOADED=$(docker ps | grep " kafka-postman:latest " | wc -l)
    CONTINUE=1
  fi
  TS_CURRENT=$(date +%s)
  WAITS=$(( WAITS - 1 ))
  if [[ $WAITS -lt 1 ]]; then
    echo "+Waited long enough ($(expr $TS_CURRENT - $TS_STARTED) seconds so far). Hoping for the best. Leaving..."
    exit 9
  fi
  SYLVESTER_COUNT=$(grep 'sylvester' /tmp/data-poller-raw/sample_poller.log | wc -l)
  BUGS_BUNNY_COUNT=$(grep 'bugs_bunny' /tmp/data-poller-raw/sample_poller.log | wc -l)
  PORKY_PIG_COUNT=$(grep 'porky_pig' /tmp/data-poller-raw/sample_poller.log | wc -l)
  if [[ $CONTINUE -eq 0 ]] && [[ $IS_KAFKA_POSTMAN_LOADED -gt 0 ]] && [[ $SYLVESTER_COUNT -gt 50 ]] && [[ $BUGS_BUNNY_COUNT -gt 50 ]] && [[ $PORKY_PIG_COUNT -gt 50 ]]; then
    echo "+It seems that the services are running (the relevant logs were found in the raw poller logs). Waiting 15secs and starting the test..."
    sleep 15s
    exit 0
  else
    echo "+Waiting for kafka-postman to load... ($(expr $TS_CURRENT - $TS_STARTED) milliseconds so far)"
    sleep 1
  fi
done
