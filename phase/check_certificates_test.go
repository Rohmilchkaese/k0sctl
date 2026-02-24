package phase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseCertOutput(t *testing.T) {
	t.Run("valid certificate with CA", func(t *testing.T) {
		output := "notAfter=Dec 31 23:59:59 2035 GMT\nsubject=CN = kubernetes-ca"
		caOutput := "X509v3 Basic Constraints: critical\n    CA:TRUE"

		info := parseCertOutput("10.0.0.1", "/var/lib/k0s/pki/ca.crt", output, caOutput)

		require.NotNil(t, info)
		require.Equal(t, "10.0.0.1", info.Host)
		require.Equal(t, "/var/lib/k0s/pki/ca.crt", info.Path)
		require.Equal(t, "CN = kubernetes-ca", info.Subject)
		require.True(t, info.IsCA)
		require.Equal(t, 2035, info.Expiry.Year())
		require.Equal(t, time.December, info.Expiry.Month())
		require.Equal(t, 31, info.Expiry.Day())
	})

	t.Run("valid leaf certificate", func(t *testing.T) {
		output := "notAfter=Mar  5 10:30:00 2027 GMT\nsubject=CN = kube-apiserver"
		caOutput := ""

		info := parseCertOutput("10.0.0.1", "/var/lib/k0s/pki/server.crt", output, caOutput)

		require.NotNil(t, info)
		require.Equal(t, "CN = kube-apiserver", info.Subject)
		require.False(t, info.IsCA)
		require.Equal(t, 2027, info.Expiry.Year())
		require.Equal(t, time.March, info.Expiry.Month())
		require.Equal(t, 5, info.Expiry.Day())
	})

	t.Run("single digit day without leading space", func(t *testing.T) {
		output := "notAfter=Jan 2 08:00:00 2030 GMT\nsubject=CN = etcd-peer"
		caOutput := ""

		info := parseCertOutput("10.0.0.2", "/var/lib/k0s/pki/etcd/peer.crt", output, caOutput)

		require.NotNil(t, info)
		require.Equal(t, 2, info.Expiry.Day())
		require.Equal(t, time.January, info.Expiry.Month())
	})

	t.Run("double digit day", func(t *testing.T) {
		output := "notAfter=Nov 15 12:00:00 2028 GMT\nsubject=CN = front-proxy-client"
		caOutput := ""

		info := parseCertOutput("10.0.0.1", "/var/lib/k0s/pki/front-proxy-client.crt", output, caOutput)

		require.NotNil(t, info)
		require.Equal(t, 15, info.Expiry.Day())
	})

	t.Run("missing expiry returns nil", func(t *testing.T) {
		output := "subject=CN = broken"
		caOutput := ""

		info := parseCertOutput("10.0.0.1", "/var/lib/k0s/pki/broken.crt", output, caOutput)

		require.Nil(t, info)
	})

	t.Run("empty output returns nil", func(t *testing.T) {
		info := parseCertOutput("10.0.0.1", "/var/lib/k0s/pki/empty.crt", "", "")

		require.Nil(t, info)
	})

	t.Run("CA:TRUE detection in various formats", func(t *testing.T) {
		output := "notAfter=Dec 31 23:59:59 2035 GMT\nsubject=CN = test"

		info := parseCertOutput("10.0.0.1", "/test.crt", output, "CA:TRUE")
		require.NotNil(t, info)
		require.True(t, info.IsCA)

		info = parseCertOutput("10.0.0.1", "/test.crt", output, "CA:FALSE")
		require.NotNil(t, info)
		require.False(t, info.IsCA)
	})
}
