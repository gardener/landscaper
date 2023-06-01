package spiffing

import (
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/mandelsoft/spiff/dynaml"
	"github.com/mandelsoft/spiff/flow"
	"github.com/mandelsoft/spiff/yaml"
)

// MODE_PRIVATE does not allow access to any external resources
const MODE_PRIVATE = 0

// MODE_OS_ACCESS allows os command execution (pipe, exec)
const MODE_OS_ACCESS = flow.MODE_OS_ACCESS

//  MODE_FILE_ACCESS allows file access to virtual filesystem
const MODE_FILE_ACCESS = flow.MODE_FILE_ACCESS

// MODE_DEFAULT (default) enables all os related spiff functions
const MODE_DEFAULT = MODE_OS_ACCESS | MODE_FILE_ACCESS

// Node is a document node of the processing representation of a document
type Node = yaml.Node

// Options described the processing options
type Options = flow.Options

// Functions provides access to a set of spiff functions used to extend
// the standard function set
type Functions = dynaml.Functions

// Function is the signature of a dynaml function
type Function = dynaml.Function

// Controls provides access to a set of spiff controls used to extend
// the standard control set
type Controls = dynaml.Controls

// Spiff is a configuration and execution context for
// executing spiff operations
type Spiff interface {
	// WithEncryptionKey creates a new context with
	// dedicated encryption key used for the spiff encryption feature
	WithEncryptionKey(key string) Spiff
	// WithMode creates a new context with the given processing mode.
	// (see MODE constants)
	WithMode(mode int) Spiff
	// WithFileSystem creates a new context with the given
	// virtual filesystem used for filesystem functions during
	// prcessing. Setting a filesystem disables the command
	// execution functions.
	WithFileSystem(fs vfs.FileSystem) Spiff
	// WithFunctions creates a new context with the given
	// additional function definitions
	WithFunctions(functions Functions) Spiff
	// WithFunctions creates a new context with the given
	// additional function definitions
	WithControls(controls Controls) Spiff

	// WithFeatures creates a new context with the given
	// additional features enabled
	WithFeatures(features ...string) Spiff
	// WithInterpolation creates a new context with the interpolation
	// feature enabled/disabled
	WithInterpolation(b bool) Spiff
	// WithControl creates a new context with the yaml based control structure
	// feature enabled/disabled
	WithControl(b bool) Spiff

	// WithValues creates a new context with the given
	// additional structured values usable by path expressions
	// during processing.
	// It is highly recommended to decide for a common root
	// value (like `values`) to minimize the blocked root
	// elements in the processed documents.
	WithValues(values map[string]interface{}) (Spiff, error)

	// SetTag sets/resets a tag for subsequent processings.
	// This can be used to set implicit document tags
	// when simulating a multi-document processing.
	// Please note: preconfiguted tags are only used by the
	// ApplyStubs method.
	SetTag(tag string, node yaml.Node) Spiff

	// CleanupTags deletes tags of spiff context
	CleanupTags() Spiff
	// Reset flushes the binding state
	Reset() Spiff
	// ResetStream flushes the document history
	// and removes all implicit document stream tags.
	ResetStream() Spiff

	// FileSystem return the virtual filesystem set for the execution context.
	FileSystem() vfs.FileSystem
	// FileSource create a new file source based on the configured file system.
	FileSource(path string) Source

	// Unmarshal parses a single document yaml representation and
	// returns the internal representation
	Unmarshal(name string, source []byte) (Node, error)
	// Unmarshal parses a single source and
	// returns the internal representation
	UnmarshalSource(source Source) (Node, error)
	// UnmarshalMulti parses a multi document yaml representation and
	// returns the list of documents in the internal representation
	UnmarshalMulti(name string, source []byte) ([]Node, error)
	// UnmarshalMultiSource parses a multi document source and
	// returns the list of documents in the internal representation
	UnmarshalMultiSource(source Source) ([]Node, error)
	// Marshal transform the internal node representation into a
	// yaml representation
	Marshal(node Node) ([]byte, error)
	// DetermineState extracts the intended new state representation from
	// a processing result.
	DetermineState(node Node) Node
	// Normalize transform the node representation to a regular go value representation
	// consisting of map[string]interface{}`, `[]interface{}`, `string `boolean`,
	// `int64`, `float64` and []byte objects
	Normalize(node Node) (interface{}, error)

	// Cascade processes a template with a list of given subs and state
	// documents.
	// The document stream history (implicit tags) is resetted prior
	// to the execution.
	Cascade(template Node, stubs []Node, states ...Node) (Node, error)
	// PrepareStubs processes a list a stubs and returns a prepared
	// represenation usable to process a template.
	// The document stream history (implicit tags) is resetted prior
	// to the execution.
	PrepareStubs(stubs ...Node) ([]Node, error)
	// ApplyStubs uses already prepared subs to process a template.
	// The document stream history (implicit tags) is resetted prior
	// to the execution. If the call is part of the processing
	// of a document stream, the optional argument must be set to
	// true. In this case every call add an entry to the document
	// history.
	ApplyStubs(template Node, preparedstubs []Node, stream ...bool) (Node, error)
}

// Source is used to get access to a template or stub source data and name
type Source interface {
	// Name resturns the name of the source
	// For file based sources this should be the path name of the file.
	Name() string
	// Data returns the yaml representation of the source document
	Data() ([]byte, error)
}
