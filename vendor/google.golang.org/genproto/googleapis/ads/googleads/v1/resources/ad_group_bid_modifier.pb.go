// Code generated by protoc-gen-go. DO NOT EDIT.
// source: google/ads/googleads/v1/resources/ad_group_bid_modifier.proto

package resources // import "google.golang.org/genproto/googleapis/ads/googleads/v1/resources"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import wrappers "github.com/golang/protobuf/ptypes/wrappers"
import common "google.golang.org/genproto/googleapis/ads/googleads/v1/common"
import enums "google.golang.org/genproto/googleapis/ads/googleads/v1/enums"
import _ "google.golang.org/genproto/googleapis/api/annotations"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// Represents an ad group bid modifier.
type AdGroupBidModifier struct {
	// The resource name of the ad group bid modifier.
	// Ad group bid modifier resource names have the form:
	//
	// `customers/{customer_id}/adGroupBidModifiers/{ad_group_id}~{criterion_id}`
	ResourceName string `protobuf:"bytes,1,opt,name=resource_name,json=resourceName,proto3" json:"resource_name,omitempty"`
	// The ad group to which this criterion belongs.
	AdGroup *wrappers.StringValue `protobuf:"bytes,2,opt,name=ad_group,json=adGroup,proto3" json:"ad_group,omitempty"`
	// The ID of the criterion to bid modify.
	//
	// This field is ignored for mutates.
	CriterionId *wrappers.Int64Value `protobuf:"bytes,3,opt,name=criterion_id,json=criterionId,proto3" json:"criterion_id,omitempty"`
	// The modifier for the bid when the criterion matches. The modifier must be
	// in the range: 0.1 - 10.0. The range is 1.0 - 6.0 for PreferredContent.
	// Use 0 to opt out of a Device type.
	BidModifier *wrappers.DoubleValue `protobuf:"bytes,4,opt,name=bid_modifier,json=bidModifier,proto3" json:"bid_modifier,omitempty"`
	// The base ad group from which this draft/trial adgroup bid modifier was
	// created. If ad_group is a base ad group then this field will be equal to
	// ad_group. If the ad group was created in the draft or trial and has no
	// corresponding base ad group, then this field will be null.
	// This field is readonly.
	BaseAdGroup *wrappers.StringValue `protobuf:"bytes,9,opt,name=base_ad_group,json=baseAdGroup,proto3" json:"base_ad_group,omitempty"`
	// Bid modifier source.
	BidModifierSource enums.BidModifierSourceEnum_BidModifierSource `protobuf:"varint,10,opt,name=bid_modifier_source,json=bidModifierSource,proto3,enum=google.ads.googleads.v1.enums.BidModifierSourceEnum_BidModifierSource" json:"bid_modifier_source,omitempty"`
	// The criterion of this ad group bid modifier.
	//
	// Types that are valid to be assigned to Criterion:
	//	*AdGroupBidModifier_HotelDateSelectionType
	//	*AdGroupBidModifier_HotelAdvanceBookingWindow
	//	*AdGroupBidModifier_HotelLengthOfStay
	//	*AdGroupBidModifier_HotelCheckInDay
	//	*AdGroupBidModifier_Device
	//	*AdGroupBidModifier_PreferredContent
	Criterion            isAdGroupBidModifier_Criterion `protobuf_oneof:"criterion"`
	XXX_NoUnkeyedLiteral struct{}                       `json:"-"`
	XXX_unrecognized     []byte                         `json:"-"`
	XXX_sizecache        int32                          `json:"-"`
}

