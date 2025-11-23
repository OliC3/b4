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

export type FragmentationStrategy = "tcp" | "ip" | "tls" | "oob" | "none";

export interface FragmentationConfig {
  strategy: FragmentationStrategy;
  sni_position: number;
  reverse_order: boolean;
  middle_sni: boolean;
  oob_position: number;
  oob_char: number;

  tlsrec_pos: number;
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

export interface TargetsConfig {
  sni_domains: string[];
  ip: string[];
  geosite_categories: string[];
  geoip_categories: string[];
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
  dport_filter: string;
  filter_quic: UdpFilterQuicMode;
  conn_bytes_limit: number;
  filter_stun: boolean;
  seg2delay: number;
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
  syn_fake: boolean;
  syn_fake_len: number;
  drop_sack: boolean;
}

export interface WebServerConfig {
  port: number;
}
export interface TableConfig {
  monitor_interval: number;
  skip_setup: false;
}

export interface GeoConfig {
  sitedat_url: string;
  ipdat_url: string;
  sitedat_path: string;
  ipdat_path: string;
}

export interface ApiConfig {
  ipinfo_token: string;
}

export interface SystemConfig {
  logging: LoggingConfig;
  web_server: WebServerConfig;
  tables: TableConfig;
  checker: CheckerConfig;
  geo: GeoConfig;
  api: ApiConfig;
}

export interface B4Config {
  queue: QueueConfig;
  system: SystemConfig;
  sets: B4SetConfig[];
}

export interface B4SetConfig {
  id: string;
  name: string;
  enabled: boolean;

  tcp: TcpConfig;
  udp: UdpConfig;
  fragmentation: FragmentationConfig;
  faking: FakingConfig;
  targets: TargetsConfig;
}

export const MAIN_SET_ID = "11111111-1111-1111-1111-111111111111";
export const NEW_SET_ID = "00000000-0000-0000-0000-000000000000";
