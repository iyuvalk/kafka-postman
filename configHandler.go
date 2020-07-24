package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DiscoveryMethod                         string
	DiscoveryRegex                          *regexp.Regexp
	DiscoveryRegexString                    string
	DiscoveryManualTopicsList               string
	DiscoveryManualTopicsListSeparator      string
	DistributionRegex                       *regexp.Regexp
	DistributionRegexString           string
	DistributionRegexGroupIndex       int
	DistributionStrategy              string
	TopicsDistributionRegexGroupIndex int
	TopicPinningEnabled               bool
	TopicPinningRedisAddresses        []string
	TopicPinningRedisDbNo             int
	TopicPinningRedisDbPassword       string
	TopicPinningRedisClusterName      string
	TopicPinningHashSlidingExpiryMs   int64
	DiscoveryIntervalSecs             int64
	LoggingFormat                     string
	LogLevel                          LogLevel
	KafkaConsumerServerHost           string
	KafkaConsumerClientId             string
	KafkaConsumerGroupId              string
	KafkaConsumerDefaultOffset        string
	KafkaProducerServerHost           string
	KafkaProducerClientId             string
	KafkaProducerGroupId              string
	DiscoveryTopicsTopic              string
	DiscoveryTopicsTopicServerHost          string
	DiscoveryTopicsTopicGroupId             string
	DiscoveryTopicsTopicClientId            string
	SourceTopic                             string
	DefaultTargetTopic                      string
	TopicPinningRegex                       *regexp.Regexp
	TopicPinningRegexString                 string
	TopicPinningRegexGroupIndexes           []int64
	AutoDestinationTopicFilteringEnabled    bool
	DiscoveryTopicsTopicMaxWaitForTopics    int64
	DiscoveryTopicsTopicMaxDiscoveryTimeout int64
	TopicsTopicMayContainJson               bool
	TopicsTopicSortByJsonField              string
	TopicsTopicTopicNameJsonField           string
	TopicsTopicSortByJsonFieldAscending     bool
	TopicsValidationWhitelist               []string
	TopicsValidationBlacklist               []string
	TopicsValidationRegexWhitelist          []*regexp.Regexp
	TopicsValidationRegexBlacklist          []*regexp.Regexp
	TopicsValidationValidateAgainstKafka    bool
}

type ConfigParamValue struct {
	Type              ConfigParamValueType
	ValueInt          int64
	ValueBool         bool
	ValueString       string
	ValueRegex        *regexp.Regexp
	ValueStringArray  []string
	ValueRegexArray   []*regexp.Regexp
}

func resolvePlaceholders(s string, instanceId uuid.UUID, timeLaunchedUnix int64) (result string) {
	instanceIdString := instanceId.String()
	timeLaunchedUnixString := fmt.Sprintf("%v", timeLaunchedUnix)

	_, err := regexp.Match("^\\$\\{[^ ]+\\}$", []byte(s))
	if err == nil {
		//Extract env var here
		result = os.Getenv(s[len("${"):len(s) - len("}")])
	}
	result = strings.Replace(s, "{{#instance_id}}", instanceIdString, -1)
	result = strings.Replace(result, "{{#time_started}}", timeLaunchedUnixString, -1)
	result = strings.Replace(result, "{{#uuid}}", uuid.Must(uuid.NewRandom()).String(), -1)

	return result
}

