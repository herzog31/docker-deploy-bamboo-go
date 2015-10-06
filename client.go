package main

import (
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	//"log"
	"os"
	"path"
)

type DockerDeployClient struct {
	Mode             string
	SSHPort          int
	SSHUser          string
	SSHHost          string
	SSHPassword      string
	ProjectName      string
	ComposeFile      string
	StartTime        int
	RemoteWorkingDir string
	LocalWorkingDir  string
	LocalArtifact    string
	config           *ssh.ClientConfig
	sshClient        *ssh.Client
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

func (c *DockerDeployClient) executeCommand(command string) (string, error) {
	session, err := c.sshClient.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run(command); err != nil {
		return "", err
	}
	return b.String(), nil
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
