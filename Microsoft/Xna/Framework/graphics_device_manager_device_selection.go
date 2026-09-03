package framework

import (
	"errors"
	"fmt"

	"github.com/openeggbert/cna-go/internal/servicebridge"
)

// This file projects the five GraphicsDeviceManager members that select a
// device, and the event that lets a consumer change the selection before it is
// used.
//
// # Why they are here and not in the Graphics package
//
// All five are declared on GraphicsDeviceManager, whose namespace is
// Microsoft.Xna.Framework, so they project into this package. Their reference
// bodies reach GraphicsAdapter, DisplayMode, PresentationParameters and
// GraphicsDevice, none of which this package can name -- the Graphics package
// imports this one. The LOGIC that belongs to GraphicsDeviceManager stays here;
// each operation it cannot spell crosses through internal/servicebridge to a
// Graphics-package implementation. See graphics_device_selection.go there.
//
// # The two Microsoft resource strings
//
// Read from the Microsoft.Xna.Framework.Resources.resources stream of the
// retained Microsoft.Xna.Framework.Game.dll
// (sha256 b5dffdd8125abef2a4507ba4e1d2f11062143f0a63d48fe4f298b95ad746a1f0).
const (
	// noCompatibleDevices carries the GraphicsProfile through String.Format's
	// {0}, which CNA-Go spells %s. The CLR string's line breaks are CRLF and
	// are reproduced byte for byte.
	noCompatibleDevices = "Could not find a Direct3D device that supports the XNA Framework %s profile.\r\n\r\n" +
		"Verify that a suitable graphics device is installed.\r\n\r\n" +
		"Make sure the desktop is not locked, and that no other application is running in full screen mode.\r\n\r\n" +
		"Avoid running under Remote Desktop or as a Windows service.\r\n\r\n" +
		"Check the display properties to make sure hardware acceleration is set to Full."
	noCompatibleDevicesAfterRanking = "The process of ranking devices removed all compatible devices."
)

// errNoSuitableGraphicsDevice projects
// Microsoft.Xna.Framework.Graphics.NoSuitableGraphicsDeviceException, which
// FindBestPlatformDevice throws from its two empty-list branches.
//
// It is an unexported sentinel for the reason every other CLR-exception
// projection in CNA-Go is: the public contract of FindBestDevice declares a
// return value, not an exception type, and giving a consumer a way to tell one
// CLR exception from another is the System.Exception public mapping decision.
// A consumer can tell success from failure, which is what the contract needs.
var errNoSuitableGraphicsDevice = errors.New("no suitable graphics device")

// errNoDeviceToReset is the failure CanResetDevice reports when the manager has
// no device at all. The reference dereferences its `device` field without a
// null check, so it throws NullReferenceException there.
var errNoDeviceToReset = errors.New("the graphics device manager has no device")

// errGraphicsDeviceManagerNil is the failure every member reports on a nil
// receiver, which is the reference's NullReferenceException.
var errGraphicsDeviceManagerNil = errors.New("graphics device manager is nil")

// ---------------------------------------------------------------------------
// The PreparingDeviceSettings event.
// ---------------------------------------------------------------------------

// AddPreparingDeviceSettingsHandler registers a handler for
// GraphicsDeviceManager::PreparingDeviceSettings, the one event of the five
// this type declares whose args are not EventArgs.
//
// The reference's add accessor is the ordinary Delegate.Combine loop every CLR
// event has, so the settled event architecture applies unchanged: registration
// order is invocation order, and the subscription token is what removes it
// again.
func (m *GraphicsDeviceManager) AddPreparingDeviceSettingsHandler(
	handler EventHandler[*PreparingDeviceSettingsEventArgs],
) (EventSubscription, error) {
	if m == nil {
		return EventSubscription{}, errGraphicsDeviceManagerNil
	}
	return m.preparingDeviceSettings.Add(handler)
}

// RemovePreparingDeviceSettingsHandler removes one registration.
func (m *GraphicsDeviceManager) RemovePreparingDeviceSettingsHandler(subscription EventSubscription) error {
	if m == nil {
		return errGraphicsDeviceManagerNil
	}
	return m.preparingDeviceSettings.Remove(subscription)
}

// OnPreparingDeviceSettings is GraphicsDeviceManager::OnPreparingDeviceSettings,
// whose whole body is the null-guarded invoke every protected raise helper in
// this type is:
//
//	if (PreparingDeviceSettings != null)
//	    PreparingDeviceSettings(sender, args);
//
// It passes the SENDER it was given rather than `this`. The reference's own
// caller, ChangeDevice, passes `this`, but the member does not impose that, and
// a consumer calling it directly chooses the sender.
func (m *GraphicsDeviceManager) OnPreparingDeviceSettings(sender any, args *PreparingDeviceSettingsEventArgs) error {
	if m == nil {
		return errGraphicsDeviceManagerNil
	}
	return m.preparingDeviceSettings.Raise(sender, args)
}

// ---------------------------------------------------------------------------
// Device selection.
// ---------------------------------------------------------------------------

// CanResetDevice is GraphicsDeviceManager::CanResetDevice, whose entire body is
//
//	return this.device.GraphicsProfile == newDeviceInfo.GraphicsProfile;
//
// A change of profile needs a new device; anything else can be a Reset. The
// reference dereferences both `this.device` and the argument without a null
// check, so either being null is a NullReferenceException, and CNA-Go reports
// each as a failure rather than answering.
func (m *GraphicsDeviceManager) CanResetDevice(newDeviceInfo *GraphicsDeviceInformation) (bool, error) {
	if m == nil {
		return false, errGraphicsDeviceManagerNil
	}
	if newDeviceInfo == nil {
		return false, errGraphicsDeviceInformationNil
	}
	profile, ok := servicebridge.ReadDeviceGraphicsProfile(m.deviceFacade)
	if !ok {
		return false, errNoDeviceToReset
	}
	return profile == newDeviceInfo.graphicsProfile, nil
}

