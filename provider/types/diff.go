package types

import (
	"reflect"
	"strings"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/go/common/resource"
)

func DiffAttribute(o any, n any) *p.PropertyDiff {
	if !resource.NewPropertyValue(o).DeepEquals(resource.NewPropertyValue(n)) {
		return &p.PropertyDiff{
			Kind:      p.Update,
			InputDiff: true,
		}
	}
	return nil
}

func DiffAttributes(olds any, news any, attributes []string) p.DiffResponse {
	diff := p.DiffResponse{
		HasChanges:   false,
		DetailedDiff: map[string]p.PropertyDiff{},
	}

	oldVal := reflect.ValueOf(olds)
	newVal := reflect.ValueOf(news)
	for _, attribute := range attributes {
		for i := 0; i < oldVal.NumField(); i++ {
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
		for key, pd := range dr.DetailedDiff {
			diff.DetailedDiff[key] = pd
		}
	}
	return diff
}
