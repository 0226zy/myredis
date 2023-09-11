package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/0226zy/myredis/pkg/constant"
)

// tagToindex tag -> field index
var tagToindex map[string]int

// tagOption tag 自定义赋值函数
var tagOption map[string]option

func init() {
	redisConfig := RedisConfig{}
	v := reflect.ValueOf(redisConfig)
	filedNum := v.NumField()
	tagToindex = map[string]int{}
	for i := 0; i < filedNum; i++ {
		tag := v.Type().Field(i).Tag.Get("conf")
		if tag != "" {
			tagToindex[tag] = i
		}
	}
	tagOption = map[string]option{"save": withSave}

}

// RedisConfig redis.conf
type RedisConfig struct {
	Daemonize bool   `conf:"daemonize"`
	PidFile   string `conf:"pidfile"`
	Port      int    `conf:"port"`
	Bind      string `conf:"bind"`
	Timeout   int    `conf:"timeout"`
	LogLevel  string `conf:"loglevel"`
	LogFile   string `conf:"logfile"`
	DataBases int    `conf:"databases"`

	// save ht db on disk
	Saves          []SaveConf `conf:"save"`
	RDBCompression bool       `conf:"rdbcompression"`
	DBFileName     string     `conf:"dbfilename"`
	Dir            string     `conf:"dir"`

	// replication
	Slaveof    string `conf:"slaveof"`
	MasterAuth string `conf:"masterauth"`

	// security
	RequirePass string `conf:"foobared"`

	// limits
	MaxClients int   `conf:"maxclients"`
	MaxMemory  int64 `conf:"maxmemory"`

	// append only mode
	AppendOnly bool   `conf:"appendonly"`
	AppendSync string `conf:"appendfsync"`
	// virtual memory

	VmEnabled    bool   `conf:"vm-enabled"`
	VmSwapFile   string `conf:"vm-swap-file"`
	VmMaxMemory  int    `conf:"vm-max-memory"`
	VmPageSize   int    `conf:"vm-page-size"`
	VmPages      int    `conf:"vm-pages"`
	VmMaxThreads int    `conf:"vm-max-threads"`

	// advanced config
	GlueOutPutBuf        bool `conf:"glueoutputbuf"`
	ShareObjects         bool `conf:"shareobjects"`
	ShareObjectsPoolSize int  `conf:"shareobjectspoolsize"`
	HashMaxZipMapEntries int  `conf:"hash-max-zipmap-entries"`
	HashMaxZipMapValue   int  `conf:"hash-max-zipmap-value"`
	interconf            string
	DBNum                int
}

// SaveConf 触发备份的配置
type SaveConf struct {
	Seconds int64
	MinKeys int64
}

// newRedisConfig 构建RedisConfig 设置默认值
func newRedisConfig() *RedisConfig {
	return &RedisConfig{
		DBNum: constant.REDIS_DEFAULT_DBNUM,
		Bind:  "127.0.0.1",
	}
}

// ParseFromFile 从文件解析配置
func PraseFromFile(filename string) *RedisConfig {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("read file failed:%v\n", err)
		os.Exit(1)
	}
	return Unmarshal(data)
}

// Unmarshal 解析反射，给配置赋值
func Unmarshal(data []byte) *RedisConfig {

	loaderr := func(lineNum int, line string, err error) {
		fmt.Fprintf(os.Stderr, "\n*** FATAL CONFIG FILE ERROR ***\n")
		fmt.Fprintf(os.Stderr, "Reading the configuration file,at line %d\n", lineNum)
		fmt.Fprintf(os.Stderr, ">>> %s\n", line)
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	redisConfig := newRedisConfig()
	v := reflect.ValueOf(redisConfig)
	v = v.Elem()
	reader := bufio.NewReader(strings.NewReader(string(data)))

	lineNum := 0
	for {
		lineNum++
		line, err := reader.ReadString('\n')

		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("ReadString failed")
			return redisConfig
		}

		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			fmt.Printf("invalid line:%s\n", line)
			os.Exit(-1)
		}

		// find key in tags
		fieldIndex, ok := tagToindex[parts[0]]
		if !ok {
			fmt.Printf("find undefine conf key:%s\n", parts[0])
			loaderr(lineNum, line, errors.New("find undefine conf key:"+parts[0]))
		}
		// get filed by index
		field := v.Field(fieldIndex)
		err = setField(field, parts[0], parts[1])
		if err != nil {
			loaderr(lineNum, line, err)
		}

	}
	return redisConfig
}

func setField(field reflect.Value, tag, value string) error {
	if opt, ok := tagOption[tag]; ok {
		return opt(field, tag, value)
	}
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			fmt.Printf("invalid in key:%s expected int,find:%s\n", tag, value)
			return errors.New("invallid value in key:" + tag + " expected int")
		}
		field.SetInt(intValue)
	case reflect.Bool:
		boolValue, err := parseBool(value)
		if err != nil {
			fmt.Printf("invalid in key:%s expected value,find:%s\n", tag, value)
			return errors.New("invallid value in key:" + tag + " expected bool")
		}
		field.SetBool(boolValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return errors.New("invallid value in key:" + tag + " expected float")
		}
		field.SetFloat(floatValue)
	default:
		fmt.Printf("unknown value tag:%s type:%s\n", tag, field.Type())
		return errors.New("unknown value type in key:%s " + tag + "type")
	}
	return nil

}

func parseBool(value string) (bool, error) {
	switch value {
	case "Yes", "YES", "yes", "1", "t", "T", "true", "TRUE", "True":
		return true, nil
	case "No", "NO", "no", "0", "f", "F", "false", "FALSE", "False":
		return false, nil
	}
	return false, errors.New("invalue syntax")
}

type option func(field reflect.Value, key, value string) error

func withSave(field reflect.Value, key, value string) error {
	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 {
		fmt.Printf("invalid value in key:%s value:%s\n", key, value)
	}

	seconds, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		fmt.Printf("invalid in key:%s expected int,find:%s\n", key, value)
		return errors.New("invalid value type in key " + key + " expected int")
	}
	keys, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		fmt.Printf("invalid in key:%s expected int,find:%s\n", key, value)
		return errors.New("invalid value type in key " + key + " expected int")
	}

	saveConf := SaveConf{Seconds: seconds, MinKeys: keys}
	field.Set(reflect.Append(field, reflect.ValueOf(saveConf)))
	return nil

}
