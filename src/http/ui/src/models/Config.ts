export type FakingStrategy =
  | "ttl"
  | "pastseq"
  | "randseq"
  | "tcp_check"
  | "md5sum";

export enum FakingPayloadType {
  RANDOM = 0,
  CUSTOM = 1,
  DEFAULT = 2,
}
export interface FakingConfig {
  strategy: FakingStrategy;
  sni: boolean;
  ttl: number;
  seq_offset: number;
  sni_seq_length: number;
  sni_type: FakingPayloadType;
  custom_payload: string;
}

export type FragmentationStrategy = "tcp" | "ip" | "none";

export interface FragmentationConfig {
  strategy: FragmentationStrategy;
  sni_position: number;
  sni_reverse: boolean;
  middle_sni: boolean;
}

export enum LogLevel {
  ERROR = 0,
  INFO = 1,
  TRACE = 2,
  DEBUG = 3,
}

export interface LoggingConfig {
  level: LogLevel;
  instaflush: boolean;
  syslog: boolean;
}

export interface DomainConfig {
  sni_domains: string[];
  geosite_path: string;
  geoip_path: string;
  geosite_categories: string[];
  geoip_categories: string[];
  block_domains: string[];
  block_geosite_categories: string[];
}

export interface DomainStatisticsConfig {
  manual_domains: number;
  geosite_domains: number;
  total_domains: number;
  category_breakdown?: Record<string, number>;
  geosite_available: boolean;
}

export interface CategoryPreviewConfig {
  category: string;
  total_domains: number;
  preview_count: number;
  preview: string[];
}

export type UdpMode = "drop" | "fake";
export type UdpFilterQuicMode = "disabled" | "all" | "parse";
export type UdpFakingStrategy = "none" | "ttl" | "checksum";

export interface UdpConfig {
  mode: UdpMode;
  fake_seq_length: number;
  fake_len: number;
  faking_strategy: UdpFakingStrategy;
  dport_min: number;
  dport_max: number;
  filter_quic: UdpFilterQuicMode;
  conn_bytes_limit: number;
  filter_stun: boolean;
}
export interface QueueConfig {
  start_num: number;
  threads: number;
  mark: number;
  ipv4: boolean;
  ipv6: boolean;
}

export interface CheckerConfig {
  timeout: number;
  max_concurrent: number;
  domains: string[];
}

export interface TcpConfig {
  conn_bytes_limit: number;
  seg2delay: number;
}

export interface BypassConfig {
  tcp: TcpConfig;
  udp: UdpConfig;
  fragmentation: FragmentationConfig;
  faking: FakingConfig;
}
export interface WebServerConfig {
  port: number;
}
export interface TableConfig {
  monitor_interval: number;
  skip_setup: false;
}
export interface SystemConfig {
  logging: LoggingConfig;
  web_server: WebServerConfig;
  tables: TableConfig;
  checker: CheckerConfig;
}

export interface B4Config {
  queue: QueueConfig;
  domains: DomainConfig;
  system: SystemConfig;
  bypass: BypassConfig;
}
