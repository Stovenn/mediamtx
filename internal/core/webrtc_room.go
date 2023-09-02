package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/google/uuid"
)

type s3Client struct {
	S3Client *s3.Client
}

// CreateBucket creates a bucket with the specified name in the specified Region.
func (c *s3Client) CreateBucket(name string, region string) error {
	_, err := c.S3Client.CreateBucket(context.TODO(), &s3.CreateBucketInput{
		Bucket: aws.String(name),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		},
	})
	if err != nil {
		switch err.(type) {
		case *types.BucketAlreadyOwnedByYou: 
		case *types.BucketAlreadyExists:
			return nil
		default:
			log.Printf("Couldn't create bucket %v in Region %v. Here's why: %v\n",
			name, region, err)
			return err
	}
}
return nil	
}

func (c *s3Client) UploadObject(bucketName string, objectKey string, file *os.File) error {
	uploader := manager.NewUploader(c.S3Client)
	_, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   file,
	})
	if err != nil {
		log.Printf("Couldn't upload large object to %v:%v. Here's why: %v\n",
			bucketName, objectKey, err)
	}
	return err
}

type Room struct {
	uuid             uuid.UUID
	clubName string
	eventName string
	recording        bool
	s3Client         *s3Client
	streamers        map[string]*streamer
	sessions         map[*webRTCSession]struct{}
	sessionsBySecret map[uuid.UUID]*webRTCSession
}
type File struct {
	Filename string
	os.File
}
type streamer struct {
	id      string
	session *webRTCSession
}

func (r *Room) join(streamID string) error {
	s := &streamer{
		id: streamID,
	}
	r.streamers[streamID] = s
	return nil
}

func (r *Room) apiItem() *apiWebRTCRoom {
	var paths []string
	for path := range r.streamers {
		paths = append(paths, path)
	}

	return &apiWebRTCRoom{
		ID:    r.uuid,
		Paths: paths,
	}
}

func (r *Room) record() error {
	bucketName := strings.ToLower(r.clubName)
	err := r.s3Client.CreateBucket(bucketName, "eu-west-3")
	if err != nil {
		return err
	}
	r.recording = true
	return nil
}

func (r *Room) cleanup() error {
	for s := range r.sessions {
		if len(s.writers) > 0 {
			for fn, _ := range s.writers {
				go func(filename string) {
					// Open the file to upload
					file, err := os.Open(filename)
					if err != nil {
						fmt.Println("Error opening file:", err)
						return
					}
					defer file.Close()
					//save file to S3

					fmt.Println(filename)
					bucketName := strings.ToLower(r.clubName)
					objectKey := fmt.Sprintf("%s/%s", r.eventName, filepath.Base(filename))
					r.s3Client.UploadObject(bucketName, objectKey, file)

					//delete file from disk
					os.Remove(filename)
				}(fn)
			}
		}
		delete(r.sessions, s)
		delete(r.sessionsBySecret, s.secret)
		s.close()
	}
	for k := range r.streamers {
		delete(r.streamers, k)
	}
	return nil
}
