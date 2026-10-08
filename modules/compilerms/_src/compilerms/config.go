package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

var YAMLDecodeOptions []yaml.DecodeOption = []yaml.DecodeOption{
	yaml.Strict(),
	yaml.UseOrderedMap(),
}

type YAMLOrderedMap[T any] []struct {
	Key   string
	Value T
}

func (m *YAMLOrderedMap[T]) UnmarshalYAML(b []byte) error {
	var (
		order yaml.MapSlice
		typed map[string]T
		out   YAMLOrderedMap[T]
		item  yaml.MapItem
		key   string
		value T
		ok    bool
		err   error
	)

	err = yaml.UnmarshalWithOptions(b, &order, YAMLDecodeOptions...)
	if err != nil {
		return err
	}

	err = yaml.UnmarshalWithOptions(b, &typed, YAMLDecodeOptions...)
	if err != nil {
		return err
	}

	out = make(YAMLOrderedMap[T], 0, len(order))
	for _, item = range order {
		key, ok = item.Key.(string)
		if !ok {
			return fmt.Errorf(
				"YAMLOrderedMap[%T]: expected type string for key, got %T (%v)",
				*new(T),
				item.Key,
				item.Key,
			)
		}

		value, ok = typed[key]
		if !ok {
			panic(fmt.Errorf(
				"YAMLOrderedMap[%T]: value for key %q is somehow missing",
				*new(T),
				key,
			))
		}

		out = append(out, struct {
			Key   string
			Value T
		}{key, value})
	}

	*m = out

	return nil
}

type ConfigFlags struct {
	YAMLOrderedMap[any]
}

func (cfg *ConfigFlags) Strings() []string {
	var (
		item struct {
			Key   string
			Value any
		}

		args   []string
		value3 any
	)

	args = make([]string, 0, len(cfg.YAMLOrderedMap)*2)

	for _, item = range cfg.YAMLOrderedMap {
		switch {
		case strings.HasPrefix(item.Key, "-"):
			// noop
		case len(item.Key) == 1:
			item.Key = "-" + item.Key
		default:
			item.Key = "--" + item.Key
		}

		args = append(args, item.Key)

		switch value := item.Value.(type) {
		case bool:
			if !value {
				args = args[:len(args)-1]
			}
		case []any:
			args = args[:len(args)-1]

			for _, value2 := range value {
				switch value2 := value2.(type) {
				case []any:
					args = append(args, item.Key)

					for _, value3 = range value2 {
						args = append(args, fmt.Sprint(value3))
					}
				default:
					args = append(args, item.Key, fmt.Sprint(value2))
				}
			}
		default:
			args = append(args, fmt.Sprint(value))
		}
	}

	return args
}

type ConfigTest struct {
	Files map[string]string `yaml:"files"`
	Stdin string            `yaml:"stdin"`
}

type ConfigStep struct {
	SystemdRun ConfigFlags `yaml:"systemd-run"`

	Command struct {
		Name  string      `yaml:"name"`
		Flags ConfigFlags `yaml:"flags"`
		Args  []string    `yaml:"args"`
	} `yaml:"command"`
}

type ConfigLanguage struct {
	Enable bool                       `yaml:"enable"`
	Tests  map[string]ConfigTest      `yaml:"tests"`
	Steps  YAMLOrderedMap[ConfigStep] `yaml:"steps"`

	Files struct {
		Whitelist []string `yaml:"whitelist"`
	} `yaml:"files"`
}

type Config struct {
	Languages map[string]ConfigLanguage `yaml:"languages"`

	Global struct {
		SystemdRun ConfigFlags `yaml:"systemd-run"`

		Strace struct {
			Flags      ConfigFlags `yaml:"flags"`
			SystemdRun ConfigFlags `yaml:"systemd-run"`
		} `yaml:"strace"`

		Commands struct {
			Shlite     string `yaml:"shlite"`
			Strace     string `yaml:"strace"`
			SystemdRun string `yaml:"systemd-run"`
		}

		HTTPServer struct {
			Address           string        `yaml:"address"`
			IdleTimeout       time.Duration `yaml:"idleTimeout"`
			MaxBytesReader    int           `yaml:"maxBytesReader"`
			MaxHeaderBytes    int           `yaml:"maxHeaderBytes"`
			ReadHeaderTimeout time.Duration `yaml:"readHeaderTimeout"`
			ReadTimeout       time.Duration `yaml:"readTimeout"`
			WriteTimeout      time.Duration `yaml:"writeTimeout"`
		} `yaml:"httpServer"`

		Language struct {
			StepMaxBytesOutput int `yaml:"stepMaxBytesOutput"`
			StepMaxBytesStdin  int `yaml:"stepMaxBytesStdin"`
		} `yaml:"language"`

		WebSocket struct {
			ReadLimit  int           `yaml:"readLimit"`
			Timeout    time.Duration `yaml:"timeout"`
			SystemdRun ConfigFlags   `yaml:"systemd-run"`
		} `yaml:"websocket"`
	} `yaml:"global"`
}

func LoadConfig() (*Config, error) {
	var (
		path string
		buf  []byte
		cfg  Config
		err  error
	)

	path = os.Getenv("COMPILERMS_CONFIG_PATH")
	if path == "" {
		return nil, errors.New("$COMPILERMS_CONFIG_PATH is unset")
	}

	buf, err = os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.UnmarshalWithOptions(buf, &cfg, YAMLDecodeOptions...)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
