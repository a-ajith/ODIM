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

//Package dphandler ...
package dphandler

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ODIM-Project/ODIM/lib-utilities/common"
	"github.com/ODIM-Project/ODIM/lib-utilities/response"
	pluginConfig "github.com/ODIM-Project/ODIM/plugin-dell/config"
	"github.com/ODIM-Project/ODIM/plugin-dell/dpmodel"
	"github.com/ODIM-Project/ODIM/plugin-dell/dpresponse"
	"github.com/ODIM-Project/ODIM/plugin-dell/dputilities"
	iris "github.com/kataras/iris/v12"
)

//CreateVolume function is used for creating a volume under storage
func CreateVolume(ctx iris.Context) {
	//Get token from Request
	token := ctx.GetHeader("X-Auth-Token")
	uri := ctx.Request().RequestURI
	//replacing the request url with south bound translation URL
	for key, value := range pluginConfig.Data.URLTranslation.SouthBoundURL {
		uri = strings.Replace(uri, key, value, -1)
	}

	//Validating the token
	if token != "" {
		flag := TokenValidation(token)
		if !flag {
			log.Println("Invalid/Expired X-Auth-Token")
			ctx.StatusCode(http.StatusUnauthorized)
			ctx.WriteString("Invalid/Expired X-Auth-Token")
			return
		}
	}

	var deviceDetails dpmodel.Device
	//Get device details from request
	err := ctx.ReadJSON(&deviceDetails)
	if err != nil {
		log.Println("Error while trying to collect data from request: ", err)
		ctx.StatusCode(http.StatusBadRequest)
		ctx.WriteString("Error: bad request.")
		return
	}

	// Transforming the payload
	reqPostBody := string(deviceDetails.PostBody)
	var reqBody dpmodel.Volume
	err = json.Unmarshal(deviceDetails.PostBody, &reqBody)
	if err != nil {
		errMsg := "error while unmarshalling the create volume request to the device: " + err.Error()
		log.Println(errMsg)
		ctx.StatusCode(http.StatusInternalServerError)
		ctx.WriteString(errMsg)
		return
	}

	systemID := ctx.Params().Get("id")
	storageInstance := ctx.Params().Get("id2")

	driveURI := reqBody.Drives[0].OdataID
	s := strings.Split(driveURI, "/")
	driveSystemID := s[4]
	reqPostBody = strings.Replace(reqPostBody, driveSystemID, systemID, -1)
	reqPostBody = convertToSouthBoundURI(reqPostBody, storageInstance)
	device := &dputilities.RedfishDevice{
		Host:     deviceDetails.Host,
		Username: deviceDetails.Username,
		Password: string(deviceDetails.Password),
		PostBody: []byte(reqPostBody),
	}

	// Getting the list of volumes before creating a new volume
	volStatusCode, volErrMsg, list1 := getVolumeCollection(uri, device)
	if volStatusCode != http.StatusOK {
		ctx.StatusCode(volStatusCode)
		ctx.WriteString(volErrMsg)
		return
	}

	// calling device for creating a volume
	statusCode, header, resp, err := queryDevice(uri, device, http.MethodPost)
	if err != nil {
		errMsg := "error while trying to create volume: " + err.Error()
		log.Println(errMsg)
		ctx.StatusCode(statusCode)
		ctx.WriteString(errMsg)
	}

	// If a create volume response contains any Location header then looping it to get final response
	if header.Get("Location") != "" {
		taskURI := header.Get("Location")
		//tracking the task id until reaches final state
		for {
			time.Sleep(10 * time.Second)
			// calling device for creating a volume
			statusCode, header, resp, err = queryDevice(taskURI, device, http.MethodGet)
			if err != nil {
				errorMessage := "Error while trying to get task id in create volume: " + err.Error()
				log.Println(errorMessage)
				ctx.StatusCode(statusCode)
				ctx.WriteString(errorMessage)
				return
			}
			if statusCode != http.StatusAccepted {
				log.Println("Final Status of task id while creating a volume : ", statusCode)
				break
			}
		}
	}
	// If volume addition is success then generating an event
	if statusCode == http.StatusOK {
		// Getting the list of volumes after creating a new volume
		volStatusCode, volErrMsg, list2 := getVolumeCollection(uri, device)
		if volStatusCode != http.StatusOK {
			ctx.StatusCode(volStatusCode)
			ctx.WriteString(volErrMsg)
			return
		}
		// Getting the origin of condition for event
		oriOfCondition := compareCollection(list1, list2)
		// creating a event payload
		event := common.MessageData{
			OdataType: "#Event.v1_2_1.Event",
			Name:      "Volume created Event",
			Context:   "/redfish/v1/$metadata#Event.Event",
			Events: []common.Event{
				common.Event{
					EventType:      "ResourceAdded",
					EventID:        "123",
					Severity:       "Critical",
					EventTimestamp: time.Now().String(),
					Message:        "Volume is created successfully",
					MessageID:      "ResourceEvent.1.0.3.ResourceCreated",
					OriginOfCondition: &common.Link{
						Oid: oriOfCondition,
					},
				},
			},
		}
		manualEvents(event, deviceDetails.Host)
		resp = createResponse()
	}
	log.Println("Response body: ", string(resp))
	ctx.StatusCode(statusCode)
	ctx.Write(resp)
}

