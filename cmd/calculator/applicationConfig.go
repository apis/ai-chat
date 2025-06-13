package main

type applicationConfig struct {
	Host string `config_default:"localhost" config_description:"Server host interface"`
	Port int    `config_default:"8081" config_description:"Server port"`
}
