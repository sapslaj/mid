package types

import (
	"maps"
	"reflect"
	"strings"
	"time"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/go/common/resource"
)

// DiffAttribute performs deep equality on two values and returns an Update
// PropertyDiff if they are different, but nil if they are equal.
func DiffAttribute(o any, n any) *p.PropertyDiff {
	if !resource.NewPropertyValue(o).DeepEquals(resource.NewPropertyValue(n)) {
		return &p.PropertyDiff{
			Kind:      p.Update,
			InputDiff: true,
		}
	}
	return nil
}

// DiffAttributes performs deep equality on two structs only for the attributes
// provided and returns a DiffResponse
func DiffAttributes(olds any, news any, attributes []string) p.DiffResponse {
	diff := p.DiffResponse{
		HasChanges:   false,
		DetailedDiff: map[string]p.PropertyDiff{},
	}

	oldVal := reflect.ValueOf(olds)
	newVal := reflect.ValueOf(news)
	for _, attribute := range attributes {
		for i := range oldVal.NumField() {
			field := oldVal.Type().Field(i)
			parts := strings.Split(field.Tag.Get("pulumi"), ",")
			if len(parts) == 0 {
				continue
			}
			if parts[0] != attribute {
				continue
			}
			propertyDiff := DiffAttribute(
				oldVal.FieldByName(field.Name).Interface(),
				newVal.FieldByName(field.Name).Interface(),
			)
			if propertyDiff == nil {
				continue
			}
			diff.HasChanges = true
			diff.DetailedDiff[attribute] = *propertyDiff
			break
		}
	}
	return diff
}

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

// MergeDiffResponses will merge an arbitrary number of DiffResponses together
// with the last taking the highest precedence. Any DiffResponse that has
// `HasChanges` or `DeleteBeforeReplace` set will result in the returned
// DiffResponse to have those set as well.
func MergeDiffResponses(drs ...p.DiffResponse) p.DiffResponse {
	diff := p.DiffResponse{
		HasChanges:   false,
		DetailedDiff: map[string]p.PropertyDiff{},
	}
	for _, dr := range drs {
		if dr.HasChanges {
			diff.HasChanges = true
		}
		if dr.DeleteBeforeReplace {
			diff.DeleteBeforeReplace = true
		}
		maps.Copy(diff.DetailedDiff, dr.DetailedDiff)
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
