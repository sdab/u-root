// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package acpi

// The IBFT is another brain-dead design.
// Lots of flexibility, which is basically impossible to use due to lots of limits.
// Basic layout is like this:
//	Name		size		offset		description
//	Header		48		0		Primary Header
//	Control		variable	48		Extended Header
//	Initiator	variable			Initiator
//	NIC		variable			NIC
//	Target		variable			Target
//	Heap		variable			Offsets point here.

import (
	"bytes"
	"fmt"
	"reflect"
)

const (
	reserved uint8 = iota
	ibftControl
	ibftInitiator
	ibftNIC
	ibftTarget
	ibftExtensions
)

const (
	ibftHeaderLen    uint16 = 48
	ibftControlLen   uint16 = 18
	ibftInitiatorLen uint16 = 74
	ibftNICLen       uint16 = 102
	ibftTargetLen    uint16 = 54
	ibftHeadersLen   uint16 = ibftHeaderLen + ibftControlLen + ibftInitiatorLen + ibftNICLen + ibftTargetLen + ibftNICLen + ibftTargetLen
)

const (
	ibftVersion uint8 = 1 // in all cases since 2009
)

// We created a bunch of structs here so you don't have to go read the Big Bad Book of ACPI.
// It turned out to be easier to use the structs we defined further below.
// These structs were generated by running scripts across the pdf.

// acpiIBFTHeader defines the acpiIBFTHeader
type acpiIBFTHeader struct {
	Signature  [4]byte  `offset:"0" desc:"iBFT Signature for the iSCSI Boot Firmware Table"`
	Length     uint32   `offset:"4" desc:"Length in bytes of the entire IBFT, including the signature"`
	Revision   uint8    `offset:"8" desc:"Revision = 1"`
	Checksum   uint8    `offset:"9" desc:"Entire table must sum to zero"`
	OEMID      [6]byte  `offset:"10" desc:"OEM ID. All unused trailing bytes must be zero. [acpi-OEMID]"`
	OEMTableID [8]byte  `offset:"16" desc:"For the iBFT the Table ID is the Manufacturer‟s Model ID. All unused trailing bytes must be zero."`
	Reserved   [24]byte `offset:"24" desc:"Reserved"`
}

var rawIBTFHeader = "IBFT\x00\x08\x00\x001\x00ACPIXXACPISUCK1234567890abcdef01234567"

// acpiIBFTStructHeader defines the common components of the structure headers.
// In the standard, IBM made the flags common, even though the values
// are all over the place. We don't do that.
// In a fit of genius, whoever designed this made Bit 0 sometimes
// mean valid, sometimes not. Awesome!
type acpiIBFTStructHeader struct {
	ID      byte   `offset:"0" desc:"ID"`
	Version uint8  `offset:"1" desc:"Structure Version"`
	Length  uint16 `offset:"2" desc:"Structure Length"`
	Index   uint8  `offset:"4" desc:"Index"`
}

type acpiIBFTControlFlags uint8

const (
	ibftMultiLogin acpiIBFTControlFlags = iota
	ibftSingleLogin
)

type acpiIBFTControl struct {
	acpiIBFTStructHeader
	Flags      acpiIBFTControlFlags `offset:"0" desc:"Bit 0 : Target Login Mode Control 0 = Multi-Login Mode 1 = Single Login Mode"`
	Extensions uint16               `offset:"6" desc:"Optional. If unused must be zero. If used, must point to an Extensions Structure with a standard Structure header."`
	Initiator  uint16               `offset:"8" desc:""`
	NIC0       uint16               `offset:"10" desc:""`
	Target0    uint16               `offset:"12" desc:""`
	NIC1       uint16               `offset:"14" desc:""`
	Target1    uint16               `offset:"16" desc:""`
}

