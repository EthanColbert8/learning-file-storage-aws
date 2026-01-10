package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
)

type ffmpegVideoStreams struct {
	Streams []struct {
		Index              int    `json:"index"`
		CodecName          string `json:"codec_name,omitempty"`
		CodecLongName      string `json:"codec_long_name,omitempty"`
		Profile            string `json:"profile,omitempty"`
		CodecType          string `json:"codec_type"`
		CodecTagString     string `json:"codec_tag_string"`
		CodecTag           string `json:"codec_tag"`
		Width              int    `json:"width,omitempty"`
		Height             int    `json:"height,omitempty"`
		CodedWidth         int    `json:"coded_width,omitempty"`
		CodedHeight        int    `json:"coded_height,omitempty"`
		HasBFrames         int    `json:"has_b_frames,omitempty"`
		SampleAspectRatio  string `json:"sample_aspect_ratio,omitempty"`
		DisplayAspectRatio string `json:"display_aspect_ratio,omitempty"`
		PixFmt             string `json:"pix_fmt,omitempty"`
		Level              int    `json:"level,omitempty"`
		ColorRange         string `json:"color_range,omitempty"`
		ColorSpace         string `json:"color_space,omitempty"`
		ColorTransfer      string `json:"color_transfer,omitempty"`
		ColorPrimaries     string `json:"color_primaries,omitempty"`
		ChromaLocation     string `json:"chroma_location,omitempty"`
		FieldOrder         string `json:"field_order,omitempty"`
		Refs               int    `json:"refs,omitempty"`
		IsAvc              string `json:"is_avc,omitempty"`
		NalLengthSize      string `json:"nal_length_size,omitempty"`
		ID                 string `json:"id"`
		RFrameRate         string `json:"r_frame_rate"`
		AvgFrameRate       string `json:"avg_frame_rate"`
		TimeBase           string `json:"time_base"`
		StartPts           int    `json:"start_pts"`
		StartTime          string `json:"start_time"`
		DurationTs         int    `json:"duration_ts"`
		Duration           string `json:"duration"`
		BitRate            string `json:"bit_rate,omitempty"`
		BitsPerRawSample   string `json:"bits_per_raw_sample,omitempty"`
		NbFrames           string `json:"nb_frames"`
		ExtradataSize      int    `json:"extradata_size"`
		SampleFmt          string `json:"sample_fmt,omitempty"`
		SampleRate         string `json:"sample_rate,omitempty"`
		Channels           int    `json:"channels,omitempty"`
		ChannelLayout      string `json:"channel_layout,omitempty"`
		BitsPerSample      int    `json:"bits_per_sample,omitempty"`
		InitialPadding     int    `json:"initial_padding,omitempty"`
		Disposition        struct {
			Default         int `json:"default"`
			Dub             int `json:"dub"`
			Original        int `json:"original"`
			Comment         int `json:"comment"`
			Lyrics          int `json:"lyrics"`
			Karaoke         int `json:"karaoke"`
			Forced          int `json:"forced"`
			HearingImpaired int `json:"hearing_impaired"`
			VisualImpaired  int `json:"visual_impaired"`
			CleanEffects    int `json:"clean_effects"`
			AttachedPic     int `json:"attached_pic"`
			TimedThumbnails int `json:"timed_thumbnails"`
			NonDiegetic     int `json:"non_diegetic"`
			Captions        int `json:"captions"`
			Descriptions    int `json:"descriptions"`
			Metadata        int `json:"metadata"`
			Dependent       int `json:"dependent"`
			StillImage      int `json:"still_image"`
			Multilayer      int `json:"multilayer"`
		} `json:"disposition"`
		Tags struct {
			Language    string `json:"language,omitempty"`
			HandlerName string `json:"handler_name,omitempty"`
			VendorID    string `json:"vendor_id,omitempty"`
			Encoder     string `json:"encoder,omitempty"`
			Timecode    string `json:"timecode,omitempty"`
		} `json:"tags,omitempty"`
	} `json:"streams"`
}

const LANDSCAPE float64 = 16.0 / 9.0
const PORTRAIT float64 = 9.0 / 16.0
const TOLERANCE float64 = 1.0e-2

func GetVideoAspectRatio(filePath string) (string, error) {
	var outputBuffer bytes.Buffer

	cmdToRun := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	cmdToRun.Stdout = &outputBuffer

	err := cmdToRun.Run()
	if err != nil {
		return "", fmt.Errorf("error running ffprobe command: %w", err)
	}

	var ffprobeOutput ffmpegVideoStreams
	err = json.Unmarshal(outputBuffer.Bytes(), &ffprobeOutput)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling video metadata: %w", err)
	}

	aspectRatio := float64(ffprobeOutput.Streams[0].Width) / float64(ffprobeOutput.Streams[0].Height)

	if math.Abs(aspectRatio-LANDSCAPE) < TOLERANCE {
		return "landscape", nil
	}
	if math.Abs(aspectRatio-PORTRAIT) < TOLERANCE {
		return "portrait", nil
	}
	return "other", nil
}

func ProcessVideoForFastStart(filePath string) (string, error) {
	newFilePath := fmt.Sprintf("%s.processed", filePath)
	cmdToRun := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", newFilePath)

	err := cmdToRun.Run()
	if err != nil {
		return "", fmt.Errorf("error configuring video file for fast start: %w", err)
	}

	return newFilePath, nil
}
