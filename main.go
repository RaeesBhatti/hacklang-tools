package main

import (
	"os/exec"
	"os"
	"io"
	"fmt"
	"strings"
	"errors"
	"path/filepath"
	"io/ioutil"
	"encoding/json"
	"runtime"
	"crypto/sha1"
	"encoding/hex"
	"bufio"
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

	cmd := exec.Command(runCommand, os.Args[1:]...)

	if conf.Provider == "wsl" {
		if runtime.GOOS != "windows" {
			fmt.Println("Windows Subsystem for linux is only supported with Windows. Please choose Docker as Provider.")
			return
		}

		cmd = exec.Command("bash", "-c", `"` + strings.Join(os.Args[1:], " ") + `"`)
		cmd.Dir = pwd

	} else if conf.Provider == "docker" {
		containerId, err := getDockerContainerId(conf.RemotePath, conf.Image)
		if err != nil {
			panic(err)
		}
		cmd = exec.Command("docker", "exec", containerId,
			"/bin/sh", "-c", `cd "` + conf.RemotePath + `"; `+ runCommand +` ` +  strings.Join(os.Args[1:], " "))
	} else {
		fmt.Println(`Please choose either "wsl" or "docker" as Provider.`)
		return
	}

	stdout, err := cmd.StdoutPipe();
	if err != nil {
		panic(err)
	}
	stderr, err := cmd.StderrPipe();
	if err != nil {
		panic(err)
	}

	if err := cmd.Start(); err != nil {
		panic(err)
	}

	defer cmd.Wait()

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

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
			config.RemotePath = determineRemotePath(filepath.Dir(file), config.Provider)
			return config, nil
		}
	}

	return config, errors.New("No .hhtools config file found in the current or any parent directory.")
}

func determineRemotePath(path string, provider string) string {
	if runtime.GOOS == "windows" && provider == "docker" {
		return filepath.ToSlash(
			strings.Replace(path, filepath.VolumeName(path),
				`/` + strings.ToLower(strings.Replace(filepath.VolumeName(path), ":", "", 1)),
				1))
	}
	return path
}

func getDockerContainerId(path string, image string) (string, error) {
	var toReturn string
	h := sha1.New()
	h.Write([]byte(path))
	hash := hex.EncodeToString(h.Sum(nil))

	tmp := filepath.Join(os.TempDir(), "hhtools_" + hash + ".tmp")

	if _, err := os.Stat(tmp); err == nil {
		dat, err := ioutil.ReadFile(tmp)
		if err != nil {
			panic(err)
		}

		if len(dat) != 64 {
			id, err := createDockerContainer(path, image, tmp)
			if err != nil {
				return toReturn, err
			}
			return id, nil
		}
		return string(dat), nil
	}
	id, err := createDockerContainer(path, image, tmp)
	if err != nil {
		return toReturn, err
	}
	return id, nil
}

func createDockerContainer(path string, image string, tmp string) (string, error) {
	var toReturn string

	fmt.Println("Starting Docker container. It may take some time if container image has to be downloaded.")

	cmd := exec.Command("docker", "run", "-t", "-d", "-v",  path + ":" + path, image)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return toReturn, err
	}
	scanner := bufio.NewScanner(stdout)
	go func() {
		for scanner.Scan() {
			toReturn += scanner.Text()
		}
	}()
	err = cmd.Start()
	if err != nil {
		return toReturn, err
	}

	err = cmd.Wait()
	if err != nil {
		return toReturn, err
	}

	splits := strings.Split(toReturn, "\n")
	toReturn = string(splits[len(splits) - 1])

	err = ioutil.WriteFile(tmp, []byte(toReturn), 0644)
	if err != nil {
		return toReturn, err
	}

	fmt.Println("Docker container has been started successfuly")

	return toReturn, nil
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
