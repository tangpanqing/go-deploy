package godeploy

import (
	"github.com/tangpanqing/godeploy/tools"
)

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

func DeployForUbuntu(serverInfo ServerInfo, appInfo AppInfo) {
	Deploy(serverInfo, appInfo, "sudo ")
}

func DeployForCentOS(serverInfo ServerInfo, appInfo AppInfo) {
	Deploy(serverInfo, appInfo, "")
}

// Deploy runParam just like "-port=3000 -profile=dev >/dev/null 2>&1" , full cmd is: nohup fileName -port=3000 -profile=dev >/dev/null 2>&1 &
func Deploy(serverInfo ServerInfo, appInfo AppInfo, sudoStr string) {
	if appInfo.RemotePath == "" {
		panic("远程路径不能为空")
	}

	tools.GenLocalFile(appInfo.FileName)
	client := tools.ConnectServer(serverInfo.Username, serverInfo.Password, serverInfo.ServerIp, serverInfo.ServerPort)
	defer client.Close()

	tools.RebuildDir(client, appInfo, sudoStr)
	tools.UploadFiles(client, appInfo, sudoStr)
	tools.RestartApp(client, appInfo, sudoStr)
	tools.DelLocalFile(appInfo.FileName)
	tools.GetAppInfo(client, appInfo)
}
