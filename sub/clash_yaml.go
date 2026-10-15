package sub

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
)

// yamlQuotedString forces values such as REALITY short IDs and numeric-looking
// passwords to remain strings for YAML 1.1 clients.
type yamlQuotedString string

func (s yamlQuotedString) MarshalYAML() ([]byte, error) {
	return []byte("'" + strings.ReplaceAll(string(s), "'", "''") + "'"), nil
}

var yamlPlainScalarNotString = regexp.MustCompile(`^(?:~|[Nn]ull|NULL|[Tt]rue|TRUE|[Ff]alse|FALSE|[Yy]es|YES|[Nn]o|NO|[Oo]n|ON|[Oo]ff|OFF|[YyNn]|[-+]?[0-9]+|0[oO]?[0-7]+|0[xX][0-9a-fA-F]+|[-+]?[0-9][0-9_]*(?::[0-5]?[0-9])+|[-+]?(?:[0-9]*\.[0-9]+|[0-9]+\.?[0-9]*)(?:[eE][-+]?[0-9]+)?|[-+]?\.(?:inf|Inf|INF)|\.(?:nan|NaN|NAN)|[0-9]{4}-[0-9]{1,2}-[0-9]{1,2}(?:[Tt ].*)?)$`)

func quoteAmbiguousYAMLScalars(v any) any {
	if v == nil {
		return nil
	}
	if s, ok := v.(string); ok {
		if s != "" && yamlPlainScalarNotString.MatchString(s) {
			return yamlQuotedString(s)
		}
		return s
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Map:
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			key, ok := iter.Key().Interface().(string)
			if !ok {
				return v
			}
			out[key] = quoteAmbiguousYAMLScalars(iter.Value().Interface())
		}
		return out
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
			return v
		}
		out := make([]any, rv.Len())
		for i := range rv.Len() {
			out[i] = quoteAmbiguousYAMLScalars(rv.Index(i).Interface())
		}
		return out
	case reflect.Ptr, reflect.Interface:
		if rv.IsNil() {
			return v
		}
		return quoteAmbiguousYAMLScalars(rv.Elem().Interface())
	default:
		return v
	}
}

func marshalClashYAML(config any) ([]byte, error) {
	return yaml.Marshal(quoteAmbiguousYAMLScalars(config))
}
