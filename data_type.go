// Copyright 2014 DoAT. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation and/or
//    other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED “AS IS” WITHOUT ANY WARRANTIES WHATSOEVER.
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO,
// THE IMPLIED WARRANTIES OF NON INFRINGEMENT, MERCHANTABILITY AND FITNESS FOR A
// PARTICULAR PURPOSE ARE HEREBY DISCLAIMED. IN NO EVENT SHALL DoAT OR CONTRIBUTORS
// BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// // THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
// NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE,
// EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
//
// The views and conclusions contained in the software and documentation are those of
// the authors and should not be interpreted as representing official policies,
// either expressed or implied, of DoAT.

// Package raml contains the parser, validator and types that implement the
// RAML specification, as documented here:
// http://raml.org/spec.html
package raml

// This file contains all of the RAML types.

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"strings"
)

var (
	arrayType   = "array"
	scalarTypes = map[string]bool{
		"string":        true,
		"number":        true,
		"integer":       true,
		"boolean":       true,
		"date-only":     true,
		"time-only":     true,
		"datetime-only": true,
		"datetime":      true,
		"file":          true,

		// comes from Number format
		"int8":   true,
		"int16":  true,
		"int32":  true,
		"int64":  true,
		"int":    true,
		"long":   true,
		"float":  true,
		"double": true,
		"object": true,
	}
)

// Any type, for our convenience
type Any interface{}

// HTTPCode defines an HTTP status code, for extra clarity
type HTTPCode string // e.g. 200

// HTTPHeader defines an HTTP header
type HTTPHeader string // e.g. Content-Length

// Header used in Methods and other types
type Header NamedParameter

// DefinitionParameters defines a map of parameter name at it's value.
// A ResourceType/Trait/SecurityScheme choice contains the name of a
// ResourceType/Trait/SecurityScheme as well as the parameters used to create
// an instance of it.
type DefinitionParameters map[string]interface{}

// DefinitionChoice defines a definition with it's parameters
type DefinitionChoice struct {
	Name string

	// The definitions of resource types and traits MAY contain parameters,
	// whose values MUST be specified when applying the resource type or trait,
	// UNLESS the parameter corresponds to a reserved parameter name, in which
	// case its value is provided by the processing application.
	// Same goes for security schemes.
	Parameters DefinitionParameters
}

// UnmarshalYAML unmarshals a node which MIGHT be a simple string or a map[string]DefinitionParameters
func (dc *DefinitionChoice) UnmarshalYAML(node *yaml.Node) (err error) {
	switch node.Kind {
	case yaml.ScalarNode:
		simpleDefinition := new(string)
		err = node.Decode(simpleDefinition)
		if err != nil {
			break
		}
		dc.Name = *simpleDefinition
		dc.Parameters = nil
	case yaml.MappingNode:
		parameterizedDefinition := make(map[string]DefinitionParameters)
		err = node.Decode(parameterizedDefinition)
		if err != nil {
			break
		}
		//TODO: this does not look like it should work....
		for choice, params := range parameterizedDefinition {
			dc.Name = choice
			dc.Parameters = params
		}
	default:
		err = fmt.Errorf("unmarshalable node kind %s", node.LongTag())
	}

	return err
}

// HasProperties is interface of all objects that
// contains RAML properties
type HasProperties interface {
	GetProperty(string) Property
}

// Type defines shared fields for all RAML data types
type Type struct {
	typeProps   `yaml:",inline"`
	Annotations Annotations    `yaml:",inline"`
	_apiDef     *APIDefinition `yaml:"-"`
}

