package version

var (
	Version = "1.0.4"
)

func GetVersionInfo() string {
	return "WSLB version: " + Version
}
