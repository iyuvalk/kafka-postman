package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/joomcode/redispipe/redis"
	"github.com/joomcode/redispipe/rediscluster"
	"github.com/joomcode/redispipe/redisconn"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DiscoveryMethod                    string
	DiscoveryRegex                     *regexp.Regexp
	DiscoveryManualTopicsList          string
	DiscoveryManualTopicsListSeparator string
	DistributionRegex                  *regexp.Regexp
	DistributionRegexGroupIndex        int
	DistributionStrategy               string
	TopicsDistributionRegexGroupIndex  int
	TopicPinningEnabled                bool
	TopicPinningRedisAddresses         []string
	TopicPinningRedisDbNo              int
	TopicPinningRedisDbPassword        string
	TopicPinningRedisClusterName       string
	TopicPinningHashSlidingExpiryMs    int64
	DiscoveryIntervalMs                int64
	LoggingFormat                      string
	LogLevel                           LogLevel
	KafkaConsumerClientId              string
	KafkaConsumerGroupId               string
	KafkaConsumerDefaultOffset         string
	KafkaConsumerServerHost            string
	KafkaProducerClientId              string
	KafkaProducerServerHost            string
	DiscoveryTopicsTopic               string
	SourceTopic                        string
	DefaultTargetTopic                 string
	AutoCreateMissingTopics            bool
	TopicPinningRegex                  *regexp.Regexp
	TopicPinningRegexGroupIndexes      []int64
}

type ConfigParamValue struct {
	Type         ConfigParamValueType
	ValueInt     int64
	ValueBool    bool
	ValueString  string
	ValueRegex   *regexp.Regexp
}

