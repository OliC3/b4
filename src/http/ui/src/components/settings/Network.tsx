import { NetworkIcon } from "@b4.icons";
import { B4FormGroup, B4Section, B4TextField, B4Slider } from "@b4.elements";
import { B4Config } from "@models/config";

interface NetworkSettingsProps {
  config: B4Config;
  onChange: (
    field: string,
    value: number | boolean | string | string[]
  ) => void;
}

export const NetworkSettings = ({ config, onChange }: NetworkSettingsProps) => (
  <B4Section
    title="Network Configuration"
    description="Configure netfilter queue and network processing parameters"
    icon={<NetworkIcon />}
  >
    <B4FormGroup label="Queue Settings" columns={2}>
      <B4TextField
        label="Queue Start Number"
        type="number"
        value={config.queue.start_num}
        onChange={(e) => onChange("queue.start_num", Number(e.target.value))}
        helperText="Netfilter queue number (0-65535)"
      />
      <B4TextField
        label="Packet Mark"
        type="number"
        value={config.queue.mark}
        onChange={(e) => onChange("queue.mark", Number(e.target.value))}
        helperText="Netfilter packet mark for iptables rules (default: 32768)"
      />
      <B4Slider
        label="Worker Threads"
        value={config.queue.threads}
        onChange={(value) => onChange("queue.threads", value)}
        min={1}
        max={16}
        step={1}
        helperText="Number of worker threads for processing packets simultaneously (default 4)"
      />
    </B4FormGroup>
    <B4FormGroup label="Web Server" columns={2}>
      <B4TextField
        label="Bind Address"
        value={config.system.web_server.bind_address || "0.0.0.0"}
        onChange={(e) =>
          onChange("system.web_server.bind_address", e.target.value)
        }
        placeholder="0.0.0.0"
        helperText="IP to bind (0.0.0.0 = all, 127.0.0.1 = localhost only, :: = all IPv6)"
      />
      <B4TextField
        label="Port"
        type="number"
        value={config.system.web_server.port}
        onChange={(e) =>
          onChange("system.web_server.port", Number(e.target.value))
        }
        helperText="Web UI port (default: 7000)"
      />
    </B4FormGroup>
  </B4Section>
);
