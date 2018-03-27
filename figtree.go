package figtree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/fatih/camelcase"
	"github.com/pkg/errors"

	yaml "gopkg.in/coryb/yaml.v2"
	logging "gopkg.in/op/go-logging.v1"
)

type Logger interface {
	Debugf(format string, args ...interface{})
}

var Log Logger = logging.MustGetLogger("figtree")

type FigTree struct {
	ConfigDir string
	Defaults  interface{}
	EnvPrefix string
	stop      bool
}

func NewFigTree() *FigTree {
	return &FigTree{
		EnvPrefix: "FIGTREE",
	}
}

func LoadAllConfigs(configFile string, options interface{}) error {
	return NewFigTree().LoadAllConfigs(configFile, options)
}

func LoadConfig(configFile string, options interface{}) error {
	return NewFigTree().LoadConfig(configFile, options)
}

func (f *FigTree) LoadAllConfigs(configFile string, options interface{}) error {
	// reset from any previous config parsing runs
	f.stop = false

	if f.ConfigDir != "" {
		configFile = path.Join(f.ConfigDir, configFile)
	}

	paths := FindParentPaths(configFile)
	paths = append([]string{fmt.Sprintf("/etc/%s", configFile)}, paths...)

	// iterate paths in reverse
	for i := len(paths) - 1; i >= 0; i-- {
		file := paths[i]
		err := f.LoadConfig(file, options)
		if err != nil {
			return err
		}
		if f.stop {
			break
		}
	}

	// apply defaults at the end to set any undefined fields
	if f.Defaults != nil {
		m := &merger{sourceFile: "default"}
		m.mergeStructs(
			reflect.ValueOf(options),
			reflect.ValueOf(f.Defaults),
		)
		f.PopulateEnv(options)
	}
	return nil
}

func (f *FigTree) LoadConfigBytes(config []byte, source string, options interface{}) (err error) {
	if !reflect.ValueOf(options).IsValid() {
		return fmt.Errorf("options argument [%#v] is not valid", options)
	}

	f.PopulateEnv(options)

	defer func(mapType, iface reflect.Type) {
		yaml.DefaultMapType = mapType
		yaml.IfaceType = iface
	}(yaml.DefaultMapType, yaml.IfaceType)

	yaml.DefaultMapType = reflect.TypeOf(map[string]interface{}{})
	yaml.IfaceType = yaml.DefaultMapType.Elem()

	m := &merger{sourceFile: source}
	type tmpOpts struct {
		Config ConfigOptions
	}

	tmp := reflect.New(reflect.ValueOf(options).Elem().Type()).Interface()
	// look for config settings first
	err = yaml.Unmarshal(config, m)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Unable to parse %s", source))
	}

	// then parse document into requested struct
	err = yaml.Unmarshal(config, tmp)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Unable to parse %s", source))
	}

	m.setSource(reflect.ValueOf(tmp))
	m.mergeStructs(
		reflect.ValueOf(options),
		reflect.ValueOf(tmp),
	)
	f.PopulateEnv(options)
	if m.Config.Stop {
		f.stop = true
		return nil
	}
	return nil
}

func (f *FigTree) LoadConfig(file string, options interface{}) (err error) {
	basePath, err := os.Getwd()
	if err != nil {
		return err
	}

	rel, err := filepath.Rel(basePath, file)
	if err != nil {
		rel = file
	}

	if stat, err := os.Stat(file); err == nil {
		if stat.Mode()&0111 == 0 {
			Log.Debugf("Loading config %s", file)
			if data, err := ioutil.ReadFile(file); err == nil {
				return f.LoadConfigBytes(data, rel, options)
			}
		} else {
			Log.Debugf("Found Executable Config file: %s", file)
			// it is executable, so run it and try to parse the output
			cmd := exec.Command(file)
			stdout := bytes.NewBufferString("")
			cmd.Stdout = stdout
			cmd.Stderr = bytes.NewBufferString("")
			if err := cmd.Run(); err != nil {
				return errors.Wrap(err, fmt.Sprintf("%s is exectuable, but it failed to execute:\n%s", file, cmd.Stderr))
			}
			return f.LoadConfigBytes(stdout.Bytes(), rel, options)
		}
	}
	return nil
}

