package operation_setting

import (
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

const LogMaxRangeDaysOptionKey = "log_setting.max_range_days"
const ClientErrorMaskOptionKey = "ClientErrorMaskPatterns"

type ClientErrorPatterns struct {
	source string
	rules  []*regexp.Regexp
}

var clientErrorPatterns atomic.Pointer[ClientErrorPatterns]

func CompileClientErrorPatterns(source string) (*ClientErrorPatterns, error) {
	if current := clientErrorPatterns.Load(); current != nil && current.source == source {
		return current, nil
	}
	patterns := &ClientErrorPatterns{source: source}
	line := 0
	for pattern := range strings.SplitSeq(source, "\n") {
		line++
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		rule, err := regexp.Compile(pattern)
		if err != nil {
			return nil, common.NewMessage("Invalid regular expression on line {{line}}", map[string]any{"line": line})
		}
		patterns.rules = append(patterns.rules, rule)
	}
	return patterns, nil
}

func (p *ClientErrorPatterns) Apply() {
	clientErrorPatterns.Store(p)
}

func MatchesClientError(message string) bool {
	if patterns := clientErrorPatterns.Load(); patterns != nil {
		for _, rule := range patterns.rules {
			if rule.MatchString(message) {
				return true
			}
		}
	}
	return false
}

// LogSetting limits how far regular users can reach into usage logs.
type LogSetting struct {
	// UserExportEnabled lets regular users export their own usage logs.
	// Administrators can always export.
	UserExportEnabled bool `json:"user_export_enabled"`
	// MaxRangeDays is the longest time range a user can search in their logs
	// and dashboard, and the longest range any export can cover.
	MaxRangeDays int `json:"max_range_days"`
}

var logSetting = LogSetting{
	UserExportEnabled: true,
	MaxRangeDays:      365,
}

func init() {
	config.GlobalConfig.Register("log_setting", &logSetting)
}

func GetLogSetting() *LogSetting {
	return &logSetting
}

func (s *LogSetting) MaxRangeSeconds() int64 {
	return int64(max(s.MaxRangeDays, 1)) * 24 * 60 * 60
}

func ValidateLogMaxRangeDays(value string) error {
	days, err := strconv.Atoi(value)
	if err != nil || days < 1 {
		return common.NewMessage("The longest log time range must be at least 1 day")
	}
	return nil
}
