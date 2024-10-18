package configfile

import (
	"context"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Users []User `yaml:"users"`
}

type User struct {
	Username string  `yaml:"username"`
	Password string  `yaml:"password"`
	Proxies  []Proxy `yaml:"proxies"`
}

type Proxy struct {
	Name          string   `yaml:"name"`
	CustomDomains []string `yaml:"custom_domains"`
	HttpUser      string   `yaml:"http_user"`
	HttpPassword  string   `yaml:"http_password"`
}

type ConfigFile struct {
	lastModified time.Time
	config       *Config
}

func New(file string, ctx context.Context) (*ConfigFile, error) {
	// Do first load
	fileInfo, err := os.Stat(file)
	if err != nil {
		return nil, err
	}
	lastModified := fileInfo.ModTime()

	fp, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	decoder := yaml.NewDecoder(fp)
	decoder.KnownFields(true)
	var config Config
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	configFile := &ConfigFile{
		lastModified: lastModified,
		config:       &config,
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

			fp, err := os.Open(file)
			if err != nil {
				log.Printf("Can't open config file: %s", err)
				continue
			}
			decoder := yaml.NewDecoder(fp)
			decoder.KnownFields(true)
			var newConfig Config
			err = decoder.Decode(&newConfig)
			if err != nil {
				log.Printf("Can't read config file: %s", err)
				continue
			}

			log.Print("New config loaded")
			encoder := yaml.NewEncoder(log.Writer())
			encoder.SetIndent(2)
			encoder.Encode(newConfig)

			configFile.config = &newConfig
			configFile.lastModified = fileInfo.ModTime()
		}
	}()

	return configFile, nil
}

func (cf *ConfigFile) CurrentConfig() *Config {
	return cf.config
}
