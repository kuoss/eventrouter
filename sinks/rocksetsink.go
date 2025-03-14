/*
Copyright 2019 The Contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sinks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	apiclient "github.com/rockset/rockset-go-client"
	models "github.com/rockset/rockset-go-client/lib/go"
	v1 "k8s.io/api/core/v1"
)

type IRocksetClientWrapper interface {
	Query(body models.QueryRequest) (models.QueryResponse, *http.Response, error)
	AddDocuments(workspace string, collection string, dinfo models.AddDocumentsRequest) (models.AddDocumentsResponse, *http.Response, error)
}

type RocksetClientWrapper struct {
	client *apiclient.RockClient
}

func NewRocksetClientWrapper(apiKey string, apiServer string) *RocksetClientWrapper {
	client := apiclient.Client(apiKey, apiServer)
	return &RocksetClientWrapper{client}
}

func (rcw *RocksetClientWrapper) Query(body models.QueryRequest) (models.QueryResponse, *http.Response, error) {
	return rcw.client.Query(body)
}

func (rcw *RocksetClientWrapper) AddDocuments(workspace string, collection string, dinfo models.AddDocumentsRequest) (models.AddDocumentsResponse, *http.Response, error) {
	docService := rcw.client.Documents
	return docService.Add(workspace, collection, dinfo)
}

/*
RocksetSink is a sink that uploads the kubernetes events as json object
and converts them to documents inside of a Rockset collection.

Rockset can later be used with
many different connectors such as Tableau or Redash to use this data.
*/
type RocksetSink struct {
	client                IRocksetClientWrapper
	rocksetCollectionName string
	rocksetWorkspaceName  string
}

// NewRocksetSink will create a new RocksetSink with default options, returned as
// an EventSinkInterface
func NewRocksetSink(rocksetAPIKey string, rocksetCollectionName string, rocksetWorkspaceName string) EventSinkInterface {
	clientWrapper := NewRocksetClientWrapper(rocksetAPIKey, "https://api.rs2.usw2.rockset.com")
	return &RocksetSink{
		client:                clientWrapper,
		rocksetCollectionName: rocksetCollectionName,
		rocksetWorkspaceName:  rocksetWorkspaceName,
	}
}

// UpdateEvents implements the EventSinkInterface
func (rs *RocksetSink) UpdateEvents(eNew *v1.Event, eOld *v1.Event) {
	eData := NewEventData(eNew, eOld)
	eJSONBytes, err := json.Marshal(eData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "json.Marshal err: %v", err)
		return
	}
	var m map[string]interface{}
	err = json.Unmarshal(eJSONBytes, &m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "json.Unmarshal err: %v", err)
		return
	}
	docs := []interface{}{m}
	dinfo := models.AddDocumentsRequest{Data: docs}
	_, _, err = rs.client.AddDocuments(rs.rocksetWorkspaceName, rs.rocksetCollectionName, dinfo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Documents.Add err: %v", err)
		return
	}
}