// pre-filled-in control structure.
var control = acpiIBFTControl{
	acpiIBFTStructHeader: acpiIBFTStructHeader{
		ID:      ibftControl,
		Version: 1,
		Length:  80, // round the next struct to 128 bytes
		Index:   0,
	},
	Flags: ibftSingleLogin,
	// Just 64-byte align everything. Anything else is painful.
	Extensions: 0,
	Initiator:  ibftHeaderLen + ibftControlLen,
	NIC0:       ibftHeaderLen + ibftControlLen + ibftInitiatorLen,
	Target0:    ibftHeaderLen + ibftControlLen + ibftInitiatorLen + ibftNICLen,
	NIC1:       ibftHeaderLen + ibftControlLen + ibftInitiatorLen + ibftNICLen + ibftTargetLen,
	Target1:    ibftHeaderLen + ibftControlLen + ibftInitiatorLen + ibftNICLen + ibftTargetLen + ibftNICLen,
	// heap starts here.
}

type acpiIBFTInitiatorFlags uint8

const (
	acpiIBFTInitiatorValid        acpiIBFTInitiatorFlags = 1
	acpiIBFTInitiatorFirmwareBoot                        = 2
)

type acpiIBFTInitiator struct {
	acpiIBFTStructHeader
	Flags                 acpiIBFTInitiatorFlags `desc:"Bit0:  block valid flag 0 = no, 1 = yes Bit1 : Firmware Boot Selected Flag 0 = no, 1 = yes"`
	iSNSServer            [16]uint8              `offset:"6" desc:"IP Address"`
	SLPServer             [16]uint8              `offset:"22" desc:"IP Address"`
	PrimaryRadiusServer   [16]uint8              `offset:"38" desc:"IP Address"`
	SecondaryRadiusServer [16]uint8              `offset:"54" desc:"IP Address"`
	InitiatorNameLength   uint16                 `offset:"70" desc:"Heap Entry Length"`
	InitiatorNameOffset   uint16                 `offset:"72" desc:"Offset from the beginning of the iBFT"`
}

type acpiIBFTNICFlags uint8

const (
	acpiIBFTNICValid        acpiIBFTNICFlags = 1
	acpiIBFTNICBootSelected                  = 2
	acpiIBFTNICGlobal                        = 4
)

type acpiIBFTNIC struct {
	acpiIBFTStructHeader
	Flags          acpiIBFTNICFlags `desc:"Bit0:  block valid flag 0 = no, 1 = yes Bit1 : Firmware Boot Selected Flag 0 = no, 1 = yes Bit2 : Global / Link Local 0 = Link Local, 1 = Global"`
	IPAddress      [16]uint8        `offset:"6" desc:"IP Address Subnet Mask Prefix 1 22 The mask prefix length. For example, 255.255.255.0 has a prefix length of 24"`
	Origin         uint8            `offset:"23" desc:"See [origin]"`
	Gateway        [16]uint8        `offset:"24" desc:"IP Address"`
	PrimaryDNS     [16]uint8        `offset:"40" desc:"IP Address"`
	SecondaryDNS   [16]uint8        `offset:"56" desc:"IP Address"`
	DHCP           [16]uint8        `offset:"72" desc:"IP Address"`
	VLAN           uint16           `offset:"88" desc:"VLAN"`
	MACAddress     [6]uint8         `offset:"90" desc:"MAC Address"`
	PCIBDF         uint16           `offset:"96" desc:"Bus = 8 bits Device = 5 bits Function = 3 bits"`
	HostNameLength uint16           `offset:"98" desc:"Heap Entry Length"`
	HostNameOffset uint16           `offset:"100" desc:"Offset from the beginning of the iBFT In a DHCP scenario this can be the name stored as Option 12 host-name."`
}

type acpiIBFTTargetFlags uint8

const (
	acpiIBFTTargetValid        acpiIBFTTargetFlags = 1
	acpiIBFTTargetBootSelected                     = 2
	acpiIBFTTargetRadiusChap                       = 4
	acpiIBFTTargetRadiusrChap                      = 8
)

