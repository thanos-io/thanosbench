package blockgen

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
)

func Test_Everything(t *testing.T) {
	if err := runGenerator(); err != nil {
		t.Errorf("Failure: %v", err)
	}
}

// A really noddy quick test. Not sure it's even a good
// idea to have it.
func runGenerator() error {
	// set this to false to retain the output dir
	// for any manual examination etc.
	removeDir := true

	dir, err := ioutil.TempDir("", "thanos-data-test")
	if err != nil {
		return errors.Wrap(err, "create temp dir")
	}

	// delete temp dir if required
	defer func() {
		if removeDir {
			// ignore errors
			os.RemoveAll(dir)
		} else {
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "Output directory: %s\n", dir)
			fmt.Fprintf(os.Stderr, "       directory retained, delete manually\n")
		}
	}()

	// Generate 2 metrics from 3 targets.
	valProviderConfig := ValProviderConfig{
		MetricCount: 2,
		TargetCount: 3,
	}

	valProvider := NewValProvider(valProviderConfig)

	// Custom generator config to make it faster :)
	generatorConfig := DefaultGeneratorConfig(2 * time.Minute)
	generatorConfig.SampleInterval = 15 * time.Second
	generatorConfig.FlushInterval = 2 * time.Minute
	generator := NewGeneratorWithConfig(generatorConfig)

	// Create block writer to write to dir
	blockWriter, err := NewBlockWriter(dir)
	if err != nil {
		return err
	}

	// Go and hope for the best :)
	if err := generator.Generate(blockWriter, valProvider); err != nil {
		return err
	}

	return nil
}