var camelCaseWords = regexp.MustCompile("[0-9A-Za-z]+")

func camelCase(name string) string {
	words := camelCaseWords.FindAllString(name, -1)
	for i, word := range words {
		words[i] = strings.Title(word)
	}
	return strings.Join(words, "")

}

// Merge will attempt to merge the data from src into dst.  They shoud be either both maps or both structs.
// The structs do not need to have the same structure, but any field name that exists in both
// structs will must be the same type.
func Merge(dst, src interface{}) {
	m := &merger{sourceFile: "merge"}
	m.mergeStructs(reflect.ValueOf(dst), reflect.ValueOf(src))
}

// MakeMergeStruct will take multiple structs and return a pointer to a zero value for the
// anonymous struct that has all the public fields from all the structs merged into one struct.
// If there are multiple structs with the same field names, the first appearance of that name
// will be used.
func MakeMergeStruct(structs ...interface{}) interface{} {
	values := []reflect.Value{}
	for _, data := range structs {
		values = append(values, reflect.ValueOf(data))
	}
	return makeMergeStruct(values...).Interface()
}

func inlineField(field reflect.StructField) bool {
	if tag := field.Tag.Get("figtree"); tag != "" {
		return strings.HasSuffix(tag, ",inline")
	}
	if tag := field.Tag.Get("yaml"); tag != "" {
		return strings.HasSuffix(tag, ",inline")
	}
	return false
}

func makeMergeStruct(values ...reflect.Value) reflect.Value {
	foundFields := map[string]reflect.StructField{}
	for i := 0; i < len(values); i++ {
		v := values[i]
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		typ := v.Type()
		var field reflect.StructField
		if typ.Kind() == reflect.Struct {
			for i := 0; i < typ.NumField(); i++ {
				field = typ.Field(i)
				if field.PkgPath != "" {
					// unexported field, skip
					continue
				}
				if f, ok := foundFields[field.Name]; ok {
					Log.Debugf("attempting to merge %s with %s", f.Type.Name(), field.Type.Name())
					if f.Type.Kind() == reflect.Struct && field.Type.Kind() == reflect.Struct {
						if fName, fieldName := f.Type.Name(), field.Type.Name(); fName == "" || fieldName == "" || fName != fieldName {
							// we have 2 fields with the same name and they are both structs, so we need
							// to merge the existig struct with the new one in case they are different
							newval := makeMergeStruct(reflect.New(f.Type).Elem(), reflect.New(field.Type).Elem()).Elem()
							f.Type = newval.Type()
							foundFields[field.Name] = f
						}
					}
					// field already found, skip
					continue
				}
				if inlineField(field) {
					values = append(values, v.Field(i))
					continue
				}
				foundFields[field.Name] = field
			}
		} else if typ.Kind() == reflect.Map {
			for _, key := range v.MapKeys() {
				keyval := reflect.ValueOf(v.MapIndex(key).Interface())
				if keyval.Kind() == reflect.Ptr && keyval.Elem().Kind() == reflect.Map {
					keyval = makeMergeStruct(keyval.Elem())
				} else if keyval.Kind() == reflect.Map {
					keyval = makeMergeStruct(keyval).Elem()
				}
				var t reflect.Type
				if !keyval.IsValid() {
					// this nonsense is to create a generic `interface{}` type.  There is
					// probably an easier to do this, but it eludes me at the moment.
					var dummy interface{}
					t = reflect.ValueOf(&dummy).Elem().Type()
				} else {
					t = reflect.ValueOf(keyval.Interface()).Type()
				}
				field = reflect.StructField{
					Name: camelCase(key.String()),
					Type: t,
					Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s" yaml:"%s"`, key.String(), key.String())),
				}
				if f, ok := foundFields[field.Name]; ok {
					if f.Type.Kind() == reflect.Struct && t.Kind() == reflect.Struct {
						// we have 2 fields with the same name and they are both structs, so we need
						// to merge the existig struct with the new one in case they are different
						newval := makeMergeStruct(reflect.New(f.Type).Elem(), reflect.New(t).Elem()).Elem()
						f.Type = newval.Type()
						foundFields[field.Name] = f
					}
					// field already found, skip
					continue
				}
				foundFields[field.Name] = field
			}
		}
	}

	fields := []reflect.StructField{}
	for _, value := range foundFields {
		fields = append(fields, value)
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})
	newType := reflect.StructOf(fields)
	return reflect.New(newType)
}

