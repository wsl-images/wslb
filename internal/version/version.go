package version

var (
	Version = "dev"
)

func GetVersionInfo() string {
	return "WSLB version: " + Version
}