func autoSelectConfig(defaultValue string, envVarNameToTest CONFIG_ENV_VARS, configValidationMode []ConfigValidationMode, configParamValueType ConfigParamValueType, stringsList []string) ConfigParamValue {
	CUR_FUNCTION := "autoSelectConfig"
	var rawStringValue string

	fallingBackToDefaultValue := false
	for _, validationRequested := range configValidationMode {
		switch validationRequested {
		case ConfigValidationMode_LIST_BASED:
			if stringsList == nil || len(stringsList) == 0 {
				panic("ERROR: The list of acceptable values should not be empty (otherwise every value would become invalid)")
			}
			if len(os.Getenv(string(envVarNameToTest))) == 0 || !stringInSlice(os.Getenv(string(envVarNameToTest)), stringsList) {
				rawStringValue = defaultValue
				fallingBackToDefaultValue = true
			} else {
				rawStringValue = os.Getenv(string(envVarNameToTest))
			}
		case ConfigValidationMode_IS_EMPTY:
			if len(os.Getenv(string(envVarNameToTest))) == 0 {
				rawStringValue = defaultValue
				fallingBackToDefaultValue = true
			} else {
				rawStringValue = os.Getenv(string(envVarNameToTest))
			}
		case ConfigValidationMode_IS_INT:
			tempString := os.Getenv(string(envVarNameToTest))
			_, err := strconv.ParseInt(tempString, 10, 64)
			if err == nil {
				rawStringValue = tempString
				fallingBackToDefaultValue = true
			} else {
				rawStringValue = defaultValue
			}
		case ConfigValidationMode_IS_BOOL:
			tempString := os.Getenv(string(envVarNameToTest))
			if tempString == "True" || tempString == "true" || tempString == "False" || tempString == "false" {
				rawStringValue = tempString
				fallingBackToDefaultValue = true
			} else {
				rawStringValue = defaultValue
			}
		case ConfigValidationMode_IS_REGEX:
			if len(os.Getenv(string(envVarNameToTest))) > 0 {
				_, err := regexp.Compile(os.Getenv(string(envVarNameToTest)))
				if err != nil {
					rawStringValue = defaultValue
					fallingBackToDefaultValue = true
				} else {
					rawStringValue = os.Getenv(string(envVarNameToTest))
				}
			} else {
				rawStringValue = defaultValue
			}
		case ConfigValidationMode_IS_REGEX_MATCH_AND:
			if stringsList == nil || len(stringsList) == 0 {
				panic("ERROR: The list of regexes to test should not be empty (otherwise every value would become invalid)")
			}
			allRegexesMatched := false
			for i, regexToTestRaw := range stringsList {
				regexToTest, err := regexp.Compile(regexToTestRaw)
				if err != nil {
					panic("ERROR: The regex " + regexToTestRaw + " (No. " + strconv.Itoa(i) + ") could not be compiled as a regex")
				}
				allRegexesMatched = allRegexesMatched && regexToTest.Match([]byte(os.Getenv(string(envVarNameToTest))))
			}
			if allRegexesMatched {
				rawStringValue = os.Getenv(string(envVarNameToTest))
			} else {
				rawStringValue = defaultValue
				fallingBackToDefaultValue = true
			}
		case ConfigValidationMode_IS_REGEX_MATCH_OR:
			if stringsList == nil || len(stringsList) == 0 {
				panic("ERROR: The list of regexes to test should not be empty (otherwise every value would become invalid)")
			}
			allRegexesMatched := false
			for i, regexToTestRaw := range stringsList {
				regexToTest, err := regexp.Compile(regexToTestRaw)
				if err != nil {
					panic("ERROR: The regex " + regexToTestRaw + " (No. " + strconv.Itoa(i) + ") could not be compiled as a regex")
				}
				allRegexesMatched = allRegexesMatched || regexToTest.Match([]byte(os.Getenv(string(envVarNameToTest))))
			}
			if allRegexesMatched {
				rawStringValue = os.Getenv(string(envVarNameToTest))
			} else {
				rawStringValue = defaultValue
				fallingBackToDefaultValue = true
			}
		default:
			panic("ERROR: Unknown config validation type.")
		}

		_, environmentVarSet := os.LookupEnv(string(envVarNameToTest))
		if fallingBackToDefaultValue && environmentVarSet {
			LogForwarder(LogMessage{Caller: CUR_FUNCTION, Level: LogLevel_WARN, Message: "The value of " + string(envVarNameToTest) + " is invalid. Using the default value " + defaultValue})
		}
	}

	switch configParamValueType {
	case ConfigParamValueType_INT:
		val, _ := strconv.ParseInt(rawStringValue, 10, 64)
		return ConfigParamValue{
		   Type: ConfigParamValueType_INT,
		   ValueInt: val,
		}
	case ConfigParamValueType_BOOL:
		if strings.ToUpper(rawStringValue) == "TRUE" {
			return ConfigParamValue{
				Type: ConfigParamValueType_BOOL,
				ValueBool: true,
			}
		} else {
			return ConfigParamValue{
				Type: ConfigParamValueType_BOOL,
				ValueBool: false,
			}
		}
	case ConfigParamValueType_STRING:
		return ConfigParamValue{
			Type: ConfigParamValueType_STRING,
			ValueString: rawStringValue,
		}
	case ConfigParamValueType_REGEX:
		val, _ := regexp.Compile(rawStringValue)
		return ConfigParamValue{
			Type: ConfigParamValueType_REGEX,
			ValueRegex: val,
		}
	default:
		panic("ERROR: Unknown config parameter value type.")
	}
}
func getConfig() Config {
	clientId := uuid.Must(uuid.NewRandom())
	const (
		DEFAULT_DISCOVERY_METHOD                       = DISCOVERY_METHOD_REGEX
		DEFAULT_DISCOVERY_MANUAL_TOPICS_LIST           = ""
		DEFAULT_DISCOVERY_MANUAL_TOPICS_LIST_SEPARATOR = ","
		DEFAULT_DISCOVERY_REGEX                        = ".*"
		DEFAULT_DISTRIBUTION_REGEX                     = "^([^\\\\.]+)\\..*$"
		DEFAULT_DISTRIBUTION_REGEX_GROUP_INDEX         = 1
		DEFAULT_DISTRIBUTION_STRATEGY                  = DISTRIBUTION_STRATEGY_REGEX
		DEFAULT_TOPIC_PINNING_REDIS_ADDRESSES          = "redis:6379"
		DEFAULT_TOPIC_PINNING_REDIS_DB_NO              = 0
		DEFAULT_TOPIC_PINNING_REDIS_DB_PASSWORD        = ""
		DEFAULT_TOPIC_PINNING_REDIS_CLUSTER_NAME       = ""
		DEFAULT_TOPIC_PINNING_HASH_SLIDING_EXPIRY_MS   = 3600000
		DEFAULT_TOPICS_DISCOVERY_INTERVAL              = 1800000
		DEFAULT_LOGGING_FORMAT                         = ""
		DEFAULT_LOG_LEVEL                              = LogLevel_VERBOSE
		DEFAULT_KAFKA_CONSUMER_SERVER_HOST             = "kafka:9092"
		DEFAULT_KAFKA_PRODUCER_SERVER_HOST             = "kafka:9092"
		DEFAULT_KAFKA_CONSUMER_GROUP_ID                = "kafka-postman"
		DEFAULT_KAFKA_CONSUMER_DEFAULT_OFFSET          = KAFKA_DEFAULT_OFFSET_END
		DEFAULT_DISCOVERY_TOPICS_TOPIC                 = "consumers"
		DEFAULT_SOURCE_TOPIC                           = "metrics"
		DEFAULT_DEFAULT_TARGET_TOPIC                   = "_unknown_recipient"
		DEFAULT_AUTO_CREATE_MISSING_TOPICS             = "true"
		DEFAULT_TOPIC_PINNING_ENABLED                  = "true"
		DEFAULT_TOPIC_PINNING_REGEX                    = "^([^\\\\.]+)\\..*$"
		DEFAULT_TOPIC_PINNING_REGEX_GROUPS_INDEXES     = "0"

	)
	var (
		DEFAULT_KAFKA_CONSUMER_CLIENT_ID        = clientId.String()
		DEFAULT_KAFKA_PRODUCER_CLIENT_ID        = clientId.String()
	)

	return Config{
		LoggingFormat:                      autoSelectConfig(DEFAULT_LOGGING_FORMAT, LOGGING_FORMAT, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		LogLevel:                           LogLevel(autoSelectConfig(strconv.FormatInt(DEFAULT_LOG_LEVEL, 10), LOG_LEVEL, []ConfigValidationMode{ConfigValidationMode_IS_INT}, ConfigParamValueType_INT, []string{}).ValueInt),
		KafkaConsumerClientId:              autoSelectConfig(DEFAULT_KAFKA_CONSUMER_CLIENT_ID, KAFKA_CONSUMER_CLIENT_ID, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		KafkaConsumerServerHost:            autoSelectConfig(DEFAULT_KAFKA_CONSUMER_SERVER_HOST, KAFKA_CONSUMER_SERVER_HOST, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		KafkaProducerClientId:              autoSelectConfig(DEFAULT_KAFKA_PRODUCER_CLIENT_ID, KAFKA_PRODUCER_CLIENT_ID, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		KafkaConsumerGroupId:               autoSelectConfig(DEFAULT_KAFKA_CONSUMER_GROUP_ID, KAFKA_CONSUMER_GROUP_ID, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		KafkaConsumerDefaultOffset:         autoSelectConfig(DEFAULT_KAFKA_CONSUMER_DEFAULT_OFFSET, KAFKA_CONSUMER_DEFAULT_OFFSET, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_LIST_BASED}, ConfigParamValueType_STRING, KAFKA_DEFAULT_OFFSET_OPTIONS).ValueString,
		KafkaProducerServerHost:            autoSelectConfig(DEFAULT_KAFKA_PRODUCER_SERVER_HOST, KAFKA_PRODUCER_SERVER_HOST, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		AutoCreateMissingTopics:            bool(autoSelectConfig(DEFAULT_AUTO_CREATE_MISSING_TOPICS, AUTO_CREATE_MISSING_TOPICS, []ConfigValidationMode{ConfigValidationMode_IS_BOOL}, ConfigParamValueType_BOOL, []string{}).ValueBool),
		TopicPinningRegex:                  autoSelectConfig(DEFAULT_TOPIC_PINNING_REGEX, TOPIC_PINNING_REGEX, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_REGEX}, ConfigParamValueType_REGEX, []string{}).ValueRegex,
		TopicPinningRegexGroupIndexes:      parseIntArrFromString(autoSelectConfig(DEFAULT_TOPIC_PINNING_REGEX_GROUPS_INDEXES, TOPIC_PINNING_REGEX_GROUPS_INDEXES, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_REGEX_MATCH_AND}, ConfigParamValueType_STRING, []string{"^[0-9,]+$"}).ValueString, ","),
		DiscoveryMethod:                    autoSelectConfig(DEFAULT_DISCOVERY_METHOD, DISCOVERY_METHOD, []ConfigValidationMode{ConfigValidationMode_LIST_BASED}, ConfigParamValueType_STRING, TOPICS_DISCOVERY_METHODS).ValueString,
		DiscoveryManualTopicsList:          autoSelectConfig(DEFAULT_DISCOVERY_MANUAL_TOPICS_LIST, DISCOVERY_MANUAL_TOPICS_LIST, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		DiscoveryManualTopicsListSeparator: autoSelectConfig(DEFAULT_DISCOVERY_MANUAL_TOPICS_LIST_SEPARATOR, DISCOVERY_MANUAL_TOPICS_LIST_SEPARATOR, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		DiscoveryRegex:                     autoSelectConfig(DEFAULT_DISCOVERY_REGEX, DISCOVERY_REGEX, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_REGEX}, ConfigParamValueType_REGEX, []string{}).ValueRegex,
		DiscoveryTopicsTopic:               autoSelectConfig(DEFAULT_DISCOVERY_TOPICS_TOPIC, DISCOVERY_TOPICS_TOPIC, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		DiscoveryIntervalMs:                int64(autoSelectConfig(strconv.FormatInt(DEFAULT_TOPICS_DISCOVERY_INTERVAL, 10), DISCOVERY_INTERVAL, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_INT}, ConfigParamValueType_INT, []string{}).ValueInt),
		DistributionStrategy:               autoSelectConfig(DEFAULT_DISTRIBUTION_STRATEGY, DISTRIBUTION_STRATEGY, []ConfigValidationMode{ConfigValidationMode_LIST_BASED}, ConfigParamValueType_STRING, TOPICS_DISTRIBUTION_STRATEGIES).ValueString,
		DistributionRegex:                  autoSelectConfig(DEFAULT_DISTRIBUTION_REGEX, DISTRIBUTION_REGEX, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_REGEX}, ConfigParamValueType_REGEX, []string{}).ValueRegex,
		DistributionRegexGroupIndex:        int(autoSelectConfig(strconv.FormatInt(DEFAULT_DISTRIBUTION_REGEX_GROUP_INDEX, 10), DISTRIBUTION_REGEX_GROUP_INDEX, []ConfigValidationMode{ConfigValidationMode_IS_INT}, ConfigParamValueType_INT, []string{}).ValueInt),
		TopicPinningEnabled:                bool(autoSelectConfig(DEFAULT_TOPIC_PINNING_ENABLED, TOPIC_PINNING_ENABLED, []ConfigValidationMode{ConfigValidationMode_IS_BOOL}, ConfigParamValueType_BOOL, []string{}).ValueBool),
		TopicPinningRedisAddresses:         strings.Split(autoSelectConfig(DEFAULT_TOPIC_PINNING_REDIS_ADDRESSES, TOPIC_PINNING_REDIS_ADDRESSES, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString, ","),
		TopicPinningRedisDbNo:              int(autoSelectConfig(strconv.FormatInt(DEFAULT_TOPIC_PINNING_REDIS_DB_NO, 10), TOPIC_PINNING_REDIS_DB_NO, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_INT}, ConfigParamValueType_INT, []string{}).ValueInt),
		TopicPinningRedisDbPassword:        autoSelectConfig(DEFAULT_TOPIC_PINNING_REDIS_DB_PASSWORD, TOPIC_PINNING_REDIS_DB_PASSWORD, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		TopicPinningRedisClusterName:       autoSelectConfig(DEFAULT_TOPIC_PINNING_REDIS_CLUSTER_NAME, TOPIC_PINNING_REDIS_CLUSTER_NAME, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		TopicPinningHashSlidingExpiryMs:    int64(autoSelectConfig(strconv.FormatInt(DEFAULT_TOPIC_PINNING_HASH_SLIDING_EXPIRY_MS, 10), TOPIC_PINNING_HASH_SLIDING_EXPIRY_MS, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_INT}, ConfigParamValueType_INT, []string{}).ValueInt),
		SourceTopic:                        autoSelectConfig(DEFAULT_SOURCE_TOPIC, SOURCE_TOPIC, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		DefaultTargetTopic:                 autoSelectConfig(DEFAULT_DEFAULT_TARGET_TOPIC, DEFAULT_TARGET_TOPIC, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
	}
}
func getMessageFingerprintByRegex(regex, message string)  {
//  regexExtracts := regex.Extract(regex, message)
//  messageFingerprint := md5.Hash(regex + strings.Join(regexExtracts))
//  return messageFingerprint
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
func stringInSlice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}
func main() {
	CUR_FUNCTION := "main"
	discoveredTopics := make([]string, 0, 0)
	roundRobinTopicIndex := 0

	LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_INFO, Message: "Kafka-Postman started."})
	lastTopicsDiscoveryTimestamp := int64(0)

	//1. Get configuration
	config := getConfig()
	configJsonBytes, err := json.Marshal(config)
	if err != nil {
		LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_ERROR, Message: fmt.Sprint("ERROR: Configuration is not JSON parsable. CANNOT CONTINUE. (", err, ")")})
		os.Exit(8)
	}
	LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_INFO, Message: fmt.Sprint("This is the loaded config:", string(configJsonBytes))})

	//2. for message_idx, message in kafka.GetMessages(KafkaPostman_SOURCE_TOPIC) {
	LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_INFO, Message:"Connecting to the Kafka..."})
	kafkaConsumer := generateConsumer(config.KafkaConsumerServerHost, config.KafkaConsumerGroupId, config.KafkaConsumerClientId, config.KafkaConsumerDefaultOffset, config.SourceTopic)
	for {
		//3. Get a message from Kafka
		LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, Message:"Starting to wait for messages from kafka..."})
		msg, err := kafkaConsumer.ReadMessage(10 * time.Second)
		if err != nil {
			if err.Error() == kafka.ErrTimedOut.String() {
				LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: err, Level: LogLevel_INFO, Message:"Waiting for a message from Kafka (" + config.KafkaConsumerClientId + "." + config.KafkaConsumerGroupId + "@" + config.KafkaConsumerServerHost + "/" + config.SourceTopic + "@" + config.KafkaConsumerDefaultOffset + ")..."})
			} else {
				// The client will automatically try to recover from all errors.
				LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: err, Level: LogLevel_ERROR, Message: fmt.Sprint("Consumer error: %v (%v)", err, msg)})
			}
		} else {
			msgValue := string(msg.Value)

			//3. (Re-)Discover topics if needed
			if timeSinceLastDiscovery := time.Now().Unix() - lastTopicsDiscoveryTimestamp; timeSinceLastDiscovery > config.DiscoveryIntervalMs || len(discoveredTopics) == 0 {
				LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, Message: "Re-discovering destination topics. (timeSinceLastDiscovery: " + strconv.FormatInt(timeSinceLastDiscovery, 10) + ", config.DiscoveryIntervalMs: " + strconv.FormatInt(config.DiscoveryIntervalMs, 10) + ", len(discoveredTopics): " + strconv.FormatInt(int64(len(discoveredTopics)), 10) + ")"})
				discoveredTopics = discoverTopics(config, kafkaConsumer, discoveredTopics)
				lastTopicsDiscoveryTimestamp = time.Now().Unix()
			}

			//4. Decide on a default destination topic (based on distribution strategy)
			LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, Message: "Getting the default destination topic based on the distribution strategy..."})
			//TODO: Must check that the destination topic is not our source topic (if the consumer and producer have the same address/port)
			defaultDestinationTopic := getDefaultDestinationTopic(config, discoveredTopics, roundRobinTopicIndex, msgValue)
			LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, Message: "Currently, the destination topic is: " + defaultDestinationTopic})

			//5. Handle topic pinning (if enabled)
			if config.TopicPinningEnabled {
				LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, Message: "Handling topic pinning..."})
				defaultDestinationTopic = handleTopicPinning(config, defaultDestinationTopic, msgValue)
				//TODO: By now the defaultDestinationTopic changes to a value that looks like an array with random numbers
				LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, Message: "Currently, the destination topic is: " + defaultDestinationTopic})
			} else {
				LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, Message: "Topic pinning is disabled..."})
			}

			LogForwarder(LogMessage{Caller: CUR_FUNCTION, Level: LogLevel_INFO, Error: nil, Message: fmt.Sprint("Will forward the message '" + string(msg.Value) + "' to topic " + defaultDestinationTopic)})
			//6. kafka.PublishMessage(selected_topic, message)
		}
	}
}