// DeleteVolume function is used for deleting a volume under storage
func DeleteVolume(ctx iris.Context) {
	//Get token from Request
	token := ctx.GetHeader("X-Auth-Token")
	uri := ctx.Request().RequestURI
	//replacing the request url with south bound translation URL
	for key, value := range pluginConfig.Data.URLTranslation.SouthBoundURL {
		uri = strings.Replace(uri, key, value, -1)
	}

	storageInstance := ctx.Params().Get("id2")
	uri = convertToSouthBoundURI(uri, storageInstance)

	//Validating the token
	if token != "" {
		flag := TokenValidation(token)
		if !flag {
			log.Println("Invalid/Expired X-Auth-Token")
			ctx.StatusCode(http.StatusUnauthorized)
			ctx.WriteString("Invalid/Expired X-Auth-Token")
			return
		}
	}

	var deviceDetails dpmodel.Device

	//Get device details from request
	err := ctx.ReadJSON(&deviceDetails)
	if err != nil {
		log.Println("Error while trying to collect data from request: ", err)
		ctx.StatusCode(http.StatusBadRequest)
		ctx.WriteString("Error: bad request.")
		return
	}
	device := &dputilities.RedfishDevice{
		Host:     deviceDetails.Host,
		Username: deviceDetails.Username,
		Password: string(deviceDetails.Password),
		PostBody: deviceDetails.PostBody,
	}

	redfishClient, err := dputilities.GetRedfishClient()
	if err != nil {
		errMsg := "error: internal processing error: " + err.Error()
		log.Println(errMsg)
		ctx.StatusCode(http.StatusInternalServerError)
		ctx.WriteString(errMsg)
		return
	}
	resp, err := redfishClient.DeviceCall(device, uri, http.MethodDelete)
	if err != nil {
		errorMessage := err.Error()
		log.Println(err)
		if resp == nil {
			ctx.StatusCode(http.StatusInternalServerError)
			ctx.WriteString("error while trying to delete volume: " + errorMessage)
			return
		}
	}

	// If a delete volume response contains any Location header then looping it to get final response
	if resp.Header.Get("Location") != "" {
		taskURI := resp.Header.Get("Location")
		//tracking the task id until reaches final state
		for {
			time.Sleep(10 * time.Second)
			resp, err = redfishClient.DeviceCall(device, taskURI, http.MethodGet)
			if err != nil {
				errorMessage := "Error while trying to get task id in delete volume: " + err.Error()
				log.Println(errorMessage)
				ctx.StatusCode(http.StatusInternalServerError)
				ctx.WriteString(errorMessage)
				return
			}
			if resp.StatusCode != http.StatusAccepted {
				log.Println("Final Status of task id while deleting a volume : ", resp.StatusCode)
				break
			}
		}
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errorMessage := err.Error()
		log.Println(err)
		ctx.StatusCode(http.StatusInternalServerError)
		ctx.WriteString("Error while trying to delete volume: " + errorMessage)
		return
	}
	log.Println("Response body: ", string(body))

	// If volume deletion is success then generating an event
	if resp.StatusCode == http.StatusOK {
		event := common.MessageData{
			OdataType: "#Event.v1_2_1.Event",
			Name:      "Volume removed event",
			Context:   "/redfish/v1/$metadata#Event.Event",
			Events: []common.Event{
				common.Event{
					EventType:      "ResourceRemoved",
					EventID:        "123",
					Severity:       "Critical",
					EventTimestamp: time.Now().String(),
					Message:        "Volume is deleted successfully",
					MessageID:      "ResourceEvent.1.0.3.ResourceRemoved",
					OriginOfCondition: &common.Link{
						Oid: uri,
					},
				},
			},
		}
		manualEvents(event, deviceDetails.Host)
	}
	ctx.StatusCode(resp.StatusCode)
	ctx.Write(body)
}

