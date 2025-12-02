package conf

import "github.com/jasontconnell/conf"

type Config struct {
	Site     Site     `json:"site"`
	Interval int      `json:"interval"`
	Domains  []Domain `json:"domains"`
}

type Site struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type Domain struct {
	Key     string            `json:"key"`
	Scheme  string            `json:"scheme"`
	Domain  string            `json:"domain"`
	Headers map[string]string `json:"headers"`
}

func LoadConfig(fn string) (Config, error) {
	cfg := Config{}

	err := conf.LoadConfig(fn, &cfg)
	return cfg, err
}