func (m *AdGroupBidModifier) Reset()         { *m = AdGroupBidModifier{} }
func (m *AdGroupBidModifier) String() string { return proto.CompactTextString(m) }
func (*AdGroupBidModifier) ProtoMessage()    {}
func (*AdGroupBidModifier) Descriptor() ([]byte, []int) {
	return fileDescriptor_ad_group_bid_modifier_69cb3cea81ff95be, []int{0}
}
func (m *AdGroupBidModifier) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AdGroupBidModifier.Unmarshal(m, b)
}
func (m *AdGroupBidModifier) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AdGroupBidModifier.Marshal(b, m, deterministic)
}
func (dst *AdGroupBidModifier) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AdGroupBidModifier.Merge(dst, src)
}
func (m *AdGroupBidModifier) XXX_Size() int {
	return xxx_messageInfo_AdGroupBidModifier.Size(m)
}
func (m *AdGroupBidModifier) XXX_DiscardUnknown() {
	xxx_messageInfo_AdGroupBidModifier.DiscardUnknown(m)
}

var xxx_messageInfo_AdGroupBidModifier proto.InternalMessageInfo

func (m *AdGroupBidModifier) GetResourceName() string {
	if m != nil {
		return m.ResourceName
	}
	return ""
}

func (m *AdGroupBidModifier) GetAdGroup() *wrappers.StringValue {
	if m != nil {
		return m.AdGroup
	}
	return nil
}

func (m *AdGroupBidModifier) GetCriterionId() *wrappers.Int64Value {
	if m != nil {
		return m.CriterionId
	}
	return nil
}

func (m *AdGroupBidModifier) GetBidModifier() *wrappers.DoubleValue {
	if m != nil {
		return m.BidModifier
	}
	return nil
}

func (m *AdGroupBidModifier) GetBaseAdGroup() *wrappers.StringValue {
	if m != nil {
		return m.BaseAdGroup
	}
	return nil
}

func (m *AdGroupBidModifier) GetBidModifierSource() enums.BidModifierSourceEnum_BidModifierSource {
	if m != nil {
		return m.BidModifierSource
	}
	return enums.BidModifierSourceEnum_UNSPECIFIED
}

type isAdGroupBidModifier_Criterion interface {
	isAdGroupBidModifier_Criterion()
}

type AdGroupBidModifier_HotelDateSelectionType struct {
	HotelDateSelectionType *common.HotelDateSelectionTypeInfo `protobuf:"bytes,5,opt,name=hotel_date_selection_type,json=hotelDateSelectionType,proto3,oneof"`
}

type AdGroupBidModifier_HotelAdvanceBookingWindow struct {
	HotelAdvanceBookingWindow *common.HotelAdvanceBookingWindowInfo `protobuf:"bytes,6,opt,name=hotel_advance_booking_window,json=hotelAdvanceBookingWindow,proto3,oneof"`
}

type AdGroupBidModifier_HotelLengthOfStay struct {
	HotelLengthOfStay *common.HotelLengthOfStayInfo `protobuf:"bytes,7,opt,name=hotel_length_of_stay,json=hotelLengthOfStay,proto3,oneof"`
}

type AdGroupBidModifier_HotelCheckInDay struct {
	HotelCheckInDay *common.HotelCheckInDayInfo `protobuf:"bytes,8,opt,name=hotel_check_in_day,json=hotelCheckInDay,proto3,oneof"`
}

type AdGroupBidModifier_Device struct {
	Device *common.DeviceInfo `protobuf:"bytes,11,opt,name=device,proto3,oneof"`
}

type AdGroupBidModifier_PreferredContent struct {
	PreferredContent *common.PreferredContentInfo `protobuf:"bytes,12,opt,name=preferred_content,json=preferredContent,proto3,oneof"`
}

func (*AdGroupBidModifier_HotelDateSelectionType) isAdGroupBidModifier_Criterion() {}

func (*AdGroupBidModifier_HotelAdvanceBookingWindow) isAdGroupBidModifier_Criterion() {}

func (*AdGroupBidModifier_HotelLengthOfStay) isAdGroupBidModifier_Criterion() {}

func (*AdGroupBidModifier_HotelCheckInDay) isAdGroupBidModifier_Criterion() {}

func (*AdGroupBidModifier_Device) isAdGroupBidModifier_Criterion() {}

func (*AdGroupBidModifier_PreferredContent) isAdGroupBidModifier_Criterion() {}

