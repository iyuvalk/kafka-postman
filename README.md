# kafka-postman
Your kafka topics organizer

## The problem we came to solve
When I wrote the Pensu service for metrics anomaly detection and prediction, I quickly realized that since that this service reads the metrics from a Kafka topic and outputs its predictions and anomaly scores to another Kafka topic, it will need some way to scale horizontally. Since that just adding more instances of the service cannot guarantee that once a service starts to get a specific metric it will continue to receive all the datapoints for that metric from that moment on, I decided to write this service to help distribute data from one source topic to several destination topics based on some configurable logic.

## The basic algorithm
This service is comprised of two steps that are running simultaneously on different threads:
1. Topics discovery
1.1. Discovery method is set to REGEX:
1.1.1. Every n seconds (configurable by an environment variable) a list of all available topics will be collected from the destination Kafka broker.
1.1.1. Every topic name that matches a configured regex (configurable by an environment variable) will be considered as discovered and will be stored in Redis by using the following key format "$KafkaPostman_REDIS_TOPICS_PREFIX-$TopicName" without an expiry date for the topics selection phase.
1.1. Discovery method is set to TOPICS_TOPIC:
1.1.1. The discovery thread will launch a kafka consumer that will be configured to consume from the topic specified by a relevant environment variable.
1.1.1. For every topic consumed, the thread will store it in Redis by using the following key format "$KafkaPostman_REDIS_TOPICS_PREFIX-$TopicName" with an expiry time set by $KafkaPostman_REDIS_TOPICS_SLIDING_EXPIRY_MS (if already exist it will reset its expiry time). This mode allows the service to detect dead consumers and start sending the data that was previously consumed by them to another consumer.
1.1 
1. Topics selection - Runs whenever a message is consumed from the source topic
1.1 Whenever a message is consumed from the source topic, if the topics regex selection mode is turned on (configurable by an environment variable) a configured regex will be used to extract part of the message, this part will be used to check against Redis to find a topic for such messages, if such topic name is found, the message will be pushed to that topic. If no topic was found on Redis for that message part, the service will choose a topic from the list of discovered topics (the topics that were saved to redis using the specified prefix) randomly or by using round-robin logic (configurable by an environment variable) and the message part will be sent with the selected topic to Redis to allow for similar messages (by the regex) to be sent to the same topic. If the topics regex selection mode is turned off, the service will choose a topic from the list of discovered topics (the topics that were saved to redis using the specified prefix) randomly or by using round-robin logic (configurable by an environment variable) and the message will be sent to the selected topic.

### This service is controlled by the folowing environment variables:
```
KafkaPostman_TOPICS_DISCOVERY_METHOD - A strategy for discovering new destination topics. Can be one of: REGEX, TOPICS_TOPIC (by using a topic that all consumers periodically send the topic they're listening on to make sure the data is sent to topics that are read by other processes)

KafkaPostman_TOPICS_DISCOVERY_REGEX - If KafkaPostman_TOPICS_DISCOVERY_METHOD is set to REGEX use this to set a regex that we'll attempt to match each topic on kafka periodically and add those who matched it to the list of discovered topics

KafkaPostman_TOPICS_SELECTION_REGEX - A regex that will be used to extract a portion of the data that arrived in the source topic. If KafkaPostman_TOPICS_DISTRIBUTION_STRATEGY is set to REGEX the extracted string will be used as a destination topic, otherwise it will be matched against previously seen data in the redis cache and if matched the same topic that was used for that data will be used again for the current data.

KafkaPostman_TOPICS_DISTRIBUTION_STRATEGY - A strategy that will be used to select a destination topic once a new type of data (or no regex was defined). Can be one of: ROUND_ROBIN, RANDOM, REGEX (Will use the selected part of the data matched by the first group selection in the regex as destination topic)

KafkaPostman_REDIS_TOPICS_PREFIX - A prefix that will be added to every topic that has been discovered. Using this prefix will allow the message distributor to get a list of all topics discovrered.

KafkaPostman_REDIS_TOPICS_SLIDING_EXPIRY_MS - A number of milliseconds that a discovered topic or a message part and the topic assigned to it will be saved in redis for since the last time they were discovered/saved.

KafkaPostman_TOPICS_DISCOVERY_INTERVAL - The time in milliseconds between each topics discovery cycle when the KafkaPostman_TOPICS_DISCOVERY_METHOD variable is set to REGEX.

KafkaPostman_LOGGING_FORMAT - The logging format that will be used. The default is: timestamp=%s;module=smart-onion_%s;method=%s;severity=%s;state=%s;metric/metric_family=%s;exception_msg=%s;exception_type=%s;message=%s

KafkaPostman_KAFKA_CONSUMER_CLIENT_ID - The client ID that will be used when consuming messages from the source server. Can include the placeholders such as "{{#instance_id}}" and "{{#time_started}}" that are automatically resolved to the instance id (a unique identifier that is generated when the service launches) and to the unix time on which the service started. Accordingly.

KafkaPostman_KAFKA_CONSUMER_SERVER - The server (such as localhost:9020) from which the service will consume the source data.

KafkaPostman_KAFKA_PRODUCER_CLIENT_ID - The client ID that will be used when sending messages to the destination server. Can include the placeholders such as "{{#instance_id}}" and "{{#time_started}}" that are automatically resolved to the instance id (a unique identifier that is generated when the service launches) and to the unix time on which the service started. Accordingly.

KafkaPostman_KAFKA_PRODUCER_SERVER - The server (such as localhost:9020) to which the service will connect to publish the data in the relevant topics.

KafkaPostman_TOPICS_KAFKA_TOPIC - The Kafka topic that will be used to by the service to discover the available destination topics if KafkaPostman_TOPICS_DISCOVERY_METHOD is set to TOPICS_TOPIC

KafkaPostman_SOURCE_TOPIC - The Kafka topic that will be used by the service to pull data from (and to distribute that data to the available destination topics)
```
## Example use case
For example, here's the case that drove me to develop this project:
The pensu project is comprised of a Docker/Kubernetes ready service that is designed to pull metrics from a Kafka topic and then use the HTM algorithm by Numenta to generate predictions and anomaly scores and push them to another Kafka topic. Both the metrics, the predictions and the anomaly scores are written in the Graphite line format, like this: 
```
acme.it.servers.webservers.public_website1.cpu.user_time <value> <timestamp_in_unixtime>
```

Since that this service pulls the metrics data from a single topic, if the number of metrics or their sample rate will increase, it would have to have the ability to scale, but if you'd simply run multiple instances of the pensu service to share the load and they would all pull from the same kafka topic, then some of the datapoints for the same metric will be consumed by instance A while other datapoints of the same metric will be consumed by other instances. This of course will not allow the pensu services to make accurate predictions and anomaly detections. But, since that pensu is able to be configured to periodically send the topic it is listening on to a Kafka topic, we can install as many instances of this project, kafka-postman, and configure it to discover topics based on a "topics topic" which is the same topic that pensu will report the topic it listens on to, this service will periodically get the topics each pensu instance is listening on, and by using a regex to extract the metric name from the metric line, it will randomly (or by using a round robin logic) select a topic for every newly seen metric and then it will keep sending the all the datapoints of every metric to the same pensu instance. Problem solved!

Also, since that it is using a sliding cache mechanism, if one of the pensu instances dies, it would stop sending the topic on which it is listening on to the "topics topic" and the kafka-postman will eventually select another pensu instance to handle the datapoints of the metrics used to be handled by the now dead pensu instance.
