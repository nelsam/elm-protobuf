package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/jalandis/elm-protobuf/pkg/stringextras"
	"github.com/jalandis/elm-protobuf/pkg/elm"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

const (
	version = "0.0.2"
	docUrl  = "https://github.com/jalandis/elm-protobuf"

	extension = ".elm"
)

var excludedFiles = map[string]bool{
	"google/protobuf/timestamp.proto":  true,
	"google/protobuf/wrappers.proto":   true,
	"google/protobuf/descriptor.proto": true,
}

type parameters struct {
	Version          bool
	Debug            bool
	RemoveDeprecated bool
	modPrefix        string
}

func parseParameters(input *string) (parameters, error) {
	var result parameters
	var err error

	if input == nil {
		return result, nil
	}

	for _, v := range strings.Split(*input, ",") {
		parts := strings.Split(v, "=")
		name := parts[0]
		v := parts[1:]
		switch name {
		case "remove-deprecated":
			result.RemoveDeprecated = true
		case "debug":
			result.Debug = true
		case "module-prefix":
			result.modPrefix = v[0]
		case "exclude":
			excludedFiles[v[0]] = true
		default:
			err = fmt.Errorf("unknown parameter: \"%s\"", name)
		}
	}

	return result, err
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Fprintf(os.Stdout, "%v %v\n", filepath.Base(os.Args[0]), version)
		os.Exit(0)
	}
	if len(os.Args) == 2 && os.Args[1] == "--help" {
		fmt.Fprintf(os.Stdout, "See "+docUrl+" for usage information.\n")
		os.Exit(0)
	}

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Could not read request from STDIN: %v", err)
	}

	req := &pluginpb.CodeGeneratorRequest{}

	err = proto.Unmarshal(data, req)
	if err != nil {
		log.Fatalf("Could not unmarshal request: %v", err)
	}

	parameters, err := parseParameters(req.Parameter)
	if err != nil {
		log.Fatalf("Failed to parse parameters: %v", err)
	}

	if parameters.Debug {
		// Remove useless source code data.
		for _, inFile := range req.GetProtoFile() {
			inFile.SourceCodeInfo = nil
		}

		result, err := proto.Marshal(req)
		if err != nil {
			log.Fatalf("Failed to marshal request: %v", err)
		}

		log.Printf("Input data: %s", result)
	}

	resp := &pluginpb.CodeGeneratorResponse{}
	for _, inFile := range req.GetProtoFile() {
		log.Printf("Processing file %s", inFile.GetName())
		// Well Known Types.
		if excludedFiles[inFile.GetName()] {
			log.Printf("Skipping well known type")
			continue
		}

		name := fileName(inFile.GetName())
		content, err := templateFile(inFile, parameters)
		if err != nil {
			log.Fatalf("Could not template file: %v", err)
		}

		resp.File = append(resp.File, &pluginpb.CodeGeneratorResponse_File{
			Name:    &name,
			Content: &content,
		})
	}

	data, err = proto.Marshal(resp)
	if err != nil {
		log.Fatalf("Could not marshal response: %v [%v]", err, resp)
	}

	_, err = os.Stdout.Write(data)
	if err != nil {
		log.Fatalf("Could not write response to STDOUT: %v", err)
	}
}

func hasMapEntries(inFile *descriptorpb.FileDescriptorProto) bool {
	for _, m := range inFile.GetMessageType() {
		if hasMapEntriesInMessage(m) {
			return true
		}
	}

	return false
}

func hasMapEntriesInMessage(inMessage *descriptorpb.DescriptorProto) bool {
	if inMessage.GetOptions().GetMapEntry() {
		return true
	}

	for _, m := range inMessage.GetNestedType() {
		if hasMapEntriesInMessage(m) {
			return true
		}
	}

	return false
}

