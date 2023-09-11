package constant

const REDIS_DEFAULT_DBNUM int = 16
const REDIS_SERVERPORT int = 6379

// REDIS_MAXIDLITIME default client timeout
const REDIS_MAXIDLITIME int = 60 * 5

const REDIS_IOBUF_LEN int = 1024
const REDIS_LOADBUF_LEN int = 1024
const REDIS_STATIC_ARGS int = 4
const REDIS_CONFIGLINE_MAX int = 1024

// REDIS_OBJFREELIST_MAX max number of objects to cache
const REDIS_OBJFREELIST_MAX int = 1000000
const REDIS_MAX_SYNC_TIME int = 60

// event
const AE_SETSIZE int = 1024 * 10

const AE_OK int = 0
const AE_ERR int = -1
const AE_NONE int = 0
const AE_READABLE int = 1
const AE_WRITABLE int = 2

const AE_FILE_EVENTS int = 1
const AE_TIME_EVENTS int = 2
const AE_ALL_EVENTS int = (AE_TIME_EVENTS | AE_FILE_EVENTS)
const AE_DONT_WAIT int = 4
const AE_NOMORE int = -1

// buf
