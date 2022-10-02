package battery

import (
	"syscall"
	"unsafe"
)

var (
	setupapi = syscall.NewLazyDLL("setupapi.dll")

	pSetupDiGetClassDevsW             = setupapi.NewProc("SetupDiGetClassDevsW")
	pSetupDiEnumDeviceInterfaces      = setupapi.NewProc("SetupDiEnumDeviceInterfaces")
	pSetupDiGetDeviceInterfaceDetailW = setupapi.NewProc("SetupDiGetDeviceInterfaceDetailW")
)

type Guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type SP_DEVICE_INTERFACE_DATA struct {
	CbSize             uint32
	InterfaceClassGuid Guid
	Flags              uint32
	Reserved           uintptr
}

type SP_DEVICE_INTERFACE_DETAIL_DATA struct {
	CbSize     uint32
	DevicePath [256]uint16
}

const (
	DIGCF_DEFAULT         = 0x00000001
	DIGCF_PRESENT         = 0x00000002
	DIGCF_ALLCLASSES      = 0x00000004
	DIGCF_PROFILE         = 0x00000008
	DIGCF_DEVICEINTERFACE = 0x00000010
)

func SetupDiGetClassDevs(guid *Guid, enumerator string, hwnd uintptr, flags uint32) (syscall.Handle, error) {
	r1, _, e1 := pSetupDiGetClassDevsW.Call(
		uintptr(unsafe.Pointer(guid)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(enumerator))),
		uintptr(hwnd),
		uintptr(flags))
	if r1 == 0 {
		return syscall.Handle(0), e1
	}

	return syscall.Handle(r1), nil
}

func SetupDiEnumDeviceInterfaces(
	hdevInfo syscall.Handle,
	devInfo uintptr,
	guid *Guid,
	memberIndex uint32,
	devInterfaceData *SP_DEVICE_INTERFACE_DATA) error {
	r1, _, e1 := pSetupDiEnumDeviceInterfaces.Call(
		uintptr(hdevInfo),
		devInfo,
		uintptr(unsafe.Pointer(guid)),
		uintptr(memberIndex),
		uintptr(unsafe.Pointer(devInterfaceData)))
	if r1 == 0 {
		return e1
	}
	return nil
}

func SetupDiGetDeviceInterfaceDetail(
	hdevInfo syscall.Handle,
	devInterfaceData *SP_DEVICE_INTERFACE_DATA,
	devDetailData *SP_DEVICE_INTERFACE_DETAIL_DATA,
	devDetailSize uint32,
	requiredSize *uint32,
	devInfoData uintptr) error {
	r1, _, e1 := pSetupDiGetDeviceInterfaceDetailW.Call(
		uintptr(hdevInfo),
		uintptr(unsafe.Pointer(devInterfaceData)),
		uintptr(unsafe.Pointer(devDetailData)),
		uintptr(devDetailSize),
		uintptr(unsafe.Pointer(requiredSize)),
		devInfoData)
	if r1 == 0 {
		return e1
	}
	return nil
}
