package main

import (
	"flag"
	"log"
)

func main() {

	c := new(DockerDeployClient)

	flag.StringVar(&c.Mode, "mode", "DEPLOY,TEST", "Comma separated list of operations that should be executed. The following commands are available: CLEAR, DEPLOY, TEST, REMOVE.")
	flag.IntVar(&c.SSHPort, "sshPort", 22, "Port for SSH authentication")
	flag.StringVar(&c.SSHUser, "sshUser", "", "Username for SSH authentication")
	flag.StringVar(&c.SSHPassword, "sshPassword", "", "Password for SSH authentication")
	flag.StringVar(&c.SSHHost, "sshHost", "", "Hostname or IP of deployment target")
	flag.StringVar(&c.ProjectName, "projectName", "", "Project name of Docker composition")
	flag.StringVar(&c.ComposeFile, "composeFile", "docker-compose.yml", "Name of Docker Compose file")
	flag.IntVar(&c.StartTime, "startTime", 10, "Time in seconds the deployment scripts waits for the containers to start")
	flag.StringVar(&c.RemoteWorkingDir, "remoteWorkingDir", "", "Remote working directory, where the build artifact is copied and executed")
	flag.StringVar(&c.LocalWorkingDir, "localWorkingDir", "", "Local working directory, where the build artifact is stored")
	flag.StringVar(&c.LocalArtifact, "artifact", "", "Relative or absolute path to zip file that contains the composition")
	flag.Parse()

	var err error

	if err = c.findLocalArtifact(); err != nil {
		log.Fatal(err)
	}
	log.Printf("Local artifact found: %v", c.LocalArtifact)
	if err = c.connect(); err != nil {
		log.Fatal(err)
	}
	if err = c.disconnect(); err != nil {
		log.Fatal(err)
	}

	log.Print("Test...")

}

// TODO(mjb): Make sure folder exist, change folder, Clean working directory
// TODO(mjb): Copy, unzip, delete artifact
// TODO(mjb): Compose stop, clear
// TODO(mjb): Compose build, run
// TODO(mjb): Test, Service Discovery
// TODO(mjb): Remove working dir, cleanup, disconnect
// TODO(mjb): Travis Build
