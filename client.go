package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"path"
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

func (c *DockerDeployClient) unzipArtifact() error {
	output, err := c.executeCommand("which unzip")
	if err != nil {
		return errors.New(fmt.Sprintf("Unzip not installed. %v", output))
	}

	output, err = c.executeCommand(fmt.Sprintf("cd %v && unzip -o %v && rm %v", c.RemoteWorkingDir, path.Base(c.LocalArtifact), path.Base(c.LocalArtifact)))
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
	command := fmt.Sprintf("mkdir -p %s && cd %s && pwd && rm -rf *", c.RemoteWorkingDir, c.RemoteWorkingDir)
	_, err := c.executeCommand(command)
	if err != nil {
		return err
	}
	return nil
}
