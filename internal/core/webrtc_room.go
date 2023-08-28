package core

import (
	"fmt"

	"github.com/google/uuid"
)

type Room struct {
	uuid uuid.UUID 
	recording bool
	streamers map[string]*streamer 
	sessions map[*webRTCSession]struct{}
	sessionsBySecret map[uuid.UUID]*webRTCSession 
}

type streamer struct {
	id string
	session *webRTCSession
}

func (r *Room) join(streamID string)error {
	s := &streamer{
		id: streamID,
	}
	r.streamers[streamID] = s
	return nil
}

func (r *Room) apiItem() *apiWebRTCRoom{

	var paths []string
	for path := range r.streamers {
		paths = append(paths, path)
	}

	return &apiWebRTCRoom{
		ID:                        r.uuid,
		Paths: paths,
	}
}

func (r *Room) record() error {
	r.recording = true
	return nil
}


func (r *Room) cleanup() error {
	for s := range r.sessions {
		if len(s.writers) > 0 {
			for filename, _ := range s.writers {
				fmt.Println(filename)
				//save file to S3
				//delete file from disk
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

// func (s *Room) Log(level logger.Level, format string, args ...interface{}) {
// 	id := hex.EncodeToString(s.uuid[:4])
// 	s.parent.Log(level, "[room %v] "+format, append([]interface{}{id}, args...)...)
// }