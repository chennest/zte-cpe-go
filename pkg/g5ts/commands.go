package g5ts

type UbusCommand interface {
	Module() string
	Method() string
	Authenticated() bool
}

type LoginInfoCommand struct{}

func (c *LoginInfoCommand) Module() string        { return "zwrt_web" }
func (c *LoginInfoCommand) Method() string        { return "web_login_info" }
func (c *LoginInfoCommand) Authenticated() bool   { return false }

type LoginCommand struct {
	Password string `json:"password"`
}

func (c *LoginCommand) Module() string        { return "zwrt_web" }
func (c *LoginCommand) Method() string        { return "web_login" }
func (c *LoginCommand) Authenticated() bool   { return false }

type LogoutCommand struct{}

func (c *LogoutCommand) Module() string        { return "zwrt_web" }
func (c *LogoutCommand) Method() string        { return "web_logout" }
func (c *LogoutCommand) Authenticated() bool   { return true }

type GetCertificateCommand struct{}

func (c *GetCertificateCommand) Module() string        { return "zwrt_web" }
func (c *GetCertificateCommand) Method() string        { return "web_crt_get" }
func (c *GetCertificateCommand) Authenticated() bool   { return true }

type SetEncryptionKeyCommand struct {
	WebEnstr string `json:"web_enstr"`
}

func (c *SetEncryptionKeyCommand) Module() string        { return "zwrt_web" }
func (c *SetEncryptionKeyCommand) Method() string        { return "web_http_enstr_set" }
func (c *SetEncryptionKeyCommand) Authenticated() bool   { return true }

type GetWwanIfaceCommand struct {
	SourceModule string `json:"source_module"`
	CID          int    `json:"cid"`
}

func NewGetWwanIfaceCommand() *GetWwanIfaceCommand {
	return &GetWwanIfaceCommand{SourceModule: "web", CID: 1}
}

func (c *GetWwanIfaceCommand) Module() string        { return "zwrt_data" }
func (c *GetWwanIfaceCommand) Method() string        { return "get_wwaniface" }
func (c *GetWwanIfaceCommand) Authenticated() bool   { return true }

type SetWwanIfaceCommand struct {
	SourceModule string `json:"source_module"`
	CID          int    `json:"cid"`
	Enable       *int   `json:"enable,omitempty"`
	ConnectMode  *int   `json:"connect_mode,omitempty"`
	RoamEnable   *int   `json:"roam_enable,omitempty"`
}

func NewSetWwanIfaceCommand() *SetWwanIfaceCommand {
	return &SetWwanIfaceCommand{SourceModule: "web", CID: 1}
}

func (c *SetWwanIfaceCommand) Module() string        { return "zwrt_data" }
func (c *SetWwanIfaceCommand) Method() string        { return "set_wwaniface" }
func (c *SetWwanIfaceCommand) Authenticated() bool   { return true }

type SetNetSelectCommand struct {
	NetSelect string `json:"net_select"`
}

func (c *SetNetSelectCommand) Module() string        { return "zte_nwinfo_api" }
func (c *SetNetSelectCommand) Method() string        { return "nwinfo_set_netselect" }
func (c *SetNetSelectCommand) Authenticated() bool   { return true }

type GetNetInfoCommand struct{}

func (c *GetNetInfoCommand) Module() string        { return "zte_nwinfo_api" }
func (c *GetNetInfoCommand) Method() string        { return "nwinfo_get_netinfo" }
func (c *GetNetInfoCommand) Authenticated() bool   { return true }

type GetSimInfoCommand struct{}

func (c *GetSimInfoCommand) Module() string        { return "zwrt_zte_mdm.api" }
func (c *GetSimInfoCommand) Method() string        { return "get_sim_info" }
func (c *GetSimInfoCommand) Authenticated() bool   { return true }

type RebootCommand struct {
	ModuleName string `json:"moduleName"`
}

func NewRebootCommand() *RebootCommand {
	return &RebootCommand{ModuleName: "web"}
}

func (c *RebootCommand) Module() string        { return "zwrt_mc.device.manager" }
func (c *RebootCommand) Method() string        { return "device_reboot" }
func (c *RebootCommand) Authenticated() bool   { return true }

type GetRouterStatusCommand struct{}

func (c *GetRouterStatusCommand) Module() string        { return "zwrt_router.api" }
func (c *GetRouterStatusCommand) Method() string        { return "router_get_status" }
func (c *GetRouterStatusCommand) Authenticated() bool   { return true }

