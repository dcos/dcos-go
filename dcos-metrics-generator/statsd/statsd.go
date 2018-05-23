// Copyright (c) 2018 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package statsd

import (
	"fmt"
	"math"
	"time"

	log "github.com/sirupsen/logrus"
	statsd "github.com/smira/go-statsd.git"
	tally "github.com/uber-go/tally"
)

type StatsdReporter struct {
	client  *statsd.Client
	options *ReporterOptions
}

func NewStatsdReporter(addr string, options ...Option) tally.StatsReporter {
	reporterOptions := &ReporterOptions{
		TagFormat: TAG_FORMAT_NONE,
	}

	clientOptions := []statsd.Option{}
	for _, option := range options {
		clientOptions = append(clientOptions, option(reporterOptions))
	}

	client := statsd.NewClient(addr, clientOptions...)
	if client == nil {
		log.Errorf("Unable to initialize StatsD client.")
		return nil
	}

	return &StatsdReporter{
		client:  client,
		options: reporterOptions,
	}
}

func (r *StatsdReporter) ReportCounter(name string, tags map[string]string, value int64) {
	r.client.Incr(name, value, r.formatTags(tags)...)
}

func (r *StatsdReporter) ReportGauge(name string, tags map[string]string, value float64) {
	r.client.FGauge(name, value, r.formatTags(tags)...)
}

func (r *StatsdReporter) ReportTimer(name string, tags map[string]string, interval time.Duration) {
	r.client.PrecisionTiming(name, interval, r.formatTags(tags)...)
}

func (r *StatsdReporter) ReportHistogramValueSamples(
	name string,
	tags map[string]string,
	buckets tally.Buckets,
	bucketLowerBound,
	bucketUpperBound float64,
	samples int64,
) {
	name = fmt.Sprintf(
		"%s.%s-%s",
		name,
		r.valueBucketString(bucketLowerBound),
		r.valueBucketString(bucketUpperBound))

	r.client.Incr(name, samples, r.formatTags(tags)...)
}

func (r *StatsdReporter) ReportHistogramDurationSamples(
	name string,
	tags map[string]string,
	buckets tally.Buckets,
	bucketLowerBound,
	bucketUpperBound time.Duration,
	samples int64,
) {

	name = fmt.Sprintf(
		"%s.%s-%s",
		name,
		r.durationBucketString(bucketLowerBound),
		r.durationBucketString(bucketUpperBound))

	r.client.Incr(name, samples, r.formatTags(tags)...)
}

func (r *StatsdReporter) Capabilities() tally.Capabilities {
	return r
}

func (r *StatsdReporter) Reporting() bool {
	return true
}

func (r *StatsdReporter) Tagging() bool {
	switch r.options.TagFormat {
	case TAG_FORMAT_INFLUX:
	case TAG_FORMAT_DATADOG:
		return true
	}
	return false
}

func (r *StatsdReporter) Flush() {
	// no-op
}

func (r *StatsdReporter) formatTags(tags map[string]string) []statsd.Tag {
	ts := []statsd.Tag{}
	if r.Tagging() {
		for k, v := range tags {
			ts = append(ts, statsd.StringTag(k, v))
		}
	}
	return ts
}

func (r *StatsdReporter) valueBucketString(upperBound float64) string {
	if upperBound == math.MaxFloat64 {
		return "infinity"
	}
	if upperBound == -math.MaxFloat64 {
		return "-infinity"
	}
	return fmt.Sprintf("%.6f", upperBound)
}

func (r *StatsdReporter) durationBucketString(upperBound time.Duration) string {
	if upperBound == time.Duration(math.MaxInt64) {
		return "infinity"
	}
	if upperBound == time.Duration(math.MinInt64) {
		return "-infinity"
	}
	return upperBound.String()
}
