# Systemd

This is a sample systemd unit and environment file that you could use to run Distro with.
It makes no assumptions about running proxies or databases on the same machine, 
therefore if you do this you may wish to add addition requirements to the unit.
The unit will write logs to the journal which you can read with
`journalctl -u distro.service`

Example install, and for convenience uninstall, scripts are located in the util subdir. 
The scripts expect that you manually install distro in /usr/bin and have the config file 
/etc/distro/config.json. The config file location can be altered via the env file, 
which the script installs as /etc/distro/env