func (m *AdGroupBidModifier) GetCriterion() isAdGroupBidModifier_Criterion {
	if m != nil {
		return m.Criterion
	}
	return nil
}

func (m *AdGroupBidModifier) GetHotelDateSelectionType() *common.HotelDateSelectionTypeInfo {
	if x, ok := m.GetCriterion().(*AdGroupBidModifier_HotelDateSelectionType); ok {
		return x.HotelDateSelectionType
	}
	return nil
}

func (m *AdGroupBidModifier) GetHotelAdvanceBookingWindow() *common.HotelAdvanceBookingWindowInfo {
	if x, ok := m.GetCriterion().(*AdGroupBidModifier_HotelAdvanceBookingWindow); ok {
		return x.HotelAdvanceBookingWindow
	}
	return nil
}

func (m *AdGroupBidModifier) GetHotelLengthOfStay() *common.HotelLengthOfStayInfo {
	if x, ok := m.GetCriterion().(*AdGroupBidModifier_HotelLengthOfStay); ok {
		return x.HotelLengthOfStay
	}
	return nil
}

func (m *AdGroupBidModifier) GetHotelCheckInDay() *common.HotelCheckInDayInfo {
	if x, ok := m.GetCriterion().(*AdGroupBidModifier_HotelCheckInDay); ok {
		return x.HotelCheckInDay
	}
	return nil
}

func (m *AdGroupBidModifier) GetDevice() *common.DeviceInfo {
	if x, ok := m.GetCriterion().(*AdGroupBidModifier_Device); ok {
		return x.Device
	}
	return nil
}

func (m *AdGroupBidModifier) GetPreferredContent() *common.PreferredContentInfo {
	if x, ok := m.GetCriterion().(*AdGroupBidModifier_PreferredContent); ok {
		return x.PreferredContent
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*AdGroupBidModifier) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _AdGroupBidModifier_OneofMarshaler, _AdGroupBidModifier_OneofUnmarshaler, _AdGroupBidModifier_OneofSizer, []interface{}{
		(*AdGroupBidModifier_HotelDateSelectionType)(nil),
		(*AdGroupBidModifier_HotelAdvanceBookingWindow)(nil),
		(*AdGroupBidModifier_HotelLengthOfStay)(nil),
		(*AdGroupBidModifier_HotelCheckInDay)(nil),
		(*AdGroupBidModifier_Device)(nil),
		(*AdGroupBidModifier_PreferredContent)(nil),
	}
}

func _AdGroupBidModifier_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*AdGroupBidModifier)
	// criterion
	switch x := m.Criterion.(type) {
	case *AdGroupBidModifier_HotelDateSelectionType:
		b.EncodeVarint(5<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.HotelDateSelectionType); err != nil {
			return err
		}
	case *AdGroupBidModifier_HotelAdvanceBookingWindow:
		b.EncodeVarint(6<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.HotelAdvanceBookingWindow); err != nil {
			return err
		}
	case *AdGroupBidModifier_HotelLengthOfStay:
		b.EncodeVarint(7<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.HotelLengthOfStay); err != nil {
			return err
		}
	case *AdGroupBidModifier_HotelCheckInDay:
		b.EncodeVarint(8<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.HotelCheckInDay); err != nil {
			return err
		}
	case *AdGroupBidModifier_Device:
		b.EncodeVarint(11<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Device); err != nil {
			return err
		}
	case *AdGroupBidModifier_PreferredContent:
		b.EncodeVarint(12<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.PreferredContent); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("AdGroupBidModifier.Criterion has unexpected type %T", x)
	}
	return nil
}

