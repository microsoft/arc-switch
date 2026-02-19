module dell-parser

go 1.21

require (
	bgp_summary_parser v0.0.0
	environment_temperature_parser v0.0.0
	interface_status_parser v0.0.0
	inventory_parser v0.0.0
	lldp_neighbor_parser v0.0.0
	processes_cpu_parser v0.0.0
	system_parser v0.0.0
	system_uptime_parser v0.0.0
	version_parser v0.0.0
)

replace (
	bgp_summary_parser => ../bgp_summary_parser
	environment_temperature_parser => ../environment_temperature_parser
	interface_status_parser => ../interface_status_parser
	inventory_parser => ../inventory_parser
	lldp_neighbor_parser => ../lldp_neighbor_parser
	processes_cpu_parser => ../processes_cpu_parser
	system_parser => ../system_parser
	system_uptime_parser => ../system_uptime_parser
	version_parser => ../version_parser
)
