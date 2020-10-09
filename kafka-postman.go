package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/joomcode/redispipe/redis"
	"github.com/joomcode/redispipe/rediscluster"
	"github.com/joomcode/redispipe/redisconn"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"
)

type SortableTopicsListItem struct {
	TopicName string
	TopicIndex int
}

type SortableTopicsListItemsDesc []SortableTopicsListItem

func (sortableTopicListItems SortableTopicsListItemsDesc) Len() int {
	return len(sortableTopicListItems)
}

func (sortableTopicListItems SortableTopicsListItemsDesc) Less(i, j int) bool {
	return sortableTopicListItems[i].TopicIndex < sortableTopicListItems[j].TopicIndex
}

func (sortableTopicListItems SortableTopicsListItemsDesc) Swap(i, j int) {
	sortableTopicListItems[i], sortableTopicListItems[j] = sortableTopicListItems[j], sortableTopicListItems[i]
}

type SortableTopicsListItemsAsc []SortableTopicsListItem

func (sortableTopicListItems SortableTopicsListItemsAsc) Len() int {
	return len(sortableTopicListItems)
}

func (sortableTopicListItems SortableTopicsListItemsAsc) Less(i, j int) bool {
	return sortableTopicListItems[i].TopicIndex > sortableTopicListItems[j].TopicIndex
}

func (sortableTopicListItems SortableTopicsListItemsAsc) Swap(i, j int) {
	sortableTopicListItems[i], sortableTopicListItems[j] = sortableTopicListItems[j], sortableTopicListItems[i]
}


func generateConsumer(bootstrapServer, groupId, clientId, defaultOffset, sourceTopic string) kafka.Consumer {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServer,
		"group.id":          groupId,
		"client.id":         clientId,
		"auto.offset.reset": defaultOffset,
	})

	if err != nil {
		panic(err)
	}

	err = consumer.SubscribeTopics([]string{sourceTopic}, nil)
	if err != nil {
		panic(err)
	}

	return *consumer
}
func generateProducer(bootstrapServer, groupId, clientId string) kafka.Producer {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServer,
		"group.id":          groupId,
		"client.id":         clientId,
	})

	if err != nil {
		panic(err)
	}

	return *producer
}

