package bsdp

import (
	"testing"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/stretchr/testify/require"
)

func TestOptVendorSpecificInformationInterfaceMethods(t *testing.T) {
	messageTypeOpt := &OptMessageType{MessageTypeList}
	versionOpt := Version1_1
	o := &OptVendorSpecificInformation{[]dhcpv4.Option{messageTypeOpt, versionOpt}}
	require.Equal(t, dhcpv4.OptionVendorSpecificInformation, o.Code(), "Code")

	expectedBytes := []byte{
		1, 1, 1, // List option
		2, 2, 1, 1, // Version option
	}
	o = &OptVendorSpecificInformation{
		[]dhcpv4.Option{
			&OptMessageType{MessageTypeList},
			Version1_1,
		},
	}
	require.Equal(t, expectedBytes, o.ToBytes(), "ToBytes")
}

func TestParseOptVendorSpecificInformation(t *testing.T) {
	var (
		o   *OptVendorSpecificInformation
		err error
	)
	o, err = ParseOptVendorSpecificInformation([]byte{1, 2})
	require.Error(t, err, "short byte stream")

	// Good byte stream
	data := []byte{
		1, 1, 1, // List option
		2, 2, 1, 1, // Version option
	}
	o, err = ParseOptVendorSpecificInformation(data)
	require.NoError(t, err)
	expected := &OptVendorSpecificInformation{
		[]dhcpv4.Option{
			&OptMessageType{MessageTypeList},
			Version1_1,
		},
	}
	require.Equal(t, 2, len(o.Options), "number of parsed suboptions")
	typ := o.GetOneOption(OptionMessageType)
	version := o.GetOneOption(OptionVersion)
	require.Equal(t, expected.Options[0].Code(), typ.Code())
	require.Equal(t, expected.Options[1].Code(), version.Code())

	// Short byte stream (length and data mismatch)
	data = []byte{
		1, 1, 1, // List option
		2, 2, 1, // Version option
	}
	o, err = ParseOptVendorSpecificInformation(data)
	require.Error(t, err)

	// Bad option
	data = []byte{
		1, 1, 1, // List option
		2, 2, 1, // Version option
		5, 3, 1, 1, 1, // Reply port option
	}
	o, err = ParseOptVendorSpecificInformation(data)
	require.Error(t, err)

	// Boot images + default.
	data = []byte{
		1, 1, 1, // List option
		2, 2, 1, 1, // Version option
		5, 2, 1, 1, // Reply port option

		// Boot image list
		9, 22,
		0x1, 0x0, 0x03, 0xe9, // ID
		6, // name length
		'b', 's', 'd', 'p', '-', '1',
		0x80, 0x0, 0x23, 0x31, // ID
		6, // name length
		'b', 's', 'd', 'p', '-', '2',

		// Default Boot Image ID
		7, 4, 0x1, 0x0, 0x03, 0xe9,
	}
	o, err = ParseOptVendorSpecificInformation(data)
	require.NoError(t, err)
	require.Equal(t, 5, len(o.Options))
	for _, opt := range []dhcpv4.OptionCode{
		OptionMessageType,
		OptionVersion,
		OptionReplyPort,
		OptionBootImageList,
		OptionDefaultBootImageID,
	} {
		require.True(t, o.Options.Has(opt))
	}
	optBootImage := o.GetOneOption(OptionBootImageList).(*OptBootImageList)
	expectedBootImages := []BootImage{
		BootImage{
			ID: BootImageID{
				IsInstall: false,
				ImageType: BootImageTypeMacOSX,
				Index:     1001,
			},
			Name: "bsdp-1",
		},
		BootImage{
			ID: BootImageID{
				IsInstall: true,
				ImageType: BootImageTypeMacOS9,
				Index:     9009,
			},
			Name: "bsdp-2",
		},
	}
	require.Equal(t, expectedBootImages, optBootImage.Images)
}

