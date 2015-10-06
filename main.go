package main

import (
	"flag"
	"log"
	"strings"
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
	flag.StringVar(&c.RemoteWorkingDir, "remoteWorkingDir", "", "Absolute path of remote working directory, where the build artifact is copied and executed")
	flag.StringVar(&c.LocalWorkingDir, "localWorkingDir", "", "Absolute or relative path to local working directory, where the build artifact is stored")
	flag.StringVar(&c.LocalArtifact, "artifact", "", "Relative or absolute path to zip file that contains the composition")
	flag.IntVar(&c.ServiceDiscoveryPort, "serviceDiscoveryPort", 8080, "Port for access to the service discovery API")
	flag.BoolVar(&c.ClearVolumes, "clearVolumes", false, "Set true to clear container volumes in CLEAR and REMOVE commands")
	flag.Parse()

	var err error

	defer func() {
		if p := recover(); p != nil {
			if err = c.remoteCleanUp(); err != nil {
				log.Print(err)
			}
			c.disconnect()
			log.Fatalf("Deployment aborted: %v", p)
		}
	}()

	if err = c.findLocalArtifact(); err != nil {
		log.Fatal(err)
	}
	if err = c.connect(); err != nil {
		log.Fatal(err)
	}
	if err = c.prepareRemoteWorkdir(); err != nil {
		panic(err)
	}
	if err = c.copyArtifact(); err != nil {
		panic(err)
	}
	if err = c.unzipArtifact(); err != nil {
		panic(err)
	}
	if err = c.checkDockerInstallation(); err != nil {
		panic(err)
	}
	if strings.Contains(c.Mode, "CLEAR") {
		if err = c.stopComposition(); err != nil {
			panic(err)
		}
		if err = c.removeComposition(); err != nil {
			panic(err)
		}
	}
	if strings.Contains(c.Mode, "DEPLOY") {
		if err = c.buildComposition(); err != nil {
			panic(err)
		}
		if err = c.runComposition(); err != nil {
			panic(err)
		}
	}
	if strings.Contains(c.Mode, "TEST") {
		if err = c.serviceDiscoveryTest(); err != nil {
			panic(err)
		}
	}
	if strings.Contains(c.Mode, "REMOVE") {
		if err = c.stopComposition(); err != nil {
			panic(err)
		}
		if err = c.removeComposition(); err != nil {
			panic(err)
		}
	}
	if err = c.remoteCleanUp(); err != nil {
		log.Fatal(err)
	}
	c.disconnect()

}