func templateFile(inFile *descriptorpb.FileDescriptorProto, p parameters) (string, error) {
	t := template.New("t").Funcs(template.FuncMap{
		"fieldSeq": func(from int, to elm.ProtobufFieldNumber) []int {
			var l []int
			for i := from; i < int(to); i++ {
				l = append(l, i)
			}
			return l
		},
		"nextFieldNum": func(n elm.ProtobufFieldNumber) int {
			return int(n) + 1
		},
		"toJSIdx": func(n elm.ProtobufFieldNumber) int {
			return int(n) - 1
		},
	})

	t, err := elm.EnumCustomTypeTemplate(t)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse enum custom type template")
	}

	t, err = elm.OneOfCustomTypeTemplate(t)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse one-of custom type template")
	}

	t, err = elm.TypeAliasTemplate(t)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse type alias template")
	}

	t, err = t.Parse(`
{{- define "nested-message" -}}
{{ template "type-alias" .TypeAlias }}
{{- range .OneOfCustomTypes }}


{{ template "oneof-custom-type" . }}
{{- end }}
{{- range .EnumCustomTypes }}


{{ template "enum-custom-type" . }}
{{- end }}
{{- range .NestedMessages }}


{{ template "nested-message" . }}
{{- end }}
{{- end -}}
`)

	if err != nil {
		return "", errors.Wrap(err, "failed to parse nested PB message template")
	}

	t, err = t.Parse(`module {{ .ModuleName }} exposing (..)

-- DO NOT EDIT
-- AUTOGENERATED BY THE ELM PROTOCOL BUFFER COMPILER
-- https://github.com/tiziano88/elm-protobuf
-- source file: {{ .SourceFile }}

import Protobuf exposing (..)

import Json.Decode as JD
import Json.Encode as JE
{{- if .ImportDict }}
import Dict
{{- end }}
{{- range .AdditionalImports }}
import {{ . }} exposing (..)
{{ end }}


-- noop is here because I don't know elm well enough to know how to provide
-- a (a -> a) function to JE.list without it.
noop : JE.Value -> JE.Value
noop v =
    v


valueList : List JE.Value -> JE.Value
valueList l =
    JE.list noop l


custom : JD.Decoder a -> JD.Decoder (a -> b) -> JD.Decoder b
custom =
    JD.map2 (|>)


idxWithDefault : Int -> JD.Decoder a -> a -> JD.Decoder (a -> b) -> JD.Decoder b
idxWithDefault idx decoder default =
    JD.map2 (|>) (JD.oneOf [ JD.index idx decoder, JD.succeed default ])


maybeEncoder : (a -> JE.Value) -> Maybe a -> JE.Value
maybeEncoder enc v =
    case v of
        Nothing ->
            JE.null

        Just av ->
            enc av


type Field a
  = Null
  | Present a


{- failOnNull helps us handle oneof fields.  In JS land,
oneofs are presented with empty slots in the backing
array.  We need the empty slots to be translated to null
since elm doesn't know how to handle empty slots in a
list (which is totally fair).

Then, the decoder for a oneof variant needs to fail if
the value is null.  That's what this function does.
-}
failOnNull : JD.Decoder a -> JD.Decoder a
failOnNull decoder =
  JD.oneOf
    [ JD.null Null
    , JD.map Present decoder
    ]
    |> JD.andThen
      (\v ->
        case v of
          Null ->
            JD.fail "received null value"

          Present fv ->
            JD.succeed fv
      )


{{- range .TopEnums }}


{{ template "enum-custom-type" . }}
{{- end }}
{{- range .Messages }}


{{ template "nested-message" . }}
{{- end }}
`)
	if err != nil {
		return "", err
	}

	buff := &bytes.Buffer{}
	if err = t.Execute(buff, struct {
		SourceFile        string
		ModuleName        string
		ImportDict        bool
		AdditionalImports []string
		TopEnums          []elm.EnumCustomType
		Messages          []pbMessage
	}{
		SourceFile:        inFile.GetName(),
		ModuleName:        moduleName(p.modPrefix, inFile.GetName()),
		ImportDict:        hasMapEntries(inFile),
		AdditionalImports: additionalImports(p.modPrefix, inFile.GetDependency()),
		TopEnums:          enumsToCustomTypes([]string{}, inFile.GetEnumType(), p),
		Messages:          messages([]string{}, inFile.GetMessageType(), p),
	}); err != nil {
		return "", err
	}

	return buff.String(), nil
}