func mapToStruct(src reflect.Value) reflect.Value {
	if src.Kind() != reflect.Map {
		return reflect.Value{}
	}

	dest := makeMergeStruct(src)
	if dest.Kind() == reflect.Ptr {
		dest = dest.Elem()
	}

	m := merger{}
	for _, key := range src.MapKeys() {
		structFieldName := camelCase(key.String())
		keyval := reflect.ValueOf(src.MapIndex(key).Interface())
		// skip invalid (ie nil) key values
		if !keyval.IsValid() {
			continue
		}
		if keyval.Kind() == reflect.Ptr && keyval.Elem().Kind() == reflect.Map {
			keyval = mapToStruct(keyval.Elem()).Addr()
			m.mergeStructs(dest.FieldByName(structFieldName), reflect.ValueOf(keyval.Interface()))
		} else if keyval.Kind() == reflect.Map {
			keyval = mapToStruct(keyval)
			m.mergeStructs(dest.FieldByName(structFieldName), reflect.ValueOf(keyval.Interface()))
		} else {
			dest.FieldByName(structFieldName).Set(reflect.ValueOf(keyval.Interface()))
		}
	}
	return dest
}

func structToMap(src reflect.Value) reflect.Value {
	if src.Kind() != reflect.Struct {
		return reflect.Value{}
	}

	dest := reflect.ValueOf(map[string]interface{}{})

	typ := src.Type()

	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		if structField.PkgPath != "" {
			// skip private fields
			continue
		}
		name := yamlFieldName(structField)
		dest.SetMapIndex(reflect.ValueOf(name), src.Field(i))
	}

	return dest
}

type ConfigOptions struct {
	Overwrite []string `json:"overwrite,omitempty" yaml:"overwrite,omitempty"`
	Stop      bool     `json:"stop,omitempty" yaml:"stop,omitempty"`
	// Merge     bool     `json:"merge,omitempty" yaml:"merge,omitempty"`
}

type merger struct {
	sourceFile string
	Config     ConfigOptions `json:"config,omitempty" yaml:"config,omitempty"`
}

func yamlFieldName(sf reflect.StructField) string {
	if tag, ok := sf.Tag.Lookup("yaml"); ok {
		// with yaml:"foobar,omitempty"
		// we just want to the "foobar" part
		parts := strings.Split(tag, ",")
		return parts[0]
	}
	// guess the field name from reversing camel case
	// so "FooBar" becomes "foo-bar"
	parts := camelcase.Split(sf.Name)
	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, "-")
}

func (m *merger) mustOverwrite(name string) bool {
	for _, prop := range m.Config.Overwrite {
		if name == prop {
			return true
		}
	}
	return false
}

func isDefault(v reflect.Value) bool {
	if v.CanAddr() {
		if option, ok := v.Addr().Interface().(Option); ok {
			if option.GetSource() == "default" {
				return true
			}
		}
	}
	return false
}

func isZero(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
}

func isSame(v1, v2 reflect.Value) bool {
	return reflect.DeepEqual(v1.Interface(), v2.Interface())
}

// recursively set the Source attribute of the Options
func (m *merger) setSource(v reflect.Value) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Map:
		for _, key := range v.MapKeys() {
			keyval := v.MapIndex(key)
			if keyval.Kind() == reflect.Struct && keyval.FieldByName("Source").IsValid() {
				// map values are immutable, so we need to copy the value
				// update the value, then re-insert the value to the map
				newval := reflect.New(keyval.Type())
				newval.Elem().Set(keyval)
				m.setSource(newval)
				v.SetMapIndex(key, newval.Elem())
			}
		}
	case reflect.Struct:
		if v.CanAddr() {
			if option, ok := v.Addr().Interface().(Option); ok {
				if option.IsDefined() {
					option.SetSource(m.sourceFile)
				}
				return
			}
		}
		for i := 0; i < v.NumField(); i++ {
			structField := v.Type().Field(i)
			// PkgPath is empty for upper case (exported) field names.
			if structField.PkgPath != "" {
				// unexported field, skipping
				continue
			}
			m.setSource(v.Field(i))
		}
	case reflect.Array:
		fallthrough
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			m.setSource(v.Index(i))
		}
	}
}

