package config

var GlobalConfig Config

type Config struct {
	Http        HttpConfig        `mapstructure:"http"`
	Kafka       KafkaConfig       `mapstructure:"kafka"`
	Lark        LarkConfig        `mapstructure:"lark"`
	AlertFields AlertFieldsConfig `mapstructure:"alertFields"`
}

type AlertFieldsConfig struct {
	AlertNameKeys    []string `mapstructure:"alertNameKeys"`
	ProjectKeys      []string `mapstructure:"projectKeys"`
	NotifyEmailsKeys []string `mapstructure:"notifyEmailsKeys"`
	GrafanaURLKeys   []string `mapstructure:"grafanaUrlKeys"`
	RunBookURLKeys   []string `mapstructure:"runbookUrlKeys"`
	DescriptionKeys  []string `mapstructure:"descriptionKeys"`
}


type HttpConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type KafkaConfig struct {
	Brokers           []string `mapstructure:"brokers"`
	Topic             string   `mapstructure:"topic"`
	ConsumerGroup     string   `mapstructure:"consumerGroup"`
	MaxBytes          int      `mapstructure:"maxBytes"`
	WriteRetries      int      `mapstructure:"writeRetries"`
	WriteRetryBackoff int      `mapstructure:"writeRetryBackoffMs"`
}

type LarkConfig struct {
	AppID             string `mapstructure:"appID"`
	AppSecret         string `mapstructure:"appSecret"`
	EncryptKey        string `mapstructure:"encryptKey"`
	VerificationToken string `mapstructure:"verificationToken"`
	ChatID            string `mapstructure:"chatID"`
	SendRetries       int    `mapstructure:"sendRetries"`
	SendRetryBackoff  int    `mapstructure:"sendRetryBackoffMs"`
}
