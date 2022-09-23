package elm

import (
	"fmt"
	"text/template"

	"github.com/jalandis/elm-protobuf/pkg/stringextras"

	"google.golang.org/protobuf/types/descriptorpb"
)

// WellKnownType - information to handle Google well known types
type WellKnownType struct {
	Type    Type
	Encoder VariableName
	Decoder VariableName
}

var (
	// WellKnownTypeMap - map of Google well known type PB identifier to encoder/decoder info
	WellKnownTypeMap = map[string]WellKnownType{
		".google.protobuf.Timestamp": {
			Type:    "Timestamp",
			Decoder: "timestampDecoder",
			Encoder: "timestampEncoder",
		},
		".google.protobuf.Int32Value": {
			Type:    intType,
			Decoder: "intValueDecoder",
			Encoder: "intValueEncoder",
		},
		".google.protobuf.Int64Value": {
			Type:    intType,
			Decoder: "intValueDecoder",
			Encoder: "numericStringEncoder",
		},
		".google.protobuf.UInt32Value": {
			Type:    intType,
			Decoder: "intValueDecoder",
			Encoder: "intValueEncoder",
		},
		".google.protobuf.UInt64Value": {
			Type:    intType,
			Decoder: "intValueDecoder",
			Encoder: "numericStringEncoder",
		},
		".google.protobuf.DoubleValue": {
			Type:    floatType,
			Decoder: "floatValueDecoder",
			Encoder: "floatValueEncoder",
		},
		".google.protobuf.FloatValue": {
			Type:    floatType,
			Decoder: "floatValueDecoder",
			Encoder: "floatValueEncoder",
		},
		".google.protobuf.StringValue": {
			Type:    stringType,
			Decoder: "stringValueDecoder",
			Encoder: "stringValueEncoder",
		},
		".google.protobuf.BytesValue": {
			Type:    bytesType,
			Decoder: "bytesValueDecoder",
			Encoder: "bytesValueEncoder",
		},
		".google.protobuf.BoolValue": {
			Type:    boolType,
			Decoder: "boolValueDecoder",
			Encoder: "boolValueEncoder",
		},
	}

	reservedKeywords = map[string]bool{
		"module":   true,
		"exposing": true,
		"import":   true,
		"type":     true,
		"let":      true,
		"in":       true,
		"if":       true,
		"then":     true,
		"else":     true,
		"where":    true,
		"case":     true,
		"of":       true,
		"port":     true,
		"as":       true,
	}
)

// TypeAlias - defines an Elm type alias (somtimes called a record)
// https://guide.elm-lang.org/types/type_aliases.html
type TypeAlias struct {
	Name          Type
	Decoder       VariableName
	Encoder       VariableName
	FieldEncoders []TypeAliasField
	Fields        []TypeAliasField
}

// FieldDecoder used in type alias decdoer (ex. )
type FieldDecoder string

// FieldEncoder used in type alias decdoer (ex. )
type FieldEncoder string

// TypeAliasField - type alias field definition
type TypeAliasField struct {
	Name    VariableName
	Type    Type
	Number  ProtobufFieldNumber
	Default string
	Decoder FieldDecoder
	Encoder FieldEncoder
}

func avoidCollision(in string) string {
	if reservedKeywords[in] {
		return fmt.Sprintf("%s_", in)
	}

	return in
}

// jsIdx returns the index that javascript generated code typically uses for a
// given field. There appear to be some scenarios where javascript reserves the
// 0th index, but I haven't encountered it in my own protobufs.
func jsIdx(i ProtobufFieldNumber) int {
	return int(i) - 1
}

// FieldName - simple camelcase variable name with first letter lower
func FieldName(in string) VariableName {
	return VariableName(avoidCollision(stringextras.LowerCamelCase(in)))
}

func RequiredFieldEncoder(pb *descriptorpb.FieldDescriptorProto) FieldEncoder {
	return FieldEncoder(fmt.Sprintf(
		"%s v.%s",
		BasicFieldEncoder(pb),
		FieldName(pb.GetName()),
	))
}

// FieldNum returns the field number for referencing indexes in json internal
// representations of messages.
func FieldNum(pb *descriptorpb.FieldDescriptorProto) ProtobufFieldNumber {
	return ProtobufFieldNumber(pb.GetNumber())
}

func RequiredFieldDecoder(pb *descriptorpb.FieldDescriptorProto) FieldDecoder {
	return FieldDecoder(fmt.Sprintf(
		"idxWithDefault %d %s %s",
		jsIdx(FieldNum(pb)),
		BasicFieldDecoder(pb),
		BasicFieldDefaultValue(pb),
	))
}

func OneOfEncoder(oneof *descriptorpb.OneofDescriptorProto, field *descriptorpb.FieldDescriptorProto, t Type) FieldEncoder {
	return FieldEncoder(fmt.Sprintf("%s %d v.%s",
		EncoderName(t),
		FieldNum(field),
		FieldName(oneof.GetName()),
	))
}