// manualEvents is used to generate an event based on the inputs provided
// It will send the received data and ip to publish method
func manualEvents(req common.MessageData, hostAddress string) {
	request, _ := json.Marshal(req)
	reqData := string(request)
	//replacing the response with north bound translation URL
	for key, value := range pluginConfig.Data.URLTranslation.NorthBoundURL {
		reqData = strings.Replace(reqData, key, value, -1)
	}
	event := common.Events{
		IP:      hostAddress,
		Request: []byte(reqData),
	}
	// Call writeEventToJobQueue to write events to worker pool
	writeEventToJobQueue(event)
}

// getVolumeCollection lists all the available volumes in the device
func getVolumeCollection(uri string, device *dputilities.RedfishDevice) (int, string, []string) {
	// Getting the list of volumes already exist in the server
	statusCode, _, resp, err := queryDevice(uri, device, http.MethodGet)
	if err != nil {
		errMsg := "error while getting volume collection details during create volume: " + err.Error()
		log.Println(errMsg)
		return statusCode, errMsg, nil
	}
	var volumes dpmodel.VolumesCollection
	err = json.Unmarshal(resp, &volumes)
	if err != nil {
		errMsg := "error while trying to unmarshal response data in create volume: " + err.Error()
		log.Println(errMsg)
		return http.StatusInternalServerError, errMsg, nil
	}

	var list []string
	for _, member := range volumes.Members {
		list = append(list, member.OdataID)
	}
	return http.StatusOK, "", list
}

// compareCollection will compare 2 slices and return the unique item from list2
func compareCollection(list1, list2 []string) string {
	var result string
	if len(list1) == 0 && len(list2) > 0 {
		result = list2[0]
	} else {
	outer:
		for i := len(list2) - 1; i >= 0; i-- {
		inner:
			for _, item := range list1 {
				if list2[i] == item {
					break inner
				} else {
					result = list2[i]
					break outer
				}
			}
		}
	}
	return result
}

// createResponse is used for creating a final response for create volume
func createResponse() []byte {
	resp := dpresponse.ErrorResopnse{
		Error: dpresponse.Error{
			Code:    response.Success,
			Message: "See @Message.ExtendedInfo for more information.",
			MessageExtendedInfo: []dpresponse.MsgExtendedInfo{
				dpresponse.MsgExtendedInfo{
					MessageID:   response.Created,
					Message:     "The resource has been created successfully",
					MessageArgs: []string{},
				},
			},
		},
	}
	body, _ := json.Marshal(resp)
	return body
}