type pbMessage struct {
	TypeAlias        elm.TypeAlias
	OneOfCustomTypes []elm.OneOfCustomType
	EnumCustomTypes  []elm.EnumCustomType
	NestedMessages   []pbMessage
}

func isDeprecated(options interface{}) bool {
	switch v := options.(type) {
	case *descriptorpb.MessageOptions:
		return v != nil && v.Deprecated != nil && *v.Deprecated
	case *descriptorpb.FieldOptions:
		return v != nil && v.Deprecated != nil && *v.Deprecated
	case *descriptorpb.EnumOptions:
		return v != nil && v.Deprecated != nil && *v.Deprecated
	case *descriptorpb.EnumValueOptions:
		return v != nil && v.Deprecated != nil && *v.Deprecated
	default:
		return false
	}
}

func enumsToCustomTypes(preface []string, enumPbs []*descriptorpb.EnumDescriptorProto, p parameters) []elm.EnumCustomType {
	var result []elm.EnumCustomType
	for _, enumPb := range enumPbs {
		if isDeprecated(enumPb.Options) && p.RemoveDeprecated {
			continue
		}

		var values []elm.EnumVariant
		for _, value := range enumPb.GetValue() {
			if isDeprecated(value.Options) && p.RemoveDeprecated {
				continue
			}

			values = append(values, elm.EnumVariant{
				Name:  elm.NestedVariantName(value.GetName(), preface),
				Value: elm.ProtobufFieldNumber(value.GetNumber()),
			})
		}

		enumType := elm.NestedType(enumPb.GetName(), preface)

		result = append(result, elm.EnumCustomType{
			Name:                   enumType,
			Decoder:                elm.DecoderName(enumType),
			Encoder:                elm.EncoderName(enumType),
			DefaultVariantVariable: elm.EnumDefaultVariantVariableName(enumType),
			DefaultVariantValue:    values[0].Name,
			Variants:               values,
		})
	}

	return result
}

func oneOfsToCustomTypes(preface []string, messagePb *descriptorpb.DescriptorProto, p parameters) []elm.OneOfCustomType {
	var result []elm.OneOfCustomType

	if isDeprecated(messagePb.Options) && p.RemoveDeprecated {
		return result
	}

	for oneofIndex, oneOfPb := range messagePb.GetOneofDecl() {
		var variants []elm.OneOfVariant
		for _, inField := range messagePb.GetField() {
			if isDeprecated(inField.Options) && p.RemoveDeprecated {
				continue
			}

			if inField.OneofIndex == nil || inField.GetOneofIndex() != int32(oneofIndex) {
				continue
			}

			variants = append(variants, elm.OneOfVariant{
				Name:    elm.NestedVariantName(inField.GetName(), preface),
				Type:    elm.BasicFieldType(inField),
				Num:     elm.ProtobufFieldNumber(inField.GetNumber()),
				Decoder: elm.BasicFieldDecoder(inField),
				Encoder: elm.BasicFieldEncoder(inField),
			})
		}

		name := elm.NestedType(oneOfPb.GetName(), preface)
		result = append(result, elm.OneOfCustomType{
			Name:     name,
			Decoder:  elm.DecoderName(name),
			Encoder:  elm.EncoderName(name),
			Variants: variants,
		})
	}

	return result
}