func getConfig() Config {
	instanceId := uuid.Must(uuid.NewRandom())
	timeLaunched := time.Now().Unix()

	return Config{
		LoggingFormat:                           autoSelectConfig(DEFAULT_LOGGING_FORMAT, LOGGING_FORMAT, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		LogLevel:                                LogLevel(autoSelectConfig(strconv.FormatInt(DEFAULT_LOG_LEVEL, 10), LOG_LEVEL, []ConfigValidationMode{ConfigValidationMode_IS_INT}, ConfigParamValueType_INT, []string{}).ValueInt),
		KafkaConsumerServerHost:                 autoSelectConfig(DEFAULT_KAFKA_CONSUMER_SERVER_HOST, KAFKA_CONSUMER_SERVER_HOST, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		KafkaConsumerClientId:                   resolvePlaceholders(autoSelectConfig(DEFAULT_KAFKA_CONSUMER_CLIENT_ID, KAFKA_CONSUMER_CLIENT_ID, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString, instanceId, timeLaunched),
		KafkaConsumerGroupId:                    resolvePlaceholders(autoSelectConfig(DEFAULT_KAFKA_CONSUMER_GROUP_ID, KAFKA_CONSUMER_GROUP_ID, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString, instanceId, timeLaunched),
		KafkaConsumerDefaultOffset:              autoSelectConfig(DEFAULT_KAFKA_CONSUMER_DEFAULT_OFFSET, KAFKA_CONSUMER_DEFAULT_OFFSET, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_LIST_BASED}, ConfigParamValueType_STRING, KAFKA_DEFAULT_OFFSET_OPTIONS).ValueString,
		KafkaProducerServerHost:                 autoSelectConfig(DEFAULT_KAFKA_PRODUCER_SERVER_HOST, KAFKA_PRODUCER_SERVER_HOST, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		KafkaProducerClientId:                   resolvePlaceholders(autoSelectConfig(DEFAULT_KAFKA_PRODUCER_CLIENT_ID, KAFKA_PRODUCER_CLIENT_ID, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString, instanceId, timeLaunched),
		KafkaProducerGroupId:                    resolvePlaceholders(autoSelectConfig(DEFAULT_KAFKA_PRODUCER_GROUP_ID, KAFKA_PRODUCER_GROUP_ID, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString, instanceId, timeLaunched),
		TopicPinningRegex:                       autoSelectConfig(DEFAULT_TOPIC_PINNING_REGEX, TOPIC_PINNING_REGEX, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_REGEX}, ConfigParamValueType_REGEX, []string{}).ValueRegex,
		TopicPinningRegexString:                 autoSelectConfig(DEFAULT_TOPIC_PINNING_REGEX, TOPIC_PINNING_REGEX, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_REGEX}, ConfigParamValueType_STRING, []string{}).ValueString,
		TopicPinningRegexGroupIndexes:           parseIntArrFromString(autoSelectConfig(DEFAULT_TOPIC_PINNING_REGEX_GROUPS_INDEXES, TOPIC_PINNING_REGEX_GROUPS_INDEXES, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_REGEX_MATCH_AND}, ConfigParamValueType_STRING, []string{"^[0-9,]+$"}).ValueString, ","),
		DiscoveryMethod:                         autoSelectConfig(DEFAULT_DISCOVERY_METHOD, DISCOVERY_METHOD, []ConfigValidationMode{ConfigValidationMode_LIST_BASED}, ConfigParamValueType_STRING, TOPICS_DISCOVERY_METHODS).ValueString,
		DiscoveryManualTopicsList:               autoSelectConfig(DEFAULT_DISCOVERY_MANUAL_TOPICS_LIST, DISCOVERY_MANUAL_TOPICS_LIST, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		DiscoveryManualTopicsListSeparator:      autoSelectConfig(DEFAULT_DISCOVERY_MANUAL_TOPICS_LIST_SEPARATOR, DISCOVERY_MANUAL_TOPICS_LIST_SEPARATOR, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		DiscoveryRegex:                          autoSelectConfig(DEFAULT_DISCOVERY_REGEX, DISCOVERY_REGEX, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_REGEX}, ConfigParamValueType_REGEX, []string{}).ValueRegex,
		DiscoveryRegexString:                    autoSelectConfig(DEFAULT_DISCOVERY_REGEX, DISCOVERY_REGEX, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_REGEX}, ConfigParamValueType_STRING, []string{}).ValueString,
		DiscoveryTopicsTopic:                    autoSelectConfig(DEFAULT_DISCOVERY_TOPICS_TOPIC, DISCOVERY_TOPICS_TOPIC, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		DiscoveryIntervalSecs:                   int64(autoSelectConfig(strconv.FormatInt(DEFAULT_TOPICS_DISCOVERY_INTERVAL, 10), DISCOVERY_INTERVAL, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_INT}, ConfigParamValueType_INT, []string{}).ValueInt),
		DistributionStrategy:                    autoSelectConfig(DEFAULT_DISTRIBUTION_STRATEGY, DISTRIBUTION_STRATEGY, []ConfigValidationMode{ConfigValidationMode_LIST_BASED}, ConfigParamValueType_STRING, TOPICS_DISTRIBUTION_STRATEGIES).ValueString,
		DistributionRegex:                       autoSelectConfig(DEFAULT_DISTRIBUTION_REGEX, DISTRIBUTION_REGEX, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_REGEX}, ConfigParamValueType_REGEX, []string{}).ValueRegex,
		DistributionRegexString:                 autoSelectConfig(DEFAULT_DISTRIBUTION_REGEX, DISTRIBUTION_REGEX, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_REGEX}, ConfigParamValueType_STRING, []string{}).ValueString,
		DistributionRegexGroupIndex:             int(autoSelectConfig(strconv.FormatInt(DEFAULT_DISTRIBUTION_REGEX_GROUP_INDEX, 10), DISTRIBUTION_REGEX_GROUP_INDEX, []ConfigValidationMode{ConfigValidationMode_IS_INT}, ConfigParamValueType_INT, []string{}).ValueInt),
		TopicPinningEnabled:                     bool(autoSelectConfig(DEFAULT_TOPIC_PINNING_ENABLED, TOPIC_PINNING_ENABLED, []ConfigValidationMode{ConfigValidationMode_IS_BOOL}, ConfigParamValueType_BOOL, []string{}).ValueBool),
		TopicPinningRedisAddresses:              autoSelectConfig(DEFAULT_TOPIC_PINNING_REDIS_ADDRESSES, TOPIC_PINNING_REDIS_ADDRESSES, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_JSON_STRING_ARRAY}, ConfigParamValueType_STRING_ARRAY, []string{}).ValueStringArray,
		TopicPinningRedisDbNo:                   int(autoSelectConfig(strconv.FormatInt(DEFAULT_TOPIC_PINNING_REDIS_DB_NO, 10), TOPIC_PINNING_REDIS_DB_NO, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_INT}, ConfigParamValueType_INT, []string{}).ValueInt),
		TopicPinningRedisDbPassword:             autoSelectConfig(DEFAULT_TOPIC_PINNING_REDIS_DB_PASSWORD, TOPIC_PINNING_REDIS_DB_PASSWORD, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		TopicPinningRedisClusterName:            autoSelectConfig(DEFAULT_TOPIC_PINNING_REDIS_CLUSTER_NAME, TOPIC_PINNING_REDIS_CLUSTER_NAME, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		TopicPinningHashSlidingExpiryMs:         int64(autoSelectConfig(strconv.FormatInt(DEFAULT_TOPIC_PINNING_HASH_SLIDING_EXPIRY_MS, 10), TOPIC_PINNING_HASH_SLIDING_EXPIRY_MS, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_INT}, ConfigParamValueType_INT, []string{}).ValueInt),
		SourceTopic:                             autoSelectConfig(DEFAULT_SOURCE_TOPIC, SOURCE_TOPIC, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		DefaultTargetTopic:                      autoSelectConfig(DEFAULT_DEFAULT_TARGET_TOPIC, DEFAULT_TARGET_TOPIC, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		AutoDestinationTopicFilteringEnabled:    autoSelectConfig(DEFAULT_AUTO_DESTINATION_TOPIC_FILTERING_ENABLED, AUTO_DESTINATION_TOPIC_FILTERING_ENABLED, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_BOOL}, ConfigParamValueType_BOOL, []string{}).ValueBool,
		DiscoveryTopicsTopicServerHost:          autoSelectConfig(DEFAULT_DISCOVERY_TOPICS_TOPIC_SERVER_HOST, DISCOVERY_TOPICS_TOPIC_SERVER_HOST, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		DiscoveryTopicsTopicGroupId:             resolvePlaceholders(autoSelectConfig(DEFAULT_DISCOVERY_TOPICS_TOPIC_GROUP_ID, DISCOVERY_TOPICS_TOPIC_GROUP_ID, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString, instanceId, timeLaunched),
		DiscoveryTopicsTopicClientId:            resolvePlaceholders(autoSelectConfig(DEFAULT_DISCOVERY_TOPICS_TOPIC_CLIENT_ID, DISCOVERY_TOPICS_TOPIC_CLIENT_ID, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString, instanceId, timeLaunched),
		DiscoveryTopicsTopicMaxDiscoveryTimeout: int64(autoSelectConfig(strconv.FormatInt(DEFAULT_DISCOVERY_TOPICS_TOPIC_MAX_DISCO_TIMEOUT, 10), DISCOVERY_TOPICS_TOPIC_MAX_DISCO_TIMEOUT, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_INT}, ConfigParamValueType_INT, []string{}).ValueInt),
		DiscoveryTopicsTopicMaxWaitForTopics:    int64(autoSelectConfig(strconv.FormatInt(DEFAULT_DISCOVERY_TOPICS_TOPIC_MAX_WAIT_FOR_TOPICS, 10), DISCOVERY_TOPICS_TOPIC_MAX_WAIT_FOR_TOPICS, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_INT}, ConfigParamValueType_INT, []string{}).ValueInt),
		TopicsTopicMayContainJson:               bool(autoSelectConfig(DEFAULT_TOPICS_TOPIC_MAY_CONTAIN_JSON, TOPICS_TOPIC_MAY_CONTAIN_JSON, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_BOOL}, ConfigParamValueType_BOOL, []string{}).ValueBool),
		TopicsTopicSortByJsonField:              autoSelectConfig(DEFAULT_TOPICS_TOPIC_SORT_BY_JSON_FIELD, TOPICS_TOPIC_SORT_BY_JSON_FIELD, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		TopicsTopicTopicNameJsonField:           autoSelectConfig(DEFAULT_TOPICS_TOPIC_TOPIC_NAME_JSON_FIELD, TOPICS_TOPIC_TOPIC_NAME_JSON_FIELD, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY}, ConfigParamValueType_STRING, []string{}).ValueString,
		TopicsTopicSortByJsonFieldAscending:     bool(autoSelectConfig(DEFAULT_TOPICS_TOPIC_SORT_BY_JSON_FIELD_ASCENDING, TOPICS_TOPIC_SORT_BY_JSON_FIELD_ASCENDING, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_BOOL}, ConfigParamValueType_BOOL, []string{}).ValueBool),
		TopicsValidationWhitelist:               []string(autoSelectConfig(DEFAULT_TOPICS_VALIDATION_WHITELIST, TOPICS_VALIDATION_WHITELIST, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_JSON_STRING_ARRAY}, ConfigParamValueType_STRING_ARRAY, []string{}).ValueStringArray),
		TopicsValidationBlacklist:               []string(autoSelectConfig(DEFAULT_TOPICS_VALIDATION_BLACKLIST, TOPICS_VALIDATION_BLACKLIST, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_JSON_STRING_ARRAY}, ConfigParamValueType_STRING_ARRAY, []string{}).ValueStringArray),
		TopicsValidationRegexWhitelist:          []*regexp.Regexp(autoSelectConfig(DEFAULT_TOPICS_VALIDATION_REGEX_WHITELIST, TOPICS_VALIDATION_REGEX_WHITELIST, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_JSON_REGEX_ARRAY}, ConfigParamValueType_REGEX_ARRAY, []string{}).ValueRegexArray),
		TopicsValidationRegexBlacklist:          []*regexp.Regexp(autoSelectConfig(DEFAULT_TOPICS_VALIDATION_REGEX_BLACKLIST, TOPICS_VALIDATION_REGEX_BLACKLIST, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_JSON_REGEX_ARRAY}, ConfigParamValueType_REGEX_ARRAY, []string{}).ValueRegexArray),
		TopicsValidationValidateAgainstKafka:    bool(autoSelectConfig(DEFAULT_TOPICS_VALIDATION_VALIDATE_AGAINST_KAFKA, TOPICS_VALIDATION_VALIDATE_AGAINST_KAFKA, []ConfigValidationMode{ConfigValidationMode_IS_EMPTY, ConfigValidationMode_IS_BOOL}, ConfigParamValueType_BOOL, []string{}).ValueBool),
	}
}

