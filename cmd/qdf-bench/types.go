package main

// Typed Go structs mirroring adalanche's localmachine.Info JSON
// (github.com/lkarlslund/adalanche-sampledata, goad/localmachine/*.json). Field
// names use json tags to load the sample data; qdf serializes the same structs
// by field name. The deeply variable Task.Definition is kept as map[string]any —
// it is genuinely heterogeneous in the data — which also exercises qdf's dynamic
// map path inside an otherwise typed payload.

type Info struct {
	Machine         MachineInfo     `json:"Machine"`
	Network         NetworkInfo     `json:"Network"`
	Availability    Availability    `json:"Availability"`
	LoginPopularity LoginPopularity `json:"LoginPopularity"`
	Users           []User          `json:"Users"`
	Groups          []Group         `json:"Groups"`
	Shares          []Share         `json:"Shares"`
	Services        []Service       `json:"Services"`
	Software        []Software      `json:"Software"`
	Tasks           []Task          `json:"Tasks"`
	Privileges      []Privilege     `json:"Privileges"`
	Collector       string          `json:"Collector"`
	Version         string          `json:"Version"`
	Commit          string          `json:"Commit"`
	Collected       string          `json:"Collected"`
}

type MachineInfo struct {
	Name               string   `json:"Name"`
	LocalSID           string   `json:"LocalSID"`
	Domain             string   `json:"Domain"`
	ComputerDomainSID  string   `json:"ComputerDomainSID"`
	IsDomainJoined     bool     `json:"IsDomainJoined"`
	Architecture       string   `json:"Architecture"`
	NumberOfProcessors int      `json:"NumberOfProcessors"`
	ProductName        string   `json:"ProductName"`
	ProductType        string   `json:"ProductType"`
	ProductSuite       string   `json:"ProductSuite"`
	EditionID          string   `json:"EditionID"`
	ReleaseID          string   `json:"ReleaseID"`
	BuildBranch        string   `json:"BuildBranch"`
	Version            string   `json:"Version"`
	BuildNumber        string   `json:"BuildNumber"`
	AppCache           []string `json:"AppCache"`
}

type NetworkInfo struct {
	InternetConnectivity string             `json:"InternetConnectivity"`
	NetworkInterfaces    []NetworkInterface `json:"NetworkInterfaces"`
}

type NetworkInterface struct {
	Name       string   `json:"Name"`
	MACAddress string   `json:"MACAddress"`
	Flags      int      `json:"Flags"`
	Addresses  []string `json:"Addresses"`
}

type Availability struct {
	Day   int `json:"Day"`
	Week  int `json:"Week"`
	Month int `json:"Month"`
}

type LoginPopularity struct {
	Day   []LoginCount `json:"Day"`
	Week  []LoginCount `json:"Week"`
	Month []LoginCount `json:"Month"`
}

type LoginCount struct {
	Name  string `json:"Name"`
	SID   string `json:"SID"`
	Count int    `json:"Count"`
}

type User struct {
	Name                 string `json:"Name"`
	SID                  string `json:"SID"`
	IsEnabled            bool   `json:"IsEnabled"`
	IsAdmin              bool   `json:"IsAdmin"`
	PasswordNeverExpires bool   `json:"PasswordNeverExpires"`
	PasswordLastSet      string `json:"PasswordLastSet"`
	LastLogon            string `json:"LastLogon"`
	LastLogoff           string `json:"LastLogoff"`
	NumberOfLogins       int    `json:"NumberOfLogins"`
}

type Group struct {
	Name    string   `json:"Name"`
	SID     string   `json:"SID"`
	Members []Member `json:"Members"`
}

type Member struct {
	Name string `json:"Name"`
	SID  string `json:"SID"`
}

type Share struct {
	Name      string `json:"Name"`
	Path      string `json:"Path"`
	Remark    string `json:"Remark"`
	DACL      string `json:"DACL"`
	PathDACL  string `json:"PathDACL"`
	PathOwner string `json:"PathOwner"`
}

type Service struct {
	RegistryOwner        string   `json:"RegistryOwner"`
	RegistryDACL         string   `json:"RegistryDACL"`
	Name                 string   `json:"Name"`
	DisplayName          string   `json:"DisplayName"`
	Description          string   `json:"Description"`
	ImagePath            string   `json:"ImagePath"`
	ImageExecutable      string   `json:"ImageExecutable"`
	ImageExecutableOwner string   `json:"ImageExecutableOwner"`
	ImageExecutableDACL  string   `json:"ImageExecutableDACL"`
	Start                int      `json:"Start"`
	Type                 int      `json:"Type"`
	Account              string   `json:"Account"`
	RequiredPrivileges   []string `json:"RequiredPrivileges"`
}

type Software struct {
	DisplayName     string `json:"displayName"`
	DisplayVersion  string `json:"displayVersion"`
	Arch            string `json:"arch"`
	Publisher       string `json:"publisher"`
	InstallDate     string `json:"installDate"`
	EstimatedSize   int    `json:"estimatedSize"`
	Contact         string `json:"Contact"`
	HelpLink        string `json:"HelpLink"`
	InstallSource   string `json:"InstallSource"`
	InstallLocation string `json:"InstallLocation"`
	UninstallString string `json:"UninstallString"`
	VersionMajor    int    `json:"VersionMajor"`
	VersionMinor    int    `json:"VersionMinor"`
}

type Task struct {
	Name        string         `json:"Name"`
	Path        string         `json:"Path"`
	Definition  map[string]any `json:"Definition"`
	Enabled     bool           `json:"Enabled"`
	State       string         `json:"State"`
	MissedRuns  int            `json:"MissedRuns"`
	NextRunTime string         `json:"NextRunTime"`
	LastRunTime string         `json:"LastRunTime"`
}

type Privilege struct {
	Name         string   `json:"Name"`
	AssignedSIDs []string `json:"AssignedSIDs"`
}

// GenService and GenTask are codegen counterparts of Service and Task, used by
// the bench's codegen-vs-reflect comparison. They are distinct defined types so
// the qdfgen-generated MarshalQDF/UnmarshalQDF methods land on THEM, leaving the
// real Service / Task on the reflection path (and the typed×bundle matrix
// honoring Options — a Marshaler ignores Options by contract). GenTask carries
// the same map[string]any Definition field, so it also exercises codegen's
// dynamic-value fallback (Encoder.EncodeValue / Decoder.DecodeValue) on real
// data — proving code generation handles arbitrary values, not only static
// schema. They are zero-cost conversions of the real types (identical layout).
//
//go:generate go run github.com/alex60217101990/qdf/cmd/qdfgen -type GenService,GenTask -output types_qdf_gen.go .
type (
	GenService Service
	GenTask    Task
)
