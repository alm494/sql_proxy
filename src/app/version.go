package app

// Versions stamped into the binary by Makefile (-ldflags). 
// ApiVersion is what clients must send in API-Version header
var (
	BuildTime    = "none"
	BuildVersion = "none"
	ApiVersion   = "1.2"
)
