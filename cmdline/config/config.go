package config

import (
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type Config struct {
	viper *viper.Viper
}

var AppConfig *Config

func init() {
	InitConfig()
}

func InitConfig() {
	viper.SetConfigName("UploaderConfig")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/azure-blob-uploader/")
	viper.AddConfigPath("$HOME/.azure-blob-uploader/")

	err := viper.ReadInConfig()
	if err != nil {
	}

	AppConfig = &Config{
		viper: viper.GetViper(),
	}
}

func (c *Config) GetRedirectURL() *string {
	url := c.viper.GetString("redirect_url")
	return &url
}

func (c *Config) GetClientID() *string {
	id := c.viper.GetString("client_id")
	return &id
}

func (c *Config) SetToken(token *oauth2.Token) {
	c.viper.Set("token", token.AccessToken)
	c.viper.WriteConfigAs("UploaderConfig.yaml")
}

func (c *Config) GetToken() string {
	return c.viper.GetString("token")
}

func (c *Config) WriteConfig() {
	c.viper.WriteConfig()
}
