package meaning

import "testing"

func TestVector(t *testing.T) {
	m, err := New()
	if err != nil {
		t.Skipf("no sentence embedding here: %v", err)
	}
	gpu, ok1 := m.Vector("writing fragment shaders for the GPU")
	webgpu, ok2 := m.Vector("a WebGPU compute pipeline tutorial")
	bread, ok3 := m.Vector("sourdough bread hydration tips")
	if !ok1 || !ok2 || !ok3 {
		t.Fatal("no vector for plain English")
	}
	if len(gpu) == 0 || m.Name() == "" {
		t.Fatalf("dimension %d, name %q", len(gpu), m.Name())
	}
	if Cosine(gpu, gpu) < 0.999 {
		t.Errorf("a text is not most similar to itself: %v", Cosine(gpu, gpu))
	}
	if Cosine(gpu, webgpu) <= Cosine(gpu, bread) {
		t.Errorf("shaders sit closer to bread (%v) than to WebGPU (%v)", Cosine(gpu, bread), Cosine(gpu, webgpu))
	}
}
