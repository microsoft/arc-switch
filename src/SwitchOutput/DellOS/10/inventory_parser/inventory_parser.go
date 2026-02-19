package inventory_parser

import (
	"regexp"
	"strings"
	"time"
)

type StandardizedEntry struct {
	DataType  string      `json:"data_type"`
	Timestamp string      `json:"timestamp"`
	Date      string      `json:"date"`
	Message   interface{} `json:"message"`
}

type InventoryData struct {
	Product             string          `json:"product"`
	Description         string          `json:"description"`
	SoftwareVersion     string          `json:"software_version"`
	ProductBase         string          `json:"product_base"`
	ProductSerialNumber string          `json:"product_serial_number"`
	ProductPartNumber   string          `json:"product_part_number"`
	Units               []InventoryUnit `json:"units"`
}

type InventoryUnit struct {
	UnitID      string `json:"unit_id"`
	Type        string `json:"type"`
	PartNumber  string `json:"part_number"`
	Revision    string `json:"revision"`
	PiecePartID string `json:"piece_part_id"`
	ServiceTag  string `json:"service_tag"`
	ExpressCode string `json:"express_code"`
}

type InventoryParser struct{}

func (p *InventoryParser) GetDescription() string {
	return "Parses 'show inventory' output"
}

func (p *InventoryParser) Parse(input []byte) (interface{}, error) {
	data := InventoryData{}
	lines := strings.Split(string(input), "\n")
	kvRegex := regexp.MustCompile(`^(.+?)\s*:\s*(.*)$`)
	separatorRegex := regexp.MustCompile(`^-{10,}$`)
	unitRegex := regexp.MustCompile(`^\*?\s*(\d+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(.+)$`)
	inUnitTable := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if separatorRegex.MatchString(trimmed) {
			inUnitTable = true
			continue
		}
		if inUnitTable {
			if match := unitRegex.FindStringSubmatch(trimmed); match != nil {
				data.Units = append(data.Units, InventoryUnit{
					UnitID: match[1], Type: match[2], PartNumber: match[3],
					Revision: match[4], PiecePartID: match[5], ServiceTag: match[6],
					ExpressCode: strings.TrimSpace(match[7]),
				})
			}
			continue
		}
		if strings.HasPrefix(trimmed, "Unit Type") {
			continue
		}
		if match := kvRegex.FindStringSubmatch(trimmed); match != nil {
			key := strings.TrimSpace(match[1])
			value := strings.TrimSpace(match[2])
			switch key {
			case "Product":
				data.Product = value
			case "Description":
				data.Description = value
			case "Software version":
				data.SoftwareVersion = value
			case "Product Base":
				data.ProductBase = value
			case "Product Serial Number":
				data.ProductSerialNumber = value
			case "Product Part Number":
				data.ProductPartNumber = value
			}
		}
	}
	now := time.Now().UTC()
	return []StandardizedEntry{{
		DataType: "dell_os10_inventory", Timestamp: now.Format(time.RFC3339),
		Date: now.Format("2006-01-02"), Message: data,
	}}, nil
}
