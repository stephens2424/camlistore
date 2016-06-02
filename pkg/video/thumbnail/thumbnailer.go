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

package thumbnail

import (
	"io"
	"net/url"
	"os/exec"

	"camlistore.org/pkg/video/ffmpeg"
)

// Thumbnailer is the interface that wraps the Command method.
//
// Command receives the (HTTP) uri from where to get the video to generate a
// thumbnail and returns program and arguments.
// The command is expected to output the thumbnail image on its stdout, or exit
// with an error code.
type Thumbnailer interface {
	Command(*url.URL) (prog string, args []string)
}

var DefaultCmd = ffmpeg.GenThumbnailCmd

var _ Thumbnailer = (*FFmpeg)(nil)

// FFmpeg is a Thumbnailer that generates a thumbnail with ffmpeg.
type FFmpeg struct{}

// Command implements the Command method for the Thumbnailer interface.
func (f FFmpeg) Command(uri *url.URL) (string, []string) {
	return DefaultCmd[0], replaceURI(DefaultCmd[1:], uri)
}

func replaceURI(withURI []string, uri *url.URL) []string {
	args := make([]string, len(withURI))
	for index, arg := range withURI {
		if arg == "$uri" {
			args[index] = uri.String()
		} else {
			args[index] = arg
		}
	}
	return args
}

var _ Thumbnailer = (*thumbnailer)(nil)

type thumbnailer struct {
	prog string
	args []string
}

func (ct *thumbnailer) Command(uri *url.URL) (string, []string) {
	return ct.prog, replaceURI(ct.args, uri)
}

func buildCmd(tn Thumbnailer, uri *url.URL, out io.Writer) *exec.Cmd {
	prog, args := tn.Command(uri)
	cmd := exec.Command(prog, args...)
	cmd.Stdout = out
	return cmd
}
