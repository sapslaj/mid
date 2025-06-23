package ansible

// These should always be returned by any module
type AnsibleCommonReturns struct {
	Changed bool    `json:"changed"`
	Failed  bool    `json:"failed"`
	Msg     *string `json:"msg,omitempty"`
	Diff    *any    `json:"diff,omitempty"`
}

// Returns true if Changed, false otherwise.
func (returns AnsibleCommonReturns) IsChanged() bool {
	// NOTE: previous versions checked if `returns.Diff` was set or not. Turns
	// out that some modules will _always_ output a diff whether there are
	// changes or not. This results in false positives from this method, so let's
	// not do that.
	return returns.Changed
}

// Returns "msg" if set, empty string if not
func (returns AnsibleCommonReturns) GetMsg() string {
	if returns.Msg == nil {
		return ""
	}
	return *returns.Msg
}
