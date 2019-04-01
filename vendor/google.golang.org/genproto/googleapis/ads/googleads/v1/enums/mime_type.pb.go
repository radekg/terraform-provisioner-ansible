// Code generated by protoc-gen-go. DO NOT EDIT.
// source: google/ads/googleads/v1/enums/mime_type.proto

package enums // import "google.golang.org/genproto/googleapis/ads/googleads/v1/enums"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
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

// The mime type
type MimeTypeEnum_MimeType int32

const (
	// The mime type has not been specified.
	MimeTypeEnum_UNSPECIFIED MimeTypeEnum_MimeType = 0
	// The received value is not known in this version.
	//
	// This is a response-only value.
	MimeTypeEnum_UNKNOWN MimeTypeEnum_MimeType = 1
	// MIME type of image/jpeg.
	MimeTypeEnum_IMAGE_JPEG MimeTypeEnum_MimeType = 2
	// MIME type of image/gif.
	MimeTypeEnum_IMAGE_GIF MimeTypeEnum_MimeType = 3
	// MIME type of image/png.
	MimeTypeEnum_IMAGE_PNG MimeTypeEnum_MimeType = 4
	// MIME type of application/x-shockwave-flash.
	MimeTypeEnum_FLASH MimeTypeEnum_MimeType = 5
	// MIME type of text/html.
	MimeTypeEnum_TEXT_HTML MimeTypeEnum_MimeType = 6
	// MIME type of application/pdf.
	MimeTypeEnum_PDF MimeTypeEnum_MimeType = 7
	// MIME type of application/msword.
	MimeTypeEnum_MSWORD MimeTypeEnum_MimeType = 8
	// MIME type of application/vnd.ms-excel.
	MimeTypeEnum_MSEXCEL MimeTypeEnum_MimeType = 9
	// MIME type of application/rtf.
	MimeTypeEnum_RTF MimeTypeEnum_MimeType = 10
	// MIME type of audio/wav.
	MimeTypeEnum_AUDIO_WAV MimeTypeEnum_MimeType = 11
	// MIME type of audio/mp3.
	MimeTypeEnum_AUDIO_MP3 MimeTypeEnum_MimeType = 12
	// MIME type of application/x-html5-ad-zip.
	MimeTypeEnum_HTML5_AD_ZIP MimeTypeEnum_MimeType = 13
)

var MimeTypeEnum_MimeType_name = map[int32]string{
	0:  "UNSPECIFIED",
	1:  "UNKNOWN",
	2:  "IMAGE_JPEG",
	3:  "IMAGE_GIF",
	4:  "IMAGE_PNG",
	5:  "FLASH",
	6:  "TEXT_HTML",
	7:  "PDF",
	8:  "MSWORD",
	9:  "MSEXCEL",
	10: "RTF",
	11: "AUDIO_WAV",
	12: "AUDIO_MP3",
	13: "HTML5_AD_ZIP",
}
var MimeTypeEnum_MimeType_value = map[string]int32{
	"UNSPECIFIED":  0,
	"UNKNOWN":      1,
	"IMAGE_JPEG":   2,
	"IMAGE_GIF":    3,
	"IMAGE_PNG":    4,
	"FLASH":        5,
	"TEXT_HTML":    6,
	"PDF":          7,
	"MSWORD":       8,
	"MSEXCEL":      9,
	"RTF":          10,
	"AUDIO_WAV":    11,
	"AUDIO_MP3":    12,
	"HTML5_AD_ZIP": 13,
}

func (x MimeTypeEnum_MimeType) String() string {
	return proto.EnumName(MimeTypeEnum_MimeType_name, int32(x))
}
func (MimeTypeEnum_MimeType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_mime_type_54c50942eaf2e264, []int{0, 0}
}

// Container for enum describing the mime types.
type MimeTypeEnum struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *MimeTypeEnum) Reset()         { *m = MimeTypeEnum{} }
func (m *MimeTypeEnum) String() string { return proto.CompactTextString(m) }
func (*MimeTypeEnum) ProtoMessage()    {}
func (*MimeTypeEnum) Descriptor() ([]byte, []int) {
	return fileDescriptor_mime_type_54c50942eaf2e264, []int{0}
}
func (m *MimeTypeEnum) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MimeTypeEnum.Unmarshal(m, b)
}
func (m *MimeTypeEnum) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MimeTypeEnum.Marshal(b, m, deterministic)
}
func (dst *MimeTypeEnum) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MimeTypeEnum.Merge(dst, src)
}
func (m *MimeTypeEnum) XXX_Size() int {
	return xxx_messageInfo_MimeTypeEnum.Size(m)
}
func (m *MimeTypeEnum) XXX_DiscardUnknown() {
	xxx_messageInfo_MimeTypeEnum.DiscardUnknown(m)
}

var xxx_messageInfo_MimeTypeEnum proto.InternalMessageInfo

func init() {
	proto.RegisterType((*MimeTypeEnum)(nil), "google.ads.googleads.v1.enums.MimeTypeEnum")
	proto.RegisterEnum("google.ads.googleads.v1.enums.MimeTypeEnum_MimeType", MimeTypeEnum_MimeType_name, MimeTypeEnum_MimeType_value)
}

func init() {
	proto.RegisterFile("google/ads/googleads/v1/enums/mime_type.proto", fileDescriptor_mime_type_54c50942eaf2e264)
}

