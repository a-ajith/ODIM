//(C) Copyright [2020] Hewlett Packard Enterprise Development LP
//
//Licensed under the Apache License, Version 2.0 (the "License"); you may
//not use this file except in compliance with the License. You may obtain
//a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
//WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
//License for the specific language governing permissions and limitations
// under the License.
package main

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/ODIM-Project/ODIM/lib-utilities/common"
	"github.com/ODIM-Project/ODIM/lib-utilities/config"
	accountproto "github.com/ODIM-Project/ODIM/lib-utilities/proto/account"
	authproto "github.com/ODIM-Project/ODIM/lib-utilities/proto/auth"
	sessiongrpcproto "github.com/ODIM-Project/ODIM/lib-utilities/proto/grpc/session"
	roleproto "github.com/ODIM-Project/ODIM/lib-utilities/proto/role"
	sessionproto "github.com/ODIM-Project/ODIM/lib-utilities/proto/session"
	"github.com/ODIM-Project/ODIM/lib-utilities/services"
	"github.com/ODIM-Project/ODIM/svc-account-session/rpc"
	"github.com/coreos/etcd/clientv3"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log = logrus.New()

func main() {
	// verifying the uid of the user
	if uid := os.Geteuid(); uid == 0 {
		log.Fatal("AccountSession Service should not be run as the root user")
	}

	if err := config.SetConfiguration(); err != nil {
		log.Fatal(err.Error())
	}

	if err := common.CheckDBConnection(); err != nil {
		log.Fatal("Error while trying to check DB connection health: " + err.Error())
	}

	configFilePath := os.Getenv("CONFIG_FILE_PATH")
	if configFilePath == "" {
		log.Fatal("error: no value get the environment variable CONFIG_FILE_PATH")
	}
	eventChan := make(chan interface{})
	// TrackConfigFileChanges monitors the odim config changes using fsnotfiy
	go common.TrackConfigFileChanges(configFilePath, eventChan)

	if err := services.InitializeService(services.AccountSession); err != nil {
		log.Fatal("Error while trying to initialize the service: " + err.Error())
	}
	go registerSession()
	registerHandlers()
	if err := services.Service.Run(); err != nil {
		log.Fatal("Failed to run a service: " + err.Error())
	}
}

func registerHandlers() {
	authproto.RegisterAuthorizationHandler(services.Service.Server(), new(rpc.Auth))
	sessionproto.RegisterSessionHandler(services.Service.Server(), new(rpc.Session))
	accountproto.RegisterAccountHandler(services.Service.Server(), new(rpc.Account))
	roleproto.RegisterRolesHandler(services.Service.Server(), new(rpc.Role))
}

func registerSession() {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"10.24.1.209:2379"},
		DialTimeout: 5 * time.Second,
	})
	kv := clientv3.NewKV(cli)
	_, err = kv.Put(context.TODO(), services.AccountSession, "10.24.1.209:8081")
	if err != nil {
		log.Fatal("While trying to register the service, got: " + err.Error())
		return
	}
	gs := grpc.NewServer()
	var session rpc.GRPCSession
	sessiongrpcproto.RegisterSessionServer(gs, &session)

	l, err := net.Listen("tcp", "10.24.1.209:8081")
	if err != nil {
		log.Fatal("While trying to get listen for the grpc, got: ", err.Error())
		return
	}

	gs.Serve(l)
}