func handleTopicPinning(config Config, defaultDestinationTopic string, msg string) string {
	CUR_FUNCTION := "handleTopicPinning"
	var sender redis.Sender
	var err error

	//TODO: Build and test a program that will test the connectivity and get/set from/to redis before continuing

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

		LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, Message:"Connecting to a single redis (the cluster name is empty)..."})
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

		LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, Message:"Connecting to a redis cluster (the cluster name is not empty)..."})
		sender, err = ClusterRedis(ctx)
	}
	if err != nil {
		LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: err, Level: LogLevel_WARN, Message: "Could not process topic pinning. An error occurred while opening a connection to Redis."})
	} else {
		defer sender.Close()

		LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, Message:"Creating the sync object..."})
		sync := redis.SyncCtx{sender} // wrapper for synchronous api

		//4.1. Create a fingerprint of the message by hashing the text extracted by the regex groups
		if config.LogLevel >= LogLevel_VERBOSE {
			LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, Message:"Successfully connected to redis. Calculating a message fingerprint for the current message... (msg: `" + fmt.Sprintf("%v", msg) + "', config.TopicPinningRegex: " + fmt.Sprintf("%v",config.TopicPinningRegex) + ", config.TopicPinningRegexGroupIndexes: " + fmt.Sprintf("%v", config.TopicPinningRegexGroupIndexes) + ")"})
		} else {
			LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, Message: "Successfully connected to redis. Calculating a message fingerprint for the current message..."})
		}
		hasher := md5.New()
		regexMatches := config.TopicPinningRegex.FindStringSubmatch(msg)
		messageFingerprintRaw :=  make([]string, 0)
		for _, groupIdx := range config.TopicPinningRegexGroupIndexes {
			messageFingerprintRaw = append(messageFingerprintRaw, regexMatches[groupIdx])
		}
		messageFingerprintRawBytes, _ := json.Marshal(messageFingerprintRaw)
		messageFingerprint := hasher.Sum(messageFingerprintRawBytes)
		LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_VERBOSE, Message: "Created the following fingerprint for the current message. (fingerprint: " + fmt.Sprintf("%x", messageFingerprint) + ", message: `" + fmt.Sprintf("%v", msg) + "')"})

		//4.2. Try to find the fingerprint on Redis by using the fingerprint as the key
		LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, Message:"Trying to find the message fingerprint on redis..."})
		res := sync.Do(ctx, "GET", messageFingerprint)
		if err := redis.AsError(res); err == nil && res != nil {
			//4.2.1. The fingerprint was found on Redis:
			LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, Message:"The fingerprint was found on redis. Selecting the topic found as the default topic for the current message."})
			defaultDestinationTopic = fmt.Sprintf("%q", res)
		} else {
			//TODO: Why this never happens?
			LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, Message:"The fingerprint was not found on redis. Selecting the topic found as the default topic so far (by other methods) for the current message."})
		}

		//4.3. Write the value of defaultDestinationTopic to Redis under the current fingerprint to either reset the fingerprint sliding expiration (if already exists) or to cache it (if it's new)
		LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, Message:"Updating the selected topic for the current message fingerprint to either reset the sliding expiration if already exist or to set the fingerprint for other similar messages if not already exist..."})
		res = sync.Do(ctx, "SET", messageFingerprint, defaultDestinationTopic, "PX", config.TopicPinningHashSlidingExpiryMs)
		if err := redis.AsError(res); err != nil {
			LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: err, Level: LogLevel_WARN, Message: "Could not process topic pinning. An error occurred while updating a value to Redis."})
		}
	}
	return defaultDestinationTopic
}