func main() {
	CUR_FUNCTION := "main"
	var topicsTopicReader *kafka.Consumer
	discoveredTopics := make([]string, 0, 0)
	allKafkaTopicsSeen := make([]string, 0, 0)
	roundRobinTopicIndex := 0

	LogForwarder(nil, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_INFO, MessageFormat: "Kafka-Postman started."})
	lastTopicsDiscoveryTimestamp := int64(0)

	//1. Get configuration
	config := getConfig()
	configJsonBytes, err := json.Marshal(config)
	if err != nil {
		LogForwarder(nil, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_ERROR, MessageFormat: "ERROR: Configuration is not JSON parsable. CANNOT CONTINUE. (%v)"}, err)
		os.Exit(8)
	}
	switch config.LogLevel {
	case LogLevel_VERBOSE:
		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "LogLevel is VERBOSE"})
	case LogLevel_DEBUG:
		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, MessageFormat: "LogLevel is DEBUG"})
	case LogLevel_INFO:
		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_INFO, MessageFormat: "LogLevel is INFO"})
	case LogLevel_WARN:
		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_WARN, MessageFormat: "LogLevel is WARN"})
	case LogLevel_ERROR:
		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_ERROR, MessageFormat: "LogLevel is ERROR"})
	default:
		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_INFO, MessageFormat: "LogLevel is UNKNOWN"})
	}
	LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_INFO, MessageFormat: "This is the loaded config: %s"}, configJsonBytes)

	//2. for message_idx, message in kafka.GetMessages(KafkaPostman_SOURCE_TOPIC) {
	LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_INFO, MessageFormat:"Connecting to the Kafka..."})
	kafkaConsumer := generateConsumer(config.KafkaConsumerServerHost, config.KafkaConsumerGroupId, config.KafkaConsumerClientId, config.KafkaConsumerDefaultOffset, config.SourceTopic)
	kafkaProducer := generateProducer(config.KafkaProducerServerHost, config.KafkaProducerGroupId, config.KafkaProducerClientId)
	for {
		//3. Get a message from Kafka
		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, MessageFormat:"Starting to wait for messages from kafka..."})
		msg, err := kafkaConsumer.ReadMessage(10 * time.Second)
		if err != nil {
			if err.Error() == kafka.ErrTimedOut.String() {
				LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: err, Level: LogLevel_INFO, MessageFormat:"Waiting for a message from Kafka (%v.%v@%v/%v@%v)..."}, config.KafkaConsumerClientId, config.KafkaConsumerGroupId, config.KafkaConsumerServerHost, config.SourceTopic, config.KafkaConsumerDefaultOffset)
			} else {
				// The client will automatically try to recover from all errors.
				LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: err, Level: LogLevel_ERROR, MessageFormat: "Consumer error: %v (%v)"}, err, msg)
			}
		} else {
			timeHandlingStarted := time.Now()
			msgValue := string(msg.Value)

			//4. (Re-)Discover topics if needed
			var timeDiscoveryTaken time.Duration
			if timeSinceLastDiscovery := time.Now().Unix() - lastTopicsDiscoveryTimestamp; timeSinceLastDiscovery > config.DiscoveryIntervalSecs || len(discoveredTopics) == 0 || len(allKafkaTopicsSeen) == 0 {
				timeDiscoveryStarted := time.Now()
				LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, MessageFormat: "Re-discovering destination topics. (timeSinceLastDiscovery: %v, config.DiscoveryIntervalSecs: %v, len(discoveredTopics): %v)"}, timeSinceLastDiscovery, config.DiscoveryIntervalSecs, len(discoveredTopics))
				discoveredTopics, allKafkaTopicsSeen = discoverTopics(config, kafkaConsumer, topicsTopicReader)
				if config.AutoDestinationTopicFilteringEnabled {
					discoveredTopics = filterOutInvalidTopics(discoveredTopics, config)
					LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "After filtering the invalid topics this is the list of discovered topics: %v"}, discoveredTopics)
				} else {
					LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_INFO, MessageFormat: "Destination topics filtering is disabled. If this feature is disabled the service might write to the same topic it is reading from or to topics that are internally used by kafka, this can cause unpredictable behaviour"})
				}
				lastTopicsDiscoveryTimestamp = time.Now().Unix()
				timeDiscoveryTaken = time.Now().Sub(timeDiscoveryStarted)
				if config.LogLevel == LogLevel_VERBOSE {
					LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "Topics discovery ran. Time taken: %v, Topics discovered: %v, All Kafka topics seen: %v"}, timeDiscoveryTaken, discoveredTopics, allKafkaTopicsSeen)
				} else {
					LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, MessageFormat: "Topics discovery ran. Time taken: %v, No. of topics discovered: %v, No. of all Kafka topics seen: %v"}, timeDiscoveryTaken, len(discoveredTopics), len(allKafkaTopicsSeen))
				}
				if len(discoveredTopics) == 0 {
					panic("ERROR: No topics were discovered. CANNOT CONTINUE.")
				}
			}

			//5. Decide on a default destination topic (based on distribution strategy)
			timeDestinationTopicDecisionStarted := time.Now()
			LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, MessageFormat: "Getting the default destination topic based on the distribution strategy..."})
			defaultDestinationTopic := getDefaultDestinationTopic(config, discoveredTopics, &roundRobinTopicIndex, msgValue)
			LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "Currently, the destination topic is: %v"}, defaultDestinationTopic)

			//6. Handle topic pinning (if enabled)
			if config.TopicPinningEnabled {
				LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, MessageFormat: "Handling topic pinning..."})
				defaultDestinationTopic = handleTopicPinning(config, defaultDestinationTopic, msgValue)
				LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "Currently, the destination topic is: %s"}, defaultDestinationTopic)
			} else {
				LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "Topic pinning is disabled..."})
			}
			timeDestinationTopicDecisionTaken := time.Now().Sub(timeDestinationTopicDecisionStarted)


			//7. Topic validation - For DISTRIBUTION_STRATEGY_REGEX, DISCOVERY_METHOD_MANUAL or DISCOVERY_METHOD_TOPICS_TOPIC (in these configuration the service DYNAMICALLY selects a topic based on information that is not provided by Kafka)
			if config.DistributionStrategy == DISTRIBUTION_STRATEGY_REGEX || config.DiscoveryMethod == DISCOVERY_METHOD_MANUAL || config.DiscoveryMethod == DISCOVERY_METHOD_TOPICS_TOPIC {
				//If the dest topic is valid proceed, otherwise resort to configured default topic
				if !validateDestinationTopic(defaultDestinationTopic, allKafkaTopicsSeen, config) {
					LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Level: LogLevel_WARN, Error: nil, MessageFormat: "Topic %v is invalid according to the topic validation methods selected. Sending this message to default destination topic %v"}, defaultDestinationTopic, config.DefaultTargetTopic)
					defaultDestinationTopic = config.DefaultTargetTopic
				}
			}

			if config.LogLevel >= LogLevel_VERBOSE {
				LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Level: LogLevel_VERBOSE, Error: nil, MessageFormat: "Will forward the message '%s' to topic %s"}, msg.Value, defaultDestinationTopic)
			} else {
				LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Level: LogLevel_INFO, Error: nil, MessageFormat: "Will forward current message to topic %s"}, defaultDestinationTopic)
			}

			//8. kafka.PublishMessage(selected_topic, message)
			deliveryChan := make(chan kafka.Event)
			kafkaProducer.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &defaultDestinationTopic, Partition: kafka.PartitionAny},
				Value:          []byte(msgValue),
				Headers:        []kafka.Header{{Key: "ProducedBy", Value: []byte("KafkaPostman_v" + GetMyVersion())}},
			}, deliveryChan)

			timeHandlingTaken := time.Now().Sub(timeHandlingStarted)
			LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Level: LogLevel_INFO, Error: nil, MessageFormat: "Finished handling message. Sent to topic '%v' (timeDiscoveryTaken: %v, timeDestinationTopicDecisionTaken: %v, timeTotalHandlingTaken: %v)"}, defaultDestinationTopic, timeDiscoveryTaken, timeDestinationTopicDecisionTaken, timeHandlingTaken)
		}
	}
}