func autoSelectConfig(defaultValue string, envVarNameToTest CONFIG_ENV_VARS, configValidationMode []ConfigValidationMode, configParamValueType ConfigParamValueType, stringsList []string) ConfigParamValue {
	CUR_FUNCTION := "autoSelectConfig"
	var rawStringValue string
	var specialResult_StringArray []string
	var specialResult_RegexArray []*regexp.Regexp

	fallingBackToDefaultValue := false
	for _, validationRequested := range configValidationMode {
		if fallingBackToDefaultValue {
			break
		}

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
			allRegexesMatched := true
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
		case ConfigValidationMode_IS_JSON_STRING_ARRAY:
			var stringArr []string
			if json.Unmarshal([]byte(os.Getenv(string(envVarNameToTest))), &stringArr) == nil {
				specialResult_StringArray = stringArr
				rawStringValue = os.Getenv(string(envVarNameToTest))
			} else {
				rawStringValue = defaultValue
				if json.Unmarshal([]byte(defaultValue), &stringArr) != nil {
					panic("ERROR: The default value for `" + string(envVarNameToTest) + "' cannot be parsed as a string array. This is a BUG. For sure!")
				}
				specialResult_StringArray = stringArr
				fallingBackToDefaultValue = true
			}
		case ConfigValidationMode_IS_JSON_REGEX_ARRAY:
			regexArr, err := toRegexArray(os.Getenv(string(envVarNameToTest)))
			if err == nil {
				specialResult_RegexArray = regexArr
			} else {
				fallingBackToDefaultValue = true
				regexArr, err = toRegexArray(defaultValue)
				if err == nil {
					specialResult_RegexArray = regexArr
				} else {
					panic("ERROR: The default value for `" + string(envVarNameToTest) + "' cannot be parsed as a regex array. This is a BUG. For sure!")
				}
			}
		default:
			panic("ERROR: Unknown config validation type.")
		}

		_, environmentVarSet := os.LookupEnv(string(envVarNameToTest))
		if fallingBackToDefaultValue && environmentVarSet {
			LogForwarder(nil, LogMessage{Caller: CUR_FUNCTION, Level: LogLevel_WARN, MessageFormat: "The value of %v is invalid. Using the default value %v"}, envVarNameToTest, defaultValue)
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
	case ConfigParamValueType_STRING_ARRAY:
		return ConfigParamValue{
			Type: ConfigParamValueType_STRING_ARRAY,
			ValueStringArray: specialResult_StringArray,
		}
	case ConfigParamValueType_REGEX_ARRAY:
		return ConfigParamValue{
			Type: ConfigParamValueType_REGEX_ARRAY,
			ValueRegexArray: specialResult_RegexArray,
		}
	default:
		panic("ERROR: Unknown config parameter value type.")
	}
}

