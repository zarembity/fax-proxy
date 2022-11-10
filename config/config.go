package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Service    Service    `yaml:"service"`
	Connection Connection `yaml:"connection"`
}

type Service struct {
	Port     int    `yaml:"port"`
	Env      string `yaml:"env"`
	Log      string `yaml:"log"`
	ErrorLog string `yaml:"errorLog"`
	Pid      string `yaml:"pid"`
	LogLevel int    `yaml:"logLevel"`
	Image    Image  `yaml:"image"`
}

type Image struct {
	MaxSize   int64    `yaml:"maxSize"`
	MimeTypes []string `yaml:"mimeType"`
}

type Connection struct {
	FreeSwitch FreeSwitch `yaml:"freeswitch"`
	Db         Db         `yaml:"db"`
}

type FreeSwitch struct {
	Addr         string `yaml:"addr"`
	Pass         string `yaml:"pass"`
	LocalPort    string `yaml:"local_port"`
	AudioPath    string `yaml:"audio_path"`
	ReceiveAudio string `yaml:"receive_audio"`
	FaxPath      string `yaml:"fax_path"`
	GW           string `yaml:"gw"`
}

type Db struct {
	DbAddr string `yaml:"addr"`
	DbPort int    `yaml:"port"`
	DbName string `yaml:"db_name"`
	DbUser string `yaml:"db_user"`
	DbPass string `yaml:"db_passwd"`
}

func ReadConfig(cfg *Config, filePath string) error {
	configFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(configFile, cfg)
	if err != nil {
		return err
	}

	return nil
}
