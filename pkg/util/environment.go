package util

import (
	"kava-challange/pkg/constants"
	"os"
)

func DevelopmentEnvironment() bool {
	return os.Getenv(constants.ENV_DEBUG_ENVIRONMENT) == "true"
}