func (m *merger) assignValue(dest, src reflect.Value, overwrite bool) {
	if src.Type().AssignableTo(dest.Type()) {
		if (isZero(dest) || isDefault(dest) || overwrite) && !isZero(src) {
			dest.Set(src)
			return
		}
		return
	}
	if dest.CanAddr() {
		if option, ok := dest.Addr().Interface().(Option); ok {
			destOptionValue := reflect.ValueOf(option.GetValue())
			// map interface type to real-ish type:
			src = reflect.ValueOf(src.Interface())
			if !src.IsValid() {
				Log.Debugf("assignValue: src isValid: %t", src.IsValid())
				return
			}
			if src.Type().AssignableTo(destOptionValue.Type()) {
				option.SetValue(src.Interface())
				option.SetSource(m.sourceFile)
				Log.Debugf("assignValue: assigned %#v to %#v", destOptionValue, src)
				return
			} else {
				panic(fmt.Errorf("%s is not assinable to %s", src.Type(), destOptionValue.Type()))
			}
		}
	}
	// make copy so we can reliably Addr it to see if it fits the
	// Option interface.
	srcCopy := reflect.New(src.Type()).Elem()
	srcCopy.Set(src)
	if option, ok := srcCopy.Addr().Interface().(Option); ok {
		srcOptionValue := reflect.ValueOf(option.GetValue())
		if srcOptionValue.Type().AssignableTo(dest.Type()) {
			m.assignValue(dest, srcOptionValue, overwrite)
			return
		} else {
			panic(fmt.Errorf("%s is not assinable to %s", srcOptionValue.Type(), dest.Type()))
		}
	}
}

func fromInterface(v reflect.Value) (reflect.Value, func()) {
	if v.Kind() == reflect.Interface {
		realV := reflect.ValueOf(v.Interface())
		if !realV.IsValid() {
			realV = reflect.New(v.Type()).Elem()
			v.Set(realV)
			return v, func() {}
		}
		tmp := reflect.New(realV.Type()).Elem()
		tmp.Set(realV)
		return tmp, func() {
			v.Set(tmp)
		}
	}
	return v, func() {}
}

