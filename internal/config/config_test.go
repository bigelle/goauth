package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		Name     string
		CheckErr func(t *testing.T, err error)
		Want     *Config
	}{
		{
			Name:     "missing-use-defaults",
			CheckErr: wantNoErr,
			Want:     DefaultConfig(),
		},
		{
			Name:     "valid-config",
			CheckErr: wantNoErr,
			Want: &Config{
				Server: ServerConfig{
					Host: "100.0.100.0",
					Port: 6942,
				},
			},
		},
	}

	const testdata = "../../testdata/"

	var (
		relPath string
		absPath string
		err     error
		got     *Config
	)
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			relPath = filepath.Join(testdata, test.Name+".yaml")
			absPath, err = filepath.Abs(relPath)
			// If I ever expect an err, it wouldn't be here
			require.NoError(t, err)

			got, err = LoadConfig(absPath)
			test.CheckErr(t, err)

			assert.NotNil(t, got)
			assert.Equal(t, test.Want, got)
		})
	}
}

func wantNoErr(t *testing.T, err error) {
	t.Helper()
	require.NoError(t, err)
}

func wantErr(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
}
