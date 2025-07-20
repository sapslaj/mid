package midtypes

import (
	"reflect"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"

	p "github.com/sapslaj/mid/pkg/providerfw"
)

// DiffTriggers extracts the `Triggers` field from two structs and performs a
// diff on them, returning a DiffResponse with Update and/or UpdateReplace
// PropertyDiffs as needed.
func DiffTriggers(olds any, news any) p.DiffResponse {
	diff := p.DiffResponse{
		HasChanges:   false,
		DetailedDiff: map[string]p.PropertyDiff{},
	}
	oldVal := reflect.ValueOf(olds)
	newVal := reflect.ValueOf(news)
	oldTriggers := oldVal.FieldByName("Triggers").Interface().(TriggersOutput)
	newTriggers := newVal.FieldByName("Triggers").Interface().(*TriggersInput)

	if newTriggers != nil {
		refreshDiff := resource.NewPropertyValue(oldTriggers.Refresh).Diff(resource.NewPropertyValue(newTriggers.Refresh))
		if refreshDiff != nil {
			diff.HasChanges = true
			diff.DetailedDiff["triggers"] = p.PropertyDiff{
				Kind:      p.Update,
				InputDiff: true,
			}
		}
		replaceDiff := resource.NewPropertyValue(oldTriggers.Replace).Diff(resource.NewPropertyValue(newTriggers.Replace))
		if replaceDiff != nil {
			diff.HasChanges = true
			diff.DetailedDiff["triggers"] = p.PropertyDiff{
				Kind:      p.UpdateReplace,
				InputDiff: true,
			}
		}
	}

	return diff
}

// UpdateTriggerState copies the replace and refresh triggers from `ins` to
// `outs` and updates `LastChanged` if `changed` is true.
func UpdateTriggerState(outs TriggersOutput, ins *TriggersInput, changed bool) TriggersOutput {
	if ins != nil {
		outs.Replace = ins.Replace
		outs.Refresh = ins.Refresh
	}
	if changed {
		outs.LastChanged = time.Now().UTC().Format(time.RFC3339)
	}
	return outs
}
