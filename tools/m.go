package tools

import (
	"fmt"
	"github.com/pkg/sftp"
	"github.com/tangpanqing/godeploy"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
)

func GenLocalFile(fileName string) {
	exec.Command("powershell", "$env:CGO_ENABLED=0;$env:GOOS=\"linux\";$env:GOARCH=\"amd64\";go build -ldflags=\"-s -w\" "+fileName+".go").Run()
	fmt.Println("已生成文件 " + fileName)
}

func ConnectServer(username string, password string, serverIp string, serverPort string) *ssh.Client {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, _ := ssh.Dial("tcp", serverIp+":"+serverPort, config)
	fmt.Println("已连接服务器 " + serverIp)
	return client
}

func RebuildDir(client *ssh.Client, appInfo godeploy.AppInfo, sudoStr string) {
	runCommand(client, strings.Join([]string{
		sudoStr + "rm -rf " + appInfo.RemotePath,
		sudoStr + "mkdir " + appInfo.RemotePath,
		sudoStr + "chmod 777 " + appInfo.RemotePath,
	}, " ; "))

	fmt.Println("已重建目录 " + appInfo.RemotePath)
}

func UploadFiles(client *ssh.Client, appInfo godeploy.AppInfo, sudoStr string) {
	//本地所有文件，包含主文件，以及配置文件，模板文件，静态文件等等
	allFiles := getAllFiles(appInfo.FileName, appInfo.DirList)
	for i := 0; i < len(allFiles); i++ {
		uploadFile(client, allFiles[i], appInfo.RemotePath, sudoStr)
	}

	//赋予新上传的文件可执行权限
	runCommand(client, sudoStr+"chmod 777 "+appInfo.RemotePath+"/"+appInfo.FileName)

	//fmt.Println("已上传文件")
}

func RestartApp(client *ssh.Client, appInfo godeploy.AppInfo, sudoStr string) {
	pid := getAppPid(client, appInfo)
	if pid != "" {
		runCommand(client, sudoStr+"kill -9 "+pid)
	}

	runCommand(client, strings.Join([]string{
		"cd " + appInfo.RemotePath,
		"nohup ./" + appInfo.FileName + " " + appInfo.RunParam + " &",
	}, " ; "))

	fmt.Println("已重启应用")
}

func DelLocalFile(fileName string) {
	exec.Command("powershell", "rm "+fileName).Run()
	fmt.Println("已删除文件 " + fileName)
}

func GetAppInfo(client *ssh.Client, appInfo godeploy.AppInfo) {
	pid := getAppPid(client, appInfo)
	if pid == "" {
		fmt.Println("部署失败 ")
	} else {
		fmt.Println("已完成部署,最新pid=" + pid)
	}
}

func runCommand(client *ssh.Client, cmd string) string {
	s1, _ := client.NewSession()
	defer s1.Close()
	buf, _ := s1.CombinedOutput(cmd)

	//fmt.Println("已执行命令 " + cmd)
	return string(buf)
}

//GetAllFiles 获取指定目录下的所有文件,包含子目录下的文件
func GetAllFiles(dirPth string) (files []string, err error) {
	var dirs []string
	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		return nil, err
	}

	PthSep := string(os.PathSeparator)
	for _, fi := range dir {
		if fi.IsDir() { // 目录, 递归遍历
			dirs = append(dirs, dirPth+PthSep+fi.Name())
			GetAllFiles(dirPth + PthSep + fi.Name())
		} else {
			files = append(files, dirPth+"/"+fi.Name())
		}
	}

	// 读取子目录下文件
	for _, table := range dirs {
		temp, _ := GetAllFiles(table)
		for _, temp1 := range temp {
			files = append(files, temp1)
		}
	}

	return files, nil
}

func getAllFiles(fileName string, dirList []string) []string {
	var allFiles []string
	allFiles = append(allFiles, fileName)
	for i := 0; i < len(dirList); i++ {
		files, _ := GetAllFiles(dirList[i])
		allFiles = append(allFiles, files...)
	}

	return allFiles
}

func uploadFile(client *ssh.Client, fileNamePath string, remotePath string, sudoStr string) {
	sftpClient, _ := sftp.NewClient(client)
	defer sftpClient.Close()

	//打开本地文件流
	srcFile, err := os.Open(fileNamePath)
	if err != nil {
		fmt.Println("os.Open error : ", fileNamePath)
		log.Fatal(err)

	}
	//关闭文件流
	defer srcFile.Close()
	//上传到远端服务器的文件名,与本地路径末尾相同
	//var remoteFileName = path.Base(fileNamePath)
	var remoteFileName = fileNamePath
	pathArr := strings.Split(remoteFileName, "/")
	if len(pathArr) > 1 {
		for i := 0; i < len(pathArr)-1; i++ {
			ss := path.Join(remotePath, pathArr[i])
			ss = strings.ReplaceAll(ss, "\\", "/")

			runCommand(client, strings.Join([]string{
				sudoStr + "mkdir " + ss,
				sudoStr + "chmod 777 " + ss,
			}, " ; "))
		}
	}

	allFileName := path.Join(remotePath, remoteFileName)
	fmt.Println("正上传文件 " + allFileName)

	//转换符号
	allFileName = strings.ReplaceAll(allFileName, "\\", "/")

	//打开远程文件,如果不存在就创建一个
	dstFile, err := sftpClient.Create(allFileName)
	if err != nil {
		fmt.Println("sftpClient.Create error : ", allFileName)
		fmt.Println("-------上传错误----")
		fmt.Println(err)
		log.Fatal(err)
	}
	//关闭远程文件
	defer dstFile.Close()
	//读取本地文件,写入到远程文件中(这里没有分快穿,自己写的话可以改一下,防止内存溢出)
	ff, err := ioutil.ReadAll(srcFile)
	if err != nil {
		fmt.Println("ReadAll error : ", fileNamePath)
		log.Fatal(err)
	}
	dstFile.Write(ff)

	fmt.Println("已上传文件 " + allFileName)
}

func getAppPid(client *ssh.Client, appInfo godeploy.AppInfo) string {
	res2 := runCommand(client, "ps -ef | grep \""+appInfo.FileName+"\" | grep -v \"grep\"")
	if res2 != "" {
		arr := strings.Fields(res2)
		return arr[1]
	}

	return ""
}
