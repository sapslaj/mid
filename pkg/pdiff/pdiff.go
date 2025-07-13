package pdiff

import (
	"maps"
	"reflect"
	"slices"
	"strings"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/contract"

	"github.com/sapslaj/mid/pkg/pulumi-go-provider/ende"
	"github.com/sapslaj/mid/pkg/pulumi-go-provider/introspect"
)

// DiffAttributes performs deep equality on two structs only for the attributes
// provided and returns a DiffResponse
func DiffAttributes(inputs any, state any, attributes []string) p.DiffResponse {
	inputProps, err := introspect.FindProperties(reflect.TypeOf(inputs))
	contract.AssertNoErrorf(err, "could not get properties")

	encoder := &ende.Encoder{}

	inputsMap, err := encoder.Encode(inputs)
	contract.AssertNoErrorf(err, "could not diff inputs")

	stateMap, err := encoder.Encode(state)
	contract.AssertNoErrorf(err, "could not diff state")

	// Olds is an Output, but news is an Input. Output should be a superset of
	// Input, so we need to filter out fields that are in Output but not Input.
	oldInputsMap := resource.PropertyMap{}
	for k := range inputsMap {
		oldInputsMap[k] = stateMap[k]
	}

	objDiff := oldInputsMap.Diff(inputsMap)
	pluginDiff := plugin.NewDetailedDiffFromObjectDiff(objDiff, true)
	diff := map[string]p.PropertyDiff{}

	for k, v := range pluginDiff {
		selected := false
		for i := range attributes {
			// FIXME: this has the potential to be too greedy and match things it
			// shouldn't. It should be breaking down the components of the key and
			// ensuring there is a full match for all the components but i ain't got
			// time to write that.
			if strings.HasPrefix(k, attributes[i]) {
				selected = true
			}
		}
		if !selected {
			continue
		}

		set := func(kind p.DiffKind) {
			diff[k] = p.PropertyDiff{
				Kind:      kind,
				InputDiff: v.InputDiff,
			}
		}

		fieldTag := inputProps[k]
		if fieldTag.ReplaceOnChanges {
			v.Kind = v.Kind.AsReplace()
		}

		switch v.Kind {
		case plugin.DiffAdd:
			set(p.Add)
		case plugin.DiffAddReplace:
			set(p.AddReplace)
		case plugin.DiffDelete:
			set(p.Delete)
		case plugin.DiffDeleteReplace:
			set(p.DeleteReplace)
		case plugin.DiffUpdate:
			set(p.Update)
		case plugin.DiffUpdateReplace:
			set(p.UpdateReplace)
		}
	}
	return p.DiffResponse{
		HasChanges:   len(diff) > 0,
		DetailedDiff: diff,
	}
}

// DiffAllAttributesExcept performs a deep equality on two structs for all
// attributes except the list provided and returns a diff response.
func DiffAllAttributesExcept(inputs any, state any, exceptAttributes []string) p.DiffResponse {
	inputProps, err := introspect.FindProperties(reflect.TypeOf(inputs))
	contract.AssertNoErrorf(err, "could not get properties")

	attributes := []string{}
	for prop := range inputProps {
		if !slices.Contains(exceptAttributes, prop) {
			attributes = append(attributes, prop)
		}
	}

	return DiffAttributes(inputs, state, attributes)
}

// DiffAllAttributes performs a deep equality on two structs for all attributes
func DiffAllAttributes(inputs any, state any) p.DiffResponse {
	return DiffAllAttributesExcept(inputs, state, []string{})
}

// ForceDiffReplace changes all of the properties in the DiffResponse to be the
// "-replace" equivalent to trigger a resource replacement.
func ForceDiffReplace(diff p.DiffResponse) p.DiffResponse {
	for key, propdiff := range diff.DetailedDiff {
		switch propdiff.Kind {
		case p.Add:
			propdiff.Kind = p.AddReplace
		case p.Delete:
			propdiff.Kind = p.DeleteReplace
		case p.Update:
			propdiff.Kind = p.UpdateReplace
		}
		diff.DetailedDiff[key] = propdiff
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
