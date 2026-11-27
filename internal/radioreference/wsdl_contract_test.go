package radioreference

// The tables below are RadioReference's own contract, generated from its
// published WSDL (http://api.radioreference.com/soap2/?wsdl&v=latest, service
// version 18, read on 26 Nov 2026) for the operations the system import
// calls. They are NOT this package's reading of the service: the fake server
// in fullsystem_test.go refuses a request that does not match wsdlOps, and
// builds every response from wsdlStructs / wsdlArrays, so a fixture cannot
// contain an element the service does not define. That is the trap #1403
// fell into: the old fixtures used <siteList>, <siteFreq> and <tgList>,
// names the parser also looked for and the WSDL does not have.
//
// TestWSDLContract re-checks the tables against a live or saved WSDL.

type wsdlField struct{ name, typ string }

type wsdlOp struct {
	params  []wsdlField
	returns string
}

var wsdlOps = map[string]wsdlOp{
	"getTrsDetails":    {params: []wsdlField{{"sid", "xsd:int"}, {"authInfo", "tns:authInfo"}}, returns: "tns:Trs"},
	"getTrsSites":      {params: []wsdlField{{"sid", "xsd:int"}, {"authInfo", "tns:authInfo"}}, returns: "tns:TrsSites"},
	"getTrsTalkgroups": {params: []wsdlField{{"sid", "xsd:int"}, {"tgCid", "xsd:int"}, {"tgTag", "xsd:int"}, {"tgDec", "xsd:int"}, {"authInfo", "tns:authInfo"}}, returns: "tns:Talkgroups"},
	"getTrsType":       {params: []wsdlField{{"id", "xsd:int"}, {"authInfo", "tns:authInfo"}}, returns: "tns:TrsType"},
	"getTrsFlavor":     {params: []wsdlField{{"id", "xsd:int"}, {"authInfo", "tns:authInfo"}}, returns: "tns:TrsFlavor"},
}

// wsdlArrays maps a SOAP-encoded array type to the type of its entries.
var wsdlArrays = map[string]string{
	"Rectangles":      "Rectangle",
	"Talkgroups":      "Talkgroup",
	"TrsBandplan":     "trsBandplanDef",
	"TrsFlavor":       "trsFlavorDef",
	"TrsSiteFreqs":    "TrsSiteFreq",
	"TrsSiteLicenses": "TrsSiteLicense",
	"TrsSites":        "TrsSite",
	"TrsSysid":        "trsSysidDef",
	"TrsType":         "trsTypeDef",
	"ctids":           "ctid",
	"stids":           "stid",
	"tags":            "tag",
}

// wsdlStructs lists each struct type's fields in the WSDL's order.
var wsdlStructs = map[string][]wsdlField{
	"Rectangle": {
		{"nw_lat", "xsd:decimal"},
		{"nw_lon", "xsd:decimal"},
		{"se_lat", "xsd:decimal"},
		{"se_lon", "xsd:decimal"},
	},
	"Talkgroup": {
		{"tgId", "xsd:int"},
		{"tgDec", "xsd:int"},
		{"tgSubfleet", "xsd:string"},
		{"tgLtr", "xsd:boolean"},
		{"tgSlot", "xsd:string"},
		{"tgDescr", "xsd:string"},
		{"tgAlpha", "xsd:string"},
		{"tgMode", "xsd:string"},
		{"enc", "xsd:int"},
		{"tags", "tns:tags"},
		{"tgCid", "xsd:int"},
		{"tgSort", "xsd:int"},
		{"tgDate", "xsd:dateTime"},
	},
	"Trs": {
		{"sName", "xsd:string"},
		{"sType", "xsd:int"},
		{"sFlavor", "xsd:int"},
		{"sVoice", "xsd:int"},
		{"sCity", "xsd:string"},
		{"sCounty", "tns:ctids"},
		{"sState", "tns:stids"},
		{"sCountry", "xsd:string"},
		{"lat", "xsd:decimal"},
		{"lon", "xsd:decimal"},
		{"range", "xsd:decimal"},
		{"rectangles", "tns:Rectangles"},
		{"lastUpdated", "xsd:dateTime"},
		{"sysid", "tns:TrsSysid"},
		{"bandplan", "tns:TrsBandplan"},
		{"fleetmap", "tns:TrsFleetmap"},
	},
	"TrsFleetmap": {
		{"b0", "xsd:string"},
		{"b1", "xsd:string"},
		{"b2", "xsd:string"},
		{"b3", "xsd:string"},
		{"b4", "xsd:string"},
		{"b5", "xsd:string"},
		{"b6", "xsd:string"},
		{"b7", "xsd:string"},
	},
	"TrsSite": {
		{"siteId", "xsd:int"},
		{"sid", "xsd:int"},
		{"siteNumber", "xsd:int"},
		{"siteDescr", "xsd:string"},
		{"zoneNumber", "xsd:int"},
		{"zoneDescr", "xsd:string"},
		{"rfss", "xsd:int"},
		{"nac", "xsd:string"},
		{"ran", "xsd:int"},
		{"siteNeighbors", "xsd:string"},
		{"siteLocation", "xsd:string"},
		{"siteCtid", "xsd:int"},
		{"siteCt", "xsd:string"},
		{"siteModulation", "xsd:string"},
		{"siteNotes", "xsd:string"},
		{"lat", "xsd:decimal"},
		{"lon", "xsd:decimal"},
		{"range", "xsd:decimal"},
		{"rectangles", "tns:Rectangles"},
		{"splinter", "xsd:int"},
		{"rebanded", "xsd:int"},
		{"tdma_cc", "xsd:int"},
		{"siteLicenses", "tns:TrsSiteLicenses"},
		{"siteFreqs", "tns:TrsSiteFreqs"},
		{"bandplan", "tns:TrsBandplan"},
	},
	"TrsSiteFreq": {
		{"lcn", "xsd:int"},
		{"freq", "xsd:decimal"},
		{"use", "xsd:string"},
		{"colorCode", "xsd:string"},
		{"ch_id", "xsd:string"},
	},
	"TrsSiteLicense": {
		{"license", "xsd:string"},
	},
	"authInfo": {
		{"username", "xsd:string"},
		{"password", "xsd:string"},
		{"appKey", "xsd:string"},
		{"version", "xsd:string"},
		{"style", "xsd:string"},
	},
	"ctid": {
		{"ctid", "xsd:int"},
	},
	"stid": {
		{"stid", "xsd:int"},
	},
	"tag": {
		{"tagId", "xsd:int"},
		{"tagDescr", "xsd:string"},
	},
	"trsBandplanDef": {
		{"base", "xsd:string"},
		{"spacing", "xsd:string"},
		{"offset", "xsd:string"},
	},
	"trsFlavorDef": {
		{"sType", "xsd:int"},
		{"sFlavor", "xsd:int"},
		{"sFlavorDescr", "xsd:string"},
	},
	"trsSysidDef": {
		{"sysid", "xsd:string"},
		{"ct", "xsd:string"},
		{"wacn", "xsd:string"},
		{"model", "xsd:string"},
	},
	"trsTypeDef": {
		{"sType", "xsd:int"},
		{"sTypeDescr", "xsd:string"},
	},
}
