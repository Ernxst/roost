package meaning

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var (
	selAlloc     = objc.RegisterName("alloc")
	selInitUTF8  = objc.RegisterName("initWithUTF8String:")
	selRelease   = objc.RegisterName("release")
	selRetain    = objc.RegisterName("retain")
	selEmbedding = objc.RegisterName("sentenceEmbeddingForLanguage:")
	selDimension = objc.RegisterName("dimension")
	selRevision  = objc.RegisterName("revision")
	selGetVector = objc.RegisterName("getVector:forString:")
)

// Model is NaturalLanguage's English sentence embedding.
type Model struct {
	mu        sync.Mutex
	embedding objc.ID
	dimension int
	name      string
}

// New opens the English sentence embedding.
func New() (*Model, error) {
	if _, err := purego.Dlopen("/System/Library/Frameworks/NaturalLanguage.framework/NaturalLanguage", purego.RTLD_GLOBAL|purego.RTLD_NOW); err != nil {
		return nil, err
	}
	lang := nsString("en")
	defer lang.Send(selRelease)
	e := objc.ID(objc.GetClass("NLEmbedding")).Send(selEmbedding, lang)
	if e == 0 {
		return nil, errors.New("macOS has no English sentence embedding available")
	}
	e.Send(selRetain)
	return &Model{
		embedding: e,
		dimension: objc.Send[int](e, selDimension),
		name:      fmt.Sprintf("NLEmbedding/en/%d", objc.Send[uint](e, selRevision)),
	}, nil
}

// Name identifies the model and its revision; vectors from different
// models can't be compared.
func (m *Model) Name() string { return m.name }

// Vector embeds text, reporting false when the model has no vector for it.
func (m *Model) Vector(text string) ([]float32, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := nsString(text)
	defer s.Send(selRelease)
	v := make([]float32, m.dimension)
	ok := objc.Send[bool](m.embedding, selGetVector, unsafe.Pointer(&v[0]), s)
	return v, ok
}

// nsString makes an NSString the caller must release.
func nsString(s string) objc.ID {
	b := append([]byte(s), 0)
	return objc.ID(objc.GetClass("NSString")).Send(selAlloc).Send(selInitUTF8, unsafe.Pointer(&b[0]))
}
