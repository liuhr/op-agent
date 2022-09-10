package util

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/openark/golib/log"
)

var (
	timeout  = 10 * time.Second
	EmptyEnv = []string{}
)

func init() {
	osPath := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s:/usr/sbin:/usr/bin:/sbin:/bin", osPath))
}

// CommandRun executes some text as a command. This is assumed to be
// text that will be run by a shell so we need to write out the
// command to a temporary file and then ask the shell to execute
// it, after which the temporary file is removed.
func RunCommandOutput(commandText string, arguments ...string) (string, error) {
	// show the actual command we have been asked to run
	log.Infof("CommandRun(%v,%+v)", commandText, arguments)

	cmd, shellScript, err := generateShellScript(commandText, arguments...)
	defer os.Remove(shellScript)
	if err != nil {
		return "", log.Errore(err)
	}

	var waitStatus syscall.WaitStatus

	log.Infof("CommandRun/running: %s", strings.Join(cmd.Args, " "))
	cmdOutput, err := cmd.CombinedOutput()
	log.Infof("CommandRun: %s\n", string(cmdOutput))
	if err != nil {
		// Did the command fail because of an unsuccessful exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus = exitError.Sys().(syscall.WaitStatus)
			log.Errorf("CommandRun: failed. exit status %d", waitStatus.ExitStatus())
		}

		return "", log.Errore(fmt.Errorf("(%s) %s", err.Error(), cmdOutput))
	}

	// Command was successful
	waitStatus = cmd.ProcessState.Sys().(syscall.WaitStatus)
	log.Infof("CommandRun successful. exit status %d", waitStatus.ExitStatus())

	return strings.Replace(string(cmdOutput), "\n", "", -1), nil
}

// generateShellScript generates a temporary shell script based on
// the given command to be executed, writes the command to a temporary
// file and returns the exec.Command which can be executed together
// with the script name that was created.
func generateShellScript(commandText string, arguments ...string) (*exec.Cmd, string, error) {
	commandBytes := []byte(commandText)
	tmpFile, err := ioutil.TempFile("", "manager-process-cmd-")
	if err != nil {
		return nil, "", log.Errorf("generateShellScript() failed to create TempFile: %v", err.Error())
	}
	// write commandText to temporary file
	ioutil.WriteFile(tmpFile.Name(), commandBytes, 0640)
	shellArguments := append([]string{}, tmpFile.Name())
	shellArguments = append(shellArguments, arguments...)

	cmd := exec.Command("bash", shellArguments...)
	//cmd.Env = env

	return cmd, tmpFile.Name(), nil
}

//no output
func RunCommandNoOutput(commandText string) error {
	cmd, tmpFileName, err := execCmd(commandText)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFileName)
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	return err
}

func execCmd(commandText string) (*exec.Cmd, string, error) {
	commandBytes := []byte(commandText)
	tmpFile, err := ioutil.TempFile("", "manager-cmd-")
	if err != nil {
		return nil, "", log.Errore(err)
	}
	ioutil.WriteFile(tmpFile.Name(), commandBytes, 0644)
	log.Debugf("execCmd: %s", commandText)
	return exec.Command("bash", tmpFile.Name()), tmpFile.Name(), nil
}

func GetLocalIP() (ipv4 []string, err error) {
	var (
		addrs   []net.Addr
		addr    net.Addr
		ipNet   *net.IPNet
		isIpNet bool
	)
	//ipv4 = make([]string,0)
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return
	}
	for _, addr = range addrs {
		if ipNet, isIpNet = addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				ipv4 = append(ipv4, ipNet.IP.String())
			}
		}
	}
	return
}


func LookupHost(name string) (addrs []string, err error) {
	addr, err := net.LookupHost(name)
	return addr,err
}


func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func IsDir(path string) (bool, error) {
	s, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return s.IsDir(), nil
}

func Zip(src_dir string, zip_file_name string) error {
        cmd := "tar zcvf " + zip_file_name + " " + src_dir
        err := RunCommandNoOutput(cmd)
	return err
}

func GetFileMd5(filename string) (string, error) {
        //path := fmt.Sprintf("./%s", filename)
        pFile, err := os.Open(filename)
        if err != nil {
                return "", err
        }
        defer pFile.Close()
        md5h := md5.New()
        io.Copy(md5h, pFile)

        return hex.EncodeToString(md5h.Sum(nil)), nil
}


func ReadFile(file string) ([]byte, error) {
        return ioutil.ReadFile(file)
        // //获得一个file
        // f, err := os.Open(file)
        // if err != nil {
        //      //fmt.Println("read fail")
        //      return nil
        // }

        // //把file读取到缓冲区中
        // defer f.Close()
        // var chunk []byte
        // buf := make([]byte, 1024)

        // for {
        //      //从file读取到buf中
        //      n, err := f.Read(buf)
        //      if err != nil && err != io.EOF {
        //              fmt.Println("read buf fail", err)
        //              return nil
        //      }
        //      //说明读取结束
        //      if n == 0 {
        //              break
        //      }
        //      //读取到最终的缓冲区中
        //      chunk = append(chunk, buf[:n]...)
        // }

        // return chunk
}

func GetFileSize(file string) (int64, error) {
	fileInfo, err := os.Stat(file)
	if err != nil {
		return 0, err
	}
	fileSize := fileInfo.Size()
	return fileSize, nil
}

func IsIP(ip string) (b bool) {

	m, _ := regexp.MatchString("^[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}$", ip)

	if  !m {
		return false
	}
	return true
}
