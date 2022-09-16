package elm

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jalandis/elm-protobuf/pkg/stringextras"

	"google.golang.org/protobuf/types/descriptorpb"
)

// Type - Basic Elm type, custom type, or type alias
type Type string

var (
	intType    Type = "Int"
	floatType  Type = "Float"
	stringType Type = "String"
	bytesType  Type = "Bytes"
	boolType   Type = "Bool"
)

// VariableName - unique camelcase identifier starting with lowercase letter.
// Used for both constants and function declarations
type VariableName string

// ProtobufFieldNumber - unique identifier required for protobuf field
// declarations. Used as the index in json array values (how generated
// javascript code stores protobuf messages internally).
type ProtobufFieldNumber int

// DecoderName - decoder function name for Elm type
func DecoderName(t Type) VariableName {
	return VariableName(stringextras.FirstLower(fmt.Sprintf("%sPortDecoder", t)))
}

// EncoderName - encoder function name for Elm type
func EncoderName(t Type) VariableName {
	return VariableName(stringextras.FirstLower(fmt.Sprintf("%sPortEncoder", t)))
}

// NestedType - top level Elm type for a possibly nested PB definition
func NestedType(name string, preface []string) Type {
	fullName := strings.Join(
		append(preface, stringextras.CamelCase(name)),
		"_",
	)
	return Type(stringextras.FirstUpper(fullName))
}

// ExternalType - handles types defined in external files
func ExternalType(inType string) Type {
	messageSegments := []string{}
	for _, s := range strings.Split(inType, ".") {
		if s == "" {
			continue
		}

		if r, _ := utf8.DecodeRuneInString(s); !unicode.IsLower(r) {
			messageSegments = append(messageSegments, stringextras.FirstUpper(s))
		}
	}
	return Type(strings.Join(messageSegments, "_"))
}

func BasicFieldEncoder(inField *descriptorpb.FieldDescriptorProto) VariableName {
	switch inField.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32:
		return "JE.int"
	case descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		return "numericStringEncoder"
	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
		descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		return "JE.float"
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		return "JE.bool"
	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		return "JE.string"
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
		descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		if n, ok := WellKnownTypeMap[inField.GetTypeName()]; ok {
			return n.Encoder
		}

		return EncoderName(ExternalType(inField.GetTypeName()))
	case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
		return "bytesFieldEncoder"
	default:
		panic(fmt.Errorf("Error generating decoder for field %s", inField.GetType()))
	}
}

func BasicFieldDecoder(inField *descriptorpb.FieldDescriptorProto) VariableName {
	switch inField.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		return "intDecoder"
	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
		descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		return "JD.float"
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		return "JD.bool"
	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		return "JD.string"
	case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
		return "bytesFieldDecoder"
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
		descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		if n, ok := WellKnownTypeMap[inField.GetTypeName()]; ok {
			return n.Decoder
		}

		return DecoderName(ExternalType(inField.GetTypeName()))
	default:
		panic(fmt.Errorf("error generating decoder for field %s", inField.GetType()))
	}
}

func BasicFieldType(inField *descriptorpb.FieldDescriptorProto) Type {
	switch inField.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		return intType
	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
		descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		return floatType
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		return boolType
	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		return stringType
	case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
		return bytesType
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM,
		descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		if n, ok := WellKnownTypeMap[inField.GetTypeName()]; ok {
			return n.Type
		}
		return ExternalType(inField.GetTypeName())
	default:
		panic(fmt.Errorf("Error generating type for field %q %s", inField.GetName(), inField.GetType()))
	}
}

func BasicFieldDefaultValue(inField *descriptorpb.FieldDescriptorProto) string {
	if inField.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
		return "[]"
	}

	switch inField.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
		descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		return "0"
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		return "False"
	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		return "\"\""
	case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
		return "[]"
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		return string(EnumDefaultVariantVariableName(ExternalType(inField.GetTypeName())))
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		return "Nothing"
	default:
		// TODO: maps and stuff
		panic(fmt.Errorf("error - no known default value for field %s", inField.GetType()))
	}
}