func validateDestinationTopic(destinationTopic string, kafkaTopics []string, config Config) (result bool) {
	CUR_FUNCTION := "validateDestinationTopic"
	result = true
	if config.TopicsValidationValidateAgainstKafka ||
		len(config.TopicsValidationWhitelist) > 0 ||
		len(config.TopicsValidationBlacklist) > 0 ||
		len(config.TopicsValidationRegexWhitelist) > 0 ||
		len(config.TopicsValidationRegexBlacklist) > 0 {
		if result && config.TopicsValidationValidateAgainstKafka && !stringInSlice(destinationTopic, kafkaTopics) {
			if config.LogLevel == LogLevel_VERBOSE {
				LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "The topic %v was found to be invalid. It is not on the list of all kafka topics. (allKafkaTopics: %v)"}, destinationTopic, kafkaTopics)
			} else {
				LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, MessageFormat: "The topic %v was found to be invalid. It is not on the list of all kafka topics."}, destinationTopic)
			}
			result = false
			return
		}
		if result && len(config.TopicsValidationWhitelist) > 0 && !stringInSlice(destinationTopic, config.TopicsValidationWhitelist) {
			LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "The topic %v was found to be invalid. It is not on the whitelist."}, destinationTopic)
			result = false
			return
		}
		if result && len(config.TopicsValidationBlacklist) > 0 && stringInSlice(destinationTopic, config.TopicsValidationBlacklist) {
			LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "The topic %v was found to be invalid. It is on the blacklist."}, destinationTopic)
			result = false
			return
		}
		if result && len(config.TopicsValidationRegexWhitelist) > 0 && len(extractMatches([]string{destinationTopic}, config.TopicsValidationRegexWhitelist)) == 0 {
			LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "The topic %v was found to be invalid. It does not match any of the whitelist regexes."}, destinationTopic)
			result = false
			return
		}
		if result && len(config.TopicsValidationRegexBlacklist) > 0 && len(extractMatches([]string{destinationTopic}, config.TopicsValidationRegexBlacklist)) > 0 {
			LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "The topic %v was found to be invalid. It matched at least one of the blacklist regexes."}, destinationTopic)
			result = false
			return
		}
	}
	LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "The topic %v was found to be valid."}, destinationTopic)
	return
}

