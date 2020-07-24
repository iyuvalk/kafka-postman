package main

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func tryParseJsonStringArray(jsonString string) (resultArray []string, err error) {
	err = json.Unmarshal([]byte(jsonString), &resultArray)

	return
}

func toRegexArray (regexStringsAsJson string) (regexArr []*regexp.Regexp, error error) {
	stringArr, err := tryParseJsonStringArray(regexStringsAsJson)
	if err == nil {
		error = nil
		for _, regexString := range stringArr {
			curRegex, err := regexp.Compile(regexString)
			if err == nil {
				regexArr = append(regexArr, curRegex)
			} else {
				regexArr = make([]*regexp.Regexp, 0)
				error = err
				return
			}
		}
	} else {
		regexArr = make([]*regexp.Regexp, 0)
		error = err
		return
	}

	return
}

func extractMatches(stringsSlice []string, regexesList []*regexp.Regexp) (result []string) {
	for _, curString := range stringsSlice {
		for _, curRegex := range regexesList {
			if curRegex.Match([]byte(curString)) {
				result = append(result, curString)
				break
			}
		}
	}

	return
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
