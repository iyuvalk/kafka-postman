#!/bin/bash

#This file will run after the test ends, regardless of the test result (which will be given as an argument)

docker stack rm kafka-postman
if [ -d /tmp/data-poller-sylvester ]; then
  rm -f /tmp/data-poller-sylvester/*
  rmdir /tmp/data-poller-sylvester
fi
if [ -d /tmp/data-poller-bugs_bunny ]; then
  rm -f /tmp/data-poller-bugs_bunny/*
  rmdir /tmp/data-poller-bugs_bunny
fi
if [ -d /tmp/data-poller-daffy_duck ]; then
  rm -f /tmp/data-poller-daffy_duck/*
  rmdir /tmp/data-poller-daffy_duck
fi
if [ -d /tmp/data-poller-raw ]; then
  rm -f /tmp/data-poller-raw/*
  rmdir /tmp/data-poller-raw
fi

while docker ps | grep kafka-postman_; do
  echo "+Waiting for all the stack containers to shutdown..."
  sleep 1
  docker stack rm kafka-postman
done
while docker network ls | grep kafka-postman; do
  echo "+Waiting for the stack network to be removed..."
  sleep 1
  docker stack rm kafka-postman
done