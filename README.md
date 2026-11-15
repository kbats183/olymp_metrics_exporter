# Olymp Metrics Exporter for Windows

A lightweight Prometheus exporter for Windows machines used in programming competitions. Collects CPU usage, memory usage, network traffic, and power status every 5 seconds.

Requires Go 1.23 or later to build:

```powershell
go build -o metrics_exporter.exe .
.\metrics_exporter.exe -ip 0.0.0.0 -port 12341
```

Listens on `0.0.0.0:12341` by default. Metrics are available at `/metrics`; `/health` returns `ok`.

| Metric | Description |
| --- | --- |
| `system_cpu_usage_percent` | CPU usage histogram, % |
| `system_memory_usage_percent` | Memory usage histogram, % |
| `system_network_receive_mbytes_total` | Traffic received across all interfaces since the exporter started, MiB |
| `system_network_transmit_mbytes_total` | Traffic sent across all interfaces since the exporter started, MiB |
| `system_power_on_ac` | `1` when connected to AC power, `0` otherwise |

Example Prometheus configuration:

```yaml
scrape_configs:
  - job_name: windows_olymp
    scrape_interval: 30s
    file_sd_configs:
      - files:
          - /etc/prometheus/targets/participants.json
        refresh_interval: 30s
    relabel_configs:
      - source_labels: [__address__]
        regex: '\d+\.\d*(\d)\.(\d{1,3})\.(.*):.*'
        target_label: instance
        replacement: '$1.$2'
```

And `participants.json`:
```json
[
  {
    "targets": [
      "10.241.1.2:12341",
      "10.241.2.2:12341",
      "10.241.3.2:12341",
      "10.241.20.2:12341"
    ],
    "labels": {
      "hall": "1"
    }
  },
  {
    "targets": [
      "10.242.1.2:12341"
    ],
    "labels": {
      "hall": "2"
    }
  }
]
```

To install as a Windows service, run these commands in CMD as administrator, using the full path to the executable:

```bat
sc create _mitrics_exporter binPath= "C:\windows\admin\_mitrics_exporter\mitrics_exporter.exe" start= auto DisplayName= "Prometheus Mitrics Exporter"
sc start _mitrics_exporter
```
