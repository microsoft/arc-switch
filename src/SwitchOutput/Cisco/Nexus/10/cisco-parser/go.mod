module cisco-parser

go 1.24.3

require (
	bgp_all_summary_parser v0.0.0
	class_map_parser v0.0.0
	environment_power_parser v0.0.0
	environment_temperature_parser v0.0.0
	interface_counters_error_parser v0.0.0
	interface_counters_parser v0.0.0
	inventory_parser v0.0.0
	ip_arp_parser v0.0.0
	ip_route_parser v0.0.0
	lldp_neighbor_parser v0.0.0
	mac_address_parser v0.0.0
	transceiver_parser v0.0.0
)

replace bgp_all_summary_parser => ../bgp_all_summary_parser

replace class_map_parser => ../class_map_parser

replace environment_power_parser => ../show-environment-power-details

replace environment_temperature_parser => ../environment_temperature_parser

replace interface_counters_parser => ../interface_counters_parser

replace interface_counters_error_parser => ../interface_counters_error_parser

replace inventory_parser => ../inventory_parser

replace ip_arp_parser => ../ip_arp_parser

replace ip_route_parser => ../ip_route_parser

replace lldp_neighbor_parser => ../lldp_neighbor_parser

replace mac_address_parser => ../mac_address_parser

replace transceiver_parser => ../transceiver_parser