func parseIntArrFromString(intArrString, delimiter string) []int64 {
	res := make([]int64, 0)
	for _, numAsStr := range strings.Split(intArrString, delimiter) {
		numAsInt, err := strconv.ParseInt(numAsStr, 10, 64)
		if err != nil {
			panic("Could not parse the given string array to an int array")
		}
		res = append(res, numAsInt)
	}
	return res
}

func getDefaultDestinationTopic(config Config, discoveredTopics []string, roundRobinTopicIndex int, msg string) string {
	CUR_FUNCTION := "getDefaultDestinationTopic"
	defaultDestinationTopic := config.DefaultTargetTopic
	switch config.DistributionStrategy {
	case DISTRIBUTION_STRATEGY_RANDOM:
		defaultDestinationTopic = discoveredTopics[rand.Intn(len(discoveredTopics))]
	case DISTRIBUTION_STRATEGY_ROUND_ROBIN:
		roundRobinTopicIndex++
		if roundRobinTopicIndex >= len(discoveredTopics) {
			roundRobinTopicIndex = 0
		}
		defaultDestinationTopic = discoveredTopics[roundRobinTopicIndex]
	case DISTRIBUTION_STRATEGY_REGEX:
		regexMatchedGroups := config.DistributionRegex.FindStringSubmatch(msg)
		if config.DistributionRegexGroupIndex < len(regexMatchedGroups) {
			defaultDestinationTopic = regexMatchedGroups[config.DistributionRegexGroupIndex]
		} else {
			LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_WARN, Message:fmt.Sprint("WARN: The requested distribution strategy is regex but the regex group index is", config.DistributionRegexGroupIndex, "which is equal to or greater than the number of groups found by the regex `", config.DistributionRegex, "' on message", msg, "which resulted in this array of matched groups", regexMatchedGroups)})
		}
	}
	return defaultDestinationTopic
}

