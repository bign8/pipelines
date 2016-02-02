#!/bin/bash

# Requires VirtualBox with vm named pfSense and properly configured
# WAN: Don't care
# LAN: 192.168.2.1/24
# DNS: 8.8.8.8;8.8.4.4
# DHCP: 192.168.2.10-192.168.2.254

VBoxHeadless --startvm pfSense
