package common

import (
	"encoding/base64"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

func RequireEnv(name string) string {
	value, specified := os.LookupEnv(name)
	if !specified {
		log.Fatalf("required environment variable \"%v\" hasn't been specified", name)
	}

	return value
}

func RequireStrArrEnv(name string) []string {
	rawValue := RequireEnv(name)
	parts := strings.Split(rawValue, ",")
	var result []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func RequireIntEnv(name string) int {
	rawValue := RequireEnv(name)

	value, err := strconv.Atoi(rawValue)
	if err != nil {
		log.Fatalf("couldn't parse environment variable \"%v\" into an integer", name)
	}

	return value
}

func RequireInt64Env(name string) int64 {
	rawValue := RequireEnv(name)

	value, err := strconv.ParseInt(rawValue, 10, 0)
	if err != nil {
		log.Fatalf("couldn't parse environment variable \"%v\" into an int64", name)
	}

	return value
}
func RequireUint32Env(name string) uint32 {
	intValue := RequireIntEnv(name)

	if intValue < 0 || intValue > math.MaxUint32 {
		log.Fatalf("couldn't parse environment variable \"%v\" into a uint32", name)
	}
	return uint32(intValue)
}
func RequireUint8Env(name string) uint8 {
	intValue := RequireIntEnv(name)

	if intValue < 0 || intValue > math.MaxUint8 {
		log.Fatalf("couldn't parse environment variable \"%v\" into a uint8", name)
	}
	return uint8(intValue)
}
func RequireSecondsEnv(name string) time.Duration {
	return time.Duration(RequireInt64Env(name)) * time.Second
}
func RequireMillisecondsEnv(name string) time.Duration {
	return time.Duration(RequireInt64Env(name)) * time.Millisecond
}

func RequireBoolEnv(name string) bool {
	rawValue := RequireEnv(name)
	lower := strings.ToLower(rawValue)

	switch lower {
	case "true":
		return true
	case "false":
		return false
	}

	log.Fatalf("couldn't parse environment variable \"%v\". \"%v\" is not true or false", name, rawValue)
	return false
}
func RequireBase64Env(name string) []byte {
	rawValue := RequireEnv(name)
	decoded, stdErr := base64.StdEncoding.DecodeString(rawValue)
	if stdErr != nil {
		log.Fatalf("couldn't parse environment variable \"%v\" into base64: %v", name, stdErr)
	}
	return decoded
}

// Mainly used for the CLI commands eg. BENCHMARK_THREAD_COUNT
func OptionalEnv(name string, defaultValue string) string {
	value, specified := os.LookupEnv(name)
	if specified {
		return value
	} else {
		return defaultValue
	}
}

// Mainly used for the CLI commands eg. BENCHMARK_THREAD_COUNT
func OptionalIntEnv(name string, defaultValue int) int {
	_, specified := os.LookupEnv(name)
	if specified {
		return RequireIntEnv(name)
	} else {
		return defaultValue
	}
}

func OptionalBoolEnv(name string, defaultValue bool) bool {
	_, specified := os.LookupEnv(name)
	if specified {
		return RequireBoolEnv(name)
	} else {
		return defaultValue
	}
}
func OptionalBase64Env(name string, defaultValue []byte) []byte {
	_, specified := os.LookupEnv(name)
	if specified {
		return RequireBase64Env(name)
	} else {
		return defaultValue
	}
}
