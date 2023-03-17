package godeploy

import (
	"github.com/tangpanqing/godeploy/structs"
	"github.com/tangpanqing/godeploy/tools"
)

func DeployForUbuntu(serverInfo structs.ServerInfo, appInfo structs.AppInfo) {
	Deploy(serverInfo, appInfo, "sudo ")
}

func DeployForCentOS(serverInfo structs.ServerInfo, appInfo structs.AppInfo) {
	Deploy(serverInfo, appInfo, "")
}

// Deploy runParam just like "-port=3000 -profile=dev >/dev/null 2>&1" , full cmd is: nohup fileName -port=3000 -profile=dev >/dev/null 2>&1 &
func Deploy(serverInfo structs.ServerInfo, appInfo structs.AppInfo, sudoStr string) {
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
