package util

import (
	"mempool/pkg/constants"
	"os"
)

func DevelopmentEnvironment() bool {
	return os.Getenv(constants.ENV_DEBUG_ENVIRONMENT) == "true"
}