type acpiIBFTTarget struct {
	acpiIBFTStructHeader
	Flags                   acpiIBFTTargetFlags `desc:"Bit0: block valid flag 0 = no, 1 = yes Bit1 : Firmware Boot Selected Flag 0 = no, 1 = yes Bit2 : Use Radius CHAP 0 = no, 1 = yes Bit3 : Use Radius rCHAP 0 = no, 1 = yes"`
	TargetIPAddress         [16]uint8           `offset:"6" desc:"IP Address"`
	TargetIPSocket          uint16              `offset:"22" desc:"Likely 3260"`
	TargetBootLUN           uint64              `offset:"24" desc:"See [iscsi] Little Endian Quad Word"`
	CHAPType                uint8               `offset:"32" desc:"0 = No CHAP 1 = CHAP 2 = Mutual CHAP"`
	NICAssociation          uint8               `offset:"33" desc:"NIC Index"`
	TargetNameLength        uint16              `offset:"34" desc:"Heap Entry Length"`
	TargetNameOffset        uint16              `offset:"36" desc:"Offset from the beginning of the iBFT"`
	CHAPNameLength          uint16              `offset:"38" desc:"Heap Entry Length"`
	CHAPNameOffset          uint16              `offset:"40" desc:"Offset from the beginning of the iBFT"`
	CHAPSecretLength        uint16              `offset:"42" desc:"Heap Entry Length"`
	CHAPSecretOffset        uint16              `offset:"44" desc:"Offset from the beginning of the iBFT"`
	ReverseCHAPNameLength   uint16              `offset:"46" desc:"Heap Entry Length"`
	ReverseCHAPNameOffset   uint16              `offset:"48" desc:"Offset from the beginning of the iBFT"`
	ReverseCHAPSecretLength uint16              `offset:"50" desc:"Heap Entry Length"`
	ReverseCHAPSecretOffset uint16              `offset:"52" desc:"Offset from the beginning of the iBFT"`
}

// The structs below are designed to be JSON friendly.
// Things are strings, but they are typed, to make marshaling
// to a table easy and setting up initialized values easy.
// The jury is out on whether this is right, but so far
// we're finding it very convenient.
// Of partiular value is having an exported struct member
// which can be ignore at marshal time. Of course, this is also
// very easy with tags, but in trying to use both, I found
// this schema like scheme easier to write to.

// IBFTInitiator defines an initiator
type IBFTInitiator struct {
	Valid                 flag
	Boot                  flag
	SNSServer             ipaddr
	SLPServer             ipaddr
	PrimaryRadiusServer   ipaddr
	SecondaryRadiusServer ipaddr
	Name                  sheap
}

// IBFTNIC defines an IBFT NIC structure.
type IBFTNIC struct {
	Valid        flag
	Boot         flag
	Global       flag
	Index        flag
	IPAddress    ipaddr
	SubNet       u8
	Origin       u8
	Gateway      ipaddr
	PrimaryDNS   ipaddr
	SecondaryDNS ipaddr
	DHCP         ipaddr
	VLAN         u16
	MACAddress   mac
	PCIBDF       bdf
	HostName     sheap
}

// IBFTTarget defines an IBFT target, a.k.a. server
type IBFTTarget struct {
	Valid             flag
	Boot              flag
	CHAP              flag
	RCHAP             flag     // can you do both? Standard implies yes.
	Index             flag     // 0 or 1
	TargetIP          sockaddr // in host:port format
	BootLUN           u64
	ChapType          u8
	Association       u8
	TargetName        sheap
	CHAPName          sheap
	CHAPSecret        sheap
	ReverseCHAPName   sheap
	ReverseCHAPSecret sheap
}

// IBFT defines all the bits of an IBFT users might want to set.
type IBFT struct {
	Generic
	// Control
	Multi     flag
	Initiator IBFTInitiator
	NIC0      IBFTNIC
	Target0   IBFTTarget
	NIC1      IBFTNIC
	Target1   IBFTTarget
}

