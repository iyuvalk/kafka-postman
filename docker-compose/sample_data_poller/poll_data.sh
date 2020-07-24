#!/bin/bash

cd /kafka_2.13-2.5.0/bin
while true; do
  ./kafka-console-consumer.sh --bootstrap-server kafka-postman_kafka:9092 --topic $TOPIC2LISTEN_TO
  sleep 1s
done