func filterOutInvalidTopics(discoveredTopics []string, config Config) []string {
	temp := make([]string, 0)
	for _, discoveredTopic := range discoveredTopics {
		if !strings.HasPrefix(discoveredTopic, "__") && (config.KafkaConsumerServerHost != config.KafkaProducerServerHost || discoveredTopic != config.SourceTopic) {
			temp = append(temp, discoveredTopic)
		}
	}
	discoveredTopics = temp
	return discoveredTopics
}

func handleTopicPinning(config Config, defaultDestinationTopic string, msg string) string {
	CUR_FUNCTION := "handleTopicPinning"
	var sender redis.Sender
	var err error

	ctx := context.Background()
	if config.TopicPinningRedisClusterName == "" {
		//Single redis
		SingleRedis := func(ctx context.Context) (redis.Sender, error) {
			opts := redisconn.Opts{
				DB:       config.TopicPinningRedisDbNo,
				Password: config.TopicPinningRedisDbPassword,
				Logger:   redisconn.NoopLogger{}, // shut up logging. Could be your custom implementation.
			}
			conn, err := redisconn.Connect(ctx, config.TopicPinningRedisAddresses[0], opts)
			return conn, err
		}

		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, MessageFormat:"Connecting to a single redis (the cluster name is empty)..."})
		sender, err = SingleRedis(ctx)
	} else {
		//Redis cluster
		ClusterRedis := func(ctx context.Context) (redis.Sender, error) {
			opts := rediscluster.Opts{
				HostOpts: redisconn.Opts{
					// No DB
					Password: config.TopicPinningRedisDbPassword,
					// Usually, no need for special logger
				},
				Name:   config.TopicPinningRedisClusterName, // name of a cluster
				Logger: rediscluster.NoopLogger{},           // shut up logging. Could be your custom implementation.
			}
			addresses := config.TopicPinningRedisAddresses // one or more of cluster addresses
			cluster, err := rediscluster.NewCluster(ctx, addresses, opts)
			return cluster, err
		}

		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, MessageFormat:"Connecting to a redis cluster (the cluster name is not empty)..."})
		sender, err = ClusterRedis(ctx)
	}
	if err != nil {
		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: err, Level: LogLevel_WARN, MessageFormat: "Could not process topic pinning. An error occurred while opening a connection to Redis."})
	} else {
		defer sender.Close()

		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat:"Creating the sync object..."})
		sync := redis.SyncCtx{sender} // wrapper for synchronous api

		//6.i.c. Create a fingerprint of the message by hashing the text extracted by the regex groups
		if config.LogLevel >= LogLevel_VERBOSE {
			LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat:"Successfully connected to redis. Calculating a message fingerprint for the current message... (msg: `%v', config.TopicPinningRegex: %v, config.TopicPinningRegexGroupIndexes: %v)"}, msg, config.TopicPinningRegex, config.TopicPinningRegexGroupIndexes)
		} else {
			LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, MessageFormat: "Successfully connected to redis. Calculating a message fingerprint for the current message..."})
		}
		regexMatches := config.TopicPinningRegex.FindStringSubmatch(msg)
		messageFingerprintRaw :=  make([]string, 0)
		for _, groupIdx := range config.TopicPinningRegexGroupIndexes {
			messageFingerprintRaw = append(messageFingerprintRaw, regexMatches[groupIdx])
		}
		messageFingerprintRawBytes, _ := json.Marshal(messageFingerprintRaw)
		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "This is the raw text used as the basis for the fingerprint: %v"}, string(messageFingerprintRawBytes))
		messageFingerprint := fmt.Sprintf("%x", md5.Sum(messageFingerprintRawBytes))
		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "Created the following fingerprint for the current message. (fingerprint: %v, message: `%v')"}, messageFingerprint, msg)

		//6.i.d. Try to find the fingerprint on Redis by using the fingerprint as the key
		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, MessageFormat:"Trying to find the message fingerprint on redis..."})
		res := sync.Do(ctx, "GET", messageFingerprint)
		if err := redis.AsError(res); err == nil && res != nil {
			//4.2.1. The fingerprint was found on Redis:
			defaultDestinationTopic = string(res.([]byte))
			if config.LogLevel == LogLevel_VERBOSE {
				LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "The fingerprint was found on redis. Selecting the topic found as the default topic for the current message. (message: `%v', fingerprint: %v, topic: %v)"}, msg, messageFingerprint, defaultDestinationTopic)
			} else {
				LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, MessageFormat: "The fingerprint was found on redis. Selecting the topic found as the default topic for the current message."})
			}
		} else {
			LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, MessageFormat:"The fingerprint was not found on redis. Selecting the topic found as the default topic so far (by other methods) for the current message. (err: %v, res: %v)"}, err, res)
		}

		//6.i.e. Write the value of defaultDestinationTopic to Redis under the current fingerprint to either reset the fingerprint sliding expiration (if already exists) or to cache it (if it's new)
		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, MessageFormat:"Updating the selected topic for the current message fingerprint to either reset the sliding expiration if already exist or to set the fingerprint for other similar messages if not already exist..."})
		res = sync.Do(ctx, "SET", messageFingerprint, defaultDestinationTopic, "PX", config.TopicPinningHashSlidingExpiryMs)
		if err := redis.AsError(res); err != nil {
			LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: err, Level: LogLevel_WARN, MessageFormat: "Could not process topic pinning. An error occurred while updating a value to Redis."})
		}
	}
	return defaultDestinationTopic
}

