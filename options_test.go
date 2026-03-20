package arigo

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionClone(t *testing.T) {
	original := &Options{
		AllProxy:             "http://proxy.example.com:8080",
		AllowOverwrite:       true,
		BTMaxPeers:           55,
		Header:               Header{"User-Agent": "curl/8.0"},
		EnableHTTPPipelining: true,
	}

	cloned := original.Clone()

	require.NotNil(t, cloned)
	assert.Equal(t, original, cloned)

	cloned.Header["User-Agent"] = "wget/1.0"
	assert.Equal(t, "curl/8.0", original.Header["User-Agent"])
}

func TestOptionFormat(t *testing.T) {
	data := []byte(`{
		"allow-overwrite": "false",
		"allow-piece-length-change": "false",
		"always-resume": "true",
		"async-dns": "true"
	}`)

	var options Options
	assert.NoError(t, json.Unmarshal(data, &options), "Couldn't unmarshal JSON")

	assert.Equal(t, Options{
		AllowOverwrite:         false,
		AllowPieceLengthChange: false,
		AlwaysResume:           true,
		AsyncDNS:               true,
	}, options)
}

func TestHeaders_RoundTrip(t *testing.T) {
	original := Header{
		"User-Agent": "curl/8.0",
		"Accept":     "*/*",
	}

	b, err := json.Marshal(original)
	require.NoError(t, err)

	// Output must be a JSON array, not an object.
	require.Equal(t, byte('['), b[0], "marshaled form must be a JSON array")

	var decoded Header
	require.NoError(t, json.Unmarshal(b, &decoded))
	assert.Equal(t, original, decoded)
}

// ── marshal ───────────────────────────────────────────────────────────────────

func TestHeaders_Marshal_ProducesArray(t *testing.T) {
	h := Header{"Authorization": "Bearer tok"}
	b, err := json.MarshalIndent(h, "", "  ")
	require.NoError(t, err)

	want := `["Authorization: Bearer tok"]`
	assert.JSONEq(t, want, string(b))
}

func TestHeaders_Marshal_Empty(t *testing.T) {
	b, err := json.Marshal(Header{})
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, string(b))
}

func TestHeaders_Marshal_MultipleHeaders(t *testing.T) {
	h := Header{
		"User-Agent":   "curl/8.0",
		"Accept":       "*/*",
		"Content-Type": "application/json",
	}
	b, err := json.Marshal(h)
	require.NoError(t, err)

	// Unmarshal into a slice to verify count & format without caring about order.
	var lines []string
	require.NoError(t, json.Unmarshal(b, &lines))
	assert.Len(t, lines, 3)
	assert.ElementsMatch(t, []string{
		"User-Agent: curl/8.0",
		"Accept: */*",
		"Content-Type: application/json",
	}, lines)
}

// ── unmarshal ─────────────────────────────────────────────────────────────────

func TestHeaders_Unmarshal_Basic(t *testing.T) {
	raw := `["User-Agent: curl/8.0", "Accept: */*"]`
	var h Header
	require.NoError(t, json.Unmarshal([]byte(raw), &h))
	assert.Equal(t, Header{"User-Agent": "curl/8.0", "Accept": "*/*"}, h)
}

func TestHeaders_Unmarshal_ValueWithColon(t *testing.T) {
	// SplitN(..., 2) must keep colons in the value intact.
	raw := `["Authorization: Bearer abc:def:ghi"]`
	var h Header
	require.NoError(t, json.Unmarshal([]byte(raw), &h))
	assert.Equal(t, "Bearer abc:def:ghi", h["Authorization"])
}

func TestHeaders_Unmarshal_TrimsWhitespace(t *testing.T) {
	raw := `["  Content-Type  :  text/html  "]`
	var h Header
	require.NoError(t, json.Unmarshal([]byte(raw), &h))
	assert.Equal(t, "text/html", h["Content-Type"])
}

func TestHeaders_Unmarshal_EmptyValue(t *testing.T) {
	// "Key:" → value is an empty string.
	raw := `["X-Empty:"]`
	var h Header
	require.NoError(t, json.Unmarshal([]byte(raw), &h))
	assert.Equal(t, "", h["X-Empty"])
}

func TestHeaders_Unmarshal_DuplicateKeys_LastWins(t *testing.T) {
	raw := `["X-Foo: first", "X-Foo: second"]`
	var h Header
	require.NoError(t, json.Unmarshal([]byte(raw), &h))
	assert.Equal(t, "second", h["X-Foo"])
}

func TestHeaders_Unmarshal_Empty(t *testing.T) {
	var h Header
	require.NoError(t, json.Unmarshal([]byte(`[]`), &h))
	assert.Empty(t, h)
}

func TestHeaders_Unmarshal_InitializesNilMap(t *testing.T) {
	// Pointer receiver must initialise a nil map before writing.
	h := new(Header) // *h is nil
	require.NoError(t, json.Unmarshal([]byte(`["X-Init: yes"]`), h))
	assert.Equal(t, "yes", (*h)["X-Init"])
}

// ── error cases ───────────────────────────────────────────────────────────────

func TestHeaders_Unmarshal_Error_MissingColon(t *testing.T) {
	var h Header
	err := json.Unmarshal([]byte(`["invalid-header"]`), &h)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid header format")
}

func TestHeaders_Unmarshal_Error_EmptyKey(t *testing.T) {
	var h Header
	err := json.Unmarshal([]byte(`[": value-only"]`), &h)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty header key")
}

func TestHeaders_Unmarshal_Error_JSONObject(t *testing.T) {
	// Expects an array, not an object.
	var h Header
	err := json.Unmarshal([]byte(`{"User-Agent": "curl/8.0"}`), &h)
	assert.Error(t, err)
}

func TestHeaders_Unmarshal_Error_InvalidJSON(t *testing.T) {
	var h Header
	err := json.Unmarshal([]byte(`not-json`), &h)
	assert.Error(t, err)
}

func TestHeaders_Unmarshal_Error_NullJSON(t *testing.T) {
	var h Header
	// null is valid JSON but cannot be unmarshalled into []string
	err := json.Unmarshal([]byte(`null`), &h)
	// null decodes to nil slice, so no entries — no error expected here
	assert.NoError(t, err)
}
