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
		return errors.Errorf("options argument [%#v] is not valid", options)
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
		return errors.WithStack(walky.ErrFilename(err, m.sourceFile))
	}

	_, err = m.mergeStructs(
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
	absFile := file
	if !filepath.IsAbs(file) {
		absFile = filepath.Clean(filepath.Join(f.workDir, file))
	}
	rel, err := filepath.Rel(f.workDir, absFile)
	if err != nil {
		rel = file
	}
	var node yaml.Node
	if stat, err := os.Stat(absFile); err == nil {
		if stat.Mode()&0o111 == 0 || !f.exec {
			Log.Debugf("Reading config %s", absFile)
			fh, err := os.Open(absFile)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to open %s", rel)
			}
			defer fh.Close()
			decoder := yaml.NewDecoder(fh)
			if err := decoder.Decode(&node); err != nil && !errors.Is(err, io.EOF) {
				return nil, errors.WithStack(walky.ErrFilename(err, file))
			}
		} else {
			Log.Debugf("Found Executable Config file: %s", absFile)
			// it is executable, so run it and try to parse the output
			cmd := exec.Command(absFile)
			stdout := bytes.NewBufferString("")
			cmd.Stdout = stdout
			cmd.Stderr = bytes.NewBufferString("")
			if err := cmd.Run(); err != nil {
				return nil, errors.Wrapf(err, "%s is executable, but it failed to execute:\n%s", file, cmd.Stderr)
			}
			rel += "[stdout]"
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
	sourceFile   string
	preserveMap  map[string]struct{}
	preserveTags map[string]struct{}
	Config       ConfigOptions `json:"config,omitempty" yaml:"config,omitempty"`
	ignore       []string
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

func PreserveTags(names ...string) MergeOption {
	return func(m *Merger) {
		for _, tag := range names {
			m.preserveTags[tag] = struct{}{}
		}
	}
}

func NewMerger(options ...MergeOption) *Merger {
	m := &Merger{
		sourceFile:  "merge",
		preserveMap: make(map[string]struct{}),
		preserveTags: map[string]struct{}{
			"json":    {},
			"yaml":    {},
			"figtree": {},
		},
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

// Merge will attempt to merge the data from src into dst. src and dst may each
// be either a map or a struct. Structs do not need to have the same structure,
// but any field name that exists in both structs will must be the same type.
func Merge(dst, src interface{}) error {
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() == reflect.Struct {
		return errors.New("dst argument cannot be a struct (should be *struct)")
	}
	m := NewMerger()
	_, err := m.mergeStructs(dstValue, newMergeSource(reflect.ValueOf(src)), false)
	return err
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

func mergeOptions(a reflect.Type, b reflect.Type) (reflect.Value, bool) {
	if !a.Implements(reflect.TypeOf((*option)(nil)).Elem()) {
		// try to see if pointer type of A implements option
		a := reflect.New(a).Type()
		if !a.Implements(reflect.TypeOf((*option)(nil)).Elem()) {
			return reflect.Value{}, false
		}
	}
	if !b.Implements(reflect.TypeOf((*option)(nil)).Elem()) {
		// try to see if pointer type of B implements option
		b := reflect.New(b).Type()
		if !b.Implements(reflect.TypeOf((*option)(nil)).Elem()) {
			return reflect.Value{}, false
		}
	}

	av, ok := a.FieldByName("Value")
	if !ok {
		return reflect.Value{}, false
	}
	avt := av.Type

	bv, ok := b.FieldByName("Value")
	if !ok {
		return reflect.Value{}, false
	}
	bvt := bv.Type

	if bvt.AssignableTo(avt) {
		return reflect.New(a).Elem(), true
	}
	if avt.AssignableTo(bvt) {
		return reflect.New(b).Elem(), true
	}
	if bvt.ConvertibleTo(avt) {
		return reflect.New(a).Elem(), true
	}
	if avt.ConvertibleTo(bvt) {
		return reflect.New(b).Elem(), true
	}
	return reflect.Value{}, false
}

func (m *Merger) mergeTags(a reflect.StructField, b reflect.StructField) string {
	allTags := []string{}
	for k := range m.preserveTags {
		allTags = append(allTags, k)
	}
	var resultTag []string
	sort.Strings(allTags)
	for _, tag := range allTags {
		aTag, aOk := a.Tag.Lookup(tag)
		bTag, bOk := b.Tag.Lookup(tag)
		switch {
		case aOk: // if a has a tag or both have the tag, use a
			resultTag = append(resultTag, fmt.Sprintf("%s:%q", tag, aTag))
		case bOk:
			resultTag = append(resultTag, fmt.Sprintf("%s:%q", tag, bTag))
		}
	}
	return strings.Join(resultTag, " ")
}

// findField will look for the field in the map.  It will first look for
// the field name as defined by CanonicalFieldName, then it will look for
// a field based on the yaml tag name.  This helps ensure that we dont add
// duplicate fields based on either struct field name or the yaml tag name,
// both of which will cause failures.
func findField(fields map[string]reflect.StructField, field reflect.StructField) (key string, found reflect.StructField, ok bool) {
	fieldName := CanonicalFieldName(field)
	if f, found := fields[fieldName]; found {
		return fieldName, f, true
	}

	yamlName := yamlFieldName(field)
	for k, f := range fields {
		if yamlName == yamlFieldName(f) {
			return k, f, true
		}
	}

	return "", reflect.StructField{}, false
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

				field.Name = CanonicalFieldName(field)
				if fieldKey, f, ok := findField(foundFields, field); ok {
					newTag := m.mergeTags(f, field)
					if newTag != string(f.Tag) {
						f.Tag = reflect.StructTag(newTag)
						foundFields[fieldKey] = f
					}

					// the canonical name is influenced by the `figtree` tag
					// so make sure the merged field has the correct name after
					// merging the tags.
					if fieldName := CanonicalFieldName(f); f.Name != fieldName {
						f.Name = fieldName
						foundFields[fieldKey] = f
					}

					if f.Type.Kind() == reflect.Struct && field.Type.Kind() == reflect.Struct {
						if fName, fieldName := f.Type.Name(), field.Type.Name(); fName == "" || fieldName == "" || fName != fieldName {
							// do we have two options?
							if newval, ok := mergeOptions(f.Type, field.Type); ok {
								// we have two fields with the same name and they are both options, so we need
								// to merge the existing option with the new one in case they are different
								f.Type = newval.Type()
								foundFields[fieldKey] = f
							} else {
								// we have 2 fields with the same name and they are both structs, so we need
								// to merge the existing struct with the new one in case they are different
								newval := m.makeMergeStruct(reflect.New(f.Type).Elem(), reflect.New(field.Type).Elem()).Elem()
								f.Type = newval.Type()
								foundFields[fieldKey] = f
							}
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
				// the yaml and json names are just taken directly from the map key
				if fieldKey, f, ok := findField(foundFields, field); ok {
					newTag := m.mergeTags(f, field)
					if newTag != string(f.Tag) {
						f.Tag = reflect.StructTag(newTag)
						foundFields[fieldKey] = f
					}
					// the canonical name is influenced by the `figtree` tag
					// so make sure the merged field has the correct name after
					// merging the tags.
					if fieldName := CanonicalFieldName(f); f.Name != fieldName {
						f.Name = fieldName
						foundFields[fieldKey] = f
					}
					if f.Type.Kind() == reflect.Struct && t.Kind() == reflect.Struct {
						if fName, tName := f.Type.Name(), t.Name(); fName == "" || tName == "" || fName != tName {
							// do we have two options?
							if newval, ok := mergeOptions(f.Type, field.Type); ok {
								// we have two fields with the same name and they are both options, so we need
								// to merge the existing option with the new one in case they are different
								f.Type = newval.Type()
								foundFields[fieldKey] = f
							} else {
								// we have 2 fields with the same name and they are both structs, so we need
								// to merge the existig struct with the new one in case they are different
								newval := m.makeMergeStruct(reflect.New(f.Type).Elem(), reflect.New(t).Elem()).Elem()
								f.Type = newval.Type()
								foundFields[fieldKey] = f
							}
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
	seen := map[string]reflect.StructField{}
	yamlSeen := map[string]reflect.StructField{}
	for _, value := range foundFields {
		fields = append(fields, value)
		if prev, ok := seen[value.Name]; ok {
			// we have a duplicate field name, this should not happen
			panic(fmt.Sprintf("Duplicate field name %q in merge struct.\n\tOld: %s %s `%s`\n\tNew: %s %s `%s`\n", value.Name, prev.Name, prev.Type.String(), prev.Tag, value.Name, value.Type.String(), value.Tag))
		}
		seen[value.Name] = value
		yamlName := yamlTagName(value.Tag)
		if prev, ok := yamlSeen[yamlName]; yamlName != "" && ok {
			// we have a duplicate field name, this should not happen
			panic(fmt.Sprintf("Duplicate YAML tag %q in merge struct.\n\tOld: %s %s `%s`\n\tNew: %s %s `%s`\n", yamlName, prev.Name, prev.Type.String(), prev.Tag, value.Name, value.Type.String(), value.Tag))
		}
		yamlSeen[yamlName] = value
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
		switch {
		case keyval.Kind() == reflect.Ptr && keyval.Elem().Kind() == reflect.Map:
			keyval, err := m.mapToStruct(keyval.Elem())
			if err != nil {
				return reflect.Value{}, err
			}
			_, err = m.mergeStructs(dest.FieldByName(structFieldName), newMergeSource(reflect.ValueOf(keyval.Addr().Interface())), false)
			if err != nil {
				return reflect.Value{}, err
			}
		case keyval.Kind() == reflect.Map:
			keyval, err := m.mapToStruct(keyval)
			if err != nil {
				return reflect.Value{}, err
			}
			_, err = m.mergeStructs(dest.FieldByName(structFieldName), newMergeSource(reflect.ValueOf(keyval.Interface())), false)
			if err != nil {
				return reflect.Value{}, err
			}
		default:
			dest.FieldByName(structFieldName).Set(reflect.ValueOf(keyval.Interface()))
		}
	}
	return dest, nil
}

func structToMap(src mergeSource) (mergeSource, error) {
	if !src.isStruct() {
		return src, nil
	}

	newMap := reflect.ValueOf(map[string]interface{}{})

	reflectedStruct, _, err := src.reflect()
	if err != nil {
		return mergeSource{}, err
	}
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

	return newMergeSource(newMap), nil
}

type ConfigOptions struct {
	Overwrite []string `json:"overwrite,omitempty" yaml:"overwrite,omitempty"`
}

func yamlTagName(tag reflect.StructTag) string {
	if tag, ok := tag.Lookup("yaml"); ok {
		// with yaml:"foobar,omitempty"
		// we just want to the "foobar" part
		parts := strings.Split(tag, ",")
		if parts[0] != "" && parts[0] != "-" {
			return parts[0]
		}
	}
	return ""
}

func yamlFieldName(sf reflect.StructField) string {
	if name := yamlTagName(sf.Tag); name != "" {
		return name
	}
	// guess the field name from reversing camel case
	// so "FooBar" becomes "foo-bar"
	parts := camelcase.Split(sf.Name)
	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, "-")
}

// CanonicalFieldName will return the the field name that will be used with
// merging maps and structs where the name casing/formatting may not
// be consistent.  If the field uses tag `figtree:",name=MyName"` then
// that name will be used instead of the default contention.
func CanonicalFieldName(sf reflect.StructField) string {
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

func isZeroOrDefaultOption(v reflect.Value) bool {
	if option := toOption(v); option != nil {
		// an option can only be `zero` if it is undefined
		if !option.IsDefined() {
			return true
		}
		return option.IsDefault()
	}
	return false
}

func toOption(v reflect.Value) option {
	v = indirect(v)
	if !v.IsValid() {
		return nil
	}
	if !v.CanAddr() {
		tmp := reflect.New(v.Type()).Elem()
		tmp.Set(v)
		v = tmp
	}
	if option, ok := v.Addr().Interface().(option); ok {
		return option
	}
	return nil
}

func indirect(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	return v
}

func uninterface(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Interface {
		return v.Elem()
	}
	return v
}

func isZero(v reflect.Value) bool {
	v = indirect(v)
	if !v.IsValid() {
		return true
	}
	if option := toOption(v); option != nil {
		return !option.IsDefined()
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

type notAssignableError struct {
	dstType        reflect.Type
	srcType        reflect.Type
	sourceLocation SourceLocation
}

func (e notAssignableError) Error() string {
	return fmt.Sprintf("%s: %s is not assignable to %s", e.sourceLocation, e.srcType, e.dstType)
}

var stringType = reflect.ValueOf("").Type()

type assignOptions struct {
	// Overwrite will attempt to replace the destination with the source
	// even if the dest is already a valid (non-zero, non-default) value.
	Overwrite bool

	// srcIsDefault is used internally when we are provided src as an Option
	// and we unwrap the option and recursively assign the raw value to dest.
	srcIsDefault bool
	// destIsDefault is used internally when we are provided dest as an Option
	// and we unwrap the option and recursively try to assign src to the raw
	// value.
	destIsDefault bool
	// sourceLocation is used internally to track the source file
	// name/line/column when that data is available. This is set
	// when we recursively call assignValue after unwrapping src Option.
	sourceLocation SourceLocation
}

// assignValue will attempt to assign src to dest.  If there is no errors, the
// bool return value will indicate if the assignment happened, which will be
// false when the trying to assign to a non-zero, or non-default value (without
// Overwrite set)
func (m *Merger) assignValue(dest reflect.Value, src mergeSource, opts assignOptions) (bool, error) {
	reflectedSrc, coord, err := src.reflect()
	if err != nil {
		return false, walky.ErrFilename(err, m.sourceFile)
	}
	Log.Debugf("assignValue: %#v to %#v [opts: %#v]\n", reflectedSrc, dest, opts)
	if !dest.IsValid() || !reflectedSrc.IsValid() {
		return false, nil
	}

	// Not much we can do here if dest is unsettable, this will happen if
	// dest comes from a map without copying first.  This is a programmer error.
	if !dest.CanSet() {
		return false, errors.Errorf("Cannot assign %#v to unsettable value %#v", reflectedSrc, dest)
	}

	// if we have a pointer value, deref (and create if nil)
	if dest.Kind() == reflect.Pointer {
		if dest.IsNil() {
			dest.Set(reflect.New(dest.Type().Elem()))
		}
		dest = dest.Elem()
	}

	// if src is a pointer, deref, return if nil and not overwriting
	if reflectedSrc.Kind() == reflect.Pointer {
		reflectedSrc = reflectedSrc.Elem()
		// reflectedSrc might be invalid if it was Nil so lets handle that now
		if !reflectedSrc.IsValid() {
			if opts.Overwrite {
				dest.Set(reflectedSrc)
				return true, nil
			}
			return false, nil
		}
	}

	// check to see if we can convert src to dest type before we check to see
	// if is assignable. We cannot assign float32 to float64, but we can
	// convert float32 to float64 and then assign.  Note we skip conversion
	// to strings since almost anything can be converted to a string
	if dest.Kind() != reflect.String && reflectedSrc.CanConvert(dest.Type()) {
		reflectedSrc = reflectedSrc.Convert(dest.Type())
	}

	// if the source is an option, get the raw value of the option
	// and try to assign that to the dest. assignValue does not require
	// the source to be addressable, but in order to check for the option
	// interface we might have to make the source addressable via a copy.
	addressableSrc := reflectedSrc
	if !addressableSrc.CanAddr() {
		addressableSrc = reflect.New(reflectedSrc.Type()).Elem()
		addressableSrc.Set(reflectedSrc)
	}
	if option := toOption(addressableSrc); option != nil {
		srcOptionValue := reflect.ValueOf(option.GetValue())
		opts.sourceLocation = option.GetSource()
		opts.srcIsDefault = option.IsDefault()
		return m.assignValue(dest, newMergeSource(srcOptionValue), opts)
	}

	// if dest is an option type, then try to assign directly to the
	// raw option value and then populate the option object
	if dest.CanAddr() {
		if option := toOption(dest); option != nil {
			destOptionValue := reflect.ValueOf(option.GetValue())
			if !destOptionValue.IsValid() {
				// this will happen when we have an Option[any], and
				// GetValue returns nil as the default value
				if _, ok := dest.Interface().(Option[any]); ok {
					// since we want an `any` we should be good with
					// just creating the src type
					destOptionValue = reflect.New(reflectedSrc.Type()).Elem()
				}
			}
			if !destOptionValue.CanSet() {
				destOptionValue = reflect.New(destOptionValue.Type()).Elem()
			}
			opts.destIsDefault = option.IsDefault()
			ok, err := m.assignValue(destOptionValue, src, opts)
			if err != nil {
				return false, err
			}
			if ok {
				if err := option.SetValue(destOptionValue.Interface()); err != nil {
					return false, err
				}
				source := opts.sourceLocation
				if source.Name == "" {
					source.Name = m.sourceFile
				}
				if coord != nil {
					source.Location = coord
				}
				option.SetSource(source)
			}
			return ok, nil
		}
	}

	// if we are assigning to a yaml.Node then try to preserve the raw
	// yaml.Node input, otherwise encode the src into the Node.
	if node, ok := dest.Interface().(yaml.Node); ok {
		if src.node != nil {
			dest.Set(reflect.ValueOf(*src.node))
			return true, nil
		}
		if err := node.Encode(reflectedSrc.Interface()); err != nil {
			return false, errors.WithStack(err)
		}
		dest.Set(reflect.ValueOf(node))
		return true, nil
	}

	if reflectedSrc.Type().AssignableTo(dest.Type()) {
		shouldAssignDest := opts.Overwrite || isZero(dest) || (opts.destIsDefault && !opts.srcIsDefault)
		if shouldAssignDest {
			switch reflectedSrc.Kind() {
			case reflect.Map:
				// maps are mutable, so create a brand new shiny one
				dup := reflect.New(reflectedSrc.Type()).Elem()
				ok, err := m.mergeMaps(dup, src, opts.Overwrite)
				if err != nil {
					return false, err
				}
				if ok {
					dest.Set(dup)
				}
				return ok, nil
			case reflect.Slice:
				if reflectedSrc.IsNil() {
					dest.Set(reflectedSrc)
				} else {
					// slices are mutable, so create a brand new shiny one
					cp := reflect.MakeSlice(reflectedSrc.Type(), reflectedSrc.Len(), reflectedSrc.Len())
					reflect.Copy(cp, reflectedSrc)
					dest.Set(cp)
				}
			default:
				dest.Set(reflectedSrc)
			}
			return true, nil
		}
		return false, nil
	}

	if dest.Kind() == reflect.Bool && reflectedSrc.Kind() == reflect.String {
		b, err := strconv.ParseBool(reflectedSrc.Interface().(string))
		if err != nil {
			return false, errors.Wrapf(err, "%s is not assignable to %s, invalid bool value %#v", reflectedSrc.Type(), dest.Type(), reflectedSrc)
		}
		dest.Set(reflect.ValueOf(b))
		return true, nil
	}

	if dest.Kind() == reflect.String && reflectedSrc.Kind() != reflect.String && stringType.AssignableTo(dest.Type()) {
		switch reflectedSrc.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map:
			return false, errors.WithStack(
				notAssignableError{
					srcType:        reflectedSrc.Type(),
					dstType:        dest.Type(),
					sourceLocation: NewSource(m.sourceFile, WithLocation(coord)),
				},
			)
		case reflect.Struct:
			// we generally dont want to assign structs to a string
			// unless that struct is an option struct in which case
			// we use convert the value
			if option := toOption(reflectedSrc); option != nil {
				dest.Set(reflect.ValueOf(fmt.Sprint(option.GetValue())))
			}
			return false, errors.WithStack(
				notAssignableError{
					srcType:        reflectedSrc.Type(),
					dstType:        dest.Type(),
					sourceLocation: NewSource(m.sourceFile, WithLocation(coord)),
				},
			)
		default:
			// if we have a scalar node we want to convert to a string, just use
			// the literal value tokenized from the document, this will
			// allow values like `False` to be preserved as a case-sensitive
			// string rather than being converted to a bool, the back to a string.
			if src.node != nil && src.node.Kind == yaml.ScalarNode {
				dest.Set(reflect.ValueOf(src.node.Value))
			} else {
				dest.Set(reflect.ValueOf(fmt.Sprint(reflectedSrc.Interface())))
			}
		}
		return true, nil
	}

	if !isSpecial(dest) && dest.CanAddr() {
		meth := dest.Addr().MethodByName("UnmarshalYAML")
		if meth.IsValid() {
			if src.node != nil {
				if err := src.node.Decode(dest.Addr().Interface()); err != nil {
					return false, errors.WithStack(walky.ErrFilename(walky.NewYAMLError(err, src.node), m.sourceFile))
				}
			} else {
				// we know we have an UnmarshalYAML function, so use yaml
				// to convert to/from between random types since we can't
				// do it with reflection alone here.
				content, err := yaml.Marshal(reflectedSrc.Interface())
				if err != nil {
					return false, errors.WithStack(err)
				}
				if err := yaml.Unmarshal(content, dest.Addr().Interface()); err != nil {
					return false, errors.WithStack(err)
				}
			}
			return true, nil
		}
	}

	// if we have a collection don't proceed to attempt to unmarshal direct
	// from the yaml.Node ... collections are process per item, rather than
	// as a whole.
	if isCollection(dest) {
		return false, errors.WithStack(
			notAssignableError{
				srcType:        reflectedSrc.Type(),
				dstType:        dest.Type(),
				sourceLocation: NewSource(m.sourceFile, WithLocation(coord)),
			},
		)
	}

	// if we are still here, try one last-ditch effort to see if we can convert,
	// this time allowing things to convert to `string` types is okay since we
	// have exhausted all other assignment options above.
	if reflectedSrc.CanConvert(dest.Type()) {
		shouldAssignDest := opts.Overwrite || isZero(dest) || (opts.destIsDefault && !opts.srcIsDefault)
		if shouldAssignDest {
			reflectedSrc = reflectedSrc.Convert(dest.Type())
			dest.Set(reflectedSrc)
			return true, nil
		}
		return false, nil
	}

	return false, errors.WithStack(
		notAssignableError{
			srcType:        reflectedSrc.Type(),
			dstType:        dest.Type(),
			sourceLocation: NewSource(m.sourceFile, WithLocation(coord)),
		},
	)
}

type mergeSource struct {
	reflected reflect.Value
	node      *yaml.Node
	coord     *FileCoordinate
}

func newMergeSource(src any) mergeSource {
	switch cast := src.(type) {
	case reflect.Value:
		cast = uninterface(indirect(cast))
		if cast.IsValid() && cast.CanInterface() {
			// handle the edge case that someone has passed in
			// reflect.ValueOf(*yaml.Node) rather than *yaml.Node directly
			if node, ok := cast.Interface().(yaml.Node); ok {
				return mergeSource{
					node: walky.Indirect(&node),
				}
			}
		}
		return mergeSource{
			reflected: cast,
		}
	case *yaml.Node:
		return mergeSource{
			node: walky.Indirect(cast),
		}
	}
	panic(fmt.Sprintf("Unknown type: %T", src))
}

func (ms *mergeSource) reflect() (reflect.Value, *FileCoordinate, error) {
	if ms.reflected.IsValid() && !ms.reflected.IsZero() {
		return ms.reflected, ms.coord, nil
	}
	if ms.node != nil {
		if ms.node.Line != 0 || ms.node.Column != 0 {
			ms.coord = &FileCoordinate{
				Line:   ms.node.Line,
				Column: ms.node.Column,
			}
		}
		var val any
		err := ms.node.Decode(&val)
		if err != nil {
			return reflect.Value{}, nil, errors.WithStack(walky.NewYAMLError(err, ms.node))
		}
		ms.reflected = uninterface(reflect.ValueOf(&val).Elem())
	}
	return ms.reflected, ms.coord, nil
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

func (ms *mergeSource) isList() bool {
	if ms.node != nil {
		return ms.node.Kind == yaml.SequenceNode
	}
	switch ms.reflected.Kind() {
	case reflect.Array, reflect.Slice:
		return true
	}
	return false
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
		if ms.node.Kind == yaml.MappingNode || ms.node.Kind == yaml.SequenceNode {
			return len(ms.node.Content)
		}
		return 0
	}
	return ms.reflected.Len()
}

func (ms *mergeSource) foreachField(f func(key string, value mergeSource, anonymous bool) error) error {
	if ms.node != nil {
		return walky.RangeMap(ms.node, func(keyNode, valueNode *yaml.Node) error {
			return f(keyNode.Value, newMergeSource(valueNode), false)
		})
	}
	switch ms.reflected.Kind() {
	case reflect.Struct:
		for i := 0; i < ms.reflected.NumField(); i++ {
			structField := ms.reflected.Type().Field(i)
			field := ms.reflected.Field(i)
			field = uninterface(indirect(field))
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
	case reflect.Map:
		for _, key := range ms.reflected.MapKeys() {
			val := ms.reflected.MapIndex(key)
			if val.IsValid() {
				val = uninterface(indirect(val))
			}
			err := f(key.String(), newMergeSource(val), false)
			if err != nil {
				return err
			}
		}
		return nil
	}

	return errors.Errorf("expected struct, got %s", ms.reflected.Kind())
}

func (ms *mergeSource) foreachKey(f func(key reflect.Value, value mergeSource) error) error {
	if ms.node != nil {
		return walky.RangeMap(ms.node, func(keyNode, valueNode *yaml.Node) error {
			newMS := newMergeSource(keyNode)
			key, _, err := newMS.reflect()
			if err != nil {
				return err
			}
			return f(key, newMergeSource(valueNode))
		})
	}
	if ms.reflected.Kind() == reflect.Map {
		for _, key := range ms.reflected.MapKeys() {
			val := ms.reflected.MapIndex(key)
			val = uninterface(indirect(val))
			err := f(key, newMergeSource(val))
			if err != nil {
				return err
			}
		}
		return nil
	}
	return errors.Errorf("not map")
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
			item = uninterface(indirect(item))
			if err := f(i, newMergeSource(item)); err != nil {
				return err
			}
		}
		return nil
	}
	return errors.Errorf("not slice or array")
}

type fieldYAML struct {
	StructField reflect.StructField
	Value       reflect.Value
}

// populateYAMLMaps will collect a map by field name where
// those field names are converted to a common name used in YAML
// documents so we can easily merge fields and maps together from
// multiple sources.
func populateYAMLMaps(v reflect.Value) map[string]fieldYAML {
	fieldsByYAML := make(map[string]fieldYAML)
	if v.Kind() != reflect.Struct {
		return fieldsByYAML
	}
	for i := 0; i < v.NumField(); i++ {
		fieldType := v.Type().Field(i)
		yamlName := yamlFieldName(fieldType)
		if _, ok := fieldsByYAML[yamlName]; !ok {
			fieldsByYAML[yamlName] = fieldYAML{
				StructField: fieldType,
				Value:       v.Field(i),
			}
		}
	}

	for i := 0; i < v.NumField(); i++ {
		fieldType := v.Type().Field(i)
		if !fieldType.Anonymous {
			continue
		}

		fv := v.Field(i)
		if fv.Kind() == reflect.Pointer && fv.IsNil() {
			// skip nil pointers for embedded structs
			continue
		}
		if indirect(fv).Type().Kind() == reflect.Struct {
			anonFields := populateYAMLMaps(indirect(fv))
			for k, v := range anonFields {
				if _, ok := fieldsByYAML[k]; !ok {
					fieldsByYAML[k] = v
				}
			}
		}
	}
	return fieldsByYAML
}

func (m *Merger) mergeStructs(dst reflect.Value, src mergeSource, overwrite bool) (changed bool, err error) {
	dst = indirect(dst)

	if dst.Kind() == reflect.Interface {
		realDst := dst.Elem()
		if realDst.IsValid() {
			newDst := reflect.New(realDst.Type()).Elem()
			newDst.Set(realDst)
			defer func(orig reflect.Value) {
				if changed {
					orig.Set(newDst)
				}
			}(dst)
			dst = newDst
		}
	}

	if dst.Kind() == reflect.Map {
		return m.mergeMaps(dst, src, overwrite)
	}

	if !dst.IsValid() || !src.isValid() {
		Log.Debugf("Valid: dst:%v src:%t", dst.IsValid(), src.isValid())
		return false, nil
	}

	// We first collect maps of struct fields by the yamlized name
	// so we can easily compare maps and structs by common names
	dstFieldsByYAML := populateYAMLMaps(dst)

	err = src.foreachField(func(fieldName string, srcField mergeSource, anon bool) error {
		if m.mustIgnore(fieldName) {
			return nil
		}

		dstFieldByYAML, ok := dstFieldsByYAML[fieldName]
		if !ok {
			if anon {
				// this is an embedded struct, and the destination does not contain
				// the same embedded struct, so try to merge the embedded struct
				// directly with the destination
				ok, err := m.mergeStructs(dst, srcField, m.mustOverwrite(fieldName))
				if err != nil {
					return errors.WithStack(err)
				}
				changed = changed || ok
			}
			// if original value does not have the same struct field
			// then just skip this field.
			return nil
		}

		// PkgPath is empty for upper case (exported) field names.
		if dstFieldByYAML.StructField.PkgPath != "" {
			// unexported field, skipping
			return nil
		}

		dstField := dstFieldByYAML.Value

		fieldChanged := false
		if dstField.Kind() == reflect.Interface {
			realDstField := dstField.Elem()
			if realDstField.IsValid() {
				newDstField := reflect.New(realDstField.Type()).Elem()
				newDstField.Set(realDstField)
				defer func(orig reflect.Value) {
					if fieldChanged {
						orig.Set(newDstField)
					}
				}(dstField)
				dstField = newDstField
			}
		}

		// if we have a pointer value, deref (and create if nil)
		if dstField.Kind() == reflect.Pointer {
			if dstField.IsNil() {
				newField := reflect.New(dstField.Type().Elem())
				defer func(origField reflect.Value) {
					if fieldChanged {
						origField.Set(newField)
					}
				}(dstField)
				dstField = newField
			}
			dstField = dstField.Elem()
		}

		val, _, err := srcField.reflect()
		if err != nil {
			return walky.ErrFilename(err, m.sourceFile)
		}

		shouldAssign := (isZero(dstField) && !srcField.isZero() || (isZeroOrDefaultOption(dstField) && !isZeroOrDefaultOption(val))) || (overwrite || m.mustOverwrite(fieldName))

		var assignErr error
		if shouldAssign && !isSame(dstField, val) {
			fieldChanged, assignErr = m.assignValue(dstField, srcField, assignOptions{
				Overwrite: overwrite || m.mustOverwrite(fieldName),
			})
			// if this is a notAssignableError then we want
			// to continue down to try to investigate more complex
			// types.  For example we  will get here when we try to
			// assign ListStringOption to []string or []interface
			// where we want to iterate below for each StringOption.
			var naErr notAssignableError
			if assignErr != nil && !errors.As(assignErr, &naErr) {
				return assignErr
			}
			changed = changed || fieldChanged
			if fieldChanged {
				return nil
			}
		}
		switch dstField.Kind() {
		case reflect.Map:
			Log.Debugf("Merging Map: %#v to %#v [overwrite: %t]", val, dstField, overwrite || m.mustOverwrite(fieldName))
			ok, err := m.mergeStructs(dstField, srcField, overwrite || m.mustOverwrite(fieldName))
			if err != nil {
				return errors.WithStack(err)
			}
			fieldChanged = fieldChanged || ok
			changed = changed || ok
			return nil
		case reflect.Slice, reflect.Array:
			Log.Debugf("Merging %#v to %#v [overwrite: %t]", val, dstField, overwrite || m.mustOverwrite(fieldName))
			merged, ok, err := m.mergeArrays(dstField, srcField, overwrite || m.mustOverwrite(fieldName))
			if err != nil {
				return err
			}
			if ok {
				dstField.Set(merged)
			}
			fieldChanged = fieldChanged || ok
			changed = changed || ok
			return nil
		case reflect.Struct:
			// only merge structs if they are not special structs (options or yaml.Node):
			if !isSpecial(dstField) {
				Log.Debugf("Merging Struct: %#v to %#v [overwrite: %t]", val, dstField, overwrite || m.mustOverwrite(fieldName))
				ok, err := m.mergeStructs(dstField, srcField, overwrite || m.mustOverwrite(fieldName))
				if err != nil {
					return errors.WithStack(err)
				}
				fieldChanged = fieldChanged || ok
				changed = changed || ok
				return nil
			}
		}
		return assignErr
	})
	if err != nil {
		return changed, walky.ErrFilename(err, m.sourceFile)
	}
	return changed, nil
}

func (m *Merger) mergeMaps(dst reflect.Value, src mergeSource, overwrite bool) (bool, error) {
	if src.isStruct() {
		var err error
		src, err = structToMap(src)
		if err != nil {
			return false, err
		}
	}
	if !src.isMap() {
		return false, nil
	}
	if overwrite {
		// truncate all the keys
		for _, key := range dst.MapKeys() {
			// setting to zero value is a "delete" operation
			dst.SetMapIndex(key, reflect.Value{})
		}
	}

	changed := false
	err := src.foreachKey(func(key reflect.Value, value mergeSource) error {
		if !dst.MapIndex(key).IsValid() {
			dstElem := reflect.New(dst.Type().Elem()).Elem()
			ok, err := m.assignValue(dstElem, value, assignOptions{
				Overwrite: overwrite,
			})
			if option := toOption(dstElem); option != nil {
				loc := option.GetSource()
				if loc.Location == nil {
					_, coord, err := value.reflect()
					if err != nil {
						return walky.ErrFilename(err, m.sourceFile)
					}
					loc.Location = coord
				}
				if loc.Name == "" {
					loc.Name = m.sourceFile
				}
				option.SetSource(loc)
			}
			var assignErr notAssignableError
			if err != nil && !errors.As(err, &assignErr) {
				return err
			} else if err == nil {
				if dst.IsNil() {
					if !dst.CanSet() {
						// TODO: Should this be an error?
						return nil
					}
					dst.Set(reflect.MakeMap(dst.Type()))
				}
				Log.Debugf("Setting %v to %#v", key.Interface(), dstElem.Interface())
				dst.SetMapIndex(key, dstElem)
				changed = changed || ok
				return nil
			}
		}

		if dst.IsNil() {
			// nil map here, we need to create one
			newMap := reflect.MakeMap(dst.Type())
			dst.Set(newMap)
		}
		if !dst.MapIndex(key).IsValid() {
			newVal := reflect.New(dst.Type().Elem()).Elem()
			dst.SetMapIndex(key, newVal)
			changed = true
		}
		dstVal := reflect.ValueOf(dst.MapIndex(key).Interface())
		dstValKind := dstVal.Kind()
		switch {
		case dstValKind == reflect.Map:
			Log.Debugf("Merging: %#v to %#v", value, dstVal)
			ok, err := m.mergeStructs(dstVal, value, overwrite || m.mustOverwrite(key.String()))
			if err != nil {
				return errors.WithStack(err)
			}
			changed = changed || ok
			return nil
		case dstValKind == reflect.Struct && !isSpecial(dstVal):
			Log.Debugf("Merging: %#v to %#v", value, dstVal)
			if !dstVal.CanAddr() {
				// we can't address dstVal so we need to make a new value
				// outside the map, merge into the new value, then
				// set the map key to the new value
				newVal := reflect.New(dstVal.Type()).Elem()
				newVal.Set(dstVal)
				ok, err := m.mergeStructs(newVal, value, overwrite || m.mustOverwrite(key.String()))
				if err != nil {
					return errors.WithStack(err)
				}
				if ok {
					dst.SetMapIndex(key, newVal)
					changed = true
				}
				return nil
			}
			ok, err := m.mergeStructs(dstVal, value, overwrite || m.mustOverwrite(key.String()))
			if err != nil {
				return errors.WithStack(err)
			}
			changed = changed || ok
			return nil
		case dstValKind == reflect.Slice, dstValKind == reflect.Array:
			Log.Debugf("Merging: %#v to %#v", value, dstVal)
			merged, ok, err := m.mergeArrays(dstVal, value, overwrite || m.mustOverwrite(key.String()))
			if err != nil {
				return err
			}
			if ok {
				dst.SetMapIndex(key, merged)
			}
			changed = changed || ok
		default:
			if !isZero(dstVal) {
				return nil
			}
			reflected, _, err := value.reflect()
			if err != nil {
				return walky.ErrFilename(err, m.sourceFile)
			}
			if !reflected.IsValid() {
				return nil
			}
			if !dstVal.IsValid() || reflected.Type().AssignableTo(dstVal.Type()) {
				dst.SetMapIndex(key, reflected)
			} else {
				if srcOption := toOption(reflected); srcOption != nil {
					dst.SetMapIndex(key, reflect.ValueOf(srcOption.GetValue()))
					return nil
				}
				settableDstVal := reflect.New(dstVal.Type()).Elem()
				settableDstVal.Set(dstVal)
				ok, err := m.assignValue(settableDstVal, value, assignOptions{
					Overwrite: overwrite || m.mustOverwrite(key.String()),
				})
				if err != nil {
					return errors.WithStack(err)
				}
				if ok {
					dst.SetMapIndex(key, settableDstVal)
					changed = true
					return nil
				}
				return errors.Errorf("map value %T is not assignable to %T", reflected.Interface(), dstVal.Interface())
			}
		}
		return nil
	})
	if err != nil {
		return changed, walky.ErrFilename(err, m.sourceFile)
	}
	return changed, nil
}

func isCollection(dst reflect.Value) bool {
	if !dst.IsValid() {
		return false
	}
	switch dst.Kind() {
	case reflect.Map, reflect.Slice, reflect.Array:
		return true
	case reflect.Struct:
		return !isSpecial(dst)
	}
	return false
}

// isSpecial returns true if the value is an Option, slice of Options
// map of Options or a yaml.Node.
func isSpecial(dst reflect.Value) bool {
	if !dst.IsValid() {
		return false
	}
	if option := toOption(dst); option != nil {
		return true
	}
	if _, ok := dst.Interface().(yaml.Node); ok {
		return true
	}
	return false
}

func (m *Merger) mergeArrays(dst reflect.Value, src mergeSource, overwrite bool) (reflect.Value, bool, error) {
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

	if !src.isList() {
		reflectedSrc, coord, err := src.reflect()
		if err != nil {
			return reflect.Value{}, false, walky.ErrFilename(err, m.sourceFile)
		}
		if !reflectedSrc.IsValid() {
			// if this is a nil interface data then
			// we don't care that we cant assign it to a
			// list, it is a no-op anyway.
			return cp, false, nil
		}
		return reflect.Value{}, false, errors.WithStack(
			notAssignableError{
				srcType:        reflectedSrc.Type(),
				dstType:        dst.Type(),
				sourceLocation: NewSource(m.sourceFile, WithLocation(coord)),
			},
		)
	}

	// when dst is an empty array we dont want to dedup those elements, they
	// should all be directly assigned.  We only want to dedup when merging
	// in arrays from alternate sources, not the original source.
	skipDedup := false
	if cp.Len() == 0 {
		skipDedup = true
	}

	var zero interface{}
	changed := overwrite
	err := src.foreach(func(ix int, item mergeSource) error {
		reflected, _, err := item.reflect()
		if err != nil {
			return walky.ErrFilename(err, m.sourceFile)
		}
		if dst.Kind() == reflect.Array {
			if dst.Len() <= ix {
				// truncate arrays, we cannot append
				return nil
			}
			dstElem := dst.Index(ix)
			if isZeroOrDefaultOption(dstElem) || dstElem.IsZero() || overwrite {
				ok, err := m.assignValue(dstElem, item, assignOptions{
					Overwrite: overwrite,
				})
				if err != nil {
					return err
				}
				changed = changed || ok
			}
			return nil
		}

		// if src or dst's are options we need to compare the
		// values to determine if we need to skip inserting this
		// element
		compareValue := reflected
		if nOption := toOption(reflected); nOption != nil {
			if !nOption.IsDefined() {
				return nil
			}
			compareValue = reflect.ValueOf(nOption.GetValue())
		}

		if !compareValue.IsValid() || reflect.DeepEqual(compareValue.Interface(), zero) {
			return nil
		}

		if !skipDedup {
			for i := 0; i < cp.Len(); i++ {
				destElem := cp.Index(i)
				if destOption := toOption(destElem); destOption != nil {
					destElem = reflect.ValueOf(destOption.GetValue())
				}

				// try to assign the input to a tmp value so all the normal
				// conversions happen before we compare it to existing elements.
				// Otherwise we might end up with extra dups in the array
				// that are the same value
				if destElem.CanInterface() {
					// ensure the destElem is not a nil value for a pointer type
					// .. in which case this will return a zero Value, which
					// cannot have `Type` called on it.
					if !reflect.ValueOf(destElem.Interface()).IsValid() {
						continue
					}
					tmpVal := reflect.New(reflect.ValueOf(destElem.Interface()).Type()).Elem()
					_, err := m.assignValue(tmpVal, item, assignOptions{})
					if err == nil {
						if reflect.DeepEqual(destElem.Interface(), tmpVal.Interface()) {
							return nil
						}
					}
				}
			}
		}

		dstElem := reflect.New(cp.Type().Elem()).Elem()
		dstKind := dstElem.Kind()
		switch {
		case dstKind == reflect.Map, (dstKind == reflect.Struct && !isSpecial(dstElem)):
			Log.Debugf("Merging: %#v to %#v", reflected, dstElem)
			ok, err := m.mergeStructs(dstElem, item, overwrite)
			if err != nil {
				return errors.WithStack(err)
			}
			changed = changed || ok
		case dstKind == reflect.Slice, dstKind == reflect.Array:
			Log.Debugf("Merging: %#v to %#v", reflected, dstElem)
			merged, ok, err := m.mergeArrays(dstElem, item, overwrite)
			if err != nil {
				return err
			}
			if ok {
				dstElem.Set(merged)
			}
			changed = changed || ok
		default:
			ok, err := m.assignValue(dstElem, item, assignOptions{
				Overwrite: overwrite,
			})
			if err != nil {
				return err
			}
			changed = changed || ok
		}

		cp = reflect.Append(cp, dstElem)
		return nil
	})
	if err != nil {
		return reflect.Value{}, false, err
	}
	return cp, changed, nil
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