func getDefaultDestinationTopic(config Config, discoveredTopics []string, roundRobinTopicIndex *int, msg string) string {
	CUR_FUNCTION := "getDefaultDestinationTopic"
	defaultDestinationTopic := config.DefaultTargetTopic
	curTopicIndex := -1
	switch config.DistributionStrategy {
	case DISTRIBUTION_STRATEGY_RANDOM:
		curTopicIndex = rand.Intn(len(discoveredTopics))
		defaultDestinationTopic = discoveredTopics[curTopicIndex]
	case DISTRIBUTION_STRATEGY_ROUND_ROBIN:
		*roundRobinTopicIndex++
		if *roundRobinTopicIndex >= len(discoveredTopics) {
			*roundRobinTopicIndex = 0
		}
		curTopicIndex = *roundRobinTopicIndex
		defaultDestinationTopic = discoveredTopics[curTopicIndex]
	case DISTRIBUTION_STRATEGY_REGEX:
		regexMatchedGroups := config.DistributionRegex.FindStringSubmatch(msg)
		if config.DistributionRegexGroupIndex < len(regexMatchedGroups) {
			defaultDestinationTopic = regexMatchedGroups[config.DistributionRegexGroupIndex]
		} else {
			LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_WARN, MessageFormat:"WARN: The requested distribution strategy is regex but the regex group index is %v which is equal to or greater than the number of groups found by the regex `%v' on message %v which resulted in this array of matched groups %v"}, config.DistributionRegexGroupIndex, config.DistributionRegex, msg, regexMatchedGroups)
		}
	default:
		panic("Unknown message distribution method. LEAVING")
	}

	return defaultDestinationTopic
}

func discoverTopics(config Config, kafkaConsumer kafka.Consumer, topicsTopicReader *kafka.Consumer) (topicsDiscovered []string, allKafkaTopics []string) {
    CUR_FUNCTION := "discoverTopics"
	additionalArgument := ""
	LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "Function start. Getting all kafka topics"})
    allKafkaTopics = getAllTopicNamesFromKafka(kafkaConsumer, config)
	tmpTopicsList := make([]string, 0, 0)
	switch config.DiscoveryMethod {
	case DISCOVERY_METHOD_REGEX:
		additionalArgument = config.DiscoveryRegexString
		tmpTopicsList = discoverTopicsByRegex(allKafkaTopics, config)
	case DISCOVERY_METHOD_MANUAL:
		additionalArgument = config.DiscoveryManualTopicsList
		if len(config.DiscoveryManualTopicsList) > 0 {
			tmpTopicsList = strings.Split(config.DiscoveryManualTopicsList, config.DiscoveryManualTopicsListSeparator)
		}
	case DISCOVERY_METHOD_TOPICS_TOPIC:
		additionalArgument = config.DiscoveryTopicsTopic
		tmpTopicsList = discoverTopicsByTopicsTopic(config, kafkaConsumer, topicsTopicReader)
	default:
		panic("Unknown topics discovery method. LEAVING")
	}
	topicsDiscovered = tmpTopicsList

	//Topics validation for DISCOVERY_METHOD_REGEX and DISTRIBUTION_STRATEGY != DISTRIBUTION_STRATEGY_REGEX
	LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "tmpTopicsList: %v. Validating them if needed."}, tmpTopicsList)
	validatedTopicsList := make([]string, 0, 0)
	for _, discoveredTopic := range tmpTopicsList {
		if validateDestinationTopic(discoveredTopic, allKafkaTopics, config) {
			validatedTopicsList = append(validatedTopicsList, discoveredTopic)
		} else {
			LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "Topic %v is invalid. SKIPPED."}, discoveredTopic)
		}
	}
	copy(topicsDiscovered, validatedTopicsList)

	LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_INFO, MessageFormat: "Discovered topics. Number of topics discovered: %v, Total number of topics seen on Kafka: %v (ValidatedTopicsList: %v, Discovery method: %v, additional argument: %v)"}, len(topicsDiscovered), len(allKafkaTopics), validatedTopicsList, config.DiscoveryMethod, additionalArgument)
	return
}

