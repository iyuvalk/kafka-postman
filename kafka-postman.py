#!/usr/bin/python2.7
##########################################################################
# Kafka-Postman Topics Distributor                                       #
# --------------------------------                                       #
#                                                                        #
# This service is part of the Kafka-Postman package. This micro-service  #
# is responsible for pulling data from a kafka topic and then pushing it #
# to other topics based on some logic. This service can be used to       #
# evenly distribute data that arrives in one topic to multiple topics    #
# while keeping a certain affinity.                                      #
#                                                                        #
##########################################################################


import sys
import os
from datetime import datetime
import threading
from bottle import Bottle
import syslog
import kafka
from multiprocessing import Value
import re
import hashlib
import json
import base64
import time
import uuid
import redis


DEBUG = False


class KafkaPostman:

    def __init__(self):
        self._time_loaded = time.time()
        self._app = Bottle()
        self._route()
        self._instance_id = str(uuid.uuid4())
        self.EXIT_ALL_THREADS_FLAG = False
        self._metrics_pulled = Value('i', 0)
        self._metrics_pushed = Value('i', 0)
        self._topics_discovered = Value('i', 0)
        # A strategy for discovering new destination topics. Can be one of: REGEX, TOPICS_TOPIC (by using a topic that all consumers periodically send the topic they're listening on to make sure the data is sent to topics that are read by other processes),
        self._topics_discovery_method = os.environ["KafkaPostman_TOPICS_DISCOVERY_METHOD"]
        # If KafkaPostman_TOPICS_DISCOVERY_METHOD is set to REGEX use this to set a regex that we'll attempt to match each topic on kafka periodically and add those who matched it to the list of discovered topics
        self._topics_discovery_regex = os.environ["KafkaPostman_TOPICS_DISCOVERY_REGEX"]
        # A regex that will be used to extract a portion of the data that arrived in the source topic. If KafkaPostman_TOPICS_DISTRIBUTION_STRATEGY is set to REGEX the extracted string will be used as a destination topic, otherwise it will be matched against previously seen data in the redis cache and if matched the same topic that was used for that data will be used again for the current data.
        self._topics_match_regex = os.environ["KafkaPostman_TOPICS_SELECTION_REGEX"]
        # A strategy that will be used to select a destination topic once a new type of data (or no regex was defined). Can be one of: ROUND_ROBIN, RANDOM, REGEX (Will use the selected part of the data matched by the first group selection in the regex as destination topic)
        self._topics_distribution_strategy = os.environ["KafkaPostman_TOPICS_DISTRIBUTION_STRATEGY"]
        self._logging_format = os.environ["KafkaPostman_LOGGING_FORMAT"]
        self._kafka_consumer_client_id = os.environ["KafkaPostman_KAFKA_CONSUMER_CLIENT_ID"].replace("{{#instance_id}}", self._instance_id).replace("{{#time_started}}", str(int(time.time())))
        self._kafka_consumer_server = os.environ["KafkaPostman_KAFKA_CONSUMER_SERVER"]
        self._kafka_producer_client_id = os.environ["KafkaPostman_KAFKA_PRODUCER_CLIENT_ID"].replace("{{#instance_id}}", self._instance_id).replace("{{#time_started}}", str(int(time.time())))
        self._kafka_producer_server = os.environ["KafkaPostman_KAFKA_PRODUCER_SERVER"]
        self._topics_list_topic = os.environ["KafkaPostman_TOPICS_KAFKA_TOPIC"]
        self._source_topic = os.environ["KafkaPostman_SOURCE_TOPIC"]
        self._kafka_producer = None
        while self._kafka_producer is None:
            try:
                self._kafka_producer = kafka.producer.KafkaProducer(
                    bootstrap_servers=self._kafka_producer_server,
                    client_id=self._kafka_producer_client_id
                )
            except Exception as ex:
                syslog.syslog(self._logging_format % (datetime.now().isoformat(), "kafka-postman", "__init__", "WARN", str(None), str(None), str(ex), type(ex).__name__, "Waiting (indefinitely in 10 sec intervals) for the Producer Kafka service to become available... (self._kafka_producer_server=" + self._kafka_producer_server + ", self._kafka_producer_client_id=" + self._kafka_producer_client_id + ")"))
                time.sleep(10)

    def _file_as_bytes(self, filename):
        with open(filename, 'rb') as file:
            return file.read()

    def _ping(self):
        return {
            "response": "PONG",
            "file": __file__,
            "hash": hashlib.md5(self._file_as_bytes(__file__)).hexdigest(),
            "uptime": time.time() - self._time_loaded,
            "instance_id": self._instance_id,
            "service_specific_info": {
                "metrics_pulled": self._metrics_pulled.value,
                "topics_discovered": self._topics_discovered.value,
                "metrics_pushed": self._metrics_pushed.value
            }
        }

    def _route(self):
        self._app.route('/ping', method="GET", callback=self._ping)

    def distribute_posts(self):
        kafka_consumer = None
        while kafka_consumer is None:
            try:
                kafka_consumer = kafka.KafkaConsumer(self._source_topic,
                                                     bootstrap_servers=self._kafka_consumer_server,
                                                     client_id=self._kafka_consumer_client_id)
                syslog.syslog(self._logging_format % (datetime.now().isoformat(), "kafka-postman", "distribute_posts", "INFO", str(None), str(None), str(None), str(None), "Loaded a Kafka consumer successfully. (self._metrics_kafka_topic=" + str(self._raw_metrics_kafka_topic) + ";bootstrap_servers=" + str(self._kafka_consumer_server) + ";client_id=" + str(self._kafka_consumer_client_id) + ")"))
            except Exception as ex:
                syslog.syslog(self._logging_format % (datetime.now().isoformat(), "kafka-postman", "distribute_posts", "INFO", str(None), str(ex), str(type(ex).__name__), str(None), "Waiting on a dedicated thread for the Kafka server to be available... Going to sleep for 10 seconds"))
                time.sleep(10)

    def discover_destination_topics_by_topics_topic(self):
        kafka_consumer = None
        while kafka_consumer is None:
            try:
                kafka_consumer = kafka.KafkaConsumer(self._source_topic,
                                                     bootstrap_servers=self._kafka_consumer_server,
                                                     client_id=self._kafka_consumer_client_id)
                syslog.syslog(self._logging_format % (datetime.now().isoformat(), "kafka-postman", "discover_destination_topics", "INFO", str(None), str(None), str(None), str(None), "Loaded a Kafka consumer successfully. (self._metrics_kafka_topic=" + str(self._raw_metrics_kafka_topic) + ";bootstrap_servers=" + str(self._kafka_consumer_server) + ";client_id=" + str(self._kafka_consumer_client_id) + ")"))
            except Exception as ex:
                syslog.syslog(self._logging_format % (datetime.now().isoformat(), "kafka-postman", "discover_destination_topics", "INFO", str(None), str(ex), str(type(ex).__name__), str(None), "Waiting on a dedicated thread for the Kafka server to be available... Going to sleep for 10 seconds"))
                time.sleep(10)

    def discover_destination_topics(self):
        pass

    def run(self, ping_listening_host="127.0.0.1", ping_listening_port=3001, save_interval=5, models_save_base_path=None, models_params_base_path=None, anomaly_likelihood_detectors_save_base_path=None):
        self._ping_listening_host = ping_listening_host
        self._ping_listening_port = ping_listening_port
        self._topics_distributor_thread = threading.Thread(target=self.distribute_posts)
        self._topics_discoverer_thread = threading.Thread(target=self.discover_destination_topics)
        syslog.syslog(self._logging_format % (datetime.now().isoformat(), "kafka-postman", "run", "INFO", str(None), str(None), str(None), str(None), "Launching the analyzer thread (save_interval=" + str(save_interval) + ";models_save_base_path=" + str(models_save_base_path) + ";models_params_base_path=" + str(models_params_base_path)+ ";anomaly_likelihood_detectors_save_base_path=" + str(anomaly_likelihood_detectors_save_base_path) + ")"))
        self._topics_distributor_thread.start()
        self._topics_discoverer_thread.start()
        self._app.run(host=self._ping_listening_host, port=self._ping_listening_port, server="gunicorn", workers=2)


script_path = os.path.dirname(os.path.realpath(__file__))
ping_listening_host = os.environ["KafkaPostman_PING_LISTEN_HOST"]
ping_listening_port = int(os.environ["KafkaPostman_PING_LISTEN_PORT"])
KafkaPostman().run(
    ping_listening_host=ping_listening_host,
    ping_listening_port=ping_listening_port,
)