func discoverTopics(config Config, kafkaConsumer kafka.Consumer, discoveredTopics []string) []string {
	//3. If time_since_last_topics_discovery > KafkaPostman_TOPICS_DISCOVERY_INTERVAL {
	//      switch KafkaPostman_TOPICS_DISCOVERY_METHOD {
	//         case TOPICS_TOPIC:
	//            uniqueMessagesOnly := true
	//            topics = pullAllMessagesFromTopic(KafkaPostman_TOPICS_KAFKA_TOPIC, uniqueMessagesOnly)
	//         case REGEX:
	//            uniqueMessagesOnly := true
	//            topicsRaw := kafka.GetAllTopics()
	//            topicsNew := make([]string, 0)
	//            for _, topic := topicsRaw {
	//               if regex.match(KafkaPostman_TOPICS_DISCOVERY_REGEX, topic) {
	//                  topicsNew = topicsNew.append(topic)
	//               }
	//            }
	//            topics = topicsNew
	//      }
	//   }
    CUR_FUNCTION := "discoverTopics"
	tmpTopicsList := make([]string, 0, 0)
	switch config.DiscoveryMethod {
	case DISCOVERY_METHOD_REGEX:
		tmpTopicsList = discoverTopicsByRegex(kafkaConsumer, config, tmpTopicsList)
	case DISCOVERY_METHOD_MANUAL:
		if len(config.DiscoveryManualTopicsList) > 0 {
			tmpTopicsList = strings.Split(config.DiscoveryManualTopicsList, config.DiscoveryManualTopicsListSeparator)
		}
	case DISCOVERY_METHOD_TOPICS_TOPIC:
		//TODO: Get all topics from the topics topic (kafka-cat?)
	default:
		LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_ERROR, Message:"ERROR: Unknown topics discovery method. LEAVING"})
		os.Exit(7)
	}
	discoveredTopics = tmpTopicsList
	LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_DEBUG, Message:"Discovered the following topics: " + fmt.Sprintf("%v", discoveredTopics)})
	return discoveredTopics
}

