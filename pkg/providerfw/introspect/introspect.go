// Copyright 2022, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package introspect has shared utilities for reflecting.
//
// Introspection is one level up from reflection.
package introspect

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/blang/semver"
	"github.com/hashicorp/go-multierror"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/contract"
)

func StructToMap(i any) map[string]any {
	typ := reflect.TypeOf(i)
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	contract.Assertf(typ.Kind() == reflect.Struct, "Expected a struct. Instead got %s (%v)", typ.Kind(), i)

	m := map[string]any{}
	value := reflect.ValueOf(i)
	for value.Type().Kind() == reflect.Pointer {
		value = value.Elem()
	}
	for _, field := range reflect.VisibleFields(typ) {
		if !field.IsExported() {
			continue
		}

		tag, has := field.Tag.Lookup("pulumi")
		if !has {
			continue
		}

		pulumiArray := strings.Split(tag, ",")
		name := pulumiArray[0]

		m[name] = value.FieldByIndex(field.Index).Interface()
	}
	return m
}

type ToPropertiesOptions struct {
	ComputedKeys []string
}

func FindProperties(typ reflect.Type) (map[string]FieldTag, error) {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	contract.Assertf(typ.Kind() == reflect.Struct, "Expected struct, found %s (%s)", typ.Kind(), typ.String())
	m := map[string]FieldTag{}
	for _, f := range reflect.VisibleFields(typ) {
		info, err := ParseTag(f)
		if err != nil {
			return nil, err
		}
		if info.Internal {
			continue
		}
		m[info.Name] = info
	}
	return m, nil
}

// GetToken calculates the Pulumi token that typ would be projected into.
func GetToken(pkg tokens.Package, typ reflect.Type) (tokens.Type, error) {
	if typ == nil {
		return "", fmt.Errorf("cannot get token of nil type")
	}

	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	name := typ.Name()
	mod := strings.Trim(typ.PkgPath(), "*")

	if name == "" {
		return "", fmt.Errorf("type %s has no name", typ)
	}
	if mod == "" {
		return "", fmt.Errorf("type %s has no module path", typ)
	}
	// Take off the pkg name, since that is supplied by `pkg`.
	mod = mod[strings.LastIndex(mod, "/")+1:]
	if mod == "main" {
		mod = "index"
	}
	m := tokens.NewModuleToken(pkg, tokens.ModuleName(mod))
	tk := tokens.NewTypeToken(m, tokens.TypeName(name))
	return tk, nil
}

// ParseTag gets tag information out of struct tags. It looks under the `pulumi` and
// `provider` tag namespaces.
func ParseTag(field reflect.StructField) (FieldTag, error) {
	pulumiTag, hasPulumiTag := field.Tag.Lookup("pulumi")
	providerTag, hasProviderTag := field.Tag.Lookup("provider")
	if hasProviderTag && !hasPulumiTag {
		return FieldTag{}, fmt.Errorf("`provider` requires a `pulumi` tag")
	}
	if !hasPulumiTag || !field.IsExported() {
		return FieldTag{Internal: true}, nil
	}

	pulumi := map[string]bool{}
	pulumiArray := strings.Split(pulumiTag, ",")
	name := pulumiArray[0]
	for _, item := range pulumiArray[1:] {
		pulumi[item] = true
	}

	var explRef *ExplicitType
	provider := map[string]bool{}
	providerArray := strings.Split(providerTag, ",")
	if hasProviderTag {
		for _, item := range providerArray {
			if strings.HasPrefix(item, "type=") {
				const typeErrMsg = `expected "type=" value of "[pkg@version:]module:name", found "%s"`
				extType := strings.TrimPrefix(item, "type=")
				parts := strings.Split(extType, ":")
				switch len(parts) {
				case 2:
					explRef = &ExplicitType{
						Module: parts[0],
						Name:   parts[1],
					}
				case 3:
					external := strings.Split(parts[0], "@")
					if len(external) != 2 {
						return FieldTag{}, fmt.Errorf(typeErrMsg, extType)
					}
					s, err := semver.ParseTolerant(external[1])
					if err != nil {
						return FieldTag{}, fmt.Errorf(`"type=" version must be valid semver: %w`, err)
					}
					explRef = &ExplicitType{
						Pkg:     external[0],
						Version: "v" + s.String(),
						Module:  parts[1],
						Name:    parts[2],
					}
				default:
					return FieldTag{}, fmt.Errorf(typeErrMsg, extType)
				}
				continue
			}
			provider[item] = true
		}
	}

	// Determine if the provider author had accidentally marked the field as secret
	// in the `pulumi` namespace. We need to error out if this is the case, as it will
	// lead to a Pulumi program runtime panic.
	// pulumi/pulumi-go-provider#192
	if pulumi["secret"] {
		return FieldTag{},
			fmt.Errorf("`marking a field as secret in the `pulumi` tag namespace is not allowed, use `provider` instead")
	}

	return FieldTag{
		Name:             name,
		Optional:         pulumi["optional"],
		Secret:           provider["secret"],
		ReplaceOnChanges: provider["replaceOnChanges"],
		ExplicitRef:      explRef,
	}, nil
}

// ExplicitType is an explicitly specified type ref token.
type ExplicitType struct {
	Pkg     string
	Version string
	Module  string
	Name    string
}

type FieldTag struct {
	Name        string        // The name of the field in the Pulumi type system.
	Optional    bool          // If the field is optional in the Pulumi type system.
	Internal    bool          // If the field should exist in the Pulumi type system.
	Secret      bool          // If the field is secret.
	ExplicitRef *ExplicitType // The name and version of the external type consumed in the field.
	// NOTE: ReplaceOnChanges will only be obeyed when the default diff implementation is used.
	ReplaceOnChanges bool // If changes in the field should force a replacement.
}

func NewFieldMatcher(i any) FieldMatcher {
	v := reflect.ValueOf(i)
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	contract.Assertf(v.Kind() == reflect.Struct, "FieldMatcher must contain a struct, found a %s.", v.Type())
	return FieldMatcher{
		value: v,
	}
}

type FieldMatcher struct {
	value reflect.Value
}

func (f *FieldMatcher) GetField(field any) (FieldTag, bool, error) {
	hostType := f.value.Type()
	for _, i := range reflect.VisibleFields(hostType) {
		f := f.value.FieldByIndex(i.Index)
		fType := hostType.FieldByIndex(i.Index)
		if !fType.IsExported() {
			continue
		}
		if f.Addr().Interface() == field {
			f, err := ParseTag(fType)
			return f, true, err
		}
	}
	return FieldTag{}, false, nil
}

// TargetStructFields returns the set of fields that `t` describes for a given matcher.
//
// If `t` is the struct that the field matcher is based on, return all visible fields on
// the struct. Otherwise `nil, false, nil` is returned.
func (f *FieldMatcher) TargetStructFields(t any) ([]FieldTag, bool, error) {
	v := reflect.ValueOf(t)
	for v.Kind() == reflect.Pointer && !v.IsNil() {
		v = v.Elem()
	}
	if f.value != v {
		return nil, false, nil
	}

	hostType := f.value.Type()
	visableFields := reflect.VisibleFields(hostType)
	fields := []FieldTag{}
	var errs multierror.Error
	for _, idx := range visableFields {
		fType := hostType.FieldByIndex(idx.Index)
		if !fType.IsExported() {
			continue
		}
		tag, err := ParseTag(fType)
		if err != nil {
			errs.Errors = append(errs.Errors, err)
			continue
		}
		if tag.Internal {
			continue
		}
		fields = append(fields, tag)
	}
	return fields, true, errs.ErrorOrNil()
}
