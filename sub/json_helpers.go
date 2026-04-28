package sub

import (
	"encoding/json"

	"github.com/superaddmin/SuperXray-gui/v2/logger"
)

func decodeJSONString(data string, v any, context string) bool {
	if err := json.Unmarshal([]byte(data), v); err != nil {
		logger.Warningf("failed to parse %s JSON: %v", context, err)
		return false
	}
	return true
}
