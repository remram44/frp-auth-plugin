package configfile

type Config struct {
}

func New(file string) (*Config, error) {
	config := &Config{}
	return config, nil
}
