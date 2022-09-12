package util

import (
	"bufio"
	"errors"
	"fmt"
	"hash/crc32"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/outbrain/golib/log"
)

func StringMapAdd(target map[string]string, source map[string]string) {
	for k, v := range source {
		if _, ok := target[k]; ok {
			log.Errorf("key conflict:", k)
		}
		target[k] = v
	}
}

func IntMapAdd(target map[string]string, source map[string]int64) {
	for k, v := range source {
		if _, ok := target[k]; ok {
			log.Errorf("key conflict:", k)
		}
		target[k] = strconv.FormatInt(v, 10)
	}
}

func ConvStrToInt64(s string) int64 {
	if s == "" || s == "NULL" {
		return 0
	}
	value := regexp.MustCompile("\\d+").FindString(s)
	i, err := strconv.ParseInt(value, 10, 64)
	if nil != err {
		log.Errorf("convStrToInt64 err: parse(%v) to int64 err:%v\n", s, err)
		return 0
	}
	return i
}

func ConvStrToInt(s string) int {
	if s == "" || s == "NULL" {
		return 0
	}
	i, err := strconv.Atoi(s)
	if nil != err {
		log.Errorf("ConvStrToInt err: parse(%v) to int err:%v\n", s, err)
		return 0
	}
	return i
}

func ConvStrToUInt(s string) uint {
	if s == "" || s == "NULL" {
		return 0
	}
	i, err := strconv.Atoi(s)
	if nil != err {
		log.Errorf("ConvStrToInt err: parse(%v) to int err:%v\n", s, err)
		return 0
	}
	return uint(i)
}

func ConvStrToFloat(s string) float64 {
	if s == "" || s == "NULL" {
		return 0
	}
	i, err := strconv.ParseFloat(s, 64)
	if nil != err {
		log.Errorf("ConvStrToInt err: parse(%v) to int err:%v\n", s, err)
		return 0
	}
	return i
}

func ConvStrToBool(s string) bool {
	if s == "" || s == "NULL" {
		return false
	}
	i, err := strconv.ParseBool(s)
	if nil != err {
		log.Errorf("ConvStrToBool err: parse(%v) to bool err:%v\n", s, err)
		return false
	}
	return i
}

func CollectAllRowsToArray(keyColName string, values []map[string]string) []string {
	var result []string
	for _, mp := range values {
		mp = ChangeKeyCase(mp)
		result = append(result, mp[keyColName])
	}
	return result
}

func ChangeKeyCase(m map[string]string) map[string]string {
	lowerMap := make(map[string]string)
	for k, v := range m {
		lowerMap[strings.ToLower(k)] = v
	}
	return lowerMap
}

func CollectAllRowsToMap(keyColName string, valueColName string, values []map[string]string) map[string]string {
	result := make(map[string]string)
	for _, mp := range values {
		result[mp[keyColName]] = mp[valueColName]
	}
	return result
}

func CollectAllRowsToPrefixKeyMap(prefixKey string, keyColName string, valueColName string, values []map[string]string) map[string]string {
	result := make(map[string]string)
	for _, mp := range values {
		mp = ChangeKeyCase(mp)
		result[prefixKey+mp[keyColName]] = mp[valueColName]
	}
	return result
}

func CollectFirstRowAsMapValue(key string, valueColName string, values []map[string]string) map[string]string {
	result := make(map[string]string)
	queryResult := values
	if 0 == len(queryResult) {
		log.Info("collectFirstRowAsMapValue:Got nothing from query: ")
		return result
	}
	mp := ChangeKeyCase(queryResult[0])
	if _, ok := mp[valueColName]; !ok {
		log.Info("collectFirstRowAsMapValue:Couldn't get %s from %s\n", valueColName)
		return result
	}
	result[key] = mp[valueColName]
	return result
}

func CollectAllRowsAsMapValue(preKey string, valueColName string, values []map[string]string) map[string]string {
	result := make(map[string]string)
	for i, mp := range values {
		mp = ChangeKeyCase(mp)
		if _, ok := mp[valueColName]; !ok {
			log.Info("collectAllRowsAsMapValue:Couldn't get %s from %s\n", valueColName)
			return result
		}
		result[preKey+strconv.Itoa(i)] = mp[valueColName]
	}
	return result
}

func CollectRowsAsMapValue(preKey string, valueColName string, lines int, values []map[string]string) map[string]string {
	result := make(map[string]string)
	line := 0
	resultMap := values
	for i, mp := range resultMap {
		if i >= lines {
			break
		}
		mp = ChangeKeyCase(mp)
		if _, ok := mp[valueColName]; !ok {
			log.Info("collectRowsAsMapValue: Couldn't get %s from %s\n", valueColName)
			return result
		}
		result[fmt.Sprintf("%s%02d", preKey, i)] = mp[valueColName]
		line++
	}

	for line < lines {
		result[fmt.Sprintf("%s%02d", preKey, line)] = "0"
		line++
	}
	return result
}

func GetCrc32(s string) uint32 {
	return crc32.ChecksumIEEE([]byte(s))
}

func ChangeNullToBlack(s string) string {
	if s == "NULL" {
		s = ""
	}
	return s
}

func GetUserPassword(msg string) string {
	fmt.Printf("请输入%s连接数据库的密码:", msg)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func TakeRandFromList(numberList []int) (int, error) {
	if len(numberList) == 0 {
		return 0, errors.New("param is null")
	}
	i := rand.Intn(len(numberList))
	if i == len(numberList) && i != 0 {
		i = i - 1
	}
	return numberList[i], nil
}