type typeProps struct {
	Name string `yaml:"-"`

	// A default value for a type. When an API request is completely missing the instance of a type, for example
	// when a query parameter described by a type is entirely missing from the request, then the API must act as if the
	// API client had sent an instance of that typeProps with the instance value being the value in the default facet.
	// Similarly, when the API response is completely missing the instance of a type, the client must act as if the API
	// server had returned an instance of that typeProps with the instance value being the value in the default facet.
	// A special case is made for URI parameters: for these, the client MUST substitute the value in the default facet
	// if no instance of the URI parameter was given.
	Default interface{} `yaml:"default"`

	// Alias for the equivalent "type" property,
	// for compatibility with RAML 0.8.
	// Deprecated - API definitions should use the "type" property,
	// as the "schema" alias for that property name may be removed in a future RAML version.
	// The "type" property allows for XML and JSON schemas.
	Schema interface{} `yaml:"schema"`

	// A base type which the current typeProps extends,
	// or more generally a type expression.
	// A base type which the current typeProps extends or just wraps.
	// The value of a type node MUST be either :
	//    a) the name of a user-defined type or
	//    b) the name of a built-in RAML data type (object, array, or one of the scalar types) or
	//    c) an inline type declaration.
	Type interface{} `yaml:"type" json:"type"`

	// An example of an instance of this type.
	// This can be used, e.g., by documentation generators to generate sample values for an object of this type.
	// Cannot be present if the examples property is present.
	// An example of an instance of this type that can be used,
	// for example, by documentation generators to generate sample values for an object of this type.
	// The "example" property MUST not be available when the "examples" property is already defined.
	Example interface{} `yaml:"example" json:"example"`

	// An object containing named examples of instances of this type.
	// This can be used, for example, by documentation generators
	// to generate sample values for an object of this type.
	// The "examples" property MUST not be available
	// when the "example" property is already defined.
	Examples map[string]interface{} `yaml:"examples" json:"examples"`

	// An alternate, human-friendly name for the type
	DisplayName string `yaml:"displayName" json:"displayName"`

	// A substantial, human-friendly description of the type.
	// Its value is a string and MAY be formatted using markdown.
	Description string `yaml:"description" json:"description"`

	// A map of additional, user-defined restrictions that will be inherited and applied by any extending subtype
	Facets map[string]interface{} `yaml:"facets"`

	// The capability to configure XML serialization of this type instance.
	XML interface{} `yaml:"xml"`

	// The properties that instances of this type may or must have.
	// we use `interface{}` as property type to support syntactic sugar & shortcut
	Properties map[string]interface{} `yaml:"properties" json:"properties"`

	// -------- Below facets are available for object typeProps --------------//

	// The minimum number of properties allowed for instances of this type.
	MinProperties int `yaml:"minProperties" json:"minProperties"`

	// The maximum number of properties allowed for instances of this type.
	MaxProperties int `yaml:"maxProperties" json:"maxProperties"`

	// A Boolean that indicates if an object instance has additional properties.
	AdditionalProperties string `yaml:"additionalProperties" json:"additionalProperties"`

	// Determines the concrete type of an individual object at runtime when,
	// for example, payloads contain ambiguous types due to unions or inheritance.
	// The value must match the name of one of the declared properties of a type.
	// Unsupported practices are inline type declarations and using discriminator with non-scalar properties.
	Discriminator string `yaml:"discriminator" json:"discriminator"`

	// Identifies the declaring type.
	// Requires including a discriminator property in the type declaration.
	// A valid value is an actual value that might identify the type
	// of an individual object and is unique in the hierarchy of the type.
	// Inline type declarations are not supported.
	DiscriminatorValue string `yaml:"discriminatorValue" json:"discriminatorValue"`

	// ---- facets for Array type --- //

	// Indicates the type all items in the array are inherited from.
	// Can be a reference to an existing type or an inline type declaration.
	Items interface{} `yaml:"items" json:"items"`

	// Minimum amount of items in array. Value MUST be equal to or greater than 0.
	MinItems int `yaml:"minItems" validate:"min=0" json:"minItems"`

	// Maximum amount of items in array. Value MUST be equal to or greater than 0.
	MaxItems int `yaml:"maxItems" validate:"min=0" json:"maxItems"`

	// Boolean value that indicates if items in the array MUST be unique.
	UniqueItems bool `yaml:"uniqueItems" json:"uniqueItems"`

	// ---------- facets for scalar type --------------------------//
	// Enumeration of possible values for this built-in scalar type.
	// The value is an array containing representations of possible values,
	// or a single value if there is only one possible value.
	Enum interface{} `yaml:"enum" json:"enum"`

	// ---------- facets for string type ------------------------//
	// Regular expression that this string should match.
	Pattern string `yaml:"pattern" json:"pattern"`

	// Minimum length of the string. Value MUST be equal to or greater than 0.
	MinLength int `yaml:"minLength" validate:"min=0" json:"minLength"`

	// Maximum length of the string. Value MUST be equal to or greater than 0.
	MaxLength int `yaml:"maxLength" validate:"max=0" json:"maxLength"`

	// ----------- facets for Number -------------------------- //
	// The minimum value of the parameter. Applicable only to parameters of type number or integer.
	Minimum int `yaml:"minimum" json:"minimum"`

	// The maximum value of the parameter. Applicable only to parameters of type number or integer.
	Maximum int `yaml:"maximum" json:"maximum"`

	// The format of the value. The value MUST be one of the following:
	// int32, int64, int, long, float, double, int16, int8
	Format string `yaml:"format" json:"format"`

	// A numeric instance is valid against "multipleOf"
	// if the result of dividing the instance by this keyword's value is an integer.
	MultipleOf int `yaml:"multipleOf" json:"multipleOf"`

	// ---------- facets for file --------------------------------//
	// A list of valid content-type strings for the file. The file type */* MUST be a valid value.
	FileTypes []string `yaml:"fileTypes" json:"fileTypes"`
}

