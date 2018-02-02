// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"fmt"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	mgo "gopkg.in/mgo.v2"

	"github.com/diegobernardes/flare"
	mgo2 "github.com/diegobernardes/flare/provider/mongodb"
	mongodb "github.com/diegobernardes/flare/provider/mongodb/repository"
)

const (
	dockerImage = "mongo"
	dockerTag   = "3.5.13-jessie"
	dockerPort  = "27017"
)

func initMongoDBContainer() (func(), string, error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, "", err
	}

	if err = ensureDockerImage(client); err != nil {
		return nil, "", err
	}

	c, err := client.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image:        fmt.Sprintf("%s:%s", dockerImage, dockerTag),
			ExposedPorts: map[docker.Port]struct{}{dockerPort: {}},
		},
	})
	if err != nil {
		return nil, "", err
	}

	err = client.StartContainer(c.ID, &docker.HostConfig{})
	if err != nil {
		return nil, "", err
	}

	cc, err := client.InspectContainer(c.ID)
	if err != nil {
		return nil, "", err
	}

	addr := fmt.Sprintf("%s:%s", cc.NetworkSettings.IPAddress, dockerPort)
	for {
		var session *mgo.Session
		session, err = mgo.Dial(addr)
		if err != nil {
			return nil, "", err
		}

		if err = session.Ping(); err == nil {
			break
		}
		<-time.After(1 * time.Second)
	}

	return func() {
		err = client.RemoveContainer(docker.RemoveContainerOptions{ID: c.ID, Force: true})
		if err != nil {
			panic(err)
		}
	}, addr, nil
}

func ensureDockerImage(client *docker.Client) error {
	imgs, err := client.ListImages(docker.ListImagesOptions{})
	if err != nil {
		return err
	}

	var hasImage bool
	for _, img := range imgs {
		if hasDockerImage(img.RepoTags, dockerImage, dockerTag) {
			hasImage = true
			break
		}
	}

	if !hasImage {
		err = client.PullImage(docker.PullImageOptions{
			Repository: dockerImage,
			Tag:        dockerTag,
		}, docker.AuthConfiguration{})
		if err != nil {
			return err
		}
	}

	return nil
}

func hasDockerImage(images []string, name, tag string) bool {
	key := fmt.Sprintf("%s:%s", name, tag)
	for _, image := range images {
		if image == key {
			return true
		}
	}
	return false
}

func initMongoDBResourceRepositoriy(ip string) (flare.ResourceRepositorier, func(), error) {
	mongodbClient, err := mgo2.NewClient(mgo2.ClientAddrs([]string{ip}))
	if err != nil {
		return nil, nil, err
	}

	client, err := mongodb.NewClient(
		mongodb.ClientConnection(mongodbClient),
		mongodb.ClientResourceOptions(mongodb.ResourcePartitionLimit(100)),
	)
	if err != nil {
		panic(err)
	}

	return client.Resource(), mongodbClient.Stop, nil
}
