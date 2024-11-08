# PVE Status Telegram Bot

This bot sends status updates from a Proxmox Virtual Environment (PVE) server to a Telegram channel.

## Features

- **Temperature Monitoring**: Monitors the temperature of the CPU and sends alerts if the temperature is too high.

## Configuration

The bot requires a configuration file in JSON format. The configuration file should contain the following fields:

- `token`: The Telegram bot token.
- `target_id`: The ID of the Telegram chat where the updates will be sent.

## Docker

```bash
docker build -t ghcr.io/tbxark-arc/pve-status .
docker run -d \
  --name pve-status-telegram-bot \
  -v /sys/class/thermal:/sys/class/thermal \
  -v /path/to/config.json:/config.json \
  ghcr.io/tbxark-arc/pve-status
```

## License
This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.