// GetProperty returns property with given name
func (t *Type) GetProperty(name string) Property {
	propInterface, ok := t.Properties[name]
	if !ok {
		panic(fmt.Errorf("property %v not exist", name))
	}

	prop := toProperty(name, propInterface)
	prop._type = t

	return prop
}

// IsBuiltin if a type is an RAML builtin type.
func (t typeProps) IsBuiltin() bool {
	_, ok := scalarTypes[t.TypeString()]
	return ok
}

// Parents returns parents of this Type.
// "object" is not considered a parent
func (t typeProps) Parents() []string {
	if parents, ok := t.MultipleInheritance(); ok {
		return parents
	}

	if parent, ok := t.SingleInheritance(); ok {
		return []string{parent}
	}
	return []string{}
}

// IsJSONType true if this Type
// has JSON scheme that defines it's type
func (t typeProps) IsJSONType() bool {
	tStr := t.TypeString()
	tStr = strings.TrimSpace(tStr)
	return strings.HasPrefix(tStr, "{") && strings.HasSuffix(tStr, "}")
}

// SingleInheritance returns true if it
// inherit from single object which is not:
// - basic/scalar type
// - object
func (t typeProps) SingleInheritance() (string, bool) {
	if t.IsJSONType() {
		return "", false
	}
	tStr := t.TypeString()
	if tStr == "object" {
		return "", false
	}
	_, ok := scalarTypes[tStr]
	if ok {
		return "", false
	}
	_, isMultiple := t.MultipleInheritance()
	return tStr, !isMultiple
}

// MultipleInheritance returns all types inherited by this type
func (t typeProps) MultipleInheritance() ([]string, bool) {
	if t.IsJSONType() {
		return nil, false
	}
	tStr := t.TypeString()
	if !strings.HasPrefix(tStr, "[") || !strings.HasSuffix(tStr, "]") {
		return nil, false
	}
	tStr = strings.TrimPrefix(strings.TrimSuffix(tStr, "]"), "[")
	splitted := strings.Split(tStr, ",")
	return splitted, len(splitted) > 1
}

// IsMultipleInheritance returns true if this type
// has multiple inheritance
func (t typeProps) IsMultipleInheritance() bool {
	_, ok := t.MultipleInheritance()
	return ok
}

// interfaceToString converts interface type to string.
//
// We can't simply do this using type casting.
// example :
// 1. string type, result would be string
// 2. []interface{} type, result would be array of string. ex: a,b,c
// Please add other type as needed.
func interfaceToString(data interface{}) string {
	if data == nil {
		return ""
	}
	switch data.(type) {
	case string:
		return data.(string)
	case []interface{}:
		interfaceArr := data.([]interface{})
		var results []string
		for _, v := range interfaceArr {
			results = append(results, interfaceToString(v))
		}
		return "[" + strings.Join(results, ",") + "]"
	default:
		return fmt.Sprintf("%v", data)
	}
}

// TypeString returns string representation
// of this Type field
func (t typeProps) TypeString() string {
	return interfaceToString(t.Type)
}

// IsArray checks if this type is an Array
// see specs at http://docs.raml.org/specs/1.0/#raml-10-spec-array-types
func (t typeProps) IsArray() bool {
	if t.IsJSONType() {
		return false
	}
	return t.TypeString() == arrayType || strings.HasSuffix(t.TypeString(), "[]")
}

// ArrayType returns type of the array
func (t typeProps) ArrayType() string {
	if t.TypeString() == "array" {
		return interfaceToString(t.Items)
	}
	return strings.TrimSuffix(t.TypeString(), "[]")
}

// IsBidimensionalArray returns true
// if it is a bidimensional array
func (t typeProps) IsBidimensionalArray() bool {
	if t.IsJSONType() {
		return false
	}
	return strings.HasSuffix(t.TypeString(), "[][]")
}

// BidimensiArrayType returns type
// of a bidimensional array
func (t typeProps) BidimensiArrayType() string {
	return strings.TrimSuffix(t.TypeString(), "[][]")
}

// IsEnum type check if this type is an enum
// http://docs.raml.org/specs/1.0/#raml-10-spec-enums
func (t typeProps) IsEnum() bool {
	return t.Enum != nil
}

// IsUnion checks if a type is Union type
// see http://docs.raml.org/specs/1.0/#raml-10-spec-union-types
func (t typeProps) IsUnion() bool {
	if t.IsJSONType() {
		return false
	}
	return strings.Index(t.TypeString(), "|") > 0
}

