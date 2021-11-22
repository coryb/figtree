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
	"strconv"
	"strings"
	"unicode"

	"github.com/fatih/camelcase"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Logger interface {
	Debugf(format string, args ...interface{})
}

type nullLogger struct{}

func (*nullLogger) Debugf(string, ...interface{}) {}

var Log Logger = &nullLogger{}

func defaultApplyChangeSet(changeSet map[string]*string) error {
	for k, v := range changeSet {
		if v != nil {
			os.Setenv(k, *v)
		} else {
			os.Unsetenv(k)
		}
	}
	return nil
}

type Option func(*FigTree)

func WithHome(home string) Option {
	return func(f *FigTree) {
		f.home = home
	}
}

func WithCwd(cwd string) Option {
	return func(f *FigTree) {
		f.workDir = cwd
	}
}

func WithEnvPrefix(env string) Option {
	return func(f *FigTree) {
		f.envPrefix = env
	}
}

func WithConfigDir(dir string) Option {
	return func(f *FigTree) {
		f.configDir = dir
	}
}

type ChangeSetFunc func(map[string]*string) error

func WithApplyChangeSet(apply ChangeSetFunc) Option {
	return func(f *FigTree) {
		f.applyChangeSet = apply
	}
}

type PreProcessor func([]byte) ([]byte, error)

func WithPreProcessor(pp PreProcessor) Option {
	return func(f *FigTree) {
		f.preProcessor = pp
	}
}

type FilterOut func([]byte) bool

func WithFilterOut(filt FilterOut) Option {
	return func(f *FigTree) {
		f.filterOut = filt
	}
}

func defaultFilterOut(f *FigTree) FilterOut {
	// looking for:
	// ```
	// config:
	//   stop: true|false
	// ```
	configStop := struct {
		Config struct {
			Stop bool `json:"stop" yaml:"stop"`
		} `json:"config" yaml:"config"`
	}{}
	return func(config []byte) bool {
		// if previous parse found a stop we should abort here
		if configStop.Config.Stop {
			return true
		}
		// now check if current doc has a stop
		f.unmarshal(config, &configStop)
		// even if current doc has a stop, we should continue to
		// process it, we dont want to process the "next" document
		return false
	}
}

func WithUnmarshaller(unmarshaller func(in []byte, out interface{}) error) Option {
	return func(f *FigTree) {
		f.unmarshal = unmarshaller
	}
}

func WithoutExec() Option {
	return func(f *FigTree) {
		f.exec = false
	}
}

type FigTree struct {
	home           string
	workDir        string
	configDir      string
	envPrefix      string
	preProcessor   PreProcessor
	applyChangeSet ChangeSetFunc
	exec           bool
	filterOut      FilterOut
	unmarshal      func(in []byte, out interface{}) error
}

func NewFigTree(opts ...Option) *FigTree {
	wd, _ := os.Getwd()
	fig := &FigTree{
		home:           os.Getenv("HOME"),
		workDir:        wd,
		envPrefix:      "FIGTREE",
		applyChangeSet: defaultApplyChangeSet,
		exec:           true,
		unmarshal:      yaml.Unmarshal,
	}
	for _, opt := range opts {
		opt(fig)
	}
	return fig
}

func (f *FigTree) WithHome(home string) {
	WithHome(home)(f)
}

func (f *FigTree) WithCwd(cwd string) {
	WithCwd(cwd)(f)
}

func (f *FigTree) WithEnvPrefix(env string) {
	WithEnvPrefix(env)(f)
}

func (f *FigTree) WithConfigDir(dir string) {
	WithConfigDir(dir)(f)
}

func (f *FigTree) WithPreProcessor(pp PreProcessor) {
	WithPreProcessor(pp)(f)
}

func (f *FigTree) WithFilterOut(filt FilterOut) {
	WithFilterOut(filt)(f)
}

func (f *FigTree) WithUnmarshaller(unmarshaller func(in []byte, out interface{}) error) {
	WithUnmarshaller(unmarshaller)(f)
}

func (f *FigTree) WithApplyChangeSet(apply ChangeSetFunc) {
	WithApplyChangeSet(apply)(f)
}

func (f *FigTree) WithIgnoreChangeSet() {
	WithApplyChangeSet(func(_ map[string]*string) error {
		return nil
	})(f)
}

