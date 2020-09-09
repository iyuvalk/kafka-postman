#!/bin/bash

cd /kafka_2.13-2.5.0/bin
while true; do
  if [[ "$REPORT_JSON" == "true" ]]; then
    echo '{"topic_name": '"$TOPIC2LISTEN_TO"', "load": '"$LOAD2REPORT"'}' | ./kafka-console-producer.sh --bootstrap-server kafka-postman_kafka:9092 --topic "$TOPICS_TOPIC"
  else
    echo $TOPIC2LISTEN_TO | ./kafka-console-producer.sh --bootstrap-server kafka-postman_kafka:9092 --topic "$TOPICS_TOPIC"
  fi
  sleep $REPORT_INTERVAL
done