func (m *merger) mergeStructs(ov, nv reflect.Value) {
	ov = reflect.Indirect(ov)
	nv = reflect.Indirect(nv)

	ov, restore := fromInterface(ov)
	defer restore()

	if nv.Kind() == reflect.Interface {
		nv = reflect.ValueOf(nv.Interface())
	}

	if ov.Kind() == reflect.Map {
		if nv.Kind() == reflect.Struct {
			nv = structToMap(nv)
		}
		m.mergeMaps(ov, nv)
		return
	}

	if ov.Kind() == reflect.Struct && nv.Kind() == reflect.Map {
		nv = mapToStruct(nv)
	}

	if !ov.IsValid() || !nv.IsValid() {
		Log.Debugf("Valid: ov:%v nv:%t", ov.IsValid(), nv.IsValid())
		return
	}

	for i := 0; i < nv.NumField(); i++ {
		nvField := nv.Field(i)
		if nvField.Kind() == reflect.Interface {
			nvField = reflect.ValueOf(nvField.Interface())
		}
		if !nvField.IsValid() {
			continue
		}

		nvStructField := nv.Type().Field(i)
		ovStructField, ok := ov.Type().FieldByName(nvStructField.Name)
		if !ok {
			if nvStructField.Anonymous {
				// this is an embedded struct, and the destination does not contain
				// the same embeded struct, so try to merge the embedded struct
				// directly with the destination
				m.mergeStructs(ov, nvField)
				continue
			}
			// if original value does not have the same struct field
			// then just skip this field.
			continue
		}

		// PkgPath is empty for upper case (exported) field names.
		if ovStructField.PkgPath != "" || nvStructField.PkgPath != "" {
			// unexported field, skipping
			continue
		}
		fieldName := yamlFieldName(ovStructField)

		ovField := ov.FieldByName(nvStructField.Name)
		ovField, restore := fromInterface(ovField)
		defer restore()

		if (isZero(ovField) || isDefault(ovField) || m.mustOverwrite(fieldName)) && !isSame(ovField, nvField) {
			Log.Debugf("Setting %s to %#v", nv.Type().Field(i).Name, nvField.Interface())
			m.assignValue(ovField, nvField, m.mustOverwrite(fieldName))
		}
		switch ovField.Kind() {
		case reflect.Map:
			Log.Debugf("Merging Map: %#v with %#v", ovField, nvField)
			m.mergeStructs(ovField, nvField)
		case reflect.Slice:
			if nvField.Len() > 0 {
				Log.Debugf("Merging Slice: %#v with %#v", ovField, nvField)
				ovField.Set(m.mergeArrays(ovField, nvField))
			}
		case reflect.Array:
			if nvField.Len() > 0 {
				Log.Debugf("Merging Array: %v with %v", ovField, nvField)
				ovField.Set(m.mergeArrays(ovField, nvField))
			}
		case reflect.Struct:
			// only merge structs if they are not an Option type:
			if _, ok := ovField.Addr().Interface().(Option); !ok {
				Log.Debugf("Merging Struct: %v with %v", ovField, nvField)
				m.mergeStructs(ovField, nvField)
			}
		}
	}
}

func (m *merger) mergeMaps(ov, nv reflect.Value) {
	for _, key := range nv.MapKeys() {
		if !ov.MapIndex(key).IsValid() {
			ovElem := reflect.New(ov.Type().Elem()).Elem()
			m.assignValue(ovElem, nv.MapIndex(key), false)
			if ov.IsNil() {
				if !ov.CanSet() {
					continue
				}
				ov.Set(reflect.MakeMap(ov.Type()))
			}
			Log.Debugf("Setting %v to %#v", key.Interface(), ovElem.Interface())
			ov.SetMapIndex(key, ovElem)
		} else {
			ovi := reflect.ValueOf(ov.MapIndex(key).Interface())
			nvi := reflect.ValueOf(nv.MapIndex(key).Interface())
			if !nvi.IsValid() {
				continue
			}
			switch ovi.Kind() {
			case reflect.Map:
				Log.Debugf("Merging: %v with %v", ovi.Interface(), nvi.Interface())
				m.mergeStructs(ovi, nvi)
			case reflect.Slice:
				Log.Debugf("Merging: %v with %v", ovi.Interface(), nvi.Interface())
				ov.SetMapIndex(key, m.mergeArrays(ovi, nvi))
			case reflect.Array:
				Log.Debugf("Merging: %v with %v", ovi.Interface(), nvi.Interface())
				ov.SetMapIndex(key, m.mergeArrays(ovi, nvi))
			default:
				if isZero(ovi) {
					if !ovi.IsValid() || nvi.Type().AssignableTo(ovi.Type()) {
						ov.SetMapIndex(key, nvi)
					} else {
						// to check for the Option interface we need the Addr of the value, but
						// we cannot take the Addr of a map value, so we have to first copy
						// it, meh not optimal
						newVal := reflect.New(nvi.Type())
						newVal.Elem().Set(nvi)
						if nOption, ok := newVal.Interface().(Option); ok {
							ov.SetMapIndex(key, reflect.ValueOf(nOption.GetValue()))
							continue
						}
						panic(fmt.Errorf("map value %T is not assignable to %T", nvi.Interface(), ovi.Interface()))
					}

				}
			}
		}
	}
}

