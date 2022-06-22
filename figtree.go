package figtree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

	"emperror.dev/errors"
	"github.com/coryb/walky"
	"github.com/fatih/camelcase"
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

type CreateOption func(*FigTree)

func WithHome(home string) CreateOption {
	return func(f *FigTree) {
		f.home = home
	}
}

func WithCwd(cwd string) CreateOption {
	return func(f *FigTree) {
		f.workDir = cwd
	}
}

func WithEnvPrefix(env string) CreateOption {
	return func(f *FigTree) {
		f.envPrefix = env
	}
}

func WithConfigDir(dir string) CreateOption {
	return func(f *FigTree) {
		f.configDir = dir
	}
}

type ChangeSetFunc func(map[string]*string) error

func WithApplyChangeSet(apply ChangeSetFunc) CreateOption {
	return func(f *FigTree) {
		f.applyChangeSet = apply
	}
}

type PreProcessor func(*yaml.Node) error

func WithPreProcessor(pp PreProcessor) CreateOption {
	return func(f *FigTree) {
		f.preProcessor = pp
	}
}

type FilterOut func(*yaml.Node) bool

func WithFilterOut(filt FilterOut) CreateOption {
	return func(f *FigTree) {
		f.filterOut = filt
	}
}

func defaultFilterOut(f *FigTree) FilterOut {
	configStop := false
	return func(config *yaml.Node) bool {
		// if previous parse found a stop we should abort here
		if configStop {
			return true
		}
		// now check if current doc has a stop, looking for:
		// ```
		// config:
		//   stop: true|false
		// ```
		if pragma := walky.GetKey(config, "config"); pragma != nil {
			if stop := walky.GetKey(pragma, "stop"); stop != nil {
				configStop, _ = strconv.ParseBool(stop.Value)
			}
		}
		// even if current doc has a stop, we should continue to
		// process it, we dont want to process the "next" document
		return false
	}
}

func WithoutExec() CreateOption {
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
}

func NewFigTree(opts ...CreateOption) *FigTree {
	wd, _ := os.Getwd()
	fig := &FigTree{
		home:           os.Getenv("HOME"),
		workDir:        wd,
		envPrefix:      "FIGTREE",
		applyChangeSet: defaultApplyChangeSet,
		exec:           true,
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
	Config   *yaml.Node
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
		if source.Config == nil || source.Config.IsZero() {
			continue
		}
		skip := filterOut(source.Config)
		if skip {
			continue
		}

		m.sourceFile = source.Filename
		err := f.loadConfigSource(m, source.Config, options)
		if err != nil {
			return err
		}
		m.advance()
	}
	return nil
}

func (f *FigTree) LoadConfigSource(config *yaml.Node, source string, options interface{}) error {
	m := NewMerger(WithSourceFile(source))
	return f.loadConfigSource(m, config, options)
}

func sourceLine(file string, node *yaml.Node) string {
	if node.Line > 0 {
		return fmt.Sprintf("%s:%d:%d", file, node.Line, node.Column)
	}
	return file
}

