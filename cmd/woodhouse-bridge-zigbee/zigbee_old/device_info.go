package zigbee_old

import (
	"fmt"
	"log"
)

type DeviceInfos []*DeviceInfo

type DeviceInfo struct {
	DateCode           string               `json:"date_code"`
	Definition         *Definition          `json:"definition"`
	Endpoints          map[string]*Endpoint `json:"endpoints"`
	FriendlyName       string               `json:"friendly_name"`
	IEEEAddress        string               `json:"ieee_address"`
	InterviewCompleted bool                 `json:"interview_completed"`
	Interviewing       bool                 `json:"interviewing"`
	ModelID            string               `json:"model_id"`
	NetworkAddress     int                  `json:"network_address"`
	PowerSource        string               `json:"power_source"`
	SoftwareBuildID    string               `json:"software_build_id"`
	Supported          bool                 `json:"supported"`
	Type               string               `json:"type"`
}

func (d *DeviceInfo) String() string {
	return fmt.Sprintf("%s (%s) %v", d.FriendlyName, d.IEEEAddress, d.Definition)
}

func (d DeviceInfo) LongString(indent string) string {
	if d.Definition != nil {
		return fmt.Sprintf("%s%s (%s)\n%v", indent, d.FriendlyName, d.IEEEAddress, d.Definition.LongString(indent+"  "))
	}
	return fmt.Sprintf("%s%s (%s)", indent, d.FriendlyName, d.IEEEAddress)
}

type Definition struct {
	Description string     `json:"description"`
	Model       string     `json:"model"`
	Vendor      string     `json:"vendor"`
	Exposes     []*Exposed `json:"exposes"`
}

func (d *Definition) String() string {
	return fmt.Sprintf("%s (%s by %s)", d.Description, d.Model, d.Vendor)
}

func (d *Definition) LongString(indent string) string {
	msg := fmt.Sprintf("%s%s (%s by %s), exposes %d:", indent, d.Description, d.Model, d.Vendor, len(d.Exposes))
	for _, ex := range d.Exposes {
		msg += "\n" + ex.LongString(indent+"  ")
	}
	return msg
}

func (d *Definition) FlattenExposes() (flattened map[string]*Exposed, typeName string) {
	flattened = make(map[string]*Exposed)
	for _, e := range d.Exposes {
		log.Printf("--- flattening %s as %s", e.Name, e)
		switch ev := e.Value.(type) {
		case *ExposedBinary, *ExposedNumeric, *ExposedEnum, *ExposedText:
			log.Printf("------ flattening %s, property: %s, simple", e.Name, e.Property)
			flattened[e.Property] = e
		case *ExposedComposite:
			log.Printf("------ flattening %s, property: %s, composite", e.Name, e.Property)
			for _, feature := range ev.Features {
				log.Printf("--------- flattening %s, property: %s, composite item: %s", e.Name, e.Property, feature.Property)
				feature.PrefixProperty = e.Property
				flattened[e.Property+"."+feature.Property] = feature
			}
		case *ExposedFeatures:
			typeName = e.Type
			log.Printf("------ flattening feature %s", e.Type)
			for _, feature := range ev.Features {
				log.Printf("--------- flattening feature %s item, property: %s", e.Type, feature.Property)
				// flattened[e.Property+"."+feature.Property] = feature
				// def := &Definition{
				// 	Exposes: []*Exposed{feature},
				// }
				// tmp, _ := def.FlattenExposes()
				// for _, feature2 := range tmp {
				// 	flattened[feature2.Property] = feature2
				// }

				switch ev := feature.Value.(type) {
				case *ExposedBinary, *ExposedNumeric, *ExposedEnum, *ExposedText:
					log.Printf("--------- flattening feature %s item, property: %s, simple", e.Type, feature.Property)
					flattened[feature.Property] = feature
				case *ExposedComposite:
					log.Printf("--------- flattening feature %s item, property: %s, composite", e.Type, feature.Property)
					for _, feature2 := range ev.Features {
						log.Printf("--------- flattening feature %s item, property: %s, composite item: %s", e.Type, feature.Property, feature2.Property)
						feature2.PrefixProperty = feature.Property
						flattened[feature.Property+"."+feature2.Property] = feature2
					}
				default:
					panic(fmt.Sprintf("invalid feature ExposedValue type (%T)", ev))
				}
			}
		default:
			panic(fmt.Sprintf("invalid ExposedValue type (%T)", ev))
		}
	}
	return flattened, typeName
}

type Endpoint struct {
	Bindings             []*Binding             `json:"bindings"`
	Clusters             map[string][]string    `json:"clusters"`
	ConfiguredReportings []*ConfiguredReporting `json:"configured_reportings"`
}

type Binding struct {
	Cluster string         `json:"cluster"`
	Target  *BindingTarget `json:"target"`
}

type BindingTarget struct {
	Endpoint    int    `json:"endpoint"`
	IEEEAddress string `json:"ieee_address"`
	Type        string `json:"type"`
}

type ConfiguredReporting struct {
	Attribute             string `json:"attribute"`
	Cluster               string `json:"cluster"`
	MaximumReportInterval int    `json:"maximum_report_interval"`
	MinimumReportInterval int    `json:"minimum_report_interval"`
	ReportableChange      int    `json:"reportable_change"`
}
