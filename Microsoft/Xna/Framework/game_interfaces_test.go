package framework

import (
	"errors"
	"testing"
)

// runtimeProbe is a test-only double. It proves the two runtime-boundary
// contracts are satisfiable and that each keeps a real failure channel; it is
// not a game component and it creates no device.
type runtimeProbe struct {
	initialized  bool
	deviceExists bool
	drawing      bool

	initializeFailure  error
	createDeviceFail   error
	beginDrawProceed   bool
	beginDrawFailure   error
	endDrawFailure     error
	endDrawCalledAfter bool
}

func (p *runtimeProbe) Initialize() error {
	if p.initializeFailure != nil {
		return p.initializeFailure
	}
	p.initialized = true
	return nil
}

func (p *runtimeProbe) CreateDevice() error {
	if p.createDeviceFail != nil {
		return p.createDeviceFail
	}
	p.deviceExists = true
	return nil
}

func (p *runtimeProbe) BeginDraw() (bool, error) {
	if p.beginDrawFailure != nil {
		return false, p.beginDrawFailure
	}
	p.drawing = p.beginDrawProceed
	return p.beginDrawProceed, nil
}

func (p *runtimeProbe) EndDraw() error {
	p.endDrawCalledAfter = p.drawing
	if p.endDrawFailure != nil {
		return p.endDrawFailure
	}
	p.drawing = false
	return nil
}

// Compile-time conformance for both contracts.
var (
	_ IGameComponent         = (*runtimeProbe)(nil)
	_ IGraphicsDeviceManager = (*runtimeProbe)(nil)
)

// TestGameComponentContractReportsFailure pins that the single operation keeps
// a failure channel, which is what DrawableGameComponent.Initialize needs when
// IGraphicsDeviceService is absent from Game.Services.
func TestGameComponentContractReportsFailure(t *testing.T) {
	probe := &runtimeProbe{}
	var component IGameComponent = probe
	if err := component.Initialize(); err != nil {
		t.Fatalf("Initialize = %v", err)
	}
	if !probe.initialized {
		t.Fatal("Initialize did not run")
	}

	missingService := errors.New("no IGraphicsDeviceService in Game.Services")
	probe.initializeFailure = missingService
	if err := component.Initialize(); !errors.Is(err, missingService) {
		t.Fatalf("Initialize failure = %v", err)
	}
}

// TestGraphicsDeviceManagerContractKeepsChannelsSeparate pins the shape that
// matters most on this contract: BeginDraw's source Boolean says whether
// drawing may proceed, and it stays distinct from the error, which says
// whether the call itself failed.
func TestGraphicsDeviceManagerContractKeepsChannelsSeparate(t *testing.T) {
	probe := &runtimeProbe{beginDrawProceed: true}
	var manager IGraphicsDeviceManager = probe

	if err := manager.CreateDevice(); err != nil {
		t.Fatalf("CreateDevice = %v", err)
	}
	if !probe.deviceExists {
		t.Fatal("CreateDevice did not run")
	}

	proceed, err := manager.BeginDraw()
	if err != nil || !proceed {
		t.Fatalf("BeginDraw = %t, %v", proceed, err)
	}
	if err := manager.EndDraw(); err != nil {
		t.Fatalf("EndDraw = %v", err)
	}
	if !probe.endDrawCalledAfter {
		t.Fatal("EndDraw did not observe the begun frame")
	}

	// Declining to draw is not a failure: false with a nil error.
	probe.beginDrawProceed = false
	proceed, err = manager.BeginDraw()
	if err != nil || proceed {
		t.Fatalf("declined BeginDraw = %t, %v", proceed, err)
	}

	// Failing is not the same as declining: the error channel carries it.
	deviceLost := errors.New("device lost")
	probe.beginDrawFailure = deviceLost
	if _, err := manager.BeginDraw(); !errors.Is(err, deviceLost) {
		t.Fatalf("BeginDraw failure = %v", err)
	}
	probe.createDeviceFail = deviceLost
	if err := manager.CreateDevice(); !errors.Is(err, deviceLost) {
		t.Fatalf("CreateDevice failure = %v", err)
	}
	probe.endDrawFailure = deviceLost
	if err := manager.EndDraw(); !errors.Is(err, deviceLost) {
		t.Fatalf("EndDraw failure = %v", err)
	}
}

// TestGraphicsDeviceManagerIsNotBoundToTheFacade records the deliberate
// boundary: declaring the contract does not make CNA-Go's partial
// native-backed GraphicsDeviceManager implement it.
func TestGraphicsDeviceManagerIsNotBoundToTheFacade(t *testing.T) {
	var candidate any = &GraphicsDeviceManager{}
	if _, ok := candidate.(IGraphicsDeviceManager); ok {
		t.Fatal("GraphicsDeviceManager unexpectedly implements IGraphicsDeviceManager")
	}
}