func discoverTopicsByRegex(kafkaConsumer kafka.Consumer, config Config, tmpTopicsList []string) []string {
	CUR_FUNCTION := "discoverTopicsByRegex"
	kafkaAdminClient, err := kafka.NewAdminClientFromConsumer(&kafkaConsumer)
	if err == nil {
		topicsInfo, topicsCollectionErr := kafkaAdminClient.GetMetadata(nil, true, -1)
		if topicsCollectionErr == nil {
			for _, topic := range topicsInfo.Topics {
				if config.DiscoveryRegex.Match([]byte(topic.Topic)) {
					tmpTopicsList = append(tmpTopicsList, topic.Topic)
				}
			}
		} else {
			LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_ERROR, Message:fmt.Sprint("ERROR: Failed to get a list of all topics on this Kafka cluster. (The following error was thrown from kafkaAdminClient.GetMetadata). ", topicsCollectionErr)})
			os.Exit(6)
		}
	} else {
		LogForwarder(LogMessage{Caller: CUR_FUNCTION, Error: nil, Level: LogLevel_ERROR, Message:fmt.Sprint("ERROR: Failed to get a list of all topics on this Kafka cluster. (The following error was thrown from kafkaAdminClient.GetMetadata). ", err)})
		os.Exit(6)
	}
	return tmpTopicsList
}
