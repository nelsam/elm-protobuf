package elm

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/jalandis/elm-protobuf/pkg/stringextras"
)

// EnumCustomType - defines an Elm custom type (sometimes called union type) for a PB enum
// https://guide.elm-lang.org/types/custom_types.html
type EnumCustomType struct {
	Name                   Type
	Decoder                VariableName
	Encoder                VariableName
	DefaultVariantVariable VariableName
	DefaultVariantValue    VariantName
	Variants               []EnumVariant
}

// VariantName - unique camelcase identifier used for custom type variants
// https://guide.elm-lang.org/types/custom_types.html
type VariantName string

// EnumVariant - a possible variant of an enum CustomType
// https://guide.elm-lang.org/types/custom_types.html
type EnumVariant struct {
	Name  VariantName
	Value ProtobufFieldNumber
}

// OneOfCustomType - defines an Elm custom type (sometimes called union type) for a PB one-of
// https://guide.elm-lang.org/types/custom_types.html
type OneOfCustomType struct {
	Name     Type
	Decoder  VariableName
	Encoder  VariableName
	Variants []OneOfVariant
}

// OneOfVariant - a possible variant of a one-of CustomType
// https://guide.elm-lang.org/types/custom_types.html
type OneOfVariant struct {
	Name    VariantName
	Type    Type
	Num     ProtobufFieldNumber
	Decoder VariableName
	Encoder VariableName
}

// NestedVariantName - Elm variant name for a possibly nested PB definition
func NestedVariantName(name string, preface []string) VariantName {
    fullName := strings.Join(
        append(preface, stringextras.CamelCase(strings.ToLower(name))),
        "_",
    )
	return VariantName(fullName)
}

// EnumDefaultVariantVariableName - convenient identifier for a enum custom types default variant
func EnumDefaultVariantVariableName(t Type) VariableName {
	return VariableName(stringextras.FirstLower(fmt.Sprintf("%sDefault", t)))
}

// EnumCustomTypeTemplate - defines template for an enum custom type
func EnumCustomTypeTemplate(t *template.Template) (*template.Template, error) {
	return t.Parse(`
{{- define "enum-custom-type" -}}
type {{ .Name }}
{{- range $i, $v := .Variants }}
    {{ if not $i }}={{ else }}|{{ end }} {{ $v.Name }} -- {{ $v.Value }}
{{- end }}


{{ .Decoder }} : JD.Decoder {{ .Name }}
{{ .Decoder }} =
    let
        lookup v =
            case v of
{{- range .Variants }}
                {{ .Value }} ->
                    {{ .Name }}
{{ end }}
                _ ->
                    {{ .DefaultVariantValue }}
    in
        JD.map lookup JD.int


{{ .DefaultVariantVariable }} : {{ .Name }}
{{ .DefaultVariantVariable }} = {{ .DefaultVariantValue }}


{{ .Encoder }} : {{ .Name }} -> JE.Value
{{ .Encoder }} v =
    let
        lookup s =
            case s of
{{- range .Variants }}
                {{ .Name }} ->
                    {{ .Value }}
{{ end }}
    in
        JE.int <| lookup v
{{- end -}}
`)
}

// OneOfCustomTypeTemplate - defines template for a one-of custom type
func OneOfCustomTypeTemplate(t *template.Template) (*template.Template, error) {
	return t.Parse(`
{{- define "oneof-custom-type" -}}
type {{ .Name }}
    = {{ .Name }}Unspecified
{{- range .Variants }}
    | {{ .Name }} {{ .Type }}
{{- end }}


{{ .Decoder }} : JD.Decoder {{ .Name }}
{{ .Decoder }} =
    JD.lazy <| \_ -> JD.oneOf
        [{{ range $i, $v := .Variants }}{{ if $i }},{{ end }} JD.map {{ .Name }} (JD.index {{ toJSIdx .Num }} (failOnNull {{ .Decoder }}))
        {{ end }}, JD.succeed {{ .Name }}Unspecified
        ]


{{ .Encoder }} : Int -> {{ .Name }} -> JE.Value
{{ .Encoder }} idx v =
    case v of
        {{ .Name }}Unspecified ->
            JE.null
        {{- range .Variants }}

        {{ .Name }} x ->
            if idx == {{ .Num }} then {{ .Encoder }} x else JE.null
        {{- end }}
{{- end -}}
`)
}
