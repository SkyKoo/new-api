package openai

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConvertImageEditRequestMultipart verifies that ConvertImageRequest
// re-serializes multipart image edit requests with all fields (including
// stream) and the file intact, both when the form was already parsed and when
// it must be re-parsed from the reusable body.
func TestConvertImageEditRequestMultipart(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newMultipartContext := func(t *testing.T, prompt string) *gin.Context {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.WriteField("model", "gpt-image-1"))
		require.NoError(t, writer.WriteField("prompt", prompt))
		require.NoError(t, writer.WriteField("stream", "true"))
		require.NoError(t, writer.WriteField("partial_images", "3"))
		part, err := writer.CreateFormFile("image", "input.png")
		require.NoError(t, err)
		_, err = part.Write([]byte("fake image"))
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		return c
	}

	convertAndReplay := func(t *testing.T, c *gin.Context, prompt string) {
		info := &relaycommon.RelayInfo{
			RelayMode: relayconstant.RelayModeImagesEdits,
		}
		request := dto.ImageRequest{
			Model:  "gpt-image-1",
			Prompt: prompt,
			Stream: common.GetPointer(true),
		}

		converted, err := (&Adaptor{}).ConvertImageRequest(c, info, request)
		require.NoError(t, err)
		convertedBody, ok := converted.(*bytes.Buffer)
		require.True(t, ok)

		replayedRequest := httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(convertedBody.Bytes()))
		replayedRequest.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
		require.NoError(t, replayedRequest.ParseMultipartForm(32<<20))

		require.Equal(t, "gpt-image-1", replayedRequest.PostForm.Get("model"))
		require.Equal(t, prompt, replayedRequest.PostForm.Get("prompt"))
		require.Equal(t, "true", replayedRequest.PostForm.Get("stream"))
		require.Equal(t, "3", replayedRequest.PostForm.Get("partial_images"))
		require.Len(t, replayedRequest.MultipartForm.File["image"], 1)

		file, err := replayedRequest.MultipartForm.File["image"][0].Open()
		require.NoError(t, err)
		defer file.Close()
		fileBytes, err := io.ReadAll(file)
		require.NoError(t, err)
		require.Equal(t, []byte("fake image"), fileBytes)
	}

	t.Run("with pre-parsed form", func(t *testing.T) {
		prompt := "edit this image"
		c := newMultipartContext(t, prompt)
		require.NoError(t, c.Request.ParseMultipartForm(32<<20))

		convertAndReplay(t, c, prompt)
	})

	t.Run("re-parses reusable body when form is missing", func(t *testing.T) {
		prompt := "edit without pre-parsed form"
		c := newMultipartContext(t, prompt)

		storage, err := common.GetBodyStorage(c)
		require.NoError(t, err)
		c.Request.Body = io.NopCloser(storage)
		c.Request.MultipartForm = nil
		c.Request.PostForm = nil

		convertAndReplay(t, c, prompt)
	})
}

// Exercise the JSON decode/copy/convert/encode path used by ImageHelper. Unknown
// fields are intentionally discarded, so GIF controls must be declared fields.
func TestConvertImageGenerationAnimationOptions(t *testing.T) {
	for _, tc := range []struct{ name, fields, want string }{
		{"custom", `,"output_format":"gif","frame_count":6,"frame_duration_ms":125`, `{"output_format":"gif","frame_count":6,"frame_duration_ms":125}`},
		{"explicit zero reaches upstream validation", `,"frame_count":0,"frame_duration_ms":0`, `{"frame_count":0,"frame_duration_ms":0}`},
		{"omitted", ``, `{}`},
		{"null uses upstream defaults", `,"frame_count":null,"frame_duration_ms":null`, `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var request dto.ImageRequest
			require.NoError(t, common.UnmarshalJsonStr(`{"model":"gpt-image-2.6-sunburst","prompt":"bouncing ball"`+tc.fields+`}`, &request))
			copied, err := common.DeepCopy(&request)
			require.NoError(t, err)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			info := &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeImagesGenerations}
			converted, err := (&Adaptor{}).ConvertImageRequest(c, info, *copied)
			require.NoError(t, err)
			raw, err := common.Marshal(converted)
			require.NoError(t, err)
			var fields map[string]any
			require.NoError(t, common.Unmarshal(raw, &fields))
			assert.Equal(t, "gpt-image-2.6-sunburst", fields["model"])
			assert.Equal(t, "bouncing ball", fields["prompt"])
			delete(fields, "model")
			delete(fields, "prompt")
			raw, err = common.Marshal(fields)
			require.NoError(t, err)
			assert.JSONEq(t, tc.want, string(raw))
		})
	}
	for _, fields := range []string{`"frame_count":"6"`, `"frame_count":2.5`, `"frame_duration_ms":true`, `"frame_duration_ms":20.5`} {
		var request dto.ImageRequest
		assert.Error(t, common.UnmarshalJsonStr(`{`+fields+`}`, &request))
	}
}