var fileDescriptor_mime_type_54c50942eaf2e264 = []byte{
	// 396 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x91, 0xc1, 0x6e, 0x9b, 0x40,
	0x10, 0x86, 0x0b, 0x6e, 0xec, 0x78, 0x6d, 0xb7, 0xab, 0x3d, 0x56, 0xcd, 0x21, 0xb9, 0x77, 0x11,
	0x8a, 0x7a, 0xd9, 0x9e, 0xd6, 0x61, 0x21, 0xb4, 0x06, 0xaf, 0x02, 0xc6, 0x51, 0x84, 0x84, 0x68,
	0x41, 0x08, 0x29, 0xb0, 0xc8, 0x8b, 0x2d, 0xf9, 0x75, 0x7a, 0xec, 0xa3, 0xb4, 0xaf, 0xd0, 0x53,
	0x8f, 0x7d, 0x8a, 0x6a, 0xa1, 0xd8, 0xa7, 0xf6, 0x82, 0xfe, 0x99, 0xff, 0x9b, 0x01, 0xfe, 0x01,
	0xef, 0x0a, 0x21, 0x8a, 0xe7, 0xdc, 0x48, 0x33, 0x69, 0xf4, 0x52, 0xa9, 0x83, 0x69, 0xe4, 0xf5,
	0xbe, 0x92, 0x46, 0x55, 0x56, 0x79, 0xd2, 0x1e, 0x9b, 0x1c, 0x37, 0x3b, 0xd1, 0x0a, 0x74, 0xd5,
	0x33, 0x38, 0xcd, 0x24, 0x3e, 0xe1, 0xf8, 0x60, 0xe2, 0x0e, 0x7f, 0xf3, 0x76, 0xd8, 0xd6, 0x94,
	0x46, 0x5a, 0xd7, 0xa2, 0x4d, 0xdb, 0x52, 0xd4, 0xb2, 0x1f, 0xbe, 0xf9, 0xa9, 0x81, 0xb9, 0x57,
	0x56, 0x79, 0x78, 0x6c, 0x72, 0x56, 0xef, 0xab, 0x9b, 0x1f, 0x1a, 0xb8, 0x1c, 0x1a, 0xe8, 0x35,
	0x98, 0x6d, 0xfc, 0x80, 0xb3, 0x3b, 0xd7, 0x76, 0x99, 0x05, 0x5f, 0xa0, 0x19, 0x98, 0x6c, 0xfc,
	0x4f, 0xfe, 0x7a, 0xeb, 0x43, 0x0d, 0xbd, 0x02, 0xc0, 0xf5, 0xa8, 0xc3, 0x92, 0x8f, 0x9c, 0x39,
	0x50, 0x47, 0x0b, 0x30, 0xed, 0x6b, 0xc7, 0xb5, 0xe1, 0xe8, 0x5c, 0x72, 0xdf, 0x81, 0x2f, 0xd1,
	0x14, 0x5c, 0xd8, 0x2b, 0x1a, 0xdc, 0xc3, 0x0b, 0xe5, 0x84, 0xec, 0x31, 0x4c, 0xee, 0x43, 0x6f,
	0x05, 0xc7, 0x68, 0x02, 0x46, 0xdc, 0xb2, 0xe1, 0x04, 0x01, 0x30, 0xf6, 0x82, 0xed, 0xfa, 0xc1,
	0x82, 0x97, 0xea, 0x4d, 0x5e, 0xc0, 0x1e, 0xef, 0xd8, 0x0a, 0x4e, 0x15, 0xf1, 0x10, 0xda, 0x10,
	0xa8, 0x49, 0xba, 0xb1, 0xdc, 0x75, 0xb2, 0xa5, 0x11, 0x9c, 0x9d, 0x4b, 0x8f, 0xdf, 0xc2, 0x39,
	0x82, 0x60, 0xae, 0x56, 0xbe, 0x4f, 0xa8, 0x95, 0x3c, 0xb9, 0x1c, 0x2e, 0x96, 0xbf, 0x34, 0x70,
	0xfd, 0x45, 0x54, 0xf8, 0xbf, 0x11, 0x2d, 0x17, 0xc3, 0x0f, 0x73, 0x95, 0x09, 0xd7, 0x9e, 0x96,
	0x7f, 0xf9, 0x42, 0x3c, 0xa7, 0x75, 0x81, 0xc5, 0xae, 0x30, 0x8a, 0xbc, 0xee, 0x12, 0x1b, 0x2e,
	0xd2, 0x94, 0xf2, 0x1f, 0x07, 0xfa, 0xd0, 0x3d, 0xbf, 0xea, 0x23, 0x87, 0xd2, 0x6f, 0xfa, 0x95,
	0xd3, 0xaf, 0xa2, 0x99, 0xc4, 0xbd, 0x54, 0x2a, 0x32, 0xb1, 0x4a, 0x5b, 0x7e, 0x1f, 0xfc, 0x98,
	0x66, 0x32, 0x3e, 0xf9, 0x71, 0x64, 0xc6, 0x9d, 0xff, 0x5b, 0xbf, 0xee, 0x9b, 0x84, 0xd0, 0x4c,
	0x12, 0x72, 0x22, 0x08, 0x89, 0x4c, 0x42, 0x3a, 0xe6, 0xf3, 0xb8, 0xfb, 0xb0, 0xdb, 0x3f, 0x01,
	0x00, 0x00, 0xff, 0xff, 0xc4, 0xe8, 0xbe, 0x08, 0x38, 0x02, 0x00, 0x00,
}