func _AdGroupBidModifier_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*AdGroupBidModifier)
	switch tag {
	case 5: // criterion.hotel_date_selection_type
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(common.HotelDateSelectionTypeInfo)
		err := b.DecodeMessage(msg)
		m.Criterion = &AdGroupBidModifier_HotelDateSelectionType{msg}
		return true, err
	case 6: // criterion.hotel_advance_booking_window
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(common.HotelAdvanceBookingWindowInfo)
		err := b.DecodeMessage(msg)
		m.Criterion = &AdGroupBidModifier_HotelAdvanceBookingWindow{msg}
		return true, err
	case 7: // criterion.hotel_length_of_stay
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(common.HotelLengthOfStayInfo)
		err := b.DecodeMessage(msg)
		m.Criterion = &AdGroupBidModifier_HotelLengthOfStay{msg}
		return true, err
	case 8: // criterion.hotel_check_in_day
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(common.HotelCheckInDayInfo)
		err := b.DecodeMessage(msg)
		m.Criterion = &AdGroupBidModifier_HotelCheckInDay{msg}
		return true, err
	case 11: // criterion.device
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(common.DeviceInfo)
		err := b.DecodeMessage(msg)
		m.Criterion = &AdGroupBidModifier_Device{msg}
		return true, err
	case 12: // criterion.preferred_content
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(common.PreferredContentInfo)
		err := b.DecodeMessage(msg)
		m.Criterion = &AdGroupBidModifier_PreferredContent{msg}
		return true, err
	default:
		return false, nil
	}
}

