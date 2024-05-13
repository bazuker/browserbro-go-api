package client

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c, err := New("http://localhost:10001", nil)
		require.NoError(t, err)
		require.NotNil(t, c)
		assert.Equal(t, "http://localhost:10001/api/v1", c.addr)
		assert.NotNil(t, c.client)
		assert.Equal(t, 30.0, c.client.Timeout.Seconds())
	})

	t.Run("success with custom client and trailing slash", func(t *testing.T) {
		customClient := &http.Client{}
		c, err := New("http://localhost:10001/", customClient)
		require.NoError(t, err)
		require.NotNil(t, c)
		assert.Equal(t, customClient, c.client)
	})

	t.Run("empty server address", func(t *testing.T) {
		c, err := New("", nil)
		require.Error(t, err)
		require.Nil(t, c)
	})
}

func TestClient_Plugins(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := mockServer(t, http.StatusOK, `{"plugins":["plugin1","plugin2"]}`)
		defer server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		plugins, err := c.Plugins()
		require.NoError(t, err)
		assert.Equal(t, []string{"plugin1", "plugin2"}, plugins)
	})

	t.Run("client error", func(t *testing.T) {
		server := mockServer(t, http.StatusOK, "")
		server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		_, err = c.Plugins()
		require.ErrorContains(t, err, "failed to fetch plugins:")
	})

	t.Run("server error", func(t *testing.T) {
		server := mockServer(t, http.StatusInternalServerError, "")
		defer server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		plugins, err := c.Plugins()
		require.EqualError(t, err, "unexpected response status: 500 Internal Server Error")
		require.Nil(t, plugins)
	})

	t.Run("invalid server response body", func(t *testing.T) {
		server := mockServer(t, http.StatusOK, "")
		defer server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		results, err := c.Plugins()
		require.ErrorContains(t, err, "failed to decode plugins:")
		require.Nil(t, results)
	})
}

func TestClient_RunPlugin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := mockServer(t, http.StatusOK, `{"plugin1": {"key": "value"}}`)
		defer server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		results, err := c.RunPlugin("plugin1", map[string]any{"my": "params"})
		require.NoError(t, err)
		assert.Equal(
			t,
			map[string]any{
				"plugin1": map[string]any{
					"key": "value",
				},
			},
			results,
		)
	})

	t.Run("client error", func(t *testing.T) {
		server := mockServer(t, http.StatusOK, "")
		server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		_, err = c.RunPlugin("plugin1", nil)
		require.ErrorContains(t, err, "failed to run plugin:")
	})

	t.Run("server error", func(t *testing.T) {
		server := mockServer(t, http.StatusInternalServerError, `{"message": "something went wrong"}`)
		defer server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		results, err := c.RunPlugin("plugin1", nil)
		require.EqualError(
			t,
			err,
			"unexpected response status: 500 Internal Server Error; message: something went wrong",
		)
		require.Nil(t, results)
	})

	t.Run("invalid server response body", func(t *testing.T) {
		server := mockServer(t, http.StatusOK, "")
		defer server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		results, err := c.RunPlugin("plugin1", nil)
		require.ErrorContains(t, err, "failed to decode plugin output:")
		require.Nil(t, results)
	})
}

func TestClient_DownloadFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := mockServer(t, http.StatusOK, "file content")
		defer server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		content, err := c.DownloadFile("file1")
		require.NoError(t, err)
		assert.Equal(t, "file content", string(content))
	})

	t.Run("client error", func(t *testing.T) {
		server := mockServer(t, http.StatusOK, "")
		server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		_, err = c.DownloadFile("file1")
		require.ErrorContains(t, err, "failed to download file:")
	})

	t.Run("server error", func(t *testing.T) {
		server := mockServer(t, http.StatusInternalServerError, "")
		defer server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		content, err := c.DownloadFile("file1")
		require.EqualError(t, err, "unexpected response status: 500 Internal Server Error")
		require.Nil(t, content)
	})
}

func TestClient_DeleteFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := mockServer(t, http.StatusOK, "")
		defer server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		err = c.DeleteFile("file1")
		require.NoError(t, err)
	})

	t.Run("client error", func(t *testing.T) {
		server := mockServer(t, http.StatusOK, "")
		server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		err = c.DeleteFile("file1")
		require.ErrorContains(t, err, "failed to delete file:")
	})

	t.Run("server error", func(t *testing.T) {
		server := mockServer(t, http.StatusInternalServerError, "")
		defer server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		err = c.DeleteFile("file1")
		require.EqualError(t, err, "unexpected response status: 500 Internal Server Error")
	})
}

func TestClient_Healthcheck(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := mockServer(t, http.StatusOK, "")
		defer server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		err = c.Healthcheck()
		require.NoError(t, err)
	})

	t.Run("client error", func(t *testing.T) {
		server := mockServer(t, http.StatusOK, "")
		server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		err = c.Healthcheck()
		require.ErrorContains(t, err, "failed to perform health check:")
	})

	t.Run("server error", func(t *testing.T) {
		server := mockServer(t, http.StatusInternalServerError, "")
		defer server.Close()

		c, err := New(server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, c)

		err = c.Healthcheck()
		require.EqualError(t, err, "unexpected response status: 500 Internal Server Error")
	})
}

func mockServer(t testing.TB, status int, body string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = io.Copy(w, bytes.NewBufferString(body))
	}))
}