// Marshal marshals an IBFT to a byte slice. It is somewhat complicated
// by the fact that we need to marshal to two things, a header and a heao;
// and record pointers to the heap in the head.
func (ibft *IBFT) Marshal() ([]byte, error) {
	var h = HeapTable{Head: &bytes.Buffer{}, Heap: &bytes.Buffer{}}
	Debug("IBFT")
	f, err := flags(ibft.Multi)
	if err != nil {
		return nil, err
	}
	control.Flags = acpiIBFTControlFlags(f)
	w(h.Head, 1, []byte(rawIBTFHeader), control)
	Debug("Done IBFTHeader: head is %d bytes", h.Head.Len())
	if err := mIBFT(&h, ibft); err != nil {
		return nil, err
	}
	if h.Head.Len() != int(ibftHeadersLen) {
		return nil, fmt.Errorf("Expected headers len is wrong; got %d, want %d", h.Head.Len(), ibftHeadersLen)
	}
	w(h.Head, 1, h.Heap.Bytes())

	return h.Head.Bytes(), nil
}

// mIBFT is the workhorse of IBFT marshaling.
func mIBFT(h *HeapTable, i interface{}) error {
	nt := reflect.TypeOf(i).Elem()
	nv := reflect.ValueOf(i).Elem()
	for i := 0; i < nt.NumField(); i++ {
		f := nt.Field(i)
		ft := f.Type
		fv := nv.Field(i)

		Debug("Field %d: (%d, %d) ml %v %T (%v, %v)", i, h.Head.Len(), h.Heap.Len(), f, f, ft, fv)
		switch s := fv.Interface().(type) {
		case Generic:
			// This is not used yet. When we started this work, we never thought we'd
			// need to go this far.
		case IBFTInitiator:
			f, err := flags(s.Valid, s.Boot)
			if err != nil {
				return fmt.Errorf("Parsing %v: %v", []flag{s.Valid, s.Boot}, err)
			}
			// we can do this hack with Index; it will only ever be
			// 0 or 1. The IBFT allows lots, in principle, but only 2, in practice.
			w(h.Head, ibftInitiator, ibftVersion, ibftInitiatorLen, uint8(0), f)
			Debug("Wrote initiatior header len is %d", h.Head.Len())
			if err := mIBFT(h, &s); err != nil {
				return err
			}
		case IBFTNIC:
			f, err := flags(s.Valid, s.Boot, s.Global)
			if err != nil {
				return fmt.Errorf("Parsing %v: %v", []flag{s.Valid, s.Boot, s.Global}, err)
			}
			// we can do this hack with Index; it will only ever be
			// 0 or 1. See above snarky comment.
			x, err := flags(s.Index)
			if err != nil {
				return fmt.Errorf("Parsing NICIndex %s: %v", s.Index, err)
			}
			w(h.Head, ibftNIC, ibftVersion, ibftNICLen, x, f)
			if err := mIBFT(h, &s); err != nil {
				return err
			}

		case IBFTTarget:
			f, err := flags(s.Valid, s.Boot, s.CHAP, s.RCHAP)
			if err != nil {
				return fmt.Errorf("Parsing %v: %v", []flag{s.Valid, s.Boot, s.CHAP, s.RCHAP}, err)
			}
			x, err := flags(s.Index)
			if err != nil {
				return fmt.Errorf("Parsing NICIndex %s: %v", s.Index, err)
			}
			w(h.Head, ibftTarget, ibftVersion, ibftTargetLen, x, f)
			if err := mIBFT(h, &s); err != nil {
				return err
			}

		default:
			if err := h.Marshal(s); err != nil {
				return err
			}
		}
	}
	Debug("mIBFT done, head is %d bytes, heap is %d bytes", h.Head.Len(), h.Heap.Len())
	return nil
}
