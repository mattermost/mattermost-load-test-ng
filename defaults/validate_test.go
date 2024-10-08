package defaults

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	type serverConfiguration struct {
		URL           string `default:"http://localhost:8065" validate:"url"`
		Email         string `default:"sysadmin@sample.mattermost.com" validate:"email"`
		AdminPassword string `default:"Sys@dmin-sample1" validate:"notempty"`
		NumTeams      int    `default:"2" validate:"range:(0,]"`
		InitialUsers  int    `default:"0" validate:"range:[0,$MaxUsers]"`
		MaxUsers      int    `default:"1000" validate:"range:(0,]"`
		LogLevel      string `default:"ERROR" validate:"oneof:{TRACE, INFO, WARN, ERROR}"`
		S3URI         string `default:"" validate:"s3uri"`
		LicenseFile   string `default:"" validate:"emptyorfile"`
	}

	t.Run("happy path", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		err := Validate(cfg)
		require.NoError(t, err)
	})

	t.Run("happy path with ptr", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		err := Validate(&cfg)
		require.NoError(t, err)
	})

	t.Run("out of given range", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.InitialUsers = -1
		err := Validate(cfg)
		require.Error(t, err)
	})

	t.Run("invalid url", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.URL = "localhost"
		err := Validate(cfg)
		require.Error(t, err)
	})

	t.Run("invalid email", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.Email = "some_text"
		err := Validate(cfg)
		require.Error(t, err)
	})

	t.Run("not one of", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.LogLevel = "DEBUG"
		valids := []string{"TRACE", "INFO", "WARN", "ERROR"}
		err := Validate(cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), fmt.Sprintf("% q", valids))
	})

	t.Run("invalid field reference", func(t *testing.T) {
		type invalidField struct {
			InitialUsers int `default:"0" validate:"range:[0,$MaxActiveUsers]"`
		}

		var cfg invalidField
		Set(&cfg)

		err := Validate(cfg)
		require.Error(t, err)
	})

	t.Run("call IsValid method", func(t *testing.T) {
		var cfg testConfig

		err := Set(&cfg)
		require.NoError(t, err)

		err = Validate(&cfg)
		require.Error(t, err)
	})

	t.Run("invalid struct tags", func(t *testing.T) {
		type missingColonConfig struct {
			InitialUsers int `default:"0" validate:"oneof:{test,text}"`
		}

		var cfg1 missingColonConfig
		Set(&cfg1)

		err := Validate(cfg1)
		require.Error(t, err, "should fail on missing colon")

		type invalidField struct {
			Name         string `default:"test" validate:"notempty"`
			InitialUsers int    `default:"0" validate:"range:[0,$Name]"`
		}

		var cfg2 invalidField
		Set(&cfg2)

		err = Validate(cfg2)
		require.Error(t, err, "should fail on wrong type")

		type invalidRange struct {
			InitialUsers int `default:"-1" validate:"range:[,(0,123),]"`
		}

		var cfg3 invalidRange
		Set(&cfg3)

		err = Validate(cfg3)
		require.Error(t, err, "should fail on wrong range declaration")
	})

	t.Run("notempty on slices", func(t *testing.T) {
		type testStruct struct {
			Slice []struct{} `validate:"notempty"`
		}

		t1 := testStruct{make([]struct{}, 0)}
		t2 := testStruct{make([]struct{}, 1)}

		err := Validate(t1)
		require.Error(t, err)
		err = Validate(t2)
		require.NoError(t, err)
	})

	t.Run("valid S3 URI", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.S3URI = "s3://test.s3bucket"

		err := Validate(&cfg)
		require.NoError(t, err)
	})

	t.Run("invalid S3 URI", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.S3URI = "not an url"

		err := Validate(&cfg)
		require.Error(t, err)
	})

	t.Run("invalid S3 URI scheme", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.S3URI = "https://validurl.com/but/wrong/scheme"

		err := Validate(&cfg)
		require.Error(t, err)
	})

	t.Run("ip validation", func(t *testing.T) {
		type ipCfg struct {
			IP string `validate:"ip"`
		}

		testCases := []struct {
			name        string
			ip          string
			expectedErr bool
		}{
			{"valid ipv4", "192.168.1.1", false},
			{"valid ipv6", "2001:db8::8a2e:370:7334", false},
			{"valid ip, but it contains port", "192.168.1.1:8065", true},
			{"invalid ip", "ceci n'est pas une ip", true},
		}

		for _, test := range testCases {
			t.Run(test.name, func(t *testing.T) {
				err := Validate(ipCfg{test.ip})
				if test.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		}

	})

	t.Run("each validation", func(t *testing.T) {
		t.Run("invalid type for each field", func(t *testing.T) {
			type cfg struct {
				Invalidtype int `validate:"each:ip"`
			}

			err := Validate(cfg{})
			require.Error(t, err)
		})

		t.Run("invalid validation type for each field", func(t *testing.T) {
			type cfg struct {
				Strings []string `validate:"each:invalid"`
			}

			err := Validate(cfg{[]string{"something"}})
			require.Error(t, err)
		})

		t.Run("each with a slice of strings", func(t *testing.T) {
			type cfg struct {
				Strings []string `validate:"each:url"`
			}

			t.Run("empty slice is valid", func(t *testing.T) {
				err := Validate(cfg{[]string{}})
				require.NoError(t, err)
			})

			t.Run("valid non-empty slice", func(t *testing.T) {
				err := Validate(cfg{[]string{
					"http://url.tld",
					"https://example.com",
				}})
				require.NoError(t, err)
			})

			t.Run("invalid and valid values in slice", func(t *testing.T) {
				err := Validate(cfg{[]string{
					"http://url.tld",
					"an invalid url",
				}})
				require.Error(t, err)
			})

			t.Run("only invalid values in slice", func(t *testing.T) {
				err := Validate(cfg{[]string{
					"an invalid url",
					"another invalid url",
				}})
				require.Error(t, err)
			})
		})

		t.Run("each with a slice of ints and range validation", func(t *testing.T) {
			type cfg struct {
				Ints []int `validate:"each:range:[2,10]"`
			}

			t.Run("empty slice is valid", func(t *testing.T) {
				err := Validate(cfg{[]int{}})
				require.NoError(t, err)
			})

			t.Run("valid non-empty slice", func(t *testing.T) {
				err := Validate(cfg{[]int{2, 3, 4, 5, 6, 7, 8, 9, 10}})
				require.NoError(t, err)
			})

			t.Run("invalid and valid values in slice", func(t *testing.T) {
				err := Validate(cfg{[]int{-654, 2}})
				require.Error(t, err)
			})

			t.Run("only invalid values in slice", func(t *testing.T) {
				err := Validate(cfg{[]int{-10, 1000}})
				require.Error(t, err)
			})
		})

		t.Run("each with a slice of strings and oneof validation", func(t *testing.T) {
			type cfg struct {
				Strings []string `validate:"each:oneof:{TRACE,DEBUG,INFO}"`
			}

			t.Run("empty slice is valid", func(t *testing.T) {
				err := Validate(cfg{[]string{}})
				require.NoError(t, err)
			})

			t.Run("valid non-empty slice", func(t *testing.T) {
				err := Validate(cfg{[]string{
					"TRACE",
					"DEBUG",
				}})
				require.NoError(t, err)
			})

			t.Run("invalid and valid values in slice", func(t *testing.T) {
				err := Validate(cfg{[]string{
					"TRACE",
					"invalida value",
				}})
				require.Error(t, err)
			})

			t.Run("only invalid values in slice", func(t *testing.T) {
				err := Validate(cfg{[]string{
					"invalid value",
					"another invalid value",
				}})
				require.Error(t, err)
			})

		})

	})

	t.Run("prefix validation", func(t *testing.T) {
		type cfg struct {
			PrefixedValue string `validate:"prefix:start"`
		}

		cases := []struct {
			name        string
			prefix      string
			expectedErr bool
		}{
			{
				"just the prefix",
				"start",
				false,
			},
			{
				"prefix and more stuff",
				"starting",
				false,
			},
			{
				"different casing",
				"StarT",
				true,
			},
			{
				"different prefix",
				"nostart",
				true,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err := Validate(cfg{tc.prefix})
				if tc.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})

	t.Run("wrong prefix tag", func(t *testing.T) {
		type cfg struct {
			PrefixedValue string `validate:"prefixasdf:start"`
		}
		err := Validate(cfg{"start"})
		require.Error(t, err)
	})

	t.Run("empty emptyorfile", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.LicenseFile = ""

		err := Validate(&cfg)
		require.NoError(t, err)
	})

	t.Run("valid non-empty emptyorfile", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		// We need a file that exists, so let's use the path to the executable running
		// this test, which is guaranteed to exist when the test is running
		f, err := os.Executable()
		require.NoError(t, err)
		cfg.LicenseFile = f

		err = Validate(&cfg)
		require.NoError(t, err)
	})

	t.Run("invalid emptyorfile", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.LicenseFile = "/invalid/path/to/inexistent/file"

		err := Validate(&cfg)
		require.Error(t, err)
	})

	t.Run("not a path for emptyorfile", func(t *testing.T) {
		var cfg serverConfiguration
		Set(&cfg)

		cfg.LicenseFile = "not a file"

		err := Validate(&cfg)
		require.Error(t, err)
	})
}

type testConfig struct {
	Name string `default:"test" validate:"notempty"`
}

func (c *testConfig) IsValid() error {
	return errors.New("some error")
}

func TestValidateComparisonConfig(t *testing.T) {
	type LoadTestType string
	type DatabaseEngine string
	type LoadTestConfig struct {
		Type            LoadTestType   `validate:"oneof:{bounded,unbounded}"`
		DBEngine        DatabaseEngine `validate:"oneof:{mysql,postgresql}"`
		DBDumpURL       string
		S3BucketDumpURI string `default:"" validate:"s3uri"`
		NumUsers        int    `default:"0" validate:"range:[0,]"`
		Duration        string
	}

	config := LoadTestConfig{
		Type:            "bounded",
		DBEngine:        "postgresql",
		DBDumpURL:       "file:///home/ubuntu/dump.sql",
		S3BucketDumpURI: "s3://test.bucket/subdir",
		NumUsers:        100,
		Duration:        "1h",
	}

	t.Run("valid config", func(t *testing.T) {
		err := Validate(config)
		require.NoError(t, err)
	})
}
