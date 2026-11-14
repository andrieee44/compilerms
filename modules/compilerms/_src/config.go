package main

import (
	"encoding/json/v2"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/go-jsonnet"
)

type duration struct {
	time.Duration
}

func (d duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

func (d *duration) UnmarshalText(b []byte) error {
	var err error

	d.Duration, err = time.ParseDuration(string(b))

	return err
}

type testConfig struct {
	Input  string            `json:"input"`
	Output string            `json:"output"`
	Files  map[string]string `json:"files"   validate:"required"`
}

type stepConfig struct {
	DisplayName    string `json:"displayName"     validate:"required"`
	Name           string `json:"name"            validate:"required"`
	MaxBytesInput  int    `json:"maxBytesInput"`
	MaxBytesOutput int    `json:"maxBytesOutput"`

	Args struct {
		Prod  []string `json:"prod"  validate:"required"`
		Debug []string `json:"debug" validate:"required"`
	} `json:"args"`
}

type cmdConfig struct {
	Name string   `json:"name" validate:"required"`
	Args []string `json:"args" validate:"required"`
}

type langConfig struct {
	AllowedFiles []string              `json:"allowedFiles" validate:"required"`
	Versions     map[string]*cmdConfig `json:"versions"     validate:"required,min=1,dive,required"`
	Compiler     *stepConfig           `json:"compiler"`

	Tests struct {
		Live  map[string]*testConfig `json:"live"  validate:"required,min=1,dive,required"`
		Batch map[string]*testConfig `json:"batch" validate:"required,min=1,dive,required"`
	} `json:"tests"`

	Program struct {
		Live  *stepConfig `json:"live"  validate:"required"`
		Batch *stepConfig `json:"batch" validate:"required"`
	} `json:"program"`
}

type httpConfig struct {
	Address           string   `json:"address"           validate:"hostname_port"`
	MaxBytesReader    int64    `json:"maxBytesReader"    validate:"required"`
	MaxHeaderBytes    int      `json:"maxHeaderBytes"    validate:"required"`
	IdleTimeout       duration `json:"idleTimeout"       validate:"required"`
	ReadHeaderTimeout duration `json:"readHeaderTimeout" validate:"required"`
	ReadTimeout       duration `json:"readTimeout"       validate:"required"`
	WriteTimeout      duration `json:"writeTimeout"      validate:"required"`
	AllowedOrigins    []string `json:"allowedOrigins"    validate:"required,min=1"`
}

type websocketConfig struct {
	ReadLimit      int64    `json:"readLimit"      validate:"required"`
	Timeout        duration `json:"timeout"        validate:"required"`
	OriginPatterns []string `json:"originPatterns" validate:"required,min=1"`
}

type config struct {
	HTTP            *httpConfig            `json:"http"            validate:"required"`
	WS              *websocketConfig       `json:"ws"              validate:"required"`
	MaxRunningSteps int64                  `json:"maxRunningSteps" validate:"required"`
	Languages       map[string]*langConfig `json:"languages"       validate:"required,min=1,dive,required"`
}

func defaultConfig() *config {
	return &config{
		HTTP: &httpConfig{
			Address:           "localhost:8080",
			MaxBytesReader:    16 << 10,
			MaxHeaderBytes:    4 << 10,
			IdleTimeout:       duration{30 * time.Second},
			ReadHeaderTimeout: duration{5 * time.Second},
			ReadTimeout:       duration{10 * time.Second},
			WriteTimeout:      duration{30 * time.Second},
			AllowedOrigins:    []string{"*"},
		},

		WS: &websocketConfig{
			ReadLimit:      16 << 10,
			Timeout:        duration{180 * time.Second},
			OriginPatterns: []string{"*"},
		},

		MaxRunningSteps: 4,
		Languages:       nil,
	}
}

func newConfig(path string) (*config, error) {
	var (
		cfg  *config
		data string
		err  error
	)

	cfg = defaultConfig()

	data, err = jsonnet.MakeVM().EvaluateFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(data), cfg, json.RejectUnknownMembers(true))
	if err != nil {
		return nil, err
	}

	return cfg, validator.New(
		validator.WithRequiredStructEnabled(),
		validator.WithTagNameFuncBlankOmit(),
	).Struct(cfg)
}
