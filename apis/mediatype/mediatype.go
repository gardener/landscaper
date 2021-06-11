// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package mediatype

import (
	"errors"
	"fmt"
	"strings"
)

var InvalidTypeError = errors.New("INVALID_MEDIA_TYPE")

type MediaTypeFormat string

const (
	DefaultFormat MediaTypeFormat = ""
	OCIConfigFormat MediaTypeFormat  = "ociConfig"
	OCILayerFormat MediaTypeFormat  = "ociLayer"
)

// MediaType describes a media type (formerly known as MIME type) defined by the IANA in THE RFC 2045.
// In addition oci specific media types as defined in
// https://github.com/opencontainers/artifacts/blob/master/artifact-authors.md#defining-layermediatypes are parsed.
type MediaType struct {
	// Orig contains the complete original type
	Orig string
	// Format describes the specific format of the media type
	Format MediaTypeFormat
	// Type contain the parsed type without suffix.
	Type string
	// Suffix contains the suffix of a media that is given after the "+"-char
	Suffix *string

	// Only valid for oci types

	// Version describes the config or layer version.
	Version *string
	// FileFormat contains only the file format.
	// This is part of the Type if the mediatype is not a ConfigType.
	// If it is a ConfigType the FileFormat is not part of the type.
	FileFormat *string
	// CompressionFormat contains the optional compression format.
	CompressionFormat *string
}

// String returns the string using the parsed type file and compression.
func (t MediaType) String() string {
	s := t.Type
	if t.CompressionFormat !=  nil  && t.Suffix == nil {
		t.Suffix = t.CompressionFormat
	}
	if t.Suffix != nil {
		s = s + "+" + *t.Suffix
	}
	return s
}

// HasSuffix checks if the media type contains the given suffix.
// if the given format is empty this functions only validates if a format is given
func (t MediaType) HasSuffix(suffix string) bool {
	if t.Suffix == nil {
		return false
	}
	if len(suffix) == 0 {
		return true
	}
	return *t.Suffix == suffix
}

// IsCompressed checks if the media type is compressed with the given format.
// if the given format is empty this functions only validates if a compression is given
func (t MediaType) IsCompressed(format string) bool {
	if t.CompressionFormat == nil {

		// try to parse the compression from the suffix or tree
		if t.Suffix == nil {
			return  false
		}

		return *t.Suffix == format
	}
	if len(format) == 0 {
		return true
	}
	return *t.CompressionFormat == format
}

// HasFileFormat checks if the media type contains the given file format.
// if the given format is empty this functions only validates if a format is given
func (t MediaType) HasFileFormat(format string) bool {
	if t.FileFormat == nil {
		return false
	}
	if len(format) == 0 {
		return true
	}
	return *t.FileFormat == format
}

// ConversionFunc describes a conversion func for a media type
type ConversionFunc func(mediaType string) (convertedType string, converted bool, err error)

// Parse parses a config and layer media type according to the oci spec.
// Config:
// [registration-tree].[org|company|entity].[objectType].[optional-subType].config.[version]+[optional-configFormat]
// Layers:
// [registration-tree].[org|company|entity].[layerType].[optional-layerSubType].layer.[version].[fileFormat]+[optional-compressionFormat]
// Layers also allow any IANA media type
//
// https://github.com/opencontainers/artifacts/blob/master/artifact-authors.md#defining-layermediatypes
//
// This method is highly landscaper specific and can also handle legacy types that are automatically converted.
func Parse(mediaType string, conversions ...ConversionFunc) (MediaType, error) {
	for _, conversion := range append(conversions, DefaultMediaTypeConversions) {
		converted, ok, err := conversion(mediaType)
		if err != nil {
			return MediaType{}, fmt.Errorf("error during conversion: %w", err)
		}
		if ok {
			mediaType = converted
		}
	}

	splitType := strings.Split(mediaType, "/")
	if len(splitType) != 2 {
		return MediaType{}, InvalidTypeError
	}
	mt := MediaType{
		Orig: mediaType,
		Type: mediaType,
	}
	t := splitType[0]
	tree := splitType[1]

	if suffixIndex := strings.Index(mediaType, "+"); suffixIndex > -1 {
		mt.Suffix = strPtr(mediaType[suffixIndex+1:])
		mt.Type = mediaType[:suffixIndex]
		tree = mediaType[len(mediaType)-len(tree) : suffixIndex]
	}

	if t != "application" {
		mt.Type = mediaType
		mt.FileFormat = strPtr(tree)
		return mt, nil
	}

	// try to detect config or layer type
	splitType = strings.Split(tree, ".")


	if len(splitType) > 2 && splitType[len(splitType)-2] == "config" {
		mt.Format = OCIConfigFormat
		mt.FileFormat = mt.Suffix
		mt.Version = strPtr(splitType[len(splitType)-1])
	} else if len(splitType) > 3 && splitType[len(splitType)-3] == "layer" {
		mt.Format =  OCILayerFormat
		mt.FileFormat = strPtr(splitType[len(splitType)-1])
		mt.Version = strPtr(splitType[len(splitType)-2])
	}

	return mt, nil
}

// Builder is a media type builder
type Builder struct {
	Type MediaType
}

// NewBuilder creates a new media type builder.
func NewBuilder(t string) *Builder {
	return &Builder{
		MediaType{
			Orig:              "",
			Type:              t,
			Format: DefaultFormat,
			Version: nil,
			FileFormat:        nil,
			CompressionFormat: nil,
		},
	}
}

// Compression sets the compression format
func (b *Builder) Compression(comp string) *Builder {
	b.Type.CompressionFormat = &comp
	b.Type.Suffix  = &comp
	return b
}

// FileFormat sets the file format of the media type.
// Will be automatically parsed from the type if it is not a config type.
func (b *Builder) FileFormat(format string) *Builder {
	b.Type.FileFormat = &format
	b.Type.Suffix  = &format
	return b
}

// IsConfigType configures the media type as oci config.
func (b *Builder) IsConfigType() *Builder {
	b.Type.Format = OCIConfigFormat
	return b
}

// IsLayerType configures the media type as oci layer.
func (b *Builder) IsLayerType() *Builder {
	b.Type.Format = OCILayerFormat
	return b
}

// Build builds the mediatype
func (b *Builder) Build() MediaType {
	return b.Type
}

// String returns the media type as string
func (b *Builder) String() string {
	return b.Build().String()
}

func strPtr(s string) *string {
	return &s
}
