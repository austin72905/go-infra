package module

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
)

const defaultPropertyEnv = "default"

/*
各服務可以在 main.go import 不同環境的 properties package，例如：

//go:embed app.properties
var Files embed.FS

這樣就可以透過對應環境 package 的 Files 讀取 app.properties。
*/
type PropertyRuntime struct {
	Store     *PropertyStore
	Validator *PropertyUsageValidator // 檢查store裡面的 app.properties設置的值都有使用到
}

func NewPropertyRuntime() *PropertyRuntime {
	return &PropertyRuntime{
		Store:     NewPropertyStore(),
		Validator: NewPropertyUsageValidator(),
	}
}

func (p *PropertyRuntime) Property(key string) string {
	p.Validator.Add(key)
	return p.Store.Get(key)
}

func (p *PropertyRuntime) RequiredProperty(key string) string {
	value := p.Property(key)
	if value == "" {
		panic("required property not found: " + key)
	}
	return value
}

func (p *PropertyRuntime) Validate() error {
	return p.Validator.ValidateUnused(p.Store.Keys())
}

func (p *PropertyRuntime) LoadProperties(envFS map[string]embed.FS, env string, fileName string) error {
	files, ok := envFS[env]
	if !ok {
		return fmt.Errorf("invalid environment %q", env)
	}

	defaultFS, ok := envFS[defaultPropertyEnv]
	if !ok {
		return fmt.Errorf("default property fs %q not found", defaultPropertyEnv)
	}

	return p.LoadPropertiesByFS(files, fileName, defaultFS)
}

func (p *PropertyRuntime) LoadPropertiesByFS(properties embed.FS, fileName string, defaultFS embed.FS) error {
	file, err := properties.Open(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			file, err = defaultFS.Open(fileName)
		}
		if err != nil {
			return fmt.Errorf("property file %q not found: %w", fileName, err)
		}
	}
	defer file.Close()

	return p.Store.Load(file)
}

type PropertyStore struct {
	values map[string]string
}

func NewPropertyStore() *PropertyStore {
	return &PropertyStore{
		values: make(map[string]string),
	}
}

func (s *PropertyStore) Set(key, value string) {
	s.values[key] = value
}

func (s *PropertyStore) Get(key string) string {
	return s.values[key]
}

func (s *PropertyStore) Keys() []string {
	keys := make([]string, 0, len(s.values))
	for key := range s.values {
		keys = append(keys, key)
	}
	return keys
}

func (s *PropertyStore) Load(file fs.File) error {
	scanner := bufio.NewScanner(file)
	var key, value string
	var isMultiLine bool

	addProperty := func() {
		if key != "" {
			s.Set(strings.TrimSpace(key), strings.TrimSpace(value))
			key, value = "", ""
		}
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if isMultiLine {
			if strings.HasSuffix(line, "\\") {
				value += line[:len(line)-1]
			} else {
				value += line
				isMultiLine = false
				addProperty()
			}
			continue
		}

		if line == "" || line[0] == '#' || strings.HasPrefix(line, "//") {
			continue
		}

		index := strings.IndexByte(line, '=')
		if index == -1 {
			continue
		}

		key = line[:index]
		value = line[index+1:]

		if strings.HasSuffix(value, "\\") {
			isMultiLine = true
			value = value[:len(value)-1]
		} else {
			addProperty()
		}
	}

	addProperty()

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read property file: %w", err)
	}
	return nil
}

type PropertyUsageValidator struct {
	usedKeys map[string]struct{}
}

func NewPropertyUsageValidator() *PropertyUsageValidator {
	return &PropertyUsageValidator{
		usedKeys: make(map[string]struct{}),
	}
}

func (v *PropertyUsageValidator) Add(key string) {
	v.usedKeys[key] = struct{}{}
}

func (v *PropertyUsageValidator) Used(key string) bool {
	_, ok := v.usedKeys[key]
	return ok
}

func (v *PropertyUsageValidator) ValidateUnused(keys []string) error {
	unused := make([]string, 0)
	for _, key := range keys {
		if !v.Used(key) {
			unused = append(unused, key)
		}
	}

	if len(unused) == 0 {
		return nil
	}

	sort.Strings(unused)
	return fmt.Errorf("unused properties: %v", unused)
}
