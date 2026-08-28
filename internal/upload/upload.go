package upload

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/christophercuongkim/studio/internal/videoyaml"
	"google.golang.org/api/youtube/v3"
)

// Options configures an upload run.
type Options struct {
	ProjectDir string
	DryRun     bool
	Privacy    string // override video.yaml privacy
	Update     bool   // metadata-only update of an existing video
	SkipQC     bool
	Now        time.Time
	// AuthPrompt, when set, receives the OAuth URL on first-time auth so a UI can
	// surface it; nil falls back to printing it on stdout (the CLI).
	AuthPrompt func(url string)
}

// Result reports what happened.
type Result struct {
	VideoID string
	Updated bool
	DryRun  bool
}

// Run validates and uploads (or updates) the project's video (plan §15).
func Run(ctx context.Context, opts Options) (*Result, error) {
	vy, err := videoyaml.Load(opts.ProjectDir)
	if err != nil {
		return nil, err
	}
	if err := Validate(vy, opts.ProjectDir); err != nil {
		return nil, err
	}

	if opts.DryRun {
		fmt.Println(DryRun(vy, opts.Privacy))
		return &Result{DryRun: true}, nil
	}

	existing := ""
	if vy.YouTube.VideoID != nil {
		existing = *vy.YouTube.VideoID
	}

	if opts.Update {
		if existing == "" {
			return nil, errors.New("--update needs an already-uploaded video (youtube.videoId is empty)")
		}
	} else {
		if existing != "" {
			return nil, fmt.Errorf("already uploaded as %s; use --update to change metadata", existing)
		}
		if err := CheckQCGate(opts.ProjectDir, opts.SkipQC); err != nil {
			return nil, err
		}
	}

	svc, err := service(ctx, opts.AuthPrompt)
	if err != nil {
		return nil, err
	}
	video := buildVideo(vy, opts.Privacy)

	var videoID string
	if opts.Update {
		video.Id = existing
		if _, err := svc.Videos.Update([]string{"snippet", "status"}, video).Do(); err != nil {
			return nil, fmt.Errorf("videos.update: %w", err)
		}
		videoID = existing
	} else {
		videoID, err = insert(ctx, svc, video, filepath.Join(opts.ProjectDir, vy.Render))
		if err != nil {
			return nil, err
		}
	}

	if err := setThumbnail(svc, videoID, opts.ProjectDir, vy); err != nil {
		return nil, err
	}
	if !opts.Update {
		if err := addToPlaylists(svc, videoID, vy.PlaylistIDs); err != nil {
			return nil, err
		}
	}

	// Write the videoId/uploadedAt back to video.yaml.
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	ts := now.Format(time.RFC3339)
	vy.YouTube.VideoID = &videoID
	vy.YouTube.UploadedAt = &ts
	if err := videoyaml.Save(opts.ProjectDir, vy); err != nil {
		return nil, fmt.Errorf("upload succeeded (%s) but writing video.yaml failed: %w", videoID, err)
	}
	return &Result{VideoID: videoID, Updated: opts.Update}, nil
}

// insert performs the resumable upload with a progress bar and simple retries.
func insert(ctx context.Context, svc *youtube.Service, video *youtube.Video, renderPath string) (string, error) {
	f, err := os.Open(renderPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	call := svc.Videos.Insert([]string{"snippet", "status"}, video).
		Media(f).
		ProgressUpdater(func(current, total int64) {
			if total > 0 {
				fmt.Printf("\ruploading… %3.0f%%", float64(current)/float64(total)*100)
			}
		})

	var resp *youtube.Video
	backoff := time.Second
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err = call.Context(ctx).Do()
		if err == nil {
			fmt.Println()
			return resp.Id, nil
		}
		if attempt < 3 {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	return "", fmt.Errorf("videos.insert failed after retries: %w", err)
}

func setThumbnail(svc *youtube.Service, videoID, projectDir string, vy *videoyaml.VideoYAML) error {
	if vy.Thumbnail == "" {
		return nil
	}
	path := filepath.Join(projectDir, vy.Thumbnail)
	f, err := os.Open(path)
	if err != nil {
		return nil // no thumbnail on disk is fine; YouTube picks a frame
	}
	defer f.Close()
	if _, err := svc.Thumbnails.Set(videoID).Media(f).Do(); err != nil {
		return fmt.Errorf("thumbnails.set: %w", err)
	}
	return nil
}

func addToPlaylists(svc *youtube.Service, videoID string, playlists []string) error {
	for _, pid := range playlists {
		item := &youtube.PlaylistItem{
			Snippet: &youtube.PlaylistItemSnippet{
				PlaylistId: pid,
				ResourceId: &youtube.ResourceId{Kind: "youtube#video", VideoId: videoID},
			},
		}
		if _, err := svc.PlaylistItems.Insert([]string{"snippet"}, item).Do(); err != nil {
			return fmt.Errorf("playlistItems.insert(%s): %w", pid, err)
		}
	}
	return nil
}
