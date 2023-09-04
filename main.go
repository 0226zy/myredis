package main

import (
	"fmt"

	"github.com/0226zy/myredis/pkg/config"
)

func main() {

	redisConfig := config.PraseFromFile("redis.conf")
	fmt.Printf("config:%+v\n", redisConfig)
}
