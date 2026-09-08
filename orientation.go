package device

// OnChangeOrientation registers a callback invoked whenever HandleOrientation
// observes the device orientation.
//
// A nil callback is ignored, mirroring the `typeof cb === 'function'` guard of
// the original. Callbacks are invoked in registration order.
//
// Timing note, inherited from the original and reproduced by cmd/wasm: the
// browser entry point calls HandleOrientation once during start-up, before any
// consumer has had a chance to register. A callback therefore never fires for
// the initial orientation, only for subsequent changes. Read Orientation for
// the current value.
func (d *Device) OnChangeOrientation(cb OrientationChangeCallback) {
	if cb == nil {
		return
	}
	d.mu.Lock()
	d.changeOrientationList = append(d.changeOrientationList, cb)
	d.mu.Unlock()
}

// walkOnChangeOrientationList invokes every registered callback in
// registration order.
//
// The callback slice is copied under the lock and the callbacks are invoked
// without it held, so that a callback may safely call back into the Device.
func (d *Device) walkOnChangeOrientationList(newOrientation DeviceOrientation) {
	d.mu.RLock()
	callbacks := make([]OrientationChangeCallback, len(d.changeOrientationList))
	copy(callbacks, d.changeOrientationList)
	d.mu.RUnlock()

	for index := 0; index < len(callbacks); index++ {
		callbacks[index](newOrientation)
	}
}

// HandleOrientation swaps the orientation class on current, notifies every
// registered callback and refreshes the cached Orientation. It returns the
// updated class string.
//
// It is the Go equivalent of the private `handleOrientation` function, which
// the original wires to the window orientation event and also calls once at
// import time.
//
// Landscape is tested first and portrait is the fallback, so a viewport that is
// neither — a perfectly square one, where both probes return false — is
// classed "portrait" here even though Orientation reports
// OrientationUnknown. That asymmetry exists upstream and is preserved.
func (d *Device) HandleOrientation(current string) string {
	var newOrientation DeviceOrientation

	if d.Landscape() {
		current = RemoveClass(current, "portrait")
		current = AddClass(current, "landscape")
		newOrientation = OrientationLandscape
	} else {
		current = RemoveClass(current, "landscape")
		current = AddClass(current, "portrait")
		newOrientation = OrientationPortrait
	}

	d.walkOnChangeOrientationList(newOrientation)
	d.setOrientationCache()

	return current
}

// Window event names used for orientation changes. These are DOM protocol
// constants, fixed by the browser, not values this package is free to choose.
const (
	EventOrientationChange = "orientationchange"
	EventResize            = "resize"
)

// OrientationEventName returns the name of the window event to listen on for
// orientation changes: EventOrientationChange when the browser exposes
// window.onorientationchange, and EventResize otherwise.
func (d *Device) OrientationEventName() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.env.HasOnOrientationChange {
		return EventOrientationChange
	}
	return EventResize
}
