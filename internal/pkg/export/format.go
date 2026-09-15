package export

import (
	"fmt"
	"strconv"
	"time"
)

// FormatValue renders a cell value as a display string. It handles the common
// scalar types produced by database queries and JSON decoding. Unknown types
// fall back to fmt.Sprint.
func FormatValue(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case time.Time:
		if t.IsZero() {
			return ""
		}
		return t.Format("2006-01-02 15:04:05")
	case *time.Time:
		if t == nil || t.IsZero() {
			return ""
		}
		return t.Format("2006-01-02 15:04:05")
	case []byte:
		return string(t)
	case float64:
		return formatFloat(t)
	case float32:
		return formatFloat(float64(t))
	case int, int8, int16, int32, int64:
		return strconv.FormatInt(toInt64(t), 10)
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(toUint64(t), 10)
	default:
		return fmt.Sprint(t)
	}
}

func formatFloat(f float64) string {
	// Render whole numbers without a decimal point, otherwise keep full precision.
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	default:
		return 0
	}
}

func toUint64(v interface{}) uint64 {
	switch n := v.(type) {
	case uint:
		return uint64(n)
	case uint8:
		return uint64(n)
	case uint16:
		return uint64(n)
	case uint32:
		return uint64(n)
	case uint64:
		return n
	default:
		return 0
	}
}