func (f *FigTree) WithoutExec() {
	WithoutExec()(f)
}

func (f *FigTree) Copy() *FigTree {
	cp := *f
	return &cp
}

func (f *FigTree) LoadAllConfigs(configFile string, options interface{}) error {
	if f.configDir != "" {
		configFile = path.Join(f.configDir, configFile)
	}

	paths := FindParentPaths(f.home, f.workDir, configFile)
	paths = append([]string{fmt.Sprintf("/etc/%s", configFile)}, paths...)

	configSources := []ConfigSource{}
	// iterate paths in reverse
	for i := len(paths) - 1; i >= 0; i-- {
		file := paths[i]
		cs, err := f.ReadFile(file)
		if err != nil {
			return err
		}
		if cs == nil {
			// no file contents to parse, file likely does not exist
			continue
		}
		configSources = append(configSources, *cs)
	}
	return f.LoadAllConfigSources(configSources, options)
}

type ConfigSource struct {
	Config   []byte
	Filename string
}

func (f *FigTree) LoadAllConfigSources(sources []ConfigSource, options interface{}) error {
	m := NewMerger()
	filterOut := f.filterOut
	if filterOut == nil {
		filterOut = defaultFilterOut(f)
	}

	for _, source := range sources {
		// automatically skip empty configs
		if len(source.Config) == 0 {
			continue
		}
		skip := filterOut(source.Config)
		if skip {
			continue
		}

		m.sourceFile = source.Filename
		err := f.loadConfigBytes(m, source.Config, options)
		if err != nil {
			return err
		}
		m.advance()
	}
	return nil
}

func (f *FigTree) LoadConfigBytes(config []byte, source string, options interface{}) error {
	m := NewMerger(WithSourceFile(source))
	return f.loadConfigBytes(m, config, options)
}

func (f *FigTree) loadConfigBytes(m *Merger, config []byte, options interface{}) error {
	if !reflect.ValueOf(options).IsValid() {
		return fmt.Errorf("options argument [%#v] is not valid", options)
	}

	var err error
	if f.preProcessor != nil {
		config, err = f.preProcessor(config)
		if err != nil {
			return errors.Wrapf(err, "Failed to process config file: %s", m.sourceFile)
		}
	}

	tmp := reflect.New(reflect.ValueOf(options).Elem().Type()).Interface()
	// look for config settings first
	err = f.unmarshal(config, m)
	if err != nil {
		return errors.Wrapf(err, "Unable to parse %s", m.sourceFile)
	}

	// then parse document into requested struct
	err = f.unmarshal(config, tmp)
	if err != nil {
		return errors.Wrapf(err, "Unable to parse %s", m.sourceFile)
	}

	m.setSource(reflect.ValueOf(tmp))
	m.mergeStructs(
		reflect.ValueOf(options),
		reflect.ValueOf(tmp),
	)
	changeSet := f.PopulateEnv(options)
	return f.applyChangeSet(changeSet)
}

func (f *FigTree) LoadConfig(file string, options interface{}) error {
	cs, err := f.ReadFile(file)
	if err != nil {
		return err
	}
	if cs == nil {
		// no file contents to parse, file likely does not exist
		return nil
	}
	return f.LoadConfigBytes(cs.Config, cs.Filename, options)
}

// ReadFile will return a ConfigSource for given file path.  If the
// file is executable (and WithoutExec was not used), it will execute
// the file and return the stdout otherwise it will return the file
// contents directly.
func (f *FigTree) ReadFile(file string) (*ConfigSource, error) {
	rel, err := filepath.Rel(f.workDir, file)
	if err != nil {
		rel = file
	}

	if stat, err := os.Stat(file); err == nil {
		if stat.Mode()&0111 == 0 || !f.exec {
			Log.Debugf("Reading config %s", file)
			data, err := ioutil.ReadFile(file)
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to read %s", rel)
			}
			return &ConfigSource{
				Config:   data,
				Filename: rel,
			}, nil
		} else {
			Log.Debugf("Found Executable Config file: %s", file)
			// it is executable, so run it and try to parse the output
			cmd := exec.Command(file)
			stdout := bytes.NewBufferString("")
			cmd.Stdout = stdout
			cmd.Stderr = bytes.NewBufferString("")
			if err := cmd.Run(); err != nil {
				return nil, errors.Wrapf(err, "%s is exectuable, but it failed to execute:\n%s", file, cmd.Stderr)
			}
			return &ConfigSource{
				Config:   stdout.Bytes(),
				Filename: rel,
			}, nil
		}
	}
	return nil, nil
}