func _AdGroupBidModifier_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*AdGroupBidModifier)
	// criterion
	switch x := m.Criterion.(type) {
	case *AdGroupBidModifier_HotelDateSelectionType:
		s := proto.Size(x.HotelDateSelectionType)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *AdGroupBidModifier_HotelAdvanceBookingWindow:
		s := proto.Size(x.HotelAdvanceBookingWindow)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *AdGroupBidModifier_HotelLengthOfStay:
		s := proto.Size(x.HotelLengthOfStay)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *AdGroupBidModifier_HotelCheckInDay:
		s := proto.Size(x.HotelCheckInDay)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *AdGroupBidModifier_Device:
		s := proto.Size(x.Device)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *AdGroupBidModifier_PreferredContent:
		s := proto.Size(x.PreferredContent)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

func init() {
	proto.RegisterType((*AdGroupBidModifier)(nil), "google.ads.googleads.v1.resources.AdGroupBidModifier")
}

func init() {
	proto.RegisterFile("google/ads/googleads/v1/resources/ad_group_bid_modifier.proto", fileDescriptor_ad_group_bid_modifier_69cb3cea81ff95be)
}

var fileDescriptor_ad_group_bid_modifier_69cb3cea81ff95be = []byte{
	// 694 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x54, 0xdd, 0x6e, 0xd3, 0x48,
	0x14, 0xde, 0xa4, 0xbb, 0xfd, 0x99, 0xa4, 0xbb, 0x5b, 0xef, 0x6a, 0xd7, 0x94, 0x0a, 0xb5, 0xa0,
	0x4a, 0x15, 0x12, 0xb6, 0xd2, 0x16, 0x2a, 0x19, 0x15, 0x48, 0x1a, 0x68, 0x83, 0xf8, 0xa9, 0x12,
	0x14, 0x24, 0x14, 0x69, 0x34, 0xf6, 0x9c, 0x38, 0xa3, 0xc6, 0x33, 0x96, 0x3d, 0x49, 0x94, 0x3b,
	0x2e, 0x78, 0x12, 0x2e, 0xb9, 0xe1, 0x3d, 0x78, 0x14, 0x9e, 0x02, 0x79, 0x66, 0x6c, 0x85, 0xb6,
	0xa1, 0xb9, 0x3b, 0x3e, 0xe7, 0xfb, 0x39, 0xe7, 0x8c, 0x67, 0xd0, 0x71, 0x28, 0x44, 0x38, 0x04,
	0x97, 0xd0, 0xd4, 0xd5, 0x61, 0x16, 0x8d, 0x6b, 0x6e, 0x02, 0xa9, 0x18, 0x25, 0x01, 0xa4, 0x2e,
	0xa1, 0x38, 0x4c, 0xc4, 0x28, 0xc6, 0x3e, 0xa3, 0x38, 0x12, 0x94, 0xf5, 0x19, 0x24, 0x4e, 0x9c,
	0x08, 0x29, 0xac, 0x1d, 0xcd, 0x71, 0x08, 0x4d, 0x9d, 0x82, 0xee, 0x8c, 0x6b, 0x4e, 0x41, 0xdf,
	0x7c, 0x30, 0xcf, 0x21, 0x10, 0x51, 0x24, 0xb8, 0x1b, 0x24, 0x4c, 0x42, 0xc2, 0x88, 0x56, 0xdc,
	0x3c, 0x9a, 0x07, 0x07, 0x3e, 0x8a, 0x52, 0x77, 0xb6, 0x07, 0xac, 0x2d, 0x0c, 0xf1, 0x8e, 0x21,
	0xaa, 0x2f, 0x7f, 0xd4, 0x77, 0x27, 0x09, 0x89, 0x63, 0x48, 0x52, 0x53, 0xdf, 0xca, 0x85, 0x63,
	0xe6, 0x12, 0xce, 0x85, 0x24, 0x92, 0x09, 0x6e, 0xaa, 0x77, 0xbf, 0xae, 0x22, 0xab, 0x4e, 0x4f,
	0xb3, 0x39, 0x1b, 0x8c, 0xbe, 0x36, 0x0e, 0xd6, 0x3d, 0xb4, 0x9e, 0x4f, 0x82, 0x39, 0x89, 0xc0,
	0x2e, 0x6d, 0x97, 0xf6, 0xd6, 0xda, 0xd5, 0x3c, 0xf9, 0x86, 0x44, 0x60, 0x1d, 0xa1, 0xd5, 0x7c,
	0x47, 0x76, 0x79, 0xbb, 0xb4, 0x57, 0xd9, 0xdf, 0x32, 0xcb, 0x70, 0xf2, 0x66, 0x9c, 0x8e, 0x4c,
	0x18, 0x0f, 0xbb, 0x64, 0x38, 0x82, 0xf6, 0x0a, 0xd1, 0x46, 0xd6, 0x13, 0x54, 0x35, 0xd3, 0x0b,
	0x8e, 0x19, 0xb5, 0x97, 0x14, 0xf9, 0xf6, 0x15, 0x72, 0x8b, 0xcb, 0x47, 0x87, 0x9a, 0x5b, 0x29,
	0x08, 0x2d, 0x6a, 0x3d, 0x45, 0xd5, 0xd9, 0x7d, 0xd8, 0xbf, 0xcf, 0x31, 0x6f, 0x8a, 0x91, 0x3f,
	0x04, 0x23, 0xe0, 0xcf, 0x8c, 0xf7, 0x0c, 0xad, 0xfb, 0x24, 0x05, 0x5c, 0xb4, 0xbf, 0xb6, 0x40,
	0xfb, 0x95, 0x8c, 0x62, 0x76, 0x65, 0x8d, 0xd1, 0x3f, 0xd7, 0x1c, 0x89, 0x8d, 0xb6, 0x4b, 0x7b,
	0x7f, 0xee, 0xbf, 0x70, 0xe6, 0xfd, 0x1e, 0xea, 0x30, 0x9d, 0x99, 0x4d, 0x77, 0x14, 0xef, 0x39,
	0x1f, 0x45, 0x57, 0xb3, 0xed, 0x0d, 0xff, 0x72, 0xca, 0x9a, 0xa0, 0x5b, 0x03, 0x21, 0x61, 0x88,
	0x29, 0x91, 0x80, 0x53, 0x18, 0x42, 0x90, 0x1d, 0x27, 0x96, 0xd3, 0x18, 0xec, 0x3f, 0xd4, 0x14,
	0xde, 0x5c, 0x77, 0xfd, 0xe7, 0x39, 0x67, 0x99, 0x40, 0x93, 0x48, 0xe8, 0xe4, 0xf4, 0x77, 0xd3,
	0x18, 0x5a, 0xbc, 0x2f, 0xce, 0x7e, 0x6b, 0xff, 0x37, 0xb8, 0xb6, 0x6a, 0x7d, 0x2c, 0xa1, 0x2d,
	0xed, 0x4c, 0xe8, 0x98, 0xf0, 0x00, 0xb0, 0x2f, 0xc4, 0x05, 0xe3, 0x21, 0x9e, 0x30, 0x4e, 0xc5,
	0xc4, 0x5e, 0x56, 0xe6, 0xc7, 0x0b, 0x99, 0xd7, 0xb5, 0x44, 0x43, 0x2b, 0xbc, 0x57, 0x02, 0xc6,
	0x5f, 0x8f, 0x77, 0x1d, 0xc0, 0x1a, 0xa0, 0x7f, 0x75, 0x07, 0x43, 0xe0, 0xa1, 0x1c, 0x60, 0xd1,
	0xc7, 0xa9, 0x24, 0x53, 0x7b, 0x45, 0x39, 0x3f, 0x5c, 0xc8, 0xf9, 0x95, 0xa2, 0xbe, 0xed, 0x77,
	0x24, 0x99, 0x1a, 0xc7, 0x8d, 0xc1, 0xe5, 0x82, 0xe5, 0x23, 0x4b, 0x3b, 0x05, 0x03, 0x08, 0x2e,
	0x30, 0xe3, 0x98, 0x92, 0xa9, 0xbd, 0xaa, 0x7c, 0x0e, 0x16, 0xf2, 0x39, 0xc9, 0x88, 0x2d, 0xde,
	0x2c, 0x5c, 0xfe, 0x1a, 0xfc, 0x9c, 0xb6, 0x9a, 0x68, 0x99, 0xc2, 0x98, 0x05, 0x60, 0x57, 0x94,
	0xee, 0xfd, 0x9b, 0x74, 0x9b, 0x0a, 0x6d, 0xe4, 0x0c, 0xd7, 0x0a, 0xd0, 0x46, 0x9c, 0x40, 0x1f,
	0x92, 0x04, 0x28, 0x0e, 0x04, 0x97, 0xc0, 0xa5, 0x5d, 0x55, 0x82, 0x87, 0x37, 0x09, 0x9e, 0xe7,
	0xc4, 0x13, 0xcd, 0x33, 0xd2, 0x7f, 0xc7, 0x97, 0xf2, 0x8d, 0x0a, 0x5a, 0x2b, 0xae, 0x5f, 0xe3,
	0x53, 0x19, 0xed, 0x06, 0x22, 0x72, 0x6e, 0x7c, 0x01, 0x1b, 0xff, 0x5f, 0x7d, 0x58, 0xce, 0xb3,
	0x8b, 0x75, 0x5e, 0xfa, 0xf0, 0xd2, 0xb0, 0x43, 0x31, 0x24, 0x3c, 0x74, 0x44, 0x12, 0xba, 0x21,
	0x70, 0x75, 0xed, 0xf2, 0xd7, 0x2f, 0x66, 0xe9, 0x2f, 0x5e, 0xe7, 0xc7, 0x45, 0xf4, 0xb9, 0xbc,
	0x74, 0x5a, 0xaf, 0x7f, 0x29, 0xef, 0x9c, 0x6a, 0xc9, 0x3a, 0x4d, 0x1d, 0x1d, 0x66, 0x51, 0xb7,
	0xe6, 0xb4, 0x73, 0xe4, 0xb7, 0x1c, 0xd3, 0xab, 0xd3, 0xb4, 0x57, 0x60, 0x7a, 0xdd, 0x5a, 0xaf,
	0xc0, 0x7c, 0x2f, 0xef, 0xea, 0x82, 0xe7, 0xd5, 0x69, 0xea, 0x79, 0x05, 0xca, 0xf3, 0xba, 0x35,
	0xcf, 0x2b, 0x70, 0xfe, 0xb2, 0x6a, 0xf6, 0xe0, 0x47, 0x00, 0x00, 0x00, 0xff, 0xff, 0xb4, 0xc7,
	0xdc, 0x80, 0x49, 0x06, 0x00, 0x00,
}
