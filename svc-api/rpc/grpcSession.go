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

//Package rpc ...
package rpc

import (
	"context"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"
	"crypto/tls"
	"github.com/ODIM-Project/ODIM/lib-utilities/config"
	sessiongrpcproto "github.com/ODIM-Project/ODIM/lib-utilities/proto/grpc/session"
	"github.com/ODIM-Project/ODIM/lib-utilities/services"
	"github.com/coreos/etcd/clientv3"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GetAllActiveSessionRequest will do the rpc call to get session
func GetAllActiveSessionRequest(sessionID, sessionToken string) (*sessiongrpcproto.GRPCResponse, error) {
	log.Info("GRPC connection initiated")
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"etcd:2379"},
		DialTimeout: 5 * time.Second,
	})
	kv := clientv3.NewKV(cli)
	resp, err := kv.Get(context.TODO(), services.AccountSession, clientv3.WithPrefix())
	if err != nil {
		log.Fatal("While trying to get the service from registry, got: " + err.Error())
		return nil, nil
	}
	log.Info("The value from etcd: ", resp)

	// tlsCredentials, err := loadTLSCredentials()
    // if err != nil {
	// 	log.Fatal("cannot load TLS credentials: "+ err.Error())
	// 	return nil, nil
    // }

	conn, err := grpc.Dial(
		string(resp.Kvs[0].Value),
		grpc.WithInsecure(),
		// grpc.WithTransportCredentials(tlsCredentials),
	)
	if err != nil {
		log.Error("While connecting with the GRPC, got: " + err.Error())
	}
	defer conn.Close()
	sc := sessiongrpcproto.NewSessionClient(conn)
	rsp, err := sc.GetAllActiveSessions(
		context.Background(),
		&sessiongrpcproto.GRPCRequest{
			SessionId:    sessionID,
			SessionToken: sessionToken,
		},
	)
	
	if err != nil && rsp == nil {
		return nil, fmt.Errorf("error while trying to make get session service rpc call: %v", err)
	}

	return rsp, err
}


func loadTLSCredentials() (credentials.TransportCredentials, error) {
    // Load certificate of the CA who signed server's certificate
    pemServerCA, err := ioutil.ReadFile(config.Data.KeyCertConf.RootCACertificatePath)
    if err != nil {
        return nil, err
    }

    certPool := x509.NewCertPool()
    if !certPool.AppendCertsFromPEM(pemServerCA) {
        return nil, fmt.Errorf("failed to add server CA's certificate")
    }

    // Create the credentials and return it
    config := &tls.Config{
        RootCAs:      certPool,
    }

    return credentials.NewTLS(config), nil
}
