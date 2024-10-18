package configfile

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"
)

type Config struct {
	Users []User `json:"users"`
}

type User struct {
	Username string  `json:"username"`
	Password string  `json:"password"`
	Proxies  []Proxy `json:"proxies"`
}

type Proxy struct {
	Name          string   `json:"name"`
	CustomDomains []string `json:"custom_domains"`
	HttpUser      string   `json:"http_user"`
	HttpPassword  string   `json:"http_password"`
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
	decoder := json.NewDecoder(fp)
	decoder.DisallowUnknownFields()
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
			case <-time.After(1 * time.Minute):
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
			decoder := json.NewDecoder(fp)
			decoder.DisallowUnknownFields()
			var newConfig Config
			err = decoder.Decode(&newConfig)
			if err != nil {
				log.Printf("Can't read config file: %s", err)
				continue
			}

			log.Print("New config loaded")
			encoder := json.NewEncoder(log.Writer())
			encoder.SetIndent("", "")
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
