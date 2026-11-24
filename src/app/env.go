package app

import (
	"os"
	"strconv"
)

// Read a string env var, fall back to the default when it's not there
func GetEnvString(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// Same thing but for integers — garbage values get scolded in the log, then we use the default anyway
func GetEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		Logger.Errorf("Invalid integer value for %s, using default value: %d", key, defaultValue)
	}
	return defaultValue
}

// Boolean variant of the above, same scolding rules apply
func GetEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		returnValue, err := strconv.ParseBool(value)
		if err != nil {
			Logger.Errorf("Invalid boolean value for %s, using default value: %v", key, defaultValue)
			return defaultValue
		}
		return returnValue
	}
	return defaultValue
}
