package domain

// StandardTermItem defines one globally shared canonical term and its aliases.
type StandardTermItem struct {
	ID      string   `json:"id"`
	Term    string   `json:"term"`
	Aliases []string `json:"aliases"`
}

// StandardTermsSetting is stored in the global system_settings table.
type StandardTermsSetting struct {
	Items []StandardTermItem `json:"items"`
}

type ReplaceStandardTermsReq struct {
	Items []StandardTermItem `json:"items"`
}
