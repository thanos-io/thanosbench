package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"

	"github.com/fatih/structtag"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/thanos-io/thanosbench/pkg/blockgen"
	"github.com/thanos-io/thanosbench/pkg/walgen"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	app := kingpin.New(filepath.Base(os.Args[0]), "Config examples generator.")
	app.HelpFlag.Short('h')
	outputDir := app.Flag("output-dir", "Output directory for generated examples.").String()

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	if _, err := app.Parse(os.Args[1:]); err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}

	typ := "blockspec"
	if err := generate([]blockgen.BlockSpec{{}}, typ, *outputDir); err != nil {
		level.Error(logger).Log("msg", "failed to generate", "type", typ, "err", err)
		os.Exit(1)
	}

	typ = "walgen"
	// TODO(bwplotka): Fill things automatically
	if err := generate(walgen.Config{InputSeries: []walgen.Series{
		{Result: walgen.QueryData{Result: model.Vector{&model.Sample{}}}}},
	}, typ, *outputDir); err != nil {
		level.Error(logger).Log("msg", "failed to generate", "type", typ, "err", err)
		os.Exit(1)
	}
	logger.Log("msg", "success")
}

func generate(obj interface{}, typ string, outputDir string) error {
	// We forbid omitempty option. This is for simplification for doc generation.
	if err := checkForOmitEmptyTagOption(obj); err != nil {
		return errors.Wrap(err, "invalid type")
	}

	out, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(outputDir, fmt.Sprintf("config_%s.txt", typ)), out, os.ModePerm)
}

func checkForOmitEmptyTagOption(obj interface{}) error {
	return checkForOmitEmptyTagOptionRec(reflect.ValueOf(obj))
}

func checkForOmitEmptyTagOptionRec(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			structField := v.Type().Field(i).Tag
			if structField == "" {
				continue
			}

			tags, err := structtag.Parse(string(structField))
			if err != nil {
				return errors.Wrapf(err, "%s: failed to parse tag %q", v.Type().Field(i).Name, v.Type().Field(i).Tag)
			}

			tag, err := tags.Get("yaml")
			if err != nil {
				return errors.Wrapf(err, "%s: failed to get tag %q", v.Type().Field(i).Name, v.Type().Field(i).Tag)
			}

			for _, opts := range tag.Options {
				if opts == "omitempty" {
					return errors.Errorf("omitempty is forbidden for config, but spotted on field '%s'", v.Type().Field(i).Name)
				}
			}

			if err := checkForOmitEmptyTagOptionRec(v.Field(i)); err != nil {
				return errors.Wrapf(err, "%s", v.Type().Field(i).Name)
			}
		}

	case reflect.Ptr:
		return errors.New("nil pointers are not allowed in configuration")

	case reflect.Interface:
		return checkForOmitEmptyTagOptionRec(v.Elem())
	}

	return nil
}
