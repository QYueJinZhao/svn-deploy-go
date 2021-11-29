package lib

import (
	"encoding/json"
	"os"
	"log"
)

type Config struct {
	Name      string  `json:"name"`
	URL       string  `json:"url"`
	Username  string  `json:"username"`
	Password  string  `json:"password"`
	LocalPath string  `json:"local"`
	Lastver   int  `json:"lastver"`
}

type ConfigArr struct {
    Persons []Config `json:"Persons"`
	savePath string
}

func NewConfig(filename string) (err error, c *ConfigArr) {
	c = &ConfigArr{}
	err = c.load(filename)
	c.savePath = filename
	return
}

func (c *ConfigArr) load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		log.Println(err)
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&c)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (c *ConfigArr) Save() error {
	file, err := os.Create(c.savePath)
	if err != nil {
		log.Println(err)
		return err
	}
	defer file.Close()
	data, err2 := json.MarshalIndent(c, "", "    ")
	if err2 != nil {
		log.Println(err2)
		return err2
	}
	_, err3 := file.Write(data)
	if err3 != nil {
		log.Println(err3)
	}
	return err3
}