func discoverTopicsByTopicsTopic(config Config, kafkaConsumer kafka.Consumer, topicsTopicReader *kafka.Consumer) []string {
	CUR_FUNCTION := "discoverTopicsByTopicsTopic"
	LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "Function start. Building a consumer for the topics_topic topic (%v) if needed."}, config.DiscoveryTopicsTopic)
	tmpTopicsList := make([]string, 0, 0)
	if topicsTopicReader == nil {
		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_INFO, MessageFormat: "Topics reader is nil. Generating consumer for the topics topic {host: %v, group_id: %v, client_id: %v, offset: %v, topic: %v}"}, config.DiscoveryTopicsTopicServerHost, config.DiscoveryTopicsTopicGroupId, config.DiscoveryTopicsTopicClientId, KAFKA_DEFAULT_OFFSET_BEGINNING, config.DiscoveryTopicsTopic)
		topicsTopicReaderObj := generateConsumer(config.DiscoveryTopicsTopicServerHost, config.DiscoveryTopicsTopicGroupId, config.DiscoveryTopicsTopicClientId, KAFKA_DEFAULT_OFFSET_BEGINNING, config.DiscoveryTopicsTopic)
		topicsTopicReader = &topicsTopicReaderObj
	} else {
		LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "Function start. The consumer for the topics_topic topic (%v) already exists. Using the same one."}, config.DiscoveryTopicsTopic)
	}
	discoveryStarted := time.Now()
	sortableTopicsList := make([]SortableTopicsListItem, 0, 0)
	jsonTopicIndexParseFailed := false
	LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "Attempting to read topics from the topics topic {host: %v, group_id: %v, client_id: %v, topic: %v, read_timeout: %v}"}, config.DiscoveryTopicsTopicServerHost, config.DiscoveryTopicsTopicGroupId, config.DiscoveryTopicsTopicClientId, config.DiscoveryTopicsTopic, config.DiscoveryTopicsTopicMaxWaitForTopics)
	for {
		msg, err := kafkaConsumer.ReadMessage(time.Duration(config.DiscoveryTopicsTopicMaxWaitForTopics) * time.Millisecond)
		if err != nil {
			if err.Error() == kafka.ErrTimedOut.String() {
				//These are probably all the topics published for now
				break
			} else {
				//Log error and attempt to recover...
				LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: err, Level: LogLevel_ERROR, MessageFormat: "Topics topic discoverer consumer error: %v (%v)"}, err, msg)
				if len(tmpTopicsList) > 0 {
					//If some topics were discovered - we'll continue with them
					break
				} else {
					//otherwise, we'll remain here and try to discover topics until timeout will kick in
				}
			}
		} else {
			topicsTopicMessage := string(msg.Value)
			topicsTopicMessageIsTopicName := false

			LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "Discovered the following raw topic: `%v' (TopicsTopicMayContainJson: %v)"}, topicsTopicMessage, config.TopicsTopicMayContainJson)
			if config.TopicsTopicMayContainJson {
				var topicsTopicObject map[string]interface{}
				e := json.Unmarshal([]byte(topicsTopicMessage), &topicsTopicObject)
				if e == nil {
					//If this is a JSON, extract topic name from message
					topicIndexInt, topicIndexIsNotInt := topicsTopicObject[config.TopicsTopicSortByJsonField].(int)
					if jsonTopicIndexParseFailed || topicIndexIsNotInt {
						if !jsonTopicIndexParseFailed {
							LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_WARN, MessageFormat: "Failed to parse the topic index from the JSON found on the topics topic. The list of topics will NOT be sorted."})
							jsonTopicIndexParseFailed = true
						}
						tmpTopicsList = append(tmpTopicsList, topicsTopicMessage)
					} else {
						if topicName, ok := topicsTopicObject[config.TopicsTopicTopicNameJsonField]; ok {
							topicNameString := fmt.Sprintf("%v", topicName)
							if len(topicNameString) > 0 && topicNameString != "<nil>" {
								sortableTopicsList = append(sortableTopicsList, SortableTopicsListItem{
									TopicName:  fmt.Sprintf("%v", topicNameString),
									TopicIndex: topicIndexInt,
								})
							} else {
								panic("ERROR: The topic name extracted from the JSON retrieved from the topics topic is either null or empty. CANNOT CONTINUE.")
							}
						} else {
							panic("ERROR: The field specified as the field that contains the topic name in the JSON objects sent to the topics topic doesn't exist in this JSON. CANNOT CONTINUE.")
						}
					}
				} else {
					//If we received an error while parsing the json we'll assume it's not a JSON but a raw topic name
					topicsTopicMessageIsTopicName = true
				}
			} else {
				//If it's not JSON it's probably a raw topic name
				topicsTopicMessageIsTopicName = true
			}

			if topicsTopicMessageIsTopicName {
				if !stringInSlice(topicsTopicMessage, tmpTopicsList) {
					tmpTopicsList = append(tmpTopicsList, topicsTopicMessage)
				} else {
					//Keeping the list unique - doing nothing here
				}
			}
		}

		if time.Now().Sub(discoveryStarted).Milliseconds() > config.DiscoveryTopicsTopicMaxDiscoveryTimeout {
			//Prevent the loop from running forever if the consumers will keep sending topic names to the topics topic at a high rate
			break
		}
	}

	LogForwarder(&config, LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, MessageFormat: "Discovered the following topics: sortableTopicsList:`%v', tmpTopicsList:`%v'"}, sortableTopicsList, tmpTopicsList)
	if config.TopicsTopicMayContainJson && !jsonTopicIndexParseFailed && len(sortableTopicsList) > 0 {
		if config.TopicsTopicSortByJsonFieldAscending {
			sort.Sort(SortableTopicsListItemsDesc(sortableTopicsList))
		} else {
			sort.Sort(SortableTopicsListItemsAsc(sortableTopicsList))
		}
		//TODO: Probably we should convert the result from sort to a strings list and return it...
	}
	return tmpTopicsList
}

