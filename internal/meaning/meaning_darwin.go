package meaning

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var (
	selAlloc     = objc.RegisterName("alloc")
	selInit      = objc.RegisterName("init")
	selInitUTF8  = objc.RegisterName("initWithUTF8String:")
	selRelease   = objc.RegisterName("release")
	selDrain     = objc.RegisterName("drain")
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
	pool := newPool()
	defer pool.drain()
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
	pool := newPool()
	defer pool.drain()
	s := nsString(text)
	defer s.Send(selRelease)
	v := make([]float32, m.dimension)
	ok := objc.Send[bool](m.embedding, selGetVector, unsafe.Pointer(&v[0]), s)
	return v, ok
}

// pool is an autorelease pool. Cocoa drains a pool on the thread that made
// it, so the goroutine stays on its OS thread until drain; without that the
// Go scheduler can move it mid-call and draining corrupts both threads' pools.
type pool objc.ID

func newPool() pool {
	runtime.LockOSThread()
	return pool(objc.ID(objc.GetClass("NSAutoreleasePool")).Send(selAlloc).Send(selInit))
}

func (p pool) drain() {
	objc.ID(p).Send(selDrain)
	runtime.UnlockOSThread()
}

// nsString makes an NSString the caller must release.
func nsString(s string) objc.ID {
	b := append([]byte(s), 0)
	return objc.ID(objc.GetClass("NSString")).Send(selAlloc).Send(selInitUTF8, unsafe.Pointer(&b[0]))
}
