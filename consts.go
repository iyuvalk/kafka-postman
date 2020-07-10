package main

type CONFIG_ENV_VARS string
const (
	AUTO_CREATE_MISSING_TOPICS                 = "KAFKA_POSTMAN_AUTO_CREATE_MISSING_TOPICS"
	DISCOVERY_METHOD                           = "KAFKA_POSTMAN_DISCOVERY_METHOD"
	DISCOVERY_MANUAL_TOPICS_LIST               = "KAFKA_POSTMAN_DISCOVERY_MANUAL_TOPICS_LIST"
	DISCOVERY_MANUAL_TOPICS_LIST_SEPARATOR     = "KAFKA_POSTMAN_DISCOVERY_MANUAL_TOPICS_LIST_SEPARATOR"
	DISCOVERY_REGEX                            = "KAFKA_POSTMAN_DISCOVERY_REGEX"
	DISTRIBUTION_STRATEGY                      = "KAFKA_POSTMAN_DISTRIBUTION_STRATEGY"
	DISTRIBUTION_REGEX                         = "KAFKA_POSTMAN_DISTRIBUTION_REGEX"
	DISTRIBUTION_REGEX_GROUP_INDEX             = "KAFKA_POSTMAN_DISTRIBUTION_REGEX_GROUP_INDEX"
	TOPIC_PINNING_ENABLED                      = "KAFKA_POSTMAN_TOPIC_PINNING_ENABLED"
	TOPIC_PINNING_REGEX                        = "KAFKA_POSTMAN_TOPIC_PINNING_REGEX"
	TOPIC_PINNING_REGEX_GROUPS_INDEXES         = "KAFKA_POSTMAN_TOPIC_PINNING_REGEX_GROUPS_INDEXES"
	TOPIC_PINNING_REDIS_ADDRESSES              = "KAFKA_POSTMAN_REDIS_ADDRESSES"
	TOPIC_PINNING_REDIS_DB_NO                  = "KAFKA_POSTMAN_REDIS_DB_NO"
	TOPIC_PINNING_REDIS_DB_PASSWORD            = "KAFKA_POSTMAN_REDIS_DB_PASSWORD"
	TOPIC_PINNING_REDIS_CLUSTER_NAME           = "KAFKA_POSTMAN_REDIS_CLUSTER_NAME"
	TOPIC_PINNING_HASH_SLIDING_EXPIRY_MS       = "KAFKA_POSTMAN_TOPIC_PINNING_HASH_SLIDING_EXPIRY_MS"
	DISCOVERY_INTERVAL                         = "KAFKA_POSTMAN_DISCOVERY_INTERVAL"
	LOGGING_FORMAT                             = "KAFKA_POSTMAN_LOGGING_FORMAT"
	LOG_LEVEL                                  = "KAFKA_POSTMAN_LOG_LEVEL"
	KAFKA_CONSUMER_SERVER_HOST                 = "KAFKA_POSTMAN_KAFKA_CONSUMER_SERVER"
	KAFKA_CONSUMER_CLIENT_ID                   = "KAFKA_POSTMAN_KAFKA_CONSUMER_CLIENT_ID"
	KAFKA_CONSUMER_GROUP_ID                    = "KAFKA_POSTMAN_KAFKA_CONSUMER_GROUP_ID"
	KAFKA_CONSUMER_DEFAULT_OFFSET              = "KAFKA_POSTMAN_KAFKA_CONSUMER_DEFAULT_OFFSET"
	KAFKA_PRODUCER_SERVER_HOST                 = "KAFKA_POSTMAN_KAFKA_PRODUCER_SERVER"
	KAFKA_PRODUCER_CLIENT_ID                   = "KAFKA_POSTMAN_KAFKA_PRODUCER_CLIENT_ID"
	KAFKA_PRODUCER_GROUP_ID                    = "KAFKA_POSTMAN_KAFKA_PRODUCER_GROUP_ID"
	DISCOVERY_TOPICS_TOPIC                     = "KAFKA_POSTMAN_TOPICS_DISCOVERY_TOPIC"
	DISCOVERY_TOPICS_TOPIC_SERVER_HOST         = "KAFKA_POSTMAN_TOPICS_DISCOVERY_TOPIC_SERVER"
	DISCOVERY_TOPICS_TOPIC_GROUP_ID            = "KAFKA_POSTMAN_TOPICS_DISCOVERY_TOPIC_GROUP_ID"
	DISCOVERY_TOPICS_TOPIC_CLIENT_ID           = "KAFKA_POSTMAN_TOPICS_DISCOVERY_TOPIC_CLIENT_ID"
	DISCOVERY_TOPICS_TOPIC_MAX_DISCO_TIMEOUT   = "KAFKA_POSTMAN_TOPICS_DISCOVERY_TOPIC_MAX_DISCO_TIMEOUT"
	DISCOVERY_TOPICS_TOPIC_MAX_WAIT_FOR_TOPICS = "KAFKA_POSTMAN_TOPICS_DISCOVERY_TOPIC_MAX_WAIT_FOR_TOPICS"
	SOURCE_TOPIC                               = "KAFKA_POSTMAN_SOURCE_TOPIC"
	DEFAULT_TARGET_TOPIC                       = "KAFKA_POSTMAN_DEFAULT_TARGET_TOPIC"
	AUTO_DESTINATION_TOPIC_FILTERING_ENABLED   = "KAFKA_POSTMAN_AUTO_DESTINATION_TOPIC_FILTERING_ENABLED"
	TOPICS_TOPIC_MAY_CONTAIN_JSON              = "KAFKA_POSTMAN_TOPICS_TOPIC_MAY_CONTAIN_JSON"
	TOPICS_TOPIC_SORT_BY_JSON_FIELD            = "KAFKA_POSTMAN_TOPICS_TOPIC_SORT_BY_JSON_FIELD"
	TOPICS_TOPIC_TOPIC_NAME_JSON_FIELD         = "KAFKA_POSTMAN_TOPICS_TOPIC_TOPIC_NAME_JSON_FIELD"
	TOPICS_TOPIC_SORT_BY_JSON_FIELD_ASCENDING  = "KAFKA_POSTMAN_TOPICS_TOPIC_SORT_BY_JSON_FIELD_ASCENDING"
	TOPICS_VALIDATION_WHITELIST                = "KAFKA_POSTMAN_TOPICS_VALIDATION_WHITELIST"
	TOPICS_VALIDATION_BLACKLIST                = "KAFKA_POSTMAN_TOPICS_VALIDATION_BLACKLIST"
	TOPICS_VALIDATION_REGEX_WHITELIST          = "KAFKA_POSTMAN_TOPICS_VALIDATION_REGEX_WHITELIST"
	TOPICS_VALIDATION_REGEX_BLACKLIST          = "KAFKA_POSTMAN_TOPICS_VALIDATION_REGEX_BLACKLIST"
	TOPICS_VALIDATION_VALIDATE_AGAINST_KAFKA   = "KAFKA_POSTMAN_TOPICS_VALIDATION_VALIDATE_AGAINST_KAFKA"
)

