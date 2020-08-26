#!/bin/bash
server_farms=(tweety sylvester bugs_bunny daffy_duck porky_pig foghorn_leghorn tasmanian_devil yosemite_sam marvin_the_martian elmer_fudd pepe_le_pew speedy_gonzales granny petunia_pig bosko beaky_buzzard henry_hawk witch_hazel melissa_duck barnyard_dawg road_runner claude_cat hector_the_bulldog wile_e_coyote slow_poke_rodrigez charlie_dog)
server_categories=(mail web storage database firewall crm portal dns kafka redis kubernetes nginx)
databases=(mysql postgres mssql mongo elasticsearch hadoop)
metric_categories=(cpu.usage cpu.load memory.usage memory.total memory.free memory.swap network.bytes_in network.bytes_out network.packets_in network.packets_out)

if [ $# -eq 2 ] && [[ "$1" == "--wait" ]] && [[ "$2" =~ [0-9]+ ]]; then
  sleep $2
fi

cd /kafka_2.13-2.5.0/bin
while true; do
  server_farm_idx=$(( RANDOM % ${#server_farms[@]} ))
  server_category_idx=$(( RANDOM % ${#server_categories[@]} ))
  database_idx=$(( RANDOM % ${#databases[@]} ))
  metric_category_idx=$(( RANDOM % ${#metric_categories[@]} ))
  percent_value=$(( RANDOM % 100 ))
  load_value=$(( RANDOM % 24 ))
  metric_value=$RANDOM

  server_farm=${server_farms[$server_farm_idx]}
  server_category=${server_categories[$server_category_idx]}
  server_hostname="${server_category}$(( RANDOM % 10 ))"
  metric_category=${metric_categories[$metric_category_idx]}


  if [[ $server_category == "database" ]]; then
    database=${databases[$database_idx]}
    metric="$server_farm.$server_category.$database.$server_hostname.$metric_category"
  else
    metric="$server_farm.$server_category.$server_hostname.$metric_category"
  fi

  if [[ $metric_category == "cpu.usage" ]]; then
    value=$percent_value
  else
    if [[ $metric_category == "cpu.load" ]]; then
      value=$load_value
    else
      value=$metric_value
    fi
  fi

  echo "$metric $value "`date +%s`
  echo "$metric $value "`date +%s` | ./kafka-console-producer.sh --bootstrap-server kafka-postman_kafka:9092 --topic $RAW_DATA_TOPIC
  sleep 0.2s
done