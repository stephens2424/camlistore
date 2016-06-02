/*
Copyright 2015 The Camlistore Authors

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

package ffmpeg

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"camlistore.org/pkg/buildinfo"
	"camlistore.org/pkg/types/camtypes"
)

const ffmpegBin = "ffmpeg"

var ErrFFmpegNotFound = errors.New("ffmpeg not found in path")

// GenThumbnailCmd is a command to generate a thumbnail image from a video with ffmpeg.
var GenThumbnailCmd = []string{
	ffmpegBin,
	"-seekable", "1",
	"-i", "$uri",
	"-vf", "thumbnail",
	"-frames:v", "1",
	"-f", "image2pipe",
	"-c:v", "png",
	"pipe:1",
}

var (
	checkAvailability sync.Once
	available         bool
	debug             bool
	noOutputFile      = []byte("At least one output file must be specified\n")
)

func Available() bool {
	checkAvailability.Do(func() {
		if p, err := exec.LookPath(ffmpegBin); p != "" && err == nil {
			available = true
			log.Printf("ffmpeg found at %s; optional video features enabled.", p)
		}
		if !available {
			log.Printf("%s not found in PATH, optional video features disabled.", ffmpegBin)
		}
	})
	return available
}

func init() {
	buildinfo.RegisterFFmpegStatusFunc(ffmpegStatus)
}

func videoDebug(msg string) {
	if debug {
		log.Print(msg)
	}
}

func ffmpegStatus() string {
	if Available() {
		return "ffmpeg available"
	}
	return "ffmpeg options unavailable"
}

// videoStreamInfoRxp is used to extract the height and width of a video from the ffmpeg stderr output.
var videoStreamInfoRxp = regexp.MustCompile(`^    Stream #\d:\d.*?: Video: .*, ((\d+)x(\d+)).*`)

func VideoInfo(r io.Reader) (*camtypes.VideoInfo, error) {
	if !Available() {
		return nil, ErrFFmpegNotFound
	}

	args := []string{ffmpegBin, "-i", "-"}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = r
	cmd.Stdout = ioutil.Discard
	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if bytes.HasSuffix(stderr.Bytes(), noOutputFile) {
			// expected error, ignoring.
		} else {
			videoDebug(fmt.Sprintf("ffmpeg error: %v, %s", err, stderr))
			return nil, fmt.Errorf("ffmpeg error: %v", err)
		}
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		l := scanner.Text()
		if !strings.HasPrefix(l, "    Stream") {
			continue
		}
		matches := videoStreamInfoRxp.FindStringSubmatch(l)
		if matches == nil {
			continue
		}
		width, err := strconv.Atoi(matches[2])
		if err != nil {
			videoDebug(fmt.Sprintf("bogus width in ffmpeg output: %q", matches[2]))
			continue
		}
		height, err := strconv.Atoi(matches[3])
		if err != nil {
			videoDebug(fmt.Sprintf("bogus height in ffmpeg output: %q", matches[3]))
			continue
		}
		return &camtypes.VideoInfo{
			Width:  uint16(width),
			Height: uint16(height),
		}, nil
	}
	return nil, errors.New("No video info found")
}
