package main

import (
	"fmt"

	"os"

	"github.com/0226zy/myredis/pkg/config"
	"github.com/0226zy/myredis/pkg/log"
	"github.com/0226zy/myredis/pkg/server"
)

func main() {

	log.InitRedisLog()

	var redisConfig *config.RedisConfig
	if len(os.Args) == 2 {
		redisConfig = config.PraseFromFile(os.Args[1])
	} else if len(os.Args) > 2 {
		fmt.Fprintf(os.Stderr, "Usage: ./redis-server [path]/to/redis.conf\n")
		os.Exit(1)
	} else {
		fmt.Printf("Warning: no config file specified,using the default config. In order to specify a config file use 'redis-server /path/to/redis.con'")
	}

	version := "0.0.1"
	fmt.Printf("Server started,Myredis versin %s\n", version)
	log.RedisLog(log.REDIS_NOTICE, "Server started, Myredis version %s", version)
	//TODO: daemonize
	redisServer := server.NewRedisServer(redisConfig)
	redisServer.Init()
	redisServer.Serve()
	redisServer.Clear()

}
