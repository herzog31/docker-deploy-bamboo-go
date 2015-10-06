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
	flag.StringVar(&c.RemoteWorkingDir, "remoteWorkingDir", "", "Absolute path of remote working directory, where the build artifact is copied and executed")
	flag.StringVar(&c.LocalWorkingDir, "localWorkingDir", "", "Absolute or relative path to local working directory, where the build artifact is stored")
	flag.StringVar(&c.LocalArtifact, "artifact", "", "Relative or absolute path to zip file that contains the composition")
	flag.IntVar(&c.ServiceDiscoveryPort, "serviceDiscoveryPort", 8080, "Port for access to the service discovery API")
	flag.BoolVar(&c.ClearVolumes, "clearVolumes", false, "Set true to clear container volumes in CLEAR and REMOVE commands")
	flag.Parse()

	var err error

	if err = c.findLocalArtifact(); err != nil {
		log.Fatal(err)
	}
	log.Printf("Local artifact found: %v", c.LocalArtifact)
	if err = c.connect(); err != nil {
		log.Fatal(err)
	}
	defer c.disconnect()
	if err = c.prepareRemoteWorkdir(); err != nil {
		log.Fatal(err)
	}
	if err = c.copyArtifact(); err != nil {
		log.Fatal(err)
	}
	if err = c.unzipArtifact(); err != nil {
		log.Fatal(err)
	}
	if err = c.checkDockerInstallation(); err != nil {
		log.Fatal(err)
	}
	if err = c.stopComposition(); err != nil {
		log.Fatal(err)
	}
	if err = c.removeComposition(); err != nil {
		log.Fatal(err)
	}
	if err = c.buildComposition(); err != nil {
		log.Fatal(err)
	}
	if err = c.runComposition(); err != nil {
		log.Fatal(err)
	}

}

// TODO(mjb): Test, Service Discovery
// TODO(mjb): Remove working dir, cleanup, disconnect
// TODO(mjb): Travis Build
