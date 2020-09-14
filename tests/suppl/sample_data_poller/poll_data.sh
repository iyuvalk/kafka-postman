#!/bin/bash

if [ -z $TOPICS_TOPIC_JSON ] || [ -z $TOPICS_TOPIC_TEXT ]; then
  topics_topic_reporter.sh &
fi


cd /kafka_2.13-2.5.0/bin
[ ! -d /log ] && mkdir /log
[ -f /log/sample_poller.log ] && rm /log/sample_poller.log
while true; do
  ./kafka-console-consumer.sh --bootstrap-server kafka-postman_kafka:9092 --topic $TOPIC2LISTEN_TO | tee -a /log/sample_poller.log
  sleep 1s
done