package main

import (
	"dagger.io/dagger/op"
	"dagger.io/alpine"
)

TestAuth: op.#RegistryAuth

TestPushContainer: {
	// Generate a random number
	random: {
		string
		#up: [
			op.#Load & {from: alpine.#Image},
			op.#Exec & {
				args: ["sh", "-c", "echo -n $RANDOM > /rand"]
				always: true
			},
			op.#Export & {
				source: "/rand"
			},
		]
	}

	// Push an image with a random tag
	push: {
		ref: "daggerio/ci-test:\(random)"
		#up: [
			op.#WriteFile & {
				content: random
				dest:    "/rand"
			},
			op.#PushContainer & {
				"ref": ref
				auth:  TestAuth
			},
		]
	}

	// Pull the image back
	pull: #up: [
		op.#FetchContainer & {
			ref: push.ref
		},
	]

	// Check the content
	check: #up: [
		op.#Load & {from: alpine.#Image},
		op.#Exec & {
			args: [
				"sh", "-c", #"""
                test "$(cat /src/rand)" = "\#(random)"
                """#,
			]
			mount: "/src": from: pull
		},
	]
}

// Ensures image metadata is preserved in a push
TestPushContainerMetadata: {
	// Generate a random number
	random: {
		string
		#up: [
			op.#Load & {from: alpine.#Image},
			op.#Exec & {
				args: ["sh", "-c", "echo -n $RANDOM > /rand"]
				always: true
			},
			op.#Export & {
				source: "/rand"
			},
		]
	}

	// `docker build` using an `ENV` and push the image
	push: {
		ref: "daggerio/ci-test:\(random)-dockerbuild"
		#up: [
			op.#DockerBuild & {
				dockerfile: #"""
					FROM alpine:latest@sha256:ab00606a42621fb68f2ed6ad3c88be54397f981a7b70a79db3d1172b11c4367d
					ENV CHECK \#(random)
					"""#
			},
			op.#PushContainer & {
				"ref": ref
				auth:  TestAuth
			},
		]
	}

	// Pull the image down and make sure the ENV is preserved
	check: #up: [
		op.#FetchContainer & {
			ref: push.ref
		},
		op.#Exec & {
			args: [
				"sh", "-c", #"""
                env
                test "$CHECK" = "\#(random)"
                """#,
			]
		},
	]

	// Do a FetchContainer followed by a PushContainer, make sure
	// the ENV is preserved
	pullPush: {
		ref: "daggerio/ci-test:\(random)-pullpush"

		#up: [
			op.#FetchContainer & {
				ref: push.ref
			},
			op.#PushContainer & {
				"ref": ref
				auth:  TestAuth
			},
		]
	}

	pullPushCheck: #up: [
		op.#FetchContainer & {
			ref: pullPush.ref
		},
		op.#Exec & {
			args: [
				"sh", "-c", #"""
                test "$CHECK" = "\#(random)"
                """#,
			]
		},
	]
}