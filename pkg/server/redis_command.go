package server

type RedisCommand struct {
	Name  string
	Proc  RedisCommandProc
	Arity int
	Flags int

	// vm
	VmPreloadProc RedisCommandProc
	VmFirstKey    int
	VmLastKey     int
	VmKeeStep     int
}

type RedisCommandProc func(client *RedisClient)