func fieldDefault(field *descriptorpb.FieldDescriptorProto) string {
	defV := field.GetDefaultValue()

	switch field.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		defV = fmt.Sprintf(`"%s"`, strings.Replace(defV, `"`, `\"`, -1))
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		// Elm uses 'False' and 'True' but golang libraries will decode
		// these as 'false' and 'true'.
		defV = strings.ToUpper(defV[:1])+defV[1:]
	default:
	}
	if defV == "" {
		return elm.BasicFieldDefaultValue(field)
	}
	return defV
}

func messages(preface []string, messagePbs []*descriptorpb.DescriptorProto, p parameters) []pbMessage {
	var result []pbMessage
	for _, messagePb := range messagePbs {
		if isDeprecated(messagePb.Options) && p.RemoveDeprecated {
			continue
		}

		name := elm.NestedType(messagePb.GetName(), preface)
		nestedPreface := append([]string{messagePb.GetName()}, preface...)
		alias := elm.TypeAlias{
			Name:    name,
			Decoder: elm.DecoderName(name),
			Encoder: elm.EncoderName(name),
		}

		for _, fieldPb := range messagePb.GetField() {
			if isDeprecated(fieldPb.Options) && p.RemoveDeprecated {
				continue
			}

			if fieldPb.OneofIndex != nil {
				// For encoding, we need one encoder for each variant in
				// the oneof, but for decoding, we only want one decoder
				// for the whole oneof.
				oneof := messagePb.GetOneofDecl()[fieldPb.GetOneofIndex()]
				typeName := elm.OneOfType(elm.NestedType(oneof.GetName(), nestedPreface))
				alias.FieldEncoders = append(alias.FieldEncoders, elm.TypeAliasField{
					Name:    elm.FieldName(oneof.GetName()),
					Type:    typeName,
					Number:  elm.ProtobufFieldNumber(fieldPb.GetNumber()),
					Encoder: elm.OneOfEncoder(oneof, fieldPb, typeName),
				})
				continue
			}

			nested := getNestedType(fieldPb, messagePb)
			if nested != nil {
				field := elm.TypeAliasField{
					Name:    elm.FieldName(fieldPb.GetName()),
					Type:    elm.MapType(nested),
					Number:  elm.ProtobufFieldNumber(fieldPb.GetNumber()),
					Default: "Nothing",
					Encoder: elm.MapEncoder(fieldPb, nested),
					Decoder: elm.MapDecoder(fieldPb, nested),
				}
				alias.Fields = append(alias.Fields, field)
				alias.FieldEncoders = append(alias.FieldEncoders, field)
				continue
			}
			if isOptional(fieldPb) {
				field := elm.TypeAliasField{
					Name:    elm.FieldName(fieldPb.GetName()),
					Type:    elm.MaybeType(elm.BasicFieldType(fieldPb)),
					Number:  elm.ProtobufFieldNumber(fieldPb.GetNumber()),
					Default: fieldDefault(fieldPb),
					Encoder: elm.MaybeEncoder(fieldPb),
					Decoder: elm.MaybeDecoder(fieldPb),
				}
				alias.Fields = append(alias.Fields, field)
				alias.FieldEncoders = append(alias.FieldEncoders, field)
				continue
			}
			if isRepeated(fieldPb) {
				field := elm.TypeAliasField{
					Name:    elm.FieldName(fieldPb.GetName()),
					Type:    elm.ListType(elm.BasicFieldType(fieldPb)),
					Number:  elm.ProtobufFieldNumber(fieldPb.GetNumber()),
					Default: "[]",
					Encoder: elm.ListEncoder(fieldPb),
					Decoder: elm.ListDecoder(fieldPb),
				}
				alias.Fields = append(alias.Fields, field)
				alias.FieldEncoders = append(alias.FieldEncoders, field)
				continue
			}
			field := elm.TypeAliasField{
				Name:    elm.FieldName(fieldPb.GetName()),
				Type:    elm.BasicFieldType(fieldPb),
				Number:  elm.ProtobufFieldNumber(fieldPb.GetNumber()),
				Default: fieldDefault(fieldPb),
				Encoder: elm.RequiredFieldEncoder(fieldPb),
				Decoder: elm.RequiredFieldDecoder(fieldPb),
			}
			alias.Fields = append(alias.Fields, field)
			alias.FieldEncoders = append(alias.FieldEncoders, field)
		}
		sort.Slice(alias.FieldEncoders, func(i, j int) bool {
			// Order matters in the encoders, since their index has to correspond to
			// their field number for port encoding.  The template fills in gaps but
			// has no way to backtrack if it has already filled in an index.
			return alias.FieldEncoders[i].Number < alias.FieldEncoders[j].Number
		})

		for _, oneOfPb := range messagePb.GetOneofDecl() {
			typeName := elm.OneOfType(elm.NestedType(oneOfPb.GetName(), nestedPreface))
			alias.Fields = append(alias.Fields, elm.TypeAliasField{
				Name:    elm.FieldName(oneOfPb.GetName()),
				Type:    typeName,
				Default: string(typeName + "Unspecified"),
				Decoder: elm.OneOfDecoder(oneOfPb, typeName),
			})
		}

		result = append(result, pbMessage{
			TypeAlias:        alias,
			OneOfCustomTypes: oneOfsToCustomTypes(nestedPreface, messagePb, p),
			EnumCustomTypes:  enumsToCustomTypes(nestedPreface, messagePb.GetEnumType(), p),
			NestedMessages:   messages(nestedPreface, messagePb.GetNestedType(), p),
		})
	}

	return result
}

