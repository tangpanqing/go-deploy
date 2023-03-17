package structs

type ServerInfo struct {
	Username   string
	Password   string
	ServerIp   string
	ServerPort string
}

type AppInfo struct {
	FileName   string
	RemotePath string
	RunParam   string
	DirList    []string
}
