package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	gosync "sync"

	"github.com/victor/coros-music/internal/pipeline"
	"github.com/victor/coros-music/internal/stream"
)

// Job represents a single track to download/transcode/stream.
type Job struct {
	TrackID  string
	Title    string
	Artist   string
	Album    string
	Filename string
	Query    string // search query for yt-dlp: "artist - title"
}

// Result holds the output of a completed job.
type Result struct {
	Job   Job
	Data  []byte
	Error error
	Index int
}

type Coordinator struct {
	workers int
}

func NewCoordinator() *Coordinator {
	workers := 4
	if w := os.Getenv("COROS_WORKERS"); w != "" {
		if n, err := strconv.Atoi(w); err == nil && n > 0 {
			workers = n
		}
	}
	return &Coordinator{workers: workers}
}

type syncRequest struct {
	PlaylistID    string   `json:"playlistId"`
	PlaylistName  string   `json:"playlistName"`
	TrackIDs      []string `json:"trackIds"`
	ExistingFiles []string `json:"existingFiles"`
	// Optional: track metadata for search queries
	Tracks []struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Artist   string `json:"artist"`
		Album    string `json:"album"`
		Filename string `json:"syncFilename"`
	} `json:"tracks,omitempty"`
}

func buildJobs(req syncRequest) ([]Job, error) {
	if len(req.TrackIDs) == 0 {
		return nil, fmt.Errorf("no tracks provided")
	}

	jobs := make([]Job, 0, len(req.TrackIDs))
	trackMap := make(map[string]struct{})
	for _, id := range req.TrackIDs {
		trackMap[id] = struct{}{}
	}

	for _, t := range req.Tracks {
		if _, ok := trackMap[t.ID]; !ok {
			continue
		}
		jobs = append(jobs, Job{
			TrackID:  t.ID,
			Title:    t.Title,
			Artist:   t.Artist,
			Album:    t.Album,
			Filename: t.Filename,
			Query:    t.Artist + " - " + t.Title,
		})
	}

	if len(jobs) == 0 {
		for _, id := range req.TrackIDs {
			jobs = append(jobs, Job{
				TrackID:  id,
				Filename: id + ".mp3",
				Query:    id,
			})
		}
	}

	return jobs, nil
}

func (c *Coordinator) HandleStream(w http.ResponseWriter, r *http.Request) {
	var req syncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	jobs, err := buildJobs(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	total := len(jobs)
	mw := stream.NewWriter(w, total)

	// Process tracks with worker pool, but write in order
	results := make([]Result, total)
	var wg gosync.WaitGroup
	sem := make(chan struct{}, c.workers)

	for i, job := range jobs {
		wg.Add(1)
		go func(idx int, j Job) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release

			var buf bytes.Buffer
			err := pipeline.Transcode(ctx, j.Query, &buf)

			var data []byte
			if err == nil {
				if buf.Len() == 0 {
					err = fmt.Errorf("empty audio output")
				}
			}

			if err == nil {
				// Tag the MP3
				tagged, tagErr := pipeline.TagMP3(buf.Bytes(), j.Title, j.Artist, j.Album)
				if tagErr != nil {
					log.Printf("tagging failed for %s: %v (using untagged)", j.Filename, tagErr)
					data = buf.Bytes()
				} else {
					data = tagged
				}
			}

			results[idx] = Result{
				Job:   j,
				Data:  data,
				Error: err,
				Index: idx,
			}
		}(i, job)
	}

	// Wait for all workers, then write results in order
	wg.Wait()

	for _, res := range results {
		if ctx.Err() != nil {
			break // client disconnected
		}

		if res.Error != nil {
			log.Printf("track %s failed: %v", res.Job.Filename, res.Error)
			if err := mw.WriteErrorPart(res.Job.Filename, res.Error.Error()); err != nil {
				log.Printf("failed to write error part: %v", err)
				return
			}
			continue
		}

		if err := mw.WritePart(res.Job.Filename, res.Data); err != nil {
			log.Printf("failed to write part for %s: %v", res.Job.Filename, err)
			return
		}
	}

	if err := mw.Close(); err != nil {
		log.Printf("failed to close multipart: %v", err)
	}
}