// Union returns union type of this type
func (t typeProps) Union() ([]string, bool) {
	if !t.IsUnion() {
		return nil, false
	}
	var tips []string
	for _, ut := range strings.Split(t.TypeString(), "|") {
		tips = append(tips, strings.TrimSpace(ut))
	}
	return tips, true
}

// IsAlias returns true if this Type is
// alias of another Type
func (t typeProps) IsAlias() bool {
	if t.IsMultipleInheritance() || t.IsArray() || t.IsUnion() || t.TypeString() == "object" {
		return false
	}
	return t.TypeString() != "" && len(t.Properties) == 0
}

// see if the 'Type' field is a JSON schema
func (t *Type) postProcess(name string, apiDef *APIDefinition) error {
	t.Name = name
	t._apiDef = apiDef

	if t.IsJSONType() {
		return t.postProcessJSONSchema()
	}

	// process type in properties
	for name := range t.Properties {
		t.parseOptionalProperty(name)
		err := t.createTypeFromPropProperty(name, apiDef)
		if err != nil {
			return err
		}
		err = t.createTypeFromPropItems(name, apiDef)
		if err != nil {
			return err
		}
	}
	return nil
}

// parse property with `?` suffix as optional property
func (t *typeProps) parseOptionalProperty(name string) {
	if !strings.HasSuffix(name, "?") {
		return
	}
	newName := strings.TrimSuffix(name, "?")

	p := t.Properties[name]

	// we will modify both property name
	// and content, so we delete it
	delete(t.Properties, name)

	switch p.(type) {
	case string:
		// if it is a simple string
		// convert it to map style property
		newProp := map[interface{}]interface{}{}
		newProp["type"] = p
		newProp["required"] = false
		t.Properties[newName] = newProp

	case map[interface{}]interface{}:
		// already in map style property
		propMap := p.(map[interface{}]interface{})
		propMap["required"] = false
		t.Properties[newName] = propMap
	default:
		log.Fatalf("unexpeced property type: %v", p)
	}
}

// create type from item with inline type definition
func (t *typeProps) createTypeFromPropItems(name string, apiDef *APIDefinition) error {
	p := t.Properties[name]

	// propMap is this properties as map
	propMap, ok := p.(map[interface{}]interface{})
	if !ok {
		return nil
	}

	// only process the array
	tip, ok := propMap["type"]
	if !ok || tip != "array" {
		return nil
	}

	// only process if it has 'items' field
	itemsIf, ok := propMap["items"]
	if !ok {
		return nil
	}

	// check it's validity
	items, ok := itemsIf.(map[interface{}]interface{})
	if !ok {
		return nil
	}

	// to define new type, it needs to have 'properties' field
	props, ok := items["properties"].(map[interface{}]interface{})
	if !ok { // doesn't define new type, no problem, we can simply return
		return nil
	}
	newName := t.Name + name + "Item"
	created := apiDef.createType(newName, tip, props)

	delete(items, "properties")
	items["type"] = newName
	propMap["items"] = items

	t.Properties[name] = propMap

	if created {
		createdType := apiDef.Types[newName]
		err := createdType.postProcess(newName, apiDef)
		if err != nil {
			return err
		}
	}
	return nil
}

// create type from property's property
func (t *typeProps) createTypeFromPropProperty(name string, apiDef *APIDefinition) error {
	p := t.Properties[name]
	// only process map[interface]interface{}
	propMap, ok := p.(map[interface{}]interface{})
	if !ok {
		return nil
	}

	// only process if it has 'properties' field
	propsIf, ok := propMap["properties"]
	if !ok {
		return nil
	}

	// check validity of the properties
	props, ok := propsIf.(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("inline properties expect properties in type:map[string]interface{}")
	}

	newName := t.Name + name
	created := apiDef.createType(newName, propMap["type"], props)

	propMap["type"] = newName

	// delete the 'properties' field
	delete(propMap, "properties")

	t.Properties[name] = propMap

	// post process the created type
	if created {
		createdType := apiDef.Types[newName]
		err := createdType.postProcess(newName, apiDef)
		if err != nil {
			return err
		}
	}
	return nil
}
func (t *typeProps) postProcessJSONSchema() error {
	var jt JSONSchema

	if err := json.Unmarshal([]byte(t.TypeString()), &jt); err != nil {
		fmt.Println("failed to marshal json")
		return err
	}
	jt.PostUnmarshal()

	// assign the properties in JSON to Type object
	if t.Properties == nil {
		t.Properties = map[string]interface{}{}
	}
	for name, prop := range jt.Properties {
		t.Properties[name] = prop.toRAMLProperty()
	}

	t.Type = "object"
	return nil
}
