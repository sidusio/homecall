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
	// Alternatively, you can set the raw key directly
	// Takes precedence over JitsiKeyFile
	JitsiKeyRaw string `envconfig:"JITSI_KEY_RAW" required:"false"`
	JitsiDomain string `envconfig:"JITSI_DOMAIN" default:"8x8.vc"`

	AuthDisabled bool   `envconfig:"AUTH_DISABLED" default:"false"`
	AuthIssuer   string `envconfig:"AUTH_ISSUER" default:"https://homecall.eu.auth0.com/"`
	AuthAudience string `envconfig:"AUTH_AUDIENCE" default:"https://homecall.sidus.io"`

	// Firebase
	FirebaseProjectId    string `envconfig:"FIREBASE_PROJECT_ID" required:"false"`
	MockNotificationsDir string `envconfig:"MOCK_NOTIFICATIONS_DIR" required:"false"`
}
