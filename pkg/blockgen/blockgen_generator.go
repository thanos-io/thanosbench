package blockgen

import (
	"time"

	"github.com/pkg/errors"
)

// GeneratorConfig configures the behaviour of block generator.
type GeneratorConfig struct {
	// Retention is the time interval for which to generate data, e.g.
	// 8days = 8 * 24 * time.Hour. This is how much time back from `startTime`
	// the metrics will go. Retention should be multiples of `FlushInterval`.
	Retention time.Duration

	// StartTime is the time from which to generate metrics. The metrics
	// are generated for the window [StartTime-Retention, StartTime].
	//
	// Good default value for this is time.Now() but may want to use some fixed
	// value to make it easier to write repeatable queries for the data later.
	StartTime time.Time

	// SampleInterval is the interval between samples, 15s is default for Prometheus.
	SampleInterval time.Duration

	// FlushInterval is the interval at which blocks are written to disk.
	// These are usually 2h.
	// FlushInterval should be multiples of `SampleInterval`.
	//
	// NOTE: Flush is generally slow.
	// Consider tuning this if you have little data or a lot of data.
	FlushInterval time.Duration
}

// DefaultGeneratorConfig is the default configuration with specified retention.
func DefaultGeneratorConfig(retention time.Duration) GeneratorConfig {
	return GeneratorConfig{
		Retention:      retention,
		StartTime:      time.Now(),
		SampleInterval: 15 * time.Second,
		FlushInterval:  2 * time.Hour,
	}
}

// NewGenerator creates a generator with specified retention and default config,
// see `DefaultGeneratorConfig()`.
// Retention is the time period for which data will be generated.
func NewGenerator(retention time.Duration) Generator {
	config := DefaultGeneratorConfig(retention)

	return &generator{
		config: config,
	}
}

// NewGeneratorWithConfig creates a generator with user-supplied config.
func NewGeneratorWithConfig(config GeneratorConfig) Generator {
	return &generator{
		config: config,
	}
}

// generator is implementation of Generator.
type generator struct {
	config GeneratorConfig
}

// Generate implements Generator interface.
func (g *generator) Generate(writer Writer, valGenerators ...ValProvider) error {
	c := &g.config

	// Basic sanity checks.
	if c.Retention <= 0 {
		return errors.New("retention must be positive duration")
	}

	if c.SampleInterval <= 0 {
		return errors.New("sampleInterval must be positive duration")
	}

	if c.FlushInterval <= 0 {
		return errors.New("flushInterval must be positive duration")
	}

	// TODO(ppanyukov): do we really need this?
	// Make sure flushInterval is exactly multiples of sampleInterval.
	// This is something to do with how TSDB is particular to block
	// sizes etc, ask Bartek (:
	// Ditto for flushInterval vs retention, as we want to produce full blocks.
	if c.FlushInterval%c.SampleInterval != 0 {
		return errors.New("flushInterval must be multiples of sampleInterval")
	}
	if c.Retention%c.FlushInterval != 0 {
		return errors.New("retention must be multiples of flushInterval")
	}

	// write to TSDB from oldest to newest.
	maxt := c.StartTime
	mint := maxt.Add(-1 * c.Retention)

	// keep hold of last flush time so we flush at regular intervals.
	elapsed := time.Duration(0)

	for t := mint; !t.After(maxt); t = t.Add(c.SampleInterval) {
		now := t

		// grab values form generators, timestamp them and shove to the writer.
		for _, generator := range valGenerators {
			c := generator.Next()

			for val := range c {
				if err := writer.Write(now, val); err != nil {
					return errors.Wrap(err, "writer.Write")
				}
			}
		}

		elapsed += c.SampleInterval

		// Flush to disk when written enough data.
		if elapsed >= c.FlushInterval {
			if err := writer.Flush(); err != nil {
				return errors.Wrap(err, "writer.Flush")
			}

			elapsed = 0
		}
	}

	// NOTE: no defer write.Flush on purpose.
	if err := writer.Flush(); err != nil {
		return errors.Wrap(err, "last writer.Flush")
	}

	return nil
}
