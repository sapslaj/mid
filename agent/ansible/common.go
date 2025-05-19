package ansible

// These should always be returned by any module
type AnsibleCommonReturns struct {
	Changed *bool `json:"changed,omitempty"`
	Diff    *any  `json:"diff,omitempty"`
}

func (returns *AnsibleCommonReturns) IsChanged() bool {
	changed := returns.Changed != nil && *returns.Changed
	hasDiff := returns.Diff != nil
	return changed || hasDiff
}
