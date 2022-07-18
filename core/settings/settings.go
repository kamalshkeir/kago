package settings

var GlobalConfig = &config{}
var Secret string

type config struct {
	Host           string
	Port           string
	Profiler       bool
	Docs           bool
	Logs           bool
	Monitoring     bool
	EmbedStatic    bool
	EmbedTemplates bool
	DbType         string
	DbDSN          string
	DbName         string
	SmtpEmail      string
	SmtpPass       string
	SmtpHost       string
	SmtpPort       string
}