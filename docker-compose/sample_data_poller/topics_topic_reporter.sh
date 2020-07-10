#!/bin/bash

cd /kafka_2.13-2.5.0/bin
while true; do
  echo '{"topic_name": '"$TOPIC2LISTEN_TO"', "load": '"$LOAD2REPORT"'}' | ./kafka-console-producer.sh --bootstrap-server kafka-postman_kafka:9092 --topic "$TOPICS_TOPIC_JSON"
  echo $TOPIC2LISTEN_TO | ./kafka-console-producer.sh --bootstrap-server kafka-postman_kafka:9092 --topic "$TOPICS_TOPIC_TEXT"
  sleep $REPORT_INTERVAL
done