func FindParentPaths(homedir, cwd, fileName string) []string {
	paths := make([]string, 0)
	if filepath.IsAbs(fileName) {
		// dont recursively look for files when fileName is an abspath
		_, err := os.Stat(fileName)
		if err == nil {
			paths = append(paths, fileName)
		}
		return paths
	}

	// special case if homedir is not in current path then check there anyway
	if !strings.HasPrefix(cwd, homedir) {
		file := path.Join(homedir, fileName)
		if _, err := os.Stat(file); err == nil {
			paths = append(paths, filepath.FromSlash(file))
		}
	}

	var dir string
	for _, part := range strings.Split(cwd, string(os.PathSeparator)) {
		if part == "" && dir == "" {
			dir = "/"
		} else {
			dir = path.Join(dir, part)
		}
		file := path.Join(dir, fileName)
		if _, err := os.Stat(file); err == nil {
			paths = append(paths, filepath.FromSlash(file))
		}
	}
	return paths
}

func (f *FigTree) FindParentPaths(fileName string) []string {
	return FindParentPaths(f.home, f.workDir, fileName)
}

var camelCaseWords = regexp.MustCompile("[0-9A-Za-z]+")

func camelCase(name string) string {
	words := camelCaseWords.FindAllString(name, -1)
	for i, word := range words {
		words[i] = strings.Title(word)
	}
	return strings.Join(words, "")

}

type Merger struct {
	sourceFile  string
	preserveMap map[string]struct{}
	Config      ConfigOptions `json:"config,omitempty" yaml:"config,omitempty"`
	ignore      []string
}

type MergeOption func(*Merger)

func WithSourceFile(source string) MergeOption {
	return func(m *Merger) {
		m.sourceFile = source
	}
}

func PreserveMap(keys ...string) MergeOption {
	return func(m *Merger) {
		for _, key := range keys {
			m.preserveMap[key] = struct{}{}
		}
	}
}

func NewMerger(options ...MergeOption) *Merger {
	m := &Merger{
		sourceFile:  "merge",
		preserveMap: make(map[string]struct{}),
	}
	for _, opt := range options {
		opt(m)
	}
	return m
}

// advance will move all the current overwrite properties to
// the ignore properties, then reset the overwrite properties.
// This is used after a document has be processed so the next
// document does not modify overwritten fields.
func (m *Merger) advance() {
	for _, overwrite := range m.Config.Overwrite {
		found := false
		for _, ignore := range m.ignore {
			if ignore == overwrite {
				found = true
				break
			}
		}
		if !found {
			m.ignore = append(m.ignore, overwrite)
		}
	}
	m.Config.Overwrite = nil
}

// Merge will attempt to merge the data from src into dst.  They shoud be either both maps or both structs.
// The structs do not need to have the same structure, but any field name that exists in both
// structs will must be the same type.
func Merge(dst, src interface{}) {
	m := NewMerger()
	m.mergeStructs(reflect.ValueOf(dst), reflect.ValueOf(src))
}

// MakeMergeStruct will take multiple structs and return a pointer to a zero value for the
// anonymous struct that has all the public fields from all the structs merged into one struct.
// If there are multiple structs with the same field names, the first appearance of that name
// will be used.
func MakeMergeStruct(structs ...interface{}) interface{} {
	m := NewMerger()
	return m.MakeMergeStruct(structs...)
}

