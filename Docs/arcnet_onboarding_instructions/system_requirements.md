# Azure Arc for Cisco Nexus Switches

## System Requirements


### Hardware Requirements

Cisco Nexus Switch Specifications:
- CPU: Quad-core processor or higher
- Memory/RAM: 16 GB RAM minimum
- Storage: 5 GB free space on bootflash
- Platform: Cisco Nexus 9k Series (recommended) or any NXOS-supported Nexus switch

### Software Requirements

- NX-OS Version: 9.x or later
- Bash shell access enabled
- Python support (included in NX-OS)


### Baseline requirements:

- Average CPU utilization < 40%
- Memory utilization < 70%
- 5 GB free storage on bootflash


### Network Requirements

Connectivity:
- Outbound HTTPS (port 443) to Azure endpoints
- DNS resolution enabled

Required endpoints:

*.his.arc.azure.com
*.guestconfiguration.azure.com
*.blob.core.windows.net
management.azure.com
login.microsoftonline.com
github.com

Ensure firewall rules allow outbound HTTPS to these domains