func TestOptVendorSpecificInformationString(t *testing.T) {
	o := &OptVendorSpecificInformation{
		[]dhcpv4.Option{
			&OptMessageType{MessageTypeList},
			Version1_1,
		},
	}
	expectedString := "Vendor Specific Information ->\n  BSDP Message Type -> LIST\n  BSDP Version -> 1.1"
	require.Equal(t, expectedString, o.String())

	// Test more complicated string - sub options of sub options.
	o = &OptVendorSpecificInformation{
		[]dhcpv4.Option{
			&OptMessageType{MessageTypeList},
			&OptBootImageList{
				[]BootImage{
					BootImage{
						ID: BootImageID{
							IsInstall: false,
							ImageType: BootImageTypeMacOSX,
							Index:     1001,
						},
						Name: "bsdp-1",
					},
					BootImage{
						ID: BootImageID{
							IsInstall: true,
							ImageType: BootImageTypeMacOS9,
							Index:     9009,
						},
						Name: "bsdp-2",
					},
				},
			},
		},
	}
	expectedString = "Vendor Specific Information ->\n" +
		"  BSDP Message Type -> LIST\n" +
		"  BSDP Boot Image List ->\n" +
		"    bsdp-1 [1001] uninstallable macOS image\n" +
		"    bsdp-2 [9009] installable macOS 9 image"
	require.Equal(t, expectedString, o.String())
}

func TestOptVendorSpecificInformationGetOptions(t *testing.T) {
	// No option
	o := &OptVendorSpecificInformation{
		[]dhcpv4.Option{
			&OptMessageType{MessageTypeList},
			Version1_1,
		},
	}
	foundOpts := o.GetOption(OptionBootImageList)
	require.Empty(t, foundOpts, "should not get any options")

	// One option
	o = &OptVendorSpecificInformation{
		[]dhcpv4.Option{
			&OptMessageType{MessageTypeList},
			Version1_1,
		},
	}
	foundOpts = o.GetOption(OptionMessageType)
	require.Equal(t, 1, len(foundOpts), "should only get one option")
	require.Equal(t, MessageTypeList, foundOpts[0].(*OptMessageType).Type)

	// Multiple options
	//
	// TODO: Remove this test when RFC 3396 is properly implemented. This
	// isn't a valid packet. RFC 2131, Section 4.1: "Options may appear
	// only once." RFC 3396 clarifies this to say that options will be
	// concatenated. I.e., in this case, the bytes would be:
	//
	// <versioncode> 4 1 1 0 1
	//
	// Which would obviously not be parsed by the Version parser, as it
	// only accepts two bytes.
	o = &OptVendorSpecificInformation{
		[]dhcpv4.Option{
			&OptMessageType{MessageTypeList},
			Version1_1,
			Version1_0,
		},
	}
	foundOpts = o.GetOption(OptionVersion)
	require.Equal(t, 2, len(foundOpts), "should get two options")
	require.Equal(t, Version1_1, foundOpts[0].(Version))
	require.Equal(t, Version1_0, foundOpts[1].(Version))
}

func TestOptVendorSpecificInformationGetOneOption(t *testing.T) {
	// No option
	o := &OptVendorSpecificInformation{
		[]dhcpv4.Option{
			&OptMessageType{MessageTypeList},
			Version1_1,
		},
	}
	foundOpt := o.GetOneOption(OptionBootImageList)
	require.Nil(t, foundOpt, "should not get options")

	// One option
	o = &OptVendorSpecificInformation{
		[]dhcpv4.Option{
			&OptMessageType{MessageTypeList},
			Version1_1,
		},
	}
	foundOpt = o.GetOneOption(OptionMessageType)
	require.Equal(t, MessageTypeList, foundOpt.(*OptMessageType).Type)

	// Multiple options
	o = &OptVendorSpecificInformation{
		[]dhcpv4.Option{
			&OptMessageType{MessageTypeList},
			Version1_1,
			Version1_0,
		},
	}
	foundOpt = o.GetOneOption(OptionVersion)
	require.Equal(t, Version1_1, foundOpt.(Version))
}