func OneOfDecoder(pb *descriptorpb.OneofDescriptorProto, t Type) FieldDecoder {
	return FieldDecoder(fmt.Sprintf("custom %s",
		DecoderName(t),
	))
}

func MapType(messagePb *descriptorpb.DescriptorProto) Type {
	keyField := messagePb.GetField()[0]
	valueField := messagePb.GetField()[1]

	return Type(fmt.Sprintf(
		"Dict.Dict %s %s",
		BasicFieldType(keyField),
		BasicFieldType(valueField),
	))
}

func MapEncoder(
	fieldPb *descriptorpb.FieldDescriptorProto,
	messagePb *descriptorpb.DescriptorProto,
) FieldEncoder {
	valueField := messagePb.GetField()[1]

	return FieldEncoder(fmt.Sprintf(
		"mapEntriesFieldEncoder %d %s v.%s",
		FieldNum(fieldPb),
		BasicFieldEncoder(valueField),
		FieldName(fieldPb.GetName()),
	))
}

func MapDecoder(
	fieldPb *descriptorpb.FieldDescriptorProto,
	messagePb *descriptorpb.DescriptorProto,
) FieldDecoder {
	valueField := messagePb.GetField()[1]

	return FieldDecoder(fmt.Sprintf(
		"mapEntries %d %s",
		FieldNum(fieldPb),
		BasicFieldDecoder(valueField),
	))
}

func MaybeType(t Type) Type {
	return Type(fmt.Sprintf("Maybe %s", t))
}

func MaybeEncoder(pb *descriptorpb.FieldDescriptorProto) FieldEncoder {
	return FieldEncoder(fmt.Sprintf(
		"maybeEncoder %s v.%s",
		BasicFieldEncoder(pb),
		FieldName(pb.GetName()),
	))
}

func MaybeDecoder(pb *descriptorpb.FieldDescriptorProto) FieldDecoder {
	return FieldDecoder(fmt.Sprintf(
		"idxWithDefault %d (JD.maybe %s) Nothing",
		jsIdx(FieldNum(pb)),
		BasicFieldDecoder(pb),
	))
}

func ListType(t Type) Type {
	return Type(fmt.Sprintf("List %s", t))
}

func ListEncoder(pb *descriptorpb.FieldDescriptorProto) FieldEncoder {
	return FieldEncoder(fmt.Sprintf(
		"JE.list %s v.%s",
		BasicFieldEncoder(pb),
		FieldName(pb.GetName()),
	))
}

func ListDecoder(pb *descriptorpb.FieldDescriptorProto) FieldDecoder {
	return FieldDecoder(fmt.Sprintf(
		"idxWithDefault %d (JD.list %s) []",
		jsIdx(FieldNum(pb)),
		BasicFieldDecoder(pb),
	))
}

// OneOfType returns the type of a oneof field.  Oneof fields will always
// be nested (they cannot be defined outside of a message type), so we know
// that we will always be passed the result of a NestedType call.
func OneOfType(in Type) Type {
	return in
}

// TypeAliasTemplate - defines templates for self contained type aliases
func TypeAliasTemplate(t *template.Template) (*template.Template, error) {
	return t.Parse(`
{{- define "type-alias" -}}
type alias {{ .Name }} =
    { {{ range $i, $v := .Fields }}
        {{- if $i }}, {{ end }}{{ .Name }} : {{ .Type }}{{ if .Number }} -- {{ .Number }}{{ end }}
    {{ end }}}


default{{ .Name }} : {{ .Name }}
default{{ .Name }} =
  { {{- range $i, $v := .Fields -}}
  {{- if $i }}
  , {{ end }}{{ .Name }} = {{ .Default }}
  {{- end }}
  }


-- {{ .Decoder }} is used to decode protobuf messages from ports, following the javascript
-- array format.
{{ .Decoder }} : JD.Decoder {{ .Name }}
{{ .Decoder }} =
    JD.lazy <| \_ -> decode {{ .Name }}{{ range .Fields }}
        |> {{ .Decoder }}{{ end }}


-- {{ .Encoder }} is used to encode protobuf messages for ports, so that javascript code
-- may use the value in the message constructor.
{{ .Encoder }} : {{ .Name }} -> JE.Value
{{ .Encoder }} v =
    -- javascript uses the field number to index into arrays, so we need to ensure that
    -- the list has empty values at indexes that don't have fields.
    {{- $idx := 1 }}
    valueList
        [ {{ range $i, $v := .FieldEncoders -}}
         {{- range (fieldSeq $idx $v.Number) -}}
         {{- if (ne . 1) }}
        , {{ end -}}
             JE.null
         {{- end }}
         {{- if (ne $v.Number 1) }}
        , {{ end -}}
         ({{ $v.Encoder }})
         {{- $idx = (nextFieldNum $v.Number) -}}
        {{ end }}
        ]
{{- end -}}
`)
}
