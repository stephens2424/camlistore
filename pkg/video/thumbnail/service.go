/*
Copyright 2014 The Camlistore Authors

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

/*
Package thumbnail generates image thumbnails from videos.

(*Service).Generate spawns an HTTP server listening on a local random
port to serve the video to an external program (see Thumbnailer interface).
The external program is expected to output the thumbnail image on its
standard output.

The default implementation uses ffmpeg.

See ServiceFromConfig for accepted configuration.
*/
package thumbnail // import "camlistore.org/pkg/video/thumbnail"

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"camlistore.org/pkg/blob"
	"camlistore.org/pkg/netutil"
	"camlistore.org/pkg/types/serverconfig"
	"camlistore.org/pkg/video/ffmpeg"

	"go4.org/syncutil"
)

// A Service controls the generation of video thumbnails.
type Service struct {
	thumbnailer Thumbnailer
	// Timeout is the maximum duration for the thumbnailing subprocess execution.
	timeout time.Duration
	gate    *syncutil.Gate // limits the number of concurrent thumbnailer subprocesses.
}

// NewService builds a new Service. If no conf.Command and ffmpeg is not installed,
// it returns ffmpeg.ErrFFmpegNotFound.
func NewService(conf *serverconfig.VideoThumbnail) (*Service, error) {
	var th Thumbnailer
	if len(conf.Command) < 1 {
		if !ffmpeg.Available() {
			return nil, ffmpeg.ErrFFmpegNotFound
		}
		th = FFmpeg{}
	} else {
		th = &thumbnailer{prog: conf.Command[0], args: conf.Command[1:]}
	}
	var g *syncutil.Gate
	if conf.MaxProcs > 0 {
		g = syncutil.NewGate(conf.MaxProcs)
	}
	if conf.Timeout == 0 {
		conf.Timeout = 5 * time.Second
	} else {
		// Because we document Timeout to be in milliseconds, as nanoseconds are not
		// user-friendly.
		conf.Timeout *= 1e6
	}

	return &Service{
		thumbnailer: th,
		timeout:     conf.Timeout,
		gate:        g,
	}, nil
}

var errTimeout = errors.New("timeout.")

// Generate reads the video given by videoRef from src and writes its thumbnail image to w.
func (s *Service) Generate(videoRef blob.Ref, w io.Writer, src blob.Fetcher) error {
	if s.gate != nil {
		s.gate.Start()
		defer s.gate.Done()
	}

	ln, err := netutil.ListenOnLocalRandomPort()
	if err != nil {
		return err
	}
	defer ln.Close()

	videoUri := &url.URL{
		Scheme: "http",
		Host:   ln.Addr().String(),
		Path:   videoRef.String(),
	}

	cmdErrc := make(chan error, 1)
	cmd := buildCmd(s.thumbnailer, videoUri, w)
	cmdErrOut, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	defer cmd.Process.Kill()
	go func() {
		out, err := ioutil.ReadAll(cmdErrOut)
		if err != nil {
			cmdErrc <- err
			return
		}
		cmd.Wait()
		if cmd.ProcessState.Success() {
			cmdErrc <- nil
			return
		}
		cmdErrc <- fmt.Errorf("thumbnail subprocess failed:\n%s", out)
	}()

	servErrc := make(chan error, 1)
	go func() {
		servErrc <- http.Serve(ln, createVideothumbnailHandler(videoRef, src))
		// N.B: not leaking because closing ln makes Serve return.
	}()

	select {
	case err := <-cmdErrc:
		return err
	case err := <-servErrc:
		return err
	case <-s.timer():
		return errTimeout
	}
}

func (s *Service) timer() <-chan time.Time {
	if s.timeout < 0 {
		return make(<-chan time.Time)
	}
	return time.After(s.timeout)
}
