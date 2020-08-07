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

package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/ODIM-Project/ODIM/lib-utilities/common"
	"github.com/ODIM-Project/ODIM/lib-utilities/config"
	aggregatorproto "github.com/ODIM-Project/ODIM/lib-utilities/proto/aggregator"
	"github.com/ODIM-Project/ODIM/lib-utilities/response"
	"github.com/ODIM-Project/ODIM/svc-aggregation/agresponse"
	"github.com/ODIM-Project/ODIM/svc-aggregation/system"
)

// GetAggregationService is an rpc handler, it gets invoked during GET on AggregationService API (/redfis/v1/AggregationService/)
func (a *Aggregator) GetAggregationService(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {
	// Fill the response header first
	resp.Header = map[string]string{
		"Allow":             "GET",
		"Cache-Control":     "no-cache",
		"Connection":        "Keep-alive",
		"Date":              time.Now().Format(http.TimeFormat),
		"Link":              "</redfish/v1/SchemaStore/en/AggregationService.json>; rel=describedby",
		"Transfer-Encoding": "chunked",
		"X-Frame-Options":   "sameorigin",
		"Content-type":      "application/json; charset=utf-8",
		"OData-Version":     "4.0",
	}
	// Validate the token, if user has Login privelege then proceed.
	//Else send 401 Unautherised
	var oemprivileges []string
	privileges := []string{common.PrivilegeLogin}
	authStatusCode, authStatusMessage := a.connector.Auth(req.SessionToken, privileges, oemprivileges)
	if authStatusCode != http.StatusOK {
		errMsg := "error while trying to authenticate session"
		generateResponse(common.GeneralError(authStatusCode, authStatusMessage, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}
	// Check whether the Aggregation Service is enbaled in configuration file.
	//If so set ServiceEnabled to true.
	isServiceEnabled := false
	serviceState := "Disabled"
	for _, service := range config.Data.EnabledServices {
		if service == "AggregationService" {
			isServiceEnabled = true
			serviceState = "Enabled"
			break
		}

	}
	// Construct the response below

	aggregationServiceResponse, _ := json.Marshal(agresponse.AggregationServiceResponse{
		OdataType:    "#AggregationService.v1_0_0.AggregationService",
		ID:           "AggregationService",
		Name:         "AggregationService",
		Description:  "AggregationService",
		OdataContext: "/redfish/v1/$metadata#AggregationService.AggregationService",
		OdataID:      "/redfish/v1/AggregationService",
		Actions: agresponse.Actions{
			Add: agresponse.Action{
				Target:     "/redfish/v1/AggregationService/Actions/AggregationService.Add/",
				ActionInfo: "/redfish/v1/AggregationService/AddActionInfo",
			},
			Remove: agresponse.Action{
				Target:     "/redfish/v1/AggregationService/Actions/AggregationService.Remove/",
				ActionInfo: "/redfish/v1/AggregationService/RemoveActionInfo",
			},
			Reset: agresponse.Action{
				Target:     "/redfish/v1/AggregationService/Actions/AggregationService.Reset/",
				ActionInfo: "/redfish/v1/AggregationService/ResetActionInfo",
			},
			SetDefaultBootOrder: agresponse.Action{
				Target:     "/redfish/v1/AggregationService/Actions/AggregationService.SetDefaultBootOrder/",
				ActionInfo: "/redfish/v1/AggregationService/SetDefaultBootOrderActionInfo",
			},
		},
		ServiceEnabled: isServiceEnabled,
		Status: agresponse.Status{
			State:        serviceState,
			HealthRollup: "OK",
			Health:       "OK",
		},
	})
	resp.StatusCode = http.StatusOK
	resp.StatusMessage = response.Success
	resp.Body = aggregationServiceResponse
	return nil
}

func isIpv4Net(host string) bool {
	return net.ParseIP(host) != nil
}

func validateManagerAddress(managerAddress string) error {
	// if the manager address is of the form <IP/FQDN>:<port>
	// will split address to obtain only IP/FQDN. If obtained
	// value is empty, then will use the actual address passed
	addr, _, _ := net.SplitHostPort(managerAddress)
	if addr == "" {
		addr = managerAddress
	}
	if _, err := net.ResolveIPAddr("ip", addr); err != nil {
		return fmt.Errorf("error: failed to resolve ManagerAddress: %v", err)
	}
	return nil
}

// AddCompute defines the operations which handles the RPC request response
// for the create account service of aggregator micro service.
// The functionality retrives the request and return backs the response to
// RPC according to the protoc file defined in the util-lib package.
// The function also checks for the session time out of the token
// which is present in the request.
func (a *Aggregator) AddCompute(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {
	var taskID string
	var oemprivileges []string
	privileges := []string{common.PrivilegeConfigureComponents}
	authStatusCode, authStatusMessage := a.connector.Auth(req.SessionToken, privileges, oemprivileges)
	if authStatusCode != http.StatusOK {
		errMsg := "error while trying to authenticate session"
		generateResponse(common.GeneralError(authStatusCode, authStatusMessage, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}
	sessionUserName, err := a.connector.GetSessionUserName(req.SessionToken)
	if err != nil {
		errMsg := "error while trying to get the session username: " + err.Error()
		generateResponse(common.GeneralError(http.StatusUnauthorized, response.NoValidSession, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}

	// parsing the AddResourceRequest
	var addResourceRequest system.AddResourceRequest
	err = json.Unmarshal(req.RequestBody, &addResourceRequest)
	if err != nil {
		errMsg := "unable to parse the add request" + err.Error()
		generateResponse(common.GeneralError(http.StatusInternalServerError, response.InternalError, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}

	//validating the AddResourceRequest
	invalidParam := validateAddResoureRequest(addResourceRequest)
	if invalidParam != "" {
		errMsg := "error: Mandatory field " + invalidParam + " Missing"
		generateResponse(common.GeneralError(http.StatusBadRequest, response.PropertyMissing, errMsg, []interface{}{invalidParam}, nil), resp)
		log.Printf(errMsg)
		return nil
	}
	managerAddress := addResourceRequest.ManagerAddress
	err = validateManagerAddress(managerAddress)
	if err != nil {
		generateResponse(common.GeneralError(http.StatusBadRequest, response.PropertyValueFormatError, err.Error(), []interface{}{managerAddress, "ManagerAddress"}, nil), resp)
		log.Printf(err.Error())
		return nil
	}

	// Task Service using RPC and get the taskID
	taskURI, err := a.connector.CreateTask(sessionUserName)
	if err != nil {
		errMsg := "error while trying to create the task: " + err.Error()
		generateResponse(common.GeneralError(http.StatusInternalServerError, response.InternalError, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}
	strArray := strings.Split(taskURI, "/")
	if strings.HasSuffix(taskURI, "/") {
		taskID = strArray[len(strArray)-2]
	} else {
		taskID = strArray[len(strArray)-1]
	}
	// spawn the thread here to process the action asynchronously
	go a.connector.AggregationServiceAdd(taskID, sessionUserName, req)

	// return 202 Accepted
	var rpcResp = response.RPC{
		StatusCode:    http.StatusAccepted,
		StatusMessage: response.TaskStarted,
		Header: map[string]string{
			"Content-type": "application/json; charset=utf-8",
			"Location":     "/taskmon/" + taskID,
		},
	}
	generateTaskRespone(taskID, taskURI, &rpcResp)
	generateResponse(rpcResp, resp)

	return nil
}

// DeleteCompute defines the operations which handles the RPC request response
// for the delete compute system of aggregator micro service.
// The functionality retrives the request and delete the compute system and return backs the response to
// RPC according to the protoc file defined in the util-lib package.
// The function also checks for the session time out of the token
// which is present in the request.
func (a *Aggregator) DeleteCompute(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {
	var oemprivileges []string
	// Task Service using RPC and get the taskID
	targetURI := "/redfish/v1/AggregationService/Actions/AggregationService.Delete"
	privileges := []string{common.PrivilegeConfigureComponents}
	authStatusCode, authStatusMessage := a.connector.Auth(req.SessionToken, privileges, oemprivileges)
	if authStatusCode != http.StatusOK {
		errMsg := "error while trying to authenticate session"
		generateResponse(common.GeneralError(authStatusCode, authStatusMessage, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}
	sessionUserName, err := a.connector.GetSessionUserName(req.SessionToken)
	if err != nil {
		errMsg := "error while trying to get the session username: " + err.Error()
		generateResponse(common.GeneralError(http.StatusUnauthorized, response.NoValidSession, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}

	// Task Service using RPC and get the taskID
	taskURI, err := a.connector.CreateTask(sessionUserName)
	if err != nil {
		errMsg := "error while trying to create the task: " + err.Error()
		generateResponse(common.GeneralError(http.StatusInternalServerError, response.InternalError, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}
	var taskID string
	strArray := strings.Split(taskURI, "/")
	if strings.HasSuffix(taskURI, "/") {
		taskID = strArray[len(strArray)-2]
	} else {
		taskID = strArray[len(strArray)-1]
	}
	go deleteServer(taskID, targetURI, a, req)
	// return 202 Accepted
	var rpcResp = response.RPC{
		StatusCode:    http.StatusAccepted,
		StatusMessage: response.TaskStarted,
		Header: map[string]string{
			"Content-type": "application/json; charset=utf-8",
			"Location":     "/taskmon/" + taskID,
		},
	}
	generateTaskRespone(taskID, taskURI, &rpcResp)
	generateResponse(rpcResp, resp)

	return nil

}

func deleteServer(taskID string, targetURI string, a *Aggregator, req *aggregatorproto.AggregatorRequest) error {
	err := a.connector.UpdateTask(common.TaskData{
		TaskID:          taskID,
		TargetURI:       targetURI,
		TaskState:       common.Running,
		TaskStatus:      common.OK,
		PercentComplete: 0,
		HTTPMethod:      http.MethodPost,
	})
	if err != nil && (err.Error() == common.Cancelling) {
		// We cant do anything here as the task has done it work completely, we cant reverse it.
		//Unless if we can do opposite/reverse action for delete server which is add server.
		a.connector.UpdateTask(common.TaskData{
			TaskID:          taskID,
			TargetURI:       targetURI,
			TaskState:       common.Cancelled,
			TaskStatus:      common.OK,
			PercentComplete: 0,
			HTTPMethod:      http.MethodPost,
		})
		go runtime.Goexit()
	}
	data := a.connector.DeleteCompute(req)
	err = a.connector.UpdateTask(common.TaskData{
		TaskID:          taskID,
		TargetURI:       targetURI,
		TaskState:       common.Completed,
		TaskStatus:      common.OK,
		Response:        data,
		PercentComplete: 100,
		HTTPMethod:      http.MethodPost,
	})
	if err != nil && (err.Error() == common.Cancelling) {
		a.connector.UpdateTask(common.TaskData{
			TaskID:          taskID,
			TargetURI:       targetURI,
			TaskState:       common.Cancelled,
			TaskStatus:      common.OK,
			PercentComplete: 100,
			HTTPMethod:      http.MethodPost,
		})
		go runtime.Goexit()
	}
	return nil
}

// Reset function is for handling the RPC communication for Reset Action
func (a *Aggregator) Reset(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {

	// Verfy the credentials here
	var oemprivileges []string

	privileges := []string{common.PrivilegeConfigureComponents}
	authStatusCode, authStatusMessage := a.connector.Auth(req.SessionToken, privileges, oemprivileges)
	if authStatusCode != http.StatusOK {
		errMsg := "error while trying to authenticate session"
		generateResponse(common.GeneralError(authStatusCode, authStatusMessage, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}
	sessionUserName, err := a.connector.GetSessionUserName(req.SessionToken)
	if err != nil {
		errMsg := "error while trying to get the session username: " + err.Error()
		generateResponse(common.GeneralError(http.StatusUnauthorized, response.NoValidSession, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}

	// Task Service using RPC and get the taskID
	taskURI, err := a.connector.CreateTask(sessionUserName)
	if err != nil {
		errMsg := "error while trying to create task: " + err.Error()
		generateResponse(common.GeneralError(http.StatusInternalServerError, response.InternalError, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}
	taskID := strings.TrimPrefix(taskURI, "/redfish/v1/TaskService/Tasks/")
	go a.reset(ctx, taskID, sessionUserName, req)
	// return 202 Accepted
	var rpcResp = response.RPC{
		StatusCode:    http.StatusAccepted,
		StatusMessage: response.TaskStarted,
		Header: map[string]string{
			"Content-type": "application/json; charset=utf-8",
			"Location":     "/taskmon/" + taskID,
		},
	}
	generateTaskRespone(taskID, taskURI, &rpcResp)
	generateResponse(rpcResp, resp)
	return nil

}
func (a *Aggregator) reset(ctx context.Context, taskID string, sessionUserName string, req *aggregatorproto.AggregatorRequest) error {
	// Update the task status here
	// PercentComplete: 0% Completed
	// TaskState: Running - This value shall represent that the operation is executing.
	err := a.connector.UpdateTask(common.TaskData{
		TaskID:          taskID,
		TaskState:       common.Running,
		TaskStatus:      common.OK,
		PercentComplete: 0,
		HTTPMethod:      http.MethodPost,
	})
	if err != nil && (err.Error() == common.Cancelling) {
		// We cant do anything here as the task has done it work completely, we cant reverse it.
		//Unless if we can do opposite/reverse action for delete server which is add server.
		a.connector.UpdateTask(common.TaskData{
			TaskID:          taskID,
			TaskState:       common.Cancelled,
			TaskStatus:      common.OK,
			PercentComplete: 0,
			HTTPMethod:      http.MethodPost,
		})
	}

	a.connector.Reset(taskID, sessionUserName, req)
	return nil
}

// SetDefaultBootOrder defines the operations which handles the RPC request response
// for the create account service of aggregator micro service.
// The functionality retrives the request and return backs the response to
// RPC according to the protoc file defined in the lib-utilities package.
// The function also checks for the session time out of the token
// which is present in the request.
func (a *Aggregator) SetDefaultBootOrder(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {
	var oemprivileges []string

	privileges := []string{common.PrivilegeConfigureComponents}
	authStatusCode, authStatusMessage := a.connector.Auth(req.SessionToken, privileges, oemprivileges)
	if authStatusCode != http.StatusOK {
		errMsg := "error while trying to authenticate session"
		generateResponse(common.GeneralError(authStatusCode, authStatusMessage, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}
	sessionUserName, err := a.connector.GetSessionUserName(req.SessionToken)
	if err != nil {
		errMsg := "error while trying to get the session username: " + err.Error()
		generateResponse(common.GeneralError(http.StatusUnauthorized, response.NoValidSession, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}
	taskURI, err := a.connector.CreateTask(sessionUserName)
	if err != nil {
		errMsg := "error while trying to create task: " + err.Error()
		generateResponse(common.GeneralError(http.StatusInternalServerError, response.InternalError, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}
	strArray := strings.Split(taskURI, "/")
	var taskID string
	if strings.HasSuffix(taskURI, "/") {
		taskID = strArray[len(strArray)-2]
	} else {
		taskID = strArray[len(strArray)-1]
	}
	err = a.connector.UpdateTask(common.TaskData{
		TaskID:          taskID,
		TargetURI:       taskURI,
		TaskState:       common.Running,
		TaskStatus:      common.OK,
		PercentComplete: 0,
		HTTPMethod:      http.MethodPost,
	})
	if err != nil {
		// print error as we are unable to communicate with svc-task and then return
		log.Printf("error while contacting task-service with UpdateTask RPC : %v", err)
	}
	go a.connector.SetDefaultBootOrder(taskID, sessionUserName, req)
	// return 202 Accepted
	var rpcResp = response.RPC{
		StatusCode:    http.StatusAccepted,
		StatusMessage: response.TaskStarted,
		Header: map[string]string{
			"Content-type": "application/json; charset=utf-8",
			"Location":     "/taskmon/" + taskID,
		},
	}
	generateTaskRespone(taskID, taskURI, &rpcResp)
	generateResponse(rpcResp, resp)

	return nil
}

// RediscoverSystemInventory defines the operations which handles the RPC request response
// for the RediscoverSystemInventory service of aggregator micro service.
// The functionality retrives the request and return backs the response to
// RPC according to the protoc file defined in the lib-utilities package.
func (a *Aggregator) RediscoverSystemInventory(ctx context.Context, req *aggregatorproto.RediscoverSystemInventoryRequest, resp *aggregatorproto.RediscoverSystemInventoryResponse) error {
	go a.connector.RediscoverSystemInventory(req.SystemID, req.SystemURL, true)
	return nil

}

// validateAddResoureRequest validates the request parameters of Add Resource request
func validateAddResoureRequest(req system.AddResourceRequest) string {
	param := ""
	if req.ManagerAddress == "" {
		param = "ManagerAddress "
	}
	if req.Password == "" {
		param = param + "Password "
	}
	if req.UserName == "" {
		param = param + "UserName "
	}
	if req.Oem != nil {
		if req.Oem.PluginID == "" {
			param = param + "PluginID"
		}
	} else {
		param = param + "Oem"
	}
	return param
}

// UpdateSystemState defines the operations which handles the RPC request response
// for the UpdateSystemState call to aggregator micro service.
// The functionality retrives the request and return backs the response to
// RPC according to the protoc file defined in the lib-utilities package.
func (a *Aggregator) UpdateSystemState(ctx context.Context, req *aggregatorproto.UpdateSystemStateRequest, resp *aggregatorproto.UpdateSystemStateResponse) error {
	return a.connector.UpdateSystemState(req)
}

// AddAggregationSource function is for handling the RPC communication for AddAggregationSource
func (a *Aggregator) AddAggregationSource(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {

	var taskID string
	var oemprivileges []string
	privileges := []string{common.PrivilegeConfigureComponents}
	authStatusCode, authStatusMessage := a.connector.Auth(req.SessionToken, privileges, oemprivileges)
	if authStatusCode != http.StatusOK {
		errMsg := "error while trying to authenticate session"
		generateResponse(common.GeneralError(authStatusCode, authStatusMessage, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}
	sessionUserName, err := a.connector.GetSessionUserName(req.SessionToken)
	if err != nil {
		errMsg := "error while trying to get the session username: " + err.Error()
		generateResponse(common.GeneralError(http.StatusUnauthorized, response.NoValidSession, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}

	// parsing the AggregationSourceRequest
	var addRequest system.AggregationSource
	err = json.Unmarshal(req.RequestBody, &addRequest)
	if err != nil {
		errMsg := "unable to parse the add request" + err.Error()
		generateResponse(common.GeneralError(http.StatusInternalServerError, response.InternalError, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}

	//validating the AggregationSourceRequest
	invalidParam := validateAggregationSourceRequest(addRequest)
	if invalidParam != "" {
		errMsg := "error: Mandatory field " + invalidParam + " Missing"
		generateResponse(common.GeneralError(http.StatusBadRequest, response.PropertyMissing, errMsg, []interface{}{invalidParam}, nil), resp)
		log.Printf(errMsg)
		return nil
	}
	managerAddress := addRequest.HostName
	err = validateManagerAddress(managerAddress)
	if err != nil {
		generateResponse(common.GeneralError(http.StatusBadRequest, response.PropertyValueFormatError, err.Error(), []interface{}{managerAddress, "ManagerAddress"}, nil), resp)
		log.Printf(err.Error())
		return nil
	}

	// Task Service using RPC and get the taskID
	taskURI, err := a.connector.CreateTask(sessionUserName)
	if err != nil {
		errMsg := "error while trying to create the task: " + err.Error()
		generateResponse(common.GeneralError(http.StatusInternalServerError, response.InternalError, errMsg, nil, nil), resp)
		log.Printf(errMsg)
		return nil
	}
	strArray := strings.Split(taskURI, "/")
	if strings.HasSuffix(taskURI, "/") {
		taskID = strArray[len(strArray)-2]
	} else {
		taskID = strArray[len(strArray)-1]
	}
	// spawn the thread here to process the action asynchronously
	go a.connector.AddAggregationSource(taskID, sessionUserName, req)

	// return 202 Accepted
	var rpcResp = response.RPC{
		StatusCode:    http.StatusAccepted,
		StatusMessage: response.TaskStarted,
		Header: map[string]string{
			"Content-type": "application/json; charset=utf-8",
			"Location":     "/taskmon/" + taskID,
		},
	}
	generateTaskRespone(taskID, taskURI, &rpcResp)
	generateResponse(rpcResp, resp)
	return nil
}

func validateAggregationSourceRequest(req system.AggregationSource) string {
	param := ""
	if req.HostName == "" {
		param = "HostName "
	}
	if req.Password == "" {
		param = param + "Password "
	}
	if req.UserName == "" {
		param = param + "UserName "
	}
	return param + validateLinks(req.Links)
}

func validateLinks(req *system.Links) string {
	var param = ""
	if req != nil {
		if req.Oem != nil {
			if req.Oem.PluginID == "" {
				param = param + "PluginID"
			}
		} else {
			param = param + "Oem"
		}
	} else {
		param = "Links"
	}
	return param
}

// GetAllAggregationSource defines the operations which handles the RPC request response
// for the GetAllAggregationSource  service of aggregation micro service.
// The functionality retrives the request and return backs the response to
// RPC according to the protoc file defined in the util-lib package.
// The function also checks for the session time out of the token
// which is present in the request.
func (a *Aggregator) GetAllAggregationSource(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {
	return nil
}

// GetAggregationSource defines the operations which handles the RPC request response
// for the GetAggregationSource  service of aggregation micro service.
// The functionality retrives the request and return backs the response to
// RPC according to the protoc file defined in the util-lib package.
// The function also checks for the session time out of the token
// which is present in the request.
func (a *Aggregator) GetAggregationSource(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {
	return nil
}

// UpdateAggregationSource defines the operations which handles the RPC request response
// for the UpdateAggregationSource  service of aggregation micro service.
// The functionality retrives the request and return backs the response to
// RPC according to the protoc file defined in the util-lib package.
// The function also checks for the session time out of the token
// which is present in the request.
func (a *Aggregator) UpdateAggregationSource(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {
	return nil
}

// DeleteAggregationSource defines the operations which handles the RPC request response
// for the UpdateAggregationSource  service of aggregation micro service.
// The functionality retrives the request and return backs the response to
// RPC according to the protoc file defined in the util-lib package.
// The function also checks for the session time out of the token
// which is present in the request.
func (a *Aggregator) DeleteAggregationSource(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {
	return nil
}


// CreateAggregate defines the operations which handles the RPC request response
// for the CreateAggregate  service of aggregation micro service.
// The functionality retrives the request and return backs the response to
// RPC according to the protoc file defined in the util-lib package.
// The function also checks for the session time out of the token
// which is present in the request.
func (a *Aggregator) CreateAggregate(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {
	return nil
}

// GetAllAggregates defines the operations which handles the RPC request response
// for the GetAllAggregates service of aggregation micro service.
// The functionality retrives the request and return backs the response to
// RPC according to the protoc file defined in the util-lib package.
// The function also checks for the session time out of the token
// which is present in the request.
func (a *Aggregator) GetAllAggregates(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {
	return nil
}

// GetAggregate defines the operations which handles the RPC request response
// for the GetAggregate service of aggregation micro service.
// The functionality retrives the request and return backs the response to
// RPC according to the protoc file defined in the util-lib package.
// The function also checks for the session time out of the token
// which is present in the request.
func (a *Aggregator) GetAggregate(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {
	return nil
}

// DeleteAggregate defines the operations which handles the RPC request response
// for the DeleteAggregate service of aggregation micro service.
// The functionality retrives the request and return backs the response to
// RPC according to the protoc file defined in the util-lib package.
// The function also checks for the session time out of the token
// which is present in the request.
func (a *Aggregator) DeleteAggregate(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {
	return nil
}

// AddElementsToAggregate defines the operations which handles the RPC request response
// for the AddElementsToAggregate service of aggregation micro service.
// The functionality retrives the request and return backs the response to
// RPC according to the protoc file defined in the util-lib package.
// The function also checks for the session time out of the token
// which is present in the request.
func (a *Aggregator) AddElementsToAggregate(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {
	return nil
}

// RemoveElementsFromAggregate defines the operations which handles the RPC request response
// for the RemoveElementsFromAggregate service of aggregation micro service.
// The functionality retrives the request and return backs the response to
// RPC according to the protoc file defined in the util-lib package.
// The function also checks for the session time out of the token
// which is present in the request.
func (a *Aggregator) RemoveElementsFromAggregate(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {
	return nil
}

// ResetElementsOfAggregate defines the operations which handles the RPC request response
// for the ResetElementsOfAggregate service of aggregation micro service.
// The functionality retrives the request and return backs the response to
// RPC according to the protoc file defined in the util-lib package.
// The function also checks for the session time out of the token
// which is present in the request.
func (a *Aggregator) ResetElementsOfAggregate(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {
	return nil
}

// SetDefaultBootOrderElementsOfAggregate defines the operations which handles the RPC request response
// for the SetDefaultBootOrderElementsOfAggregate service of aggregation micro service.
// The functionality retrives the request and return backs the response to
// RPC according to the protoc file defined in the util-lib package.
// The function also checks for the session time out of the token
// which is present in the request.
func (a *Aggregator) SetDefaultBootOrderElementsOfAggregate(ctx context.Context, req *aggregatorproto.AggregatorRequest, resp *aggregatorproto.AggregatorResponse) error {
	return nil
}