type GetUserListNumCommand struct{}

func (c *GetUserListNumCommand) Module() string        { return "zwrt_router.api" }
func (c *GetUserListNumCommand) Method() string        { return "router_get_user_list_num" }
func (c *GetUserListNumCommand) Authenticated() bool   { return true }

type GetLanAccessListCommand struct{}

func (c *GetLanAccessListCommand) Module() string        { return "zwrt_router.api" }
func (c *GetLanAccessListCommand) Method() string        { return "router_lan_access_list" }
func (c *GetLanAccessListCommand) Authenticated() bool   { return true }

type UciGetCommand struct {
	Config  string  `json:"config"`
	Section *string `json:"section,omitempty"`
}

func (c *UciGetCommand) Module() string        { return "uci" }
func (c *UciGetCommand) Method() string        { return "get" }
func (c *UciGetCommand) Authenticated() bool   { return true }

type GetApnModeCommand struct{}

func (c *GetApnModeCommand) Module() string        { return "zwrt_apn_object" }
func (c *GetApnModeCommand) Method() string        { return "get_apn_mode" }
func (c *GetApnModeCommand) Authenticated() bool   { return true }

type SetApnModeCommand struct {
	ApnMode int `json:"apn_mode"`
}

func (c *SetApnModeCommand) Module() string        { return "zwrt_apn_object" }
func (c *SetApnModeCommand) Method() string        { return "set_apn_mode" }
func (c *SetApnModeCommand) Authenticated() bool   { return true }

type GetManuApnListCommand struct{}

func (c *GetManuApnListCommand) Module() string        { return "zwrt_apn_object" }
func (c *GetManuApnListCommand) Method() string        { return "getManuApnList" }
func (c *GetManuApnListCommand) Authenticated() bool   { return true }

type ModifyManuApnCommand struct {
	ProfileName string `json:"profilename"`
	PdpType     string `json:"pdpType"`
	Apn         string `json:"wanapn"`
	AuthMode    string `json:"pppAuthMode"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	ProfileID   string `json:"profileId"`
}

func (c *ModifyManuApnCommand) Module() string        { return "zwrt_apn_object" }
func (c *ModifyManuApnCommand) Method() string        { return "modifyManuApn" }
func (c *ModifyManuApnCommand) Authenticated() bool   { return true }

type EnableManuApnCommand struct {
	ProfileID string `json:"profileId"`
}

func (c *EnableManuApnCommand) Module() string        { return "zwrt_apn_object" }
func (c *EnableManuApnCommand) Method() string        { return "enable_manu_apn_id" }
func (c *EnableManuApnCommand) Authenticated() bool   { return true }

type SetLanParaCommand struct {
	IPAddr    string `json:"ipaddr"`
	Netmask   string `json:"netmask"`
	Ignore    string `json:"ignore"`
	Leasetime string `json:"leasetime"`
}

func (c *SetLanParaCommand) Module() string        { return "zwrt_router.api" }
func (c *SetLanParaCommand) Method() string        { return "router_set_lan_para" }
func (c *SetLanParaCommand) Authenticated() bool   { return true }

type SetWanMtuCommand struct {
	Mtu string `json:"mtu"`
	Mss string `json:"mss"`
}

func (c *SetWanMtuCommand) Module() string        { return "zwrt_router.api" }
func (c *SetWanMtuCommand) Method() string        { return "router_set_wan_mtu" }
func (c *SetWanMtuCommand) Authenticated() bool   { return true }

type SetUpnpCommand struct {
	EnableUpnp int `json:"enable_upnp"`
}

func (c *SetUpnpCommand) Module() string        { return "zwrt_router.api" }
func (c *SetUpnpCommand) Method() string        { return "router_set_upnp_switch" }
func (c *SetUpnpCommand) Authenticated() bool   { return true }

type SetDmzCommand struct {
	DmzEnable int     `json:"dmz_enable"`
	DmzIP     *string `json:"dmz_ip,omitempty"`
}

func (c *SetDmzCommand) Module() string        { return "zwrt_router.api" }
func (c *SetDmzCommand) Method() string        { return "router_set_dmz" }
func (c *SetDmzCommand) Authenticated() bool   { return true }

type GetSmsParameterCommand struct{}

func (c *GetSmsParameterCommand) Module() string        { return "zwrt_wms" }
func (c *GetSmsParameterCommand) Method() string        { return "zte_wms_get_parameter" }
func (c *GetSmsParameterCommand) Authenticated() bool   { return true }
