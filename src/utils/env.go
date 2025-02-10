package utils

import (
	"os"
	"strconv"
)

func GetIntEnvOrDefault(env string, defaultValue uint32) uint32 {
	strValue := os.Getenv(env)

	if len(strValue) == 0 {
		return defaultValue
	} else {
		uintValue, err := strconv.ParseUint(strValue, 10, 32)
		if err != nil {
			Log.Error(env + " env value cannot be parsed as uint, reset to " + strconv.FormatUint(uint64(defaultValue), 10))
			return defaultValue
		} else {
			return uint32(uintValue)
		}
	}

}
