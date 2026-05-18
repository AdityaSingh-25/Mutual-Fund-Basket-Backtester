package config

import "testing"

func TestConfigValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "complete config",
			cfg:     Config{DBUrl: "postgres://x", RedisUrl: "redis://y"},
			wantErr: false,
		},
		{
			name:    "port optional",
			cfg:     Config{DBUrl: "postgres://x", RedisUrl: "redis://y"},
			wantErr: false,
		},
		{
			name:    "missing db url",
			cfg:     Config{RedisUrl: "redis://y"},
			wantErr: true,
		},
		{
			name:    "missing redis url",
			cfg:     Config{DBUrl: "postgres://x"},
			wantErr: true,
		},
		{
			name:    "missing both",
			cfg:     Config{},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
