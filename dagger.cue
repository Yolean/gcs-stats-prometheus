package main

import (
	"strings"
	"dagger.io/dagger"
	"universe.dagger.io/docker"
)

// This action builds a docker image from a python app.
// Build steps are defined in an inline Dockerfile.
#Build: docker.#Dockerfile & {
}

// Example usage in a plan
dagger.#Plan & {
	client: {
		filesystem: "./": read: contents: dagger.#FS

		commands: {
			gitref: {
				name: "git"
				args: ["rev-parse", "HEAD"]
			}
		}
	}

	actions: {
		_gitref: {
			tag: strings.TrimSpace(client.commands.gitref.stdout)
		}
		build: #Build & {
			source: client.filesystem."./".read.contents
		}
		push: docker.#Push & {
			image: build.output
			dest:  "yolean/gcs-stats-prometheus:\(_gitref.tag)"
		}
	}
}
