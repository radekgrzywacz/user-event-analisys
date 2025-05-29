package env

import (
	"log"
	"os"
	"strconv"
)

func GetEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}

	return defaultValue
}

func GetEnvString(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		log.Println(value)
		return value
	}
	return defaultValue
}
