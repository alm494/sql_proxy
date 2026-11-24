package db

var (
	Handler        DbList         // Pool of all open connections — the beating heart of this whole app
	MaxRows        uint32 = 10000 // Max rows we ever return to a caller; beyond that we cut and flag it
	MaxConnections int    = 100   // Hard ceiling on pooled DBs. Past this point, no (HTTP 429 for you)
	MaxPoolConns   int    = 32    // Open backend connections allowed per pool entry
)