func (m *Merger) MakeMergeStruct(structs ...interface{}) interface{} {
	values := []reflect.Value{}
	for _, data := range structs {
		values = append(values, reflect.ValueOf(data))
	}
	return m.makeMergeStruct(values...).Interface()
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

func (m *Merger) makeMergeStruct(values ...reflect.Value) reflect.Value {
	foundFields := map[string]reflect.StructField{}
	for i := 0; i < len(values); i++ {
		v := values[i]
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		typ := v.Type()
		var field reflect.StructField
		if typ.Kind() == reflect.Struct {
			for j := 0; j < typ.NumField(); j++ {
				field = typ.Field(j)
				if field.PkgPath != "" {
					// unexported field, skip
					continue
				}

				field.Name = canonicalFieldName(field)

				if f, ok := foundFields[field.Name]; ok {
					if f.Type.Kind() == reflect.Struct && field.Type.Kind() == reflect.Struct {
						if fName, fieldName := f.Type.Name(), field.Type.Name(); fName == "" || fieldName == "" || fName != fieldName {
							// we have 2 fields with the same name and they are both structs, so we need
							// to merge the existing struct with the new one in case they are different
							newval := m.makeMergeStruct(reflect.New(f.Type).Elem(), reflect.New(field.Type).Elem()).Elem()
							f.Type = newval.Type()
							foundFields[field.Name] = f
						}
					}
					// field already found, skip
					continue
				}
				if inlineField(field) {
					// insert inline after this value, it will have a higher
					// "type" priority than later values
					values = append(values[:i+1], append([]reflect.Value{v.Field(j)}, values[i+1:]...)...)
					continue
				}
				foundFields[field.Name] = field
			}
		} else if typ.Kind() == reflect.Map {
			for _, key := range v.MapKeys() {
				keyval := reflect.ValueOf(v.MapIndex(key).Interface())
				if _, ok := m.preserveMap[key.String()]; !ok {
					if keyval.Kind() == reflect.Ptr && keyval.Elem().Kind() == reflect.Map {
						keyval = m.makeMergeStruct(keyval.Elem())
					} else if keyval.Kind() == reflect.Map {
						keyval = m.makeMergeStruct(keyval).Elem()
					}
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
						if fName, tName := f.Type.Name(), t.Name(); fName == "" || tName == "" || fName != tName {
							// we have 2 fields with the same name and they are both structs, so we need
							// to merge the existig struct with the new one in case they are different
							newval := m.makeMergeStruct(reflect.New(f.Type).Elem(), reflect.New(t).Elem()).Elem()
							f.Type = newval.Type()
							foundFields[field.Name] = f
						}
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

func (m *Merger) mapToStruct(src reflect.Value) reflect.Value {
	if src.Kind() != reflect.Map {
		return reflect.Value{}
	}

	dest := m.makeMergeStruct(src)
	if dest.Kind() == reflect.Ptr {
		dest = dest.Elem()
	}

	for _, key := range src.MapKeys() {
		structFieldName := camelCase(key.String())
		keyval := reflect.ValueOf(src.MapIndex(key).Interface())
		// skip invalid (ie nil) key values
		if !keyval.IsValid() {
			continue
		}
		if keyval.Kind() == reflect.Ptr && keyval.Elem().Kind() == reflect.Map {
			keyval = m.mapToStruct(keyval.Elem()).Addr()
			m.mergeStructs(dest.FieldByName(structFieldName), reflect.ValueOf(keyval.Interface()))
		} else if keyval.Kind() == reflect.Map {
			keyval = m.mapToStruct(keyval)
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
}

func yamlFieldName(sf reflect.StructField) string {
	if tag, ok := sf.Tag.Lookup("yaml"); ok {
		// with yaml:"foobar,omitempty"
		// we just want to the "foobar" part
		parts := strings.Split(tag, ",")
		if parts[0] != "" && parts[0] != "-" {
			return parts[0]
		}
	}
	// guess the field name from reversing camel case
	// so "FooBar" becomes "foo-bar"
	parts := camelcase.Split(sf.Name)
	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, "-")
}

func canonicalFieldName(sf reflect.StructField) string {
	if tag, ok := sf.Tag.Lookup("figtree"); ok {
		for _, part := range strings.Split(tag, ",") {
			if strings.HasPrefix(part, "name=") {
				return strings.TrimPrefix(part, "name=")
			}
		}
	}

	// For consistency with YAML data, determine a canonical field name
	// based on the YAML tag. Do not rely on the Go struct field name unless
	// there is no YAML tag.
	return camelCase(yamlFieldName(sf))
}

func (m *Merger) mustOverwrite(name string) bool {
	for _, prop := range m.Config.Overwrite {
		if name == prop {
			return true
		}
	}
	return false
}

func (m *Merger) mustIgnore(name string) bool {
	for _, prop := range m.ignore {
		if name == prop {
			return true
		}
	}
	return false
}

func isDefault(v reflect.Value) bool {
	if v.CanAddr() {
		if option, ok := v.Addr().Interface().(option); ok {
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
func (m *Merger) setSource(v reflect.Value) {
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
			if option, ok := v.Addr().Interface().(option); ok {
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

func (m *Merger) assignValue(dest, src reflect.Value, overwrite bool) {
	if src.Type().AssignableTo(dest.Type()) {
		shouldAssignDest := overwrite || isZero(dest) || (isDefault(dest) && !isDefault(src))
		isValidSrc := !isZero(src)
		if shouldAssignDest && isValidSrc {
			if src.Kind() == reflect.Map {
				// maps are mutable, so create a brand new shiny one
				dup := reflect.New(src.Type()).Elem()
				m.mergeMaps(dup, src)
				dest.Set(dup)
			} else {
				dest.Set(src)
			}
			return
		}
		return
	}
	if dest.CanAddr() {
		if option, ok := dest.Addr().Interface().(option); ok {
			destOptionValue := reflect.ValueOf(option.GetValue())
			// map interface type to real-ish type:
			src = reflect.ValueOf(src.Interface())
			if !src.IsValid() {
				Log.Debugf("assignValue: src isValid: %t", src.IsValid())
				return
			}
			// if destOptionValue is a Zero value then the Type call will panic.  If it
			// dest is zero, we should just overwrite it with src anyway.
			if isZero(destOptionValue) || src.Type().AssignableTo(destOptionValue.Type()) {
				option.SetValue(src.Interface())
				option.SetSource(m.sourceFile)
				Log.Debugf("assignValue: assigned %#v to %#v", destOptionValue, src)
				return
			}
			if destOptionValue.Kind() == reflect.Bool && src.Kind() == reflect.String {
				b, err := strconv.ParseBool(src.Interface().(string))
				if err != nil {
					panic(fmt.Errorf("%s is not assignable to %s, invalid bool value: %s", src.Type(), destOptionValue.Type(), err))
				}
				option.SetValue(b)
				option.SetSource(m.sourceFile)
				Log.Debugf("assignValue: assigned %#v to %#v", destOptionValue, b)
				return
			}
			if destOptionValue.Kind() == reflect.String && src.Kind() != reflect.String {
				option.SetValue(fmt.Sprintf("%v", src.Interface()))
				option.SetSource(m.sourceFile)
				Log.Debugf("assignValue: assigned %#v to %#v", destOptionValue, src)
				return
			}
			panic(fmt.Errorf("%s is not assignable to %s", src.Type(), destOptionValue.Type()))
		}
	}
	// make copy so we can reliably Addr it to see if it fits the
	// Option interface.
	srcCopy := reflect.New(src.Type()).Elem()
	srcCopy.Set(src)
	if option, ok := srcCopy.Addr().Interface().(option); ok {
		srcOptionValue := reflect.ValueOf(option.GetValue())
		if srcOptionValue.Type().AssignableTo(dest.Type()) {
			m.assignValue(dest, srcOptionValue, overwrite)
			return
		} else {
			panic(fmt.Errorf("%s is not assignable to %s", srcOptionValue.Type(), dest.Type()))
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

func (m *Merger) mergeStructs(ov, nv reflect.Value) {
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
		nv = m.mapToStruct(nv)
	}

	if !ov.IsValid() || !nv.IsValid() {
		Log.Debugf("Valid: ov:%v nv:%t", ov.IsValid(), nv.IsValid())
		return
	}

	ovFieldTypesByYAML := make(map[string]reflect.StructField)
	ovFieldValuesByYAML := make(map[string]reflect.Value)

	var populateYAMLMaps func(reflect.Value)
	populateYAMLMaps = func(ov reflect.Value) {
		for i := 0; i < ov.NumField(); i++ {
			fieldType := ov.Type().Field(i)
			yamlName := yamlFieldName(fieldType)
			if _, ok := ovFieldTypesByYAML[yamlName]; !ok {
				ovFieldTypesByYAML[yamlName] = fieldType
				ovFieldValuesByYAML[yamlName] = ov.Field(i)
			}
		}

		for i := 0; i < ov.NumField(); i++ {
			fieldType := ov.Type().Field(i)
			if fieldType.Anonymous && reflect.Indirect(ov.Field(i)).Type().Kind() == reflect.Struct {
				populateYAMLMaps(reflect.Indirect(ov.Field(i)))
			}
		}
	}
	populateYAMLMaps(ov)

	for i := 0; i < nv.NumField(); i++ {
		nvField := nv.Field(i)
		if nvField.Kind() == reflect.Interface {
			nvField = reflect.ValueOf(nvField.Interface())
		}
		if !nvField.IsValid() {
			continue
		}

		nvStructField := nv.Type().Field(i)
		fieldName := yamlFieldName(nvStructField)
		ovStructField, ok := ovFieldTypesByYAML[fieldName]
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

		ovField := ovFieldValuesByYAML[fieldName]
		ovField, restore := fromInterface(ovField)
		defer restore()

		if m.mustIgnore(fieldName) {
			continue
		}

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
			if _, ok := ovField.Addr().Interface().(option); !ok {
				Log.Debugf("Merging Struct: %v with %v", ovField, nvField)
				m.mergeStructs(ovField, nvField)
			}
		}
	}
}

func (m *Merger) mergeMaps(ov, nv reflect.Value) {
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
						if nOption, ok := newVal.Interface().(option); ok {
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

func (m *Merger) mergeArrays(ov, nv reflect.Value) reflect.Value {
	var zero interface{}
Outer:
	for ni := 0; ni < nv.Len(); ni++ {
		niv := nv.Index(ni)

		n := niv
		if n.CanAddr() {
			if nOption, ok := n.Addr().Interface().(option); ok {
				if !nOption.IsDefined() {
					continue
				}
				n = reflect.ValueOf(nOption.GetValue())
			}
		}

		if reflect.DeepEqual(n.Interface(), zero) {
			continue
		}

		for oi := 0; oi < ov.Len(); oi++ {
			o := ov.Index(oi)
			if o.CanAddr() {
				if oOption, ok := o.Addr().Interface().(option); ok {
					o = reflect.ValueOf(oOption.GetValue())
				}
			}
			if reflect.DeepEqual(n.Interface(), o.Interface()) {
				continue Outer
			}
		}

		nvElem := reflect.New(ov.Type().Elem()).Elem()
		m.assignValue(nvElem, niv, false)

		Log.Debugf("Appending %v to %v", nvElem.Interface(), ov)
		ov = reflect.Append(ov, nvElem)
	}
	return ov
}

func (f *FigTree) formatEnvName(name string) string {
	name = fmt.Sprintf("%s_%s", f.envPrefix, strings.ToUpper(name))

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

func (f *FigTree) PopulateEnv(data interface{}) (changeSet map[string]*string) {
	changeSet = make(map[string]*string)

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
					changeSet[envName] = &val
				} else {
					changeSet[envName] = nil
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

			envNames := []string{strings.Join(camelcase.Split(structField.Name), "_")}
			formatName := true
			if tag := structField.Tag.Get("figtree"); tag != "" {
				if strings.Contains(tag, ",inline") {
					// if we have a tag like: `figtree:",inline"` then we
					// want to the field as a top level member and not serialize
					// the raw struct to json, so just recurse here
					nestedEnvSet := f.PopulateEnv(options.Field(i).Interface())
					for k, v := range nestedEnvSet {
						changeSet[k] = v
					}
					continue
				}
				if strings.Contains(tag, ",raw") {
					formatName = false
				}
				// next look for `figtree:"env,..."` to set the env name to that
				parts := strings.Split(tag, ",")
				if len(parts) > 0 {
					// if the env name is "-" then we should not populate this data into the env
					if parts[0] == "-" {
						continue
					}
					for _, part := range parts {
						if strings.HasPrefix(part, "name=") {
							continue
						}
						envNames = strings.Split(part, ";")
						break
					}
				}
			}
			for _, name := range envNames {
				envName := name
				if formatName {
					envName = f.formatEnvName(name)
				}
				val, ok := f.formatEnvValue(options.Field(i))
				if ok {
					changeSet[envName] = &val
				} else {
					changeSet[envName] = nil
				}
			}
		}
	}

	return changeSet
}
