package main

import (
	"os/exec"
	"os"
	"fmt"
	"strings"
	"errors"
	"path/filepath"
	"io/ioutil"
	"encoding/json"
	"runtime"
	"crypto/sha1"
	"encoding/hex"
	"bytes"
)

func main() {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	conf, err := readConfig(pwd)
	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}

	runSplits := strings.Split(os.Args[0], string(os.PathSeparator))
	runCommand := string(runSplits[len(runSplits) - 1])

	// Remove this when building
	runCommand = "hh_client"

	if conf.Provider == "wsl" {
		if runtime.GOOS != "windows" {
			fmt.Println("Windows Subsystem for linux is only supported with Windows. Please choose Docker as Provider.")
			return
		}

		outBuff, errBuff, err := executeCommand("bash", []string{"-c", `"` + strings.Join(os.Args[1:], " ") + `"`}, pwd)

		if err != nil {
			panic(err)
		}
		fmt.Print(outBuff, errBuff)

		return

	} else if conf.Provider == "docker" {

		containerName := getContainerName(conf.LocalPath)

		_, errBuff, err := executeCommand("docker", []string{"exec", containerName,
			"/bin/sh", "-c", `cd "` + conf.RemotePath + `"; `+ runCommand +` ` +  strings.Join(os.Args[1:], " ")},
			"")

		if err != nil {
			panic(err)
		}

		if len(errBuff.String()) > 0 {
			if strings.Contains(errBuff.String(), "is not running") {
				fmt.Println("Docker container is not running, going to start again")
				err := startDockerContainer(containerName)
				if err != nil {
					panic(err)
				}
			} else if strings.Contains(errBuff.String(), "No such container: ") {
				err := createDockerContainer(conf.RemotePath, conf.Image, containerName)
				if err != nil {
					panic(err)
				}

				outBuff, errBuff, err := executeCommand("docker", []string{"exec", containerName,
					"/bin/sh", "-c", `cd "` + conf.RemotePath + `"; `+ runCommand +` ` +  strings.Join(os.Args[1:], " ")},
					"")
				if err != nil {
					panic(err)
				}

				fmt.Print(outBuff.String(), errBuff.String())

				return
			}

		}

		outBuff, errBuff, err := executeCommand("docker", []string{"exec", containerName,
			"/bin/sh", "-c", `cd "` + conf.RemotePath + `"; `+ runCommand +` ` +  strings.Join(os.Args[1:], " ")},
			"")
		if err != nil {
			panic(err)
		}

		fmt.Print(outBuff.String(), errBuff.String())

		return

	}

	fmt.Println(`Please choose either "wsl" or "docker" as Provider.`)

}

func readConfig(pwd string) (Config, error) {
	splits := strings.Split(pwd, string(os.PathSeparator))

	var path string
	var config Config

	for i := 0; i < len(splits); i++ {
		path = strings.Join(splits[:len(splits)-i], string(os.PathSeparator))
		if strings.EqualFold(path, ""){
			path = string(os.PathSeparator)
		}
		file := filepath.Join(path, ".hhtools")

		if _, err := os.Stat(file); err == nil {

			dat, err := ioutil.ReadFile(file)
			if err != nil {
				panic(err)
			}

			err = json.Unmarshal(dat, &config)
			if err != nil {
				fmt.Println("Invalid JSON at", file, ".")
				panic(err)
			}

			if config.verify() != true {
				return config, errors.New("Invalid configuration detected")
			}
			config.LocalPath = filepath.Dir(file)
			config.RemotePath = translateLocalPath(filepath.Dir(file), config.Provider)
			return config, nil
		}
	}

	return config, errors.New("No .hhtools config file found in the current or any parent directory.")
}

func translateLocalPath(path string, provider string) string {
	if runtime.GOOS == "windows" && provider == "docker" {
		return filepath.ToSlash(
			strings.Replace(path, filepath.VolumeName(path),
				`/` + strings.ToLower(strings.Replace(filepath.VolumeName(path), ":", "", 1)),
				1))
	}
	return path
}

func escapePathForWindows(path string) string {
	if runtime.GOOS == "windows" {
		return strings.Join(
			strings.Split(
				strings.Join(
					strings.Split(path, string(os.PathSeparator)), string(os.PathSeparator) + string(os.PathSeparator)),
				"/"),
			string(os.PathSeparator) + string(os.PathSeparator))
	}
	return path
}

func getContainerName(path string) string {
	h := sha1.New()
	h.Write([]byte(path))
	hash := hex.EncodeToString(h.Sum(nil))

	return "hhtools_" + hash
}

func createDockerContainer(path string, image string, containerName string) error {

	fmt.Println("Starting Docker container. It may take some time if container image has to be downloaded.")

	_, errBuff, err := executeCommand("docker",
		[]string{"run", "-t", "-d", "--name=" + containerName, "-v", path + ":" + path, image}, "")
	if err != nil {
		return err
	} else if len(errBuff.String()) > 0 {
		return errors.New(errBuff.String())
	}

	fmt.Println("Docker container has been started successfuly")
	return nil
}

func startDockerContainer(containerID string) error {
	_, errBuff, err := executeCommand("docker", []string{"start", containerID}, "")
	if err != nil {
		return err
	} else if len(errBuff.String()) > 0 {
		return errors.New(errBuff.String())
	}

	return nil
}

func executeCommand(command string, args []string, pwd string) (*bytes.Buffer, *bytes.Buffer, error) {
	cmd := exec.Command(command, args...)

	if len(pwd) > 0 {
		cmd.Dir = pwd
	}

	outBuff := new(bytes.Buffer)
	errBuff := new(bytes.Buffer)

	stdout, err := cmd.StdoutPipe();
	if err != nil {
		return outBuff, errBuff, nil
	}
	stderr, err := cmd.StderrPipe();
	if err != nil {
		return outBuff, errBuff, nil
	}

	if err := cmd.Start(); err != nil {
		return outBuff, errBuff, nil
	}

	defer cmd.Wait()

	outBuff.ReadFrom(stdout)
	errBuff.ReadFrom(stderr)

	return outBuff, errBuff, nil
}

func dockerExec(command string, args []string, containerName string, runCommand string, conf Config) (*bytes.Buffer, *bytes.Buffer, error) {

	outBuff, errBuff, err := executeCommand(command, args, "")

	if err != nil {
		panic(err)
	}

	if len(errBuff.String()) > 0 {
		if strings.Contains(errBuff.String(), "is not running") {
			fmt.Println("Docker container is not running, going to start again")
			err := startDockerContainer(containerName)
			if err != nil {
				return outBuff, errBuff, err
			}
		} else if strings.Contains(errBuff.String(), "No such container: ") {
			err := createDockerContainer(conf.RemotePath, conf.Image, containerName)
			if err != nil {
				return outBuff, errBuff, err
			}

			outBuff, errBuff, err := executeCommand("docker", []string{"exec", containerName,
				"/bin/sh", "-c", `cd "` + conf.RemotePath + `"; `+ runCommand +` ` +  strings.Join(os.Args[1:], " ")},
				"")
			return outBuff, errBuff, err
		}

	}

	outBuff, errBuff, err = executeCommand("docker", []string{"exec", containerName,
		"/bin/sh", "-c", `cd "` + conf.RemotePath + `"; `+ runCommand +` ` +  strings.Join(os.Args[1:], " ")},
		"")

	return outBuff, errBuff, nil
}

type Config struct {
	LocalPath	string
	RemotePath	string
	Provider	string	`json:"provider"`
	Image		string	`json:"image"`
}

func (c Config) verify() bool {
	if len(c.Provider) > 0 {
		return true
	}
	return false
}
