package sinks

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	models "github.com/rockset/rockset-go-client/lib/go"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
)

// MockRocksetClient is a mock implementation of the IRocksetClientWrapper interface
type MockRocksetClient struct {
	mock.Mock
}

func (m *MockRocksetClient) Query(body models.QueryRequest) (models.QueryResponse, *http.Response, error) {
	args := m.Called(body)
	return args.Get(0).(models.QueryResponse), args.Get(1).(*http.Response), args.Error(2)
}

func (m *MockRocksetClient) AddDocuments(workspace string, collection string, dinfo models.AddDocumentsRequest) (models.AddDocumentsResponse, *http.Response, error) {
	args := m.Called(workspace, collection, dinfo)
	return args.Get(0).(models.AddDocumentsResponse), args.Get(1).(*http.Response), args.Error(2)
}

func TestRocksetSink_UpdateEvents(t *testing.T) {
	mockClient := new(MockRocksetClient)
	rocksetSink := &RocksetSink{
		client:                mockClient,
		rocksetWorkspaceName:  "testWorkspace",
		rocksetCollectionName: "testCollection",
	}

	eventNew := &v1.Event{
		// populate the event object as you need it to be
	}
	eventOld := &v1.Event{
		// populate the event object as you need it to be
	}

	eData := NewEventData(eventNew, eventOld)
	eJSONBytes, _ := json.Marshal(eData)
	var m map[string]interface{}
	_ = json.Unmarshal(eJSONBytes, &m)
	docs := []interface{}{m}
	dinfo := models.AddDocumentsRequest{Data: docs}

	mockClient.On("AddDocuments", "testWorkspace", "testCollection", dinfo).Return(models.AddDocumentsResponse{}, &http.Response{}, nil)

	rocksetSink.UpdateEvents(eventNew, eventOld)

	mockClient.AssertExpectations(t)
}

func TestRocksetSink_UpdateEvents_Error(t *testing.T) {
	mockClient := new(MockRocksetClient)
	rocksetSink := &RocksetSink{
		client:                mockClient,
		rocksetWorkspaceName:  "testWorkspace",
		rocksetCollectionName: "testCollection",
	}

	eventNew := &v1.Event{
		// populate the event object as you need it to be
	}
	eventOld := &v1.Event{
		// populate the event object as you need it to be
	}

	eData := NewEventData(eventNew, eventOld)
	eJSONBytes, _ := json.Marshal(eData)
	var m map[string]interface{}
	_ = json.Unmarshal(eJSONBytes, &m)
	docs := []interface{}{m}
	dinfo := models.AddDocumentsRequest{Data: docs}

	mockClient.On("AddDocuments", "testWorkspace", "testCollection", dinfo).Return(models.AddDocumentsResponse{}, &http.Response{}, errors.New("error adding documents"))

	rocksetSink.UpdateEvents(eventNew, eventOld)

	mockClient.AssertExpectations(t)
}
