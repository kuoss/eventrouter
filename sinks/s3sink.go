package sinks

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	v1 "k8s.io/api/core/v1"
)

type IUploader interface {
	Upload(*s3manager.UploadInput, ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
}

/*
S3Sink is the sink that uploads the kubernetes events as json object stored in a file.
The sinker uploads it to s3 if any of the below criteria gets fulfilled
1) Time(uploadInterval): If the specfied time has passed since the last upload it uploads
2) [TODO] Data size: If the total data getting uploaded becomes greater than N bytes

S3 is cheap and the sink can be used to store events data. S3 can later then be used with
Redshift and other visualization tools to use this data.
*/
type S3Sink struct {
	// uploader is the uploader client from aws which makes the API call to aws for upload
	uploader IUploader

	// bucket is the s3 bucket name where the events data would be stored
	bucket string

	// bucketDir is the first level directory in the bucket where the events would be stored
	bucketDir string

	// outPutFormat is the format in which the data is stored in the s3 file
	outputFormat string

	// lastUploadTimestamp stores the timestamp when the last upload to s3 happened
	lastUploadTimestamp int64

	// uploadInterval tells after how many seconds the next upload can happen
	// sink waits till this time is passed before next upload can happen
	uploadInterval time.Duration

	// eventCh is used to interact eventRouter and the sharedInformer
	eventCh chan EventData

	// overflow, when set, makes UpdateEvents drop events instead of blocking
	// once eventCh's buffer is full.
	overflow bool

	// bodyBuf stores all the event captured data in a buffer before upload
	bodyBuf *bytes.Buffer
}

// NewS3Sink is the factory method constructing a new S3Sink
func NewS3Sink(awsAccessKeyID string, s3SinkSecretAccessKey string, s3SinkRegion string, s3SinkBucket string, s3SinkBucketDir string, s3SinkUploadInterval int, overflow bool, bufferSize int, outputFormat string) (*S3Sink, error) {
	awsConfig := &aws.Config{
		Region:      aws.String(s3SinkRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, s3SinkSecretAccessKey, ""),
	}

	awsConfig = awsConfig.WithCredentialsChainVerboseErrors(true)
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}

	uploader := s3manager.NewUploader(sess)

	s := &S3Sink{
		uploader:       uploader,
		bucket:         s3SinkBucket,
		bucketDir:      s3SinkBucketDir,
		uploadInterval: time.Second * time.Duration(s3SinkUploadInterval),
		outputFormat:   outputFormat,
		bodyBuf:        bytes.NewBuffer(make([]byte, 0, 4096)),
	}

	s.eventCh = make(chan EventData, bufferSize)
	s.overflow = overflow

	return s, nil
}

// UpdateEvents implements the EventSinkInterface. If overflow is set, this
// never blocks: events beyond the channel's buffer are discarded. Otherwise
// it blocks once the buffer is full, applying backpressure to the caller.
func (s *S3Sink) UpdateEvents(eNew *v1.Event, eOld *v1.Event) {
	evt := NewEventData(eNew, eOld)
	if s.overflow {
		select {
		case s.eventCh <- evt:
		default:
		}
		return
	}
	s.eventCh <- evt
}

// Run sits in a loop, waiting for data to come in through h.eventCh,
// and forwarding them to the HTTP sink. If multiple events have happened
// between loop iterations, it puts all of them in one request instead of
// making a single request per event.
func (s *S3Sink) Run(stopCh <-chan bool) {
loop:
	for {
		select {
		case evt := <-s.eventCh:
			// Start with just this event...
			arr := []EventData{evt}

			// Consume all buffered events into an array, in case more have been written
			// since we last forwarded them
			numEvents := len(s.eventCh)
			for i := 0; i < numEvents; i++ {
				arr = append(arr, <-s.eventCh)
			}

			s.drainEvents(arr)
		case <-stopCh:
			break loop
		}
	}
}

// drainEvents takes an array of event data and sends it to s3
func (s *S3Sink) drainEvents(events []EventData) {
	var written int64
	for _, evt := range events {
		switch s.outputFormat {
		case "rfc5424":
			w, err := evt.WriteRFC5424(s.bodyBuf)
			written += w
			if err != nil {
				slog.Warn("could not write to event request body", "written", written, "err", err)
				return
			}
		case "flatjson":
			w, err := evt.WriteFlattenedJSON(s.bodyBuf)
			written += w
			if err != nil {
				slog.Warn("could not write to event request body", "written", written, "err", err)
				return
			}
		default:
			err := errors.New("invalid Sink Output Format specified")
			panic(err.Error())
		}
		s.bodyBuf.Write([]byte{'\n'})
		written++
	}

	if !s.canUpload() {
		return
	}

	s.upload()
}

// canUpload verifies the conditions suitable for a new file upload and upload the data
func (s *S3Sink) canUpload() bool {
	now := time.Now().UnixNano()
	return (s.lastUploadTimestamp + s.uploadInterval.Nanoseconds()) < now
}

// getNewKey gets the key name based on time
func (s *S3Sink) getNewKey(t time.Time) string {
	return fmt.Sprintf("%s/%d/%d/%d/%d.txt", s.bucketDir, t.Year(), t.Month(), t.Day(), t.UnixNano())
}

// upload uploads the events stored in buffer to s3 in the specified key
// and clears the buffer
func (s *S3Sink) upload() {
	now := time.Now()
	key := s.getNewKey(now)

	_, err := s.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   s.bodyBuf,
	})
	if err != nil {
		slog.Error("error uploading to s3", "key", key, "err", err)
	}
	slog.Info("uploaded to s3", "key", key)
	s.lastUploadTimestamp = now.UnixNano()

	s.bodyBuf.Truncate(0)
}
