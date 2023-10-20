// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.12.4
// source: greeeing.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type GreetingServiceRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *GreetingServiceRequest) Reset() {
	*x = GreetingServiceRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_greeeing_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GreetingServiceRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GreetingServiceRequest) ProtoMessage() {}

func (x *GreetingServiceRequest) ProtoReflect() protoreflect.Message {
	mi := &file_greeeing_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GreetingServiceRequest.ProtoReflect.Descriptor instead.
func (*GreetingServiceRequest) Descriptor() ([]byte, []int) {
	return file_greeeing_proto_rawDescGZIP(), []int{0}
}

func (x *GreetingServiceRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type GreetingServiceReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Message string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *GreetingServiceReply) Reset() {
	*x = GreetingServiceReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_greeeing_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GreetingServiceReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GreetingServiceReply) ProtoMessage() {}

func (x *GreetingServiceReply) ProtoReflect() protoreflect.Message {
	mi := &file_greeeing_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GreetingServiceReply.ProtoReflect.Descriptor instead.
func (*GreetingServiceReply) Descriptor() ([]byte, []int) {
	return file_greeeing_proto_rawDescGZIP(), []int{1}
}

func (x *GreetingServiceReply) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

var File_greeeing_proto protoreflect.FileDescriptor

var file_greeeing_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x67, 0x72, 0x65, 0x65, 0x65, 0x69, 0x6e, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0x2c, 0x0a, 0x16, 0x47, 0x72, 0x65, 0x65, 0x74, 0x69, 0x6e, 0x67, 0x53, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x30,
	0x0a, 0x14, 0x47, 0x72, 0x65, 0x65, 0x74, 0x69, 0x6e, 0x67, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x32, 0x4f, 0x0a, 0x0f, 0x47, 0x72, 0x65, 0x65, 0x74, 0x69, 0x6e, 0x67, 0x53, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x12, 0x3c, 0x0a, 0x08, 0x47, 0x72, 0x65, 0x65, 0x74, 0x69, 0x6e, 0x67, 0x12,
	0x17, 0x2e, 0x47, 0x72, 0x65, 0x65, 0x74, 0x69, 0x6e, 0x67, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x15, 0x2e, 0x47, 0x72, 0x65, 0x65, 0x74,
	0x69, 0x6e, 0x67, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22,
	0x00, 0x42, 0x05, 0x5a, 0x03, 0x2f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_greeeing_proto_rawDescOnce sync.Once
	file_greeeing_proto_rawDescData = file_greeeing_proto_rawDesc
)

func file_greeeing_proto_rawDescGZIP() []byte {
	file_greeeing_proto_rawDescOnce.Do(func() {
		file_greeeing_proto_rawDescData = protoimpl.X.CompressGZIP(file_greeeing_proto_rawDescData)
	})
	return file_greeeing_proto_rawDescData
}

var file_greeeing_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_greeeing_proto_goTypes = []interface{}{
	(*GreetingServiceRequest)(nil), // 0: GreetingServiceRequest
	(*GreetingServiceReply)(nil),   // 1: GreetingServiceReply
}
var file_greeeing_proto_depIdxs = []int32{
	0, // 0: GreetingService.Greeting:input_type -> GreetingServiceRequest
	1, // 1: GreetingService.Greeting:output_type -> GreetingServiceReply
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_greeeing_proto_init() }
func file_greeeing_proto_init() {
	if File_greeeing_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_greeeing_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GreetingServiceRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_greeeing_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GreetingServiceReply); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_greeeing_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_greeeing_proto_goTypes,
		DependencyIndexes: file_greeeing_proto_depIdxs,
		MessageInfos:      file_greeeing_proto_msgTypes,
	}.Build()
	File_greeeing_proto = out.File
	file_greeeing_proto_rawDesc = nil
	file_greeeing_proto_goTypes = nil
	file_greeeing_proto_depIdxs = nil
}
