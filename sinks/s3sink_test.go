package sinks

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

// MockUploader struct fulfills the IUploader interface for testing
type MockUploader struct {
	mock.Mock
}

func (m *MockUploader) Upload(input *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*s3manager.UploadOutput), args.Error(1)
}

func TestS3Sink_Upload(t *testing.T) {
	mockUploader := new(MockUploader)
	s3Sink, err := NewS3Sink("accessKeyID", "secretAccessKey", "region", "bucket", "bucketDir", 10, true, 1024, "flatjson")
	require.NoError(t, err)
	s3Sink.uploader = mockUploader

	// Create a mock event
	event := &v1.Event{
		Message: "Test event",
	}

	// Set up the expected call to the mock uploader
	mockUploader.On("Upload", mock.AnythingOfType("*s3manager.UploadInput")).Return(&s3manager.UploadOutput{}, nil).Times(2)

	// Simulate receiving a new event
	s3Sink.UpdateEvents(event, nil)

	// Set the last upload timestamp to a past time to force an upload
	s3Sink.lastUploadTimestamp = time.Now().Add(-11 * time.Second).UnixNano()

	// Simulate event processing
	s3Sink.drainEvents([]EventData{NewEventData(event, nil)})

	require.False(t, s3Sink.canUpload())

	// Make upload happen
	s3Sink.upload()

	// Verify that the upload happened once
	mockUploader.AssertNumberOfCalls(t, "Upload", 2)
}

func TestCanUpload(t *testing.T) {
	s3Sink, err := NewS3Sink("accessKeyID", "secretAccessKey", "region", "bucket", "bucketDir", 5, true, 1024, "flatjson")
	require.NoError(t, err)

	// Set last upload time to now
	s3Sink.lastUploadTimestamp = time.Now().Add(-10 * time.Second).UnixNano()
	require.True(t, s3Sink.canUpload())

	s3Sink.lastUploadTimestamp = time.Now().UnixNano()
	require.False(t, s3Sink.canUpload())
}
