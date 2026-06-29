package artifacts

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FlexString JSON 可为字符串或字符串数组（模型常把 sfx 输出为数组）。
type FlexString string

// UnmarshalJSON 兼容 string | []string | null。
func (f *FlexString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*f = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexString(s)
		return nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = FlexString(strings.Join(arr, ", "))
		return nil
	}
	return fmt.Errorf("flexString: unsupported JSON %s", string(data))
}

// String 返回普通字符串。
func (f FlexString) String() string {
	return string(f)
}
