package main

import (
	"strings"
	"dagger.io/dagger"
	"universe.dagger.io/docker"
	"universe.dagger.io/docker/cli"
)

// This action builds a docker image from a python app.
// Build steps are defined in an inline Dockerfile.
#Build: docker.#Dockerfile & {
}

// Example usage in a plan
dagger.#Plan & {
	client: {
		filesystem: "./": read: contents: dagger.#FS

		network: "unix:///var/run/docker.sock": connect: dagger.#Socket

		commands: {
			gitref: {
				name: "git"
				args: ["rev-parse", "HEAD"]
			}
		}
	}

	actions: {
		_image: "yolean/gcs-stats-prometheus"
		_gitref: {
			tag: strings.TrimSpace(client.commands.gitref.stdout)
		}
		build: #Build & {
			source: client.filesystem."./".read.contents
		}
		push: docker.#Push & {
			image: build.output
			dest:  "\(_image):\(_gitref.tag)"
		}
		load: cli.#Load & {
			image: build.output
			host:  client.network."unix:///var/run/docker.sock".connect
			tag:   "\(_image):\(_gitref.tag)"
		}
	}
}