// RankDevices is GraphicsDeviceManager::RankDevices, which forwards to the
// private RankDevicesPlatform:
//
//	foundDevices.Sort(new GraphicsDeviceInformationComparer(this));
//
// so the whole member is one sort, in place, with the manager's own preferences
// as the comparison's context. The comparer is projected in the Graphics
// package because every value it reads is a Graphics type; its exact ordering
// is documented there.
//
// # The List<T> parameter
//
// This is the ONLY position in the whole XNA 4.0 Windows profile that carries
// System.Collections.Generic.List<T>, and what the reference does with it here
// is sort it and read it by index. The settled signature rule -- a BCL type at
// a public signature position takes the standard-library Go type whose ROLE it
// is, chosen from what the profile's positions measurably do with it -- gives a
// Go slice, which sorts in place and indexes exactly as the list does. It is
// the same reasoning that made System.IO.Stream an io.Reader.
//
// A Go slice cannot grow or shrink through the callee, and List<T> can. Nothing
// in the profile does: this method sorts, and the private AddDevices that
// appends is not projected surface.
func (m *GraphicsDeviceManager) RankDevices(foundDevices []*GraphicsDeviceInformation) error {
	if m == nil {
		return errGraphicsDeviceManagerNil
	}
	candidates := make([]any, len(foundDevices))
	for i, candidate := range foundDevices {
		candidates[i] = candidate
	}
	servicebridge.RankDeviceCandidates(m, candidates)
	for i := range foundDevices {
		typed, _ := candidates[i].(*GraphicsDeviceInformation)
		foundDevices[i] = typed
	}
	return nil
}

// FindBestDevice is GraphicsDeviceManager::FindBestDevice, which forwards to
// the private FindBestPlatformDevice:
//
//	List<GraphicsDeviceInformation> found = new List<...>();
//	AddDevices(anySuitableDevice, found);
//	if (found.Count == 0 && PreferMultiSampling) {
//	    PreferMultiSampling = false;            // and it STAYS false
//	    AddDevices(anySuitableDevice, found);
//	}
//	if (found.Count == 0)
//	    throw new NoSuitableGraphicsDeviceException(Format(NoCompatibleDevices, graphicsProfile));
//	RankDevices(found);                          // virtual
//	if (found.Count == 0)
//	    throw new NoSuitableGraphicsDeviceException(NoCompatibleDevicesAfterRanking);
//	return found[0];
//
// Three details are load-bearing and are reproduced rather than tidied:
//
//   - the multisampling retry is a real MUTATION of the manager. Turning it off
//     is a setter call, so it raises isDeviceDirty and pushes to CNA, and it is
//     not restored afterwards. A consumer who asked for multisampling and got
//     none has had the preference silently cleared, which is the reference's
//     behaviour;
//   - RankDevices is reached VIRTUALLY, so an override that empties the list is
//     what the second throw exists for. CNA-Go has no override mechanism for
//     this type, so the second branch is unreachable today -- it is projected
//     anyway, because the member it guards is projected;
//   - the result is found[0], the FIRST after ranking, not the best by any
//     measure this method takes itself.
func (m *GraphicsDeviceManager) FindBestDevice(anySuitableDevice bool) (*GraphicsDeviceInformation, error) {
	if m == nil {
		return nil, errGraphicsDeviceManagerNil
	}
	found, err := servicebridge.CollectDeviceCandidates(m, anySuitableDevice)
	if err != nil {
		return nil, err
	}
	if len(found) == 0 && m.allowMultiSampling {
		if err := m.SetPreferMultiSampling(false); err != nil {
			return nil, err
		}
		if found, err = servicebridge.CollectDeviceCandidates(m, anySuitableDevice); err != nil {
			return nil, err
		}
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("%w: %s", errNoSuitableGraphicsDevice,
			fmt.Sprintf(noCompatibleDevices, graphicsProfileName(m.graphicsProfile)))
	}
	servicebridge.RankDeviceCandidates(m, found)
	if len(found) == 0 {
		return nil, fmt.Errorf("%w: %s", errNoSuitableGraphicsDevice, noCompatibleDevicesAfterRanking)
	}
	best, _ := found[0].(*GraphicsDeviceInformation)
	return best, nil
}

// graphicsProfileName renders the GraphicsProfile the NoCompatibleDevices
// message carries.
//
// The reference boxes the enum and lets String.Format call its ToString, which
// for a defined GraphicsProfile is its NAME -- "Reach" or "HiDef" -- and for an
// undefined value is the decimal number. This package cannot name the enum, so
// the two defined names are spelled here from the pinned metadata; anything
// else renders as the CLR renders an undefined enum value.
func graphicsProfileName(value int32) string {
	switch value {
	case 0:
		return "Reach"
	case 1:
		return "HiDef"
	default:
		return fmt.Sprintf("%d", value)
	}
}

// init installs this package's half of the manager-window accessor, so the
// Graphics-package AddDevices body can reach the window whose handle every
// candidate carries. The reference reads `this.game.Window` there, and `game`
// is private on the manager exactly as it is here.
func init() {
	servicebridge.SetManagerWindowReader(func(manager any) any {
		typed, ok := manager.(*GraphicsDeviceManager)
		if !ok || typed == nil || typed.game == nil {
			return nil
		}
		return typed.game.Window()
	})
}
