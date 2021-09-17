package producer

import (
	"fmt"
	"reflect"

	"github.com/netsampler/goflow2/decoders/netflow"
	flowmessage "github.com/netsampler/goflow2/pb"
)

func MapCustom(flowMessage *flowmessage.FlowMessage, df netflow.DataField, mapper *NetFlowMapper) {
	mapped, ok := mapper.Map(df)
	if ok {
		vfm := reflect.ValueOf(flowMessage)
		vfm = reflect.Indirect(vfm)

		fieldValue := vfm.FieldByName(mapped.Destination)
		if fieldValue.IsValid() {
			typeDest := fieldValue.Type()
			fieldValueAddr := fieldValue.Addr()
			v := df.Value.([]byte)
			if typeDest.Kind() == reflect.Slice && typeDest.Elem().Kind() == reflect.Uint8 {
				fieldValue.SetBytes(v)
			} else if fieldValueAddr.IsValid() && (typeDest.Kind() == reflect.Uint8 || typeDest.Kind() == reflect.Uint16 || typeDest.Kind() == reflect.Uint32 || typeDest.Kind() == reflect.Uint64) {
				DecodeUNumber(v, fieldValueAddr.Interface())
			} else if fieldValueAddr.IsValid() && (typeDest.Kind() == reflect.Int8 || typeDest.Kind() == reflect.Int16 || typeDest.Kind() == reflect.Int32 || typeDest.Kind() == reflect.Int64) {
				DecodeNumber(v, fieldValueAddr.Interface())
			}
		}
	}
}

type NetFlowMapField struct {
	PenProvided bool
	Type        uint16 `json:"field"`
	Pen         uint32 `json:"pen"`

	Destination string `json:"destination"`
	//DestinationLength uint8  `json:"dlen"` // could be used if populating a slice of uint16 that aren't in protobuf
}

type IPFIXProducerConfig struct {
	Mapping []NetFlowMapField `json:"mapping"`
	//PacketMapping []SFlowMapField   `json:"packet-mapping"` // for embedded frames: use sFlow configuration
}

type NetFlowV9ProducerConfig struct {
	Mapping []NetFlowMapField `json:"mapping"`
}

type SFlowMapField struct {
	Layer  int `json:"layer"`
	Offset int `json:"offset"`
	Length int `json:"length"`

	Destination       string `json:"destination"`
	DestinationLength uint8  `json:"dlen"`
}

type SFlowProducerConfig struct {
	Mapping []SFlowMapField `json:"mapping"`
}

type ProducerConfig struct {
	IPFIX     IPFIXProducerConfig     `json:"ipfix"`
	NetFlowV9 NetFlowV9ProducerConfig `json:"netflowv9"`
	SFlow     SFlowProducerConfig     `json:"sflow"` // also used for IPFIX data frames

	// should do a rename map list for when printing
}

type DataMap struct {
	Destination string
}

type NetFlowMapper struct {
	data map[string]DataMap // maps field to destination
}

func (m *NetFlowMapper) Map(field netflow.DataField) (DataMap, bool) {
	mapped, found := m.data[fmt.Sprintf("%v-%d-%d", field.PenProvided, field.Pen, field.Type)]
	return mapped, found
}

func MapFieldsNetFlow(fields []NetFlowMapField) *NetFlowMapper {
	ret := make(map[string]DataMap)
	for _, field := range fields {
		ret[fmt.Sprintf("%v-%d-%d", field.PenProvided, field.Pen, field.Type)] = DataMap{Destination: field.Destination}
	}
	return &NetFlowMapper{ret}
}

type DataMapLayer struct {
	Offset      int
	Length      int
	Destination string
}

type SFlowMapper struct {
	data map[int][]DataMapLayer // map layer to list of offsets
}

func MapFieldsSFlow(fields []SFlowMapField) *SFlowMapper {
	ret := make(map[int][]DataMapLayer)
	for _, field := range fields {
		retLayerEntry := DataMapLayer{
			Offset:      field.Offset,
			Length:      field.Length,
			Destination: field.Destination,
		}
		retLayer, ok := ret[field.Layer]
		if !ok {
			retLayer = make([]DataMapLayer, 0)
		}
		retLayer = append(retLayer, retLayerEntry)
		ret[field.Layer] = retLayer

	}
	return &SFlowMapper{ret}
}

type ProducerConfigMapped struct {
	IPFIX     *NetFlowMapper
	NetFlowV9 *NetFlowMapper
	SFlow     *SFlowMapper
}

func NewProducerConfigMapped(config *ProducerConfig) *ProducerConfigMapped {
	newCfg := &ProducerConfigMapped{}
	if config != nil {
		newCfg.IPFIX = MapFieldsNetFlow(config.IPFIX.Mapping)
		newCfg.NetFlowV9 = MapFieldsNetFlow(config.NetFlowV9.Mapping)
		newCfg.SFlow = MapFieldsSFlow(config.SFlow.Mapping)
	}
	return newCfg
}
