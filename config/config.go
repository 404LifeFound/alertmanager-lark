package config

var GlobalConfig Config

type Config struct {
	Http  HttpConfig  `mapstructure:"http"`
	Kafka KafkaConfig `mapstructure:"kafka"`
	Lark  LarkConfig  `mapstructure:"lark"`
}

type HttpConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type KafkaConfig struct {
	Brokers       []string `mapstructure:"brokers"`
	Topic         string   `mapstructure:"topic"`
	ConsumerGroup string   `mapstructure:"consumerGroup"`
	MaxBytes      int      `mapstructure:"maxBytes"`
}

type LarkConfig struct {
	AppID     string `mapstructure:"appID"`
	AppSecret string `mapstructure:"appSecret"`
	ChatID    string `mapstructure:"chatID"`
}
