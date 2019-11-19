//Copyright 2017-2019 Baidu Inc.
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http: //www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package environment

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"os"
	"os/exec"
	"rasp-cloud/conf"
	"rasp-cloud/tools"
	"syscall"
)

var (
	Version             = "1.2"
	UpdateMappingConfig map[string]interface{}
	StartBeego          = true
)

func init() {
	chdir()
	StartFlag := &conf.Flag{}
	StartFlag.StartType = flag.String("type", "", "Specify startup type")
	StartFlag.Daemon = flag.Bool("d", false, "Run program in background")
	StartFlag.Version = flag.Bool("version", false, "Print program version")
	StartFlag.Upgrade = flag.String("upgrade", "", "Execute upgrade job, e.g update ElasticSearch mapping")
	flag.Parse()
	if *StartFlag.Version {
		handleVersionFlag()
	}
	beego.Info("Version: " + Version)
	if tools.BuildTime != "" {
		beego.Info("Build Time: " + tools.BuildTime)
	}
	if tools.CommitID != "" {
		beego.Info("Git Commit ID: " + tools.CommitID)
	}
	if *StartFlag.Upgrade != "" {
		StartBeego = false
		HandleUpgrade(*StartFlag.Upgrade)
	}
	if *StartFlag.StartType == conf.StartTypeReset {
		HandleReset(StartFlag)
	}
	if *StartFlag.Daemon {
		HandleDaemon()
	}
	initLogger()
	initEnvConf()
	if *StartFlag.StartType == "" {
		allType := conf.StartTypeDefault
		StartFlag.StartType = &allType
	}
	conf.InitConfig(StartFlag)
	beego.Info("===== Startup type: " + *StartFlag.StartType + " =====")
}

func handleVersionFlag() {
	fmt.Println("Version:       " + Version)
	if tools.BuildTime != "" {
		fmt.Println("Build Time:    " + tools.BuildTime)
	}
	if tools.CommitID != "" {
		fmt.Println("Git Commit ID: " + tools.CommitID)
	}
	os.Exit(0)
}

func chdir() {
	path, err := tools.GetCurrentPath()
	if err != nil {
		tools.Panic(tools.ErrCodeChDirFailed, "failed to get current dir", err)
	}
	fmt.Println(path)
	err = os.Chdir(path)
	if err != nil {
		tools.Panic(tools.ErrCodeMongoInitFailed, "failed to change dir to "+path, err)
	}
}

func HandleUpgrade(flag string) {
	UpdateMappingConfig = make(map[string]interface{})
	switch flag {
	case "120to121":
		log.Println("Going to update ElasticSearch mapping")

		UpdateMappingConfig["attack-alarm-template"] = "120to121"
		UpdateMappingConfig["policy-alarm-template"] = "120to121"
		UpdateMappingConfig["error-alarm-template"] = "120to121"
	case "121to122":
		log.Println("Going to update 121to122")
	default:
		log.Println("Unknown upgrade job specified: " + flag)
	}
}

func HandleReset(startFlag *conf.Flag) {
	fmt.Print("Enter new admin password: ")
	pwd1, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		fmt.Println("failed to read password from terminal: " + err.Error())
		os.Exit(tools.ErrCodeResetUserFailed)
	}
	fmt.Print("Retype new admin password: ")
	pwd2, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		fmt.Println("failed to read password from terminal: " + err.Error())
		os.Exit(tools.ErrCodeResetUserFailed)
	}
	if bytes.Compare(pwd1, pwd2) != 0 {
		fmt.Println("Sorry, passwords do not match")
		os.Exit(tools.ErrCodeResetUserFailed)
	} else {
		pwd := string(pwd1)
		startFlag.Password = &pwd
	}
}

func HandleDaemon() {
	err := fork()
	if err != nil {
		tools.Panic(tools.ErrCodeInitChildProcessFailed, "failed to launch child process, error", err)
	}
	log.Println("rasp-cloud started successfully, for details please refer to 'logs/api/agent-cloud.log'")
	os.Exit(0)
}

func fork() (err error) {
	path := os.Args[0]
	var args []string
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			if arg == "-d" {
				break
			}
			args = append(args, arg)
		}
	}
	log.Println("args:", args)
	cmd := exec.Command(path, args...)
	err = cmd.Start()
	return
}

func initLogger() {
	logPath := "logs/api"
	if isExists, _ := tools.PathExists(logPath); !isExists {
		err := os.MkdirAll(logPath, os.ModePerm)
		if err != nil {
			tools.Panic(tools.ErrCodeLogInitFailed, "failed to create logs/api dir", err)
		}
	}
	logs.SetLogFuncCall(true)
	logs.SetLogger(logs.AdapterFile,
		`{"filename":"`+logPath+`/agent-cloud.log","daily":true,"maxdays":10,"perm":"0777"}`)

}

func initEnvConf() {
	if beego.BConfig.RunMode == "dev" {
		logs.SetLevel(beego.LevelDebug)
	} else {
		logs.SetLevel(beego.LevelInformational)
		beego.BConfig.EnableErrorsShow = false
		beego.BConfig.EnableErrorsRender = false
	}
}
