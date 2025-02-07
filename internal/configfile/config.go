package configfile

import (
	"bytes"
	"context"
	"log"
	"github.com/Masterminds/sprig/v3"
	"os"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Users map[string]User `yaml:"users"`
}

type User struct {
	Password string           `yaml:"password"`
	Proxies  map[string]Proxy `yaml:"proxies"`
}

type Proxy struct {
	CustomDomains []string `yaml:"customDomains"`
	HttpUser      string   `yaml:"httpUser"`
	HttpPassword  string   `yaml:"httpPassword"`
}

type ConfigFile struct {
	lastModified time.Time
	config       *Config
}

type Values struct {
	Envs map[string]string // Environment variables
}

func load(file string, values *Values) (*Config, error) {
	// Load file
	inputBuffer, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	// Process template
	tmpl, err := template.New("frp-auth").Funcs(sprig.FuncMap()).Parse(string(inputBuffer))
	if err != nil {
		return nil, err
	}
	outputBuffer := bytes.NewBufferString("")
	err = tmpl.Execute(outputBuffer, values)
	if err != nil {
		return nil, err
	}

	// Load as YAML
	decoder := yaml.NewDecoder(outputBuffer)
	decoder.KnownFields(true)
	var config Config
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func New(file string, ctx context.Context) (*ConfigFile, error) {
	// Get values for template rendering (environment variables)
	values := &Values {
		Envs: make(map[string]string),
	}
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			continue
		}
		values.Envs[pair[0]] = pair[1]
	}

	// Do first load
	fileInfo, err := os.Stat(file)
	if err != nil {
		return nil, err
	}
	lastModified := fileInfo.ModTime()

	config, err := load(file, values)
	if err != nil {
		return nil, err
	}

	configFile := &ConfigFile{
		lastModified: lastModified,
		config:       config,
	}

	// Reload file automatically in the background
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Second):
			}

			fileInfo, err := os.Stat(file)
			if err != nil {
				log.Printf("Can't stat config file: %s", err)
				continue
			}

			if fileInfo.ModTime() == configFile.lastModified {
				continue
			}

			newConfig, err := load(file, values)
			if err != nil {
				log.Printf("Can't read config file: %s", err)
				continue
			}

			log.Print("New config loaded")
			configFile.config = newConfig
			configFile.lastModified = fileInfo.ModTime()
		}
	}()

	return configFile, nil
}

type ConfigProvider interface {
	CurrentConfig() *Config
}

func (cf *ConfigFile) CurrentConfig() *Config {
	return cf.config
}