type ConfigValidationMode int
const (
	ConfigValidationMode_LIST_BASED = iota
	ConfigValidationMode_IS_EMPTY
	ConfigValidationMode_IS_INT
	ConfigValidationMode_IS_BOOL
	ConfigValidationMode_IS_REGEX
	ConfigValidationMode_IS_REGEX_MATCH_AND
	ConfigValidationMode_IS_REGEX_MATCH_OR
	ConfigValidationMode_IS_JSON_STRING_ARRAY
	ConfigValidationMode_IS_JSON_REGEX_ARRAY
)

type ConfigParamValueType int
const (
	ConfigParamValueType_INT = iota
	ConfigParamValueType_BOOL
	ConfigParamValueType_STRING
	ConfigParamValueType_REGEX
	ConfigParamValueType_STRING_ARRAY
	ConfigParamValueType_REGEX_ARRAY
)

const (
	DISCOVERY_METHOD_REGEX        = "REGEX"
	DISCOVERY_METHOD_MANUAL       = "MANUAL"
	DISCOVERY_METHOD_TOPICS_TOPIC = "TOPICS_TOPIC"
)
const (
	DISTRIBUTION_STRATEGY_ROUND_ROBIN = "ROUND_ROBIN"
	DISTRIBUTION_STRATEGY_RANDOM      = "RANDOM"
	DISTRIBUTION_STRATEGY_REGEX       = "REGEX"
)
const (
	KAFKA_DEFAULT_OFFSET_SMALLEST  = "smallest"
	KAFKA_DEFAULT_OFFSET_EARLIEST  = "earliest"
	KAFKA_DEFAULT_OFFSET_BEGINNING = "beginning"
	KAFKA_DEFAULT_OFFSET_LARGEST   = "largest"
	KAFKA_DEFAULT_OFFSET_LATEST    = "latest"
	KAFKA_DEFAULT_OFFSET_END       = "end"
)

var TOPICS_DISCOVERY_METHODS       = []string{
	DISCOVERY_METHOD_REGEX,
	DISCOVERY_METHOD_MANUAL,
	DISCOVERY_METHOD_TOPICS_TOPIC,
}

var TOPICS_DISTRIBUTION_STRATEGIES = []string{
	DISTRIBUTION_STRATEGY_ROUND_ROBIN,
	DISTRIBUTION_STRATEGY_RANDOM,
	DISTRIBUTION_STRATEGY_REGEX,
}

var KAFKA_DEFAULT_OFFSET_OPTIONS   = []string{
	KAFKA_DEFAULT_OFFSET_SMALLEST,
	KAFKA_DEFAULT_OFFSET_EARLIEST,
	KAFKA_DEFAULT_OFFSET_BEGINNING,
	KAFKA_DEFAULT_OFFSET_LARGEST,
	KAFKA_DEFAULT_OFFSET_LATEST,
	KAFKA_DEFAULT_OFFSET_END,
}
