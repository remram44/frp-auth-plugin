package configfile

import (
	"context"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type yamlConfig struct {
	Users []yamlUser `yaml:"users"`
}

type yamlUser struct {
	Username string      `yaml:"username"`
	Password string      `yaml:"password"`
	Proxies  []yamlProxy `yaml:"proxies"`
}

type yamlProxy struct {
	Name          string   `yaml:"name"`
	CustomDomains []string `yaml:"custom_domains"`
	HttpUser      string   `yaml:"http_user"`
	HttpPassword  string   `yaml:"http_password"`
}

type Config struct {
	Users map[string]User
}

type User struct {
	Password string
	Proxies  map[string]Proxy
}

type Proxy struct {
	CustomDomains []string
	HttpUser      string
	HttpPassword  string
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
			var newYamlConfig yamlConfig
			err = decoder.Decode(&newYamlConfig)
			if err != nil {
				log.Printf("Can't read config file: %s", err)
				configFile.lastModified = fileInfo.ModTime()
				continue
			}

			// Convert from YAML format (lists) to internal (maps)
			var newConfig Config
			newConfig.Users = make(map[string]User)
			for _, user := range newYamlConfig.Users {
				newUser := User{
					Password: user.Password,
					Proxies:  make(map[string]Proxy),
				}
				for _, proxy := range user.Proxies {
					newUser.Proxies[proxy.Name] = Proxy{
						CustomDomains: proxy.CustomDomains,
						HttpUser:      proxy.HttpUser,
						HttpPassword:  proxy.HttpPassword,
					}
				}
				newConfig.Users[user.Username] = newUser
			}

			log.Print("New config loaded")
			configFile.config = &newConfig
			configFile.lastModified = fileInfo.ModTime()
		}
	}()

	return configFile, nil
}

func (cf *ConfigFile) CurrentConfig() *Config {
	return cf.config
}