func discoverTopicsByRegex(allKafkaTopics []string, config Config) []string {
	//CUR_FUNCTION := "discoverTopicsByRegex"
	tmpTopicsList := make([]string, 0, 0)
	for _, kafkaTopic := range allKafkaTopics {
		if config.DiscoveryRegex.Match([]byte(kafkaTopic)) {
			tmpTopicsList = append(tmpTopicsList, kafkaTopic)
		}
	}

	return tmpTopicsList
}

func getAllTopicNamesFromKafka(kafkaConsumer kafka.Consumer, config Config) []string {
	//CUR_FUNCTION := "getAllTopicNamesFromKafka"
	tmpTopicsList := make([]string, 0, 0)
	kafkaAdminClient, err := kafka.NewAdminClientFromConsumer(&kafkaConsumer)
	if err == nil {
		topicsInfo, topicsCollectionErr := kafkaAdminClient.GetMetadata(nil, true, -1)
		if topicsCollectionErr == nil {
			for _, topic := range topicsInfo.Topics {
				tmpTopicsList = append(tmpTopicsList, topic.Topic)
			}
		} else {
			panic("Failed to get a list of all topics on this Kafka cluster. (The following error was thrown from kafkaAdminClient.GetMetadata). " + fmt.Sprintf("%v", topicsCollectionErr))
		}
	} else {
		panic("Failed to get a list of all topics on this Kafka cluster. (The following error was thrown from kafkaAdminClient.GetMetadata)." + fmt.Sprintf("%v", err))
	}

	return tmpTopicsList
}
