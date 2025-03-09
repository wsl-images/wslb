package version

var (
	Version = "1.0.8"
)

func GetVersionInfo() string {
	return "WSLB version: " + Version
}
