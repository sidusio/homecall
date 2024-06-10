package app

type Config struct {
	DBHost     string `envconfig:"DB_HOST" default:"localhost"`
	DBPort     string `envconfig:"DB_PORT" default:"8036"`
	DBUser     string `envconfig:"DB_USER" default:"homecall"`
	DBPassword string `envconfig:"DB_PASSWORD" default:"supersecret"`
	DBName     string `envconfig:"DB_NAME" default:"homecall"`

	Port string `envconfig:"PORT" default:"8080"`

	JitsiAppId   string `envconfig:"JITSI_APP_ID" required:"false"`
	JitsiKeyId   string `envconfig:"JITSI_KEY_ID" required:"false"`
	JitsiKeyFile string `envconfig:"JITSI_KEY_FILE" required:"false"`

	AuthDisabled bool   `envconfig:"AUTH_DISABLED" default:"false"`
	AuthIssuer   string `envconfig:"AUTH_ISSUER" default:"https://homecall.eu.auth0.com/"`
	AuthAudience string `envconfig:"AUTH_AUDIENCE" default:"https://homecall.sidus.io"`
}