func isOptional(inField *descriptorpb.FieldDescriptorProto) bool {
	return inField.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL &&
		inField.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
}

func isRepeated(inField *descriptorpb.FieldDescriptorProto) bool {
	return inField.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED
}

func getLocalType(fullyQualifiedTypeName string) string {
	splitName := strings.Split(fullyQualifiedTypeName, ".")
	return splitName[len(splitName)-1]
}

func getNestedType(inField *descriptorpb.FieldDescriptorProto, inMessage *descriptorpb.DescriptorProto) *descriptorpb.DescriptorProto {
	localTypeName := getLocalType(inField.GetTypeName())
	for _, nested := range inMessage.GetNestedType() {
		if nested.GetName() == localTypeName && nested.GetOptions().GetMapEntry() {
			return nested
		}
	}

	return nil
}

func fileName(inFilePath string) string {
	inFileDir, inFileName := filepath.Split(inFilePath)

	trimmed := strings.TrimSuffix(inFileName, ".proto")
	shortFileName := stringextras.FirstUpper(trimmed)

	fullFileName := ""
	for _, segment := range strings.Split(inFileDir, "/") {
		if segment == "" {
			continue
		}

		fullFileName += stringextras.FirstUpper(segment) + "/"
	}

	return fullFileName + shortFileName + extension
}

func moduleName(modPrefix, inFilePath string) string {
	inFileDir, inFileName := filepath.Split(inFilePath)

	trimmed := strings.TrimSuffix(inFileName, ".proto")
	shortModuleName := stringextras.FirstUpper(trimmed)

	path := strings.Split(inFileDir, string(filepath.Separator))
	if modPrefix != "" {
		path = append(strings.Split(modPrefix, "."), path...)
	}

	var final []string
	for _, segment := range path {
		if segment == "" {
			continue
		}

		final = append(final, stringextras.FirstUpper(segment))
	}

	return strings.Join(append(final, shortModuleName), ".")
}

func additionalImports(modPrefix string, dependencies []string) []string {
	prefix := strings.Split(modPrefix, ".")
	var additions []string
	for _, d := range dependencies {
		if excludedFiles[d] {
			continue
		}

		final := append([]string(nil), prefix...)
		for _, segment := range strings.Split(strings.TrimSuffix(d, ".proto"), "/") {
			if segment == "" {
				continue
			}
			final = append(final, stringextras.FirstUpper(segment))
		}

		additions = append(additions, strings.Join(final, "."))
	}
	return additions
}
