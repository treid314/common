// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// A package for exposing Prometheus metrics.
package expfmt

import (
	"fmt"
	"io"
	"net/http"

	"bitbucket.org/ww/goautoneg"
	"github.com/golang/protobuf/proto"
	"github.com/matttproud/golang_protobuf_extensions/pbutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt/text"
)

const (
	ProtocolVersion = "0.0.4"

	// The Content-Type values for the different wire protocols.
	FmtText         = `text/plain; version=` + ProtocolVersion
	FmtProtoDelim   = `application/vnd.google.protobuf; proto=io.prometheus.client.MetricFamily; encoding=delimited`
	FmtProtoText    = `application/vnd.google.protobuf; proto=io.prometheus.client.MetricFamily; encoding=text`
	FmtProtoCompact = `application/vnd.google.protobuf; proto=io.prometheus.client.MetricFamily; encoding=compact-text`
)

// Encoder types encode metric families into an underlying wire protocol.
type Encoder interface {
	Encode(*dto.MetricFamily) error
}

type encoder func(*dto.MetricFamily) error

func (e encoder) Encode(v *dto.MetricFamily) error {
	return e(v)
}

// New returns a new encoder based on content type negotiation.
func New(w io.Writer, h http.Header) (e Encoder, ct string) {
	switch ct = formatType(h.Get("Accept")); ct {
	case FmtProtoDelim:
		e = encoder(func(v *dto.MetricFamily) error {
			_, err := pbutil.WriteDelimited(w, v)
			return err
		})
	case FmtProtoCompact:
		e = encoder(func(v *dto.MetricFamily) error {
			_, err := fmt.Fprintln(w, v.String())
			return err
		})
	case FmtProtoText:
		e = encoder(func(v *dto.MetricFamily) error {
			_, err := fmt.Fprintln(w, proto.MarshalTextString(v))
			return err
		})
	default:
		// By default we return the plain text format.
		e = encoder(func(v *dto.MetricFamily) error {
			_, err := text.MetricFamilyToText(w, v)
			return err
		})
	}
	return e, ct
}

const (
	protoType    = "application"
	protoSubType = "vnd.google.protobuf"
	protoProto   = "io.prometheus.client.MetricFamily"
)

// formatType returns the Content-Type based on the given Accept header.
// If no appropriate accepted type is found, FmtText is returned.
func formatType(as string) string {
	for _, ac := range goautoneg.ParseAccept(as) {
		// Check for protocol buffer
		if ac.Type == protoType && ac.SubType == protoSubType && ac.Params["proto"] == protoProto {
			switch ac.Params["encoding"] {
			case "delimited":
				return FmtProtoDelim
			case "text":
				return FmtProtoText
			case "compact-text":
				return FmtProtoCompact
			}
		}
		// Check for text format.
		ver := ac.Params["version"]
		if ac.Type == "text" && ac.SubType == "plain" && (ver == "0.0.4" || ver == "") {
			return FmtText
		}
	}
	return FmtText
}