func (f *FigTree) loadConfigSource(m *Merger, config *yaml.Node, options interface{}) error {
	if !reflect.ValueOf(options).IsValid() {
		return fmt.Errorf("options argument [%#v] is not valid", options)
	}

	var err error
	if f.preProcessor != nil {
		err = f.preProcessor(config)
		if err != nil {
			return errors.Wrapf(err, "failed to process config file %s", sourceLine(m.sourceFile, config))
		}
	}

	err = config.Decode(m)
	if err != nil {
		return errors.Wrapf(err, "unable to parse %s", sourceLine(m.sourceFile, config))
	}

	err = m.mergeStructs(
		reflect.ValueOf(options),
		newMergeSource(walky.UnwrapDocument(config)),
		false,
	)
	if err != nil {
		return err
	}
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
	return f.LoadConfigSource(cs.Config, cs.Filename, options)
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
	var node yaml.Node
	if stat, err := os.Stat(file); err == nil {
		if stat.Mode()&0o111 == 0 || !f.exec {
			Log.Debugf("Reading config %s", file)
			fh, err := os.Open(file)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to open %s", rel)
			}
			defer fh.Close()
			decoder := yaml.NewDecoder(fh)
			if err := decoder.Decode(&node); err != nil && !errors.Is(err, io.EOF) {
				return nil, errors.Wrapf(err, "unable to decode %s as yaml", rel)
			}
		} else {
			Log.Debugf("Found Executable Config file: %s", file)
			// it is executable, so run it and try to parse the output
			cmd := exec.Command(file)
			stdout := bytes.NewBufferString("")
			cmd.Stdout = stdout
			cmd.Stderr = bytes.NewBufferString("")
			if err := cmd.Run(); err != nil {
				return nil, errors.Wrapf(err, "%s is executable, but it failed to execute:\n%s", file, cmd.Stderr)
			}
			rel = rel + "[stdout]"
			if err := yaml.Unmarshal(stdout.Bytes(), &node); err != nil {
				return nil, err
			}

		}
		return &ConfigSource{
			Config:   &node,
			Filename: rel,
		}, nil
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
	if homedir != "" && !strings.HasPrefix(cwd, homedir) {
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
func Merge(dst, src interface{}) error {
	m := NewMerger()
	return m.mergeStructs(reflect.ValueOf(dst), newMergeSource(reflect.ValueOf(src)), false)
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

func (m *Merger) mapToStruct(src reflect.Value) (reflect.Value, error) {
	if src.Kind() != reflect.Map {
		return reflect.Value{}, nil
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
			keyval, err := m.mapToStruct(keyval.Elem())
			if err != nil {
				return reflect.Value{}, err
			}
			err = m.mergeStructs(dest.FieldByName(structFieldName), newMergeSource(reflect.ValueOf(keyval.Addr().Interface())), false)
			if err != nil {
				return reflect.Value{}, err
			}
		} else if keyval.Kind() == reflect.Map {
			keyval, err := m.mapToStruct(keyval)
			if err != nil {
				return reflect.Value{}, err
			}
			err = m.mergeStructs(dest.FieldByName(structFieldName), newMergeSource(reflect.ValueOf(keyval.Interface())), false)
			if err != nil {
				return reflect.Value{}, err
			}
		} else {
			dest.FieldByName(structFieldName).Set(reflect.ValueOf(keyval.Interface()))
		}
	}
	return dest, nil
}

func structToMap(src mergeSource) mergeSource {
	if !src.isStruct() {
		return src
	}

	newMap := reflect.ValueOf(map[string]interface{}{})

	reflectedStruct, _ := src.reflect()
	typ := reflectedStruct.Type()

	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		if structField.PkgPath != "" {
			// skip private fields
			continue
		}
		name := yamlFieldName(structField)
		newMap.SetMapIndex(reflect.ValueOf(name), reflectedStruct.Field(i))
	}

	return newMergeSource(newMap)
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
	v1Valid := v1.IsValid()
	v2Valid := v2.IsValid()
	if !v1Valid && !v2Valid {
		return true
	}
	if !v1Valid || !v2Valid {
		return false
	}
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

type fileCoordinate struct {
	Line   int
	Column int
}

type assignOptions struct {
	Overwrite bool
}

type notAssignableError struct {
	dstType reflect.Type
	srcType reflect.Type
}

func (e notAssignableError) Error() string {
	return fmt.Sprintf("%s is not assignable to %s", e.srcType, e.dstType)
}

func (m *Merger) assignValue(dest reflect.Value, src mergeSource, opts assignOptions) error {
	reflectedSrc, coord := src.reflect()
	if !src.isValid() {
		if opts.Overwrite {
			dest.Set(reflectedSrc)
		}
		return nil
	}

	// if not directly assignable, must be convertable, so
	// try to convert now. (ie convert float32 to float64)
	if reflectedSrc.CanConvert(dest.Type()) {
		reflectedSrc = reflectedSrc.Convert(dest.Type())
	}
	if reflectedSrc.Type().AssignableTo(dest.Type()) {
		shouldAssignDest := opts.Overwrite || isZero(dest) || (isDefault(dest) && !isDefault(reflectedSrc))
		isValidSrc := !isZero(reflectedSrc)
		if shouldAssignDest && isValidSrc {
			if reflectedSrc.Kind() == reflect.Map {
				// maps are mutable, so create a brand new shiny one
				dup := reflect.New(reflectedSrc.Type()).Elem()
				if err := m.mergeMaps(dup, src, opts.Overwrite); err != nil {
					return err
				}
				dest.Set(dup)
			} else {
				dest.Set(reflectedSrc)
			}
			return nil
		}
		return nil
	}

	if dest.CanAddr() {
		if option, ok := dest.Addr().Interface().(option); ok {
			destOptionValue := reflect.ValueOf(option.GetValue())
			// map interface type to real-ish type:
			reflectedSrc = reflect.ValueOf(reflectedSrc.Interface())
			if !src.isValid() {
				Log.Debugf("assignValue: src isValid: %t", src.isValid())
				return nil
			}
			source := m.sourceFile
			if coord != nil {
				source += ":" + strconv.Itoa(coord.Line) + ":" + strconv.Itoa(coord.Column)
			}

			if isZero(destOptionValue) {
				if err := option.SetValue(reflectedSrc.Interface()); err != nil {
					return err
				}
				option.SetSource(source)
			}

			if reflectedSrc.CanConvert(destOptionValue.Type()) {
				reflectedSrc = reflectedSrc.Convert(destOptionValue.Type())
			}

			if reflectedSrc.Type().AssignableTo(destOptionValue.Type()) {
				if err := option.SetValue(reflectedSrc.Interface()); err != nil {
					return err
				}
				option.SetSource(source)
				Log.Debugf("assignValue: assigned %#v to %#v", destOptionValue, src)
				return nil
			}
			if destOptionValue.Kind() == reflect.Bool && reflectedSrc.Kind() == reflect.String {
				b, err := strconv.ParseBool(reflectedSrc.Interface().(string))
				if err != nil {
					return errors.Wrapf(err, "%s is not assignable to %s, invalid bool value", reflectedSrc.Type(), destOptionValue.Type())
				}
				if err := option.SetValue(b); err != nil {
					return err
				}
				option.SetSource(source)
				Log.Debugf("assignValue: assigned %#v to %#v", destOptionValue, b)
				return nil
			}
			if destOptionValue.Kind() == reflect.String && reflectedSrc.Kind() != reflect.String {
				if err := option.SetValue(fmt.Sprintf("%v", reflectedSrc.Interface())); err != nil {
					return err
				}
				option.SetSource(source)
				Log.Debugf("assignValue: assigned %#v to %#v", destOptionValue, src)
				return nil
			}
			return errors.WithStack(
				notAssignableError{
					srcType: reflectedSrc.Type(),
					dstType: destOptionValue.Type(),
				},
			)
		}
	}
	// make copy so we can reliably Addr it to see if it fits the
	// Option interface.
	srcCopy := reflect.New(reflectedSrc.Type()).Elem()
	srcCopy.Set(reflectedSrc)
	if option, ok := srcCopy.Addr().Interface().(option); ok {
		srcOptionValue := reflect.ValueOf(option.GetValue())
		if srcOptionValue.Type().AssignableTo(dest.Type()) {
			return m.assignValue(dest, mergeSource{reflected: srcOptionValue, coord: coord}, opts)
		} else {
			return errors.WithStack(
				notAssignableError{
					srcType: srcOptionValue.Type(),
					dstType: dest.Type(),
				},
			)
		}
	}

	if node, ok := dest.Interface().(yaml.Node); ok {
		if src.node != nil {
			node = *src.node
		} else {
			if err := node.Encode(src.reflected.Interface()); err != nil {
				return errors.WithStack(err)
			}
		}
		dest.Set(reflect.ValueOf(node))
		return nil
	}

	return errors.WithStack(
		notAssignableError{
			srcType: reflectedSrc.Type(),
			dstType: dest.Type(),
		},
	)
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

type mergeSource struct {
	reflected reflect.Value
	node      *yaml.Node
	coord     *fileCoordinate
}

func newMergeSource(src any) mergeSource {
	switch cast := src.(type) {
	case reflect.Value:
		if cast.IsValid() {
			cast = reflect.Indirect(cast)
			if cast.Kind() == reflect.Interface {
				cast = reflect.ValueOf(cast.Interface())
			}
		}
		return mergeSource{
			reflected: cast,
		}
	case *yaml.Node:
		return mergeSource{
			node: cast,
		}
	}
	panic(fmt.Sprintf("Unknown type: %T", src))
}

func (ms *mergeSource) reflect() (reflect.Value, *fileCoordinate) {
	if ms.reflected.IsValid() && !ms.reflected.IsZero() {
		return ms.reflected, ms.coord
	}
	if ms.node != nil {
		if ms.node.Line != 0 || ms.node.Column != 0 {
			ms.coord = &fileCoordinate{
				Line:   ms.node.Line,
				Column: ms.node.Column,
			}
		}
		var val any
		ms.node.Decode(&val)
		ms.reflected = reflect.ValueOf(&val).Elem()
		if ms.reflected.Kind() == reflect.Interface {
			ms.reflected = reflect.ValueOf(ms.reflected.Interface())
		}
		// ms.node = nil
	}
	return ms.reflected, ms.coord
}

func (ms *mergeSource) isMap() bool {
	if ms.node != nil {
		return ms.node.Kind == yaml.MappingNode
	}
	return ms.reflected.Kind() == reflect.Map
}

func (ms *mergeSource) isStruct() bool {
	if ms.node != nil {
		return false
	}
	return ms.reflected.Kind() == reflect.Struct
}

func (ms *mergeSource) isZero() bool {
	if ms.node != nil {
		// values directly from config files cannot be 'zero'
		// ie `foo: false` is still non-zero even though the
		// value is the zero value (false)
		return false
	}
	return isZero(ms.reflected)
}

func (ms *mergeSource) isValid() bool {
	if ms.node != nil {
		return !ms.node.IsZero()
	}
	return ms.reflected.IsValid()
}

func (ms *mergeSource) len() int {
	if ms.node != nil {
		return len(ms.node.Content)
	}
	return ms.reflected.Len()
}

func (ms *mergeSource) foreachField(f func(key string, value mergeSource, anonymous bool) error) error {
	if ms.node != nil {
		for i := 0; i < len(ms.node.Content); i += 2 {
			err := f(ms.node.Content[i].Value, newMergeSource(ms.node.Content[i+1]), false)
			if err != nil {
				return err
			}
		}
		return nil
	}
	if ms.reflected.Kind() == reflect.Struct {
		for i := 0; i < ms.reflected.NumField(); i++ {
			structField := ms.reflected.Type().Field(i)
			field := ms.reflected.Field(i)
			field = reflect.Indirect(field)
			if field.Kind() == reflect.Interface {
				field = reflect.ValueOf(field.Interface())
			}
			// if !field.IsValid() {
			// 	continue
			// }
			fieldName := yamlFieldName(structField)
			if structField.PkgPath != "" && !structField.Anonymous {
				continue
			}
			err := f(fieldName, newMergeSource(field), structField.Anonymous)
			if err != nil {
				return err
			}
		}
		return nil
	}
	if ms.reflected.Kind() == reflect.Map {
		for _, key := range ms.reflected.MapKeys() {
			val := ms.reflected.MapIndex(key)
			if val.IsValid() {
				val = reflect.Indirect(val)
				if val.Kind() == reflect.Interface {
					val = reflect.ValueOf(val.Interface())
				}
			}
			err := f(key.String(), newMergeSource(val), false)
			if err != nil {
				return err
			}
		}
		return nil
	}

	return fmt.Errorf("not struct")
}

func (ms *mergeSource) foreachKey(f func(key reflect.Value, value mergeSource) error) error {
	if ms.node != nil {
		for i := 0; i < len(ms.node.Content); i += 2 {
			newMS := newMergeSource(ms.node.Content[i])
			key, _ := newMS.reflect()
			err := f(key, newMergeSource(ms.node.Content[i+1]))
			if err != nil {
				return err
			}
		}
		return nil
	}
	if ms.reflected.Kind() == reflect.Map {
		for _, key := range ms.reflected.MapKeys() {
			val := ms.reflected.MapIndex(key)
			val = reflect.Indirect(val)
			if val.Kind() == reflect.Interface {
				val = reflect.ValueOf(val.Interface())
			}
			err := f(key, newMergeSource(val))
			if err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("not map")
}

func (ms *mergeSource) foreach(f func(ix int, item mergeSource) error) error {
	if ms.node != nil {
		for i := 0; i < len(ms.node.Content); i += 1 {
			if err := f(i, newMergeSource(ms.node.Content[i])); err != nil {
				return err
			}
		}
		return nil
	}
	switch ms.reflected.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < ms.reflected.Len(); i++ {
			item := ms.reflected.Index(i)
			item = reflect.Indirect(item)
			if item.Kind() == reflect.Interface {
				item = reflect.ValueOf(item.Interface())
			}
			if err := f(i, newMergeSource(item)); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("not slice or array")
}

func (m *Merger) mergeStructs(dst reflect.Value, src mergeSource, overwrite bool) error {
	dst = reflect.Indirect(dst)
	dst, restore := fromInterface(dst)
	defer restore()

	if dst.Kind() == reflect.Map {
		return m.mergeMaps(dst, src, overwrite)
	}

	if !dst.IsValid() || !src.isValid() {
		Log.Debugf("Valid: dst:%v src:%t", dst.IsValid(), src.isValid())
		return nil
	}

	dstFieldTypesByYAML := make(map[string]reflect.StructField)
	dstFieldValuesByYAML := make(map[string]reflect.Value)

	var populateYAMLMaps func(reflect.Value)
	populateYAMLMaps = func(ov reflect.Value) {
		for i := 0; i < ov.NumField(); i++ {
			fieldType := ov.Type().Field(i)
			yamlName := yamlFieldName(fieldType)
			if _, ok := dstFieldTypesByYAML[yamlName]; !ok {
				dstFieldTypesByYAML[yamlName] = fieldType
				dstFieldValuesByYAML[yamlName] = ov.Field(i)
			}
		}

		for i := 0; i < ov.NumField(); i++ {
			fieldType := ov.Type().Field(i)
			if fieldType.Anonymous && reflect.Indirect(ov.Field(i)).Type().Kind() == reflect.Struct {
				populateYAMLMaps(reflect.Indirect(ov.Field(i)))
			}
		}
	}
	populateYAMLMaps(dst)

	return src.foreachField(func(fieldName string, srcField mergeSource, anon bool) error {
		dstStructField, ok := dstFieldTypesByYAML[fieldName]
		if !ok {
			if anon {
				// this is an embedded struct, and the destination does not contain
				// the same embeded struct, so try to merge the embedded struct
				// directly with the destination
				return m.mergeStructs(dst, srcField, m.mustOverwrite(fieldName))
			}
			// if original value does not have the same struct field
			// then just skip this field.
			return nil
		}

		// PkgPath is empty for upper case (exported) field names.
		if dstStructField.PkgPath != "" {
			// unexported field, skipping
			return nil
		}

		dstField := dstFieldValuesByYAML[fieldName]
		dstField, restore := fromInterface(dstField)
		defer restore()

		if m.mustIgnore(fieldName) {
			return nil
		}

		val, _ := srcField.reflect()
		var err error
		shouldAssign := (isZero(dstField) && !srcField.isZero() || (isDefault(dstField) && !isDefault(val))) || m.mustOverwrite(fieldName)
		if (shouldAssign) && !isSame(dstField, val) {
			err = m.assignValue(dstField, srcField, assignOptions{
				Overwrite: overwrite || m.mustOverwrite(fieldName),
			})
			// if this is a notAssignableError then we want
			// to continue down to try to investigate more complex
			// types.  For example we  will get here when we try to
			// assign ListStringOption to []string or []interface
			// where we want to iterate below for each StringOption.
			var assignErr notAssignableError
			if err != nil && !errors.As(err, &assignErr) {
				return err
			}
		}
		switch dstField.Kind() {
		case reflect.Map:
			Log.Debugf("Merging Map: %#v with %#v", dstField, srcField)
			return m.mergeStructs(dstField, srcField, overwrite || m.mustOverwrite(fieldName))
		case reflect.Slice:
			if srcField.len() > 0 {
				Log.Debugf("Merging Slice: %#v with %#v", dstField, srcField)
				merged, err := m.mergeArrays(dstField, srcField, overwrite || m.mustOverwrite(fieldName))
				if err != nil {
					return err
				}
				dstField.Set(merged)
				return nil
			}
		case reflect.Array:
			if srcField.len() > 0 {
				Log.Debugf("Merging Array: %v with %v", dstField, srcField)
				merged, err := m.mergeArrays(dstField, srcField, overwrite || m.mustOverwrite(fieldName))
				if err != nil {
					return err
				}
				dstField.Set(merged)
				return nil
			}
		case reflect.Struct:
			// only merge structs if they are not special structs (options or yaml.Node):
			if !isSpecialStruct(dstField) {
				Log.Debugf("Merging Struct: %#v with %#v", dstField, srcField)
				return m.mergeStructs(dstField, srcField, overwrite || m.mustOverwrite(fieldName))
			}
		}
		return err
	})
}

func (m *Merger) mergeMaps(dst reflect.Value, src mergeSource, overwrite bool) error {
	if src.isStruct() {
		src = structToMap(src)
	}
	if !src.isMap() {
		return nil
	}
	if overwrite {
		// truncate all the keys
		for _, key := range dst.MapKeys() {
			// setting to zero value is a "delete" operation
			dst.SetMapIndex(key, reflect.Value{})
		}
	}
	return src.foreachKey(func(key reflect.Value, value mergeSource) error {
		if !dst.MapIndex(key).IsValid() {
			dstElem := reflect.New(dst.Type().Elem()).Elem()
			err := m.assignValue(dstElem, value, assignOptions{
				Overwrite: overwrite,
			})
			if err != nil {
				return err
			}
			if dst.IsNil() {
				if !dst.CanSet() {
					// TODO: Should this be an error?
					return nil
				}
				dst.Set(reflect.MakeMap(dst.Type()))
			}
			Log.Debugf("Setting %v to %#v", key.Interface(), dstElem.Interface())
			dst.SetMapIndex(key, dstElem)
			return nil
		}

		dstVal := reflect.ValueOf(dst.MapIndex(key).Interface())
		switch dstVal.Kind() {
		case reflect.Map:
			Log.Debugf("Merging: %v with %v", dstVal, value)
			return m.mergeStructs(dstVal, value, overwrite || m.mustOverwrite(key.String()))
		case reflect.Slice, reflect.Array:
			Log.Debugf("Merging: %v with %v", dstVal, value)
			merged, err := m.mergeArrays(dstVal, value, overwrite || m.mustOverwrite(key.String()))
			if err != nil {
				return err
			}
			dst.SetMapIndex(key, merged)
		default:
			if isZero(dstVal) {
				reflected, _ := value.reflect()
				if !reflected.IsValid() {
					return nil
				}
				if !dstVal.IsValid() || reflected.Type().AssignableTo(dstVal.Type()) {
					dst.SetMapIndex(key, reflected)
				} else {
					// to check for the Option interface we need the Addr of the value, but
					// we cannot take the Addr of a map value, so we have to first copy
					// it, meh not optimal
					newVal := reflect.New(reflected.Type())
					newVal.Elem().Set(reflected)
					if nOption, ok := newVal.Interface().(option); ok {
						dst.SetMapIndex(key, reflect.ValueOf(nOption.GetValue()))
						return nil
					}
					return fmt.Errorf("map value %T is not assignable to %T", reflected.Interface(), dstVal.Interface())
				}
			}
		}
		return nil
	})
}

func isSpecialStruct(dst reflect.Value) bool {
	if !dst.IsValid() {
		return false
	}
	if dst.CanAddr() {
		if _, ok := dst.Addr().Interface().(option); ok {
			return true
		}
	}
	if _, ok := dst.Interface().(yaml.Node); ok {
		return true
	}
	return false
}

func (m *Merger) mergeArrays(dst reflect.Value, src mergeSource, overwrite bool) (reflect.Value, error) {
	var cp reflect.Value
	switch dst.Type().Kind() {
	case reflect.Slice:
		if overwrite {
			// overwriting so just make a new slice
			cp = reflect.MakeSlice(dst.Type(), 0, 0)
		} else {
			cp = reflect.MakeSlice(dst.Type(), dst.Len(), dst.Len())
			reflect.Copy(cp, dst)
		}
	case reflect.Array:
		// arrays are copied, not passed by reference, so we dont need to copy
		cp = dst
	}
	var zero interface{}
	err := src.foreach(func(ix int, item mergeSource) error {
		reflected, _ := item.reflect()
		if dst.Kind() == reflect.Array {
			if dst.Len() <= ix {
				// truncate arrays, we cannot append
				return nil
			}
			dstElem := dst.Index(ix)
			if isDefault(dstElem) || dstElem.IsZero() || overwrite {
				err := m.assignValue(dstElem, item, assignOptions{
					Overwrite: overwrite,
				})
				if err != nil {
					return err
				}
			}
			return nil
		}

		// if src or dst's are options we need to compare the
		// values to determine if we need to skip inserting this
		// element
		compareValue := reflected
		if reflected.CanAddr() {
			if nOption, ok := reflected.Addr().Interface().(option); ok {
				if !nOption.IsDefined() {
					return nil
				}
				compareValue = reflect.ValueOf(nOption.GetValue())
			}
		}

		if !compareValue.IsValid() || reflect.DeepEqual(compareValue.Interface(), zero) {
			return nil
		}

		for oi := 0; oi < cp.Len(); oi++ {
			o := cp.Index(oi)
			if o.CanAddr() {
				if oOption, ok := o.Addr().Interface().(option); ok {
					o = reflect.ValueOf(oOption.GetValue())
				}
			}
			if reflect.DeepEqual(compareValue.Interface(), o.Interface()) {
				return nil
			}
		}
		dstElem := reflect.New(cp.Type().Elem()).Elem()
		dstKind := dstElem.Kind()
		switch {
		case dstKind == reflect.Map, (dstKind == reflect.Struct && !isSpecialStruct(dstElem)):
			Log.Debugf("Merging: %#v with %#v", dstElem, reflected)
			if err := m.mergeStructs(dstElem, item, overwrite); err != nil {
				return err
			}
		case dstKind == reflect.Slice, dstKind == reflect.Array:
			Log.Debugf("Merging: %v with %v", dstElem, reflected)
			var err error
			dstElem, err = m.mergeArrays(dstElem, item, overwrite)
			if err != nil {
				return err
			}
		default:
			err := m.assignValue(dstElem, item, assignOptions{
				Overwrite: overwrite,
			})
			if err != nil {
				return err
			}
		}

		cp = reflect.Append(cp, dstElem)
		return nil
	})
	if err != nil {
		return reflect.Value{}, err
	}
	return cp, nil
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
