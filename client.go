package main

import (
	"errors"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type DockerDeployClient struct {
	Mode                 string
	SSHPort              int
	SSHUser              string
	SSHHost              string
	SSHPassword          string
	ProjectName          string
	ComposeFile          string
	StartTime            int
	RemoteWorkingDir     string
	LocalWorkingDir      string
	LocalArtifact        string
	ServiceDiscoveryPort int
	ClearVolumes         bool
	config               *ssh.ClientConfig
	sshClient            *ssh.Client
}

func (c *DockerDeployClient) connect() error {
	c.config = &ssh.ClientConfig{
		User: c.SSHUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.SSHPassword),
		},
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.SSHHost, c.SSHPort), c.config)
	if err != nil {
		return err
	}
	c.sshClient = client
	return nil
}

func (c *DockerDeployClient) disconnect() error {
	return c.sshClient.Close()
}

func (c *DockerDeployClient) executeCommand(command string, sudo bool) (string, error) {
	session, err := c.sshClient.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	if sudo {
		session.Stdin = strings.NewReader(fmt.Sprintf("%v\n", c.SSHPassword))
	}

	output, err := session.CombinedOutput(command)
	log.Printf("Command: %v", command)
	log.Printf("Output: %v", string(output))
	if err != nil {
		return "", errors.New(fmt.Sprintf("%v: %v", err.Error(), string(output)))
	}
	return string(output), nil
}

func (c *DockerDeployClient) findLocalArtifact() error {
	if c.LocalArtifact != "" {
		if _, err := os.Stat(c.LocalArtifact); err != nil {
			return errors.New(fmt.Sprintf("Could not find artifact at %v.", c.LocalArtifact))
		}
		if path.Ext(c.LocalArtifact) != ".zip" {
			return errors.New(fmt.Sprintf("Given artifact %v is no zip file.", c.LocalArtifact))
		}
		return nil
	}

	if c.LocalWorkingDir == "" {
		return errors.New("No local working directory specified.")
	}

	if _, err := os.Stat(c.LocalWorkingDir); err != nil {
		return errors.New(fmt.Sprintf("Local working directory \"%v\" does not exist!", c.LocalWorkingDir))
	}

	d, err := os.Open(c.LocalWorkingDir)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not open local working directory: %v", err.Error()))
	}
	defer d.Close()

	files, err := d.Readdir(-1)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.Mode().IsRegular() {
			fpath := path.Join(c.LocalWorkingDir, file.Name())
			if path.Ext(fpath) == ".zip" {
				c.LocalArtifact = fpath
				return nil
			}
		}
	}
	return errors.New(fmt.Sprintf("Could not find local artifact in working directory %v", c.LocalWorkingDir))
}

func (c *DockerDeployClient) unzipArtifact() error {
	output, err := c.executeCommand("which unzip", false)
	if err != nil {
		return errors.New(fmt.Sprintf("Unzip not installed. %v", output))
	}

	output, err = c.executeCommand(fmt.Sprintf("cd %v && unzip -o %v && rm %v", c.RemoteWorkingDir, path.Base(c.LocalArtifact), path.Base(c.LocalArtifact)), false)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not unzip artifact: %v %v", err.Error(), output))
	}
	return nil
}

func (c *DockerDeployClient) copyArtifact() error {
	err := c.copyFile(c.LocalArtifact, path.Join(c.RemoteWorkingDir, path.Base(c.LocalArtifact)))
	return err
}

func (c *DockerDeployClient) copyFile(source string, target string) error {
	sftp, err := sftp.NewClient(c.sshClient)
	if err != nil {
		log.Fatalf("Could not initialize SFTP connection: %v", err)
	}
	defer sftp.Close()

	tf, err := sftp.Create(target)
	if err != nil {
		return err
	}
	defer tf.Close()

	sf, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sf.Close()

	n, err := io.Copy(tf, sf)
	if err != nil {
		return err
	}

	log.Printf("Artifact from %v to %v copied. %v Bytes transferred.", source, target, n)
	return nil
}

func (c *DockerDeployClient) prepareRemoteWorkdir() error {
	if c.RemoteWorkingDir == "" {
		return errors.New("No remote working directory specified.")
	}

	command := fmt.Sprintf("mkdir -p %s && cd %s && pwd && rm -rf *", c.RemoteWorkingDir, c.RemoteWorkingDir)
	_, err := c.executeCommand(command, false)
	if err != nil {
		return err
	}
	return nil
}

func (c *DockerDeployClient) checkDockerInstallation() error {
	output, err := c.executeCommand("which docker", false)
	if err != nil {
		return errors.New(fmt.Sprintf("Docker not installed. %v", output))
	}
	output, err = c.executeCommand("which docker-compose", false)
	if err != nil {
		return errors.New(fmt.Sprintf("Docker Compose not installed. %v", output))
	}
	if c.ComposeFile == "" {
		return errors.New("No compose file specified.")
	}
	if c.ProjectName == "" {
		return errors.New("No project name specified.")
	}
	return nil
}

func (c *DockerDeployClient) stopComposition() error {
	_, err := c.executeCommand(fmt.Sprintf("cd %v && sudo -S docker-compose -p %v -f %v stop", c.RemoteWorkingDir, c.ProjectName, c.ComposeFile), true)
	return err
}

func (c *DockerDeployClient) removeComposition() error {
	var err error = nil
	if c.ClearVolumes {
		_, err = c.executeCommand(fmt.Sprintf("cd %v && sudo -S docker-compose -p %v -f %v rm -v --force", c.RemoteWorkingDir, c.ProjectName, c.ComposeFile), true)
	} else {
		_, err = c.executeCommand(fmt.Sprintf("cd %v && sudo -S docker-compose -p %v -f %v rm --force", c.RemoteWorkingDir, c.ProjectName, c.ComposeFile), true)
	}
	return err
}

func (c *DockerDeployClient) buildComposition() error {
	_, err := c.executeCommand(fmt.Sprintf("cd %v && sudo -S docker-compose -p %v -f %v build", c.RemoteWorkingDir, c.ProjectName, c.ComposeFile), true)
	return err
}

func (c *DockerDeployClient) runComposition() error {
	_, err := c.executeCommand(fmt.Sprintf("cd %v && sudo -S docker-compose -p %v -f %v up -d", c.RemoteWorkingDir, c.ProjectName, c.ComposeFile), true)
	return err
}

func (c *DockerDeployClient) serviceDiscoveryTest() error {
	iterations := 0
	var response []byte
	ticks := time.Duration(c.StartTime/5) * time.Second

	for _ = range time.Tick(ticks) {
		iterations += 1
		if iterations >= 6 {
			break
		}

		res, err := http.Get(fmt.Sprintf("http://%v:%d/api/projectUp/%v", c.SSHHost, c.ServiceDiscoveryPort, c.ProjectName))
		if err != nil {
			log.Printf("#%d: Service Discovery is not reachable: %v", iterations, err.Error())
			continue
		}

		response, err = ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err == nil && string(response) == "true" {
			log.Printf("#%d: Composition was started: %d %v", iterations, res.StatusCode, string(response))
			return nil
		}

		log.Printf("#%d: Composition could not be started: %d %v", iterations, res.StatusCode, string(response))
	}

	return errors.New("Service Discovery test failed.")
}
