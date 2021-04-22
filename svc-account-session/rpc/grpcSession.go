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

// Package rpc ...
package rpc

import (
	"context"
	"encoding/json"

	sessiongrpcproto "github.com/ODIM-Project/ODIM/lib-utilities/proto/grpc/session"
	"github.com/ODIM-Project/ODIM/svc-account-session/session"

	"log"
	"net/http"
)

// GRPCSession struct helps to register service
type GRPCSession struct{}

// GetAllActiveSessions is a rpc call to get all active sessions
// This method will accepts the sessionrequest which has session id and session token
// and it will call GetAllActiveSessions from the session package
// and respond all the sessionresponse values along with error if there is.
func (s *GRPCSession) GetAllActiveSessions(ctx context.Context, req *sessiongrpcproto.GRPCRequest) (resp *sessiongrpcproto.GRPCResponse, err error) {

	response := session.GetAllActiveSessions(req)
	body, err := json.Marshal(response.Body)
	if err != nil {
		resp.StatusCode = http.StatusInternalServerError
		resp.StatusMessage = "error while trying marshal the response body for get all active session: " + err.Error()
		log.Printf(response.StatusMessage)
		return resp, nil
	}
	resp.StatusCode = response.StatusCode
	resp.StatusMessage = response.StatusMessage
	resp.Header = response.Header
	resp.Body = body
	return resp, nil

}
