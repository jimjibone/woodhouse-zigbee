package zigbee

type DeviceInfo struct {
	DateCode   string `json:"date_code"`
	Definition struct {
		Description string       `json:"description"`
		Model       string       `json:"model"`
		Vendor      string       `json:"vendor"`
		Exposes     []ExposeInfo `json:"exposes"`
		Options     []ExposeInfo `json:"options"`
	} `json:"definition"`
	Endpoints map[string]struct {
		Bindings []*struct {
			Cluster string `json:"cluster"`
			Target  *struct {
				Endpoint    int    `json:"endpoint"`
				IEEEAddress string `json:"ieee_address"`
				Type        string `json:"type"`
			} `json:"target"`
		} `json:"bindings"`
		Clusters             map[string][]string `json:"clusters"`
		ConfiguredReportings []struct {
			Attribute             string `json:"attribute"`
			Cluster               string `json:"cluster"`
			MaximumReportInterval int    `json:"maximum_report_interval"`
			MinimumReportInterval int    `json:"minimum_report_interval"`
			ReportableChange      int    `json:"reportable_change"`
		} `json:"configured_reportings"`
	} `json:"endpoints"`
	FriendlyName       string `json:"friendly_name"`
	IEEEAddress        string `json:"ieee_address"`
	InterviewCompleted bool   `json:"interview_completed"`
	Interviewing       bool   `json:"interviewing"`
	ModelID            string `json:"model_id"`
	Manufacturer       string `json:"manufacturer"`
	NetworkAddress     int    `json:"network_address"`
	PowerSource        string `json:"power_source"`
	SoftwareBuildID    string `json:"software_build_id"`
	Supported          bool   `json:"supported"`
	Type               string `json:"type"`
}
