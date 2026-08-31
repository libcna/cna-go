package framework

import "github.com/openeggbert/cna-go/internal/servicebridge"

// setManagerSlotForTest exercises the installed writer the same way the
// Graphics package does, without this package naming a Graphics type.
func setManagerSlotForTest(m *GraphicsDeviceManager, slot int, value int32) error {
	return servicebridge.WriteManagerConfiguration(m, servicebridge.ManagerConfigurationSlot(slot), value)
}
