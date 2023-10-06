package entity

import (
	"github.com/bloeys/nmage/assert"
	"github.com/bloeys/nmage/registry"
)

type Comp interface {
	// This ensures that implementors of the Comp interface
	// always embed BaseComp
	baseComp()

	Name() string
	Init(parentHandle registry.Handle)
	Update()
	Destroy()
}

func NewCompContainer() CompContainer {
	return CompContainer{Comps: []Comp{}}
}

type CompContainer struct {
	Comps []Comp
}

func AddComp[T Comp](entityHandle registry.Handle, cc *CompContainer, c T) {

	assert.T(!HasComp[T](cc), "Entity with id '%v' already has component of type '%T'", entityHandle, c)

	cc.Comps = append(cc.Comps, c)
	c.Init(entityHandle)
}

func HasComp[T Comp](e *CompContainer) bool {

	for i := 0; i < len(e.Comps); i++ {

		_, ok := e.Comps[i].(T)
		if ok {
			return true
		}
	}

	return false
}

func GetComp[T Comp](e *CompContainer) (out T) {

	for i := 0; i < len(e.Comps); i++ {

		comp, ok := e.Comps[i].(T)
		if ok {
			return comp
		}
	}

	return out
}

// DestroyComp calls Destroy on the component and then removes it from the entities component list
func DestroyComp[T Comp](e *CompContainer) {

	for i := 0; i < len(e.Comps); i++ {

		comp, ok := e.Comps[i].(T)
		if ok {
			comp.Destroy()
			e.Comps = append(e.Comps[:i], e.Comps[i+1:]...)
			return
		}
	}
}