func (m *merger) mergeArrays(ov, nv reflect.Value) reflect.Value {
Outer:
	for ni := 0; ni < nv.Len(); ni++ {
		niv := nv.Index(ni)

		nvElem := reflect.New(ov.Type().Elem()).Elem()
		m.assignValue(nvElem, niv, false)

		for oi := 0; oi < ov.Len(); oi++ {
			oiv := ov.Index(oi)
			if oiv.CanAddr() && nvElem.CanAddr() {
				if oOption, ok := oiv.Addr().Interface().(Option); ok {
					if nOption, ok := nvElem.Addr().Interface().(Option); ok {
						if reflect.DeepEqual(oOption.GetValue(), nOption.GetValue()) {
							continue Outer
						}
					}
				}
			}
			if reflect.DeepEqual(nvElem.Interface(), oiv.Interface()) {
				continue Outer
			}
		}
		Log.Debugf("Appending %v to %v", nvElem.Interface(), ov)
		ov = reflect.Append(ov, nvElem)
	}
	return ov
}

func (f *FigTree) formatEnvName(name string) string {
	name = fmt.Sprintf("%s_%s", f.EnvPrefix, strings.ToUpper(name))

	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) || unicode.IsLetter(r) {
			return r
		}
		return '_'
	}, name)
}

func (f *FigTree) formatEnvValue(value reflect.Value) (string, bool) {
	switch t := value.Interface().(type) {
	case string:
		return t, true
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return fmt.Sprintf("%v", t), true
	default:
		switch value.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
			if value.IsNil() {
				return "", false
			}
		}
		if t == nil {
			return "", false
		}
		type definable interface {
			IsDefined() bool
		}
		if def, ok := t.(definable); ok {
			// skip fields that are not defined
			if !def.IsDefined() {
				return "", false
			}
		}
		type gettable interface {
			GetValue() interface{}
		}
		if get, ok := t.(gettable); ok {
			return fmt.Sprintf("%v", get.GetValue()), true
		} else {
			if b, err := json.Marshal(t); err == nil {
				val := strings.TrimSpace(string(b))
				if val == "null" {
					return "", true
				}
				return val, true
			}
		}
	}
	return "", false
}

func (f *FigTree) PopulateEnv(data interface{}) {
	options := reflect.ValueOf(data)
	if options.Kind() == reflect.Ptr {
		options = reflect.ValueOf(options.Elem().Interface())
	}
	if options.Kind() == reflect.Map {
		for _, key := range options.MapKeys() {
			if strKey, ok := key.Interface().(string); ok {
				// first chunk up string so that `foo-bar` becomes ["foo", "bar"]
				parts := strings.FieldsFunc(strKey, func(r rune) bool {
					return !unicode.IsLetter(r) && !unicode.IsNumber(r)
				})
				// now for each chunk split again on camelcase so ["fooBar", "baz"]
				// becomes ["foo", "Bar", "baz"]
				allParts := []string{}
				for _, part := range parts {
					allParts = append(allParts, camelcase.Split(part)...)
				}

				name := strings.Join(allParts, "_")
				envName := f.formatEnvName(name)
				val, ok := f.formatEnvValue(options.MapIndex(key))
				if ok {
					os.Setenv(envName, val)
				} else {
					os.Unsetenv(envName)
				}
			}
		}
	} else if options.Kind() == reflect.Struct {
		for i := 0; i < options.NumField(); i++ {
			structField := options.Type().Field(i)
			// PkgPath is empty for upper case (exported) field names.
			if structField.PkgPath != "" {
				// unexported field, skipping
				continue
			}

			name := strings.Join(camelcase.Split(structField.Name), "_")

			if tag := structField.Tag.Get("figtree"); tag != "" {
				if strings.HasSuffix(tag, ",inline") {
					// if we have a tag like: `figtree:",inline"` then we
					// want to the field as a top level member and not serialize
					// the raw struct to json, so just recurse here
					f.PopulateEnv(options.Field(i).Interface())
					continue
				}
				// next look for `figtree:"env,..."` to set the env name to that
				parts := strings.Split(tag, ",")
				if len(parts) > 0 {
					// if the env name is "-" then we should not populate this data into the env
					if parts[0] == "-" {
						continue
					}
					name = parts[0]
				}
			}

			envName := f.formatEnvName(name)
			val, ok := f.formatEnvValue(options.Field(i))
			if ok {
				os.Setenv(envName, val)
			} else {
				os.Unsetenv(envName)
			}
		}
	}
}
