package statsd

import (
	statsd "github.com/smira/go-statsd.git"
)

type TagFormat int

const (
	TAG_FORMAT_NONE    = iota
	TAG_FORMAT_INFLUX  = iota
	TAG_FORMAT_DATADOG = iota
)

type ReporterOptions struct {
	TagFormat TagFormat
}

type Option func(o *ReporterOptions) statsd.Option

func TagStyle(format TagFormat) Option {
	return func(o *ReporterOptions) statsd.Option {
		switch format {
		case TAG_FORMAT_INFLUX:
			o.TagFormat = format
			return statsd.TagStyle(statsd.TagFormatInfluxDB)
		case TAG_FORMAT_DATADOG:
			o.TagFormat = format
			return statsd.TagStyle(statsd.TagFormatDatadog)
		default:
			o.TagFormat = TAG_FORMAT_NONE
			return statsd.TagStyle(&statsd.TagFormat{})
		}
	}
}
